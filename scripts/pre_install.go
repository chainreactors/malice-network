//go:build ignore

package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const tempDir = "temp_dir"

type asset struct {
	cache    string // filename in temp_dir
	target   string // destination in server/assets/
	url      string // remote download URL
	localSrc string // local source (professional mode override)
	isZip    bool
}

var sgnAssets = []asset{
	// Linux
	{
		cache:  "sgn-linux-amd64",
		target: filepath.Join("server", "assets", "linux", "amd64", "sgn"),
		url:    "https://github.com/moloch--/sgn/releases/download/v0.0.5/sgn_linux-amd64",
	},
	{
		cache:  "sgn-linux-arm64",
		target: filepath.Join("server", "assets", "linux", "arm64", "sgn"),
		url:    "https://github.com/moloch--/sgn/releases/download/v0.0.5/sgn_linux-arm64",
	},
	// Windows
	{
		cache:  "sgn-windows-amd64.exe",
		target: filepath.Join("server", "assets", "windows", "amd64", "sgn.exe"),
		url:    "https://github.com/moloch--/sgn/releases/download/v0.0.5/sgn_windows-amd64.exe",
	},
	{
		cache:  "sgn-windows-arm64.exe",
		target: filepath.Join("server", "assets", "windows", "arm64", "sgn.exe"),
		url:    "https://github.com/moloch--/sgn/releases/download/v0.0.5/sgn_windows-arm64.exe",
	},
	// Darwin
	{
		cache:  "sgn-darwin-amd64",
		target: filepath.Join("server", "assets", "darwin", "amd64", "sgn"),
		url:    "https://github.com/moloch--/sgn/releases/download/v0.0.5/sgn_darwin-amd64",
	},
	{
		cache:  "sgn-darwin-arm64",
		target: filepath.Join("server", "assets", "darwin", "arm64", "sgn"),
		url:    "https://github.com/moloch--/sgn/releases/download/v0.0.5/sgn_darwin-arm64",
	},
}

var communityMutant = []asset{
	// Linux (amd64 only)
	{
		cache:  "malefic-mutant-linux-amd64",
		target: filepath.Join("server", "assets", "linux", "amd64", "malefic-mutant"),
		url:    "https://github.com/chainreactors/malefic/releases/latest/download/malefic-mutant-x86_64-unknown-linux-musl",
	},
	// Windows (amd64 only)
	{
		cache:  "malefic-mutant-windows-amd64.exe",
		target: filepath.Join("server", "assets", "windows", "amd64", "malefic-mutant.exe"),
		url:    "https://github.com/chainreactors/malefic/releases/latest/download/malefic-mutant-x86_64-pc-windows-gnu.exe",
	},
	// Darwin (amd64 + arm64)
	{
		cache:  "malefic-mutant-darwin-amd64",
		target: filepath.Join("server", "assets", "darwin", "amd64", "malefic-mutant"),
		url:    "https://github.com/chainreactors/malefic/releases/latest/download/malefic-mutant-x86_64-apple-darwin",
	},
	{
		cache:  "malefic-mutant-darwin-arm64",
		target: filepath.Join("server", "assets", "darwin", "arm64", "malefic-mutant"),
		url:    "https://github.com/chainreactors/malefic/releases/latest/download/malefic-mutant-aarch64-apple-darwin",
	},
}

var professionalMutant = []asset{
	{
		target:   filepath.Join("server", "assets", "windows", "malefic-mutant.exe"),
		localSrc: filepath.Join("helper", "consts", "professional", "malefic-mutant.exe"),
	},
	{
		target:   filepath.Join("server", "assets", "linux", "malefic-mutant"),
		localSrc: filepath.Join("helper", "consts", "professional", "malefic-mutant"),
	},
}

func getAssets(professional bool) []asset {
	assets := append([]asset{}, sgnAssets...)
	if professional {
		assets = append(assets, professionalMutant...)
	} else {
		assets = append(assets, communityMutant...)
	}
	return assets
}

func hasFlag(flag string) bool {
	for _, arg := range os.Args[1:] {
		if arg == flag {
			return true
		}
	}
	return false
}

func main() {
	professional := hasFlag("--professional")
	assets := getAssets(professional)

	if hasFlag("--clean") {
		if err := os.RemoveAll(tempDir); err == nil {
			fmt.Printf("[removed] %s/\n", tempDir)
		}
		for _, a := range assets {
			if err := os.Remove(a.target); err == nil {
				fmt.Printf("[removed] %s\n", a.target)
			}
		}
		return
	}

	os.MkdirAll(tempDir, 0o755)
	for _, a := range assets {
		if _, err := os.Stat(a.target); err == nil {
			fmt.Printf("[skip] %s already exists\n", a.target)
			continue
		}
		os.MkdirAll(filepath.Dir(a.target), 0o755)
		// local source: copy directly, no cache needed
		if a.localSrc != "" {
			fmt.Printf("[copy] %s -> %s\n", a.localSrc, a.target)
			if err := copyFile(a.localSrc, a.target); err != nil {
				fmt.Fprintf(os.Stderr, "copy: %v\n", err)
				os.Exit(1)
			}
			continue
		}
		// remote source: download to cache, then copy
		cachePath := filepath.Join(tempDir, a.cache)
		if _, err := os.Stat(cachePath); err != nil {
			fmt.Printf("[download] %s\n", a.url)
			if a.isZip {
				if err := downloadAndExtract(a.url, cachePath); err != nil {
					fmt.Fprintf(os.Stderr, "download+extract: %v\n", err)
					os.Exit(1)
				}
			} else {
				if err := downloadFile(a.url, cachePath); err != nil {
					fmt.Fprintf(os.Stderr, "download: %v\n", err)
					os.Exit(1)
				}
			}
		} else {
			fmt.Printf("[cache hit] %s\n", cachePath)
		}
		if err := copyFile(cachePath, a.target); err != nil {
			fmt.Fprintf(os.Stderr, "copy: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[installed] %s\n", a.target)
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func downloadAndExtract(url, targetFile string) error {
	zipPath := targetFile + ".zip"
	if err := downloadFile(url, zipPath); err != nil {
		return err
	}
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	baseName := filepath.Base(targetFile)
	for _, zf := range r.File {
		if filepath.Base(zf.Name) == baseName {
			rc, err := zf.Open()
			if err != nil {
				return err
			}
			out, err := os.Create(targetFile)
			if err != nil {
				rc.Close()
				return err
			}
			_, err = io.Copy(out, rc)
			rc.Close()
			out.Close()
			return err
		}
	}
	return fmt.Errorf("%s not found in zip", baseName)
}
