package main

import (
	"os"
	"strings"
	"testing"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("../../tests/sizecheck/_fixtures/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestInspectGoLimits(t *testing.T) {
	source := string(fixture(t, "sample.go"))
	for _, tt := range []struct {
		name        string
		extra       int
		wantLevel   string
		wantSubject string
	}{
		{"at warning", 37, "", ""},
		{"above warning", 38, "warning", "function sample"},
		{"at error", 77, "warning", "function sample"},
		{"above error", 78, "error", "function sample"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			expanded := strings.Replace(source, "// padding", "// padding"+strings.Repeat("\n", tt.extra), 1)
			findings, err := inspectGo("example.go", []byte(expanded))
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantLevel == "" {
				if len(findings) != 0 {
					t.Fatalf("unexpected findings: %+v", findings)
				}
			} else if len(findings) != 1 || findings[0].level != tt.wantLevel || findings[0].subject != tt.wantSubject {
				t.Fatalf("unexpected findings: %+v", findings)
			}
		})
	}
}

func TestInspectGoFilesAndGenerated(t *testing.T) {
	source := fixture(t, "sample.go")
	for _, tt := range []struct {
		name  string
		lines int
		level string
	}{
		{"example.go", 350, ""},
		{"example.go", 351, "warning"},
		{"example.go", 501, "error"},
		{"example_test.go", 500, ""},
		{"example_test.go", 501, "warning"},
		{"example_test.go", 601, "error"},
	} {
		padding := strings.Repeat("\n", tt.lines-bytesLines(source))
		findings, err := inspectGo(tt.name, append(source, padding...))
		if err != nil {
			t.Fatal(err)
		}
		if tt.level == "" && len(findings) != 0 {
			t.Fatalf("%s: unexpected findings: %+v", tt.name, findings)
		}
		if tt.level != "" && (len(findings) != 1 || findings[0].level != tt.level || findings[0].subject != "file") {
			t.Fatalf("%s: unexpected findings: %+v", tt.name, findings)
		}
	}
	generated := append(fixture(t, "generated.go"), strings.Repeat("\n", 700)...)
	findings, err := inspectGo("generated.go", generated)
	if err != nil || len(findings) != 0 {
		t.Fatalf("generated file: findings=%+v err=%v", findings, err)
	}
	if _, err := inspectGo("invalid.go", fixture(t, "invalid.txt")); err == nil {
		t.Fatal("invalid Go must be reported")
	}
}

func TestInspectGoTestFunctionLimits(t *testing.T) {
	source := string(fixture(t, "sample.go"))
	for _, tt := range []struct {
		extra int
		level string
	}{
		{97, ""},
		{98, "warning"},
		{147, "warning"},
		{148, "error"},
	} {
		expanded := strings.Replace(source, "// padding", "// padding"+strings.Repeat("\n", tt.extra), 1)
		findings, err := inspectGo("sample_test.go", []byte(expanded))
		if err != nil {
			t.Fatal(err)
		}
		if tt.level == "" && len(findings) != 0 {
			t.Fatalf("unexpected findings: %+v", findings)
		}
		if tt.level != "" && (len(findings) != 1 || findings[0].level != tt.level) {
			t.Fatalf("unexpected findings: %+v", findings)
		}
	}
}

func TestInspectGoNestedFunctionsUsePhysicalPositions(t *testing.T) {
	source := string(fixture(t, "nested.go"))
	source = strings.Replace(source, "// padding", "// padding"+strings.Repeat("\n", 40), 1)
	findings, err := inspectGo("nested.go", []byte(strings.TrimSuffix(source, "\n")))
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 || findings[0].location != "nested.go:4" || findings[1].subject != "function literal" {
		t.Fatalf("unexpected findings: %+v", findings)
	}
}

func bytesLines(source []byte) int {
	return strings.Count(string(source), "\n")
}

func TestInspectShellFunctionLimits(t *testing.T) {
	source := string(fixture(t, "sample.sh"))
	for _, tt := range []struct {
		extra int
		level string
	}{
		{36, ""},
		{37, "warning"},
		{77, "error"},
	} {
		expanded := strings.Replace(source, "# padding", "# padding"+strings.Repeat("\n", tt.extra), 1)
		findings, err := inspectShell("sample.sh", []byte(expanded))
		if err != nil {
			t.Fatal(err)
		}
		if tt.level == "" && len(findings) != 0 {
			t.Fatalf("unexpected findings: %+v", findings)
		}
		if tt.level != "" && (len(findings) != 1 || findings[0].level != tt.level || findings[0].subject != "function sample") {
			t.Fatalf("unexpected findings: %+v", findings)
		}
	}
	if _, err := inspectShell("invalid.sh", fixture(t, "invalid.txt")); err == nil {
		t.Fatal("invalid shell must be reported")
	}
	longFile := append(fixture(t, "sample.sh"), strings.Repeat("\n", 495)...)
	findings, err := inspectShell("sample.sh", append(longFile, ' '))
	if err != nil || len(findings) != 1 || findings[0].level != "error" {
		t.Fatalf("long shell file: findings=%+v err=%v", findings, err)
	}
}
