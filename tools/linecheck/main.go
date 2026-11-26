package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"unicode/utf8"
)

const limit = 120

var generatedJS = regexp.MustCompile(`^\s*// Code generated .* DO NOT EDIT\.(\r?\n|$)`)

type finding struct {
	file       string
	line, size int
}

func sourceFile(name string) bool {
	for _, prefix := range []string{
		"tests/commands/", "tests/fixtures/", "tests/format/", "tests/terraform/", "tests/sizecheck/_fixtures/",
		".specify/scripts/",
	} {
		if strings.HasPrefix(name, prefix) {
			return false
		}
	}
	if name == "tools/coverage/evidence.go" {
		return false
	}
	for _, suffix := range []string{
		".go", ".sh", ".jq", ".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx", ".mts", ".cts",
	} {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return strings.HasPrefix(name, ".githooks/")
}

func generated(name string, data []byte) (bool, error) {
	if strings.HasSuffix(name, ".go") {
		file, err := parser.ParseFile(token.NewFileSet(), name, data, parser.ParseComments)
		if err != nil {
			return false, fmt.Errorf("parse %s: %w", name, err)
		}
		return ast.IsGenerated(file), nil
	}
	for _, suffix := range []string{".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx", ".mts", ".cts"} {
		if strings.HasSuffix(name, suffix) {
			return generatedJS.Match(data), nil
		}
	}
	return false, nil
}

func inspect(name string, data []byte) ([]finding, error) {
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("%s is not valid UTF-8", name)
	}
	skip, err := generated(name, data)
	if err != nil || skip {
		return nil, err
	}
	var result []finding
	lines := bytes.Split(data, []byte{'\n'})
	for index, line := range lines {
		if index < len(lines)-1 {
			line = bytes.TrimSuffix(line, []byte{'\r'})
		}
		if size := utf8.RuneCount(line); size > limit {
			result = append(result, finding{name, index + 1, size})
		}
	}
	return result, nil
}

func checkFiles(names []string, staged bool) ([]finding, error) {
	var all []finding
	for _, name := range names {
		if !sourceFile(name) {
			continue
		}
		var data []byte
		var err error
		if staged {
			data, err = exec.Command("git", "show", ":"+name).Output()
		} else {
			data, err = os.ReadFile(name)
		}
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		found, err := inspect(name, data)
		if err != nil {
			return nil, err
		}
		all = append(all, found...)
	}
	return all, nil
}

func formatFinding(item finding, github bool) string {
	message := fmt.Sprintf("line has %d characters (limit %d)", item.size, limit)
	if !github {
		return fmt.Sprintf("%s:%d: error: %s", item.file, item.line, message)
	}
	escape := strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A", ":", "%3A", ",", "%2C")
	return fmt.Sprintf("::error file=%s,line=%d::%s", escape.Replace(item.file), item.line, message)
}

func run(staged bool) error {
	args := []string{"ls-files", "-co", "--exclude-standard", "-z"}
	if staged {
		args = []string{"ls-files", "-z"}
	}
	output, err := exec.Command("git", args...).Output()
	if err != nil {
		return fmt.Errorf("list repository files: %w", err)
	}
	findings, err := checkFiles(strings.Split(string(output), "\x00"), staged)
	if err != nil {
		return err
	}
	for _, item := range findings {
		fmt.Println(formatFinding(item, os.Getenv("GITHUB_ACTIONS") == "true"))
	}
	if len(findings) > 0 {
		return fmt.Errorf("line length limit exceeded")
	}
	fmt.Println("Line length checks passed.")
	return nil
}

func main() {
	staged := flag.Bool("staged", false, "inspect the Git index instead of the working tree")
	flag.Parse()
	if err := run(*staged); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
