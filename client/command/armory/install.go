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
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/alias"
	"github.com/bishopfox/sliver/client/command/extensions"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/util"
	"github.com/bishopfox/sliver/util/minisign"
)

// ErrPackageNotFound - The package was not found
// ErrPackageNotFound - 未找到 The 包
var ErrPackageNotFound = errors.New("package not found")
var ErrPackageAlreadyInstalled = errors.New("package is already installed")

const (
	doNotInstallOption      = "Do not install this package"
	doNotInstallPackageName = "do not install"
)

// ArmoryInstallCmd - The armory install command
// ArmoryInstallCmd - The armory 安装命令
func ArmoryInstallCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var promptToOverwrite bool
	names := []string{}
	if len(args) > 0 {
		names = []string{args[0]}
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
	// Find PK 表示 armory 名称
	armoryPK := getArmoryPublicKey(armoryName)

	// If the armory with the name is not found, print a warning
	// 未找到该名称的 If 和 armory，打印警告
	if cmd.Flags().Changed("armory") && armoryPK == "" {
		con.PrintWarnf("Could not find a configured armory named %q - searching all configured armories\n\n", armoryName)
	}

	clientConfig := parseArmoryHTTPConfig(cmd)
	refresh(clientConfig)

	if len(names) == 0 {
		armoryFilterName := ""
		if armoryPK != "" {
			armoryFilterName = armoryName
		}
		options := armoryInstallOptions(armoryPK, armoryFilterName)
		if len(options) == 0 {
			con.PrintInfof("No packages or bundles found\n")
			return
		}
		result, err := forms.ArmoryInstallForm(options)
		if err != nil {
			if errors.Is(err, forms.ErrUserAborted) {
				return
			}
			con.PrintErrorf("Armory install form failed: %s\n", err)
			return
		}
		if len(result.SelectedNames) == 0 {
			return
		}
		names = result.SelectedNames
	}
	if len(names) == 0 {
		con.PrintErrorf("A package or bundle name is required")
		return
	}
	if slices.Contains(names, "all") {
		names = []string{"all"}
	}

	for _, name := range names {
		if name == "" {
			continue
		}

		promptOverwrite := promptToOverwrite
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
			forms.Confirm(fmt.Sprintf("Install %d alias%s and %d extension%s?",
				aliasCount, pluralAliases, extCount, pluralExtensions,
			), &confirm)
			if !confirm {
				return
			}
			promptOverwrite = false
		}

		err = installPackageByName(name, armoryPK, forceInstallation, promptOverwrite, clientConfig, con)
		if err == nil {
			continue
		}
		if errors.Is(err, ErrPackageNotFound) {
			bundles := bundlesInCache()
			for _, bundle := range bundles {
				if bundle.Name == name {
					installBundle(bundle, armoryPK, forceInstallation, clientConfig, con)
					err = nil
					break
				}
			}
			if err == nil {
				continue
			}
			// If we have made it here, then there was not a bundle or package that matched the provided name
			// If 我们已经做到了这里，然后没有与提供的名称匹配的捆绑包或包
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
			// Some 扩展清单正在使用旧版本
			// To maintain compatibility with those extensions, we will
			// To 保持与这些扩展的兼容性，我们将
			// re-unmarshal the data as the older version
			// re__PH0__ 数据为旧版本
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
// This 是一个方便的函数，用于获取缓存中命令的名称
func getCommandsInCache(armoryPK string) []string {
	commandNames := []string{}

	pkgCache.Range(func(key, value interface{}) bool {
		cacheEntry := value.(pkgCacheEntry)
		if cacheEntry.LastErr == nil {
			if armoryPK != "" && cacheEntry.ArmoryConfig.PublicKey != armoryPK {
				return true
			}
			if cacheEntry.Pkg.IsAlias {
				if cacheEntry.Alias.CommandName != "" && !slices.Contains(commandNames, cacheEntry.Alias.CommandName) {
					commandNames = append(commandNames, cacheEntry.Alias.CommandName)
				}
			} else {
				for _, command := range cacheEntry.Extension.ExtCommand {
					if command.CommandName != "" && !slices.Contains(commandNames, command.CommandName) {
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
	// Add 取消选项
	optionKeys = append(optionKeys, doNotInstallOption)
	options[doNotInstallOption] = doNotInstallPackageName
	forms.Select(fmt.Sprintf("More than one package contains the command %s. Please choose an option from the list below:", name), optionKeys, &selectedPackageKey)
	selectedPackageID := options[selectedPackageKey]

	return selectedPackageID
}

func getPackageForCommand(name, armoryPK, minimumVersion string) (*pkgCacheEntry, error) {
	packagesWithCommand := getPackagesWithCommandName(name, armoryPK, minimumVersion)

	if len(packagesWithCommand) > 1 {
		// Build an option map for the user to choose from (option -> pkgID)
		// Build 供用户选择的选项映射（选项 -> pkgID）
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
		Gather 有关我们正在处理的内容的信息

		Find all conflicts within aliases for a given name (or all names), same thing with extensions
		Find 给定名称（或所有名称）的别名内的所有冲突，与扩展名相同
		Then if there are aliases and extensions with a given name, make sure to note that for when we ask the user what to do
		Then 如果有给定名称的别名和扩展名，请务必注意，以便当我们询问用户要做什么时
	*/
	var requestedPackageList []string
	if name == "all" {
		requestedPackageList = []string{}
		allCommands := getCommandsInCache(armoryPK)
		for _, cmdName := range allCommands {
			if !slices.Contains(installedPackages, cmdName) || forceInstallation {
				// Check to see if there is a package pending with that name
				// Check 查看是否有具有该名称的待处理包
				if _, ok := pendingPackages[cmdName]; !ok {
					requestedPackageList = append(requestedPackageList, cmdName)
				}
			}
		}
	} else {
		if !slices.Contains(installedPackages, name) || forceInstallation {
			// Check to see if there is a package pending with that name
			// Check 查看是否有具有该名称的待处理包
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
			// We 已经要安装具有此名称的软件包，因此不要尝试解析它
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
	if len(packageInstallList) == 0 && name == "all" {
		availableCommands := getCommandsInCache(armoryPK)
		if len(availableCommands) == 0 {
			con.PrintInfof("No packages or bundles found\n")
		} else {
			con.PrintInfof("All available packages are already installed\n")
		}
		return nil
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
		con.PrintSuccessf("Operation complete\n")
	}

	return nil
}

func armoryInstallOptions(armoryPK, armoryName string) []forms.ArmoryInstallOption {
	options := []forms.ArmoryInstallOption{}
	seen := make(map[string]bool)
	packageCount := 0

	pkgCache.Range(func(key, value interface{}) bool {
		cacheEntry, ok := value.(pkgCacheEntry)
		if !ok {
			return true
		}
		if cacheEntry.LastErr != nil {
			return true
		}
		if armoryPK != "" && cacheEntry.ArmoryConfig.PublicKey != armoryPK {
			return true
		}

		if cacheEntry.Pkg.IsAlias {
			name := cacheEntry.Alias.CommandName
			if name == "" {
				return true
			}
			if seen[name] {
				return true
			}
			label := formatArmoryInstallLabel("alias", name, cacheEntry.Alias.Help)
			options = append(options, forms.ArmoryInstallOption{
				Value: name,
				Label: label,
			})
			seen[name] = true
			packageCount++
			return true
		}

		for _, command := range cacheEntry.Extension.ExtCommand {
			name := command.CommandName
			if name == "" {
				continue
			}
			if seen[name] {
				continue
			}
			label := formatArmoryInstallLabel("extension", name, command.Help)
			options = append(options, forms.ArmoryInstallOption{
				Value: name,
				Label: label,
			})
			seen[name] = true
			packageCount++
		}

		return true
	})

	for _, bundle := range bundlesInCache() {
		if bundle.Name == "" {
			continue
		}
		if armoryName != "" && bundle.ArmoryName != armoryName {
			continue
		}
		if seen[bundle.Name] {
			continue
		}
		label := fmt.Sprintf("bundle: %s (%d packages)", bundle.Name, len(bundle.Packages))
		options = append(options, forms.ArmoryInstallOption{
			Value: bundle.Name,
			Label: label,
		})
		seen[bundle.Name] = true
	}

	if len(options) == 0 {
		return options
	}

	sort.Slice(options, func(i, j int) bool {
		return options[i].Label < options[j].Label
	})

	if packageCount > 0 {
		options = append([]forms.ArmoryInstallOption{
			{
				Value: "all",
				Label: "all packages (aliases + extensions)",
			},
		}, options...)
	}

	return options
}

func formatArmoryInstallLabel(kind, name, help string) string {
	help = strings.TrimSpace(help)
	if help == "" {
		return fmt.Sprintf("%s: %s", kind, name)
	}
	help = strings.Join(strings.Fields(help), " ")
	return fmt.Sprintf("%s: %s - %s", kind, name, help)
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

	tmpFileName, err := writeArmoryTempFile(tarGz)
	if err != nil {
		return err
	}
	defer os.Remove(tmpFileName)

	con.Printf(console.Clearln + "\r") // Clear the line
	con.Printf(console.Clearln + "\r") // Clear 线

	installPath := alias.InstallFromFile(tmpFileName, entry.Alias.CommandName, promptToOverwrite, con)
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
const maxDepDepth = 10 // Arbitrary 依赖项的递归限制

func resolveExtensionPackageDependencies(pkg *pkgCacheEntry, deps map[string]*pkgCacheEntry, pendingPackages map[string]string) error {
	for _, multiExt := range pkg.Extension.ExtCommand {
		if multiExt.DependsOn == "" {
			continue // Avoid adding empty dependency
			continue // Avoid 添加空依赖项
		}

		if multiExt.DependsOn == pkg.Extension.Name {
			continue // Avoid infinite loop of something that depends on itself
			continue // Avoid 依赖于自身的事物的无限循环
		}
		// We also need to look out for circular dependencies, so if we've already
		// We 还需要注意循环依赖，所以如果我们已经
		// seen this dependency, we stop resolving
		// 看到这种依赖关系，我们停止解析
		if _, ok := deps[multiExt.DependsOn]; ok {
			continue // Already resolved
			continue // Already 已解决
		}
		// Check to make sure we are not already going to install a package with this name
		// Check 以确保我们不会安装具有此名称的软件包
		if _, ok := pendingPackages[multiExt.DependsOn]; ok {
			continue
		}
		if maxDepDepth < len(deps) {
			continue
		}
		// Figure out what package we need for the dependency
		// Figure 出我们需要什么依赖包
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

	tmpFileName, err := writeArmoryTempFile(tarGz)
	if err != nil {
		return err
	}
	defer os.Remove(tmpFileName)

	con.Printf(console.Clearln + "\r") // Clear download message
	con.Printf(console.Clearln + "\r") // Clear 下载消息

	extensions.InstallFromDir(tmpFileName, promptToOverwrite, con, true)

	return nil
}

func writeArmoryTempFile(data []byte) (string, error) {
	if len(data) == 0 {
		return "", errors.New("downloaded archive is empty")
	}
	tmpFile, err := os.CreateTemp("", "sliver-armory-")
	if err != nil {
		return "", err
	}
	tmpFileName := tmpFile.Name()
	for len(data) > 0 {
		n, err := tmpFile.Write(data)
		if err != nil {
			tmpFile.Close()
			os.Remove(tmpFileName)
			return "", err
		}
		data = data[n:]
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		os.Remove(tmpFileName)
		return "", err
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFileName)
		return "", err
	}
	return tmpFileName, nil
}
