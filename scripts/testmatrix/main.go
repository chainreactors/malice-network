package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	var layer string
	var format string
	var root string
	var failEmpty bool

	flag.StringVar(&layer, "layer", "", "build tag or test layer to discover")
	flag.StringVar(&format, "format", "shell", "output format: shell or lines")
	flag.StringVar(&root, "root", ".", "repository root to scan")
	flag.BoolVar(&failEmpty, "fail-empty", true, "exit with an error when no tagged packages are found")
	flag.Parse()

	if strings.TrimSpace(layer) == "" {
		exitErr(errors.New("layer is required"))
	}

	packages, err := discoverTaggedPackages(root, layer)
	if err != nil {
		exitErr(err)
	}
	if len(packages) == 0 && failEmpty {
		exitErr(fmt.Errorf("no packages found for layer %q", layer))
	}

	switch format {
	case "shell":
		fmt.Println(strings.Join(packages, " "))
	case "lines":
		for _, pkg := range packages {
			fmt.Println(pkg)
		}
	default:
		exitErr(fmt.Errorf("unsupported format %q", format))
	}
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
