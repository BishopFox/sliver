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
	"net/url"
	"slices"
	"sync"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/alias"
	"github.com/bishopfox/sliver/client/command/extensions"
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/util/minisign"
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

	IsAlias    bool   `json:"-"`
	ArmoryName string `json:"-"`
	/*
		With support for multiple armories, the command name of a package
		is not unique anymore, so we need something that is unique
		to be able to keep track of packages.

		This ID will be a hash calculated from properties of the package.
	*/
	ID       string `json:"-"`
	ArmoryPK string `json:"-"`
}

// ArmoryBundle - A list of packages
type ArmoryBundle struct {
	Name       string   `json:"name"`
	Packages   []string `json:"packages"`
	ArmoryName string   `json:"-"`
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
	Extension    *extensions.ExtensionManifest
	LastErr      error
	// This corresponds to Pkg.ID
	ID string
}

var (
	// public key -> armoryCacheEntry
	indexCache = sync.Map{}
	// package ID -> armoryPkgCacheEntry
	pkgCache = sync.Map{}
	// public key -> assets.ArmoryConfig
	currentArmories = sync.Map{}

	// cacheTime - How long to cache the index/pkg manifests
	//cacheTime = time.Hour
	cacheTime = 2 * time.Minute

	// This will kill a download if exceeded so needs to be large
	defaultTimeout = 15 * time.Minute

	// Track whether armories have been initialized so that we know if we need to pull from the config
	armoriesInitialized = false

	// Track whether the default armory has been removed by the user (this is needed to prevent it from being readded in if they have removed it)
	defaultArmoryRemoved = false
)

// ArmoryCmd - The main armory command
func ArmoryCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	armoriesConfig := getCurrentArmoryConfiguration()
	if len(armoriesConfig) == 1 {
		con.Printf("Reading armory index ... ")
	} else {
		con.PrintInfof("Reading %d armory indexes ... ", len(armoriesConfig))
	}
	clientConfig := parseArmoryHTTPConfig(cmd)
	indexes := fetchIndexes(clientConfig)
	if len(indexes) != len(armoriesConfig) {
		indexCache.Range(func(key, value interface{}) bool {
			cacheEntry := value.(indexCacheEntry)
			if cacheEntry.LastErr != nil {
				con.PrintErrorf("%s (%s) - %s\n", cacheEntry.ArmoryConfig.Name, cacheEntry.RepoURL, cacheEntry.LastErr)
			}
			return true
		})
	} else {
		con.Printf("done!\n")
	}
	armoriesInitialized = true
	if len(indexes) == 0 {
		con.PrintInfof("No indexes found\n")
		return
	}
	aliases := []*alias.AliasManifest{}
	exts := []*extensions.ExtensionManifest{}

	for _, index := range indexes {
		errorCount := 0
		con.PrintInfof("Reading package information for armory %s ... ", index.ArmoryConfig.Name)
		fetchPackageSignatures(index, clientConfig)
		pkgCache.Range(func(key, value interface{}) bool {
			cacheEntry, ok := value.(pkgCacheEntry)
			if !ok {
				// Something is wrong with this entry
				pkgCache.Delete(value)
				return true
			}
			if cacheEntry.ArmoryConfig.PublicKey != index.ArmoryConfig.PublicKey {
				return true
			}
			if cacheEntry.LastErr != nil {
				errorCount++
				if errorCount == 0 {
					con.Printf("errors!\n")
				}
				con.PrintErrorf("%s - %s\n", cacheEntry.RepoURL, cacheEntry.LastErr)
			} else {
				if cacheEntry.Pkg.IsAlias {
					aliases = append(aliases, cacheEntry.Alias)
				} else {
					exts = append(exts, cacheEntry.Extension) //todo: check this isn't a bug
				}
			}
			return true
		})
		if errorCount == 0 {
			con.Printf("done!\n")
		}
	}
	if 0 < len(aliases) || 0 < len(exts) {
		con.Println()
		PrintArmoryPackages(aliases, exts, con)
	} else {
		con.PrintInfof("No packages found\n")
	}
	con.Println()
	bundles := bundlesInCache()
	if 0 < len(bundles) {
		PrintArmoryBundles(bundles, con)
	} else {
		con.PrintInfof("No bundles found\n")
	}
}

func refresh(clientConfig ArmoryHTTPConfig) {
	getCurrentArmoryConfiguration()
	indexes := fetchIndexes(clientConfig)
	for _, index := range indexes {
		fetchPackageSignatures(index, clientConfig)
	}
}

func countUniqueCommandsFromManifests(aliases []*alias.AliasManifest, exts []*extensions.ExtensionManifest) (int, int) {
	uniqueAliasNames := []string{}
	uniqueExtensionNames := []string{}

	for _, alias := range aliases {
		if !slices.Contains(uniqueAliasNames, alias.CommandName) {
			uniqueAliasNames = append(uniqueAliasNames, alias.CommandName)
		}
	}

	for _, ext := range exts {
		for _, cmd := range ext.ExtCommand {
			if !slices.Contains(uniqueExtensionNames, cmd.CommandName) {
				uniqueExtensionNames = append(uniqueExtensionNames, cmd.CommandName)
			}
		}
	}

	return len(uniqueAliasNames), len(uniqueExtensionNames)
}

func packageManifestsInCache() ([]*alias.AliasManifest, []*extensions.ExtensionManifest) {
	aliases := []*alias.AliasManifest{}
	exts := []*extensions.ExtensionManifest{}
	pkgCache.Range(func(key, value interface{}) bool {
		cacheEntry := value.(pkgCacheEntry)
		if cacheEntry.LastErr == nil {
			if cacheEntry.Pkg.IsAlias {
				aliases = append(aliases, cacheEntry.Alias)
			} else {
				exts = append(exts, cacheEntry.Extension) //todo: check this isn't a bug
			}
		}
		return true
	})
	return aliases, exts
}

func armoryLookupByName(name string) *assets.ArmoryConfig {
	var result *assets.ArmoryConfig

	indexCache.Range(func(key, value interface{}) bool {
		indexEntry, ok := value.(indexCacheEntry)
		if !ok {
			// Keep going
			return true
		}
		if indexEntry.ArmoryConfig.Name == name {
			result = indexEntry.ArmoryConfig
			return false
		}
		return true
	})

	return result
}

// Returns the packages in the cache with a given name
func packageCacheLookupByName(name string) []*pkgCacheEntry {
	var result []*pkgCacheEntry = make([]*pkgCacheEntry, 0)

	pkgCache.Range(func(key, value interface{}) bool {
		cacheEntry, ok := value.(pkgCacheEntry)
		if !ok {
			// Keep going
			return true
		}
		if cacheEntry.Pkg.Name == name {
			result = append(result, &cacheEntry)
		}
		return true
	})

	return result
}

// Returns the packages in the cache for a given command name
func packageCacheLookupByCmd(commandName string) []*pkgCacheEntry {
	var result []*pkgCacheEntry = make([]*pkgCacheEntry, 0)

	pkgCache.Range(func(key, value interface{}) bool {
		cacheEntry, ok := value.(pkgCacheEntry)
		if !ok {
			// Keep going
			return true
		}
		if cacheEntry.Pkg.CommandName == commandName {
			result = append(result, &cacheEntry)
		}
		return true
	})

	return result
}

// Returns the package in the cache for a given command name and armory
func packageCacheLookupByCmdAndArmory(commandName string, armoryPublicKey string) *pkgCacheEntry {
	var result *pkgCacheEntry

	pkgCache.Range(func(key, value interface{}) bool {
		cacheEntry, ok := value.(pkgCacheEntry)
		if !ok {
			// Keep going
			return true
		}
		if cacheEntry.ArmoryConfig.PublicKey == armoryPublicKey && cacheEntry.Pkg.CommandName == commandName {
			result = &cacheEntry
			// Stop iterating
			return false
		}
		return true
	})

	return result
}

// Returns the package hashes in the cache for a given armory
func packageHashLookupByArmory(armoryPublicKey string) []string {
	result := []string{}

	pkgCache.Range(func(key, value interface{}) bool {
		cacheEntry, ok := value.(pkgCacheEntry)
		if !ok {
			// Keep going
			return true
		}
		if cacheEntry.ArmoryConfig.PublicKey == armoryPublicKey {
			result = append(result, cacheEntry.ID)
		}
		return true
	})

	return result
}

func packageCacheLookupByID(packageID string) *pkgCacheEntry {
	var packageEntry *pkgCacheEntry

	pkgCache.Range(func(key, value interface{}) bool {
		cacheEntry, ok := value.(pkgCacheEntry)
		if !ok {
			// Keep going
			return true
		}
		if cacheEntry.LastErr == nil {
			if cacheEntry.ID == packageID {
				packageEntry = &cacheEntry
				// Stop iterating
				return false
			}
		}
		return true
	})

	return packageEntry
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
func AliasExtensionOrBundleCompleter() carapace.Action {
	comps := func(ctx carapace.Context) carapace.Action {
		var action carapace.Action

		results := []string{}
		aliases, exts := packageManifestsInCache()
		bundles := bundlesInCache()

		for _, aliasPkg := range aliases {
			results = append(results, aliasPkg.CommandName)
			results = append(results, aliasPkg.Help)
		}
		aliasesComps := carapace.ActionValuesDescribed(results...).Tag("aliases").Invoke(ctx)
		results = make([]string, 0)

		for _, extension := range exts {
			for _, extensionPkg := range extension.ExtCommand {
				results = append(results, extensionPkg.CommandName)
				results = append(results, extensionPkg.Help)
			}
		}
		extentionComps := carapace.ActionValuesDescribed(results...).Tag("extensions").Invoke(ctx)
		results = make([]string, 0)

		for _, bundle := range bundles {
			results = append(results, bundle.Name)
		}
		bundleComps := carapace.ActionValues(results...).Tag("bundles").Invoke(ctx)

		return action.Invoke(ctx).Merge(
			aliasesComps,
			extentionComps,
			bundleComps,
		).ToA()
	}

	return carapace.ActionCallback(comps)
}

// PrintArmoryPackages - Prints the armory packages
func PrintArmoryPackages(aliases []*alias.AliasManifest, exts []*extensions.ExtensionManifest, con *console.SliverClient) {
	width, _, err := term.GetSize(0)
	if err != nil {
		width = 1
	}

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.SetTitle(console.Bold + "Packages" + console.Normal)

	urlMargin := 150 // Extra margin needed to show URL column

	if con.Settings.SmallTermWidth+urlMargin < width {
		tw.AppendHeader(table.Row{
			"Armory",
			"Command Name",
			"Version",
			"Type",
			"Help",
			"URL",
		})
	} else {
		tw.AppendHeader(table.Row{
			"Armory",
			"Command Name",
			"Version",
			"Type",
			"Help",
		})
	}

	// Columns start at 1 for some dumb reason
	tw.SortBy([]table.SortBy{
		{Number: 2, Mode: table.Asc},
	})

	type pkgInfo struct {
		ArmoryName  string
		CommandName string
		Version     string
		Type        string
		Help        string
		URL         string
	}
	entries := []pkgInfo{}

	for _, aliasPkg := range aliases {
		entries = append(entries, pkgInfo{
			ArmoryName:  aliasPkg.ArmoryName,
			CommandName: aliasPkg.CommandName,
			Version:     aliasPkg.Version,
			Type:        "Alias",
			Help:        aliasPkg.Help,
			URL:         aliasPkg.RepoURL,
		})
	}
	for _, extm := range exts {
		for _, extension := range extm.ExtCommand {
			entries = append(entries, pkgInfo{
				ArmoryName:  extm.ArmoryName,
				CommandName: extension.CommandName,
				Version:     extension.Manifest.Version,
				Type:        "Extension",
				Help:        extension.Help,
				URL:         extension.Manifest.RepoURL,
			})
		}
	}

	sliverMenu := con.App.Menu("implant")

	rows := []table.Row{}
	for _, pkg := range entries {
		color := console.Normal
		if extensions.CmdExists(pkg.CommandName, sliverMenu.Command) {
			color = console.Green
		}
		if con.Settings.SmallTermWidth+urlMargin < width {
			rows = append(rows, table.Row{
				fmt.Sprintf(color+"%s"+console.Normal, pkg.ArmoryName),
				fmt.Sprintf(color+"%s"+console.Normal, pkg.CommandName),
				fmt.Sprintf(color+"%s"+console.Normal, pkg.Version),
				fmt.Sprintf(color+"%s"+console.Normal, pkg.Type),
				fmt.Sprintf(color+"%s"+console.Normal, pkg.Help),
				fmt.Sprintf(color+"%s"+console.Normal, pkg.URL),
			})
		} else {
			rows = append(rows, table.Row{
				fmt.Sprintf(color+"%s"+console.Normal, pkg.ArmoryName),
				fmt.Sprintf(color+"%s"+console.Normal, pkg.CommandName),
				fmt.Sprintf(color+"%s"+console.Normal, pkg.Version),
				fmt.Sprintf(color+"%s"+console.Normal, pkg.Type),
				fmt.Sprintf(color+"%s"+console.Normal, pkg.Help),
			})
		}
	}
	tw.AppendRows(rows)
	con.Printf("%s\n", tw.Render())
}

// PrintArmoryBundles - Prints the armory bundles
func PrintArmoryBundles(bundles []*ArmoryBundle, con *console.SliverClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.SetTitle(console.Bold + "Bundles" + console.Normal)
	tw.AppendHeader(table.Row{
		"Armory Name",
		"Name",
		"Contains",
	})
	tw.SortBy([]table.SortBy{
		{Name: "Name", Mode: table.Asc},
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
		tw.AppendRow(table.Row{
			bundle.ArmoryName,
			bundle.Name,
			packages,
		})
	}
	con.Printf("%s\n", tw.Render())
}

func parseArmoryHTTPConfig(cmd *cobra.Command) ArmoryHTTPConfig {
	var proxyURL *url.URL
	rawProxyURL, _ := cmd.Flags().GetString("proxy")
	if rawProxyURL != "" {
		proxyURL, _ = url.Parse(rawProxyURL)
	}

	timeout := defaultTimeout
	rawTimeout, _ := cmd.Flags().GetString("timeout")
	if rawTimeout != "" {
		var err error
		timeout, err = time.ParseDuration(rawTimeout)
		if err != nil {
			timeout = defaultTimeout
		}
	}

	ignoreCache, _ := cmd.Flags().GetBool("ignore-cache")
	disableTLSValidation, _ := cmd.Flags().GetBool("insecure")

	return ArmoryHTTPConfig{
		IgnoreCache:          ignoreCache,
		ProxyURL:             proxyURL,
		Timeout:              timeout,
		DisableTLSValidation: disableTLSValidation,
	}
}

// fetch armory indexes, only returns indexes that were fetched successfully
// errors are still in the cache objects however and can be checked
func fetchIndexes(clientConfig ArmoryHTTPConfig) []ArmoryIndex {
	wg := &sync.WaitGroup{}
	// Try to get a max of 10 indexes at a time
	currentRequests := make(chan struct{}, 10)
	currentArmories.Range(func(key, value interface{}) bool {
		armoryEntry := value.(assets.ArmoryConfig)
		if armoryEntry.Enabled {
			wg.Add(1)
			currentRequests <- struct{}{}
			go fetchIndex(&armoryEntry, currentRequests, clientConfig, wg)
		}
		return true
	})
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

func fetchIndex(armoryConfig *assets.ArmoryConfig, requestChannel chan struct{}, clientConfig ArmoryHTTPConfig, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		<-requestChannel
	}()
	cacheEntry, ok := indexCache.Load(armoryConfig.PublicKey)
	if ok {
		cached := cacheEntry.(indexCacheEntry)
		if time.Since(cached.Fetched) < cacheTime && cached.LastErr == nil && !clientConfig.IgnoreCache {
			return
		} else if time.Since(cached.Fetched) >= cacheTime {
			// If an index has gone stale, remove it from the index cache
			indexCache.Delete(armoryConfig.PublicKey)
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

func calculateHashesForIndex(index ArmoryIndex) []string {
	result := []string{}

	for _, pkg := range index.Aliases {
		result = append(result, calculatePackageHash(pkg))
	}

	for _, pkg := range index.Extensions {
		result = append(result, calculatePackageHash(pkg))
	}

	return result
}

func makePackageCacheConsistent(index ArmoryIndex) {
	packagesToRemove := []string{}

	// Get the packages for the armory out of the cache
	cacheHashesForArmory := packageHashLookupByArmory(index.ArmoryConfig.PublicKey)
	indexHashesForArmory := calculateHashesForIndex(index)

	if len(cacheHashesForArmory) > len(indexHashesForArmory) {
		// Then there are packages in the cache that do not exist in the armory
		if len(indexHashesForArmory) == 0 {
			packagesToRemove = cacheHashesForArmory
		} else {
			for _, packageHash := range indexHashesForArmory {
				if !slices.Contains(cacheHashesForArmory, packageHash) {
					packagesToRemove = append(packagesToRemove, packageHash)
				}
			}
		}
	}
	// The remaining case of there being packages in the armory that do not exist in the cache
	// will have to be solved with fetchPackageSignatures, and that function calls this one
	// after fetching signatures and storing them in the cache, so that case should not apply

	for _, packageHash := range packagesToRemove {
		pkgCache.Delete(packageHash)
	}
}

func fetchPackageSignatures(index ArmoryIndex, clientConfig ArmoryHTTPConfig) {
	wg := &sync.WaitGroup{}
	// Be kind to armories and limit concurrent requests to 10
	// This is an arbritrary number and we may have to tweak it if it causes problems
	currentRequests := make(chan struct{}, 10)
	for _, armoryPkg := range index.Extensions {
		wg.Add(1)
		currentRequests <- struct{}{}
		armoryPkg.IsAlias = false
		go fetchPackageSignature(wg, currentRequests, index.ArmoryConfig, armoryPkg, clientConfig)
	}
	for _, armoryPkg := range index.Aliases {
		wg.Add(1)
		currentRequests <- struct{}{}
		armoryPkg.IsAlias = true
		go fetchPackageSignature(wg, currentRequests, index.ArmoryConfig, armoryPkg, clientConfig)
	}
	wg.Wait()

	// If packages were deleted from the index, make sure the cache is consistent
	makePackageCacheConsistent(index)
}

func fetchPackageSignature(wg *sync.WaitGroup, requestChannel chan struct{}, armoryConfig *assets.ArmoryConfig, armoryPkg *ArmoryPackage, clientConfig ArmoryHTTPConfig) {
	defer wg.Done()
	defer func() {
		<-requestChannel
	}()
	cacheEntry, ok := pkgCache.Load(armoryPkg.ID)
	if ok {
		cached := cacheEntry.(pkgCacheEntry)
		if time.Since(cached.Fetched) < cacheTime && cached.LastErr == nil && !clientConfig.IgnoreCache {
			return
		} else if time.Since(cached.Fetched) >= cacheTime {
			// If a package has gone stale, remove it from the package cache
			pkgCache.Delete(armoryPkg.ID)
		}
	}

	pkgCacheEntry := &pkgCacheEntry{
		ArmoryConfig: armoryConfig,
		RepoURL:      armoryPkg.RepoURL,
		ID:           armoryPkg.ID,
	}
	defer func() {
		pkgCacheEntry.Fetched = time.Now()
		pkgCache.Store(armoryPkg.ID, *pkgCacheEntry)
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

	manifestData, err := base64.StdEncoding.DecodeString(sig.TrustedComment)
	if err != nil {
		pkgCacheEntry.LastErr = fmt.Errorf("failed to b64 decode trusted comment: %s", err)
		return
	}
	if armoryPkg.IsAlias {
		pkgCacheEntry.Alias, err = alias.ParseAliasManifest(manifestData)
		pkgCacheEntry.Alias.ArmoryName = armoryConfig.Name
		pkgCacheEntry.Alias.ArmoryPK = armoryConfig.PublicKey
	} else {
		pkgCacheEntry.Extension, err = extensions.ParseExtensionManifest(manifestData)
		pkgCacheEntry.Extension.ArmoryName = armoryConfig.Name
		pkgCacheEntry.Extension.ArmoryPK = armoryConfig.PublicKey
	}
	if err != nil {
		pkgCacheEntry.LastErr = fmt.Errorf("failed to parse trusted manifest in pkg signature: %s", err)
	}

}

func clearAllCaches() {
	currentArmories.Range(func(key, value any) bool {
		currentArmories.Delete(key)
		return true
	})
	indexCache.Range(func(key, value any) bool {
		indexCache.Delete(key)
		return true
	})
	pkgCache.Range(func(key, value any) bool {
		pkgCache.Delete(key)
		return true
	})
}

func getArmoryPublicKey(armoryName string) string {
	// Find PK for the armory name
	armoryPK := ""
	currentArmories.Range(func(key, value any) bool {
		armoryEntry := value.(assets.ArmoryConfig)
		if armoryEntry.Name == armoryName {
			armoryPK = armoryEntry.PublicKey
			return false
		}
		return true
	})

	return armoryPK
}
