package extension

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/gookit/config/v2"
	yamlDriver "github.com/gookit/config/v2/yaml"
	"github.com/spf13/cobra"
)

func TestInstallFromDirResolvesInstalledFileByManifestName(t *testing.T) {
	h := newExtensionHarness(t)
	srcDir := writeExtensionDir(t, extensionFixture{
		name:        "demo-extension",
		commandName: "demo-cmd",
		files: []fixtureFile{
			{manifestPath: `bin\demo.dll`, archivePath: "bin/demo.dll", content: []byte("dll")},
		},
	})

	installPath, err := InstallFromDir(srcDir, false, h.Console, false)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	manifest, err := LoadExtensionManifest(filepath.Join(installPath, ManifestFileName))
	if err != nil {
		t.Fatalf("load manifest failed: %v", err)
	}
	got, err := manifest.ExtCommand[0].getFileForTarget("windows", "amd64")
	if err != nil {
		t.Fatalf("getFileForTarget failed: %v", err)
	}

	want := filepath.Join(installPath, fileutils.ResolvePath(`bin\demo.dll`))
	if got != want {
		t.Fatalf("file path = %q, want %q", got, want)
	}
	if _, err := os.Stat(got); err != nil {
		t.Fatalf("installed file missing at %q: %v", got, err)
	}
}

func TestInstallFromDirForceOverwriteRemovesStaleFiles(t *testing.T) {
	h := newExtensionHarness(t)
	first := writeExtensionArchive(t, extensionFixture{
		name:        "demo-extension",
		commandName: "demo-cmd",
		files: []fixtureFile{
			{manifestPath: "bin/old.dll", archivePath: "bin/old.dll", content: []byte("old")},
		},
	})
	second := writeExtensionArchive(t, extensionFixture{
		name:        "demo-extension",
		commandName: "demo-cmd",
		files: []fixtureFile{
			{manifestPath: "bin/new.dll", archivePath: "bin/new.dll", content: []byte("new")},
		},
	})

	installPath, err := InstallFromDir(first, false, h.Console, true)
	if err != nil {
		t.Fatalf("first install failed: %v", err)
	}
	_, err = InstallFromDir(second, false, h.Console, true)
	if err != nil {
		t.Fatalf("second install failed: %v", err)
	}

	oldPath := filepath.Join(installPath, fileutils.ResolvePath("bin/old.dll"))
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("stale file still exists at %q", oldPath)
	}
	newPath := filepath.Join(installPath, fileutils.ResolvePath("bin/new.dll"))
	if data, err := os.ReadFile(newPath); err != nil || string(data) != "new" {
		t.Fatalf("new file = %q, err = %v, want %q", data, err, "new")
	}
}

func TestInstallFromDirTarGzAcceptsWindowsManifestPaths(t *testing.T) {
	h := newExtensionHarness(t)
	archivePath := writeExtensionArchive(t, extensionFixture{
		name:        "demo-extension",
		commandName: "demo-cmd",
		files: []fixtureFile{
			{manifestPath: `bin\payload.dll`, archivePath: "bin/payload.dll", content: []byte("payload")},
		},
	})

	installPath, err := InstallFromDir(archivePath, false, h.Console, true)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	payloadPath := filepath.Join(installPath, fileutils.ResolvePath(`bin\payload.dll`))
	if data, err := os.ReadFile(payloadPath); err != nil || string(data) != "payload" {
		t.Fatalf("payload = %q, err = %v, want payload", data, err)
	}
}

func TestRemoveExtensionByCommandNameRemovesManifestDirectoryAndProfileEntry(t *testing.T) {
	h := newExtensionHarness(t)
	srcDir := writeExtensionDir(t, extensionFixture{
		name:        "demo-extension",
		commandName: "demo-cmd",
		files: []fixtureFile{
			{manifestPath: "demo.dll", archivePath: "demo.dll", content: []byte("dll")},
		},
	})

	installPath, err := InstallFromDir(srcDir, false, h.Console, false)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}
	manifest, err := LoadExtensionManifest(filepath.Join(installPath, ManifestFileName))
	if err != nil {
		t.Fatalf("load manifest failed: %v", err)
	}
	ExtensionRegisterCommand(manifest.ExtCommand[0], h.Console.ImplantMenu(), h.Console)

	profile, err := assets.GetProfile()
	if err != nil {
		t.Fatalf("get profile failed: %v", err)
	}
	found := false
	for _, name := range profile.Extensions {
		if name == "demo-extension" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected profile to track manifest install name")
	}

	if err := RemoveExtensionByCommandName("demo-cmd", h.Console); err != nil {
		t.Fatalf("remove failed: %v", err)
	}

	if _, err := os.Stat(installPath); !os.IsNotExist(err) {
		t.Fatalf("install path still exists: %q", installPath)
	}
	if _, ok := loadedManifests["demo-extension"]; ok {
		t.Fatalf("loaded manifest still present after removal")
	}
	profile, err = assets.GetProfile()
	if err != nil {
		t.Fatalf("get profile failed: %v", err)
	}
	for _, name := range profile.Extensions {
		if name == "demo-extension" {
			t.Fatalf("profile still contains removed extension")
		}
	}
}

func TestGetInstalledManifestsIndexesCommandNames(t *testing.T) {
	h := newExtensionHarness(t)
	srcDir := writeExtensionDir(t, extensionFixture{
		name:        "demo-extension",
		commandName: "demo-cmd",
		files: []fixtureFile{
			{manifestPath: "demo.dll", archivePath: "demo.dll", content: []byte("dll")},
		},
	})

	installPath, err := InstallFromDir(srcDir, false, h.Console, false)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}
	manifest, err := LoadExtensionManifest(filepath.Join(installPath, ManifestFileName))
	if err != nil {
		t.Fatalf("load manifest failed: %v", err)
	}
	ExtensionRegisterCommand(manifest.ExtCommand[0], h.Console.ImplantMenu(), h.Console)

	installed := getInstalledManifests()
	if _, ok := installed["demo-extension"]; !ok {
		t.Fatalf("installed manifests missing manifest name key")
	}
	if _, ok := installed["demo-cmd"]; !ok {
		t.Fatalf("installed manifests missing command name key")
	}
}

type extensionFixture struct {
	name        string
	commandName string
	files       []fixtureFile
}

type fixtureFile struct {
	manifestPath string
	archivePath  string
	content      []byte
}

type extensionHarness struct {
	Console *core.Console
}

func newExtensionHarness(t testing.TB) *extensionHarness {
	t.Helper()

	oldLoadedExtensions := loadedExtensions
	oldLoadedManifests := loadedManifests
	loadedExtensions = map[string]*loadedExt{}
	loadedManifests = map[string]*ExtensionManifest{}
	t.Cleanup(func() {
		loadedExtensions = oldLoadedExtensions
		loadedManifests = oldLoadedManifests
	})

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

	if _, err := assets.LoadProfile(); err != nil {
		t.Fatalf("load profile failed: %v", err)
	}
	return &extensionHarness{Console: con}
}

func writeExtensionDir(t testing.TB, fixture extensionFixture) string {
	t.Helper()

	dir := t.TempDir()
	writeExtensionManifest(t, dir, fixture)
	for _, file := range fixture.files {
		targetPath := filepath.Join(dir, fileutils.ResolvePath(file.manifestPath))
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}
		if err := os.WriteFile(targetPath, file.content, 0o600); err != nil {
			t.Fatalf("write file failed: %v", err)
		}
	}
	return dir
}

func writeExtensionArchive(t testing.TB, fixture extensionFixture) string {
	t.Helper()

	archivePath := filepath.Join(t.TempDir(), fixture.name+".tar.gz")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive failed: %v", err)
	}
	defer file.Close()

	gzw := gzip.NewWriter(file)
	tw := tar.NewWriter(gzw)

	addTarFile(t, tw, ManifestFileName, mustManifestJSON(t, fixture))
	for _, artifact := range fixture.files {
		addTarFile(t, tw, artifact.archivePath, artifact.content)
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close tar failed: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("close gzip failed: %v", err)
	}
	return archivePath
}

func writeExtensionManifest(t testing.TB, dir string, fixture extensionFixture) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, ManifestFileName), mustManifestJSON(t, fixture), 0o600); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}
}

func mustManifestJSON(t testing.TB, fixture extensionFixture) []byte {
	t.Helper()

	files := make([]map[string]string, 0, len(fixture.files))
	for _, file := range fixture.files {
		files = append(files, map[string]string{
			"os":   "windows",
			"arch": "amd64",
			"path": file.manifestPath,
		})
	}

	manifest := map[string]any{
		"name":    fixture.name,
		"version": "1.0.0",
		"commands": []map[string]any{
			{
				"command_name": fixture.commandName,
				"help":         "demo help",
				"entrypoint":   "Run",
				"files":        files,
			},
		},
	}

	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest failed: %v", err)
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
