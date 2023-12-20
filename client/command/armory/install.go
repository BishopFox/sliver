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
	"net/url"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/command/alias"
	"github.com/bishopfox/sliver/client/command/extensions"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/util/minisign"
)

// ErrPackageNotFound - The package was not found
var ErrPackageNotFound = errors.New("package not found")

// ArmoryInstallCmd - The armory install command
func ArmoryInstallCmd(cmd *cobra.Command, con *console.SliverConsoleClient, args []string) {
	name := args[0]
	// name := ctx.Args.String("name")
	if name == "" {
		con.PrintErrorf("A package or bundle name is required")
		return
	}
	clientConfig := parseArmoryHTTPConfig(cmd)
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
	con.PrintErrorf("No package or bundle named '%s' was found", name)
}

func installBundle(bundle *ArmoryBundle, clientConfig ArmoryHTTPConfig, con *console.SliverConsoleClient) {
	for _, pkgName := range bundle.Packages {
		err := installPackageByName(pkgName, clientConfig, con)
		if err != nil {
			con.PrintErrorf("Failed to install '%s': %s", pkgName, err)
		}
	}
}

func installPackageByName(name string, clientConfig ArmoryHTTPConfig, con *console.SliverConsoleClient) error {
	aliases, extensions := packagesInCache()
	for _, alias := range aliases {
		if alias.CommandName == name || name == "all" {
			installAlias(alias, clientConfig, con)
			if name != "all" {
				return nil
			}
		}
	}
	for _, extm := range extensions {
		for _, ext := range extm.ExtCommand {
			if ext.CommandName == name || name == "all" {
				installExtension(ext.Manifest, clientConfig, con)
				if name != "all" {
					return nil
				}
			}
		}
	}
	if name == "all" {
		con.Printf("\n")
		con.PrintInfof("All packages installed\n")
		return nil
	}
	return ErrPackageNotFound
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

	con.PrintInfof("Downloading alias ...")

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

	tmpFile, err := os.CreateTemp("", "sliver-armory-")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.Write(tarGz)
	if err != nil {
		return err
	}
	tmpFile.Close()

	con.Printf(console.Clearln + "\r") // Clear the line

	installPath := alias.InstallFromFile(tmpFile.Name(), true, con)
	if installPath == nil {
		return errors.New("failed to install alias")
	}

	menuCmd := con.App.Menu(constants.ImplantMenu).Root()

	_, err = alias.LoadAlias(filepath.Join(*installPath, alias.ManifestFileName), menuCmd, con)
	if err != nil {
		return err
	}
	return nil
}

func installExtension(extm *extensions.ExtensionManifest, clientConfig ArmoryHTTPConfig, con *console.SliverConsoleClient) {
	deps := make(map[string]struct{})
	for _, ext := range extm.ExtCommand {
		resolveExtensionPackageDependencies(ext.CommandName, deps, clientConfig, con)
		sliverMenu := con.App.Menu(constants.ImplantMenu).Root()
		for dep := range deps {
			if extensions.CmdExists(dep, sliverMenu) {
				continue // Dependency is already installed
			}
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
}

const maxDepDepth = 10 // Arbitrary recursive limit for dependencies

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
	for _, multiExt := range entry.Extension.ExtCommand {
		if multiExt.DependsOn == "" {
			continue // Avoid adding empty dependency
		}

		if multiExt.DependsOn == name {
			continue // Avoid infinite loop of something that depends on itself
		}
		// We also need to look out for circular dependencies, so if we've already
		// seen this dependency, we stop resolving
		if _, ok := deps[multiExt.DependsOn]; ok {
			continue // Already resolved
		}
		if maxDepDepth < len(deps) {
			continue
		}
		deps[multiExt.DependsOn] = struct{}{}
		resolveExtensionPackageDependencies(multiExt.DependsOn, deps, clientConfig, con)
	}
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

	con.PrintInfof("Downloading extension ...")

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

	tmpFile, err := os.CreateTemp("", "sliver-armory-")
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

	con.Printf(console.Clearln + "\r") // Clear download message

	extensions.InstallFromDir(tmpFile.Name(), con, true)

	return nil
}
