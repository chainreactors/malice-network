package build

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

// --- helpers ---

// createZip builds an in-memory zip from a map of filename→content.
func createZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("zip create %s: %v", name, err)
		}
		if _, err := f.Write(content); err != nil {
			t.Fatalf("zip write %s: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}

// --- parseArchive tests ---

func TestParseArchive_Full(t *testing.T) {
	data := createZip(t, map[string][]byte{
		"implant.yaml":      []byte("implant content"),
		"prelude.yaml":      []byte("prelude content"),
		"resources/a.bin":   []byte("aaa"),
		"resources/b.bin":   []byte("bbb"),
	})

	implant, prelude, resources, err := parseArchive(data)
	if err != nil {
		t.Fatalf("parseArchive: %v", err)
	}
	if string(implant) != "implant content" {
		t.Errorf("implant: got %q, want %q", implant, "implant content")
	}
	if string(prelude) != "prelude content" {
		t.Errorf("prelude: got %q, want %q", prelude, "prelude content")
	}
	if resources == nil || len(resources.Entries) != 2 {
		t.Fatalf("resources: got %v entries, want 2", resources)
	}
	rmap := make(map[string]string)
	for _, e := range resources.Entries {
		rmap[e.Filename] = string(e.Content)
	}
	if rmap["a.bin"] != "aaa" {
		t.Errorf("resource a.bin: got %q, want %q", rmap["a.bin"], "aaa")
	}
	if rmap["b.bin"] != "bbb" {
		t.Errorf("resource b.bin: got %q, want %q", rmap["b.bin"], "bbb")
	}
}

func TestParseArchive_ImplantOnly(t *testing.T) {
	data := createZip(t, map[string][]byte{
		"implant.yaml": []byte("yaml only"),
	})

	implant, prelude, resources, err := parseArchive(data)
	if err != nil {
		t.Fatalf("parseArchive: %v", err)
	}
	if string(implant) != "yaml only" {
		t.Errorf("implant: got %q", implant)
	}
	if prelude != nil {
		t.Errorf("prelude should be nil, got %q", prelude)
	}
	if resources != nil {
		t.Errorf("resources should be nil, got %v", resources)
	}
}

func TestParseArchive_PreludeOnly(t *testing.T) {
	data := createZip(t, map[string][]byte{
		"prelude.yaml": []byte("prelude only"),
	})

	implant, prelude, resources, err := parseArchive(data)
	if err != nil {
		t.Fatalf("parseArchive: %v", err)
	}
	if implant != nil {
		t.Errorf("implant should be nil, got %q", implant)
	}
	if string(prelude) != "prelude only" {
		t.Errorf("prelude: got %q", prelude)
	}
	if resources != nil {
		t.Errorf("resources should be nil, got %v", resources)
	}
}

func TestParseArchive_Empty(t *testing.T) {
	data := createZip(t, map[string][]byte{})

	implant, prelude, resources, err := parseArchive(data)
	if err != nil {
		t.Fatalf("parseArchive: %v", err)
	}
	if implant != nil || prelude != nil || resources != nil {
		t.Errorf("all should be nil for empty archive")
	}
}

func TestParseArchive_IgnoresUnknownFiles(t *testing.T) {
	data := createZip(t, map[string][]byte{
		"implant.yaml": []byte("impl"),
		"readme.txt":   []byte("ignored"),
		"other/foo":    []byte("ignored"),
	})

	implant, prelude, resources, err := parseArchive(data)
	if err != nil {
		t.Fatalf("parseArchive: %v", err)
	}
	if string(implant) != "impl" {
		t.Errorf("implant: got %q", implant)
	}
	if prelude != nil {
		t.Errorf("prelude should be nil")
	}
	if resources != nil {
		t.Errorf("resources should be nil")
	}
}

func TestParseArchive_InvalidZip(t *testing.T) {
	_, _, _, err := parseArchive([]byte("not a zip"))
	if err == nil {
		t.Fatal("expected error for invalid zip data")
	}
}

// --- readResourcesDir tests ---

func TestReadResourcesDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.bin"), []byte("aaa"), 0644)
	os.WriteFile(filepath.Join(dir, "b.bin"), []byte("bbb"), 0644)

	resources, err := readResourcesDir(dir)
	if err != nil {
		t.Fatalf("readResourcesDir: %v", err)
	}
	if resources == nil || len(resources.Entries) != 2 {
		t.Fatalf("resources: got %v entries, want 2", resources)
	}
	rmap := make(map[string]string)
	for _, e := range resources.Entries {
		rmap[e.Filename] = string(e.Content)
	}
	if rmap["a.bin"] != "aaa" {
		t.Errorf("a.bin: got %q", rmap["a.bin"])
	}
	if rmap["b.bin"] != "bbb" {
		t.Errorf("b.bin: got %q", rmap["b.bin"])
	}
}

func TestReadResourcesDir_Empty(t *testing.T) {
	dir := t.TempDir()

	resources, err := readResourcesDir(dir)
	if err != nil {
		t.Fatalf("readResourcesDir: %v", err)
	}
	if resources != nil {
		t.Errorf("resources should be nil for empty dir, got %v", resources)
	}
}

func TestReadResourcesDir_SkipsSubdirs(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "file.bin"), []byte("data"), 0644)
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)
	os.WriteFile(filepath.Join(dir, "subdir", "nested.bin"), []byte("nested"), 0644)

	resources, err := readResourcesDir(dir)
	if err != nil {
		t.Fatalf("readResourcesDir: %v", err)
	}
	if resources == nil || len(resources.Entries) != 1 {
		t.Fatalf("resources: got %v, want 1 entry (subdir skipped)", resources)
	}
	if resources.Entries[0].Filename != "file.bin" {
		t.Errorf("filename: got %q, want %q", resources.Entries[0].Filename, "file.bin")
	}
}

func TestReadResourcesDir_NotExist(t *testing.T) {
	_, err := readResourcesDir("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
}

// --- loadBuildInputs tests ---

// newTestCmd creates a cobra.Command with the given flags registered,
// then parses the provided args to set flag values.
func newTestCmd(t *testing.T, flagSet func(cmd *cobra.Command), args []string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "test", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	flagSet(cmd)
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	return cmd
}

func TestLoadBuildInputs_NoFlags(t *testing.T) {
	cmd := newTestCmd(t, func(cmd *cobra.Command) {
		BuildInputFlagSet(cmd.Flags())
	}, nil)

	implant, prelude, resources, err := loadBuildInputs(cmd)
	if err != nil {
		t.Fatalf("loadBuildInputs: %v", err)
	}
	if implant != nil || prelude != nil || resources != nil {
		t.Error("all should be nil when no flags set")
	}
}

func TestLoadBuildInputs_ImplantPath(t *testing.T) {
	dir := t.TempDir()
	implantFile := filepath.Join(dir, "implant.yaml")
	os.WriteFile(implantFile, []byte("basic:\n  name: test\n"), 0644)

	cmd := newTestCmd(t, func(cmd *cobra.Command) {
		BuildInputFlagSet(cmd.Flags())
	}, []string{"--implant-path", implantFile})

	implant, prelude, resources, err := loadBuildInputs(cmd)
	if err != nil {
		t.Fatalf("loadBuildInputs: %v", err)
	}
	if string(implant) != "basic:\n  name: test\n" {
		t.Errorf("implant: got %q", implant)
	}
	if prelude != nil || resources != nil {
		t.Error("prelude and resources should be nil")
	}
}

func TestLoadBuildInputs_ArchivePath(t *testing.T) {
	dir := t.TempDir()
	archiveFile := filepath.Join(dir, "build.zip")
	data := createZip(t, map[string][]byte{
		"implant.yaml":    []byte("impl"),
		"prelude.yaml":    []byte("prel"),
		"resources/r.bin": []byte("res"),
	})
	os.WriteFile(archiveFile, data, 0644)

	cmd := newTestCmd(t, func(cmd *cobra.Command) {
		BuildInputFlagSet(cmd.Flags())
	}, []string{"--archive-path", archiveFile})

	implant, prelude, resources, err := loadBuildInputs(cmd)
	if err != nil {
		t.Fatalf("loadBuildInputs: %v", err)
	}
	if string(implant) != "impl" {
		t.Errorf("implant: got %q", implant)
	}
	if string(prelude) != "prel" {
		t.Errorf("prelude: got %q", prelude)
	}
	if resources == nil || len(resources.Entries) != 1 {
		t.Fatalf("resources: got %v, want 1", resources)
	}
	if resources.Entries[0].Filename != "r.bin" || string(resources.Entries[0].Content) != "res" {
		t.Errorf("resource: got %v", resources.Entries[0])
	}
}

func TestLoadBuildInputs_IndividualOverridesArchive(t *testing.T) {
	dir := t.TempDir()

	// Archive with implant and prelude
	archiveFile := filepath.Join(dir, "build.zip")
	data := createZip(t, map[string][]byte{
		"implant.yaml": []byte("archive-implant"),
		"prelude.yaml": []byte("archive-prelude"),
	})
	os.WriteFile(archiveFile, data, 0644)

	// Individual implant file (should override archive's implant)
	implantFile := filepath.Join(dir, "my-implant.yaml")
	os.WriteFile(implantFile, []byte("file-implant"), 0644)

	cmd := newTestCmd(t, func(cmd *cobra.Command) {
		BuildInputFlagSet(cmd.Flags())
	}, []string{"--archive-path", archiveFile, "--implant-path", implantFile})

	implant, prelude, _, err := loadBuildInputs(cmd)
	if err != nil {
		t.Fatalf("loadBuildInputs: %v", err)
	}
	// implant should be from individual file, not archive
	if string(implant) != "file-implant" {
		t.Errorf("implant should be overridden: got %q, want %q", implant, "file-implant")
	}
	// prelude should still be from archive
	if string(prelude) != "archive-prelude" {
		t.Errorf("prelude should come from archive: got %q, want %q", prelude, "archive-prelude")
	}
}

func TestLoadBuildInputs_ResourcesPath(t *testing.T) {
	dir := t.TempDir()
	resDir := filepath.Join(dir, "resources")
	os.MkdirAll(resDir, 0755)
	os.WriteFile(filepath.Join(resDir, "x.bin"), []byte("xxx"), 0644)

	cmd := newTestCmd(t, func(cmd *cobra.Command) {
		BuildInputFlagSet(cmd.Flags())
	}, []string{"--resources-path", resDir})

	_, _, resources, err := loadBuildInputs(cmd)
	if err != nil {
		t.Fatalf("loadBuildInputs: %v", err)
	}
	if resources == nil || len(resources.Entries) != 1 {
		t.Fatalf("resources: got %v, want 1 entry", resources)
	}
	if resources.Entries[0].Filename != "x.bin" || string(resources.Entries[0].Content) != "xxx" {
		t.Errorf("resource: got %v", resources.Entries[0])
	}
}

func TestLoadBuildInputs_PreludeInputFlagSet(t *testing.T) {
	dir := t.TempDir()
	preludeFile := filepath.Join(dir, "prelude.yaml")
	os.WriteFile(preludeFile, []byte("prelude data"), 0644)

	// PreludeInputFlagSet does not include --implant-path
	cmd := newTestCmd(t, func(cmd *cobra.Command) {
		PreludeInputFlagSet(cmd.Flags())
	}, []string{"--prelude-path", preludeFile})

	implant, prelude, _, err := loadBuildInputs(cmd)
	if err != nil {
		t.Fatalf("loadBuildInputs: %v", err)
	}
	if implant != nil {
		t.Error("implant should be nil (no --implant-path flag in PreludeInputFlagSet)")
	}
	if string(prelude) != "prelude data" {
		t.Errorf("prelude: got %q", prelude)
	}
}

func TestLoadBuildInputs_ImplantInputFlagSet(t *testing.T) {
	dir := t.TempDir()
	implantFile := filepath.Join(dir, "implant.yaml")
	os.WriteFile(implantFile, []byte("pulse implant"), 0644)

	// ImplantInputFlagSet only has --implant-path
	cmd := newTestCmd(t, func(cmd *cobra.Command) {
		ImplantInputFlagSet(cmd.Flags())
	}, []string{"--implant-path", implantFile})

	implant, prelude, resources, err := loadBuildInputs(cmd)
	if err != nil {
		t.Fatalf("loadBuildInputs: %v", err)
	}
	if string(implant) != "pulse implant" {
		t.Errorf("implant: got %q", implant)
	}
	if prelude != nil || resources != nil {
		t.Error("prelude and resources should be nil (no such flags in ImplantInputFlagSet)")
	}
}

func TestLoadBuildInputs_BadImplantPath(t *testing.T) {
	cmd := newTestCmd(t, func(cmd *cobra.Command) {
		BuildInputFlagSet(cmd.Flags())
	}, []string{"--implant-path", "/nonexistent/file.yaml"})

	_, _, _, err := loadBuildInputs(cmd)
	if err == nil {
		t.Fatal("expected error for non-existent implant file")
	}
}

func TestLoadBuildInputs_BadArchivePath(t *testing.T) {
	cmd := newTestCmd(t, func(cmd *cobra.Command) {
		BuildInputFlagSet(cmd.Flags())
	}, []string{"--archive-path", "/nonexistent/archive.zip"})

	_, _, _, err := loadBuildInputs(cmd)
	if err == nil {
		t.Fatal("expected error for non-existent archive file")
	}
}
