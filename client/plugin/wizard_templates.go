package plugin

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/wizard"
)

const wizardSpecDir = "resources/wizards"

func registerWizardTemplatesFromEmbedFS(pluginName, pluginRoot string, f fs.FS) int {
	root := path.Join(pluginRoot, wizardSpecDir)
	if _, err := fs.Stat(f, root); err != nil {
		return 0
	}

	registered := 0
	_ = fs.WalkDir(f, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}

		rel := strings.TrimPrefix(p, root+"/")
		relNoExt := strings.TrimSuffix(rel, path.Ext(rel))
		if !isWizardSpecPath(rel) {
			return nil
		}

		specPath := "embed://" + p
		spec, err := wizard.LoadSpec(specPath)
		if err != nil {
			logs.Log.Warnf("Failed to load wizard spec %s: %v\n", specPath, err)
			return nil
		}

		templateName := makeWizardTemplateName(pluginName, spec, relNoExt)
		if err := wizard.RegisterTemplateFromSpec(templateName, spec); err != nil {
			logs.Log.Warnf("Failed to register wizard template %s from %s: %v\n", templateName, specPath, err)
			return nil
		}
		registered++
		return nil
	})

	return registered
}

func registerWizardTemplatesFromDisk(pluginName, pluginPath string) int {
	root := filepath.Join(pluginPath, wizardSpecDir)
	info, err := os.Stat(root)
	if err != nil || info == nil || !info.IsDir() {
		return 0
	}

	registered := 0
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		relNoExt := strings.TrimSuffix(rel, path.Ext(rel))
		if !isWizardSpecPath(rel) {
			return nil
		}

		spec, err := wizard.LoadSpec(p)
		if err != nil {
			logs.Log.Warnf("Failed to load wizard spec %s: %v\n", p, err)
			return nil
		}

		templateName := makeWizardTemplateName(pluginName, spec, relNoExt)
		if err := wizard.RegisterTemplateFromSpec(templateName, spec); err != nil {
			logs.Log.Warnf("Failed to register wizard template %s from %s: %v\n", templateName, p, err)
			return nil
		}
		registered++
		return nil
	})

	return registered
}

func isWizardSpecPath(p string) bool {
	ext := strings.ToLower(path.Ext(p))
	switch ext {
	case ".yaml", ".yml", ".json":
		return true
	default:
		return false
	}
}

func makeWizardTemplateName(pluginName string, spec *wizard.WizardSpec, fallback string) string {
	base := strings.TrimSpace(fallback)
	if spec != nil && strings.TrimSpace(spec.ID) != "" {
		base = strings.TrimSpace(spec.ID)
	}
	if base == "" {
		base = "wizard"
	}
	base = strings.TrimPrefix(base, "/")
	base = strings.TrimPrefix(base, "./")

	prefix := strings.TrimSpace(pluginName)
	if prefix == "" {
		return base
	}
	if strings.HasPrefix(base, prefix+":") {
		return base
	}
	return prefix + ":" + base
}
