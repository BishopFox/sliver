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
	"net/url"
	"sync"
	"time"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/alias"
	"github.com/bishopfox/sliver/client/command/extensions"
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/server/cryptography/minisign"
	"github.com/desertbit/grumble"
	"github.com/jedib0t/go-pretty/v6/table"
)

type ArmoryIndex struct {
	Aliases    []*ArmoryPackage `json:"aliases"`
	Extensions []*ArmoryPackage `json:"extensions"`
}

type ArmoryPackage struct {
	Name        string `json:"name"`
	CommandName string `json:"command_name"`
	RepoURL     string `json:"repo_url"`
	PublicKey   string `json:"public_key"`

	IsAlias bool `json:"-"`
}

type ArmoryHTTPConfig struct {
	IgnoreCache          bool
	ProxyURL             *url.URL
	Timeout              time.Duration
	DisableTLSValidation bool
}

type indexCacheEntry struct {
	RepoURL string
	Fetched time.Time
	Index   ArmoryIndex
	LastErr error
}

type pkgCacheEntry struct {
	RepoURL   string
	Fetched   time.Time
	Pkg       ArmoryPackage
	Sig       minisign.Signature
	Alias     *alias.AliasManifest
	Extension *extensions.ExtensionManifest
	LastErr   error
}

var (
	// public key -> armoryCacheEntry
	indexCache = sync.Map{}
	// public key -> armoryPkgCacheEntry
	pkgCache = sync.Map{}

	cacheTime      = time.Hour // This will kill a download if exceeded so needs to be large
	defaultTimeout = 15 * time.Minute
)

// ArmoryCmd - The main armory command
func ArmoryCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	armoriesConfig := assets.GetArmoriesConfig()

	con.PrintInfof("Fetching %d armory index(es) ... ", len(armoriesConfig))
	clientConfig := parseArmoryHTTPConfig(ctx)
	indexes := fetchIndexes(armoriesConfig, clientConfig)
	if len(indexes) != len(armoriesConfig) {
		con.Printf("errors!\n")
		indexCache.Range(func(key, value interface{}) bool {
			cacheEntry := value.(indexCacheEntry)
			if cacheEntry.LastErr != nil {
				con.PrintErrorf("%s: %s\n", cacheEntry.RepoURL, cacheEntry.LastErr)
			}
			return true
		})
	} else {
		con.Printf("done!\n")
	}

	if 0 < len(indexes) {
		con.PrintInfof("Fetching package information ... ")
		fetchPackageSignatures(indexes, clientConfig)
		errorCount := 0
		aliases := []*alias.AliasManifest{}
		extensions := []*extensions.ExtensionManifest{}
		pkgCache.Range(func(key, value interface{}) bool {
			cacheEntry := value.(pkgCacheEntry)
			if cacheEntry.LastErr != nil {
				errorCount++
				if errorCount == 0 {
					con.Printf("errors!\n")
				}
				con.PrintErrorf("%s: %s\n", cacheEntry.RepoURL, cacheEntry.LastErr)
			} else {
				if cacheEntry.Pkg.IsAlias {
					aliases = append(aliases, cacheEntry.Alias)
				} else {
					extensions = append(extensions, cacheEntry.Extension)
				}
			}
			return true
		})
		if errorCount == 0 {
			con.Printf("done!\n")
		}
		if 0 < len(aliases) || 0 < len(extensions) {
			con.Println()
			PrintArmoryPackages(aliases, extensions, con)
		} else {
			con.PrintInfof("No packages found\n")
		}
	} else {
		con.PrintInfof("No indexes found\n")
	}
}

// PrintArmoryPackages - Prints the armory packages
func PrintArmoryPackages(aliases []*alias.AliasManifest, extensions []*extensions.ExtensionManifest, con *console.SliverConsoleClient) {

	type pkgInfo struct {
		CommandName string
		Version     string
		Type        string
		URL         string
	}
	packages := []pkgInfo{}
	for _, alias := range aliases {
		packages = append(packages, pkgInfo{
			CommandName: alias.CommandName,
			Version:     alias.Version,
			Type:        "Alias",
			URL:         alias.RepoURL,
		})
	}
	for _, extension := range extensions {
		packages = append(packages, pkgInfo{
			CommandName: extension.CommandName,
			Version:     extension.Version,
			Type:        "Extension",
			URL:         extension.RepoURL,
		})
	}

	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))
	tw.AppendHeader(table.Row{
		"Command Name",
		"Version",
		"Type",
		"URL",
	})
	tw.SortBy([]table.SortBy{
		{Name: "Command Name", Mode: table.Asc},
	})

	for _, pkg := range packages {
		tw.AppendRow(table.Row{
			pkg.CommandName,
			pkg.Version,
			pkg.Type,
			pkg.URL,
		})
	}

	con.Printf("%s\n", tw.Render())
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

	armoryResult := &indexCacheEntry{RepoURL: armoryConfig.RepoURL}
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
		armoryResult.LastErr = errors.New("invalid URL scheme")
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
	armoryResult.LastErr = err
}

func fetchPackageSignatures(indexes []ArmoryIndex, clientConfig ArmoryHTTPConfig) {
	wg := &sync.WaitGroup{}
	for _, index := range indexes {
		for _, armoryPkg := range index.Extensions {
			wg.Add(1)
			armoryPkg.IsAlias = false
			go fetchPackageSignature(wg, armoryPkg, clientConfig)
		}
		for _, armoryPkg := range index.Aliases {
			wg.Add(1)
			armoryPkg.IsAlias = true
			go fetchPackageSignature(wg, armoryPkg, clientConfig)
		}
	}
	wg.Wait()
}

func fetchPackageSignature(wg *sync.WaitGroup, armoryPkg *ArmoryPackage, clientConfig ArmoryHTTPConfig) {
	defer wg.Done()
	cacheEntry, ok := pkgCache.Load(armoryPkg.PublicKey)
	if ok {
		cached := cacheEntry.(pkgCacheEntry)
		if time.Since(cached.Fetched) < cacheTime && cached.LastErr == nil && !clientConfig.IgnoreCache {
			return
		}
	}

	pkgCacheEntry := &pkgCacheEntry{RepoURL: armoryPkg.RepoURL}
	defer func() {
		pkgCacheEntry.Fetched = time.Now()
		pkgCache.Store(armoryPkg.PublicKey, *pkgCacheEntry)
	}()

	repoURL, err := url.Parse(armoryPkg.RepoURL)
	if err != nil {
		pkgCacheEntry.LastErr = err
		return
	}
	if repoURL.Scheme != "https" && repoURL.Scheme != "http" {
		pkgCacheEntry.LastErr = errors.New("invalid URL scheme")
		return
	}

	var sig *minisign.Signature
	if pkgParser, ok := pkgParsers[repoURL.Hostname()]; ok {
		sig, _, err = pkgParser(armoryPkg, true, clientConfig)
	} else {
		sig, _, err = DefaultArmoryPkgParser(armoryPkg, true, clientConfig)
	}
	if sig != nil {
		pkgCacheEntry.Sig = *sig
	}
	if armoryPkg != nil {
		pkgCacheEntry.Pkg = *armoryPkg
	}
	pkgCacheEntry.LastErr = err
	if err == nil {
		manifestData, err := base64.StdEncoding.DecodeString(sig.TrustedComment)
		if err != nil {
			pkgCacheEntry.LastErr = err
			return
		}
		if armoryPkg.IsAlias {
			pkgCacheEntry.Alias, err = alias.ParseAliasManifest(manifestData)
		} else {
			pkgCacheEntry.Extension, err = extensions.ParseExtensionManifest(manifestData)
		}
		pkgCacheEntry.LastErr = err
	}
}
