package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
)

func prepareTerraformFixture(t *testing.T, dir, fixture string) string {
	t.Helper()
	config, err := os.ReadFile(filepath.Join("tests", "terraform", fixture, "main.tf"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), config, 0o600); err != nil {
		t.Fatal(err)
	}
	cliConfig, err := os.ReadFile(filepath.Join("tests", "terraform", "terraform.tfrc"))
	if err != nil {
		t.Fatal(err)
	}
	cliConfigPath := filepath.Join(dir, "terraform.rc")
	if err := os.WriteFile(cliConfigPath, cliConfig, 0o600); err != nil {
		t.Fatal(err)
	}
	return cliConfigPath
}

func terraformEnvironment(cliConfigPath, serverURL, providerDir, description string) []string {
	env := make([]string, 0, len(os.Environ())+8)
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		switch key {
		case "TF_CLI_CONFIG_FILE", "TF_IN_AUTOMATION", "CHECKPOINT_DISABLE", "PFSENSE_TEST_PROVIDER_DIR",
			"TF_VAR_nexus_url", "TF_VAR_nexus_username", "TF_VAR_nexus_password", "TF_VAR_alias_description":
			continue
		}
		env = append(env, entry)
	}
	env = append(env, "TF_CLI_CONFIG_FILE="+cliConfigPath, "TF_IN_AUTOMATION=1", "CHECKPOINT_DISABLE=1",
		"PFSENSE_TEST_PROVIDER_DIR="+providerDir, "TF_VAR_nexus_url="+serverURL,
		"TF_VAR_nexus_username=test-user", "TF_VAR_nexus_password="+strings.Repeat("x", 12))
	if description != "" {
		env = append(env, "TF_VAR_alias_description="+description)
	}
	return env
}

func buildTerraformProvider(t *testing.T, providerDir string) {
	t.Helper()
	if err := os.Mkdir(providerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	build := exec.Command("go", "build", "-o", filepath.Join(providerDir, "terraform-provider-pfsense-nexus"), ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build provider: %v\n%s", err, output)
	}
}

func terraformExecutable(t *testing.T) string {
	t.Helper()
	terraform, err := exec.LookPath("terraform")
	if err != nil {
		t.Skip("Terraform CLI is not installed")
	}
	return terraform
}

func TestTerraformFirewallAliasesDataSource(t *testing.T) {
	terraform := terraformExecutable(t)
	dir := t.TempDir()
	providerDir := filepath.Join(dir, "provider${fixture}%{fixture}")
	buildTerraformProvider(t, providerDir)

	var mu sync.Mutex
	var loginCount, readCount int
	var rejectLogin bool
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/api/login":
			loginCount++
			if rejectLogin {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, _ = fmt.Fprint(w, `{"token":"test-token"}`)
		case req.Method == http.MethodGet && req.URL.Path == "/api/device/pfsense/localhost/api/aliases":
			readCount++
			if req.Header.Get("Authorization") != "Bearer test-token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_, _ = fmt.Fprint(w, `{"aliases":[{"name":"example","type":"host","truncated":true}]}`)
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	cliConfigPath := prepareTerraformFixture(t, dir, "data-source")
	plan := func() ([]byte, error) {
		cmd := exec.Command(terraform, "plan", "-input=false", "-no-color")
		cmd.Dir = dir
		cmd.Env = terraformEnvironment(cliConfigPath, server.URL, providerDir, "")
		return cmd.CombinedOutput()
	}
	output, err := plan()
	if err != nil || !strings.Contains(string(output), `"example"`) {
		t.Fatalf("terraform plan did not return the alias: %v\n%s", err, output)
	}
	mu.Lock()
	if loginCount != 1 || readCount != 1 {
		mu.Unlock()
		t.Fatalf("login requests = %d, alias reads = %d", loginCount, readCount)
	}
	rejectLogin = true
	mu.Unlock()
	output, err = plan()
	if err == nil || !strings.Contains(string(output), "Nexus authentication failed") {
		t.Fatalf("expected authentication error: %v\n%s", err, output)
	}
	mu.Lock()
	defer mu.Unlock()
	if loginCount != 2 || readCount != 1 {
		t.Fatalf("login requests = %d, alias reads = %d", loginCount, readCount)
	}
}

func TestTerraformAliasLifecycle(t *testing.T) {
	terraform := terraformExecutable(t)
	dir := t.TempDir()
	providerDir := filepath.Join(dir, "provider")
	buildTerraformProvider(t, providerDir)

	var mu sync.Mutex
	var stored *api.FWAliasReq
	var reloads int
	var failReload bool
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if req.URL.Path == "/api/login" && req.Method == http.MethodPost {
			_, _ = fmt.Fprint(w, `{"token":"test-token"}`)
			return
		}
		if req.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch {
		case req.URL.Path == "/api/device/pfsense/localhost/api/aliases" && req.Method == http.MethodPut,
			req.URL.Path == "/api/device/pfsense/localhost/api/aliases/example" && req.Method == http.MethodPost:
			var body api.FWAliasReq
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Error(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			stored = &body
			_, _ = fmt.Fprint(w, `{"msg":"ok"}`)
		case req.URL.Path == "/api/device/pfsense/localhost/api/aliases/example" && req.Method == http.MethodGet:
			if stored == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if err := json.NewEncoder(w).Encode(stored); err != nil {
				t.Error(err)
			}
		case req.URL.Path == "/api/device/pfsense/localhost/api/aliases/example" && req.Method == http.MethodDelete:
			stored = nil
			_, _ = fmt.Fprint(w, `{"msg":"ok"}`)
		case req.URL.Path == "/api/device/pfsense/localhost/api/system/filterreload/reload" && req.Method == http.MethodPost:
			reloads++
			if failReload {
				failReload = false
				w.WriteHeader(http.StatusBadRequest)
				_, _ = fmt.Fprint(w, `{"errmsg":"reload failed"}`)
				return
			}
			_, _ = fmt.Fprint(w, `{"result":"started"}`)
		default:
			t.Errorf("unexpected request %s %s", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	cliConfigPath := prepareTerraformFixture(t, dir, "lifecycle")
	description := ""
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(terraform, args...)
		cmd.Dir = dir
		cmd.Env = terraformEnvironment(cliConfigPath, server.URL, providerDir, description)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("terraform %s: %v\n%s", strings.Join(args, " "), err, output)
		}
	}
	run("apply", "-auto-approve", "-input=false", "-no-color")
	run("plan", "-detailed-exitcode", "-input=false", "-no-color")
	description = "updated"
	run("apply", "-auto-approve", "-input=false", "-no-color")
	run("plan", "-detailed-exitcode", "-input=false", "-no-color")
	run("state", "rm", "pfsense_firewall_alias.test")
	run("import", "-input=false", "-no-color", "pfsense_firewall_alias.test", "example")
	run("plan", "-detailed-exitcode", "-input=false", "-no-color")
	run("destroy", "-auto-approve", "-input=false", "-no-color")
	mu.Lock()
	if stored != nil || reloads != 3 {
		mu.Unlock()
		t.Fatalf("stored = %#v, reloads = %d", stored, reloads)
	}
	failReload = true
	mu.Unlock()

	cmd := exec.Command(terraform, "apply", "-auto-approve", "-input=false", "-no-color")
	cmd.Dir = dir
	cmd.Env = terraformEnvironment(cliConfigPath, server.URL, providerDir, description)
	output, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "Alias created but firewall reload failed") {
		t.Fatalf("expected a reload failure after creation: %v\n%s", err, output)
	}
	run("state", "show", "-no-color", "pfsense_firewall_alias.test")
	run("apply", "-auto-approve", "-input=false", "-no-color")
	run("destroy", "-auto-approve", "-input=false", "-no-color")
}
