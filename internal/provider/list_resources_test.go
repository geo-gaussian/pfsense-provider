package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/geo-gaussian/terraform-provider-pfsense-nexus/internal/api"
)

func prepareTerraformQueryFixture(t *testing.T) (string, string) {
	t.Helper()
	terraform, err := exec.LookPath("terraform")
	if err != nil {
		t.Skip("Terraform CLI is not installed")
	}
	versionOutput, err := exec.Command(terraform, "version", "-json").Output()
	if err != nil {
		t.Fatalf("read Terraform version: %v", err)
	}
	var version struct {
		TerraformVersion string `json:"terraform_version"`
	}
	if err := json.Unmarshal(versionOutput, &version); err != nil {
		t.Fatalf("parse Terraform version: %v", err)
	}
	var major, minor int
	if _, err := fmt.Sscanf(version.TerraformVersion, "%d.%d", &major, &minor); err != nil {
		t.Fatalf("parse Terraform version %q: %v", version.TerraformVersion, err)
	}
	if major < 1 || (major == 1 && minor < 14) {
		t.Skip("Terraform query requires Terraform 1.14 or newer")
	}

	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, name := range []string{"main.tf", "terraform.tfrc"} {
		data, err := os.ReadFile(filepath.Join(root, "tests", "terraform", "list", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	query, err := os.ReadFile(filepath.Join(root, "examples", "discovery", "discover.tfquery.hcl"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "discover.tfquery.hcl"), query, 0600); err != nil {
		t.Fatal(err)
	}
	providerDir := filepath.Join(dir, "mirror", "registry.terraform.io", "geo-gaussian", "pfsense-nexus", "0.0.1", runtime.GOOS+"_"+runtime.GOARCH)
	if err := os.MkdirAll(providerDir, 0700); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(providerDir, "terraform-provider-pfsense-nexus_v0.0.1")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build provider: %v\n%s", err, output)
	}
	return terraform, dir
}

func TestTerraformQueryGeneratesConfiguration(t *testing.T) {
	terraform, dir := prepareTerraformQueryFixture(t)
	var unexpected atomic.Bool
	var aliasDetails atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var value any
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/login":
			value = map[string]string{"token": "test-token"}
		case r.Method == http.MethodGet && r.URL.Path == "/api/device/pfsense/localhost/api/aliases":
			value = map[string]any{"aliases": []api.FWAlias{{Name: "example"}}}
		case r.Method == http.MethodGet && r.URL.Path == "/api/device/pfsense/localhost/api/aliases/example":
			aliasDetails.Add(1)
			kind := api.FWAliasType("host")
			value = api.FWAlias{Name: "example", Type: &kind}
		case r.Method == http.MethodGet && r.URL.Path == "/api/device/pfsense/localhost/api/firewall/nat/npt":
			address := map[string]any{"address": "example.test", "type": "network"}
			value = map[string]any{"rules": []any{map[string]any{"id": "npt-1", "source": address, "destination": address}}}
		case r.Method == http.MethodGet && r.URL.Path == "/api/device/pfsense/localhost/api/firewall/nat/onetoone":
			address := map[string]any{"address": "example.test", "type": "network"}
			value = map[string]any{"rules": []any{map[string]any{"id": "one-1", "source": address, "destination": address, "external": address}}}
		case r.Method == http.MethodGet && r.URL.Path == "/api/device/pfsense/localhost/api/firewall/nat/port-forward":
			value = map[string]any{"rules": []any{map[string]any{"rule": map[string]any{"id": "port-1"}}}}
		case r.Method == http.MethodGet && r.URL.Path == "/api/device/pfsense/localhost/api/firewall/rules/interface":
			value = map[string]any{"interfaces": []string{"lan"}}
		case r.Method == http.MethodGet && r.URL.Path == "/api/device/pfsense/localhost/api/firewall/rules/interface/lan":
			value = map[string]any{"entries": []any{
				map[string]any{"rule": map[string]any{"id": "rule-1", "interface": "lan"}},
				map[string]any{"rule": map[string]any{"id": "system-rule", "readonly": true}},
			}}
		case r.Method == http.MethodGet && r.URL.Path == "/api/device/pfsense/localhost/api/firewall/schedules":
			value = map[string]any{"schedules": []any{map[string]any{"id": "schedule-1", "name": "weekdays"}}}
		case r.Method == http.MethodGet && r.URL.Path == "/api/device/pfsense/localhost/api/firewall/traffic-shaper/altq":
			value = map[string]any{"altq": []any{map[string]any{"name": "queue-1", "if_ident": "lan", "scheduler": "HFSC", "bandwidth": "100", "bandwidthtype": "Mb"}}}
		case r.Method == http.MethodGet && r.URL.Path == "/api/device/pfsense/localhost/api/firewall/traffic-shaper/limiter":
			value = map[string]any{"limiter": []any{map[string]any{"name": "limiter-1"}}}
		case r.Method == http.MethodGet && r.URL.Path == "/api/device/pfsense/localhost/api/firewall/pflow":
			value = map[string]any{"exporters": []any{map[string]any{"id": "exporter-1"}}}
		case r.Method == http.MethodGet && r.URL.Path == "/api/device/pfsense/localhost/api/firewall/virtualip":
			value = map[string]any{"virtualips": []any{map[string]any{"id": "vip-1"}}}
		case r.Method == http.MethodGet && r.URL.Path == "/api/device/pfsense/localhost/api/vpn/wg":
			value = map[string]any{
				"config":  []any{map[string]any{"enable": true}},
				"tunnels": map[string]any{"item": []any{map[string]any{"name": "wg0", "publickey": "public-key-example", "privatekey": strings.Repeat("x", 12)}}},
				"peers":   map[string]any{"item": []any{map[string]any{"publickey": "peer-key-example", "presharedkey": strings.Repeat("x", 12)}}},
			}
		default:
			unexpected.Store(true)
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		if err := json.NewEncoder(w).Encode(value); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()

	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(terraform, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"TF_CLI_CONFIG_FILE="+filepath.Join(dir, "terraform.tfrc"),
			"TF_IN_AUTOMATION=1",
			"TF_INPUT=0",
			"TF_VAR_nexus_url="+server.URL,
			"TF_VAR_nexus_username=test-user",
			"TF_VAR_nexus_password="+strings.Repeat("x", 12),
		)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("terraform %s: %v\n%s", strings.Join(args, " "), err, output)
		}
		return string(output)
	}
	run("init", "-input=false")
	run("query")
	if aliasDetails.Load() != 0 {
		t.Fatal("identity-only query unexpectedly fetched alias details")
	}
	queryOutput := run("query", "-generate-config-out=generated.tf")
	if aliasDetails.Load() != 1 {
		t.Fatalf("configuration generation fetched alias details %d times", aliasDetails.Load())
	}
	if !strings.Contains(queryOutput, "Read-only firewall rules omitted") {
		t.Fatal("query did not report the omitted read-only rule")
	}
	data, err := os.ReadFile(filepath.Join(dir, "generated.tf"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"firewall_alias", "firewall_nat_npt", "firewall_nat_one_to_one", "firewall_nat_port_forward",
		"firewall_rule", "firewall_schedule", "firewall_altq", "firewall_limiter", "firewall_pflow_exporter",
		"firewall_virtual_ip", "wireguard_settings", "wireguard_tunnel", "wireguard_peer",
	} {
		if !strings.Contains(string(data), `resource "pfsense_`+name+`"`) {
			t.Errorf("generated configuration omitted resource type %s", name)
		}
	}
	if strings.Count(string(data), "import {") != 13 {
		t.Fatal("generated configuration did not include thirteen import blocks")
	}
	if strings.Contains(string(data), strings.Repeat("x", 12)) ||
		strings.Count(string(data), "null # sensitive") != 2 {
		t.Fatal("generated configuration did not redact the mock WireGuard secrets")
	}
	planOutput := run("plan", "-input=false", "-no-color")
	if !strings.Contains(planOutput, "Plan: 13 to import, 0 to add, 0 to change, 0 to destroy.") {
		t.Fatal("generated configuration did not produce an import-only plan")
	}
	if unexpected.Load() {
		t.Fatal("Terraform sent an unexpected request")
	}
	if _, err := os.Stat(filepath.Join(dir, "terraform.tfstate")); !os.IsNotExist(err) {
		t.Fatalf("query unexpectedly created Terraform state: %v", err)
	}
}
