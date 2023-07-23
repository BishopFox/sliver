package wasm

import (
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Commands returns the “ command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	wasmCmd := &cobra.Command{
		Use:     consts.WasmStr,
		Short:   "Execute a Wasm Module Extension",
		Long:    help.GetHelpFor([]string{consts.WasmStr}),
		GroupID: consts.ExecutionHelpGroup,
		Run: func(cmd *cobra.Command, args []string) {
			WasmCmd(cmd, con, args)
		},
	}
	flags.Bind("", true, wasmCmd, func(f *pflag.FlagSet) {
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	flags.Bind("", false, wasmCmd, func(f *pflag.FlagSet) {
		f.BoolP("pipe", "P", false, "pipe module stdin/stdout/stderr to the current terminal (session only)")
		f.StringP("file", "f", "", "include local file(s) in wasm module's /memfs (glob pattern) ")
		f.StringP("dir", "d", "", "recursively include local directory in wasm module's /memfs (glob pattern)")
		f.BoolP("skip-registration", "s", false, "assume the extension is already registered")
		f.BoolP("loot", "X", false, "save output as loot, incompatible with --pipe")
	})
	flags.BindFlagCompletions(wasmCmd, func(comp *carapace.ActionMap) {
		(*comp)["file"] = carapace.ActionFiles()
		(*comp)["dir"] = carapace.ActionDirectories()
	})
	wasmComp := carapace.Gen(wasmCmd)
	wasmComp.PositionalCompletion(carapace.ActionFiles().Usage("wasm/wasi module file (.wasm)"))
	wasmComp.PositionalAnyCompletion(carapace.ActionValues().Usage("arguments to pass to the wasm module (optional)"))

	wasmLsCmd := &cobra.Command{
		Use:   consts.LsStr,
		Short: "List registered wasm extensions with current session/beacon",
		Long:  help.GetHelpFor([]string{consts.WasmStr, consts.LsStr}),
		Run: func(cmd *cobra.Command, args []string) {
			WasmLsCmd(cmd, con, args)
		},
	}
	wasmCmd.AddCommand(wasmLsCmd)

	return []*cobra.Command{wasmCmd}
}
