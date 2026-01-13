package plugin

import (
	"os"
	"path/filepath"
	"testing"

	wizardfw "github.com/chainreactors/malice-network/client/wizard"
)

func TestRegisterWizardTemplatesFromDisk(t *testing.T) {
	tmp := t.TempDir()
	wizDir := filepath.Join(tmp, wizardSpecDir)
	if err := os.MkdirAll(wizDir, 0o755); err != nil {
		t.Fatalf("mkdir wizards dir: %v", err)
	}

	specPath := filepath.Join(wizDir, "priv_esc.yaml")
	if err := os.WriteFile(specPath, []byte(`
title: Privilege Escalation
fields:
  - name: method
    title: Method
    type: select
    options: [uac, token]
`), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	pluginName := "testplug"
	if n := registerWizardTemplatesFromDisk(pluginName, tmp); n != 1 {
		t.Fatalf("expected 1 registered template, got %d", n)
	}

	wiz, ok := wizardfw.GetTemplate("testplug:priv_esc")
	if !ok || wiz == nil {
		t.Fatalf("expected registered template testplug:priv_esc")
	}
	if wiz.Title == "" {
		t.Fatalf("expected non-empty title")
	}
	if len(wiz.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(wiz.Fields))
	}
}
