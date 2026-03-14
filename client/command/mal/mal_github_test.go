package mal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/plugin"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/mals/m"
)

func TestRealGitHubMalInstallSmoke(t *testing.T) {
	requireRealGitHub(t)
	con := newMalTestConsole(t, false)

	releases, err := fetchGitHubReleases("https://api.github.com/repos/chainreactors/mal-community/releases")
	if err != nil {
		t.Fatalf("fetch live mal releases failed: %v", err)
	}
	if len(releases) == 0 {
		t.Fatalf("live mal repo returned no releases")
	}

	var failures []string
	for _, release := range releases {
		for _, asset := range release.Assets {
			if !strings.HasSuffix(asset.Name, ".tar.gz") {
				continue
			}
			name := strings.TrimSuffix(asset.Name, ".tar.gz")
			t.Logf("trying live mal package %q from release %q", name, release.TagName)

			err := m.GithubMalPackageParser(RepoUrl, name, release.TagName, assets.GetMalsDir(), m.MalHTTPConfig{
				Timeout: 2 * time.Minute,
			})
			if err != nil {
				failures = append(failures, fmt.Sprintf("%s@%s: download: %v", name, release.TagName, err))
				continue
			}

			archivePath := filepath.Join(assets.GetMalsDir(), asset.Name)
			manifestData, err := fileutils.ReadFileFromTarGz(archivePath, m.ManifestFileName)
			if err != nil {
				failures = append(failures, fmt.Sprintf("%s@%s: read manifest: %v", name, release.TagName, err))
				continue
			}
			manifest, err := plugin.ParseMalManifest(manifestData)
			if err != nil {
				failures = append(failures, fmt.Sprintf("%s@%s: parse manifest: %v", name, release.TagName, err))
				continue
			}

			updated, err := InstallFromDir(archivePath, false, con, nil)
			if err != nil {
				failures = append(failures, fmt.Sprintf("%s@%s: install: %v", name, release.TagName, err))
				continue
			}
			if !updated {
				failures = append(failures, fmt.Sprintf("%s@%s: install returned no update", name, release.TagName))
				continue
			}

			if _, err := os.Stat(filepath.Join(assets.GetMalsDir(), manifest.Name, m.ManifestFileName)); err != nil {
				failures = append(failures, fmt.Sprintf("%s@%s: manifest missing after install: %v", name, release.TagName, err))
				continue
			}

			return
		}
	}

	if len(failures) > 6 {
		failures = failures[:6]
	}
	t.Fatalf("no live mal package installed successfully: %s", strings.Join(failures, " | "))
}

func fetchGitHubReleases(apiURL string) ([]m.GithubRelease, error) {
	req, err := http.NewRequest(http.MethodGet, apiURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := (&http.Client{Timeout: 2 * time.Minute}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned %s", resp.Status)
	}

	var releases []m.GithubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}
	return releases, nil
}

func requireRealGitHub(t testing.TB) {
	t.Helper()

	if os.Getenv("MALICE_REAL_GITHUB_TESTS") == "" {
		t.Skip("set MALICE_REAL_GITHUB_TESTS=1 to run real GitHub smoke tests")
	}
}
