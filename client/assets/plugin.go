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
	var manifests []string
	aliases, err := GetAliases()
	if err != nil {
		logs.Log.Errorf("Failed to get aliases %s", err)
		return manifests
	}
	for _, alias := range aliases {
		manifestPath := filepath.Join(aliasDir, alias, "alias.json")
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			logs.Log.Errorf("no manifest in %s, skipping ...\n", manifestPath)
			continue
		}
		manifests = append(manifests, manifestPath)
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
	var manifests []string
	extensions, err := GetExtensions()
	if err != nil {
		logs.Log.Errorf("Failed to get extensions %s", err)
		return manifests
	}
	for _, extension := range extensions {
		manifestPath := filepath.Join(extDir, extension, "extension.json")
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			logs.Log.Errorf("no manifest in %s, skipping ...\n", manifestPath)
			continue
		}
		manifests = append(manifests, manifestPath)
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
	var manifests []string
	mals, err := GetMals()
	if err != nil {
		logs.Log.Errorf("Failed to get mals %s", err)
		return manifests
	}
	for _, mal := range mals {
		manifestPath := filepath.Join(dir, mal, "mal.yaml")
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			logs.Log.Errorf("no manifest in %s, skipping ...\n", manifestPath)
			continue
		}
		manifests = append(manifests, manifestPath)
	}
	return manifests
}
