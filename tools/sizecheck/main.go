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
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

type limits struct {
	fileWarning, fileError         int
	functionWarning, functionError int
}

type finding struct {
	level, location, subject string
	lines, limit             int
}

func isFixture(name string) bool {
	for _, prefix := range []string{
		"tests/commands/", "tests/format/", "tests/terraform/", "tests/sizecheck/_fixtures/", ".specify/scripts/",
	} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func threshold(lines, warning, hard int) (string, int) {
	if lines > hard {
		return "error", hard
	}
	if lines > warning {
		return "warning", warning
	}
	return "", 0
}

func fileFinding(name string, source []byte, warning, hard int) []finding {
	lines := bytes.Count(source, []byte{'\n'})
	if len(source) > 0 && source[len(source)-1] != '\n' {
		lines++
	}
	if level, limit := threshold(lines, warning, hard); level != "" {
		return []finding{{level, name, "file", lines, limit}}
	}
	return nil
}

func inspectGo(name string, source []byte) ([]finding, error) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, name, source, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", name, err)
	}
	if ast.IsGenerated(file) {
		return nil, nil
	}
	size := limits{350, 500, 40, 80}
	if strings.HasSuffix(name, "_test.go") {
		size = limits{500, 600, 100, 150}
	}
	findings := fileFinding(name, source, size.fileWarning, size.fileError)
	record := func(location, subject string, count, warning, hard int) {
		if level, limit := threshold(count, warning, hard); level != "" {
			findings = append(findings, finding{level, location, subject, count, limit})
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		var start, end token.Pos
		var subject string
		switch fn := node.(type) {
		case *ast.FuncDecl:
			if fn.Body == nil {
				return true
			}
			start, end, subject = fn.Pos(), fn.Body.End(), "function "+fn.Name.Name
		case *ast.FuncLit:
			start, end, subject = fn.Pos(), fn.Body.End(), "function literal"
		default:
			return true
		}
		first, last := fileSet.PositionFor(start, false).Line, fileSet.PositionFor(end, false).Line
		record(fmt.Sprintf("%s:%d", name, first), subject, last-first+1, size.functionWarning, size.functionError)
		return true
	})
	return findings, nil
}

func inspectShell(name string, source []byte) ([]finding, error) {
	file, err := syntax.NewParser(syntax.Variant(syntax.LangPOSIX)).Parse(bytes.NewReader(source), name)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", name, err)
	}
	size := limits{350, 500, 40, 80}
	findings := fileFinding(name, source, size.fileWarning, size.fileError)
	syntax.Walk(file, func(node syntax.Node) bool {
		fn, ok := node.(*syntax.FuncDecl)
		if !ok {
			return true
		}
		first, last := int(fn.Pos().Line()), int(fn.End().Line())
		count := last - first + 1
		if level, limit := threshold(count, size.functionWarning, size.functionError); level != "" {
			findings = append(
				findings,
				finding{level, fmt.Sprintf("%s:%d", name, first), "function " + fn.Name.Value, count, limit},
			)
		}
		return true
	})
	return findings, nil
}

func checkFiles(names []string, staged bool) ([]finding, error) {
	var all []finding
	for _, name := range names {
		isGo := strings.HasSuffix(name, ".go")
		isShell := strings.HasSuffix(name, ".sh") || strings.HasPrefix(name, ".githooks/")
		isJQ := strings.HasSuffix(name, ".jq")
		if (!isGo && !isShell && !isJQ) || isFixture(name) {
			continue
		}
		var source []byte
		var err error
		if staged {
			source, err = exec.Command("git", "show", ":"+name).Output()
		} else {
			source, err = os.ReadFile(name)
		}
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		var findings []finding
		if isGo {
			findings, err = inspectGo(name, source)
		} else if isShell {
			findings, err = inspectShell(name, source)
		} else {
			findings = fileFinding(name, source, 350, 500)
		}
		if err != nil {
			return nil, err
		}
		all = append(all, findings...)
	}
	return all, nil
}

func formatFinding(item finding, githubActions bool) string {
	message := fmt.Sprintf("%s has %d lines (limit %d)", item.subject, item.lines, item.limit)
	if !githubActions {
		return fmt.Sprintf("%s: %s: %s", item.location, item.level, message)
	}
	name, line := item.location, ""
	if item.subject != "file" {
		index := strings.LastIndex(name, ":")
		name, line = name[:index], ",line="+name[index+1:]
	}
	propertyEscaper := strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A", ":", "%3A", ",", "%2C")
	messageEscaper := strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A")
	return fmt.Sprintf(
		"::%s file=%s%s::%s",
		item.level,
		propertyEscaper.Replace(name),
		line,
		messageEscaper.Replace(message),
	)
}

func run(staged bool) error {
	args := []string{"ls-files", "-co", "--exclude-standard", "-z"}
	if staged {
		args = []string{"ls-files", "-z"}
	}
	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("list repository files: %w", err)
	}
	findings, err := checkFiles(strings.Split(string(output), "\x00"), staged)
	if err != nil {
		return err
	}
	failed := false
	for _, item := range findings {
		fmt.Println(formatFinding(item, os.Getenv("GITHUB_ACTIONS") == "true"))
		failed = failed || item.level == "error"
	}
	if failed {
		return fmt.Errorf("Go, shell, or jq size limits exceeded")
	}
	fmt.Println("Go, shell, and jq file size checks passed.")
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
