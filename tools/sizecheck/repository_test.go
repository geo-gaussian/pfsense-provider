package main

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestChecksStagedContentsRatherThanWorkingTree(t *testing.T) {
	source := fixture(t, "sample.go")
	invalid := fixture(t, "invalid.txt")
	dir := t.TempDir()
	t.Chdir(dir)
	if output, err := exec.Command("git", "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("initialise repository: %v: %s", err, output)
	}
	file := filepath.Join(dir, "sample.go")
	oversized := append(source, strings.Repeat("\n", 501-bytesLines(source))...)
	if err := os.WriteFile(file, oversized, 0600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("git", "add", "sample.go").CombinedOutput(); err != nil {
		t.Fatalf("stage fixture: %v: %s", err, output)
	}
	if err := os.WriteFile(file, source, 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(false); err != nil {
		t.Fatalf("working tree should pass: %v", err)
	}
	if err := run(true); err == nil {
		t.Fatal("staged oversized file must fail")
	}
	if _, err := checkFiles([]string{"absent.go"}, true); err == nil {
		t.Fatal("missing staged source must fail")
	}
	if err := os.WriteFile(file, invalid, 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(false); err == nil {
		t.Fatal("invalid working tree source must fail")
	}
}

func TestCheckFilesReportsParseAndReadFailures(t *testing.T) {
	source := fixture(t, "invalid.txt")
	shell := fixture(t, "sample.sh")
	filter, err := os.ReadFile("../../scripts/current-commit-sha.jq")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile("bad.go", source, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := checkFiles([]string{"bad.go"}, false); err == nil {
		t.Fatal("invalid source must fail")
	}
	if _, err := checkFiles([]string{"bad\x00.go"}, false); err == nil {
		t.Fatal("unreadable source must fail")
	}
	findings, err := checkFiles([]string{"missing.go", "tests/format/fixture.go", "README.md"}, false)
	if err != nil || len(findings) != 0 {
		t.Fatalf("excluded or deleted paths: findings=%v err=%v", findings, err)
	}
	if err := os.WriteFile("check.sh", shell, 0600); err != nil {
		t.Fatal(err)
	}
	findings, err = checkFiles([]string{"check.sh"}, false)
	if err != nil || len(findings) != 0 {
		t.Fatalf("valid shell source: findings=%v err=%v", findings, err)
	}
	if err := os.WriteFile("check.jq", append(filter, strings.Repeat("\n", 500)...), 0600); err != nil {
		t.Fatal(err)
	}
	findings, err = checkFiles([]string{"check.jq"}, false)
	if err != nil || len(findings) != 1 || findings[0].level != "error" {
		t.Fatalf("oversized jq filter: findings=%v err=%v", findings, err)
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
	os.Args = []string{"sizecheck"}
	flag.CommandLine = flag.NewFlagSet("sizecheck", flag.ContinueOnError)
	main()
}

func TestMainCommandFailure(t *testing.T) {
	if os.Getenv("SIZECHECK_TEST_CHILD") == "1" {
		os.Args = []string{"sizecheck"}
		flag.CommandLine = flag.NewFlagSet("sizecheck", flag.ContinueOnError)
		main()
		return
	}
	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestMainCommandFailure$")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "SIZECHECK_TEST_CHILD=1")
	output, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "list repository files") {
		t.Fatalf("expected command failure, got %v: %s", err, output)
	}
}
