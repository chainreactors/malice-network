package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var skipDirs = map[string]struct{}{
	".claude":  {},
	".git":     {},
	".idea":    {},
	".malice":  {},
	"bin":      {},
	"dist":     {},
	"external": {},
}

func discoverTaggedPackages(root, layer string) ([]string, error) {
	packages := make(map[string]struct{})

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if _, skip := skipDirs[d.Name()]; skip && sameDir(path, root) == false {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		expr, err := readBuildExpr(path)
		if err != nil {
			return err
		}
		if !buildExprContains(expr, layer) {
			return nil
		}

		relDir, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return fmt.Errorf("resolve relative package path for %s: %w", path, err)
		}
		packages[toPackagePattern(relDir)] = struct{}{}
		return nil
	})
	if err != nil {
		return nil, err
	}

	results := make([]string, 0, len(packages))
	for pkg := range packages {
		results = append(results, pkg)
	}
	sort.Strings(results)
	return results, nil
}

func readBuildExpr(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read build tags from %s: %w", path, err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "//go:build ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "//go:build ")), nil
		}
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		break
	}

	return "", nil
}

func buildExprContains(expr, tag string) bool {
	if expr == "" {
		return false
	}

	separators := func(r rune) bool {
		switch r {
		case ' ', '\t', '\r', '\n', '&', '|', '!', '(', ')':
			return true
		default:
			return false
		}
	}

	for _, token := range strings.FieldsFunc(expr, separators) {
		if token == tag {
			return true
		}
	}
	return false
}

func toPackagePattern(relDir string) string {
	if relDir == "." {
		return "."
	}
	return "./" + filepath.ToSlash(relDir)
}

func sameDir(left, right string) bool {
	leftClean := filepath.Clean(left)
	rightClean := filepath.Clean(right)
	return strings.EqualFold(leftClean, rightClean)
}
