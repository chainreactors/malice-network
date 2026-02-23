package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
)

func TestWriteAndReadProfileDisk(t *testing.T) {
	dir := t.TempDir()

	implant := []byte("basic:\n  name: test\n")
	prelude := []byte("prelude content")
	resources := &clientpb.BuildResources{
		Entries: []*clientpb.ResourceEntry{
			{Filename: "a.bin", Content: []byte("aaa")},
			{Filename: "b.bin", Content: []byte("bbb")},
		},
	}

	// write
	if err := writeProfileDisk(dir, implant, prelude, resources); err != nil {
		t.Fatalf("writeProfileDisk failed: %v", err)
	}

	// verify files exist
	if _, err := os.Stat(filepath.Join(dir, "implant.yaml")); err != nil {
		t.Fatalf("implant.yaml not found: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "prelude.yaml")); err != nil {
		t.Fatalf("prelude.yaml not found: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "resources", "a.bin")); err != nil {
		t.Fatalf("resources/a.bin not found: %v", err)
	}

	// read back
	gotImplant, gotPrelude, gotResources, err := readProfileDisk(dir)
	if err != nil {
		t.Fatalf("readProfileDisk failed: %v", err)
	}

	if string(gotImplant) != string(implant) {
		t.Errorf("implant mismatch: got %q, want %q", gotImplant, implant)
	}
	if string(gotPrelude) != string(prelude) {
		t.Errorf("prelude mismatch: got %q, want %q", gotPrelude, prelude)
	}
	if gotResources == nil || len(gotResources.Entries) != 2 {
		t.Fatalf("resources count: got %v, want 2", gotResources)
	}

	resourceMap := make(map[string]string)
	for _, e := range gotResources.Entries {
		resourceMap[e.Filename] = string(e.Content)
	}
	if resourceMap["a.bin"] != "aaa" {
		t.Errorf("resource a.bin: got %q, want %q", resourceMap["a.bin"], "aaa")
	}
	if resourceMap["b.bin"] != "bbb" {
		t.Errorf("resource b.bin: got %q, want %q", resourceMap["b.bin"], "bbb")
	}
}

func TestWriteAndReadProfileDisk_ImplantOnly(t *testing.T) {
	dir := t.TempDir()

	implant := []byte("basic:\n  name: minimal\n")

	if err := writeProfileDisk(dir, implant, nil, nil); err != nil {
		t.Fatalf("writeProfileDisk failed: %v", err)
	}

	gotImplant, gotPrelude, gotResources, err := readProfileDisk(dir)
	if err != nil {
		t.Fatalf("readProfileDisk failed: %v", err)
	}

	if string(gotImplant) != string(implant) {
		t.Errorf("implant mismatch: got %q, want %q", gotImplant, implant)
	}
	if gotPrelude != nil {
		t.Errorf("prelude should be nil, got %q", gotPrelude)
	}
	if gotResources != nil {
		t.Errorf("resources should be nil, got %v", gotResources)
	}
}

func TestReadProfileDisk_MissingImplant(t *testing.T) {
	dir := t.TempDir()

	_, _, _, err := readProfileDisk(dir)
	if err == nil {
		t.Fatal("expected error for missing implant.yaml, got nil")
	}
}

func TestWriteProfileDisk_EmptyResources(t *testing.T) {
	dir := t.TempDir()

	implant := []byte("basic:\n  name: test\n")
	emptyResources := &clientpb.BuildResources{Entries: nil}

	if err := writeProfileDisk(dir, implant, nil, emptyResources); err != nil {
		t.Fatalf("writeProfileDisk failed: %v", err)
	}

	// resources 目录不应该被创建
	if _, err := os.Stat(filepath.Join(dir, "resources")); !os.IsNotExist(err) {
		t.Error("resources dir should not exist for empty entries")
	}
}
