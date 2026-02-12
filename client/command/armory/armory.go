package armory

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox
	Copyright (C) 2021 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
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
// ArmoryIndex - Index JSON 包含 alias/extension/bundle 信息
type ArmoryIndex struct {
	ArmoryConfig *assets.ArmoryConfig `json:"-"`
	Aliases      []*ArmoryPackage     `json:"aliases"`
	Extensions   []*ArmoryPackage     `json:"extensions"`
	Bundles      []*ArmoryBundle      `json:"bundles"`
}

// ArmoryPackage - JSON metadata for alias or extension
// 别名或扩展名的 ArmoryPackage - JSON 元数据
type ArmoryPackage struct {
	Name        string `json:"name"`
	CommandName string `json:"command_name"`
	RepoURL     string `json:"repo_url"`
	PublicKey   string `json:"public_key"`

	IsAlias    bool   `json:"-"`
	ArmoryName string `json:"-"`
	/*
		With support for multiple armories, the command name of a package
		With 支持多个armouries，一个包的命令名
		is not unique anymore, so we need something that is unique
		不再是唯一的，所以我们需要一些独特的东西
		to be able to keep track of packages.
		能够跟踪 packages.

		This ID will be a hash calculated from properties of the package.
		This ID 将是根据 package. 的属性计算出的哈希值
	*/
	ID       string `json:"-"`
	ArmoryPK string `json:"-"`
}

// ArmoryBundle - A list of packages
// ArmoryBundle - A 包列表
type ArmoryBundle struct {
	Name       string   `json:"name"`
	Packages   []string `json:"packages"`
	ArmoryName string   `json:"-"`
}

// ArmoryHTTPConfig - Configuration for armory HTTP client
// ArmoryHTTPConfig - Configuration 用于 armory HTTP 客户端
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
	// This 对应于 Pkg.ID
	ID string
}

var (
	// public key -> armoryCacheEntry
	// 公钥 -> armoryCacheEntry
	indexCache = sync.Map{}
	// package ID -> armoryPkgCacheEntry
	// 包 ID -> armoryPkgCacheEntry
	pkgCache = sync.Map{}
	// public key -> assets.ArmoryConfig
	// 公钥 -> assets.ArmoryConfig
	currentArmories = sync.Map{}

	// cacheTime - How long to cache the index/pkg manifests
	// cacheTime - How long 用于缓存 index/pkg 清单
	//cacheTime = time.Hour
	cacheTime = 2 * time.Minute

	// This will kill a download if exceeded so needs to be large
	// 如果超过 This 将终止下载，因此需要很大
	defaultTimeout = 15 * time.Minute

	// Track whether armories have been initialized so that we know if we need to pull from the config
	// Track 军械库是否已初始化，以便我们知道是否需要从配置中提取
	armoriesInitialized = false

	// Track whether the default armory has been removed by the user (this is needed to prevent it from being readded in if they have removed it)
	// Track 默认的 armory 是否已被用户删除（这​​是为了防止在删除它时将其读入）
	defaultArmoryRemoved = false
)

// ArmoryCmd - The main armory command
// ArmoryCmd - The 主 armory 命令
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
				// Something 此条目有误
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
					exts = append(exts, cacheEntry.Extension) //todo：检查这不是一个错误
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
				exts = append(exts, cacheEntry.Extension) //todo：检查这不是一个错误
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
			// Keep 去
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
// Returns 缓存中具有给定名称的包
func packageCacheLookupByName(name string) []*pkgCacheEntry {
	var result []*pkgCacheEntry = make([]*pkgCacheEntry, 0)

	pkgCache.Range(func(key, value interface{}) bool {
		cacheEntry, ok := value.(pkgCacheEntry)
		if !ok {
			// Keep going
			// Keep 去
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
// Returns 缓存中给定命令名称的包
func packageCacheLookupByCmd(commandName string) []*pkgCacheEntry {
	var result []*pkgCacheEntry = make([]*pkgCacheEntry, 0)

	pkgCache.Range(func(key, value interface{}) bool {
		cacheEntry, ok := value.(pkgCacheEntry)
		if !ok {
			// Keep going
			// Keep 去
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
// Returns 缓存中给定命令名的包和 armory
func packageCacheLookupByCmdAndArmory(commandName string, armoryPublicKey string) *pkgCacheEntry {
	var result *pkgCacheEntry

	pkgCache.Range(func(key, value interface{}) bool {
		cacheEntry, ok := value.(pkgCacheEntry)
		if !ok {
			// Keep going
			// Keep 去
			return true
		}
		if cacheEntry.ArmoryConfig.PublicKey == armoryPublicKey && cacheEntry.Pkg.CommandName == commandName {
			result = &cacheEntry
			// Stop iterating
			// Stop 迭代
			return false
		}
		return true
	})

	return result
}

// Returns the package hashes in the cache for a given armory
// Returns 包在缓存中针对给定的 armory 进行哈希值
func packageHashLookupByArmory(armoryPublicKey string) []string {
	result := []string{}

	pkgCache.Range(func(key, value interface{}) bool {
		cacheEntry, ok := value.(pkgCacheEntry)
		if !ok {
			// Keep going
			// Keep 去
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
			// Keep 去
			return true
		}
		if cacheEntry.LastErr == nil {
			if cacheEntry.ID == packageID {
				packageEntry = &cacheEntry
				// Stop iterating
				// Stop 迭代
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
// AliasExtensionOrBundleCompleter - Completer 用于别名、扩展名和捆绑包名称
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
// PrintArmoryPackages - Prints armory 包
func PrintArmoryPackages(aliases []*alias.AliasManifest, exts []*extensions.ExtensionManifest, con *console.SliverClient) {
	width, _, err := term.GetSize(0)
	if err != nil {
		width = 1
	}

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.SetTitle(console.StyleBold.Render("Packages"))

	urlMargin := 150 // Extra margin needed to show URL column
	urlMargin := 150 // 显示 URL 列需要 Extra 边距

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
	// 由于某些愚蠢的原因 Columns 从 1 开始
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
		style := console.StyleNormal
		if extensions.CmdExists(pkg.CommandName, sliverMenu.Command) {
			style = console.StyleGreen
		}
		if con.Settings.SmallTermWidth+urlMargin < width {
			rows = append(rows, table.Row{
				style.Render(pkg.ArmoryName),
				style.Render(pkg.CommandName),
				style.Render(pkg.Version),
				style.Render(pkg.Type),
				style.Render(pkg.Help),
				style.Render(pkg.URL),
			})
		} else {
			rows = append(rows, table.Row{
				style.Render(pkg.ArmoryName),
				style.Render(pkg.CommandName),
				style.Render(pkg.Version),
				style.Render(pkg.Type),
				style.Render(pkg.Help),
			})
		}
	}
	tw.AppendRows(rows)
	con.Printf("%s\n", tw.Render())
}

// PrintArmoryBundles - Prints the armory bundles
// PrintArmoryBundles - Prints armory 捆绑包
func PrintArmoryBundles(bundles []*ArmoryBundle, con *console.SliverClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.SetTitle(console.StyleBold.Render("Bundles"))
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
// 获取 armory 索引，仅返回成功获取的索引
// errors are still in the cache objects however and can be checked
// 但是错误仍然存​​在于缓存对象中并且可以检查
func fetchIndexes(clientConfig ArmoryHTTPConfig) []ArmoryIndex {
	wg := &sync.WaitGroup{}
	// Try to get a max of 10 indexes at a time
	// Try 一次最多获取 10 个索引
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
			// If 索引已过时，将其从索引缓存中删除
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
	// Get 缓存中 armory 的包
	cacheHashesForArmory := packageHashLookupByArmory(index.ArmoryConfig.PublicKey)
	indexHashesForArmory := calculateHashesForIndex(index)

	if len(cacheHashesForArmory) > len(indexHashesForArmory) {
		// Then there are packages in the cache that do not exist in the armory
		// Then 缓存中存在 armory 中不存在的包
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
	// The 剩下的情况是 armory 中存在缓存中不存在的包
	// will have to be solved with fetchPackageSignatures, and that function calls this one
	// 必须用 fetchPackageSignatures 来解决，该函数调用这个函数
	// after fetching signatures and storing them in the cache, so that case should not apply
	// 在获取签名并将其存储在缓存中之后，因此这种情况不适用

	for _, packageHash := range packagesToRemove {
		pkgCache.Delete(packageHash)
	}
}

func fetchPackageSignatures(index ArmoryIndex, clientConfig ArmoryHTTPConfig) {
	wg := &sync.WaitGroup{}
	// Be kind to armories and limit concurrent requests to 10
	// Be 对军械库友善并将并发请求限制为 10
	// This is an arbritrary number and we may have to tweak it if it causes problems
	// This 是一个任意数字，如果它导致问题，我们可能需要调整它
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
	// If 包已从索引中删除，确保缓存一致
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
			// If 包已过时，将其从包缓存中删除
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
	// Find PK 表示 armory 名称
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
