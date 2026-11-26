package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/client"
)

type acceptanceConfig struct {
	url, username, password, name string
	insecure                      bool
}

func acceptanceEnvironment(config acceptanceConfig, cliConfig, providerDir, description string) []string {
	env := make([]string, 0, len(os.Environ())+10)
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if !strings.HasPrefix(key, "TF_") && key != "PFSENSE_TEST_PROVIDER_DIR" && key != "CHECKPOINT_DISABLE" {
			env = append(env, entry)
		}
	}
	return append(env,
		"TF_CLI_CONFIG_FILE="+cliConfig, "TF_IN_AUTOMATION=1", "CHECKPOINT_DISABLE=1",
		"PFSENSE_TEST_PROVIDER_DIR="+providerDir, "TF_VAR_nexus_url="+config.url,
		"TF_VAR_nexus_username="+config.username, "TF_VAR_nexus_password="+config.password,
		"TF_VAR_nexus_insecure="+strconv.FormatBool(config.insecure), "TF_VAR_alias_name="+config.name,
		"TF_VAR_alias_description="+description,
	)
}

type acceptanceHarness struct {
	config      acceptanceConfig
	terraform   string
	dir         string
	cliConfig   string
	providerDir string
	description string
	api         *api.ClientWithResponses
}

func newAcceptanceHarness(t *testing.T, config acceptanceConfig) *acceptanceHarness {
	t.Helper()
	terraform, err := exec.LookPath("terraform")
	if err != nil {
		t.Fatal("Terraform CLI is required for acceptance tests")
	}
	nexus, err := client.New(config.url, config.username, config.password, config.insecure)
	if err != nil {
		t.Fatalf("configure acceptance client: %v", err)
	}
	apiClient, err := api.NewClientWithResponses(
		strings.TrimRight(config.url, "/")+"/api/device/pfsense/localhost/api",
		api.WithHTTPClient(nexus),
	)
	if err != nil {
		t.Fatalf("configure acceptance API: %v", err)
	}
	dir := t.TempDir()
	providerDir := filepath.Join(dir, "provider")
	buildTerraformProvider(t, providerDir)
	return &acceptanceHarness{
		config: config, terraform: terraform, dir: dir, providerDir: providerDir, api: apiClient,
		cliConfig: prepareTerraformFixture(t, dir, "acceptance"), description: "acceptance initial",
	}
}

func (h *acceptanceHarness) run(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, h.terraform, args...)
	cmd.Dir = h.dir
	cmd.Env = acceptanceEnvironment(h.config, h.cliConfig, h.providerDir, h.description)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("terraform %s: %w\n%s",
			strings.Join(args, " "), err, strings.ReplaceAll(string(output), h.config.password, "[redacted]"))
	}
	return output, nil
}

func (h *acceptanceHarness) aliasExists() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	response, err := h.api.FirewallGetAliasesWithResponse(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("list aliases: %w", err)
	}
	if response.StatusCode() != 200 || response.JSON200 == nil || response.JSON200.Aliases == nil {
		return false, fmt.Errorf("list aliases: HTTP %d or missing alias list", response.StatusCode())
	}
	for _, alias := range *response.JSON200.Aliases {
		if strings.EqualFold(alias.Name, h.config.name) {
			return true, nil
		}
	}
	return false, nil
}

func (h *acceptanceHarness) cleanup(t *testing.T) {
	t.Helper()
	exists, err := h.aliasExists()
	if err != nil {
		t.Errorf("cleanup: %v; inspect alias %s manually", err, h.config.name)
		return
	}
	if !exists {
		return
	}
	tracked, err := h.aliasTracked()
	if err != nil {
		t.Errorf("cleanup: %v; inspect alias %s manually", err, h.config.name)
		return
	}
	if !tracked {
		if _, err := h.run("import", "-input=false", "-no-color", "pfsense_firewall_alias.test", h.config.name); err != nil {
			t.Errorf("cleanup import: %v; inspect alias %s manually", err, h.config.name)
			return
		}
	}
	if _, err := h.run("destroy", "-auto-approve", "-input=false", "-no-color"); err != nil {
		t.Errorf("cleanup: %v; inspect alias %s manually", err, h.config.name)
		return
	}
	exists, err = h.aliasExists()
	if err != nil || exists {
		t.Errorf("cleanup could not verify removal of alias %s: exists=%t, error=%v", h.config.name, exists, err)
	}
}

func (h *acceptanceHarness) aliasTracked() (bool, error) {
	_, err := os.Stat(filepath.Join(h.dir, "terraform.tfstate"))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect Terraform state: %w", err)
	}
	state, err := h.run("state", "list")
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(string(state), "\n") {
		if strings.TrimSpace(line) == "pfsense_firewall_alias.test" {
			return true, nil
		}
	}
	return false, nil
}

func (h *acceptanceHarness) requireRun(t *testing.T, args ...string) []byte {
	t.Helper()
	output, err := h.run(args...)
	if err != nil {
		t.Fatal(err)
	}
	return output
}

func (h *acceptanceHarness) requireState(t *testing.T) {
	t.Helper()
	var state struct {
		Values struct {
			Root struct {
				Resources []struct {
					Address string `json:"address"`
					Values  struct {
						Name  string `json:"name"`
						Descr string `json:"descr"`
					} `json:"values"`
				} `json:"resources"`
			} `json:"root_module"`
		} `json:"values"`
	}
	if err := json.Unmarshal(h.requireRun(t, "show", "-json"), &state); err != nil {
		t.Fatalf("parse Terraform state: %v", err)
	}
	for _, resource := range state.Values.Root.Resources {
		if resource.Address == "pfsense_firewall_alias.test" {
			if resource.Values.Name != h.config.name || resource.Values.Descr != h.description {
				t.Fatalf("unexpected alias state: name=%q description=%q", resource.Values.Name, resource.Values.Descr)
			}
			return
		}
	}
	t.Fatal("alias missing from Terraform state")
}

func runAliasAcceptance(t *testing.T, config acceptanceConfig) {
	t.Helper()
	h := newAcceptanceHarness(t, config)
	exists, err := h.aliasExists()
	if err != nil {
		t.Fatalf("check existing aliases: %v", err)
	}
	if exists {
		t.Fatalf("refusing to modify existing alias %s", config.name)
	}
	t.Cleanup(func() { h.cleanup(t) })
	h.requireRun(t, "apply", "-auto-approve", "-input=false", "-no-color")
	h.requireState(t)
	h.requireRun(t, "plan", "-detailed-exitcode", "-input=false", "-no-color")
	h.description = "acceptance updated"
	h.requireRun(t, "apply", "-auto-approve", "-input=false", "-no-color")
	h.requireState(t)
	h.requireRun(t, "state", "rm", "pfsense_firewall_alias.test")
	h.requireRun(t, "import", "-input=false", "-no-color", "pfsense_firewall_alias.test", config.name)
	h.requireState(t)
	h.requireRun(t, "plan", "-detailed-exitcode", "-input=false", "-no-color")
	h.requireRun(t, "destroy", "-auto-approve", "-input=false", "-no-color")
	exists, err = h.aliasExists()
	if err != nil || exists {
		t.Fatalf("alias remains after destroy or could not be verified: exists=%t, error=%v", exists, err)
	}
}

func TestAccFirewallAliasLifecycle(t *testing.T) {
	if os.Getenv("TF_ACC") != "1" {
		t.Skip("set TF_ACC=1 to run live acceptance tests")
	}
	if os.Getenv("PFSENSE_ACC_DISPOSABLE") != "1" {
		t.Fatal("set PFSENSE_ACC_DISPOSABLE=1 only for an isolated, disposable appliance")
	}
	config := acceptanceConfig{
		url: os.Getenv("PFSENSE_ACC_URL"), username: os.Getenv("PFSENSE_ACC_USERNAME"),
		password: os.Getenv("PFSENSE_ACC_PASSWORD"),
	}
	if config.url == "" || config.username == "" || config.password == "" {
		t.Fatal("PFSENSE_ACC_URL, PFSENSE_ACC_USERNAME and PFSENSE_ACC_PASSWORD are required")
	}
	switch os.Getenv("PFSENSE_ACC_INSECURE") {
	case "", "false":
	case "true":
		config.insecure = true
	default:
		t.Fatal("PFSENSE_ACC_INSECURE must be true or false")
	}
	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		t.Fatalf("generate alias name: %v", err)
	}
	config.name = "tf_acceptance_" + hex.EncodeToString(suffix[:])
	runAliasAcceptance(t, config)
}

func runMockAcceptanceHarness(t *testing.T, serverURL string) {
	t.Helper()
	config := acceptanceConfig{
		url: serverURL, username: "test-user", password: strings.Repeat("x", 12), insecure: true, name: "example",
	}
	t.Run("acceptance harness", func(t *testing.T) {
		runAliasAcceptance(t, config)
	})
	var h *acceptanceHarness
	t.Run("cleanup after state loss", func(t *testing.T) {
		h = newAcceptanceHarness(t, config)
		t.Cleanup(func() { h.cleanup(t) })
		h.requireRun(t, "apply", "-auto-approve", "-input=false", "-no-color")
		if err := os.Remove(filepath.Join(h.dir, "terraform.tfstate")); err != nil {
			t.Fatal(err)
		}
	})
	if h != nil {
		exists, err := h.aliasExists()
		if err != nil || exists {
			t.Fatalf("acceptance cleanup left an alias or could not verify removal: exists=%t, error=%v", exists, err)
		}
	}
}

func TestAcceptanceRequiresExplicitConfiguration(t *testing.T) {
	tests := []struct {
		name, disposable, diagnostic string
	}{
		{"disposable acknowledgement", "", "PFSENSE_ACC_DISPOSABLE=1"},
		{"credentials", "1", "PFSENSE_ACC_URL, PFSENSE_ACC_USERNAME and PFSENSE_ACC_PASSWORD"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestAccFirewallAliasLifecycle$")
			for _, entry := range os.Environ() {
				key, _, _ := strings.Cut(entry, "=")
				if key != "TF_ACC" && !strings.HasPrefix(key, "PFSENSE_ACC_") {
					cmd.Env = append(cmd.Env, entry)
				}
			}
			cmd.Env = append(cmd.Env, "TF_ACC=1", "PFSENSE_ACC_DISPOSABLE="+test.disposable)
			output, err := cmd.CombinedOutput()
			if err == nil || !strings.Contains(string(output), test.diagnostic) {
				t.Fatalf("expected explicit configuration error: %v\n%s", err, output)
			}
		})
	}
}
