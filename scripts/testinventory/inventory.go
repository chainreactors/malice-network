package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var defaultSkipDirs = map[string]struct{}{
	".claude":  {},
	".git":     {},
	".idea":    {},
	".malice":  {},
	"bin":      {},
	"dist":     {},
	"external": {},
}

type Manifest struct {
	Components []ComponentSpec `json:"components"`
	Chains     []ChainSpec     `json:"chains"`
}

type ComponentSpec struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Path           string   `json:"path"`
	Tier           string   `json:"tier"`
	Summary        string   `json:"summary"`
	Chains         []string `json:"chains"`
	ExpectedLayers []string `json:"expected_layers"`
	CILanes        []string `json:"ci_lanes"`
}

type ChainSpec struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Summary       string   `json:"summary"`
	PrimaryLayers []string `json:"primary_layers"`
	Components    []string `json:"components"`
}

type PackageStats struct {
	Path            string         `json:"path"`
	GoFiles         int            `json:"go_files"`
	TestFiles       int            `json:"test_files"`
	Layers          []string       `json:"layers"`
	LayerFileCounts map[string]int `json:"layer_file_counts"`
	TestFilePaths   []string       `json:"test_file_paths"`
}

type Summary struct {
	GeneratedAt           string `json:"generated_at"`
	PackagesScanned       int    `json:"packages_scanned"`
	PackagesWithTests     int    `json:"packages_with_tests"`
	PackagesWithoutTests  int    `json:"packages_without_tests"`
	CoreComponents        int    `json:"core_components"`
	HealthyCoreComponents int    `json:"healthy_core_components"`
	AttentionComponents   int    `json:"attention_components"`
}

type ComponentReport struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Path           string   `json:"path"`
	Tier           string   `json:"tier"`
	Summary        string   `json:"summary"`
	Chains         []string `json:"chains"`
	ExpectedLayers []string `json:"expected_layers"`
	ObservedLayers []string `json:"observed_layers"`
	MissingLayers  []string `json:"missing_layers"`
	ObservedTests  int      `json:"observed_tests"`
	ObservedFiles  []string `json:"observed_files"`
	CILanes        []string `json:"ci_lanes"`
	Status         string   `json:"status"`
	Recommendation string   `json:"recommendation"`
}

type ChainReport struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Summary             string   `json:"summary"`
	PrimaryLayers       []string `json:"primary_layers"`
	Components          []string `json:"components"`
	MissingComponents   []string `json:"missing_components"`
	MissingLayers       []string `json:"missing_layers"`
	HealthyComponents   int      `json:"healthy_components"`
	AttentionComponents int      `json:"attention_components"`
	Status              string   `json:"status"`
	Recommendation      string   `json:"recommendation"`
}

type GapReport struct {
	Path           string   `json:"path"`
	GoFiles        int      `json:"go_files"`
	TestFiles      int      `json:"test_files"`
	Layers         []string `json:"layers"`
	Recommendation string   `json:"recommendation"`
}

type Report struct {
	Summary        Summary           `json:"summary"`
	Components     []ComponentReport `json:"components"`
	Chains         []ChainReport     `json:"chains"`
	Packages       []PackageStats    `json:"packages"`
	TopGapPackages []GapReport       `json:"top_gap_packages"`
}

func loadManifest(path string) (Manifest, error) {
	var manifest Manifest

	data, err := os.ReadFile(path)
	if err != nil {
		return manifest, fmt.Errorf("read manifest: %w", err)
	}

	if err := json.Unmarshal(data, &manifest); err != nil {
		return manifest, fmt.Errorf("parse manifest: %w", err)
	}
	if len(manifest.Components) == 0 {
		return manifest, fmt.Errorf("manifest has no components")
	}

	return manifest, nil
}

func collectPackageStats(root string) ([]PackageStats, error) {
	packages := make(map[string]*PackageStats)

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if _, skip := defaultSkipDirs[d.Name()]; skip && path != root {
				return filepath.SkipDir
			}
			return nil
		}

		if filepath.Ext(path) != ".go" {
			return nil
		}

		relDir, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		relDir = normalizePath(relDir)

		stats := packages[relDir]
		if stats == nil {
			stats = &PackageStats{
				Path:            relDir,
				LayerFileCounts: make(map[string]int),
			}
			packages[relDir] = stats
		}

		name := filepath.Base(path)
		if strings.HasSuffix(name, "_test.go") {
			stats.TestFiles++
			layer, err := detectLayer(path, relDir)
			if err != nil {
				return err
			}

			stats.LayerFileCounts[layer]++
			stats.TestFilePaths = append(stats.TestFilePaths, normalizePath(mustRel(root, path)))
			return nil
		}

		stats.GoFiles++
		return nil
	})
	if err != nil {
		return nil, err
	}

	results := make([]PackageStats, 0, len(packages))
	for _, stats := range packages {
		stats.Layers = sortedKeys(stats.LayerFileCounts)
		sort.Strings(stats.TestFilePaths)
		results = append(results, *stats)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Path < results[j].Path
	})
	return results, nil
}

func detectLayer(path, relDir string) (string, error) {
	expr, err := readBuildExpr(path)
	if err != nil {
		return "", err
	}

	switch {
	case buildExprContains(expr, "integration"):
		return "integration", nil
	case buildExprContains(expr, "mockimplant"):
		return "mockimplant", nil
	case buildExprContains(expr, "realimplant"):
		return "realimplant", nil
	case expr != "":
		return "tagged", nil
	case strings.HasPrefix(relDir, "client/command/"):
		return "command_conformance", nil
	default:
		return "unit", nil
	}
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

func buildReport(manifest Manifest, packages []PackageStats) Report {
	componentReports := make([]ComponentReport, 0, len(manifest.Components))
	healthyComponents := 0
	for _, spec := range manifest.Components {
		report := mergeComponentCoverage(componentReport(spec), packages)
		componentReports = append(componentReports, report)
		if report.Status == "healthy" {
			healthyComponents++
		}
	}

	sort.Slice(componentReports, func(i, j int) bool {
		if componentReports[i].Tier != componentReports[j].Tier {
			return componentReports[i].Tier < componentReports[j].Tier
		}
		return componentReports[i].Path < componentReports[j].Path
	})

	componentByID := make(map[string]ComponentReport, len(componentReports))
	for _, component := range componentReports {
		componentByID[component.ID] = component
	}

	chainReports := make([]ChainReport, 0, len(manifest.Chains))
	for _, spec := range manifest.Chains {
		chainReports = append(chainReports, buildChainReport(spec, componentByID))
	}
	sort.Slice(chainReports, func(i, j int) bool {
		return chainReports[i].Name < chainReports[j].Name
	})

	packagesWithTests := 0
	for _, pkg := range packages {
		if pkg.TestFiles > 0 {
			packagesWithTests++
		}
	}

	return Report{
		Summary: Summary{
			GeneratedAt:           time.Now().UTC().Format(time.RFC3339),
			PackagesScanned:       len(packages),
			PackagesWithTests:     packagesWithTests,
			PackagesWithoutTests:  len(packages) - packagesWithTests,
			CoreComponents:        len(componentReports),
			HealthyCoreComponents: healthyComponents,
			AttentionComponents:   len(componentReports) - healthyComponents,
		},
		Components:     componentReports,
		Chains:         chainReports,
		Packages:       packages,
		TopGapPackages: buildGapReports(packages, manifest),
	}
}
