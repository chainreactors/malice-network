package assets

import (
	"log"
	"os"
	"path/filepath"
)

const (
	AliasesDirName = "aliases"
)

// GetAliasesDir - Returns the path to the config dir
func GetAliasesDir() string {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	dir := filepath.Join(rootDir, AliasesDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			log.Fatal(err)
		}
	}
	return dir
}

// GetInstalledAliasManifests - Returns a list of installed alias manifests
func GetInstalledAliasManifests() []string {
	aliasDir := GetAliasesDir()
	aliasDirContent, err := os.ReadDir(aliasDir)
	if err != nil {
		log.Printf("error loading aliases: %s", err)
		return []string{}
	}
	manifests := []string{}
	for _, fi := range aliasDirContent {
		if fi.IsDir() {
			manifestPath := filepath.Join(aliasDir, fi.Name(), "alias.json")
			if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
				log.Printf("no manifest in %s, skipping ...", manifestPath)
				continue
			}
			manifests = append(manifests, manifestPath)
		}
	}
	return manifests
}
