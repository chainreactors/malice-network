package armory

import (
	"errors"
	"fmt"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/command/alias"
	"github.com/chainreactors/malice-network/client/command/extension"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/cryptography/minisign"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
)

var (
	// ErrPackageNotFound - The package was not found
	ErrPackageNotFound = errors.New("package not found")
)

// ArmoryInstallCmd - The armory install command
func ArmoryInstallCmd(ctx *grumble.Context, con *console.Console) {
	name := ctx.Args.String("name")
	if name == "" {
		console.Log.Errorf("A package or bundle name is required")
		return
	}
	clientConfig := parseArmoryHTTPConfig(ctx)
	refresh(clientConfig)
	if name == "all" {
		aliases, extensions := packagesInCache()
		confirm := false
		prompt := &survey.Confirm{
			Message: fmt.Sprintf("Install %d aliases and %d extensions?",
				len(aliases), len(extensions),
			),
		}
		survey.AskOne(prompt, &confirm)
		if !confirm {
			return
		}
	}
	err := installPackageByName(name, clientConfig, con)
	if err == nil {
		return
	}
	if err == ErrPackageNotFound {
		bundles := bundlesInCache()
		for _, bundle := range bundles {
			if bundle.Name == name {
				installBundle(bundle, clientConfig, con)
				return
			}
		}
	}
	console.Log.Errorf("No package or bundle named '%s' was found", name)
}

func installBundle(bundle *ArmoryBundle, clientConfig ArmoryHTTPConfig, con *console.Console) {
	for _, pkgName := range bundle.Packages {
		err := installPackageByName(pkgName, clientConfig, con)
		if err != nil {
			console.Log.Errorf("Failed to install '%s': %s", pkgName, err)
		}
	}
}

func installPackageByName(name string, clientConfig ArmoryHTTPConfig, con *console.Console) error {
	aliases, extensions := packagesInCache()
	for _, alias := range aliases {
		if alias.CommandName == name || name == "all" {
			installAlias(alias, clientConfig, con)
			if name != "all" {
				return nil
			}
		}
	}
	for _, ext := range extensions {
		if ext.CommandName == name || name == "all" {
			installExtension(ext, clientConfig, con)
			if name != "all" {
				return nil
			}
		}
	}
	if name == "all" {
		console.Log.Infof("All packages installed\n")
		return nil
	}
	return ErrPackageNotFound
}

func installAlias(alias *alias.AliasManifest, clientConfig ArmoryHTTPConfig, con *console.Console) {
	err := installAliasPackageByName(alias.CommandName, clientConfig, con)
	if err != nil {
		console.Log.Errorf("Failed to install alias '%s': %s", alias.CommandName, err)
		return
	}
}

func installAliasPackageByName(name string, clientConfig ArmoryHTTPConfig, con *console.Console) error {
	var entry *pkgCacheEntry
	pkgCache.Range(func(key, value interface{}) bool {
		cacheEntry := value.(pkgCacheEntry)
		if cacheEntry.Pkg.CommandName == name {
			entry = &cacheEntry
			return false
		}
		return true
	})
	if entry == nil {
		return errors.New("package not found")
	}
	repoURL, err := url.Parse(entry.RepoURL)
	if err != nil {
		return err
	}

	console.Log.Infof("Downloading alias ...")

	var sig *minisign.Signature
	var tarGz []byte
	if pkgParser, ok := pkgParsers[repoURL.Hostname()]; ok {
		sig, tarGz, err = pkgParser(entry.ArmoryConfig, &entry.Pkg, false, clientConfig)
	} else {
		sig, tarGz, err = DefaultArmoryPkgParser(entry.ArmoryConfig, &entry.Pkg, false, clientConfig)
	}
	if err != nil {
		return err
	}

	var publicKey minisign.PublicKey
	publicKey.UnmarshalText([]byte(entry.Pkg.PublicKey))
	rawSig, _ := sig.MarshalText()
	valid := minisign.Verify(publicKey, tarGz, []byte(rawSig))
	if !valid {
		return errors.New("signature verification failed")
	}

	tmpFile, err := ioutil.TempFile("", "sliver-armory-")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.Write(tarGz)
	if err != nil {
		return err
	}
	tmpFile.Close()

	console.Log.Infof(console.Clearln + "\r") // Clear the line

	installPath := alias.InstallFromFile(tmpFile.Name(), true, con)
	if installPath == nil {
		return errors.New("failed to install alias")
	}
	_, err = alias.LoadAlias(filepath.Join(*installPath, alias.ManifestFileName), con)
	if err != nil {
		return err
	}
	return nil
}

func installExtension(ext *extension.ExtensionManifest, clientConfig ArmoryHTTPConfig, con *console.Console) {
	deps := make(map[string]struct{})
	resolveExtensionPackageDependencies(ext.CommandName, deps, clientConfig, con)
	for dep := range deps {
		if extension.CmdExists(dep, con.App) {
			continue // Dependency is already installed
		}
		err := installExtensionPackageByName(dep, clientConfig, con)
		if err != nil {
			console.Log.Errorf("Failed to install extension dependency '%s': %s", dep, err)
			return
		}
	}
	err := installExtensionPackageByName(ext.CommandName, clientConfig, con)
	if err != nil {
		console.Log.Errorf("Failed to install extension '%s': %s", ext.CommandName, err)
		return
	}
}

const maxDepDepth = 10 // Arbitrary recursive limit for dependencies

func resolveExtensionPackageDependencies(name string, deps map[string]struct{}, clientConfig ArmoryHTTPConfig, con *console.Console) {
	var entry *pkgCacheEntry
	pkgCache.Range(func(key, value interface{}) bool {
		cacheEntry := value.(pkgCacheEntry)
		if cacheEntry.Pkg.CommandName == name {
			entry = &cacheEntry
			return false
		}
		return true
	})
	if entry == nil {
		return
	}

	if entry.Extension.DependsOn == "" {
		return // Avoid adding empty dependency
	}

	if entry.Extension.DependsOn == name {
		return // Avoid infinite loop of something that depends on itself
	}
	// We also need to look out for circular dependencies, so if we've already
	// seen this dependency, we stop resolving
	if _, ok := deps[entry.Extension.DependsOn]; ok {
		return // Already resolved
	}
	if maxDepDepth < len(deps) {
		return
	}
	deps[entry.Extension.DependsOn] = struct{}{}
	resolveExtensionPackageDependencies(entry.Extension.DependsOn, deps, clientConfig, con)
}

func installExtensionPackageByName(name string, clientConfig ArmoryHTTPConfig, con *console.Console) error {
	var entry *pkgCacheEntry
	pkgCache.Range(func(key, value interface{}) bool {
		cacheEntry := value.(pkgCacheEntry)
		if cacheEntry.Pkg.CommandName == name {
			entry = &cacheEntry
			return false
		}
		return true
	})
	if entry == nil {
		return errors.New("package not found")
	}
	repoURL, err := url.Parse(entry.RepoURL)
	if err != nil {
		return err
	}

	console.Log.Infof("Downloading extension ...")

	var sig *minisign.Signature
	var tarGz []byte
	if pkgParser, ok := pkgParsers[repoURL.Hostname()]; ok {
		sig, tarGz, err = pkgParser(entry.ArmoryConfig, &entry.Pkg, false, clientConfig)
	} else {
		sig, tarGz, err = DefaultArmoryPkgParser(entry.ArmoryConfig, &entry.Pkg, false, clientConfig)
	}
	if err != nil {
		return err
	}

	var publicKey minisign.PublicKey
	publicKey.UnmarshalText([]byte(entry.Pkg.PublicKey))
	rawSig, _ := sig.MarshalText()
	valid := minisign.Verify(publicKey, tarGz, []byte(rawSig))
	if !valid {
		return errors.New("signature verification failed")
	}

	tmpFile, err := ioutil.TempFile("", "sliver-armory-")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.Write(tarGz)
	if err != nil {
		return err
	}
	err = tmpFile.Sync()
	if err != nil {
		return err
	}

	console.Log.Infof(console.Clearln + "\r") // Clear download message

	installPath := extension.InstallFromFilePath(tmpFile.Name(), true, con)
	if installPath == nil {
		return errors.New("failed to install extension")
	}
	extCmd, err := extension.LoadExtensionManifest(filepath.Join(*installPath, extension.ManifestFileName))
	if err != nil {
		return err
	}
	if extension.CmdExists(extCmd.Name, con.App) {
		con.App.Commands().Remove(extCmd.Name)
	}
	extension.ExtensionRegisterCommand(extCmd, con)
	return nil
}
