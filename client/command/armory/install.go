package armory

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/alias"
	"github.com/chainreactors/malice-network/client/command/extension"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/cryptography/minisign"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
)

var (
	// ErrPackageNotFound - The package was not found
	ErrPackageNotFound         = errors.New("package not found")
	ErrPackageAlreadyInstalled = errors.New("package is already installed")
)

const (
	doNotInstallOption      = "Do not install this package"
	doNotInstallPackageName = "do not install"
)

// ArmoryInstallCmd - The armory install command
func ArmoryInstallCmd(cmd *cobra.Command, con *repl.Console) {
	var promptToOverwrite bool
	name := cmd.Flags().Arg(0)
	if name == "" {
		con.Log.Errorf("A package or bundle name is required\n")
		return
	}
	forceInstallation, _ := cmd.Flags().GetBool("force")
	if forceInstallation {
		promptToOverwrite = false
	} else {
		promptToOverwrite = true
	}

	armoryName, _ := cmd.Flags().GetString("armory")
	if armoryName == "Default" {
		armoriesConfig := getCurrentArmoryConfiguration()
		if len(armoriesConfig) == 1 {
			con.Log.Infof("Reading armory index ... \n")
		} else {
			con.Log.Infof("Reading %d armory indexes ... \n", len(armoriesConfig))
		}
		clientConfig := parseArmoryHTTPConfig(cmd)
		indexes := fetchIndexes(clientConfig)
		if len(indexes) != len(armoriesConfig) {
			con.Log.Infof("errors!\n")
			indexCache.Range(func(key, value interface{}) bool {
				cacheEntry := value.(indexCacheEntry)
				if cacheEntry.LastErr != nil {
					con.Log.Errorf("%s - %s\n", cacheEntry.RepoURL, cacheEntry.LastErr)
				}
				return true
			})
		} else {
			con.Log.Infof("done!\n")
		}
		armoriesInitialized = true
		if len(indexes) == 0 {
			con.Log.Infof("No indexes found\n")
			return
		}
	}
	// Find PK for the armory name
	armoryPK := getArmoryPublicKey(armoryName)
	if armoryPK == "" {
		con.Log.Warnf("Armory '%s' not found\n", armoryName)
		//return
	}

	clientConfig := parseArmoryHTTPConfig(cmd)
	refresh(clientConfig)
	if name == "all" {
		aliases, extensions := packageManifestsInCache()
		aliasCount, extCount := countUniqueCommandsFromManifests(aliases, extensions)
		pluralAliases := "es"
		if aliasCount == 1 {
			pluralAliases = ""
		}
		pluralExtensions := "s"
		if extCount == 1 {
			pluralExtensions = ""
		}
		confirmModel := tui.NewConfirm(fmt.Sprintf("Install %d alias%s and %d extension%s?",
			aliasCount, pluralAliases, extCount, pluralExtensions,
		))
		newconfirm := tui.NewModel(confirmModel, nil, false, true)
		err := newconfirm.Run()
		if err != nil {
			con.Log.Errorf("Error running confirm model: %s\n", err)
			return
		}
		if !confirmModel.Confirmed {
			return
		}
		promptToOverwrite = false
	}
	err := installPackageByName(name, armoryPK, forceInstallation, promptToOverwrite, clientConfig, con)
	if err == nil {
		con.Log.Infof("\n%s install complete\n", name)
		return
	}
	if errors.Is(err, ErrPackageNotFound) {
		bundles := bundlesInCache()
		for _, bundle := range bundles {
			if bundle.Name == name {
				installBundle(bundle, armoryPK, forceInstallation, clientConfig, con)
				return
			}
		}
		if armoryPK == "" {
			con.Log.Errorf("No package or bundle named '%s' was found\n", name)
		} else {
			con.Log.Errorf("No package or bundle named '%s' was found for armory '%s'\n", name, armoryName)
		}
	} else if errors.Is(err, ErrPackageAlreadyInstalled) {
		con.Log.Errorf("Package %q is already installed - use the force option to overwrite it\n", name)
	} else {
		con.Log.Errorf("Could not install package: %s\n", err)
	}
}

func installBundle(bundle *ArmoryBundle, armoryPK string, forceInstallation bool, clientConfig ArmoryHTTPConfig,
	con *repl.Console) {
	installList := []string{}
	pendingPackages := make(map[string]string)

	for _, bundlePkgName := range bundle.Packages {
		packageInstallList, err := buildInstallList(bundlePkgName, armoryPK, forceInstallation, pendingPackages)
		if err != nil {
			if errors.Is(err, ErrPackageAlreadyInstalled) {
				con.Log.Infof("Package %s is already installed. Skipping...\n", bundlePkgName)
				continue
			} else {
				con.Log.Errorf("Failed to install package %s: %s\n", bundlePkgName, err)
			}
		}
		for _, pkgID := range packageInstallList {
			if !slices.Contains(installList, pkgID) {
				installList = append(installList, pkgID)
			}
		}
	}

	for _, packageID := range installList {
		packageEntry := packageCacheLookupByID(packageID)
		if packageEntry == nil {
			con.Log.Errorf("The package cache is out of date. Please run armory refresh and try again.\n")
			return
		}
		if packageEntry.Pkg.IsAlias {
			err := installAliasPackage(packageEntry, false, clientConfig, con)
			if err != nil {
				con.Log.Errorf("Failed to install alias '%s': %s\n", packageEntry.Alias.CommandName, err)
				return
			}
		} else {
			err := installExtensionPackage(packageEntry, false, clientConfig, con)
			if err != nil {
				con.Log.Errorf("Failed to install extension '%s': %s\n", packageEntry.Extension.Name, err)
				return
			}
		}
	}
}

func installPackageByName(name, armoryPK string, forceInstallation, promptToOverwrite bool,
	clientConfig ArmoryHTTPConfig, con *repl.Console) error {
	pendingPackages := make(map[string]string)
	packageInstallList, err := buildInstallList(name, armoryPK, forceInstallation, pendingPackages)
	if err != nil {
		return nil
	}
	if len(packageInstallList) != 0 {
		for _, packageID := range packageInstallList {
			entry := packageCacheLookupByID(packageID)
			if entry == nil {
				return errors.New("cache consistency error - please refresh the cache and try again")
			}
			if entry.Pkg.IsAlias {
				err := installAliasPackage(entry, promptToOverwrite, clientConfig, con)
				if err != nil {
					return fmt.Errorf("failed to install alias '%s': %s", entry.Alias.CommandName, err)
				}
			} else {
				err := installExtensionPackage(entry, promptToOverwrite, clientConfig, con)
				if err != nil {
					return fmt.Errorf("failed to install extension '%s': %s", entry.Extension.Name, err)
				}
			}
		}
	} else {
		return ErrPackageNotFound
	}
	if name == "all" {
		con.Log.Infof("\nOperation complete\n")
	}
	return nil
}

func getInstalledPackageNames() []string {
	packageNames := []string{}

	installedAliases := assets.GetInstalledAliasManifests()
	installedExtensions := assets.GetInstalledExtensionManifests()

	for _, aliasFileName := range installedAliases {
		alias := &alias.AliasManifest{}
		manifestData, err := os.ReadFile(aliasFileName)
		if err != nil {
			continue
		}
		err = json.Unmarshal(manifestData, alias)
		if err != nil {
			continue
		}
		if !slices.Contains(packageNames, alias.CommandName) {
			packageNames = append(packageNames, alias.CommandName)
		}
	}

	for _, extensionFileName := range installedExtensions {
		ext := &extension.ExtensionManifest{}
		manifestData, err := os.ReadFile(extensionFileName)
		if err != nil {
			continue
		}
		err = json.Unmarshal(manifestData, ext)
		if err != nil {
			continue
		}
		if len(ext.ExtCommand) == 0 {
			extensionOld := &extension.ExtensionManifest_{}
			// Some extension manifests are using an older version
			// To maintain compatibility with those extensions, we will
			// re-unmarshal the data as the older version
			err = json.Unmarshal(manifestData, extensionOld)
			if err != nil {
				continue
			}
			if !slices.Contains(packageNames, extensionOld.CommandName) {
				packageNames = append(packageNames, extensionOld.CommandName)
			}
		} else {
			for _, command := range ext.ExtCommand {
				if !slices.Contains(packageNames, command.CommandName) {
					packageNames = append(packageNames, command.CommandName)
				}
			}
		}
	}

	return packageNames
}

// This is a convenience function to get the names of the commands in the cache
func getCommandsInCache() []string {
	commandNames := []string{}

	pkgCache.Range(func(key, value interface{}) bool {
		cacheEntry := value.(pkgCacheEntry)
		if cacheEntry.LastErr == nil {
			if cacheEntry.Pkg.IsAlias {
				if !slices.Contains(commandNames, cacheEntry.Alias.CommandName) {
					commandNames = append(commandNames, cacheEntry.Alias.CommandName)
				}
			} else {
				for _, command := range cacheEntry.Extension.ExtCommand {
					if !slices.Contains(commandNames, command.CommandName) {
						commandNames = append(commandNames, command.CommandName)
					}
				}
			}
		}
		return true
	})

	return commandNames
}

func getPackagesWithCommandName(name, armoryPK, minimumVersion string) []*pkgCacheEntry {
	packages := []*pkgCacheEntry{}

	pkgCache.Range(func(key, value interface{}) bool {
		cacheEntry := value.(pkgCacheEntry)
		if cacheEntry.LastErr == nil {
			if cacheEntry.Pkg.IsAlias {
				if cacheEntry.Alias.CommandName == name {
					if minimumVersion == "" || (minimumVersion != "" && cacheEntry.Alias.Version >= minimumVersion) {
						if armoryPK == "" || (armoryPK != "" && cacheEntry.ArmoryConfig.PublicKey == armoryPK) {
							packages = append(packages, &cacheEntry)
						}
					}
				}
			} else {
				for _, command := range cacheEntry.Extension.ExtCommand {
					if command.CommandName == name {
						if minimumVersion == "" || (minimumVersion != "" && cacheEntry.Extension.Version >= minimumVersion) {
							if armoryPK == "" || (armoryPK != "" && cacheEntry.ArmoryConfig.PublicKey == armoryPK) {
								packages = append(packages, &cacheEntry)
							}
							break
						}
					}
				}
			}
		}
		return true
	})

	return packages
}

func getPackageIDFromUser(name string, options map[string]string) string {
	optionKeys := repl.Keys(options)
	slices.Sort(optionKeys)
	// Add a cancel option
	optionKeys = append(optionKeys, doNotInstallOption)
	options[doNotInstallOption] = doNotInstallPackageName
	selectModel := tui.NewSelect(optionKeys)
	selectModel.Title = fmt.Sprintf("More than one package contains the command %s. Please choose an option from the list below:", name)
	newSelect := tui.NewModel(selectModel, nil, false, false)
	err := newSelect.Run()
	if err != nil {
		core.Log.Errorf("Failed to run select model: %s\n", err)
		return ""
	}
	if selectModel.SelectedItem >= 0 && selectModel.SelectedItem < len(selectModel.Choices) {
		selectedPackageKey := selectModel.Choices[selectModel.SelectedItem]
		selectedPackageID := options[selectedPackageKey]
		return selectedPackageID
	}
	return ""
}

func getPackageForCommand(name, armoryPK, minimumVersion string) (*pkgCacheEntry, error) {
	packagesWithCommand := getPackagesWithCommandName(name, armoryPK, minimumVersion)

	if len(packagesWithCommand) > 1 {
		// Build an option map for the user to choose from (option -> pkgID)
		optionMap := make(map[string]string)
		for _, packageEntry := range packagesWithCommand {
			var optionName string
			if packageEntry.Pkg.IsAlias {
				optionName = fmt.Sprintf("Alias %s %s from armory %s (%s)",
					name,
					packageEntry.Alias.Version,
					packageEntry.ArmoryConfig.Name,
					packageEntry.Pkg.RepoURL,
				)
			} else {
				optionName = fmt.Sprintf("Extension %s %s from armory %s with command %s (%s)",
					packageEntry.Pkg.Name,
					packageEntry.Extension.Version,
					packageEntry.ArmoryConfig.Name,
					name,
					packageEntry.Pkg.RepoURL,
				)
			}
			optionMap[optionName] = packageEntry.ID
		}
		selectedPackageID := getPackageIDFromUser(name, optionMap)
		if selectedPackageID == doNotInstallPackageName {
			return nil, fmt.Errorf("user cancelled installation")
		}
		for _, packageEntry := range packagesWithCommand {
			if packageEntry.ID == selectedPackageID {
				return packageEntry, nil
			}
		}
	} else if len(packagesWithCommand) == 1 {
		return packagesWithCommand[0], nil
	}
	return nil, ErrPackageNotFound
}

func buildInstallList(name, armoryPK string, forceInstallation bool, pendingPackages map[string]string) ([]string, error) {
	packageInstallList := []string{}
	installedPackages := getInstalledPackageNames()

	/*
		Gather information about what we are working with

		Find all conflicts within aliases for a given name (or all names), same thing with extensions
		Then if there are aliases and extensions with a given name, make sure to note that for when we ask the user what to do
	*/
	var requestedPackageList []string
	if name == "all" {
		requestedPackageList = []string{}
		allCommands := getCommandsInCache()
		for _, cmdName := range allCommands {
			if !slices.Contains(installedPackages, cmdName) || forceInstallation {
				// Check to see if there is a package pending with that name
				if _, ok := pendingPackages[cmdName]; !ok {
					requestedPackageList = append(requestedPackageList, cmdName)
				}
			}
		}
	} else {
		if !slices.Contains(installedPackages, name) || forceInstallation {
			// Check to see if there is a package pending with that name
			if _, ok := pendingPackages[name]; !ok {
				requestedPackageList = []string{name}
			}
		} else {
			return nil, ErrPackageAlreadyInstalled
		}
	}

	for _, packageName := range requestedPackageList {
		if _, ok := pendingPackages[packageName]; ok {
			// We are already going to install a package with this name, so do not try to resolve it
			continue
		}
		packageEntry, err := getPackageForCommand(packageName, armoryPK, "")
		if err != nil {
			return nil, err
		}
		if !slices.Contains(packageInstallList, packageEntry.ID) {
			packageInstallList = append(packageInstallList, packageEntry.ID)
			pendingPackages[packageName] = packageEntry.ID
		}

		if !packageEntry.Pkg.IsAlias {
			dependencies := make(map[string]*pkgCacheEntry)
			//err = resolveExtensionPackageDependencies(packageEntry, dependencies, pendingPackages)
			//if err != nil {
			//	return nil, err
			//}
			for pkgName, packageEntry := range dependencies {
				if !slices.Contains(packageInstallList, packageEntry.ID) {
					packageInstallList = append(packageInstallList, packageEntry.ID)
				}
				if _, ok := pendingPackages[pkgName]; !ok {
					pendingPackages[pkgName] = packageEntry.ID
				}
			}
		}
	}

	return packageInstallList, nil
}

func installAliasPackage(entry *pkgCacheEntry, promptToOverwrite bool, clientConfig ArmoryHTTPConfig,
	con *repl.Console) error {
	if entry == nil {
		return errors.New("package not found")
	}
	if !entry.Pkg.IsAlias {
		return errors.New("package is not an alias")
	}
	repoURL, err := url.Parse(entry.RepoURL)
	if err != nil {
		return err
	}

	con.Log.Infof("Downloading alias ...")

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
	tui.Clear()

	installPath := alias.InstallFromFile(tmpFile.Name(), entry.Alias.CommandName, promptToOverwrite, con)
	if installPath == nil {
		return errors.New("failed to install alias")
	}
	manifest, err := alias.LoadAlias(filepath.Join(*installPath, alias.ManifestFileName), con)
	if err != nil {
		return err
	}
	err = alias.RegisterAlias(manifest, con.ImplantMenu(), con)
	if err != nil {
		return err
	}
	return nil
}

const maxDepDepth = 10 // Arbitrary recursive limit for dependencies

func resolveExtensionPackageDependencies(pkg *pkgCacheEntry, deps map[string]*pkgCacheEntry, pendingPackages map[string]string) error {
	for _, multiExt := range pkg.Extension.ExtCommand {
		if multiExt.DependsOn == "" {
			continue // Avoid adding empty dependency
		}
		if multiExt.DependsOn == pkg.Extension.Name {
			continue // Avoid infinite loop of something that depends on itself
		}
		// We also need to look out for circular dependencies, so if we've already
		// seen this dependency, we stop resolving
		if _, ok := deps[multiExt.DependsOn]; ok {
			continue // Already resolved
		}
		// Check to make sure we are not already going to install a package with this name
		if _, ok := pendingPackages[multiExt.DependsOn]; ok {
			continue
		}
		if maxDepDepth < len(deps) {
			continue
		}
		// Figure out what package we need for the dependency
		dependencyEntry, err := getPackageForCommand(multiExt.DependsOn, "", "")
		if err != nil {
			return fmt.Errorf("could not resolve dependency %s for %s: %s", multiExt.DependsOn, pkg.Extension.Name, err)
		}
		deps[multiExt.DependsOn] = dependencyEntry
		err = resolveExtensionPackageDependencies(dependencyEntry, deps, pendingPackages)
		if err != nil {
			return err
		}
	}
	return nil
}

func installExtensionPackage(entry *pkgCacheEntry, promptToOverwrite bool, clientConfig ArmoryHTTPConfig, con *repl.Console) error {
	if entry == nil {
		return errors.New("package not found")
	}
	repoURL, err := url.Parse(entry.RepoURL)
	if err != nil {
		return err
	}

	con.Log.Infof("Downloading extension ...")

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

	tui.Clear()

	extension.InstallFromDir(tmpFile.Name(), promptToOverwrite, con, true)

	return nil
}
