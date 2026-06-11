package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseSkillDataAndRenderSkill(t *testing.T) {
	raw := []byte(`---
name: recon
description: Enumerate target
---
Inspect target: $ARGUMENTS
Primary: $0
Secondary: $ARGUMENTS[1]
`)

	skill, err := parseSkillData(raw)
	if err != nil {
		t.Fatalf("parseSkillData returned error: %v", err)
	}
	if skill.Name != "recon" {
		t.Fatalf("skill.Name = %q, want %q", skill.Name, "recon")
	}
	if skill.Description != "Enumerate target" {
		t.Fatalf("skill.Description = %q, want %q", skill.Description, "Enumerate target")
	}

	got := renderSkill(skill, []string{"web", "db"})
	want := "Inspect target: web db\nPrimary: web\nSecondary: db"
	if got != want {
		t.Fatalf("renderSkill() = %q, want %q", got, want)
	}
}

func TestLoadSkillFromLocalDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(oldWD); chdirErr != nil {
			t.Fatalf("restore wd: %v", chdirErr)
		}
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir(%q): %v", tmpDir, err)
	}

	skillDir := filepath.Join(tmpDir, "skills", "demo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	skillBody := `---
name: demo
description: Demo skill
---
Run checks for $ARGUMENTS
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillBody), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	skill, err := LoadSkill("demo")
	if err != nil {
		t.Fatalf("LoadSkill returned error: %v", err)
	}
	if skill.Name != "demo" {
		t.Fatalf("skill.Name = %q, want %q", skill.Name, "demo")
	}
	if skill.Description != "Demo skill" {
		t.Fatalf("skill.Description = %q, want %q", skill.Description, "Demo skill")
	}
	wantDir := filepath.Join(".", "skills", "demo")
	if skill.Dir != wantDir {
		t.Fatalf("skill.Dir = %q, want %q", skill.Dir, wantDir)
	}

	got := renderSkill(skill, []string{"network"})
	want := "Run checks for network"
	if got != want {
		t.Fatalf("renderSkill() = %q, want %q", got, want)
	}

	skills := DiscoverSkills()
	found := false
	for _, info := range skills {
		if info.Name == "demo" && info.Source == "local" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("DiscoverSkills did not return local demo skill")
	}
}
