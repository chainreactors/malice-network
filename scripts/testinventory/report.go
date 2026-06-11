package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func componentReport(spec ComponentSpec) ComponentReport {
	return ComponentReport{
		ID:             spec.ID,
		Name:           spec.Name,
		Path:           normalizePath(spec.Path),
		Tier:           spec.Tier,
		Summary:        spec.Summary,
		Chains:         append([]string(nil), spec.Chains...),
		ExpectedLayers: uniqueSorted(spec.ExpectedLayers),
		CILanes:        uniqueSorted(spec.CILanes),
		Status:         "needs_attention",
	}
}

func mergeComponentCoverage(report ComponentReport, packages []PackageStats) ComponentReport {
	prefix := report.Path
	layers := make(map[string]struct{})
	files := make([]string, 0)
	for _, pkg := range packages {
		if pkg.Path != prefix && !strings.HasPrefix(pkg.Path, prefix+"/") {
			continue
		}

		for _, layer := range pkg.Layers {
			layers[layer] = struct{}{}
		}
		files = append(files, pkg.TestFilePaths...)
	}

	report.ObservedLayers = sortedSet(layers)
	sort.Strings(files)
	report.ObservedFiles = files
	report.ObservedTests = len(files)
	report.MissingLayers = diff(report.ExpectedLayers, report.ObservedLayers)
	report.Status = componentStatus(report)
	report.Recommendation = recommendComponent(report)
	return report
}

func componentStatus(report ComponentReport) string {
	if len(report.MissingLayers) == 0 {
		return "healthy"
	}
	if len(report.ObservedLayers) == 0 {
		return "missing"
	}
	return "needs_attention"
}

func recommendComponent(report ComponentReport) string {
	if len(report.MissingLayers) == 0 {
		return "Keep the current coverage shape and only extend on regression."
	}
	if len(report.ObservedLayers) == 0 {
		return fmt.Sprintf("Add the first %s suite under %s.", joinNatural(report.ExpectedLayers), report.Path)
	}
	return fmt.Sprintf("Extend %s with %s coverage.", report.Path, joinNatural(report.MissingLayers))
}

func buildChainReport(spec ChainSpec, components map[string]ComponentReport) ChainReport {
	report := ChainReport{
		ID:            spec.ID,
		Name:          spec.Name,
		Summary:       spec.Summary,
		PrimaryLayers: uniqueSorted(spec.PrimaryLayers),
		Components:    append([]string(nil), spec.Components...),
		Status:        "healthy",
	}

	missingLayerSet := make(map[string]struct{})
	for _, componentID := range spec.Components {
		component, ok := components[componentID]
		if !ok {
			report.MissingComponents = append(report.MissingComponents, componentID)
			continue
		}

		if component.Status == "healthy" {
			report.HealthyComponents++
			continue
		}

		report.AttentionComponents++
		for _, layer := range component.MissingLayers {
			missingLayerSet[layer] = struct{}{}
		}
	}

	report.MissingLayers = sortedSet(missingLayerSet)
	if len(report.MissingComponents) > 0 || report.AttentionComponents > 0 {
		report.Status = "needs_attention"
	}

	switch {
	case len(report.MissingComponents) > 0:
		report.Recommendation = fmt.Sprintf("Resolve manifest/component drift and then add %s coverage.", joinNatural(report.MissingLayers))
	case len(report.MissingLayers) > 0:
		report.Recommendation = fmt.Sprintf("Prioritize %s coverage for this chain.", joinNatural(report.MissingLayers))
	default:
		report.Recommendation = "Keep the chain under the current CI gates."
	}

	return report
}

func buildGapReports(packages []PackageStats, manifest Manifest) []GapReport {
	manifestPaths := make(map[string]struct{}, len(manifest.Components))
	for _, component := range manifest.Components {
		manifestPaths[normalizePath(component.Path)] = struct{}{}
	}

	type scoredGap struct {
		GapReport
		score int
	}

	gaps := make([]scoredGap, 0)
	for _, pkg := range packages {
		if pkg.GoFiles < 4 {
			continue
		}
		if _, tracked := manifestPaths[pkg.Path]; tracked {
			continue
		}
		if pkg.TestFiles > 1 {
			continue
		}

		recommendation := "Add a default test entrypoint for this package."
		if pkg.TestFiles == 1 {
			recommendation = "Expand beyond the current thin test surface."
		}
		if len(pkg.Layers) == 1 && pkg.Layers[0] == "tagged" {
			recommendation = "Add an untagged package-level test to keep the default suite honest."
		}

		score := pkg.GoFiles*10 + (2-pkg.TestFiles)*25
		gaps = append(gaps, scoredGap{
			GapReport: GapReport{
				Path:           pkg.Path,
				GoFiles:        pkg.GoFiles,
				TestFiles:      pkg.TestFiles,
				Layers:         append([]string(nil), pkg.Layers...),
				Recommendation: recommendation,
			},
			score: score,
		})
	}

	sort.Slice(gaps, func(i, j int) bool {
		if gaps[i].score != gaps[j].score {
			return gaps[i].score > gaps[j].score
		}
		return gaps[i].Path < gaps[j].Path
	})

	limit := len(gaps)
	if limit > 12 {
		limit = 12
	}

	results := make([]GapReport, 0, limit)
	for _, gap := range gaps[:limit] {
		results = append(results, gap.GapReport)
	}
	return results
}

func writeReport(outputDir string, report Report) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	jsonPath := filepath.Join(outputDir, "core-testing-report.json")
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report json: %w", err)
	}
	if err := os.WriteFile(jsonPath, jsonData, 0o644); err != nil {
		return fmt.Errorf("write report json: %w", err)
	}

	markdownPath := filepath.Join(outputDir, "core-testing-report.md")
	if err := os.WriteFile(markdownPath, []byte(renderMarkdown(report)), 0o644); err != nil {
		return fmt.Errorf("write report markdown: %w", err)
	}
	return nil
}

func renderMarkdown(report Report) string {
	var builder strings.Builder

	builder.WriteString("# Core Testing Report\n\n")
	builder.WriteString(fmt.Sprintf("- Generated at: `%s`\n", report.Summary.GeneratedAt))
	builder.WriteString(fmt.Sprintf("- Packages scanned: `%d`\n", report.Summary.PackagesScanned))
	builder.WriteString(fmt.Sprintf("- Packages with tests: `%d`\n", report.Summary.PackagesWithTests))
	builder.WriteString(fmt.Sprintf("- Packages without tests: `%d`\n", report.Summary.PackagesWithoutTests))
	builder.WriteString(fmt.Sprintf("- Core components healthy: `%d/%d`\n\n", report.Summary.HealthyCoreComponents, report.Summary.CoreComponents))

	builder.WriteString("## Core Components\n\n")
	builder.WriteString("| Component | Tier | Expected | Observed | Missing | Status | Recommendation |\n")
	builder.WriteString("| --- | --- | --- | --- | --- | --- | --- |\n")
	for _, component := range report.Components {
		builder.WriteString(fmt.Sprintf(
			"| %s | %s | %s | %s | %s | %s | %s |\n",
			component.Name,
			component.Tier,
			escapeTable(joinNatural(component.ExpectedLayers)),
			escapeTable(joinNatural(component.ObservedLayers)),
			escapeTable(joinNatural(component.MissingLayers)),
			component.Status,
			escapeTable(component.Recommendation),
		))
	}

	builder.WriteString("\n## Core Chains\n\n")
	builder.WriteString("| Chain | Primary Layers | Missing Layers | Status | Recommendation |\n")
	builder.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, chain := range report.Chains {
		builder.WriteString(fmt.Sprintf(
			"| %s | %s | %s | %s | %s |\n",
			chain.Name,
			escapeTable(joinNatural(chain.PrimaryLayers)),
			escapeTable(joinNatural(chain.MissingLayers)),
			chain.Status,
			escapeTable(chain.Recommendation),
		))
	}

	builder.WriteString("\n## Top Package Gaps\n\n")
	if len(report.TopGapPackages) == 0 {
		builder.WriteString("No package-level gaps matched the current heuristic.\n")
		return builder.String()
	}

	builder.WriteString("| Package | Go Files | Test Files | Layers | Recommendation |\n")
	builder.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, gap := range report.TopGapPackages {
		builder.WriteString(fmt.Sprintf(
			"| %s | %d | %d | %s | %s |\n",
			gap.Path,
			gap.GoFiles,
			gap.TestFiles,
			escapeTable(joinNatural(gap.Layers)),
			escapeTable(gap.Recommendation),
		))
	}

	return builder.String()
}
