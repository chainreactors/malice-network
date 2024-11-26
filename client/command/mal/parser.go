package mal

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/helper/cryptography/minisign"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	malIndexFileName    = "mals.yaml"
	malIndexSigFileName = "mal.minisig"
)

type MalsYaml struct {
	Mals []assets.MalConfig `yaml:"mals"`
}

// GitHub API Parsers for Mal

type GithubAsset struct {
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	URL                string `json:"url"`
	Size               int    `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type GithubRelease struct {
	ID          int           `json:"id"`
	Name        string        `json:"name"`
	URL         string        `json:"url"`
	HTMLURL     string        `json:"html_url"`
	TagName     string        `json:"tag_name"`
	Body        string        `json:"body"`
	Prerelease  bool          `json:"prerelease"`
	TarballURL  string        `json:"tarball_url"`
	ZipballURL  string        `json:"zipball_url"`
	CreatedAt   string        `json:"created_at"`
	PublishedAt string        `json:"published_at"`
	Assets      []GithubAsset `json:"assets"`
}

func httpClient(config MalHTTPConfig) *http.Client {
	return &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: config.Timeout,
			}).Dial,
			IdleConnTimeout:     time.Millisecond,
			Proxy:               http.ProxyURL(config.ProxyURL),
			TLSHandshakeTimeout: config.Timeout,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.DisableTLSValidation,
			},
		},
	}
}

func httpRequest(clientConfig MalHTTPConfig, reqURL string, extraHeaders http.Header) (*http.Response, []byte, error) {
	client := httpClient(clientConfig)
	req, err := http.NewRequest(http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return nil, nil, err
	}

	if len(extraHeaders) > 0 {
		for key := range extraHeaders {
			req.Header.Add(key, strings.Join(extraHeaders[key], ","))
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	return resp, body, err
}

func downloadRequest(clientConfig MalHTTPConfig, reqURL string) ([]byte, error) {
	downloadHdr := http.Header{
		"Accept": {"application/octet-stream"},
	}
	resp, body, err := httpRequest(clientConfig, reqURL, downloadHdr)
	if resp == nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		return nil, fmt.Errorf("Error downloading asset: http %d", resp.StatusCode)
	}

	return body, err
}

func parsePkgMinsig(data []byte) (*minisign.Signature, error) {
	var sig minisign.Signature
	err := sig.UnmarshalText(data)
	if err != nil {
		return nil, err
	}
	if len(sig.TrustedComment) < 1 {
		return nil, errors.New("missing trusted comment")
	}
	return &sig, nil
}

// Intercepts 302 redirect to determine the latest version tag
func githubTagParser(repoUrl string, version string, clientConfig MalHTTPConfig) (string, error) {
	client := httpClient(clientConfig)
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	latestURL, err := url.Parse(repoUrl)
	if err != nil {
		return "", fmt.Errorf("failed to parse mal pkg url '%s': %s", repoUrl, err)
	}
	latestURL.Path = path.Join(latestURL.Path, "releases", version)
	latestRedirect, err := client.Get(latestURL.String())
	if err != nil {
		return "", fmt.Errorf("http get failed mal pkg url '%s': %s", repoUrl, err)
	}
	defer latestRedirect.Body.Close()
	if latestRedirect.StatusCode != http.StatusFound && latestRedirect.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected response status (wanted 302) '%s': %s", repoUrl, latestRedirect.Status)
	}
	if latestRedirect.Header.Get("Location") == "" {
		return "", fmt.Errorf("no location header in response '%s'", repoUrl)
	}
	latestLocationURL, err := url.Parse(latestRedirect.Header.Get("Location"))
	if err != nil {
		return "", fmt.Errorf("failed to parse location header '%s'->'%s': %s",
			repoUrl, latestRedirect.Header.Get("Location"), err)
	}
	pathSegments := strings.Split(latestLocationURL.Path, "/")
	for index, segment := range pathSegments {
		if segment == "tag" && index+1 < len(pathSegments) {
			return pathSegments[index+1], nil
		}
	}
	return "", errors.New("tag not found in location header")
}

func parserMalYaml(clientConfig MalHTTPConfig) (MalsYaml, error) {
	var malData MalsYaml
	resp, body, err := httpRequest(clientConfig, assets.DefaultMalRepoURL, http.Header{})
	if err != nil {
		return malData, err
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden {
			return malData, errors.New("GitHub API rate limit reached (60 req/hr), try again later")
		}
		return malData, errors.New(fmt.Sprintf("API returned non-200 status code: %d", resp.StatusCode))
	}

	releases := []GithubRelease{}
	err = json.Unmarshal(body, &releases)
	if err != nil {
		return malData, err
	}
	release := releases[0]

	var malsYaml []byte
	for _, asset := range release.Assets {
		if asset.Name == malIndexFileName {
			malsYaml, err = downloadRequest(clientConfig, asset.URL)
			if err != nil {
				break
			}
		}
	}

	err = yaml.Unmarshal(malsYaml, &malData)
	if err != nil {
		return malData, err
	}

	fileData, err := yaml.Marshal(&malData)
	if err != nil {
		return malData, fmt.Errorf("failed to marshal malData to YAML: %v", err)
	}

	filePath := filepath.Join(assets.GetConfigDir(), malIndexFileName)
	err = os.WriteFile(filePath, fileData, 0644)
	if err != nil {
		return malData, fmt.Errorf("failed to write YAML data to file: %v", err)
	}

	return malData, nil

}

// GithubMalPackageParser - Uses github.com instead of api.github.com to download packages
func GithubMalPackageParser(repoURL string, pkgName string, version string, clientConfig MalHTTPConfig) error {
	var tarGz []byte

	tarGzURL, err := url.Parse(repoURL)
	if err != nil {
		return fmt.Errorf("failed to parse mal pkg url '%s': %s", repoURL, err)
	}
	tarGzURL.Path = path.Join(tarGzURL.Path, "releases", "download", version, fmt.Sprintf("%s.tar.gz", pkgName))
	tarGz, err = downloadRequest(clientConfig, tarGzURL.String())
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(assets.GetMalsDir(), fmt.Sprintf("%s.tar.gz", pkgName)), tarGz, 0644)
	if err != nil {
		return err
	}
	return nil
}
