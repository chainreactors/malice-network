package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/server/internal/configs"
)

func TestMoveBuildOutputUsesBaseTargetForGlibcZigTarget(t *testing.T) {
	oldTargetPath := configs.TargetPath
	oldTempPath := configs.TempPath
	t.Cleanup(func() {
		configs.TargetPath = oldTargetPath
		configs.TempPath = oldTempPath
	})

	root := t.TempDir()
	configs.TargetPath = filepath.Join(root, "target")
	configs.TempPath = filepath.Join(root, "temp")
	if err := os.MkdirAll(configs.TempPath, 0700); err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}

	outputDir := filepath.Join(configs.TargetPath, consts.TargetX64LinuxGnu, release)
	if err := os.MkdirAll(outputDir, 0700); err != nil {
		t.Fatalf("mkdir output: %v", err)
	}
	sourceFile := filepath.Join(outputDir, malefic)
	if err := os.WriteFile(sourceFile, []byte("payload"), 0600); err != nil {
		t.Fatalf("write source artifact: %v", err)
	}

	gotSource, gotDest, err := MoveBuildOutput(consts.TargetX64LinuxGnu217, consts.CommandBuildBeacon, false, "", false, false)
	if err != nil {
		t.Fatalf("MoveBuildOutput failed: %v", err)
	}
	if gotSource != sourceFile {
		t.Fatalf("source path = %q, want %q", gotSource, sourceFile)
	}
	if _, err := os.Stat(gotDest); err != nil {
		t.Fatalf("dest artifact missing: %v", err)
	}
}

func TestMoveBuildOutputUsesCargoLibNameForBeaconLibrary(t *testing.T) {
	oldSourceCodePath := configs.SourceCodePath
	oldTargetPath := configs.TargetPath
	oldTempPath := configs.TempPath
	t.Cleanup(func() {
		configs.SourceCodePath = oldSourceCodePath
		configs.TargetPath = oldTargetPath
		configs.TempPath = oldTempPath
	})

	root := t.TempDir()
	configs.SourceCodePath = filepath.Join(root, "source_code")
	configs.TargetPath = filepath.Join(configs.SourceCodePath, "target")
	configs.TempPath = filepath.Join(root, "temp")
	if err := os.MkdirAll(configs.TempPath, 0700); err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}

	cargoDir := filepath.Join(configs.SourceCodePath, "malefic")
	if err := os.MkdirAll(cargoDir, 0700); err != nil {
		t.Fatalf("mkdir cargo dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cargoDir, "Cargo.toml"), []byte(`
[package]
name = "malefic"

[lib]
name = "malefic_lib"
crate-type = ["cdylib", "rlib"]
`), 0600); err != nil {
		t.Fatalf("write Cargo.toml: %v", err)
	}

	outputDir := filepath.Join(configs.TargetPath, consts.TargetX64WindowsGnu, release)
	if err := os.MkdirAll(outputDir, 0700); err != nil {
		t.Fatalf("mkdir output: %v", err)
	}
	sourceFile := filepath.Join(outputDir, "malefic_lib.dll")
	if err := os.WriteFile(sourceFile, []byte("payload"), 0600); err != nil {
		t.Fatalf("write source artifact: %v", err)
	}

	gotSource, gotDest, err := MoveBuildOutput(consts.TargetX64WindowsGnu, consts.CommandBuildBeacon, false, "lib", false, false)
	if err != nil {
		t.Fatalf("MoveBuildOutput failed: %v", err)
	}
	if gotSource != sourceFile {
		t.Fatalf("source path = %q, want %q", gotSource, sourceFile)
	}
	if filepath.Ext(gotDest) != ".dll" {
		t.Fatalf("dest extension = %q, want .dll", filepath.Ext(gotDest))
	}
	if _, err := os.Stat(gotDest); err != nil {
		t.Fatalf("dest artifact missing: %v", err)
	}
}
