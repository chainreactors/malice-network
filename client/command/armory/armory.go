package armory

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/alias"
	"github.com/chainreactors/malice-network/client/command/extension"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/tui"
	"github.com/chainreactors/malice-network/helper/cryptography/minisign"
	"github.com/pterm/pterm"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/table"
)

// ArmoryIndex - Index JSON containing alias/extension/bundle information
type ArmoryIndex struct {
	ArmoryConfig *assets.ArmoryConfig `json:"-"`
	Aliases      []*ArmoryPackage     `json:"aliases"`
	Extensions   []*ArmoryPackage     `json:"extensions"`
	Bundles      []*ArmoryBundle      `json:"bundles"`
}

// ArmoryPackage - JSON metadata for alias or extension
type ArmoryPackage struct {
	Name        string `json:"name"`
	CommandName string `json:"command_name"`
	RepoURL     string `json:"repo_url"`
	PublicKey   string `json:"public_key"`

	IsAlias bool `json:"-"`
}

// ArmoryBundle - A list of packages
type ArmoryBundle struct {
	Name     string   `json:"name"`
	Packages []string `json:"packages"`
}

// ArmoryHTTPConfig - Configuration for armory HTTP client
type ArmoryHTTPConfig struct {
	ArmoryConfig         *assets.ArmoryConfig
	IgnoreCache          bool
	ProxyURL             *url.URL
	Timeout              time.Duration
	DisableTLSValidation bool
}

type indexCacheEntry struct {
	ArmoryConfig *assets.ArmoryConfig
	RepoURL      string
	Fetched      time.Time
	Index        ArmoryIndex
	LastErr      error
}

type pkgCacheEntry struct {
	ArmoryConfig *assets.ArmoryConfig
	RepoURL      string
	Fetched      time.Time
	Pkg          ArmoryPackage
	Sig          minisign.Signature
	Alias        *alias.AliasManifest
	Extension    *extension.ExtensionManifest
	LastErr      error
}

var (
	// public key -> armoryCacheEntry
	indexCache = sync.Map{}
	// public key -> armoryPkgCacheEntry
	pkgCache = sync.Map{}

	// cacheTime - How long to cache the index/pkg manifests
	cacheTime = time.Hour

	// This will kill a download if exceeded so needs to be large
	defaultTimeout = 15 * time.Minute
)

// ArmoryCmd - The main armory command
func ArmoryCmd(ctx *grumble.Context, con *console.Console) {
	armoriesConfig := assets.GetArmoriesConfig()
	console.Log.Importantf("Fetching %d armory index(es) ... ", len(armoriesConfig))
	clientConfig := parseArmoryHTTPConfig(ctx)
	indexes := fetchIndexes(armoriesConfig, clientConfig)
	if len(indexes) != len(armoriesConfig) {
		console.Log.Infof("errors!\n")
		indexCache.Range(func(key, value interface{}) bool {
			cacheEntry := value.(indexCacheEntry)
			if cacheEntry.LastErr != nil {
				console.Log.Errorf("%s - %s\n", cacheEntry.RepoURL, cacheEntry.LastErr)
			}
			return true
		})
	} else {
		console.Log.Infof("done!\n")
	}

	if 0 < len(indexes) {
		console.Log.Infof("Fetching package information ... ")
		fetchPackageSignatures(indexes, clientConfig)
		errorCount := 0
		aliases := []*alias.AliasManifest{}
		//exts := []*extension.ExtensionManifest{}
		pkgCache.Range(func(key, value interface{}) bool {
			cacheEntry := value.(pkgCacheEntry)
			if cacheEntry.LastErr != nil {
				errorCount++
				if errorCount == 0 {
					console.Log.Infof("errors!\n")
				}
				console.Log.Errorf("%s - %s\n", cacheEntry.RepoURL, cacheEntry.LastErr)
			} else {
				if cacheEntry.Pkg.IsAlias {
					aliases = append(aliases, cacheEntry.Alias)
				} else {
					//exts = append(exts, cacheEntry.Extension)
				}
			}
			return true
		})
		if errorCount == 0 {
			console.Log.Infof("done!\n")
		}
		//if 0 < len(aliases) || 0 < len(exts) {
		if 0 < len(aliases) {
			PrintArmoryPackages(aliases, nil, con)
		} else {
			console.Log.Infof("No packages found")
		}

		bundles := bundlesInCache()
		if 0 < len(bundles) {
			PrintArmoryBundles(bundles, con)
		} else {
			console.Log.Infof("No bundles found\n")
		}
	} else {
		console.Log.Infof("No indexes found\n")
	}
}

func refresh(clientConfig ArmoryHTTPConfig) {
	armoriesConfig := assets.GetArmoriesConfig()
	indexes := fetchIndexes(armoriesConfig, clientConfig)
	fetchPackageSignatures(indexes, clientConfig)
}

func packagesInCache() ([]*alias.AliasManifest, []*extension.ExtensionManifest) {
	aliases := []*alias.AliasManifest{}
	exts := []*extension.ExtensionManifest{}
	pkgCache.Range(func(key, value interface{}) bool {
		cacheEntry := value.(pkgCacheEntry)
		if cacheEntry.LastErr == nil {
			if cacheEntry.Pkg.IsAlias {
				aliases = append(aliases, cacheEntry.Alias)
			} else {
				exts = append(exts, cacheEntry.Extension)
			}
		}
		return true
	})
	return aliases, exts
}

func bundlesInCache() []*ArmoryBundle {
	bundles := []*ArmoryBundle{}
	indexCache.Range(func(key, value interface{}) bool {
		indexBundles := value.(indexCacheEntry).Index.Bundles
		bundles = append(bundles, indexBundles...)
		return true
	})
	return bundles
}

// AliasExtensionOrBundleCompleter - Completer for alias, extension, and bundle names
func AliasExtensionOrBundleCompleter(prefix string, args []string, con *console.Console) []string {
	results := []string{}
	aliases, exts := packagesInCache()
	bundles := bundlesInCache()
	for _, aliasPkg := range aliases {
		if strings.HasPrefix(aliasPkg.CommandName, prefix) {
			results = append(results, aliasPkg.CommandName)
		}
	}
	for _, extensionPkg := range exts {
		if strings.HasPrefix(extensionPkg.CommandName, prefix) {
			results = append(results, extensionPkg.CommandName)
		}
	}
	for _, bundle := range bundles {
		if strings.HasPrefix(bundle.Name, prefix) {
			results = append(results, bundle.Name)
		}
	}
	return results
}

// PrintArmoryPackages - Prints the armory packages
func PrintArmoryPackages(aliases []*alias.AliasManifest, exts []*extension.ExtensionManifest, con *console.Console) {
	var rowEntries []table.Row
	var row table.Row

	tableModel := tui.NewTable([]table.Column{
		{Title: "Command Name", Width: 10},
		{Title: "Version", Width: 10},
		{Title: "Type", Width: 4},
		{Title: "Help", Width: 10},
		{Title: "URL", Width: 15},
	})

	type pkgInfo struct {
		CommandName string
		Version     string
		Type        string
		Help        string
		URL         string
	}
	entries := []pkgInfo{}
	for _, aliasPkg := range aliases {
		entries = append(entries, pkgInfo{
			CommandName: aliasPkg.CommandName,
			Version:     aliasPkg.Version,
			Type:        "Alias",
			Help:        aliasPkg.Help,
			URL:         aliasPkg.RepoURL,
		})
	}
	for _, extension := range exts {
		entries = append(entries, pkgInfo{
			CommandName: extension.CommandName,
			Version:     extension.Version,
			Type:        "Extension",
			Help:        extension.Help,
			URL:         extension.RepoURL,
		})
	}

	for _, pkg := range entries {
		var commandName string
		if extension.CmdExists(pkg.CommandName, con.App) {
			commandName = pterm.FgGreen.Sprint(pkg.CommandName)
		} else {
			commandName = pkg.CommandName
		}
		row = table.Row{
			commandName,
			pkg.Version,
			pkg.Type,
			pkg.Help,
			pkg.URL,
		}

		rowEntries = append(rowEntries, row)
	}
	tableModel.Rows = rowEntries
	tableModel.SetRows()
	err := tui.Run(tableModel)
	if err != nil {
		return
	}
}

// PrintArmoryBundles - Prints the armory bundles
func PrintArmoryBundles(bundles []*ArmoryBundle, con *console.Console) {
	var rowEntries []table.Row
	var row table.Row

	tableModel := tui.NewTable([]table.Column{
		{Title: "Name", Width: 10},
		{Title: "Contains", Width: 20},
		{Title: "Type", Width: 4},
		{Title: "Help", Width: 10},
		{Title: "URL", Width: 15},
	})
	for _, bundle := range bundles {
		if len(bundle.Packages) < 1 {
			continue
		}
		packages := bundle.Packages[0]
		if 1 < len(packages) {
			packages += ", "
		}
		for index, pkgName := range bundle.Packages[1:] {
			if index%5 == 4 {
				packages += pkgName + "\n"
			} else {
				packages += pkgName
				if index != len(bundle.Packages)-2 {
					packages += ", "
				}
			}
		}
		row = table.Row{
			bundle.Name,
			packages,
		}
		rowEntries = append(rowEntries, row)
	}
	tableModel.Rows = rowEntries
	tableModel.SetRows()
	err := tui.Run(tableModel)
	if err != nil {
		return
	}
}

func parseArmoryHTTPConfig(ctx *grumble.Context) ArmoryHTTPConfig {
	var proxyURL *url.URL
	rawProxyURL := ctx.Flags.String("proxy")
	if rawProxyURL != "" {
		proxyURL, _ = url.Parse(rawProxyURL)
	}

	timeout := defaultTimeout
	rawTimeout := ctx.Flags.String("timeout")
	if rawTimeout != "" {
		var err error
		timeout, err = time.ParseDuration(rawTimeout)
		if err != nil {
			timeout = defaultTimeout
		}
	}

	return ArmoryHTTPConfig{
		IgnoreCache:          ctx.Flags.Bool("ignore-cache"),
		ProxyURL:             proxyURL,
		Timeout:              timeout,
		DisableTLSValidation: ctx.Flags.Bool("insecure"),
	}
}

// fetch armory indexes, only returns indexes that were fetched successfully
// errors are still in the cache objects however and can be checked
func fetchIndexes(armoryConfigs []*assets.ArmoryConfig, clientConfig ArmoryHTTPConfig) []ArmoryIndex {
	wg := &sync.WaitGroup{}
	for _, armoryConfig := range armoryConfigs {
		wg.Add(1)
		go fetchIndex(armoryConfig, clientConfig, wg)
	}
	wg.Wait()
	indexes := []ArmoryIndex{}
	indexCache.Range(func(key, value interface{}) bool {
		cacheEntry := value.(indexCacheEntry)
		if cacheEntry.LastErr == nil {
			indexes = append(indexes, cacheEntry.Index)
		}
		return true
	})
	return indexes
}

func fetchIndex(armoryConfig *assets.ArmoryConfig, clientConfig ArmoryHTTPConfig, wg *sync.WaitGroup) {
	defer wg.Done()
	cacheEntry, ok := indexCache.Load(armoryConfig.PublicKey)
	if ok {
		cached := cacheEntry.(indexCacheEntry)
		if time.Since(cached.Fetched) < cacheTime && cached.LastErr == nil && !clientConfig.IgnoreCache {
			return
		}
	}

	armoryResult := &indexCacheEntry{
		ArmoryConfig: armoryConfig,
		RepoURL:      armoryConfig.RepoURL,
	}
	defer func() {
		armoryResult.Fetched = time.Now()
		indexCache.Store(armoryConfig.PublicKey, *armoryResult)
	}()

	repoURL, err := url.Parse(armoryConfig.RepoURL)
	if err != nil {
		armoryResult.LastErr = err
		return
	}
	if repoURL.Scheme != "https" && repoURL.Scheme != "http" {
		armoryResult.LastErr = errors.New("invalid repo url scheme in index")
		return
	}

	var index *ArmoryIndex
	if indexParser, ok := indexParsers[repoURL.Hostname()]; ok {
		index, err = indexParser(armoryConfig, clientConfig)
	} else {
		index, err = DefaultArmoryIndexParser(armoryConfig, clientConfig)
	}
	if index != nil {
		armoryResult.Index = *index
	}
	if err != nil {
		armoryResult.LastErr = fmt.Errorf("failed to parse armory index: %s", err)
	}
}

func fetchPackageSignatures(indexes []ArmoryIndex, clientConfig ArmoryHTTPConfig) {
	wg := &sync.WaitGroup{}
	for _, index := range indexes {
		for _, armoryPkg := range index.Extensions {
			wg.Add(1)
			armoryPkg.IsAlias = false
			go fetchPackageSignature(wg, index.ArmoryConfig, armoryPkg, clientConfig)
		}
		for _, armoryPkg := range index.Aliases {
			wg.Add(1)
			armoryPkg.IsAlias = true
			go fetchPackageSignature(wg, index.ArmoryConfig, armoryPkg, clientConfig)
		}
	}
	wg.Wait()
}

func fetchPackageSignature(wg *sync.WaitGroup, armoryConfig *assets.ArmoryConfig, armoryPkg *ArmoryPackage, clientConfig ArmoryHTTPConfig) {
	defer wg.Done()
	cacheEntry, ok := pkgCache.Load(armoryPkg.CommandName)
	if ok {
		cached := cacheEntry.(pkgCacheEntry)
		if time.Since(cached.Fetched) < cacheTime && cached.LastErr == nil && !clientConfig.IgnoreCache {
			return
		}
	}

	pkgCacheEntry := &pkgCacheEntry{
		ArmoryConfig: armoryConfig,
		RepoURL:      armoryPkg.RepoURL,
	}
	defer func() {
		pkgCacheEntry.Fetched = time.Now()
		pkgCache.Store(armoryPkg.CommandName, *pkgCacheEntry)
	}()

	repoURL, err := url.Parse(armoryPkg.RepoURL)
	if err != nil {
		pkgCacheEntry.LastErr = fmt.Errorf("failed to parse repo url: %s", err)
		return
	}
	if repoURL.Scheme != "https" && repoURL.Scheme != "http" {
		pkgCacheEntry.LastErr = errors.New("invalid repo url scheme in pkg")
		return
	}

	var sig *minisign.Signature
	if pkgParser, ok := pkgParsers[repoURL.Hostname()]; ok {
		sig, _, err = pkgParser(armoryConfig, armoryPkg, true, clientConfig)
	} else {
		sig, _, err = DefaultArmoryPkgParser(armoryConfig, armoryPkg, true, clientConfig)
	}
	if err != nil {
		pkgCacheEntry.LastErr = fmt.Errorf("failed to parse pkg manifest: %s", err)
		return
	}
	if sig != nil {
		pkgCacheEntry.Sig = *sig
	} else {
		pkgCacheEntry.LastErr = errors.New("nil signature")
		return
	}
	if armoryPkg != nil {
		pkgCacheEntry.Pkg = *armoryPkg
	}
	if err == nil {
		manifestData, err := base64.StdEncoding.DecodeString(sig.TrustedComment)
		if err != nil {
			pkgCacheEntry.LastErr = fmt.Errorf("failed to b64 decode trusted comment: %s", err)
			return
		}
		if armoryPkg.IsAlias {
			pkgCacheEntry.Alias, err = alias.ParseAliasManifest(manifestData)
		} else {
			pkgCacheEntry.Extension, err = extension.ParseExtensionManifest(manifestData)
		}
		if err != nil {
			pkgCacheEntry.LastErr = fmt.Errorf("failed to parse trusted manifest in pkg signature: %s", err)
		}
	}
}
