package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
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
	result, matched, err := collectOperations(spec.Paths, mappings)
	if err != nil {
		return nil, err
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

func collectOperations(
	paths map[string]map[string]yaml.Node, mappings map[string]evidence,
) ([]operation, map[string]bool, error) {
	var result []operation
	seen := map[string]bool{}
	matched := map[string]bool{}
	for path, items := range paths {
		if _, ok := items["$ref"]; ok {
			return nil, nil, fmt.Errorf("path reference is not supported: %s", path)
		}
		for method, node := range items {
			if !isHTTPMethod(method) {
				continue
			}
			op, err := collectOperation(method, path, node, mappings, seen, matched)
			if err != nil {
				return nil, nil, err
			}
			result = append(result, op)
		}
	}
	return result, matched, nil
}

func isHTTPMethod(method string) bool {
	switch method {
	case "get", "put", "post", "delete", "options", "head", "patch", "trace":
		return true
	}
	return false
}

func collectOperation(
	method, path string, node yaml.Node, mappings map[string]evidence, seen, matched map[string]bool,
) (operation, error) {
	var details struct {
		ID string `yaml:"operationId"`
	}
	if err := node.Decode(&details); err != nil {
		return operation{}, fmt.Errorf("%s %s: %w", method, path, err)
	}
	if details.ID != "" && seen[details.ID] {
		return operation{}, fmt.Errorf("duplicate operation ID %q", details.ID)
	}
	if details.ID != "" {
		seen[details.ID] = true
	}
	op := operation{Method: strings.ToUpper(method), Path: path, ID: details.ID}
	if entry, ok := mappings[op.ID]; ok {
		if op.Method != entry.Method || op.Path != entry.Path {
			return operation{}, fmt.Errorf("mapping for %s does not match %s %s", op.ID, op.Method, op.Path)
		}
		op.Supported, op.Evidence = true, entry
		matched[op.ID] = true
	}
	return op, nil
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
	profile, covered, total, err := readProfile(os.Args[1])
	if err != nil {
		return err
	}
	dir := os.Args[2]
	if err := writeUnitProfile(dir, profile); err != nil {
		return err
	}
	unit, api, err := writeCoverageBadges(dir, covered, total, len(ops))
	if err != nil {
		return err
	}
	if err := writeAPIReport(dir, ops, api); err != nil {
		return err
	}
	printCoverage(covered, total, unit, api, len(ops))
	return nil
}

func printCoverage(covered, total int64, unit, api badge, operations int) {
	fmt.Printf(
		"Unit statements: %d/%d (%s); Nexus operations: %d/%d (%s); acceptance: unavailable\n",
		covered,
		total,
		unit.Message,
		len(implemented),
		operations,
		api.Message,
	)
}

func writeCoverageBadges(dir string, covered, total int64, operations int) (badge, badge, error) {
	unit, err := coverageBadge("unit", covered, total)
	if err != nil {
		return badge{}, badge{}, err
	}
	api, err := coverageBadge("Nexus API", int64(len(implemented)), int64(operations))
	if err != nil {
		return badge{}, badge{}, err
	}
	acceptance, err := coverageBadge("acceptance", 0, 0)
	if err != nil {
		return badge{}, badge{}, err
	}
	badgeDir := filepath.Join(dir, "badges")
	if err := os.MkdirAll(badgeDir, 0o755); err != nil {
		return badge{}, badge{}, err
	}
	for name, value := range map[string]badge{"unit.json": unit, "api.json": api, "acceptance.json": acceptance} {
		if err := writeBadge(filepath.Join(badgeDir, name), value); err != nil {
			return badge{}, badge{}, err
		}
	}
	return unit, api, nil
}

func writeAPIReport(dir string, ops []operation, api badge) error {
	var report strings.Builder
	fmt.Fprintf(
		&report,
		"# 📊 Nexus API operation coverage\n\nImplemented: %d/%d (%s). "+
			"Mock-tested means an offline test exists; manually observed is not live acceptance coverage.\n\n"+
			"| Method | Path | Operation ID | Implemented | Mock test | Live evidence |\n"+
			"| --- | --- | --- | --- | --- | --- |\n",
		len(implemented), len(ops), api.Message,
	)
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
	return os.WriteFile(filepath.Join(dir, "api-report.md"), []byte(report.String()), 0o644)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
