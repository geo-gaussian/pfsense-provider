package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const module = "github.com/geo-gaussian/terraform-provider-pfsense-nexus/"

type evidence struct {
	Method string
	Path   string
	Source string
	Test   string
	Live   string
}

// Generated methods alone do not constitute provider support.
var implemented = map[string]evidence{
	"Login":               {"POST", "/login", "internal/client/client.go", "internal/client/client_test.go", "manually observed"},
	"RefreshAccessToken":  {"POST", "/login/refresh", "internal/client/client.go", "internal/client/client_test.go", ""},
	"FirewallGetAliases":  {"GET", "/aliases", "internal/provider/data_firewall_aliases/firewall_aliases_data_source.go", "internal/provider/data_firewall_aliases/firewall_aliases_data_source_test.go", "manually observed"},
	"FirewallAddAlias":    {"PUT", "/aliases", "internal/provider/resource_firewall_alias/firewall_alias_resource.go", "internal/provider/resource_firewall_alias/firewall_alias_resource_test.go", ""},
	"FirewallGetAlias":    {"GET", "/aliases/{id}", "internal/provider/resource_firewall_alias/firewall_alias_resource.go", "internal/provider/resource_firewall_alias/firewall_alias_resource_test.go", ""},
	"FirewallUpdateAlias": {"POST", "/aliases/{id}", "internal/provider/resource_firewall_alias/firewall_alias_resource.go", "internal/provider/resource_firewall_alias/firewall_alias_resource_test.go", ""},
	"FirewallDeleteAlias": {"DELETE", "/aliases/{id}", "internal/provider/resource_firewall_alias/firewall_alias_resource.go", "internal/provider/resource_firewall_alias/firewall_alias_resource_test.go", ""},
	"DoFilterReload":      {"POST", "/system/filterreload/reload", "internal/provider/resource_firewall_alias/firewall_alias_resource.go", "internal/provider/resource_firewall_alias/firewall_alias_resource_test.go", ""},
}

type operation struct {
	Method    string
	Path      string
	ID        string
	Evidence  evidence
	Supported bool
}

type badge struct {
	SchemaVersion int    `json:"schemaVersion"`
	Label         string `json:"label"`
	Message       string `json:"message"`
	Color         string `json:"color"`
}

func coverageBadge(label string, covered, total int64) (badge, error) {
	b := badge{SchemaVersion: 1, Label: label, Message: "unavailable", Color: "lightgrey"}
	if covered < 0 || total < 0 || covered > total {
		return b, fmt.Errorf("invalid coverage: %d of %d", covered, total)
	}
	if total == 0 {
		return b, nil
	}
	basis := new(big.Int).Mul(big.NewInt(covered), big.NewInt(10000))
	basis.Quo(basis, big.NewInt(total))
	points := basis.Int64()
	b.Message = fmt.Sprintf("%d.%02d%%", points/100, points%100)
	switch {
	case points >= 9500:
		b.Color = "brightgreen"
	case points >= 8000:
		b.Color = "yellowgreen"
	case points >= 7000:
		b.Color = "yellow"
	case points >= 5000:
		b.Color = "orange"
	default:
		b.Color = "red"
	}
	return b, nil
}

func inventory(data []byte, mappings map[string]evidence) ([]operation, error) {
	var spec struct {
		Paths map[string]map[string]yaml.Node `yaml:"paths"`
	}
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parse specification: %w", err)
	}
	if len(spec.Paths) == 0 {
		return nil, fmt.Errorf("specification has no paths")
	}
	methods := map[string]bool{"get": true, "put": true, "post": true, "delete": true, "options": true, "head": true, "patch": true, "trace": true}
	var result []operation
	seen := map[string]bool{}
	matched := map[string]bool{}
	for path, items := range spec.Paths {
		if _, ok := items["$ref"]; ok {
			return nil, fmt.Errorf("path reference is not supported: %s", path)
		}
		for method, node := range items {
			if !methods[method] {
				continue
			}
			var details struct {
				ID string `yaml:"operationId"`
			}
			if err := node.Decode(&details); err != nil {
				return nil, fmt.Errorf("%s %s: %w", method, path, err)
			}
			if details.ID != "" && seen[details.ID] {
				return nil, fmt.Errorf("duplicate operation ID %q", details.ID)
			}
			if details.ID != "" {
				seen[details.ID] = true
			}
			op := operation{Method: strings.ToUpper(method), Path: path, ID: details.ID}
			if entry, ok := mappings[op.ID]; ok {
				if op.Method != entry.Method || op.Path != entry.Path {
					return nil, fmt.Errorf("mapping for %s does not match %s %s", op.ID, op.Method, op.Path)
				}
				op.Supported, op.Evidence = true, entry
				matched[op.ID] = true
			}
			result = append(result, op)
		}
	}
	for id := range mappings {
		if !matched[id] {
			return nil, fmt.Errorf("mapping %s is absent from specification", id)
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("specification has no operations")
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Path != result[j].Path {
			return result[i].Path < result[j].Path
		}
		return result[i].Method < result[j].Method
	})
	return result, nil
}

func validateEvidence(mappings map[string]evidence) error {
	for id, entry := range mappings {
		source, err := os.ReadFile(entry.Source)
		if err != nil {
			return fmt.Errorf("%s source: %w", id, err)
		}
		if !strings.Contains(string(source), "."+id+"WithResponse(") {
			return fmt.Errorf("%s: provider call not found in %s", id, entry.Source)
		}
		if _, err := os.Stat(entry.Test); err != nil {
			return fmt.Errorf("%s test: %w", id, err)
		}
	}
	return nil
}

func filterProfile(data []byte) ([]byte, int64, int64, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	if !scanner.Scan() || !strings.HasPrefix(scanner.Text(), "mode: ") {
		return nil, 0, 0, fmt.Errorf("invalid coverage profile header")
	}
	var out bytes.Buffer
	fmt.Fprintln(&out, scanner.Text())
	generated := map[string]bool{}
	var covered, total int64
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) != 3 {
			return nil, 0, 0, fmt.Errorf("invalid profile line %q", line)
		}
		pos := strings.LastIndex(fields[0], ":")
		if pos < 0 || !strings.HasPrefix(fields[0], module) {
			return nil, 0, 0, fmt.Errorf("unexpected profile path %q", fields[0])
		}
		path := strings.TrimPrefix(fields[0][:pos], module)
		if strings.HasPrefix(path, "tools/coverage/") {
			continue
		}
		isGenerated, ok := generated[path]
		if !ok {
			file, err := parser.ParseFile(token.NewFileSet(), filepath.FromSlash(path), nil, parser.ParseComments)
			if err != nil {
				return nil, 0, 0, fmt.Errorf("parse coverage source %s: %w", path, err)
			}
			isGenerated = ast.IsGenerated(file)
			generated[path] = isGenerated
		}
		if isGenerated {
			continue
		}
		num, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil || num <= 0 {
			return nil, 0, 0, fmt.Errorf("invalid statement count in %q", line)
		}
		count, err := strconv.ParseInt(fields[2], 10, 64)
		if err != nil || count < 0 {
			return nil, 0, 0, fmt.Errorf("invalid execution count in %q", line)
		}
		total += num
		if count > 0 {
			covered += num
		}
		fmt.Fprintln(&out, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, 0, err
	}
	if total == 0 {
		return nil, 0, 0, fmt.Errorf("coverage profile contains no eligible statements")
	}
	return out.Bytes(), covered, total, nil
}

func writeBadge(path string, value badge) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func run() error {
	if len(os.Args) != 3 {
		return fmt.Errorf("usage: go run ./tools/coverage RAW_PROFILE OUTPUT_DIRECTORY")
	}
	if err := validateEvidence(implemented); err != nil {
		return err
	}
	spec, err := os.ReadFile("docs/nexus.openapi.yaml")
	if err != nil {
		return err
	}
	ops, err := inventory(spec, implemented)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		return err
	}
	profile, covered, total, err := filterProfile(raw)
	if err != nil {
		return err
	}
	dir := os.Args[2]
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "unit.out"), profile, 0o644); err != nil {
		return err
	}
	unit, err := coverageBadge("unit", covered, total)
	if err != nil {
		return err
	}
	api, err := coverageBadge("Nexus API", int64(len(implemented)), int64(len(ops)))
	if err != nil {
		return err
	}
	acceptance, err := coverageBadge("acceptance", 0, 0)
	if err != nil {
		return err
	}
	badgeDir := filepath.Join(dir, "badges")
	if err := os.MkdirAll(badgeDir, 0o755); err != nil {
		return err
	}
	for name, value := range map[string]badge{"unit.json": unit, "api.json": api, "acceptance.json": acceptance} {
		if err := writeBadge(filepath.Join(badgeDir, name), value); err != nil {
			return err
		}
	}
	var report strings.Builder
	fmt.Fprintf(&report, "# 📊 Nexus API operation coverage\n\nImplemented: %d/%d (%s). Mock-tested means an offline test exists; manually observed is not live acceptance coverage.\n\n| Method | Path | Operation ID | Implemented | Mock test | Live evidence |\n| --- | --- | --- | --- | --- | --- |\n", len(implemented), len(ops), api.Message)
	for _, op := range ops {
		source, test, live := "-", "-", "-"
		if op.Supported {
			source, test = op.Evidence.Source, op.Evidence.Test
			if op.Evidence.Live != "" {
				live = op.Evidence.Live
			}
		}
		fmt.Fprintf(&report, "| %s | `%s` | %s | %s | %s | %s |\n", op.Method, op.Path, op.ID, source, test, live)
	}
	if err := os.WriteFile(filepath.Join(dir, "api-report.md"), []byte(report.String()), 0o644); err != nil {
		return err
	}
	fmt.Printf("Unit statements: %d/%d (%s); Nexus operations: %d/%d (%s); acceptance: unavailable\n", covered, total, unit.Message, len(implemented), len(ops), api.Message)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
