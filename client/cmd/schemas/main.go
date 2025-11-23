package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/basic"
	"github.com/chainreactors/malice-network/client/command/exec"
	"github.com/chainreactors/malice-network/client/command/file"
	"github.com/chainreactors/malice-network/client/command/filesystem"
	"github.com/chainreactors/malice-network/client/command/privilege"
	"github.com/chainreactors/malice-network/client/command/reg"
	"github.com/chainreactors/malice-network/client/command/service"
	"github.com/chainreactors/malice-network/client/command/sys"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/plugin"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	"github.com/spf13/cobra"
)

func init() {
	config.WithOptions(func(opt *config.Options) {
		opt.DecoderConfig.TagName = "config"
		opt.ParseDefault = true
	}, config.WithHookFunc(assets.HookFn))
	config.AddDriver(yaml.Driver)
}

func main() {
	var (
		outputFile string
		pretty     bool
	)

	flag.StringVar(&outputFile, "output", "schemas.json", "Output file path")
	flag.BoolVar(&pretty, "pretty", true, "Pretty print JSON")
	flag.Parse()

	// Initialize console
	logs.Log.Infof("Initializing console...\n")
	con, err := core.NewConsole()
	if err != nil {
		logs.Log.Errorf("Failed to create console: %v\n", err)
		os.Exit(1)
	}

	// Extract schemas from real commands
	logs.Log.Infof("Extracting schemas from real commands...\n")
	schemas := extractRealCommandSchemas(con)

	logs.Log.Infof("Found %d packages with %d total commands\n", len(schemas), countTotalCommands(schemas))

	// Convert to JSON
	var jsonData []byte
	if pretty {
		jsonData, err = json.MarshalIndent(schemas, "", "  ")
	} else {
		jsonData, err = json.Marshal(schemas)
	}

	if err != nil {
		logs.Log.Errorf("Failed to marshal schemas: %v\n", err)
		os.Exit(1)
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputFile)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			logs.Log.Errorf("Failed to create output directory: %v\n", err)
			os.Exit(1)
		}
	}

	// Write to file
	if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
		logs.Log.Errorf("Failed to write output file: %v\n", err)
		os.Exit(1)
	}

	logs.Log.Infof("Successfully exported schemas to: %s\n", outputFile)
	logs.Log.Infof("File size: %d bytes\n", len(jsonData))

	// Print summary
	fmt.Println("\n=== Schema Export Summary ===")
	for pkgName, commands := range schemas {
		fmt.Printf("Package: %s\n", pkgName)
		fmt.Printf("  Commands: %d\n", len(commands))
		for cmdName := range commands {
			fmt.Printf("    - %s\n", cmdName)
		}
		fmt.Println()
	}
}

func extractRealCommandSchemas(con *core.Console) map[string]map[string]*plugin.CommandSchema {
	packages := make(map[string]map[string]*plugin.CommandSchema)

	// Define command packages with their Commands() functions
	commandPackages := []struct {
		name     string
		commands func(*core.Console) []*cobra.Command
	}{
		{"basic", basic.Commands},
		{"exec", exec.Commands},
		{"file", file.Commands},
		{"filesystem", filesystem.Commands},
		{"privilege", privilege.Commands},
		{"registry", reg.Commands},
		{"service", service.Commands},
		{"sys", sys.Commands},
	}

	// Extract schemas from each package
	for _, pkg := range commandPackages {
		logs.Log.Infof("Processing package: %s\n", pkg.name)

		// Get commands from the package
		commands := pkg.commands(con)

		// Set mal annotation for all commands
		for _, cmd := range commands {
			if cmd == nil {
				continue
			}
			if cmd.Annotations == nil {
				cmd.Annotations = make(map[string]string)
			}
			if _, ok := cmd.Annotations["mal"]; !ok {
				cmd.Annotations["mal"] = pkg.name
			}
		}

		// Use unified API: []*cobra.Command -> schemas
		schemas, err := plugin.GenerateSchemasFromCommands(commands)
		if err != nil {
			logs.Log.Warnf("Failed to generate schemas for package %s: %v\n", pkg.name, err)
			continue
		}

		if len(schemas) > 0 {
			packages[pkg.name] = schemas
			logs.Log.Infof("Package %s: %d commands\n", pkg.name, len(schemas))
		}
	}

	return packages
}

func countTotalCommands(packages map[string]map[string]*plugin.CommandSchema) int {
	total := 0
	for _, commands := range packages {
		total += len(commands)
	}
	return total
}
