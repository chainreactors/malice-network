package armory

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/extension"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/cryptography/minisign"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/gookit/config/v2"
	yamlDriver "github.com/gookit/config/v2/yaml"
	"github.com/spf13/cobra"
)

func TestInstallPackageByNameReturnsBuildInstallListError(t *testing.T) {
	resetArmoryState(t)
	con := newArmoryTestConsole(t)

	err := installPackageByName("missing-command", "", false, false, ArmoryHTTPConfig{}, con)
	if !errors.Is(err, ErrPackageNotFound) {
		t.Fatalf("installPackageByName error = %v, want %v", err, ErrPackageNotFound)
	}
}

func TestPackageHashLookupByArmoryFiltersArmoryPackages(t *testing.T) {
	resetArmoryState(t)

	pkgCache.Store("pkg-a", pkgCacheEntry{
		ID:           "pkg-a",
		ArmoryConfig: &assets.ArmoryConfig{PublicKey: "armory-a"},
	})
	pkgCache.Store("pkg-b", pkgCacheEntry{
		ID:           "pkg-b",
		ArmoryConfig: &assets.ArmoryConfig{PublicKey: "armory-b"},
	})

	got := packageHashLookupByArmory("armory-a")
	if len(got) != 1 || got[0] != "pkg-a" {
		t.Fatalf("package hashes = %v, want [pkg-a]", got)
	}
}

func TestMakePackageCacheConsistentRemovesOnlyStalePackagesForArmory(t *testing.T) {
	resetArmoryState(t)

	livePkg := &ArmoryPackage{
		Name:        "live",
		CommandName: "live-cmd",
		RepoURL:     "https://example.com/live",
		PublicKey:   "live-pk",
		ArmoryName:  "armory-a",
	}
	stalePkg := &ArmoryPackage{
		Name:        "stale",
		CommandName: "stale-cmd",
		RepoURL:     "https://example.com/stale",
		PublicKey:   "stale-pk",
		ArmoryName:  "armory-a",
	}
	otherArmoryPkg := &ArmoryPackage{
		Name:        "other",
		CommandName: "other-cmd",
		RepoURL:     "https://example.com/other",
		PublicKey:   "other-pk",
		ArmoryName:  "armory-b",
	}

	liveID := calculatePackageHash(livePkg)
	staleID := calculatePackageHash(stalePkg)
	otherID := calculatePackageHash(otherArmoryPkg)

	pkgCache.Store(liveID, pkgCacheEntry{
		ID:           liveID,
		ArmoryConfig: &assets.ArmoryConfig{PublicKey: "armory-a"},
	})
	pkgCache.Store(staleID, pkgCacheEntry{
		ID:           staleID,
		ArmoryConfig: &assets.ArmoryConfig{PublicKey: "armory-a"},
	})
	pkgCache.Store(otherID, pkgCacheEntry{
		ID:           otherID,
		ArmoryConfig: &assets.ArmoryConfig{PublicKey: "armory-b"},
	})

	makePackageCacheConsistent(ArmoryIndex{
		ArmoryConfig: &assets.ArmoryConfig{PublicKey: "armory-a"},
		Aliases:      []*ArmoryPackage{livePkg},
	})

	if _, ok := pkgCache.Load(staleID); ok {
		t.Fatalf("stale package %q still present in cache", staleID)
	}
	if _, ok := pkgCache.Load(liveID); !ok {
		t.Fatalf("live package %q removed unexpectedly", liveID)
	}
	if _, ok := pkgCache.Load(otherID); !ok {
		t.Fatalf("package from other armory %q removed unexpectedly", otherID)
	}
}

func TestFetchPackageSignatureInvalidTrustedManifestDoesNotPanic(t *testing.T) {
	for _, tc := range []struct {
		name    string
		isAlias bool
	}{
		{name: "alias", isAlias: true},
		{name: "extension", isAlias: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetArmoryState(t)
			restore := stubArmoryPackageParser(t, "unit.test", func(*assets.ArmoryConfig, *ArmoryPackage, bool, ArmoryHTTPConfig) (*minisign.Signature, []byte, error) {
				return mustSignatureWithTrustedComment(t, []byte("payload"), []byte("not-json")), nil, nil
			})
			defer restore()

			pkg := &ArmoryPackage{
				ID:          "pkg-id",
				CommandName: "demo",
				RepoURL:     "https://unit.test/repo",
				IsAlias:     tc.isAlias,
			}
			wg := &sync.WaitGroup{}
			ch := make(chan struct{}, 1)
			wg.Add(1)
			ch <- struct{}{}

			fetchPackageSignature(wg, ch, &assets.ArmoryConfig{
				Name:      "Unit",
				PublicKey: "armory-pk",
			}, pkg, ArmoryHTTPConfig{Timeout: time.Second})
			wg.Wait()

			cached := packageCacheLookupByID("pkg-id")
			if cached != nil {
				t.Fatalf("invalid manifest should not be returned as a valid cache entry")
			}
			raw, ok := pkgCache.Load("pkg-id")
			if !ok {
				t.Fatalf("expected package cache entry to be stored")
			}
			entry := raw.(pkgCacheEntry)
			if entry.LastErr == nil || !strings.Contains(entry.LastErr.Error(), "failed to parse trusted manifest") {
				t.Fatalf("LastErr = %v, want trusted manifest parse error", entry.LastErr)
			}
		})
	}
}

func TestGithubAPIArmoryPackageParserRejectsMissingReleases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	defer server.Close()

	err := githubPackageParserError(server.URL, "demo")
	if err == nil || !strings.Contains(err.Error(), "no releases found") {
		t.Fatalf("parser error = %v, want no releases found", err)
	}
}

func TestGithubAPIArmoryPackageParserRejectsMissingAssets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		payload := []GithubRelease{{
			Assets: []GithubAsset{{
				Name: "other.txt",
				URL:  serverURLWithPath(r, "/asset"),
			}},
		}}
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			t.Fatalf("encode response failed: %v", err)
		}
	}))
	defer server.Close()

	err := githubPackageParserError(server.URL, "demo")
	if err == nil || !strings.Contains(err.Error(), "missing minisig asset") {
		t.Fatalf("parser error = %v, want missing minisig asset", err)
	}
}

func TestInstallExtensionPackageRegistersCommand(t *testing.T) {
	resetArmoryState(t)
	con := newArmoryTestConsole(t)

	manifestData, archiveData := mustExtensionPackage(t, extensionFixture{
		name:        "demo-extension",
		commandName: "demo-cmd",
		files: []fixtureFile{
			{manifestPath: `bin\demo.dll`, archivePath: "bin/demo.dll", content: []byte("dll")},
		},
	})
	pubText, sig := mustSignedArchive(t, archiveData, manifestData)

	restore := stubArmoryPackageParser(t, "unit.test", func(*assets.ArmoryConfig, *ArmoryPackage, bool, ArmoryHTTPConfig) (*minisign.Signature, []byte, error) {
		return sig, archiveData, nil
	})
	defer restore()

	manifest, err := extension.ParseExtensionManifest(manifestData)
	if err != nil {
		t.Fatalf("parse extension manifest failed: %v", err)
	}
	entry := &pkgCacheEntry{
		ID:           "pkg-demo",
		ArmoryConfig: &assets.ArmoryConfig{Name: "Unit", PublicKey: "armory-pk"},
		RepoURL:      "https://unit.test/repo",
		Pkg: ArmoryPackage{
			Name:        manifest.Name,
			CommandName: manifest.ExtCommand[0].CommandName,
			RepoURL:     "https://unit.test/repo",
			PublicKey:   pubText,
		},
		Extension: manifest,
	}

	if err := installExtensionPackage(entry, false, ArmoryHTTPConfig{Timeout: time.Second}, con); err != nil {
		t.Fatalf("installExtensionPackage failed: %v", err)
	}

	if !hasCommand(con.ImplantMenu(), "demo-cmd") {
		t.Fatalf("expected demo-cmd to be registered after install")
	}
	installPath := filepath.Join(assets.GetExtensionsDir(), "demo-extension")
	if _, err := os.Stat(filepath.Join(installPath, extension.ManifestFileName)); err != nil {
		t.Fatalf("installed manifest missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(installPath, fileutils.ResolvePath(`bin\demo.dll`))); err != nil {
		t.Fatalf("installed payload missing: %v", err)
	}
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
		t.Fatalf("profile did not record installed extension manifest name")
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

func newArmoryTestConsole(t testing.TB) *core.Console {
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

	if _, err := assets.LoadProfile(); err != nil {
		t.Fatalf("load profile failed: %v", err)
	}
	return con
}

func resetArmoryState(t testing.TB) {
	t.Helper()

	oldPkgCache := pkgCache
	oldIndexCache := indexCache
	oldCurrentArmories := currentArmories
	pkgCache = &sync.Map{}
	indexCache = &sync.Map{}
	currentArmories = &sync.Map{}
	t.Cleanup(func() {
		pkgCache = oldPkgCache
		indexCache = oldIndexCache
		currentArmories = oldCurrentArmories
	})
}

func stubArmoryPackageParser(t testing.TB, host string, parser ArmoryPackageParser) func() {
	t.Helper()

	oldParser, existed := pkgParsers[host]
	pkgParsers[host] = parser
	return func() {
		if existed {
			pkgParsers[host] = oldParser
		} else {
			delete(pkgParsers, host)
		}
	}
}

func githubPackageParserError(repoURL, commandName string) error {
	pub, _, err := minisign.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	pubText, err := pub.MarshalText()
	if err != nil {
		return err
	}
	_, _, err = GithubAPIArmoryPackageParser(&assets.ArmoryConfig{}, &ArmoryPackage{
		RepoURL:     repoURL,
		CommandName: commandName,
		PublicKey:   string(pubText),
	}, false, ArmoryHTTPConfig{Timeout: time.Second})
	return err
}

func mustSignedArchive(t testing.TB, archiveData, manifestData []byte) (string, *minisign.Signature) {
	t.Helper()

	pub, priv, err := minisign.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key failed: %v", err)
	}
	pubText, err := pub.MarshalText()
	if err != nil {
		t.Fatalf("marshal public key failed: %v", err)
	}
	sigText := minisign.SignWithComments(priv, archiveData, base64.StdEncoding.EncodeToString(manifestData), "")
	var sig minisign.Signature
	if err := sig.UnmarshalText(sigText); err != nil {
		t.Fatalf("unmarshal signature failed: %v", err)
	}
	return string(pubText), &sig
}

func mustSignatureWithTrustedComment(t testing.TB, payload, trustedCommentData []byte) *minisign.Signature {
	t.Helper()

	_, priv, err := minisign.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key failed: %v", err)
	}
	sigText := minisign.SignWithComments(priv, payload, base64.StdEncoding.EncodeToString(trustedCommentData), "")
	var sig minisign.Signature
	if err := sig.UnmarshalText(sigText); err != nil {
		t.Fatalf("unmarshal signature failed: %v", err)
	}
	return &sig
}

func mustExtensionPackage(t testing.TB, fixture extensionFixture) ([]byte, []byte) {
	t.Helper()

	manifestData := mustManifestJSON(t, fixture)
	var archive bytes.Buffer
	gzw := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gzw)

	addTarFile(t, tw, extension.ManifestFileName, manifestData)
	for _, artifact := range fixture.files {
		addTarFile(t, tw, artifact.archivePath, artifact.content)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar failed: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("close gzip failed: %v", err)
	}
	return manifestData, archive.Bytes()
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

func hasCommand(root *cobra.Command, name string) bool {
	for _, cmd := range root.Commands() {
		if cmd.Name() == name {
			return true
		}
	}
	return false
}

func serverURLWithPath(r *http.Request, path string) string {
	return "http://" + r.Host + path
}
