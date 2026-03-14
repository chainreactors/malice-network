package mal

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/plugin"
	"github.com/chainreactors/mals/m"
	"github.com/gookit/config/v2"
	yamlDriver "github.com/gookit/config/v2/yaml"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func TestUpdateMalReturnsErrorWhenPluginIsNotLoaded(t *testing.T) {
	con := newMalTestConsole(t, true)

	err := updateMal(con, "missing-mal-update", m.MalHTTPConfig{})
	if err == nil || !strings.Contains(err.Error(), "is not loaded") {
		t.Fatalf("updateMal error = %v, want missing mal error", err)
	}
}

func TestInstallFromDirTarGzInstallsMalArchive(t *testing.T) {
	con := newMalTestConsole(t, false)
	archivePath := writeTarGzMalArchive(t, malFixture{
		name:    "demo-mal",
		version: "1.0.0",
		files: []malFile{
			{name: "main.lua", content: []byte("return {}")},
		},
	})

	updated, err := InstallFromDir(archivePath, false, con, nil)
	if err != nil {
		t.Fatalf("InstallFromDir tar.gz failed: %v", err)
	}
	if !updated {
		t.Fatalf("InstallFromDir tar.gz reported no update")
	}

	installPath := filepath.Join(assets.GetMalsDir(), "demo-mal")
	if _, err := os.Stat(filepath.Join(installPath, "mal.yaml")); err != nil {
		t.Fatalf("installed manifest missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(installPath, "main.lua")); err != nil {
		t.Fatalf("installed entry file missing: %v", err)
	}
}

func TestInstallFromDirZipInstallsMalArchive(t *testing.T) {
	con := newMalTestConsole(t, false)
	archivePath := writeZipMalArchive(t, malFixture{
		name:    "demo-mal",
		version: "1.0.0",
		files: []malFile{
			{name: "main.lua", content: []byte("return {}")},
		},
	})

	updated, err := InstallFromDir(archivePath, false, con, nil)
	if err != nil {
		t.Fatalf("InstallFromDir zip failed: %v", err)
	}
	if !updated {
		t.Fatalf("InstallFromDir zip reported no update")
	}

	installPath := filepath.Join(assets.GetMalsDir(), "demo-mal")
	if _, err := os.Stat(filepath.Join(installPath, "mal.yaml")); err != nil {
		t.Fatalf("installed manifest missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(installPath, "main.lua")); err != nil {
		t.Fatalf("installed entry file missing: %v", err)
	}
}

func TestInstallFromDirSkipsIdenticalManifest(t *testing.T) {
	con := newMalTestConsole(t, false)
	archivePath := writeTarGzMalArchive(t, malFixture{
		name:    "demo-mal",
		version: "1.0.0",
		files: []malFile{
			{name: "main.lua", content: []byte("return {}")},
		},
	})

	firstUpdated, err := InstallFromDir(archivePath, false, con, nil)
	if err != nil {
		t.Fatalf("first install failed: %v", err)
	}
	if !firstUpdated {
		t.Fatalf("first install reported no update")
	}

	secondUpdated, err := InstallFromDir(archivePath, false, con, nil)
	if err != nil {
		t.Fatalf("second install failed: %v", err)
	}
	if secondUpdated {
		t.Fatalf("second install should have been skipped for identical manifest")
	}
}

func TestInstallFromDirLibMovesResourcesDirectory(t *testing.T) {
	con := newMalTestConsole(t, false)
	archivePath := writeTarGzMalArchive(t, malFixture{
		name:    "demo-lib",
		version: "1.0.0",
		lib:     true,
		files: []malFile{
			{name: "main.lua", content: []byte("return {}")},
			{name: "resources/tool.txt", content: []byte("payload")},
		},
	})

	updated, err := InstallFromDir(archivePath, false, con, nil)
	if err != nil {
		t.Fatalf("InstallFromDir lib failed: %v", err)
	}
	if !updated {
		t.Fatalf("InstallFromDir lib reported no update")
	}

	resourcePath := filepath.Join(assets.GetResourceDir(), "tool.txt")
	if data, err := os.ReadFile(resourcePath); err != nil || string(data) != "payload" {
		t.Fatalf("resource file = %q, err = %v, want payload", data, err)
	}
	if _, err := os.Stat(filepath.Join(assets.GetMalsDir(), "demo-lib", "resources")); !os.IsNotExist(err) {
		t.Fatalf("resources directory still exists in mal install path")
	}
}

type malFixture struct {
	name    string
	version string
	lib     bool
	files   []malFile
}

type malFile struct {
	name    string
	content []byte
}

func newMalTestConsole(t testing.TB, withManager bool) *core.Console {
	t.Helper()

	oldMaliceDirName := assets.MaliceDirName
	config.Reset()
	config.WithOptions(func(opt *config.Options) {
		opt.DecoderConfig.TagName = "config"
		opt.ParseDefault = true
	}, config.WithHookFunc(assets.HookFn))
	config.AddDriver(yamlDriver.Driver)
	assets.MaliceDirName = t.TempDir()
	assets.InitLogDir()
	t.Cleanup(func() {
		assets.MaliceDirName = oldMaliceDirName
		assets.InitLogDir()
		config.Reset()
	})

	con := &core.Console{
		Log:     iomclient.Log,
		CMDs:    map[string]*cobra.Command{},
		Helpers: map[string]*cobra.Command{},
	}
	con.NewConsole()
	con.App.Menu(consts.ClientMenu).Command = &cobra.Command{Use: "client"}
	con.App.Menu(consts.ImplantMenu).Command = &cobra.Command{Use: "implant"}
	if withManager {
		con.MalManager = plugin.GetGlobalMalManager()
	}

	if _, err := assets.LoadProfile(); err != nil {
		t.Fatalf("load profile failed: %v", err)
	}
	return con
}

func writeTarGzMalArchive(t testing.TB, fixture malFixture) string {
	t.Helper()

	var archive bytes.Buffer
	gzw := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gzw)

	addTarFile(t, tw, "mal.yaml", mustMalManifestYAML(t, fixture))
	for _, file := range fixture.files {
		addTarFile(t, tw, file.name, file.content)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar failed: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("close gzip failed: %v", err)
	}

	archivePath := filepath.Join(t.TempDir(), fixture.name+".tar.gz")
	if err := os.WriteFile(archivePath, archive.Bytes(), 0o600); err != nil {
		t.Fatalf("write archive failed: %v", err)
	}
	return archivePath
}

func writeZipMalArchive(t testing.TB, fixture malFixture) string {
	t.Helper()

	archivePath := filepath.Join(t.TempDir(), fixture.name+".zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create zip failed: %v", err)
	}
	defer file.Close()

	zw := zip.NewWriter(file)
	addZipFile(t, zw, "mal.yaml", mustMalManifestYAML(t, fixture))
	for _, artifact := range fixture.files {
		addZipFile(t, zw, artifact.name, artifact.content)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip failed: %v", err)
	}
	return archivePath
}

func mustMalManifestYAML(t testing.TB, fixture malFixture) []byte {
	t.Helper()

	data, err := yaml.Marshal(&plugin.MalManiFest{
		Name:      fixture.name,
		Type:      plugin.LuaScript,
		Version:   fixture.version,
		EntryFile: "main.lua",
		Lib:       fixture.lib,
	})
	if err != nil {
		t.Fatalf("marshal mal manifest failed: %v", err)
	}
	return data
}

func addTarFile(t testing.TB, tw *tar.Writer, name string, content []byte) {
	t.Helper()

	header := &tar.Header{
		Name: name,
		Mode: 0o600,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatalf("write tar header failed: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("write tar body failed: %v", err)
	}
}

func addZipFile(t testing.TB, zw *zip.Writer, name string, content []byte) {
	t.Helper()

	writer, err := zw.Create(name)
	if err != nil {
		t.Fatalf("create zip entry failed: %v", err)
	}
	if _, err := writer.Write(content); err != nil {
		t.Fatalf("write zip entry failed: %v", err)
	}
}
