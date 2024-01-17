package registry

import (
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
)

// Commands returns the “ command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	registryCmd := &cobra.Command{
		Use:         consts.RegistryStr,
		Short:       "Windows registry operations",
		Long:        help.GetHelpFor([]string{consts.RegistryStr}),
		GroupID:     consts.InfoHelpGroup,
		Annotations: flags.RestrictTargets(consts.WindowsCmdsFilter),
	}
	flags.Bind("registry", true, registryCmd, func(f *pflag.FlagSet) {
		f.IntP("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})

	registryReadCmd := &cobra.Command{
		Use:   consts.RegistryReadStr,
		Short: "Read values from the Windows registry",
		Long:  help.GetHelpFor([]string{consts.RegistryReadStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			RegReadCmd(cmd, con, args)
		},
	}
	registryCmd.AddCommand(registryReadCmd)
	flags.Bind("", false, registryReadCmd, func(f *pflag.FlagSet) {
		f.StringP("hive", "H", "HKCU", "registry hive")
		f.StringP("hostname", "o", "", "remote host to read values from")
	})
	carapace.Gen(registryReadCmd).PositionalCompletion(carapace.ActionValues().Usage("registry path"))

	registryReadHiveCmd := &cobra.Command{
		Use:   consts.RegistryReadHiveStr,
		Short: "Read a hive into a binary file",
		Long:  help.GetHelpFor([]string{consts.RegistryReadStr + consts.RegistryReadHiveStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			RegReadHiveCommand(cmd, con, args)
		},
	}
	flags.Bind("", false, registryReadHiveCmd, func(f *pflag.FlagSet) {
		f.StringP("hive", "H", "HKLM", "root registry hive")
		f.StringP("save", "s", "", "location to store data, required if not looting")
		f.BoolP("loot", "X", false, "save output as loot (loot is saved without formatting)")
		f.StringP("name", "n", "", "name to assign loot (optional)")
		f.StringP("type", "T", "", "force a specific loot type (file/cred) if looting (optional)")
	})
	registryReadCmd.AddCommand(registryReadHiveCmd)

	registryWriteCmd := &cobra.Command{
		Use:   consts.RegistryWriteStr,
		Short: "Write values to the Windows registry",
		Long:  help.GetHelpFor([]string{consts.RegistryWriteStr}),
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			RegWriteCmd(cmd, con, args)
		},
	}
	registryCmd.AddCommand(registryWriteCmd)
	flags.Bind("", false, registryWriteCmd, func(f *pflag.FlagSet) {
		f.StringP("hive", "H", "HKCU", "registry hive")
		f.StringP("hostname", "o", "", "remote host to write values to")
		f.StringP("type", "T", "string", "type of the value to write (string, dword, qword, binary). If binary, you must provide a path to a file with --path")
		f.StringP("path", "p", "", "path to the binary file to write")
	})
	carapace.Gen(registryWriteCmd).PositionalCompletion(
		carapace.ActionValues().Usage("registry path"),
		carapace.ActionValues().Usage("value to write"),
	)

	registryCreateKeyCmd := &cobra.Command{
		Use:   consts.RegistryCreateKeyStr,
		Short: "Create a registry key",
		Long:  help.GetHelpFor([]string{consts.RegistryCreateKeyStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			RegCreateKeyCmd(cmd, con, args)
		},
	}
	registryCmd.AddCommand(registryCreateKeyCmd)
	flags.Bind("", false, registryCreateKeyCmd, func(f *pflag.FlagSet) {
		f.StringP("hive", "H", "HKCU", "registry hive")
		f.StringP("hostname", "o", "", "remote host to write values to")
	})
	carapace.Gen(registryCreateKeyCmd).PositionalCompletion(carapace.ActionValues().Usage("registry path"))

	registryDeleteKeyCmd := &cobra.Command{
		Use:   consts.RegistryDeleteKeyStr,
		Short: "Remove a registry key",
		Long:  help.GetHelpFor([]string{consts.RegistryDeleteKeyStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			RegDeleteKeyCmd(cmd, con, args)
		},
	}
	registryCmd.AddCommand(registryDeleteKeyCmd)
	flags.Bind("", false, registryDeleteKeyCmd, func(f *pflag.FlagSet) {
		f.StringP("hive", "H", "HKCU", "registry hive")
		f.StringP("hostname", "o", "", "remote host to remove value from")
	})
	carapace.Gen(registryDeleteKeyCmd).PositionalCompletion(carapace.ActionValues().Usage("registry path"))

	registryListSubCmd := &cobra.Command{
		Use:   consts.RegistryListSubStr,
		Short: "List the sub keys under a registry key",
		Long:  help.GetHelpFor([]string{consts.RegistryListSubStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			RegListSubKeysCmd(cmd, con, args)
		},
	}
	registryCmd.AddCommand(registryListSubCmd)
	flags.Bind("", false, registryListSubCmd, func(f *pflag.FlagSet) {
		f.StringP("hive", "H", "HKCU", "registry hive")
		f.StringP("hostname", "o", "", "remote host to write values to")
	})
	carapace.Gen(registryListSubCmd).PositionalCompletion(carapace.ActionValues().Usage("registry path"))

	registryListValuesCmd := &cobra.Command{
		Use:   consts.RegistryListValuesStr,
		Short: "List the values for a registry key",
		Long:  help.GetHelpFor([]string{consts.RegistryListValuesStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			RegListValuesCmd(cmd, con, args)
		},
	}
	registryCmd.AddCommand(registryListValuesCmd)
	flags.Bind("", false, registryListValuesCmd, func(f *pflag.FlagSet) {
		f.StringP("hive", "H", "HKCU", "registry hive")
		f.StringP("hostname", "o", "", "remote host to write values to")
	})
	carapace.Gen(registryListValuesCmd).PositionalCompletion(carapace.ActionValues().Usage("registry path"))

	return []*cobra.Command{registryCmd}
}
