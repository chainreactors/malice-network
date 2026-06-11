package main

import (
	"path/filepath"
	"sort"
	"strings"
)

func normalizePath(path string) string {
	return filepath.ToSlash(path)
}

func mustRel(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}

func uniqueSorted(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		set[value] = struct{}{}
	}
	return sortedSet(set)
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedSet(values map[string]struct{}) []string {
	items := make([]string, 0, len(values))
	for value := range values {
		items = append(items, value)
	}
	sort.Strings(items)
	return items
}

func diff(expected, observed []string) []string {
	observedSet := make(map[string]struct{}, len(observed))
	for _, value := range observed {
		observedSet[value] = struct{}{}
	}

	missing := make([]string, 0)
	for _, value := range expected {
		if _, ok := observedSet[value]; ok {
			continue
		}
		missing = append(missing, value)
	}
	return uniqueSorted(missing)
}

func joinNatural(values []string) string {
	switch len(values) {
	case 0:
		return "-"
	case 1:
		return values[0]
	case 2:
		return values[0] + " and " + values[1]
	default:
		return strings.Join(values[:len(values)-1], ", ") + ", and " + values[len(values)-1]
	}
}

func escapeTable(value string) string {
	return strings.ReplaceAll(value, "|", "\\|")
}
