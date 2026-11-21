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

	"github.com/geo-gaussian/pfsense-provider/internal/api"
)

func TestTerraformAliasLifecycle(t *testing.T) {
	terraform, err := exec.LookPath("terraform")
	if err != nil {
		t.Skip("Terraform CLI is not installed")
	}
	dir := t.TempDir()
	providerDir := filepath.Join(dir, "provider")
	if err := os.Mkdir(providerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	build := exec.Command("go", "build", "-o", filepath.Join(providerDir, "terraform-provider-pfsense-nexus"), ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build provider: %v\n%s", err, output)
	}

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
	config := fmt.Sprintf(`
terraform {
  required_providers {
    pfsense-nexus = { source = "geo-gaussian/pfsense-nexus" }
  }
}
provider "pfsense-nexus" {
  url = %q
  username = "test-user"
  password = %q
  insecure = true
}
resource "pfsense_nexus_firewall_alias" "test" {
  provider = pfsense-nexus
  name = "example"
  type = "host"
  targets = [{ name = "example.test" }]
}
`, server.URL, strings.Repeat("x", 12))
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	cliConfig := fmt.Sprintf("provider_installation {\n  dev_overrides { \"geo-gaussian/pfsense-nexus\" = %q }\n  direct {}\n}\n", providerDir)
	cliConfigPath := filepath.Join(dir, "terraform.rc")
	if err := os.WriteFile(cliConfigPath, []byte(cliConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(terraform, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "TF_CLI_CONFIG_FILE="+cliConfigPath, "TF_IN_AUTOMATION=1", "CHECKPOINT_DISABLE=1")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("terraform %s: %v\n%s", strings.Join(args, " "), err, output)
		}
	}
	run("apply", "-auto-approve", "-input=false", "-no-color")
	run("plan", "-detailed-exitcode", "-input=false", "-no-color")
	config = strings.Replace(config, `type = "host"`, "type = \"host\"\n  descr = \"updated\"", 1)
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	run("apply", "-auto-approve", "-input=false", "-no-color")
	run("plan", "-detailed-exitcode", "-input=false", "-no-color")
	run("state", "rm", "pfsense_nexus_firewall_alias.test")
	run("import", "-input=false", "-no-color", "pfsense_nexus_firewall_alias.test", "example")
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
	cmd.Env = append(os.Environ(), "TF_CLI_CONFIG_FILE="+cliConfigPath, "TF_IN_AUTOMATION=1", "CHECKPOINT_DISABLE=1")
	output, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "Alias created but firewall reload failed") {
		t.Fatalf("expected a reload failure after creation: %v\n%s", err, output)
	}
	run("state", "show", "-no-color", "pfsense_nexus_firewall_alias.test")
	run("apply", "-auto-approve", "-input=false", "-no-color")
	run("destroy", "-auto-approve", "-input=false", "-no-color")
}
