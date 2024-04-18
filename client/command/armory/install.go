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
	"github.com/bishopfox/sliver/util"
	"github.com/bishopfox/sliver/util/minisign"
)

// ErrPackageNotFound - The package was not found
var ErrPackageNotFound = errors.New("package not found")
var ErrPackageAlreadyInstalled = errors.New("package is already installed")

const (
	doNotInstallOption      = "Do not install this package"
	doNotInstallPackageName = "do not install"
)

// ArmoryInstallCmd - The armory install command
func ArmoryInstallCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var promptToOverwrite bool
	name := args[0]
	if name == "" {
		con.PrintErrorf("A package or bundle name is required")
		return
	}
	forceInstallation, err := cmd.Flags().GetBool("force")
	if err != nil {
		con.PrintErrorf("Could not parse %q flag: %s\n", "force", err)
		return
	}
	if forceInstallation {
		promptToOverwrite = false
	} else {
		promptToOverwrite = true
	}

	armoryName, err := cmd.Flags().GetString("armory")
	if err != nil {
		con.PrintErrorf("Could not parse %q flag: %s\n", "armory", err)
		return
	}

	// Find PK for the armory name
	armoryPK := getArmoryPublicKey(armoryName)

	// If the armory with the name is not found, print a warning
	if cmd.Flags().Changed("armory") && armoryPK == "" {
		con.PrintWarnf("Could not find a configured armory named %q - searching all configured armories\n\n", armoryName)
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
		promptToOverwrite = false
	}
	err = installPackageByName(name, armoryPK, forceInstallation, promptToOverwrite, clientConfig, con)
	if err == nil {
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
		// If we have made it here, then there was not a bundle or package that matched the provided name
		if armoryPK == "" {
			con.PrintErrorf("No package or bundle named %q was found\n", name)
		} else {
			con.PrintErrorf("No package or bundle named %q was found in armory %s\n", name, armoryName)
		}
	} else if errors.Is(err, ErrPackageAlreadyInstalled) {
		con.PrintErrorf("Package %q is already installed - use the force option to overwrite it\n", name)
	} else {
		con.PrintErrorf("Could not install package: %s\n", err)
	}

}

func installBundle(bundle *ArmoryBundle, armoryPK string, forceInstallation bool, clientConfig ArmoryHTTPConfig, con *console.SliverClient) {
	installList := []string{}
	pendingPackages := make(map[string]string)

	for _, bundlePkgName := range bundle.Packages {
		packageInstallList, err := buildInstallList(bundlePkgName, armoryPK, forceInstallation, pendingPackages)
		if err != nil {
			if errors.Is(err, ErrPackageAlreadyInstalled) {
				con.PrintInfof("Package %s is already installed. Skipping...\n", bundlePkgName)
				continue
			} else {
				con.PrintErrorf("Error for package %s: %s\n", bundlePkgName, err)
				return
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
			con.PrintErrorf("The package cache is out of date. Please run armory refresh and try again.\n")
			return
		}
		if packageEntry.Pkg.IsAlias {
			err := installAliasPackage(packageEntry, false, clientConfig, con)
			if err != nil {
				con.PrintErrorf("Failed to install alias '%s': %s", packageEntry.Alias.CommandName, err)
				return
			}
		} else {
			err := installExtensionPackage(packageEntry, false, clientConfig, con)
			if err != nil {
				con.PrintErrorf("Failed to install extension '%s': %s", packageEntry.Extension.Name, err)
				return
			}
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
	selectedPackageKey := ""
	optionKeys := util.Keys(options)
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
			err = resolveExtensionPackageDependencies(packageEntry, dependencies, pendingPackages)
			if err != nil {
				return nil, err
			}
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

func installPackageByName(name, armoryPK string, forceInstallation, promptToOverwrite bool, clientConfig ArmoryHTTPConfig, con *console.SliverClient) error {
	pendingPackages := make(map[string]string)
	packageInstallList, err := buildInstallList(name, armoryPK, forceInstallation, pendingPackages)
	if err != nil {
		return err
	}
	if len(packageInstallList) > 0 {
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
		con.Printf("\n")
		con.PrintInfof("Operation complete\n")
	}

	return nil
}

func installAliasPackage(entry *pkgCacheEntry, promptToOverwrite bool, clientConfig ArmoryHTTPConfig, con *console.SliverClient) error {
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

	installPath := alias.InstallFromFile(tmpFile.Name(), entry.Alias.CommandName, promptToOverwrite, con)
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

func installExtensionPackage(entry *pkgCacheEntry, promptToOverwrite bool, clientConfig ArmoryHTTPConfig, con *console.SliverClient) error {
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

	extensions.InstallFromDir(tmpFile.Name(), promptToOverwrite, con, true)

	return nil
}
