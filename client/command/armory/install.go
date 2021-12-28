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
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/bishopfox/sliver/client/command/alias"
	"github.com/bishopfox/sliver/client/command/extensions"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/server/cryptography/minisign"
	"github.com/desertbit/grumble"
)

// ArmoryInstallCmd - The armory install command
func ArmoryInstallCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	name := ctx.Args.String("name")
	if name == "" {
		con.PrintErrorf("Package name is required")
		return
	}
	clientConfig := parseArmoryHTTPConfig(ctx)

	refresh(clientConfig)
	aliases, extensions := packagesInCache()
	for _, alias := range aliases {
		if alias.CommandName == name {
			installAlias(alias, clientConfig, con)
			return
		}
	}
	for _, ext := range extensions {
		if ext.CommandName == name {
			installExtension(ext, clientConfig, con)
			return
		}
	}
	con.PrintErrorf("Package '%s' not found", name)
}

func installAlias(alias *alias.AliasManifest, clientConfig ArmoryHTTPConfig, con *console.SliverConsoleClient) {
	err := installAliasPackageByName(alias.CommandName, clientConfig, con)
	if err != nil {
		con.PrintErrorf("Failed to install alias '%s': %s", alias.CommandName, err)
		return
	}
}

func installAliasPackageByName(name string, clientConfig ArmoryHTTPConfig, con *console.SliverConsoleClient) error {
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

	var sig *minisign.Signature
	var tarGz []byte
	if pkgParser, ok := pkgParsers[repoURL.Hostname()]; ok {
		sig, tarGz, err = pkgParser(&entry.Pkg, false, clientConfig)
	} else {
		sig, tarGz, err = DefaultArmoryPkgParser(&entry.Pkg, false, clientConfig)
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

	installPath := alias.InstallFromFile(tmpFile.Name(), con)
	if installPath == nil {
		return errors.New("failed to install alias")
	}
	_, err = alias.LoadAlias(filepath.Join(*installPath, alias.ManifestFileName), con)
	if err != nil {
		return err
	}
	return nil
}

func installExtension(ext *extensions.ExtensionManifest, clientConfig ArmoryHTTPConfig, con *console.SliverConsoleClient) {
	deps := make(map[string]struct{})
	resolveExtensionPackageDependencies(ext.CommandName, deps, clientConfig, con)
	for dep := range deps {
		err := installExtensionPackageByName(dep, clientConfig, con)
		if err != nil {
			con.PrintErrorf("Failed to install extension dependency '%s': %s", dep, err)
			return
		}
	}
	err := installExtensionPackageByName(ext.CommandName, clientConfig, con)
	if err != nil {
		con.PrintErrorf("Failed to install extension '%s': %s", ext.CommandName, err)
		return
	}
}

func resolveExtensionPackageDependencies(name string, deps map[string]struct{}, clientConfig ArmoryHTTPConfig, con *console.SliverConsoleClient) {
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
	if entry.Extension.DependsOn == name {
		return // Avoid infinite loop of something that depends on itself
	}
	// We also need to look out for circular dependencies, so if we've already
	// seen this dependency, we stop resolving
	if _, ok := deps[entry.Extension.DependsOn]; ok {
		return // Already resolved
	}
	deps[entry.Extension.DependsOn] = struct{}{}
	resolveExtensionPackageDependencies(entry.Extension.DependsOn, deps, clientConfig, con)
}

func installExtensionPackageByName(name string, clientConfig ArmoryHTTPConfig, con *console.SliverConsoleClient) error {
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

	var sig *minisign.Signature
	var tarGz []byte
	if pkgParser, ok := pkgParsers[repoURL.Hostname()]; ok {
		sig, tarGz, err = pkgParser(&entry.Pkg, false, clientConfig)
	} else {
		sig, tarGz, err = DefaultArmoryPkgParser(&entry.Pkg, false, clientConfig)
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

	installPath := extensions.InstallFromFilePath(tmpFile.Name(), con)
	if installPath == nil {
		return errors.New("failed to install extension")
	}
	extCmd, err := extensions.LoadExtensionManifest(filepath.Join(*installPath, extensions.ManifestFileName))
	if err != nil {
		return err
	}

	// do not add if the command already exists
	if extensions.CmdExists(extCmd.Name, con.App) {
		return fmt.Errorf("%s command already exists", extCmd.Name)
	}
	extensions.ExtensionRegisterCommand(extCmd, con)
	return nil
}
