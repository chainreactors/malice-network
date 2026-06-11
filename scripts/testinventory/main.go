package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	var manifestPath string
	var outputDir string

	flag.StringVar(&manifestPath, "manifest", filepath.Join("docs", "development", "core-testing-manifest.json"), "path to the core testing manifest")
	flag.StringVar(&outputDir, "output", filepath.Join("dist", "testing"), "directory for generated inventory reports")
	flag.Parse()

	manifest, err := loadManifest(manifestPath)
	if err != nil {
		exitErr(err)
	}

	root, err := os.Getwd()
	if err != nil {
		exitErr(fmt.Errorf("resolve repo root: %w", err))
	}

	packages, err := collectPackageStats(root)
	if err != nil {
		exitErr(err)
	}

	report := buildReport(manifest, packages)
	if err := writeReport(outputDir, report); err != nil {
		exitErr(err)
	}

	fmt.Printf("generated %d component records at %s\n", len(report.Components), outputDir)
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
