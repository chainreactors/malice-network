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
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
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

func TestRefreshMalReloadsProfileChangedByAnotherClient(t *testing.T) {
	con := newMalTestConsole(t, true)
	config.Set("mals", []string{"stale-client-value"})

	profilePath := filepath.Join(assets.GetRootAppDir(), "malice.yaml")
	if err := os.WriteFile(profilePath, []byte("mals:\n  - other-client-value\n"), 0o600); err != nil {
		t.Fatalf("write profile from other client: %v", err)
	}

	if err := RefreshMalCmd(&cobra.Command{}, con); err != nil {
		t.Fatalf("RefreshMalCmd failed: %v", err)
	}
	profile, err := assets.GetProfile()
	if err != nil {
		t.Fatalf("GetProfile failed: %v", err)
	}
	if len(profile.Mals) != 1 || profile.Mals[0] != "other-client-value" {
		t.Fatalf("profile mals = %v, want other-client-value", profile.Mals)
	}
}

func TestRegisterAndUnregisterMalPluginUpdatesRuntimeState(t *testing.T) {
	con := newMalTestConsole(t, false)
	con.Server = &core.Server{ServerState: &iomclient.ServerState{
		EventHook: map[iomclient.EventCondition][]iomclient.OnEventFunc{},
	}}
	index, err := core.NewSearchIndex(filepath.Join(t.TempDir(), "search.db"))
	if err != nil {
		t.Fatalf("create search index: %v", err)
	}
	t.Cleanup(func() { _ = index.Close() })
	con.SearchIndex = index

	condition := iomclient.EventCondition{Type: "mal-test-event"}
	hook := func(*clientpb.Event) (bool, error) { return false, nil }
	cmd := &cobra.Command{
		Use:         "dynamic-mal-command",
		Short:       "dynamic registration test",
		Annotations: map[string]string{"depend": "execute"},
	}
	plug := &fakeMalPlugin{
		manifest: &plugin.MalManiFest{Name: "dynamic-mal"},
		commands: plugin.Commands{
			cmd.Name(): {Name: cmd.Name(), Command: cmd},
		},
		events: map[iomclient.EventCondition]iomclient.OnEventFunc{condition: hook},
	}

	if err := registerMalPlugin(con, con.ImplantMenu(), plug); err != nil {
		t.Fatalf("register mal plugin: %v", err)
	}
	if cmd.Parent() != con.ImplantMenu() || cmd.GroupID != plug.manifest.Name {
		t.Fatalf("registered command parent/group = %v/%q", cmd.Parent(), cmd.GroupID)
	}
	if cmd.Annotations["menu"] != consts.ImplantMenu || cmd.Annotations["source"] != "mal" {
		t.Fatalf("registered command annotations = %#v", cmd.Annotations)
	}
	if con.CMDs[cmd.Name()] != cmd || con.Helpers["execute"] != cmd {
		t.Fatal("registered command is missing from console runtime indexes")
	}
	if len(con.EventHook[condition]) != 1 {
		t.Fatalf("registered event hooks = %d, want 1", len(con.EventHook[condition]))
	}
	results, err := index.Search(cmd.Name(), "", "", 10)
	if err != nil || len(results) != 1 {
		t.Fatalf("search registered command = %#v, err = %v", results, err)
	}

	unregisterMalPlugin(con, con.ImplantMenu(), plug)
	if cmd.Parent() != nil {
		t.Fatal("unregistered command remains attached")
	}
	if _, ok := con.CMDs[cmd.Name()]; ok {
		t.Fatal("unregistered command remains in console CMDs")
	}
	if _, ok := con.Helpers["execute"]; ok {
		t.Fatal("unregistered command remains in console Helpers")
	}
	if len(con.EventHook[condition]) != 0 {
		t.Fatalf("event hooks after unregister = %d, want 0", len(con.EventHook[condition]))
	}
	results, err = index.Search(cmd.Name(), "", "", 10)
	if err != nil || len(results) != 0 {
		t.Fatalf("search unregistered command = %#v, err = %v", results, err)
	}
}

func TestRegisterMalPluginAppliesActiveSessionVisibility(t *testing.T) {
	con := newMalTestConsole(t, false)
	activeTarget := &iomclient.ActiveTarget{}
	activeTarget.Set(&iomclient.Session{Session: &clientpb.Session{
		Type: "malefic",
		Os:   &implantpb.Os{Name: "linux", Arch: "x64"},
	}})
	con.Server = &core.Server{ServerState: &iomclient.ServerState{
		ActiveTarget: activeTarget,
		EventHook:    map[iomclient.EventCondition][]iomclient.OnEventFunc{},
	}}
	cmd := &cobra.Command{
		Use:         "windows-dynamic-command",
		Annotations: map[string]string{"os": "windows"},
	}
	plug := &fakeMalPlugin{
		manifest: &plugin.MalManiFest{Name: "windows-mal"},
		commands: plugin.Commands{cmd.Name(): {Name: cmd.Name(), Command: cmd}},
	}

	if err := registerMalPlugin(con, con.ImplantMenu(), plug); err != nil {
		t.Fatalf("register mal plugin: %v", err)
	}
	if !cmd.Hidden {
		t.Fatal("incompatible dynamic command is visible for the active Linux session")
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

type fakeMalPlugin struct {
	manifest *plugin.MalManiFest
	commands plugin.Commands
	events   map[iomclient.EventCondition]iomclient.OnEventFunc
}

func (p *fakeMalPlugin) Run() error                    { return nil }
func (p *fakeMalPlugin) Manifest() *plugin.MalManiFest { return p.manifest }
func (p *fakeMalPlugin) Commands() plugin.Commands     { return p.commands }
func (p *fakeMalPlugin) Destroy() error                { return nil }
func (p *fakeMalPlugin) GetEvents() map[iomclient.EventCondition]iomclient.OnEventFunc {
	return p.events
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
