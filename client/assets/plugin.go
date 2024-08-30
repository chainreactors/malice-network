package assets

import (
	"github.com/chainreactors/logs"
	"os"
	"path/filepath"
)

const (
	AliasesDirName    = "aliases"
	ExtensionsDirName = "extensions"
	MalsDirName       = "mals"
)

// GetAliasesDir - Returns the path to the config dir
func GetAliasesDir() string {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	dir := filepath.Join(rootDir, AliasesDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			panic(err)
		}
	}
	return dir
}

// GetInstalledAliasManifests - Returns a list of installed alias manifests
func GetInstalledAliasManifests() []string {
	aliasDir := GetAliasesDir()
	aliasDirContent, err := os.ReadDir(aliasDir)
	if err != nil {
		logs.Log.Errorf("error loading aliases: %s", err)
		return []string{}
	}
	manifests := []string{}
	for _, fi := range aliasDirContent {
		if fi.IsDir() {
			manifestPath := filepath.Join(aliasDir, fi.Name(), "alias.json")
			if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
				logs.Log.Errorf("no manifest in %s, skipping ...", manifestPath)
				continue
			}
			manifests = append(manifests, manifestPath)
		}
	}
	return manifests
}

// GetExtensionsDir
func GetExtensionsDir() string {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	dir := filepath.Join(rootDir, ExtensionsDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			panic(err)
		}
	}
	return dir
}

// GetInstalledExtensionManifests - Returns a list of installed extension manifests
func GetInstalledExtensionManifests() []string {
	extDir := GetExtensionsDir()
	extDirContent, err := os.ReadDir(extDir)
	if err != nil {
		logs.Log.Errorf("error loading aliases: %s", err)
		return []string{}
	}
	manifests := []string{}
	for _, fi := range extDirContent {
		if fi.IsDir() {
			manifestPath := filepath.Join(extDir, fi.Name(), "extension.json")
			if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
				logs.Log.Errorf("no manifest in %s, skipping ...", manifestPath)
				continue
			}
			manifests = append(manifests, manifestPath)
		}
	}
	return manifests
}

func GetMalsDir() string {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	dir := filepath.Join(rootDir, MalsDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			panic(err)
		}
	}
	return dir
}

func GetInstalledMalManifests() []string {
	dir := GetMalsDir()
	dirList, err := os.ReadDir(dir)
	if err != nil {
		logs.Log.Errorf("error loading aliases: %s", err)
		return []string{}
	}
	manifests := []string{}
	for _, fi := range dirList {
		if fi.IsDir() {
			manifestPath := filepath.Join(dir, fi.Name(), "mal.yaml")
			if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
				logs.Log.Errorf("no manifest in %s, skipping ...", manifestPath)
				continue
			}
			manifests = append(manifests, manifestPath)
		}
	}
	return manifests
}
