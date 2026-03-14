package armory

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/extension"
)

func TestRealGitHubArmoryExtensionInstallSmoke(t *testing.T) {
	requireRealGitHub(t)
	resetArmoryState(t)
	con := newArmoryTestConsole(t)

	armoryConfig := &assets.ArmoryConfig{
		Name:      assets.DefaultArmoryName,
		PublicKey: assets.DefaultArmoryPublicKey,
		RepoURL:   assets.DefaultArmoryRepoURL,
		Enabled:   true,
	}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		armoryConfig.Authorization = "Bearer " + token
	}
	clientConfig := ArmoryHTTPConfig{Timeout: 2 * time.Minute}

	index, err := GithubAPIArmoryIndexParser(armoryConfig, clientConfig)
	if err != nil {
		t.Fatalf("fetch live armory index failed: %v", err)
	}
	if len(index.Extensions) == 0 {
		t.Fatalf("live armory index returned no extension packages")
	}

	var failures []string
	for _, pkg := range index.Extensions {
		t.Logf("trying live armory extension package %q from %s", pkg.CommandName, pkg.RepoURL)

		repoURL, err := url.Parse(pkg.RepoURL)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: parse repo url: %v", pkg.CommandName, err))
			continue
		}
		parser, ok := pkgParsers[repoURL.Hostname()]
		if !ok {
			parser = DefaultArmoryPkgParser
		}

		sig, _, err := parser(armoryConfig, pkg, true, clientConfig)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: fetch minisig: %v", pkg.CommandName, err))
			continue
		}
		if sig == nil {
			failures = append(failures, fmt.Sprintf("%s: nil minisig", pkg.CommandName))
			continue
		}

		manifestData, err := base64.StdEncoding.DecodeString(sig.TrustedComment)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: decode trusted comment: %v", pkg.CommandName, err))
			continue
		}
		manifest, err := extension.ParseExtensionManifest(manifestData)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: parse extension manifest: %v", pkg.CommandName, err))
			continue
		}

		entry := &pkgCacheEntry{
			ArmoryConfig: armoryConfig,
			RepoURL:      pkg.RepoURL,
			Pkg:          *pkg,
			Extension:    manifest,
		}
		if err := installExtensionPackage(entry, false, clientConfig, con); err != nil {
			failures = append(failures, fmt.Sprintf("%s: install: %v", pkg.CommandName, err))
			continue
		}

		registered := false
		for _, cmd := range manifest.ExtCommand {
			if hasCommand(con.ImplantMenu(), cmd.CommandName) {
				registered = true
				break
			}
		}
		if !registered {
			failures = append(failures, fmt.Sprintf("%s: install completed but no command registered", pkg.CommandName))
			continue
		}

		return
	}

	if len(failures) > 6 {
		failures = failures[:6]
	}
	t.Fatalf("no live armory extension package installed successfully: %s", strings.Join(failures, " | "))
}

func requireRealGitHub(t testing.TB) {
	t.Helper()

	if os.Getenv("MALICE_REAL_GITHUB_TESTS") == "" {
		t.Skip("set MALICE_REAL_GITHUB_TESTS=1 to run real GitHub smoke tests")
	}
}
