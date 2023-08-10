package armory

/*
   Sliver Implant Framework
   Copyright (C) 2019  Bishop Fox

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
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/rsteube/carapace"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/alias"
	"github.com/bishopfox/sliver/client/command/extensions"
)

// AliasExtensionOrBundleCompleter - Completer for alias, extension, and bundle names.
func AliasExtensionOrBundleCompleter() carapace.Action {
	comps := func(ctx carapace.Context) carapace.Action {
		var action carapace.Action

		results := []string{}

		// In-memory packages are newer.
		aliases, exts := packagesInCache()
		bundles := bundlesInCache()

		// Or load the cache from file if in-memory cache is empty.
		// Inform user if the cache file is old (1 week or more).
		if len(aliases)+len(exts)+len(bundles) == 0 {
			filePath := filepath.Join(assets.GetRootAppDir(), armoryCacheFileName)

			info, err := os.Stat(filePath)
			if err == nil {
				mustUpdateCache := time.Since(info.ModTime()) > (24 * 7 * time.Hour)
				if mustUpdateCache {
					modTime := time.Since(info.ModTime()).Truncate(time.Hour)
					action = carapace.ActionMessage("armory cache is %d old, `sliver armory update` recommended", modTime)
				}

				aliases, exts, bundles, err = loadArmoryCompletionCache(filePath)
				if err != nil {
					return carapace.ActionMessage("failed to read armory file cache: %s", err)
				}
			}
		}

		for _, aliasPkg := range aliases {
			results = append(results, aliasPkg.CommandName)
			results = append(results, aliasPkg.Help)
		}
		aliasesComps := carapace.ActionValuesDescribed(results...).Tag("aliases").Invoke(ctx)
		results = make([]string, 0)

		for _, extensionPkg := range exts {
			results = append(results, extensionPkg.CommandName)
			results = append(results, extensionPkg.Help)
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

func saveArmoryCompletionCache() error {
	aliases, exts := packagesInCache()
	bundles := bundlesInCache()

	ArmoryCache := struct {
		Aliases    []*alias.AliasManifest
		Extensions []*extensions.ExtensionManifest
		Bundles    []*ArmoryBundle
	}{
		Aliases:    aliases,
		Extensions: exts,
		Bundles:    bundles,
	}

	data, err := json.MarshalIndent(ArmoryCache, "", "    ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(assets.GetRootAppDir(), armoryCacheFileName)

	return os.WriteFile(filePath, data, 0o600)
}

func loadArmoryCompletionCache(filePath string) ([]*alias.AliasManifest, []*extensions.ExtensionManifest, []*ArmoryBundle, error) {
	data, err := os.ReadFile(filePath)
	if err != nil && os.IsNotExist(err) {
		return nil, nil, nil, errors.New("no armory file cache")
	}

	ArmoryCache := struct {
		Aliases    []*alias.AliasManifest
		Extensions []*extensions.ExtensionManifest
		Bundles    []*ArmoryBundle
	}{}

	err = json.Unmarshal(data, &ArmoryCache)
	if err != nil {
		return nil, nil, nil, err
	}

	return ArmoryCache.Aliases, ArmoryCache.Extensions, ArmoryCache.Bundles, nil
}
