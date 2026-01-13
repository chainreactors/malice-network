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

// specWalker abstracts the file walking logic for different file systems
type specWalker struct {
	pluginName string
	registered int
}

func (w *specWalker) processSpec(specPath, rel string, loadFn func(string) (*wizard.WizardSpec, error)) {
	relNoExt := strings.TrimSuffix(rel, path.Ext(rel))
	if !isWizardSpecPath(rel) {
		return
	}

	spec, err := loadFn(specPath)
	if err != nil {
		logs.Log.Warnf("Failed to load wizard spec %s: %v\n", specPath, err)
		return
	}

	templateName := makeWizardTemplateName(w.pluginName, spec, relNoExt)
	if err := wizard.RegisterTemplateFromSpec(templateName, spec); err != nil {
		logs.Log.Warnf("Failed to register wizard template %s from %s: %v\n", templateName, specPath, err)
		return
	}
	w.registered++
}

func registerWizardTemplatesFromEmbedFS(pluginName, pluginRoot string, f fs.FS) int {
	root := path.Join(pluginRoot, wizardSpecDir)
	if _, err := fs.Stat(f, root); err != nil {
		return 0
	}

	w := &specWalker{pluginName: pluginName}
	_ = fs.WalkDir(f, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(p, root+"/")
		w.processSpec("embed://"+p, rel, wizard.LoadSpec)
		return nil
	})
	return w.registered
}

func registerWizardTemplatesFromDisk(pluginName, pluginPath string) int {
	root := filepath.Join(pluginPath, wizardSpecDir)
	if info, err := os.Stat(root); err != nil || info == nil || !info.IsDir() {
		return 0
	}

	w := &specWalker{pluginName: pluginName}
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}
		w.processSpec(p, filepath.ToSlash(rel), wizard.LoadSpec)
		return nil
	})
	return w.registered
}

func isWizardSpecPath(p string) bool {
	switch strings.ToLower(path.Ext(p)) {
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
	base = strings.TrimPrefix(strings.TrimPrefix(base, "/"), "./")

	prefix := strings.TrimSpace(pluginName)
	if prefix == "" || strings.HasPrefix(base, prefix+":") {
		return base
	}
	return prefix + ":" + base
}
