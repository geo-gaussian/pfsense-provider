package main

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("../../tests/sizecheck/_fixtures", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestInspectCountsCharactersAndReportsLines(t *testing.T) {
	source := strings.Repeat("é", 120) + "\r\n" + strings.Repeat("x", 120) + "\t\n"
	findings, err := inspect("sample.ts", []byte(source))
	if err != nil || len(findings) != 1 || findings[0] != (finding{"sample.ts", 2, 121}) {
		t.Fatalf("findings=%+v err=%v", findings, err)
	}
	found, err := inspect("sample.ts", []byte(strings.Repeat("x", 120)+"\r"))
	if err != nil || len(found) != 1 || found[0].size != 121 {
		t.Fatalf("bare carriage return: findings=%+v err=%v", found, err)
	}
	if _, err := inspect("sample.ts", []byte{0xff}); err == nil {
		t.Fatal("invalid UTF-8 must fail")
	}
}

func TestGeneratedAndExcludedFiles(t *testing.T) {
	for _, name := range []string{"generated.go", "generated.ts"} {
		source := append(fixture(t, name), strings.Repeat("// x", 121)...)
		found, err := inspect(name, source)
		if err != nil || len(found) != 0 {
			t.Fatalf("%s: findings=%+v err=%v", name, found, err)
		}
	}
	shell := append(fixture(t, "generated.ts"), strings.Repeat("x", 121)...)
	if found, err := inspect("sample.sh", shell); err != nil || len(found) != 1 {
		t.Fatalf("shell source: findings=%+v err=%v", found, err)
	}
	if _, err := inspect("invalid.go", fixture(t, "invalid.txt")); err == nil {
		t.Fatal("invalid Go must fail")
	}
	for _, name := range []string{
		"tools/coverage/evidence.go", "tests/fixtures/sample.go", "tests/format/sample.ts",
		".specify/scripts/bash/common.sh", "README.md",
	} {
		if sourceFile(name) {
			t.Errorf("%s must be excluded", name)
		}
	}
	for _, name := range []string{
		"sample.go", "sample.test.ts", "sample.mjs", "sample.sh", "sample.jq", ".githooks/pre-commit",
	} {
		if !sourceFile(name) {
			t.Errorf("%s must be checked", name)
		}
	}
}

func TestStagedContentsAndFailures(t *testing.T) {
	source := fixture(t, "sample.go")
	invalid := fixture(t, "invalid.txt")
	dir := t.TempDir()
	t.Chdir(dir)
	if output, err := exec.Command("git", "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("initialise repository: %v: %s", err, output)
	}
	oversized := append(source, []byte("// "+strings.Repeat("x", 118)+"\n")...)
	if err := os.WriteFile("sample.go", oversized, 0600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("git", "add", "sample.go").CombinedOutput(); err != nil {
		t.Fatalf("stage fixture: %v: %s", err, output)
	}
	if err := os.WriteFile("sample.go", source, 0600); err != nil {
		t.Fatal(err)
	}
	if found, err := checkFiles([]string{"sample.go"}, false); err != nil || len(found) != 0 {
		t.Fatalf("working tree: findings=%+v err=%v", found, err)
	}
	if found, err := checkFiles([]string{"sample.go"}, true); err != nil || len(found) != 1 {
		t.Fatalf("index: findings=%+v err=%v", found, err)
	}
	if err := run(true); err == nil {
		t.Fatal("staged long line must fail")
	}
	if _, err := checkFiles([]string{"absent.go"}, true); err == nil {
		t.Fatal("missing staged source must fail")
	}
	if found, err := checkFiles([]string{"deleted.go"}, false); err != nil || len(found) != 0 {
		t.Fatalf("deleted working-tree file: findings=%+v err=%v", found, err)
	}
	if _, err := checkFiles([]string{"bad\x00.go"}, false); err == nil {
		t.Fatal("unreadable working-tree source must fail")
	}
	if err := os.WriteFile("invalid.go", invalid, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := checkFiles([]string{"invalid.go"}, false); err == nil {
		t.Fatal("invalid source must fail")
	}
	if err := run(false); err == nil {
		t.Fatal("invalid repository source must fail")
	}
}

func TestFormatFinding(t *testing.T) {
	item := finding{"path/a%,:\n.go", 2, 121}
	want := "::error file=path/a%25%2C%3A%0A.go,line=2::line has 121 characters (limit 120)"
	if got := formatFinding(item, true); got != want {
		t.Fatalf("unexpected annotation %q", got)
	}
	want = "sample.go:3: error: line has 121 characters (limit 120)"
	if got := formatFinding(finding{"sample.go", 3, 121}, false); got != want {
		t.Fatalf("unexpected local output %q", got)
	}
}

func TestRunReportsGitFailure(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if err := run(false); err == nil {
		t.Fatal("missing git must fail")
	}
}

func TestMainCommandSuccess(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if output, err := exec.Command("git", "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("initialise repository: %v: %s", err, output)
	}
	oldArgs, oldFlags := os.Args, flag.CommandLine
	t.Cleanup(func() { os.Args, flag.CommandLine = oldArgs, oldFlags })
	os.Args = []string{"linecheck"}
	flag.CommandLine = flag.NewFlagSet("linecheck", flag.ContinueOnError)
	main()
}

func TestMainCommandFailure(t *testing.T) {
	if os.Getenv("LINECHECK_TEST_CHILD") == "1" {
		os.Args = []string{"linecheck"}
		flag.CommandLine = flag.NewFlagSet("linecheck", flag.ContinueOnError)
		main()
		return
	}
	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestMainCommandFailure$")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "LINECHECK_TEST_CHILD=1")
	output, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "list repository files") {
		t.Fatalf("expected command failure, got %v: %s", err, output)
	}
}
