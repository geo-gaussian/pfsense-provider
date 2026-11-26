package main

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func readProfile(path string) ([]byte, int64, int64, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, 0, err
	}
	return filterProfile(raw)
}

func writeUnitProfile(dir string, profile []byte) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "unit.out"), profile, 0o644)
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
		num, count, err := profileLineCounts(line, generated)
		if err != nil {
			return nil, 0, 0, err
		}
		if num == 0 {
			continue
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

func profileLineCounts(line string, generated map[string]bool) (int64, int64, error) {
	fields := strings.Fields(line)
	if len(fields) != 3 {
		return 0, 0, fmt.Errorf("invalid profile line %q", line)
	}
	pos := strings.LastIndex(fields[0], ":")
	if pos < 0 || !strings.HasPrefix(fields[0], module) {
		return 0, 0, fmt.Errorf("unexpected profile path %q", fields[0])
	}
	path := strings.TrimPrefix(fields[0][:pos], module)
	if strings.HasPrefix(path, "tools/coverage/") {
		return 0, 0, nil
	}
	isGenerated, ok := generated[path]
	if !ok {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.FromSlash(path), nil, parser.ParseComments)
		if err != nil {
			return 0, 0, fmt.Errorf("parse coverage source %s: %w", path, err)
		}
		isGenerated = ast.IsGenerated(file)
		generated[path] = isGenerated
	}
	if isGenerated {
		return 0, 0, nil
	}
	num, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil || num < 0 {
		return 0, 0, fmt.Errorf("invalid statement count in %q", line)
	}
	count, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil || count < 0 {
		return 0, 0, fmt.Errorf("invalid execution count in %q", line)
	}
	return num, count, nil
}
