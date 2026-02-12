package armory

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/settings"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func printArmories(con *console.SliverClient) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))

	tw.AppendHeader(table.Row{
		"Armory Name",
		"Authorization Required",
		"Enabled",
		"URL",
	})

	configuredArmories := getCurrentArmoryConfiguration()

	for _, armory := range configuredArmories {
		var authRequired string
		var enabled string
		if armory.Authorization != "" {
			authRequired = "yes"
		} else {
			authRequired = "no"
		}
		if armory.Enabled {
			enabled = "yes"
		} else {
			enabled = "no"
		}
		tw.AppendRow(table.Row{
			armory.Name,
			authRequired,
			enabled,
			armory.RepoURL,
		})
	}

	con.Printf("%s\n", tw.Render())
}

func generateRowsForArmory(armoryInfo *assets.ArmoryConfig) []table.Row {
	rows := make([]table.Row, 0)
	rows = append(rows, table.Row{
		"Name",
		armoryInfo.Name,
	})

	rows = append(rows, table.Row{
		"Public Key",
		armoryInfo.PublicKey,
	})

	enabled := "yes"
	if !armoryInfo.Enabled {
		enabled = "no"
	}

	rows = append(rows, table.Row{
		"Enabled",
		enabled,
	})

	authorizationRequired := "no"
	if armoryInfo.Authorization != "" {
		authorizationRequired = fmt.Sprintf("yes; current authorization: %s", armoryInfo.Authorization)
	}
	rows = append(rows, table.Row{
		"Authorization required?",
		authorizationRequired,
	})
	if authorizationRequired != "no" {
		authCmd := armoryInfo.AuthorizationCmd
		if authCmd == "" {
			authCmd = "not configured"
		}
		rows = append(rows, table.Row{
			"Authorization Command",
			authCmd,
		})
	}

	rows = append(rows, table.Row{
		"URL",
		armoryInfo.RepoURL,
	})

	return rows
}

func generateRowsForPackage(packageInfo *pkgCacheEntry) []table.Row {
	rows := make([]table.Row, 0)

	rows = append(rows, table.Row{
		"Name",
		packageInfo.Pkg.Name,
	})

	rows = append(rows, table.Row{
		"Command Name",
		packageInfo.Pkg.CommandName,
	})

	rows = append(rows, table.Row{
		"Armory Name",
		packageInfo.Pkg.ArmoryName,
	})

	packageType := "alias"
	if !packageInfo.Pkg.IsAlias {
		packageType = "extension"
	}
	rows = append(rows, table.Row{
		"Package Type",
		packageType,
	})
	if packageType == "alias" {
		rows = append(rows, table.Row{
			"Version",
			packageInfo.Alias.Version,
		})
		rows = append(rows, table.Row{
			"Author",
			packageInfo.Alias.OriginalAuthor,
		})
		rows = append(rows, table.Row{
			"Repo URL",
			packageInfo.Alias.RepoURL,
		})
		rows = append(rows, table.Row{
			"Help",
			packageInfo.Alias.Help,
		})
	} else {
		rows = append(rows, table.Row{
			"Version",
			packageInfo.Extension.Version,
		})
		rows = append(rows, table.Row{
			"Original Author",
			packageInfo.Extension.OriginalAuthor,
		})
		rows = append(rows, table.Row{
			"Extension Author",
			packageInfo.Extension.ExtensionAuthor,
		})
		rows = append(rows, table.Row{
			"URL",
			packageInfo.Extension.RepoURL,
		})
		extCommands := []string{}
		for _, cmd := range packageInfo.Extension.ExtCommand {
			extCommands = append(extCommands, cmd.CommandName)
		}
		rows = append(rows, table.Row{
			"Contains commands",
			strings.Join(extCommands, ", "),
		})
		rows = append(rows, table.Row{"", ""})
		rows = append(rows, table.Row{"Command Info", ""})
		for _, extCmd := range packageInfo.Extension.ExtCommand {
			rows = append(rows, table.Row{
				"Command Name",
				extCmd.CommandName,
			})
			rows = append(rows, table.Row{
				"Dependencies",
				extCmd.DependsOn,
			})
			rows = append(rows, table.Row{
				"Help",
				extCmd.Help,
			})
			rows = append(rows, table.Row{"", ""})
		}
	}
	rows = append(rows, table.Row{"", ""})

	return rows
}

func viewDetailedInformation(con *console.SliverClient, entity string) {
	tw := table.NewWriter()
	tw.SetStyle(settings.GetTableStyle(con))

	tw.AppendHeader(table.Row{
		"Property",
		"Value",
	})

	// See where this name exists in the armory and package indexes
	// See，该名称存在于 armory 和包索引中
	armoryResult := armoryLookupByName(entity)
	if armoryResult != nil {
		tw.AppendRows(generateRowsForArmory(armoryResult))
		con.Printf("Armory information for %s\n\n", entity)
		con.Printf("%s\n", tw.Render())
		tw.ResetRows()
		con.Printf("\n")
	}

	// Get packages with this name
	// 具有此名称的 Get 包
	packageResult := packageCacheLookupByName(entity)
	if len(packageResult) > 0 {
		con.Printf("Packages named %s\n\n", entity)
		for _, pkg := range packageResult {
			tw.AppendRows(generateRowsForPackage(pkg))
			con.Printf("%s\n", tw.Render())
			tw.ResetRows()
			con.Printf("\n")
		}
	}

	// Get extensions containing commands with this name
	// Get 扩展包含具有此名称的命令
	commandResult := packageCacheLookupByCmd(entity)
	commandResultFiltered := make([]*pkgCacheEntry, 0)
	for _, cmd := range commandResult {
		if !cmd.Pkg.IsAlias {
			commandResultFiltered = append(commandResultFiltered, cmd)
		}
	}
	if len(commandResultFiltered) > 0 {
		con.Printf("Packages containing commands named %s\n\n", entity)
		for _, cmd := range commandResultFiltered {
			tw.AppendRows(generateRowsForPackage(cmd))
			con.Printf("%s\n", tw.Render())
			tw.ResetRows()
		}
	}

}

func ArmoryInfoCommand(cmd *cobra.Command, con *console.SliverClient, args []string) {
	if len(args) == 0 {
		printArmories(con)
		return
	}
	// If the user wants info on something specific, make sure the caches are up to date
	// If 用户想要有关特定内容的信息，请确保缓存是最新的
	refresh(parseArmoryHTTPConfig(cmd))
	viewDetailedInformation(con, args[0])
}

func verifyArmory(armoryInfo *assets.ArmoryConfig, clientConfig ArmoryHTTPConfig) error {
	wg := &sync.WaitGroup{}
	// Only making one request, so we can limit ourselves to one concurrent request
	// Only 发出一个请求，因此我们可以将自己限制为一个并发请求
	requestChannel := make(chan struct{}, 1)
	wg.Add(1)
	requestChannel <- struct{}{}
	go fetchIndex(armoryInfo, requestChannel, clientConfig, wg)
	wg.Wait()

	result, ok := indexCache.Load(armoryInfo.PublicKey)
	if !ok {
		return fmt.Errorf("failed to retrieve armory index")
	}
	cacheResult := result.(indexCacheEntry)
	return cacheResult.LastErr
}

func getCurrentArmoryConfiguration() []*assets.ArmoryConfig {
	configs := []*assets.ArmoryConfig{}
	armoryNames := []string{}
	// If the default armory is in the configuration, force it to be last
	// If 默认 armory 在配置中，强制将其放在最后
	var defaultConfig *assets.ArmoryConfig

	currentArmories.Range(func(key, value interface{}) bool {
		armoryEntry := value.(assets.ArmoryConfig)
		// Skip over the default armory for now
		// 目前 Skip 超过默认 armory
		if armoryEntry.Name != assets.DefaultArmoryName {
			configs = append(configs, &armoryEntry)
			armoryNames = append(armoryNames, armoryEntry.Name)
		} else {
			defaultConfig = &armoryEntry
		}
		return true
	})

	if !armoriesInitialized {
		/*
			Armories are initialized on the first call to the armory command
			Armories 在第一次调用 armory 命令时初始化
			If armories are added or removed before the first call, we want
			If 军械库在第一次调用之前添加或删除，我们想要
			to make sure we still load in the ones from the configuration
			确保我们仍然从配置中加载
			file.
		*/
		persistentConfigs := assets.GetArmoriesConfig()
		for _, config := range persistentConfigs {
			if !slices.Contains(armoryNames, config.Name) {
				if config.Name == assets.DefaultArmoryName {
					if defaultArmoryRemoved {
						continue
					} else if defaultConfig != nil {
						// The user potentially changed something about the default config
						// The 用户可能更改了默认配置的某些内容
						configs = append(configs, defaultConfig)
						currentArmories.Store(config.PublicKey, *defaultConfig)
						continue
					}
				}
				configs = append(configs, config)
				currentArmories.Store(config.PublicKey, *config)
			}
		}
		return configs
	}
	if !defaultArmoryRemoved {
		if defaultConfig != nil {
			configs = append(configs, defaultConfig)
		} else {
			configs = append(configs, assets.DefaultArmoryConfig)
		}
	}
	assets.RefreshArmoryAuthorization(configs)
	return configs
}

func armoryNameExists(name string) bool {
	currentArmoriesConfigured := getCurrentArmoryConfiguration()
	for _, armory := range currentArmoriesConfigured {
		if armory.Name == name {
			return true
		}
	}

	return false
}

func armoryKeyExists(pubKey string) bool {
	currentArmoriesConfigured := getCurrentArmoryConfiguration()
	for _, armory := range currentArmoriesConfigured {
		if armory.PublicKey == pubKey {
			return true
		}
	}

	return false
}

func AddArmoryCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	// Get necessary information
	// Get必要信息
	url, _ := cmd.Flags().GetString("url")
	pubKey, _ := cmd.Flags().GetString("pubkey")
	auth, _ := cmd.Flags().GetString("auth")
	authCmd, _ := cmd.Flags().GetString("authcmd")
	noSave, _ := cmd.Flags().GetBool("no-save")

	name := args[0]

	if name == "" {
		con.PrintErrorf("an armory name is required\n")
		return
	}
	// Make sure we do not already have an armory config with the supplied name
	// Make 确保我们还没有具有所提供名称的 armory 配置
	if armoryNameExists(name) {
		con.PrintErrorf("armory %s already exists\n", name)
		return
	}

	clientConfig := parseArmoryHTTPConfig(cmd)
	armoryConfig := assets.ArmoryConfig{
		PublicKey:        pubKey,
		RepoURL:          url,
		Authorization:    auth,
		AuthorizationCmd: authCmd,
		Name:             name,
		Enabled:          true,
	}

	// verifyArmory will add the armory index to the cache on success, so we do not have to force a refresh
	// 成功后 verifyArmory 会将 armory 索引添加到缓存中，因此我们不必强制刷新
	err := verifyArmory(&armoryConfig, clientConfig)
	if err != nil {
		con.PrintErrorf("could not add armory: %s\n", err)
		return
	}
	currentArmories.Store(armoryConfig.PublicKey, armoryConfig)
	if !noSave {
		configs := getCurrentArmoryConfiguration()
		err = assets.SaveArmoriesConfig(configs)
		if err != nil {
			con.PrintErrorf("Could not save armory configuration: %s\n", err)
		} else {
			con.PrintSuccessf("Armory configuration saved\n")
		}
	}
	con.PrintSuccessf("Added armory %s\n", name)
}

// Get a list of package IDs for an armory by its public key
// Get 一个 armory 的包 IDs 的列表（按其公钥）
func getPackageIDsForArmory(armoryPublicKey string) []string {
	packageIDs := []string{}

	pkgCache.Range(func(key, value interface{}) bool {
		pkgEntry := value.(pkgCacheEntry)
		if pkgEntry.ArmoryConfig.PublicKey == armoryPublicKey {
			packageIDs = append(packageIDs, pkgEntry.ID)
		}
		return true
	})

	return packageIDs
}

func RemoveArmoryCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	name := args[0]

	if name == "" {
		con.PrintErrorf("an armory name is required\n")
		return
	}

	armories := getCurrentArmoryConfiguration()
	deleted := false
	newConfiguration := []*assets.ArmoryConfig{}
	for _, armory := range armories {
		if armory.Name != name {
			newConfiguration = append(newConfiguration, armory)
		} else {
			currentArmories.Delete(armory.PublicKey)
			indexCache.Delete(armory.PublicKey)
			/*
				To delete from the package cache, we will have
				To 从包缓存中删除，我们将有
				to gather a list of package IDs for the
				收集 IDs 包的列表
				armory then delete them from the cache.
				armory 然后从 cache. 中删除它们
			*/
			armoryPackageIDs := getPackageIDsForArmory(armory.PublicKey)
			for _, id := range armoryPackageIDs {
				pkgCache.Delete(id)
			}
			deleted = true
		}
	}

	if deleted {
		con.PrintSuccessf("Armory %s successfully deleted, saving configuration...\n", name)
		if name == assets.DefaultArmoryName {
			defaultArmoryRemoved = true
		}
	} else {
		con.PrintErrorf("Armory %s is not a configured armory\n", name)
		return
	}

	err := assets.SaveArmoriesConfig(newConfiguration)
	if err != nil {
		con.PrintErrorf("Could not save new armory configuration: %s\n", err)
	} else {
		con.PrintSuccessf("Saved armory configuration\n")
	}
}

func SaveArmories(con *console.SliverClient) {
	configs := getCurrentArmoryConfiguration()
	err := assets.SaveArmoriesConfig(configs)
	if err != nil {
		con.PrintErrorf("Could not save armory configuration: %s\n", err)
	} else {
		con.PrintSuccessf("Successfully saved armory configuration\n")
	}
}

func getArmoryConfig(name string) (assets.ArmoryConfig, error) {
	armories := getCurrentArmoryConfiguration()
	var armoryPublicKey string
	armoryConfig := assets.ArmoryConfig{}
	for _, armory := range armories {
		if armory.Name == name {
			armoryPublicKey = armory.PublicKey
			break
		}
	}

	if armoryPublicKey == "" {
		return armoryConfig, fmt.Errorf("could not retrieve armory configuration for armory %s. Try running the armory command to refresh indexes", name)
	}

	value, ok := currentArmories.Load(armoryPublicKey)
	if !ok {
		return armoryConfig, fmt.Errorf("could not retrieve armory configuration for armory %s. Try running the armory command to refresh indexes", name)
	}
	armoryConfig = value.(assets.ArmoryConfig)
	return armoryConfig, nil
}

func ChangeArmoryEnabledState(cmd *cobra.Command, con *console.SliverClient, args []string, enabled bool) {
	// Get the armory's public key, change its state, then update the cache to remove packages from the armory
	// Get armory 的公钥，更改其状态，然后更新缓存以从 armory 中删除包
	var name string
	if len(args) > 0 && args[0] != "" {
		name = args[0]
	} else {
		con.PrintErrorf("an armory name is required")
		return
	}

	armoryConfig, err := getArmoryConfig(name)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	armoryPublicKey := armoryConfig.PublicKey
	armoryConfig.Enabled = enabled
	currentArmories.Store(armoryPublicKey, armoryConfig)

	if !enabled {
		// Remove cached info for the armory if it is disabled
		// Remove 缓存 armory 的信息（如果已禁用）
		armoryPackageIDs := getPackageIDsForArmory(armoryPublicKey)
		for _, id := range armoryPackageIDs {
			pkgCache.Delete(id)
		}
		indexCache.Delete(armoryPublicKey)
	}
	// Force a refresh
	// Force 刷新
	clientConfig := parseArmoryHTTPConfig(cmd)
	con.PrintInfof("Refreshing armory information...\n")
	refresh(clientConfig)
	SaveArmories(con)
	con.PrintSuccessf("Done\n")
}

func ModifyArmoryCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	// Quick check to make sure we were provided with a name
	// Quick 检查以确保我们获得了姓名
	var name string
	if len(args) > 0 && args[0] != "" {
		name = args[0]
	}
	if name == "" {
		con.PrintErrorf("an armory name is required\n")
		return
	}

	// Get the record for the armory
	// Get armory 的记录
	armoryConfig, err := getArmoryConfig(name)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	// Save the old config so we can delete the existing config
	// Save 旧配置，以便我们可以删除现有配置
	// and revert if needed
	// 并在需要时恢复
	previousConfig := assets.ArmoryConfig{
		PublicKey:        armoryConfig.PublicKey,
		RepoURL:          armoryConfig.RepoURL,
		Authorization:    armoryConfig.Authorization,
		AuthorizationCmd: armoryConfig.AuthorizationCmd,
		Name:             armoryConfig.Name,
		Enabled:          armoryConfig.Enabled,
	}

	if cmd.Flags().Changed("name") {
		newName, _ := cmd.Flags().GetString("name")
		// Make sure the name does not already exist
		// Make 确保该名称尚不存在
		if armoryNameExists(newName) {
			con.PrintErrorf("armory with name %s already exists\n", name)
			return
		}
		armoryConfig.Name = newName
	}
	if cmd.Flags().Changed("pubkey") {
		newPubKey, _ := cmd.Flags().GetString("pubkey")
		/*
			Since public keys are used as the key for the index cache,
			Since 公钥用作索引缓存的密钥，
			make sure we have not been given a key that already exists
			确保我们没有获得已经存在的密钥
		*/
		if armoryKeyExists(newPubKey) {
			con.PrintErrorf("armory provided public key already exists\n")
			return
		}
		armoryConfig.PublicKey = newPubKey
	}
	if cmd.Flags().Changed("url") {
		newUrl, _ := cmd.Flags().GetString("url")
		armoryConfig.RepoURL = newUrl
	}
	if cmd.Flags().Changed("auth") {
		newAuth, _ := cmd.Flags().GetString("auth")
		armoryConfig.Authorization = newAuth
	}
	if cmd.Flags().Changed("authcmd") {
		newAuthCmd, _ := cmd.Flags().GetString("authcmd")
		armoryConfig.AuthorizationCmd = newAuthCmd
	}

	// We will remove the old armory and add a new one with the changed properties to force a refresh
	// We 将删除旧的 armory 并添加一个具有更改属性的新的 armory 以强制刷新
	currentArmories.Delete(previousConfig.PublicKey)
	indexCache.Delete(previousConfig.PublicKey)
	armoryPackageIDs := getPackageIDsForArmory(previousConfig.PublicKey)
	for _, id := range armoryPackageIDs {
		pkgCache.Delete(id)
	}
	clientConfig := parseArmoryHTTPConfig(cmd)
	err = verifyArmory(&armoryConfig, clientConfig)
	if err != nil {
		con.PrintErrorf("could not modify armory: %s\n", err)
		confirm := false
		_ = forms.Confirm("Would you like to revert the properties of the armory back to their previous values?", &confirm)
		if !confirm {
			return
		} else {
			err = verifyArmory(&previousConfig, clientConfig)
			if err != nil {
				con.PrintErrorf("could not re-add armory %s: %s\n", name, err)
				return
			}
			armoryConfig = previousConfig
			con.PrintSuccessf("Re-added armory with previous values")
		}
	}
	currentArmories.Store(armoryConfig.PublicKey, armoryConfig)
	if !cmd.Flags().Changed("no-save") {
		SaveArmories(con)
	}
	// Force a refresh
	// Force 刷新
	con.PrintInfof("Refreshing armory information...\n")
	refresh(clientConfig)
	con.PrintSuccessf("Done\n")
}

func RefreshArmories(cmd *cobra.Command, con *console.SliverClient) {
	clientConfig := parseArmoryHTTPConfig(cmd)
	// Since this being called from the refresh command, force the refresh
	// Since 这是从刷新命令调用的，强制刷新
	clientConfig.IgnoreCache = true
	clearAllCaches()
	armoriesInitialized = false
	con.PrintInfof("Refreshing armory information...")
	refresh(clientConfig)
	con.PrintSuccessf("Refreshed armory information\n")
}

func ResetArmoryConfig(cmd *cobra.Command, con *console.SliverClient) {
	armoryConfigPath := filepath.Join(assets.GetRootAppDir(), assets.ArmoryConfigFileName)
	con.PrintInfof("Removing armory configuration file %s...", armoryConfigPath)
	err := os.Remove(armoryConfigPath)
	if err != nil {
		con.PrintErrorf("Could not delete armory configuration file %s: %s", armoryConfigPath, err)
		return
	}
	con.PrintSuccessf("Removed armory configuration file %s\n", armoryConfigPath)

	// Force a refresh
	// Force 刷新
	RefreshArmories(cmd, con)
	con.PrintSuccessf("Successfully reset armory configuration\n")
}
