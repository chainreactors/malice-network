package assets

import (
	"log"
	"os"
	"path/filepath"
)

const (
	// ExtensionsDirName - Directory storing the client side extensions
	ExtensionsDirName = "extensions"
)

// GetExtensionsDir
func GetExtensionsDir() string {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	dir := filepath.Join(rootDir, ExtensionsDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			log.Fatal(err)
		}
	}
	return dir
}

// GetInstalledExtensionManifests - Returns a list of installed extension manifests
func GetInstalledExtensionManifests() []string {
	extDir := GetExtensionsDir()
	extDirContent, err := os.ReadDir(extDir)
	if err != nil {
		log.Printf("error loading aliases: %s", err)
		return []string{}
	}
	manifests := []string{}
	for _, fi := range extDirContent {
		if fi.IsDir() {
			manifestPath := filepath.Join(extDir, fi.Name(), "extension.json")
			if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
				log.Printf("no manifest in %s, skipping ...", manifestPath)
				continue
			}
			manifests = append(manifests, manifestPath)
		}
	}
	return manifests
}
