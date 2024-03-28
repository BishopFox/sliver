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
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/alias"
	"github.com/bishopfox/sliver/client/command/extensions"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/util/minisign"
)

// ErrPackageNotFound - The package was not found
var ErrPackageNotFound = errors.New("package not found")

const (
	doNotInstallOption      = "Do not install this package"
	doNotInstallPackageName = "do not install"
)

// ArmoryInstallCmd - The armory install command
func ArmoryInstallCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	name := args[0]
	// name := ctx.Args.String("name")
	if name == "" {
		con.PrintErrorf("A package or bundle name is required")
		return
	}
	clientConfig := parseArmoryHTTPConfig(cmd)
	refresh(clientConfig)
	if name == "all" {
		aliases, extensions := packageManifestsInCache()
		aliasCount, extCount := countUniqueCommandsFromManifests(aliases, extensions)
		confirm := false
		pluralAliases := "es"
		if aliasCount == 1 {
			pluralAliases = ""
		}
		pluralExtensions := "s"
		if extCount == 1 {
			pluralExtensions = ""
		}
		prompt := &survey.Confirm{
			Message: fmt.Sprintf("Install %d alias%s and %d extension%s?",
				aliasCount, pluralAliases, extCount, pluralExtensions,
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

func installBundle(bundle *ArmoryBundle, clientConfig ArmoryHTTPConfig, con *console.SliverClient) {
	for _, pkgName := range bundle.Packages {
		err := installPackageByName(pkgName, clientConfig, con)
		if err != nil {
			con.PrintErrorf("Failed to install '%s': %s", pkgName, err)
		}
	}
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
		extension := &extensions.ExtensionManifest{}
		manifestData, err := os.ReadFile(extensionFileName)
		if err != nil {
			continue
		}
		err = json.Unmarshal(manifestData, extension)
		if err != nil {
			continue
		}
		if len(extension.ExtCommand) == 0 {
			extensionOld := &extensions.ExtensionManifest_{}
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
			for _, command := range extension.ExtCommand {
				if !slices.Contains(packageNames, command.CommandName) {
					packageNames = append(packageNames, command.CommandName)
				}
			}
		}
		/*if !slices.Contains(packageNames, extension.Name) {
			packageNames = append(packageNames, extension.Name)
		}*/
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

func getPackagesWithCommandName(name string) []*pkgCacheEntry {
	packages := []*pkgCacheEntry{}

	pkgCache.Range(func(key, value interface{}) bool {
		cacheEntry := value.(pkgCacheEntry)
		if cacheEntry.LastErr == nil {
			if cacheEntry.Pkg.IsAlias {
				if cacheEntry.Alias.CommandName == name {
					packages = append(packages, &cacheEntry)
				}
			} else {
				for _, command := range cacheEntry.Extension.ExtCommand {
					if command.CommandName == name {
						packages = append(packages, &cacheEntry)
						break
					}
				}
			}
		}
		return true
	})

	return packages
}

func keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func getPackageIDFromUser(name string, options map[string]string) string {
	selectedPackageKey := ""
	optionKeys := keys(options)
	slices.Sort(optionKeys)
	// Add a cancel option
	optionKeys = append(optionKeys, doNotInstallOption)
	options[doNotInstallOption] = doNotInstallPackageName
	prompt := &survey.Select{
		Message: fmt.Sprintf("More than one package contains the command %s. Please choose an option from the list below:", name),
		Options: optionKeys,
	}
	survey.AskOne(prompt, &selectedPackageKey)
	selectedPackageID := options[selectedPackageKey]

	return selectedPackageID
}

func getPackageForCommand(name string) (*pkgCacheEntry, error) {
	packagesWithCommand := getPackagesWithCommandName(name)

	if len(packagesWithCommand) > 1 {
		// Build an option map for the user to choose from (option -> pkgID)
		optionMap := make(map[string]string)
		for _, packageEntry := range packagesWithCommand {
			var optionName string
			if packageEntry.Pkg.IsAlias {
				optionName = fmt.Sprintf("Alias %s from armory %s (%s)", name, packageEntry.ArmoryConfig.Name, packageEntry.Pkg.RepoURL)
			} else {
				optionName = fmt.Sprintf("Extension %s from armory %s with command %s (%s)",
					packageEntry.Pkg.Name,
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

func buildInstallList(name string) ([]*pkgCacheEntry, error) {
	packageInstallList := []*pkgCacheEntry{}

	/*
		Gather information about what we are working with

		Find all conflicts within aliases for a given name (or all names), same thing with extensions
		Then if there are aliases and extensions with a given name, make sure to note that for when we ask the user what to do
	*/
	if name != "all" {
		packageEntry, err := getPackageForCommand(name)
		if err != nil {
			return nil, err
		}
		packageInstallList = append(packageInstallList, packageEntry)
	} else {
		// Get a list of all command names and resolve conflicts as we find them
		commandNames := getCommandsInCache()
		for _, cmdName := range commandNames {
			packageEntry, err := getPackageForCommand(cmdName)
			if err != nil {
				return nil, err
			}
			packageInstallList = append(packageInstallList, packageEntry)
		}
	}

	return packageInstallList, nil
}

func installPackageByName(name string, clientConfig ArmoryHTTPConfig, con *console.SliverClient) error {
	packageInstallList, err := buildInstallList(name)
	if err != nil {
		return err
	}
	if len(packageInstallList) > 0 {
		for _, entry := range packageInstallList {
			if entry.Pkg.IsAlias {
				installAlias(entry, clientConfig, con)
			} else {
				installExtension(entry, clientConfig, con)
			}
		}
	} else {
		return ErrPackageNotFound
	}

	if name == "all" {
		con.Printf("\n")
		con.PrintInfof("Operation complete\n")
	}

	return nil
}

func installAlias(alias *pkgCacheEntry, clientConfig ArmoryHTTPConfig, con *console.SliverClient) {
	err := installAliasPackageByName(alias, clientConfig, con)
	if err != nil {
		con.PrintErrorf("Failed to install alias '%s': %s", alias.Pkg.CommandName, err)
		return
	}
}

func installAliasPackageByName(entry *pkgCacheEntry, clientConfig ArmoryHTTPConfig, con *console.SliverClient) error {
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

	installPath := alias.InstallFromFile(tmpFile.Name(), entry.Alias.CommandName, true, con)
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

func installExtension(extPkg *pkgCacheEntry, clientConfig ArmoryHTTPConfig, con *console.SliverClient) {
	if extPkg == nil {
		con.PrintErrorf("requested extension does not have a valid package entry")
		return
	}
	if extPkg.Pkg.IsAlias {
		con.PrintErrorf("%s is not an extension", extPkg.Pkg.Name)
		return
	}
	// Name -> package ID
	deps := make(map[string]string)
	// Resolve dependencies
	var entry *pkgCacheEntry = extPkg
	var err error
	installedExtensions := getInstalledPackageNames()

	for _, ext := range extPkg.Extension.ExtCommand {
		if ext.CommandName != entry.Pkg.CommandName {
			// Then we need to get the package for this command
			entry, err = getPackageForCommand(ext.CommandName)
			if err != nil {
				con.PrintErrorf("Could not get package information for command %s: %s", ext.CommandName, err)
				return
			}
		}
		err = resolveExtensionPackageDependencies(entry, deps, clientConfig, con)
		if err != nil {
			con.PrintErrorf("Could not resolve dependencies for %s: %s\n", extPkg.Extension.Name, err)
			return
		}

		for dep, depKey := range deps {
			value, ok := pkgCache.Load(depKey)
			if !ok {
				con.PrintErrorf("Could not satisfy dependency %s for %s. Please refresh the armory information and try again.", dep, ext.CommandName)
				return
			}
			extensionPackage := value.(pkgCacheEntry)
			if extensionPackage.Extension == nil {
				continue
			}

			if slices.Contains(installedExtensions, dep) {
				continue // Dependency is already installed
			}

			err := installExtensionPackageByName(&extensionPackage, clientConfig, con)
			if err != nil {
				con.PrintErrorf("Failed to install extension dependency '%s': %s", dep, err)
				return
			}
			installedExtensions = append(installedExtensions, dep)
		}
	}

	// Now that depedencies are resolved, install the package
	err = installExtensionPackageByName(extPkg, clientConfig, con)
	if err != nil {
		con.PrintErrorf("Failed to install extension '%s': %s", extPkg.Extension.Name, err)
		return
	}
}

const maxDepDepth = 10 // Arbitrary recursive limit for dependencies

func resolveExtensionPackageDependencies(pkg *pkgCacheEntry, deps map[string]string, clientConfig ArmoryHTTPConfig, con *console.SliverClient) error {
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
		if maxDepDepth < len(deps) {
			continue
		}
		// Figure out what package we need for the dependency
		dependencyEntry, err := getPackageForCommand(multiExt.DependsOn)
		if err != nil {
			return fmt.Errorf("could not resolve dependency %s for %s: %s", multiExt.DependsOn, pkg.Extension.Name, err)
		}
		deps[multiExt.DependsOn] = dependencyEntry.ID
		err = resolveExtensionPackageDependencies(dependencyEntry, deps, clientConfig, con)
		if err != nil {
			return err
		}
	}
	return nil
}

func installExtensionPackageByName(entry *pkgCacheEntry, clientConfig ArmoryHTTPConfig, con *console.SliverClient) error {
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
