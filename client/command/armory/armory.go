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
	"errors"
	"net/url"
	"sync"
	"time"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/console"
	"github.com/desertbit/grumble"
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
}

type ArmoryHTTPConfig struct {
	SkipCache            bool
	ProxyURL             *url.URL
	Timeout              time.Duration
	DisableTLSValidation bool
}

type armoryIndexCacheEntry struct {
	Fetched time.Time
	Index   ArmoryIndex
	LastErr error
}

type armoryPkgCacheEntry struct {
	Fetched time.Time
	Pkg     ArmoryPackage
	LastErr error
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
	con.PrintInfof("Fetching %d armory index(es) ...", len(armoriesConfig))
	clientConfig := parseArmoryHTTPConfig()
	indexes := fetchIndexes(armoriesConfig, clientConfig)
	if 0 < len(indexes) {
		con.PrintInfof("Fetching package information ...", len(armoriesConfig))
		fetchPackageSignatures(indexes, clientConfig)
	}

}

func parseArmoryHTTPConfig() ArmoryHTTPConfig {
	return ArmoryHTTPConfig{
		SkipCache:            false,
		ProxyURL:             nil,
		Timeout:              defaultTimeout,
		DisableTLSValidation: false,
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
		cacheEntry := value.(armoryIndexCacheEntry)
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
		cached := cacheEntry.(*armoryIndexCacheEntry)
		if time.Since(cached.Fetched) < cacheTime && cached.LastErr == nil && !clientConfig.SkipCache {
			return
		}
	}

	armoryResult := &armoryIndexCacheEntry{}
	defer func() {
		armoryResult.Fetched = time.Now()
		indexCache.Store(armoryConfig.PublicKey, armoryResult)
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
	armoryResult.Index = *index
	armoryResult.LastErr = err
}

func fetchPackageSignatures(indexes []ArmoryIndex, clientConfig ArmoryHTTPConfig) {
	wg := &sync.WaitGroup{}
	for _, index := range indexes {
		for _, armoryPkg := range index.Extensions {
			wg.Add(1)
			go fetchPackageSignature(wg, armoryPkg.RepoURL, armoryPkg.PublicKey, clientConfig)
		}
		for _, armoryPkg := range index.Aliases {
			wg.Add(1)
			go fetchPackageSignature(wg, armoryPkg.RepoURL, armoryPkg.PublicKey, clientConfig)
		}
	}
	wg.Wait()
}

func fetchPackageSignature(wg *sync.WaitGroup, rawRepoURL string, rawPublicKey string, clientConfig ArmoryHTTPConfig) {
	defer wg.Done()
	cacheEntry, ok := pkgCache.Load(rawPublicKey)
	if ok {
		cached := cacheEntry.(*armoryIndexCacheEntry)
		if time.Since(cached.Fetched) < cacheTime && cached.LastErr == nil && !clientConfig.SkipCache {
			return
		}
	}

	var pkgCacheEntry *armoryPkgCacheEntry
	defer func() {
		pkgCacheEntry.Fetched = time.Now()
		indexCache.Store(rawPublicKey, pkgCacheEntry)
	}()

	repoURL, err := url.Parse(rawRepoURL)
	if err != nil {
		pkgCacheEntry.LastErr = err
		return
	}
	if repoURL.Scheme != "https" && repoURL.Scheme != "http" {
		pkgCacheEntry.LastErr = errors.New("invalid URL scheme")
		return
	}

	var pkg *ArmoryPackage
	if pkgParser, ok := pkgParsers[repoURL.Hostname()]; ok {
		pkg, err = pkgParser(rawRepoURL, rawPublicKey, true, clientConfig)
	} else {
		pkg, err = DefaultArmoryPkgParser(rawRepoURL, rawPublicKey, true, clientConfig)
	}
	pkgCacheEntry.Pkg = *pkg
	pkgCacheEntry.LastErr = err
}
