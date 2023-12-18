package command

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox

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
	"fmt"

	"github.com/reeflective/console"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/command/alias"
	"github.com/bishopfox/sliver/client/command/backdoor"
	"github.com/bishopfox/sliver/client/command/completers"
	"github.com/bishopfox/sliver/client/command/cursed"
	"github.com/bishopfox/sliver/client/command/dllhijack"
	"github.com/bishopfox/sliver/client/command/environment"
	"github.com/bishopfox/sliver/client/command/exec"
	"github.com/bishopfox/sliver/client/command/extensions"
	"github.com/bishopfox/sliver/client/command/filesystem"
	"github.com/bishopfox/sliver/client/command/generate"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/command/info"
	"github.com/bishopfox/sliver/client/command/kill"
	"github.com/bishopfox/sliver/client/command/network"
	"github.com/bishopfox/sliver/client/command/pivots"
	"github.com/bishopfox/sliver/client/command/portfwd"
	"github.com/bishopfox/sliver/client/command/privilege"
	"github.com/bishopfox/sliver/client/command/processes"
	"github.com/bishopfox/sliver/client/command/reconfig"
	"github.com/bishopfox/sliver/client/command/registry"
	"github.com/bishopfox/sliver/client/command/rportfwd"
	"github.com/bishopfox/sliver/client/command/screenshot"
	"github.com/bishopfox/sliver/client/command/sessions"
	"github.com/bishopfox/sliver/client/command/shell"
	"github.com/bishopfox/sliver/client/command/socks"
	"github.com/bishopfox/sliver/client/command/tasks"
	"github.com/bishopfox/sliver/client/command/use"
	"github.com/bishopfox/sliver/client/command/wasm"
	"github.com/bishopfox/sliver/client/command/wireguard"
	client "github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
)

// SliverCommands returns all commands bound to the implant menu.
func SliverCommands(con *client.SliverConsoleClient) console.Commands {
	sliverCommands := func() *cobra.Command {
		sliver := &cobra.Command{
			Short: "Implant commands",
			CompletionOptions: cobra.CompletionOptions{
				HiddenDefaultCmd: true,
			},
		}

		groups := []*cobra.Group{
			{ID: consts.SliverCoreHelpGroup, Title: consts.SliverCoreHelpGroup},
			{ID: consts.InfoHelpGroup, Title: consts.InfoHelpGroup},
			{ID: consts.FilesystemHelpGroup, Title: consts.FilesystemHelpGroup},
			{ID: consts.NetworkHelpGroup, Title: consts.NetworkHelpGroup},
			{ID: consts.ExecutionHelpGroup, Title: consts.ExecutionHelpGroup},
			{ID: consts.PrivilegesHelpGroup, Title: consts.PrivilegesHelpGroup},
			{ID: consts.ProcessHelpGroup, Title: consts.ProcessHelpGroup},
			{ID: consts.AliasHelpGroup, Title: consts.AliasHelpGroup},
			{ID: consts.ExtensionHelpGroup, Title: consts.ExtensionHelpGroup},
		}
		sliver.AddGroup(groups...)

		// Load Aliases
		aliasManifests := assets.GetInstalledAliasManifests()
		for _, manifest := range aliasManifests {
			_, err := alias.LoadAlias(manifest, sliver, con)
			if err != nil {
				con.PrintErrorf("Failed to load alias: %s", err)
				continue
			}
		}

		// Load Extensions
		extensionManifests := assets.GetInstalledExtensionManifests()
		for _, manifest := range extensionManifests {
			mext, err := extensions.LoadExtensionManifest(manifest)
			// Absorb error in case there's no extensions manifest
			if err != nil {
				//con doesn't appear to be initialised here?
				//con.PrintErrorf("Failed to load extension: %s", err)
				fmt.Printf("Failed to load extension: %s\n", err)
				continue
			}
			for _, ext := range mext.ExtCommand {
				extensions.ExtensionRegisterCommand(ext, sliver, con)
			}
		}

		// [ Reconfig ] ---------------------------------------------------------------

		reconfigCmd := &cobra.Command{
			Use:   consts.ReconfigStr,
			Short: "Reconfigure the active beacon/session",
			Long:  help.GetHelpFor([]string{consts.ReconfigStr}),
			Run: func(cmd *cobra.Command, args []string) {
				reconfig.ReconfigCmd(cmd, con, args)
			},
			GroupID:     consts.SliverCoreHelpGroup,
			Annotations: hideCommand(consts.BeaconCmdsFilter),
		}
		sliver.AddCommand(reconfigCmd)
		Flags("reconfig", false, reconfigCmd, func(f *pflag.FlagSet) {
			f.StringP("reconnect-interval", "r", "", "reconnect interval for implant")
			f.StringP("beacon-interval", "i", "", "beacon callback interval")
			f.StringP("beacon-jitter", "j", "", "beacon callback jitter (random up to)")
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		renameCmd := &cobra.Command{
			Use:   consts.RenameStr,
			Short: "Rename the active beacon/session",
			Long:  help.GetHelpFor([]string{consts.RenameStr}),
			Run: func(cmd *cobra.Command, args []string) {
				reconfig.RenameCmd(cmd, con, args)
			},
			GroupID: consts.SliverCoreHelpGroup,
		}
		sliver.AddCommand(renameCmd)
		Flags("rename", false, renameCmd, func(f *pflag.FlagSet) {
			f.StringP("name", "n", "", "change implant name to")
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		// [ Sessions ] --------------------------------------------------------------

		sessionsCmd := &cobra.Command{
			Use:   consts.SessionsStr,
			Short: "Session management",
			Long:  help.GetHelpFor([]string{consts.SessionsStr}),
			Run: func(cmd *cobra.Command, args []string) {
				sessions.SessionsCmd(cmd, con, args)
			},
			GroupID: consts.SliverCoreHelpGroup,
		}
		Flags("sessions", true, sessionsCmd, func(f *pflag.FlagSet) {
			f.IntP("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		Flags("sessions", false, sessionsCmd, func(f *pflag.FlagSet) {
			f.StringP("interact", "i", "", "interact with a session")
			f.StringP("kill", "k", "", "kill the designated session")
			f.BoolP("kill-all", "K", false, "kill all the sessions")
			f.BoolP("clean", "C", false, "clean out any sessions marked as [DEAD]")
			f.BoolP("force", "F", false, "force session action without waiting for results")

			f.StringP("filter", "f", "", "filter sessions by substring")
			f.StringP("filter-re", "e", "", "filter sessions by regular expression")
		})
		FlagComps(sessionsCmd, func(comp *carapace.ActionMap) {
			(*comp)["interact"] = use.BeaconAndSessionIDCompleter(con)
			(*comp)["kill"] = use.BeaconAndSessionIDCompleter(con)
		})
		sliver.AddCommand(sessionsCmd)

		sessionsPruneCmd := &cobra.Command{
			Use:   consts.PruneStr,
			Short: "Kill all stale/dead sessions",
			Long:  help.GetHelpFor([]string{consts.SessionsStr, consts.PruneStr}),
			Run: func(cmd *cobra.Command, args []string) {
				sessions.SessionsPruneCmd(cmd, con, args)
			},
		}
		Flags("prune", false, sessionsPruneCmd, func(f *pflag.FlagSet) {
			f.BoolP("force", "F", false, "Force the killing of stale/dead sessions")
		})
		sessionsCmd.AddCommand(sessionsPruneCmd)

		backgroundCmd := &cobra.Command{
			Use:   consts.BackgroundStr,
			Short: "Background an active session",
			Long:  help.GetHelpFor([]string{consts.BackgroundStr}),
			Run: func(cmd *cobra.Command, args []string) {
				sessions.BackgroundCmd(cmd, con, args)
			},
			GroupID: consts.SliverCoreHelpGroup,
		}
		Flags("use", false, backgroundCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		sliver.AddCommand(backgroundCmd)

		killCmd := &cobra.Command{
			Use:   consts.KillStr,
			Short: "Kill a session",
			Long:  help.GetHelpFor([]string{consts.KillStr}),
			Run: func(cmd *cobra.Command, args []string) {
				kill.KillCmd(cmd, con, args)
			},
			GroupID: consts.SliverCoreHelpGroup,
		}
		sliver.AddCommand(killCmd)
		Flags("use", false, backgroundCmd, func(f *pflag.FlagSet) {
			f.BoolP("force", "F", false, "Force kill,  does not clean up")
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		openSessionCmd := &cobra.Command{
			Use:   consts.InteractiveStr,
			Short: "Task a beacon to open an interactive session (Beacon only)",
			Long:  help.GetHelpFor([]string{consts.InteractiveStr}),
			Run: func(cmd *cobra.Command, args []string) {
				sessions.InteractiveCmd(cmd, con, args)
			},
			GroupID:     consts.SliverCoreHelpGroup,
			Annotations: hideCommand(consts.BeaconCmdsFilter),
		}
		sliver.AddCommand(openSessionCmd)
		Flags("interactive", false, openSessionCmd, func(f *pflag.FlagSet) {
			f.StringP("mtls", "m", "", "mtls connection strings")
			f.StringP("wg", "g", "", "wg connection strings")
			f.StringP("http", "b", "", "http(s) connection strings")
			f.StringP("dns", "n", "", "dns connection strings")
			f.StringP("named-pipe", "p", "", "namedpipe connection strings")
			f.StringP("tcp-pivot", "i", "", "tcppivot connection strings")

			f.StringP("delay", "d", "0s", "delay opening the session (after checkin) for a given period of time")

			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		// [ Use ] --------------------------------------------------------------

		useCmd := &cobra.Command{
			Use:   consts.UseStr,
			Short: "Switch the active session or beacon",
			Long:  help.GetHelpFor([]string{consts.UseStr}),
			Run: func(cmd *cobra.Command, args []string) {
				use.UseCmd(cmd, con, args)
			},
			GroupID: consts.SliverCoreHelpGroup,
		}
		Flags("use", true, useCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		carapace.Gen(useCmd).PositionalCompletion(use.BeaconAndSessionIDCompleter(con))

		if !con.IsCLI {
			sliver.AddCommand(useCmd)
		}

		useSessionCmd := &cobra.Command{
			Use:   consts.SessionsStr,
			Short: "Switch the active session",
			Long:  help.GetHelpFor([]string{consts.UseStr, consts.SessionsStr}),
			Run: func(cmd *cobra.Command, args []string) {
				use.UseSessionCmd(cmd, con, args)
			},
		}
		carapace.Gen(useSessionCmd).PositionalCompletion(use.SessionIDCompleter(con))
		useCmd.AddCommand(useSessionCmd)

		useBeaconCmd := &cobra.Command{
			Use:   consts.BeaconsStr,
			Short: "Switch the active beacon",
			Long:  help.GetHelpFor([]string{consts.UseStr, consts.BeaconsStr}),
			Run: func(cmd *cobra.Command, args []string) {
				use.UseBeaconCmd(cmd, con, args)
			},
		}
		carapace.Gen(useBeaconCmd).PositionalCompletion(use.BeaconIDCompleter(con))
		useCmd.AddCommand(useBeaconCmd)

		// [ Close ] --------------------------------------------------------------
		closeSessionCmd := &cobra.Command{
			Use:   consts.CloseStr,
			Short: "Close an interactive session without killing the remote process",
			Long:  help.GetHelpFor([]string{consts.CloseStr}),
			Run: func(cmd *cobra.Command, args []string) {
				sessions.CloseSessionCmd(cmd, con, args)
			},
			GroupID: consts.SliverCoreHelpGroup,
		}
		sliver.AddCommand(closeSessionCmd)
		Flags("", false, closeSessionCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		// [ Tasks ] --------------------------------------------------------------

		tasksCmd := &cobra.Command{
			Use:   consts.TasksStr,
			Short: "Beacon task management",
			Long:  help.GetHelpFor([]string{consts.TasksStr}),
			Run: func(cmd *cobra.Command, args []string) {
				tasks.TasksCmd(cmd, con, args)
			},
			GroupID:     consts.SliverCoreHelpGroup,
			Annotations: hideCommand(consts.BeaconCmdsFilter),
		}
		Flags("tasks", true, tasksCmd, func(f *pflag.FlagSet) {
			f.IntP("timeout", "t", defaultTimeout, "grpc timeout in seconds")
			f.BoolP("overflow", "O", false, "overflow terminal width (display truncated rows)")
			f.IntP("skip-pages", "S", 0, "skip the first n page(s)")
			f.StringP("filter", "f", "", "filter based on task type (case-insensitive prefix matching)")
		})
		sliver.AddCommand(tasksCmd)

		fetchCmd := &cobra.Command{
			Use:   consts.FetchStr,
			Short: "Fetch the details of a beacon task",
			Long:  help.GetHelpFor([]string{consts.TasksStr, consts.FetchStr}),
			Args:  cobra.RangeArgs(0, 1),
			Run: func(cmd *cobra.Command, args []string) {
				tasks.TasksFetchCmd(cmd, con, args)
			},
		}
		tasksCmd.AddCommand(fetchCmd)
		carapace.Gen(fetchCmd).PositionalCompletion(tasks.BeaconTaskIDCompleter(con).Usage("beacon task ID"))

		cancelCmd := &cobra.Command{
			Use:   consts.CancelStr,
			Short: "Cancel a pending beacon task",
			Long:  help.GetHelpFor([]string{consts.TasksStr, consts.CancelStr}),
			Args:  cobra.RangeArgs(0, 1),
			Run: func(cmd *cobra.Command, args []string) {
				tasks.TasksCancelCmd(cmd, con, args)
			},
		}
		tasksCmd.AddCommand(cancelCmd)
		carapace.Gen(cancelCmd).PositionalCompletion(tasks.BeaconPendingTasksCompleter(con).Usage("beacon task ID"))

		// [ Info ] --------------------------------------------------------------

		infoCmd := &cobra.Command{
			Use:   consts.InfoStr,
			Short: "Get info about session",
			Long:  help.GetHelpFor([]string{consts.InfoStr}),
			Run: func(cmd *cobra.Command, args []string) {
				info.InfoCmd(cmd, con, args)
			},
			GroupID: consts.InfoHelpGroup,
		}
		Flags("use", false, infoCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		carapace.Gen(infoCmd).PositionalCompletion(use.BeaconAndSessionIDCompleter(con))
		sliver.AddCommand(infoCmd)

		pingCmd := &cobra.Command{
			Use:   consts.PingStr,
			Short: "Send round trip message to implant (does not use ICMP)",
			Long:  help.GetHelpFor([]string{consts.PingStr}),
			Run: func(cmd *cobra.Command, args []string) {
				info.PingCmd(cmd, con, args)
			},
			GroupID: consts.InfoHelpGroup,
		}
		sliver.AddCommand(pingCmd)
		Flags("", false, pingCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		getPIDCmd := &cobra.Command{
			Use:   consts.GetPIDStr,
			Short: "Get session pid",
			Long:  help.GetHelpFor([]string{consts.GetPIDStr}),
			Run: func(cmd *cobra.Command, args []string) {
				info.PIDCmd(cmd, con, args)
			},
			GroupID: consts.InfoHelpGroup,
		}
		sliver.AddCommand(getPIDCmd)
		Flags("", false, getPIDCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		getUIDCmd := &cobra.Command{
			Use:   consts.GetUIDStr,
			Short: "Get session process UID",
			Long:  help.GetHelpFor([]string{consts.GetUIDStr}),
			Run: func(cmd *cobra.Command, args []string) {
				info.UIDCmd(cmd, con, args)
			},
			GroupID: consts.InfoHelpGroup,
		}
		sliver.AddCommand(getUIDCmd)
		Flags("", false, getUIDCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		getGIDCmd := &cobra.Command{
			Use:   consts.GetGIDStr,
			Short: "Get session process GID",
			Long:  help.GetHelpFor([]string{consts.GetGIDStr}),
			Run: func(cmd *cobra.Command, args []string) {
				info.GIDCmd(cmd, con, args)
			},
			GroupID: consts.InfoHelpGroup,
		}
		sliver.AddCommand(getGIDCmd)
		Flags("", false, getGIDCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		whoamiCmd := &cobra.Command{
			Use:   consts.WhoamiStr,
			Short: "Get session user execution context",
			Long:  help.GetHelpFor([]string{consts.WhoamiStr}),
			Run: func(cmd *cobra.Command, args []string) {
				info.WhoamiCmd(cmd, con, args)
			},
			GroupID: consts.InfoHelpGroup,
		}
		sliver.AddCommand(whoamiCmd)
		Flags("", false, whoamiCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		// [ Shell ] --------------------------------------------------------------

		shellCmd := &cobra.Command{
			Use:   consts.ShellStr,
			Short: "Start an interactive shell",
			Long:  help.GetHelpFor([]string{consts.ShellStr}),
			Run: func(cmd *cobra.Command, args []string) {
				shell.ShellCmd(cmd, con, args)
			},
			GroupID:     consts.ExecutionHelpGroup,
			Annotations: hideCommand(consts.SessionCmdsFilter),
		}
		sliver.AddCommand(shellCmd)
		Flags("", false, shellCmd, func(f *pflag.FlagSet) {
			f.BoolP("no-pty", "y", false, "disable use of pty on macos/linux")
			f.StringP("shell-path", "s", "", "path to shell interpreter")

			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		// [ Exec ] --------------------------------------------------------------

		executeCmd := &cobra.Command{
			Use:   consts.ExecuteStr,
			Short: "Execute a program on the remote system",
			Long:  help.GetHelpFor([]string{consts.ExecuteStr}),
			Args:  cobra.MinimumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				exec.ExecuteCmd(cmd, con, args)
			},
			GroupID: consts.ExecutionHelpGroup,
		}
		sliver.AddCommand(executeCmd)
		Flags("", false, executeCmd, func(f *pflag.FlagSet) {
			f.BoolP("token", "T", false, "execute command with current token (Windows only)")
			f.BoolP("output", "o", false, "capture command output")
			f.BoolP("save", "s", false, "save output to a file")
			f.BoolP("loot", "X", false, "save output as loot")
			f.BoolP("ignore-stderr", "S", false, "don't print STDERR output")
			f.StringP("stdout", "O", "", "remote path to redirect STDOUT to")
			f.StringP("stderr", "E", "", "remote path to redirect STDERR to")
			f.StringP("name", "n", "", "name to assign loot (optional)")
			f.Uint32P("ppid", "P", 0, "parent process id (optional, Windows only)")
			f.BoolP("hidden", "H", false, "hide the window of the spawned process (Windows only)")

			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		executeCmd.Flags().ParseErrorsWhitelist.UnknownFlags = true

		carapace.Gen(executeCmd).PositionalCompletion(carapace.ActionValues().Usage("command to execute (required)"))
		carapace.Gen(executeCmd).PositionalAnyCompletion(carapace.ActionValues().Usage("arguments to the command (optional)"))

		executeAssemblyCmd := &cobra.Command{
			Use:   consts.ExecuteAssemblyStr,
			Short: "Loads and executes a .NET assembly in a child process (Windows Only)",
			Long:  help.GetHelpFor([]string{consts.ExecuteAssemblyStr}),
			Args:  cobra.MinimumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				exec.ExecuteAssemblyCmd(cmd, con, args)
			},
			GroupID:     consts.ExecutionHelpGroup,
			Annotations: hideCommand(consts.WindowsCmdsFilter),
		}
		sliver.AddCommand(executeAssemblyCmd)
		Flags("", false, executeAssemblyCmd, func(f *pflag.FlagSet) {
			f.StringP("process", "p", "notepad.exe", "hosting process to inject into")
			f.StringP("method", "m", "", "Optional method (a method is required for a .NET DLL)")
			f.StringP("class", "c", "", "Optional class name (required for .NET DLL)")
			f.StringP("app-domain", "d", "", "AppDomain name to create for .NET assembly. Generated randomly if not set.")
			f.StringP("arch", "a", "x84", "Assembly target architecture: x86, x64, x84 (x86+x64)")
			f.BoolP("in-process", "i", false, "Run in the current sliver process")
			f.StringP("runtime", "r", "", "Runtime to use for running the assembly (only supported when used with --in-process)")
			f.BoolP("save", "s", false, "save output to file")
			f.BoolP("loot", "X", false, "save output as loot")
			f.StringP("name", "n", "", "name to assign loot (optional)")
			f.Uint32P("ppid", "P", 0, "parent process id (optional)")
			f.StringP("process-arguments", "A", "", "arguments to pass to the hosting process")
			f.BoolP("amsi-bypass", "M", false, "Bypass AMSI on Windows (only supported when used with --in-process)")
			f.BoolP("etw-bypass", "E", false, "Bypass ETW on Windows (only supported when used with --in-process)")

			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		executeAssemblyCmd.Flags().ParseErrorsWhitelist.UnknownFlags = true

		carapace.Gen(executeAssemblyCmd).PositionalCompletion(carapace.ActionFiles().Usage("path to assembly file (required)"))
		carapace.Gen(executeAssemblyCmd).PositionalAnyCompletion(carapace.ActionValues().Usage("arguments to pass to the assembly entrypoint (optional)"))

		executeShellcodeCmd := &cobra.Command{
			Use:   consts.ExecuteShellcodeStr,
			Short: "Executes the given shellcode in the sliver process",
			Long:  help.GetHelpFor([]string{consts.ExecuteShellcodeStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				exec.ExecuteShellcodeCmd(cmd, con, args)
			},
			GroupID: consts.ExecutionHelpGroup,
		}
		sliver.AddCommand(executeShellcodeCmd)
		Flags("", false, executeShellcodeCmd, func(f *pflag.FlagSet) {
			f.BoolP("rwx-pages", "r", false, "Use RWX permissions for memory pages")
			f.Uint32P("pid", "p", 0, "Pid of process to inject into (0 means injection into ourselves)")
			f.StringP("process", "n", `c:\windows\system32\notepad.exe`, "Process to inject into when running in interactive mode")
			f.BoolP("interactive", "i", false, "Inject into a new process and interact with it")
			f.BoolP("shikata-ga-nai", "S", false, "encode shellcode using shikata ga nai prior to execution")
			f.StringP("architecture", "A", "amd64", "architecture of the shellcode: 386, amd64 (used with --shikata-ga-nai flag)")
			f.Uint32P("iterations", "I", 1, "number of encoding iterations (used with --shikata-ga-nai flag)")

			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		FlagComps(executeShellcodeCmd, func(comp *carapace.ActionMap) {
			(*comp)["shikata-ga-nai"] = carapace.ActionValues("386", "amd64").Tag("shikata-ga-nai architectures")
		})
		carapace.Gen(executeShellcodeCmd).PositionalCompletion(carapace.ActionFiles().Usage("path to shellcode file (required)"))

		sideloadCmd := &cobra.Command{
			Use:   consts.SideloadStr,
			Short: "Load and execute a shared object (shared library/DLL) in a remote process",
			Long:  help.GetHelpFor([]string{consts.SideloadStr}),
			Args:  cobra.MinimumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				exec.SideloadCmd(cmd, con, args)
			},
			GroupID: consts.ExecutionHelpGroup,
		}
		sliver.AddCommand(sideloadCmd)
		Flags("", false, sideloadCmd, func(f *pflag.FlagSet) {
			f.StringP("entry-point", "e", "", "Entrypoint for the DLL (Windows only)")
			f.StringP("process", "p", `c:\windows\system32\notepad.exe`, "Path to process to host the shellcode")
			f.BoolP("unicode", "w", false, "Command line is passed to unmanaged DLL function in UNICODE format. (default is ANSI)")
			f.BoolP("save", "s", false, "save output to file")
			f.BoolP("loot", "X", false, "save output as loot")
			f.StringP("name", "n", "", "name to assign loot (optional)")
			f.BoolP("keep-alive", "k", false, "don't terminate host process once the execution completes")
			f.Uint32P("ppid", "P", 0, "parent process id (optional)")
			f.StringP("process-arguments", "A", "", "arguments to pass to the hosting process")

			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		sideloadCmd.Flags().ParseErrorsWhitelist.UnknownFlags = true

		carapace.Gen(sideloadCmd).PositionalCompletion(carapace.ActionFiles().Usage("path to shared library file (required)"))
		carapace.Gen(sideloadCmd).PositionalAnyCompletion(carapace.ActionValues().Usage("arguments to pass to the binary (optional)"))

		spawnDllCmd := &cobra.Command{
			Use:   consts.SpawnDllStr,
			Short: "Load and execute a Reflective DLL in a remote process",
			Long:  help.GetHelpFor([]string{consts.SpawnDllStr}),
			Args:  cobra.MinimumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				exec.SpawnDllCmd(cmd, con, args)
			},
			GroupID:     consts.ExecutionHelpGroup,
			Annotations: hideCommand(consts.WindowsCmdsFilter),
		}
		sliver.AddCommand(spawnDllCmd)
		Flags("", false, spawnDllCmd, func(f *pflag.FlagSet) {
			f.StringP("process", "p", `c:\windows\system32\notepad.exe`, "Path to process to host the shellcode")
			f.StringP("export", "e", "ReflectiveLoader", "Entrypoint of the Reflective DLL")
			f.BoolP("save", "s", false, "save output to file")
			f.BoolP("loot", "X", false, "save output as loot")
			f.StringP("name", "n", "", "name to assign loot (optional)")
			f.BoolP("keep-alive", "k", false, "don't terminate host process once the execution completes")
			f.UintP("ppid", "P", 0, "parent process id (optional)")
			f.StringP("process-arguments", "A", "", "arguments to pass to the hosting process")

			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		spawnDllCmd.Flags().ParseErrorsWhitelist.UnknownFlags = true

		carapace.Gen(spawnDllCmd).PositionalCompletion(carapace.ActionFiles().Usage("path to DLL file (required)"))
		carapace.Gen(spawnDllCmd).PositionalAnyCompletion(carapace.ActionValues().Usage("arguments to pass to the DLL entrypoint (optional)"))

		migrateCmd := &cobra.Command{
			Use:   consts.MigrateStr,
			Short: "Migrate into a remote process",
			Long:  help.GetHelpFor([]string{consts.MigrateStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				exec.MigrateCmd(cmd, con, args)
			},
			GroupID:     consts.ExecutionHelpGroup,
			Annotations: hideCommand(consts.WindowsCmdsFilter),
		}
		sliver.AddCommand(migrateCmd)
		Flags("", false, migrateCmd, func(f *pflag.FlagSet) {
			f.BoolP("disable-sgn", "S", true, "disable shikata ga nai shellcode encoder")
			f.Uint32P("pid", "p", 0, "process id to migrate into")
			f.StringP("process-name", "n", "", "name of the process to migrate into")
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		carapace.Gen(migrateCmd).PositionalCompletion(carapace.ActionValues().Usage("PID of process to migrate into"))

		msfCmd := &cobra.Command{
			Use:   consts.MsfStr,
			Short: "Execute an MSF payload in the current process",
			Long:  help.GetHelpFor([]string{consts.MsfStr}),
			Run: func(cmd *cobra.Command, args []string) {
				exec.MsfCmd(cmd, con, args)
			},
			GroupID: consts.ExecutionHelpGroup,
		}
		sliver.AddCommand(msfCmd)
		Flags("", false, msfCmd, func(f *pflag.FlagSet) {
			f.StringP("payload", "m", "meterpreter_reverse_https", "msf payload")
			f.StringP("lhost", "L", "", "listen host")
			f.IntP("lport", "l", 4444, "listen port")
			f.StringP("encoder", "e", "", "msf encoder")
			f.IntP("iterations", "i", 1, "iterations of the encoder")

			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		msfInjectCmd := &cobra.Command{
			Use:   consts.MsfInjectStr,
			Short: "Inject an MSF payload into a process",
			Long:  help.GetHelpFor([]string{consts.MsfInjectStr}),
			Run: func(cmd *cobra.Command, args []string) {
				exec.MsfInjectCmd(cmd, con, args)
			},
			GroupID: consts.ExecutionHelpGroup,
		}
		sliver.AddCommand(msfInjectCmd)
		Flags("", false, msfInjectCmd, func(f *pflag.FlagSet) {
			f.IntP("pid", "p", -1, "pid to inject into")
			f.StringP("payload", "m", "meterpreter_reverse_https", "msf payload")
			f.StringP("lhost", "L", "", "listen host")
			f.IntP("lport", "l", 4444, "listen port")
			f.StringP("encoder", "e", "", "msf encoder")
			f.IntP("iterations", "i", 1, "iterations of the encoder")

			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		psExecCmd := &cobra.Command{
			Use:   consts.PsExecStr,
			Short: "Start a sliver service on a remote target",
			Long:  help.GetHelpFor([]string{consts.PsExecStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				exec.PsExecCmd(cmd, con, args)
			},
			GroupID:     consts.ExecutionHelpGroup,
			Annotations: hideCommand(consts.WindowsCmdsFilter),
		}
		sliver.AddCommand(psExecCmd)
		Flags("", false, psExecCmd, func(f *pflag.FlagSet) {
			f.StringP("service-name", "s", "Sliver", "name that will be used to register the service")
			f.StringP("service-description", "d", "Sliver implant", "description of the service")
			f.StringP("profile", "p", "", "profile to use for service binary")
			f.StringP("binpath", "b", "c:\\windows\\temp", "directory to which the executable will be uploaded")
			f.StringP("custom-exe", "c", "", "custom service executable to use instead of generating a new Sliver")

			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		FlagComps(psExecCmd, func(comp *carapace.ActionMap) {
			(*comp)["custom-exe"] = carapace.ActionFiles()
		})
		carapace.Gen(psExecCmd).PositionalCompletion(carapace.ActionValues().Usage("hostname (required)"))

		sshCmd := &cobra.Command{
			Use:   consts.SSHStr,
			Short: "Run a SSH command on a remote host",
			Long:  help.GetHelpFor([]string{consts.SSHStr}),
			Args:  cobra.MinimumNArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				exec.SSHCmd(cmd, con, args)
			},
			GroupID: consts.ExecutionHelpGroup,
		}
		sliver.AddCommand(sshCmd)
		Flags("", false, sshCmd, func(f *pflag.FlagSet) {
			f.UintP("port", "p", 22, "SSH port")
			f.StringP("private-key", "i", "", "path to private key file")
			f.StringP("password", "P", "", "SSH user password")
			f.StringP("login", "l", "", "username to use to connect")
			f.BoolP("skip-loot", "s", false, "skip the prompt to use loot credentials")
			f.StringP("kerberos-config", "c", "/etc/krb5.conf", "path to remote Kerberos config file")
			f.StringP("kerberos-keytab", "k", "", "path to Kerberos keytab file")
			f.StringP("kerberos-realm", "r", "", "Kerberos realm")

			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		sshCmd.Flags().ParseErrorsWhitelist.UnknownFlags = true

		FlagComps(sshCmd, func(comp *carapace.ActionMap) {
			(*comp)["private-key"] = carapace.ActionFiles()
			(*comp)["kerberos-keytab"] = carapace.ActionFiles()
		})

		carapace.Gen(sshCmd).PositionalCompletion(carapace.ActionValues().Usage("remote host to SSH to (required)"))
		carapace.Gen(sshCmd).PositionalAnyCompletion(carapace.ActionValues().Usage("command line with arguments"))

		// [ Extensions ] -----------------------------------------------------------------
		extensionCmd := &cobra.Command{
			Use:     consts.ExtensionsStr,
			Short:   "Manage extensions",
			Long:    help.GetHelpFor([]string{consts.ExtensionsStr}),
			GroupID: consts.ExtensionHelpGroup,
			Run: func(cmd *cobra.Command, _ []string) {
				extensions.ExtensionsCmd(cmd, con)
			},
		}
		sliver.AddCommand(extensionCmd)

		extensionCmd.AddCommand(&cobra.Command{
			Use:   consts.ListStr,
			Short: "List extensions loaded in the current session or beacon",
			Long:  help.GetHelpFor([]string{consts.ExtensionsStr, consts.ListStr}),
			Run: func(cmd *cobra.Command, args []string) {
				extensions.ExtensionsListCmd(cmd, con, args)
			},
		})

		extensionLoadCmd := &cobra.Command{
			Use:   consts.LoadStr,
			Short: "Temporarily load an extension from a local directory",
			Long:  help.GetHelpFor([]string{consts.ExtensionsStr, consts.LoadStr}),
			Run: func(cmd *cobra.Command, args []string) {
				extensions.ExtensionLoadCmd(cmd, con, args)
			},
		}
		extensionCmd.AddCommand(extensionLoadCmd)
		carapace.Gen(extensionLoadCmd).PositionalCompletion(carapace.ActionDirectories().Usage("path to the extension directory"))

		extensionInstallCmd := &cobra.Command{
			Use:   consts.InstallStr,
			Short: "Install an extension from a local directory or .tar.gz file",
			Long:  help.GetHelpFor([]string{consts.ExtensionsStr, consts.InstallStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				extensions.ExtensionsInstallCmd(cmd, con, args)
			},
		}
		extensionCmd.AddCommand(extensionInstallCmd)
		carapace.Gen(extensionInstallCmd).PositionalCompletion(carapace.ActionFiles().Usage("path to the extension .tar.gz or directory"))

		extensionRmCmd := &cobra.Command{
			Use:   consts.RmStr,
			Short: "Remove an installed extension",
			Args:  cobra.ExactArgs(1),
			Long:  help.GetHelpFor([]string{consts.ExtensionsStr, consts.RmStr}),
			Run: func(cmd *cobra.Command, args []string) {
				extensions.ExtensionsRemoveCmd(cmd, con, args)
			},
		}
		extensionCmd.AddCommand(extensionRmCmd)
		carapace.Gen(extensionRmCmd).PositionalCompletion(extensions.ExtensionsCommandNameCompleter(con).Usage("the command name of the extension to remove"))

		// [ Filesystem ] ---------------------------------------------

		mvCmd := &cobra.Command{
			Use:   consts.MvStr,
			Short: "Move or rename a file",
			Long:  help.GetHelpFor([]string{consts.MvStr}),
			Args:  cobra.ExactArgs(2),
			Run: func(cmd *cobra.Command, args []string) {
				filesystem.MvCmd(cmd, con, args)
			},
			GroupID: consts.FilesystemHelpGroup,
		}
		sliver.AddCommand(mvCmd)
		Flags("", false, mvCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		carapace.Gen(mvCmd).PositionalCompletion(
			carapace.ActionValues().Usage("path to source file (required)"),
			carapace.ActionValues().Usage("path to dest file (required)"),
		)

		cpCmd := &cobra.Command{
			Use:   consts.CpStr,
			Short: "Copy a file",
			Long:  help.GetHelpFor([]string{consts.CpStr}),
			Args:  cobra.ExactArgs(2),
			Run: func(cmd *cobra.Command, args []string) {
				filesystem.CpCmd(cmd, con, args)
			},
			GroupID: consts.FilesystemHelpGroup,
		}
		sliver.AddCommand(cpCmd)
		Flags("", false, cpCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		carapace.Gen(cpCmd).PositionalCompletion(
			carapace.ActionValues().Usage("path to source file (required)"),
			carapace.ActionValues().Usage("path to dest file (required)"),
		)

		lsCmd := &cobra.Command{
			Use:   consts.LsStr,
			Short: "List current directory",
			Long:  help.GetHelpFor([]string{consts.LsStr}),
			Args:  cobra.RangeArgs(0, 1),
			Run: func(cmd *cobra.Command, args []string) {
				filesystem.LsCmd(cmd, con, args)
			},
			GroupID: consts.FilesystemHelpGroup,
		}
		sliver.AddCommand(lsCmd)
		Flags("", false, lsCmd, func(f *pflag.FlagSet) {
			f.BoolP("reverse", "r", false, "reverse sort order")
			f.BoolP("modified", "m", false, "sort by modified time")
			f.BoolP("size", "s", false, "sort by size")
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		carapace.Gen(lsCmd).PositionalCompletion(carapace.ActionValues().Usage("path to enumerate (optional)"))

		rmCmd := &cobra.Command{
			Use:   consts.RmStr,
			Short: "Remove a file or directory",
			Long:  help.GetHelpFor([]string{consts.RmStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				filesystem.RmCmd(cmd, con, args)
			},
			GroupID: consts.FilesystemHelpGroup,
		}
		sliver.AddCommand(rmCmd)
		Flags("", false, rmCmd, func(f *pflag.FlagSet) {
			f.BoolP("recursive", "r", false, "recursively remove files")
			f.BoolP("force", "F", false, "ignore safety and forcefully remove files")
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		carapace.Gen(rmCmd).PositionalCompletion(carapace.ActionValues().Usage("path to the file to remove"))

		mkdirCmd := &cobra.Command{
			Use:   consts.MkdirStr,
			Short: "Make a directory",
			Long:  help.GetHelpFor([]string{consts.MkdirStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				filesystem.MkdirCmd(cmd, con, args)
			},
			GroupID: consts.FilesystemHelpGroup,
		}
		sliver.AddCommand(mkdirCmd)
		Flags("", false, mkdirCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		carapace.Gen(mkdirCmd).PositionalCompletion(carapace.ActionValues().Usage("path to the directory to create"))

		cdCmd := &cobra.Command{
			Use:   consts.CdStr,
			Short: "Change directory",
			Long:  help.GetHelpFor([]string{consts.CdStr}),
			Args:  cobra.RangeArgs(0, 1),
			Run: func(cmd *cobra.Command, args []string) {
				filesystem.CdCmd(cmd, con, args)
			},
			GroupID: consts.FilesystemHelpGroup,
		}
		sliver.AddCommand(cdCmd)
		Flags("", false, cdCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		carapace.Gen(cdCmd).PositionalCompletion(carapace.ActionValues().Usage("path to the directory"))

		pwdCmd := &cobra.Command{
			Use:   consts.PwdStr,
			Short: "Print working directory",
			Long:  help.GetHelpFor([]string{consts.PwdStr}),
			Run: func(cmd *cobra.Command, args []string) {
				filesystem.PwdCmd(cmd, con, args)
			},
			GroupID: consts.FilesystemHelpGroup,
		}
		sliver.AddCommand(pwdCmd)
		Flags("", false, pwdCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		catCmd := &cobra.Command{
			Use:   consts.CatStr,
			Short: "Dump file to stdout",
			Long:  help.GetHelpFor([]string{consts.CatStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				filesystem.CatCmd(cmd, con, args)
			},
			GroupID: consts.FilesystemHelpGroup,
		}
		sliver.AddCommand(catCmd)
		Flags("", false, catCmd, func(f *pflag.FlagSet) {
			f.BoolP("colorize-output", "c", false, "colorize output")
			f.BoolP("hex", "x", false, "display as a hex dump")
			f.BoolP("loot", "X", false, "save output as loot")
			f.StringP("name", "n", "", "name to assign loot (optional)")
			f.StringP("type", "T", "", "force a specific loot type (file/cred) if looting (optional)")
			f.StringP("file-type", "F", "", "force a specific file type (binary/text) if looting (optional)")
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		carapace.Gen(catCmd).PositionalCompletion(carapace.ActionValues().Usage("path to the file to print"))

		downloadCmd := &cobra.Command{
			Use:   consts.DownloadStr,
			Short: "Download a file",
			Long:  help.GetHelpFor([]string{consts.DownloadStr}),
			Args:  cobra.RangeArgs(1, 2),
			Run: func(cmd *cobra.Command, args []string) {
				filesystem.DownloadCmd(cmd, con, args)
			},
			GroupID: consts.FilesystemHelpGroup,
		}
		sliver.AddCommand(downloadCmd)
		Flags("", false, downloadCmd, func(f *pflag.FlagSet) {
			f.BoolP("loot", "X", false, "save output as loot")
			f.StringP("type", "T", "", "force a specific loot type (file/cred) if looting")
			f.StringP("file-type", "F", "", "force a specific file type (binary/text) if looting")
			f.StringP("name", "n", "", "name to assign the download if looting")
			f.BoolP("recurse", "r", false, "recursively download all files in a directory")
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		carapace.Gen(downloadCmd).PositionalCompletion(
			carapace.ActionValues().Usage("path to the file or directory to download"),
			carapace.ActionFiles().Usage("local path where the downloaded file will be saved (optional)"),
		)

		grepCmd := &cobra.Command{
			Use:   consts.GrepStr,
			Short: "Search for strings that match a regex within a file or directory",
			Long:  help.GetHelpFor([]string{consts.GrepStr}),
			Args:  cobra.ExactArgs(2),
			Run: func(cmd *cobra.Command, args []string) {
				filesystem.GrepCmd(cmd, con, args)
			},
			GroupID: consts.FilesystemHelpGroup,
		}
		sliver.AddCommand(grepCmd)
		Flags("", false, grepCmd, func(f *pflag.FlagSet) {
			f.BoolP("colorize-output", "c", false, "colorize output")
			f.BoolP("loot", "X", false, "save output as loot (loot is saved without formatting)")
			f.StringP("name", "n", "", "name to assign loot (optional)")
			f.StringP("type", "T", "", "force a specific loot type (file/cred) if looting (optional)")
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
			f.BoolP("recursive", "r", false, "search recursively")
			f.BoolP("insensitive", "i", false, "case-insensitive search")
			f.Int32P("after", "A", 0, "number of lines to print after a match (ignored if the file is binary)")
			f.Int32P("before", "B", 0, "number of lines to print before a match (ignored if the file is binary)")
			f.Int32P("context", "C", 0, "number of lines to print before and after a match (ignored if the file is binary), equivalent to -A x -B x")
			f.BoolP("exact", "e", false, "match the search term exactly")
		})
		carapace.Gen(grepCmd).PositionalCompletion(
			carapace.ActionValues().Usage("regex to search the file for"),
			carapace.ActionValues().Usage("remote path / file to search in"),
		)

		headCmd := &cobra.Command{
			Use:   consts.HeadStr,
			Short: "Grab the first number of bytes or lines from a file",
			Long:  help.GetHelpFor([]string{consts.HeadStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				/*
					The last argument tells head if the user requested the head or tail of the file
					True means head, false means tail
				*/
				filesystem.HeadCmd(cmd, con, args, true)
			},
			GroupID: consts.FilesystemHelpGroup,
		}
		sliver.AddCommand(headCmd)
		Flags("", false, headCmd, func(f *pflag.FlagSet) {
			f.BoolP("colorize-output", "c", false, "colorize output")
			f.BoolP("hex", "x", false, "display as a hex dump")
			f.BoolP("loot", "X", false, "save output as loot")
			f.StringP("name", "n", "", "name to assign loot (optional)")
			f.StringP("type", "T", "", "force a specific loot type (file/cred) if looting (optional)")
			f.StringP("file-type", "F", "", "force a specific file type (binary/text) if looting (optional)")
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
			f.Int64P("bytes", "b", 0, "Grab the first number of bytes from the file")
			f.Int64P("lines", "l", 0, "Grab the first number of lines from the file")
		})
		carapace.Gen(headCmd).PositionalCompletion(carapace.ActionValues().Usage("path to the file to print"))

		tailCmd := &cobra.Command{
			Use:   consts.TailStr,
			Short: "Grab the last number of bytes or lines from a file",
			Long:  help.GetHelpFor([]string{consts.TailStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				/*
					The last argument tells head if the user requested the head or tail of the file
					True means head, false means tail
				*/
				filesystem.HeadCmd(cmd, con, args, false)
			},
			GroupID: consts.FilesystemHelpGroup,
		}
		sliver.AddCommand(tailCmd)
		Flags("", false, tailCmd, func(f *pflag.FlagSet) {
			f.BoolP("colorize-output", "c", false, "colorize output")
			f.BoolP("hex", "x", false, "display as a hex dump")
			f.BoolP("loot", "X", false, "save output as loot")
			f.StringP("name", "n", "", "name to assign loot (optional)")
			f.StringP("type", "T", "", "force a specific loot type (file/cred) if looting (optional)")
			f.StringP("file-type", "F", "", "force a specific file type (binary/text) if looting (optional)")
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
			f.Int64P("bytes", "b", 0, "Grab the last number of bytes from the file")
			f.Int64P("lines", "l", 0, "Grab the last number of lines from the file")
		})
		carapace.Gen(tailCmd).PositionalCompletion(carapace.ActionValues().Usage("path to the file to print"))

		uploadCmd := &cobra.Command{
			Use:   consts.UploadStr,
			Short: "Upload a file or directory",
			Long:  help.GetHelpFor([]string{consts.UploadStr}),
			Args:  cobra.RangeArgs(1, 2),
			Run: func(cmd *cobra.Command, args []string) {
				filesystem.UploadCmd(cmd, con, args)
			},
			GroupID: consts.FilesystemHelpGroup,
		}
		sliver.AddCommand(uploadCmd)
		Flags("", false, uploadCmd, func(f *pflag.FlagSet) {
			f.BoolP("ioc", "i", false, "track uploaded file as an ioc")
			f.BoolP("recurse", "r", false, "recursively upload a directory")
			f.BoolP("overwrite", "o", false, "overwrite files that exist in the destination")
			f.BoolP("preserve", "p", false, "preserve directory structure when uploading a directory")
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		carapace.Gen(uploadCmd).PositionalCompletion(
			carapace.ActionFiles().Usage("local path to the file to upload"),
			carapace.ActionValues().Usage("path to the file or directory to upload to (optional)"),
		)

		memfilesCmd := &cobra.Command{
			Use:     consts.MemfilesStr,
			Short:   "List current memfiles",
			Long:    help.GetHelpFor([]string{consts.MemfilesStr}),
			GroupID: consts.FilesystemHelpGroup,
			Run: func(cmd *cobra.Command, args []string) {
				filesystem.MemfilesListCmd(cmd, con, args)
			},
		}
		Flags("", true, memfilesCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		sliver.AddCommand(memfilesCmd)

		memfilesAddCmd := &cobra.Command{
			Use:   consts.AddStr,
			Short: "Add a memfile",
			Long:  help.GetHelpFor([]string{consts.MemfilesStr, consts.AddStr}),
			Run: func(cmd *cobra.Command, args []string) {
				filesystem.MemfilesAddCmd(cmd, con, args)
			},
		}
		memfilesCmd.AddCommand(memfilesAddCmd)

		memfilesRmCmd := &cobra.Command{
			Use:   consts.RmStr,
			Short: "Remove a memfile",
			Long:  help.GetHelpFor([]string{consts.MemfilesStr, consts.RmStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				filesystem.MemfilesRmCmd(cmd, con, args)
			},
		}
		memfilesCmd.AddCommand(memfilesRmCmd)

		carapace.Gen(memfilesRmCmd).PositionalCompletion(carapace.ActionValues().Usage("memfile file descriptor"))

		// [ Network ] ---------------------------------------------

		ifconfigCmd := &cobra.Command{
			Use:   consts.IfconfigStr,
			Short: "View network interface configurations",
			Long:  help.GetHelpFor([]string{consts.IfconfigStr}),
			Run: func(cmd *cobra.Command, args []string) {
				network.IfconfigCmd(cmd, con, args)
			},
			GroupID: consts.NetworkHelpGroup,
		}
		sliver.AddCommand(ifconfigCmd)
		Flags("", false, ifconfigCmd, func(f *pflag.FlagSet) {
			f.BoolP("all", "A", false, "show all network adapters (default only shows IPv4)")
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		netstatCmd := &cobra.Command{
			Use:   consts.NetstatStr,
			Short: "Print network connection information",
			Long:  help.GetHelpFor([]string{consts.NetstatStr}),
			Run: func(cmd *cobra.Command, args []string) {
				network.NetstatCmd(cmd, con, args)
			},
			GroupID: consts.NetworkHelpGroup,
		}
		sliver.AddCommand(netstatCmd)
		Flags("", false, netstatCmd, func(f *pflag.FlagSet) {
			f.BoolP("tcp", "T", true, "display information about TCP sockets")
			f.BoolP("udp", "u", false, "display information about UDP sockets")
			f.BoolP("ip4", "4", true, "display information about IPv4 sockets")
			f.BoolP("ip6", "6", false, "display information about IPv6 sockets")
			f.BoolP("listen", "l", false, "display information about listening sockets")
			f.BoolP("numeric", "n", false, "display numeric addresses (disable hostname resolution)")
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		// [ Processes ] ---------------------------------------------

		psCmd := &cobra.Command{
			Use:   consts.PsStr,
			Short: "List remote processes",
			Long:  help.GetHelpFor([]string{consts.PsStr}),
			Run: func(cmd *cobra.Command, args []string) {
				processes.PsCmd(cmd, con, args)
			},
			GroupID: consts.ProcessHelpGroup,
		}
		sliver.AddCommand(psCmd)
		Flags("", false, psCmd, func(f *pflag.FlagSet) {
			f.IntP("pid", "p", -1, "filter based on pid")
			f.StringP("exe", "e", "", "filter based on executable name")
			f.StringP("owner", "o", "", "filter based on owner")
			f.BoolP("print-cmdline", "c", false, "print command line arguments")
			f.BoolP("overflow", "O", false, "overflow terminal width (display truncated rows)")
			f.IntP("skip-pages", "S", 0, "skip the first n page(s)")
			f.BoolP("tree", "T", false, "print process tree")

			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		procdumpCmd := &cobra.Command{
			Use:   consts.ProcdumpStr,
			Short: "Dump process memory",
			Long:  help.GetHelpFor([]string{consts.ProcdumpStr}),
			Run: func(cmd *cobra.Command, args []string) {
				processes.ProcdumpCmd(cmd, con, args)
			},
			GroupID: consts.ProcessHelpGroup,
		}
		sliver.AddCommand(procdumpCmd)
		Flags("", false, procdumpCmd, func(f *pflag.FlagSet) {
			f.IntP("pid", "p", -1, "target pid")
			f.StringP("name", "n", "", "target process name")
			f.StringP("save", "s", "", "save to file (will overwrite if exists)")
			f.BoolP("loot", "X", false, "save output as loot")
			f.StringP("loot-name", "N", "", "name to assign when adding the memory dump to the loot store (optional)")

			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		terminateCmd := &cobra.Command{
			Use:   consts.TerminateStr,
			Short: "Terminate a process on the remote system",
			Long:  help.GetHelpFor([]string{consts.TerminateStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				processes.TerminateCmd(cmd, con, args)
			},
			GroupID: consts.ProcessHelpGroup,
		}
		sliver.AddCommand(terminateCmd)
		Flags("", false, terminateCmd, func(f *pflag.FlagSet) {
			f.BoolP("force", "F", false, "disregard safety and kill the PID")
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		carapace.Gen(terminateCmd).PositionalCompletion(carapace.ActionValues().Usage("process ID"))

		// [ Privileges ] ---------------------------------------------

		runAsCmd := &cobra.Command{
			Use:   consts.RunAsStr,
			Short: "Run a new process in the context of the designated user (Windows Only)",
			Long:  help.GetHelpFor([]string{consts.RunAsStr}),
			Run: func(cmd *cobra.Command, args []string) {
				privilege.RunAsCmd(cmd, con, args)
			},
			GroupID:     consts.PrivilegesHelpGroup,
			Annotations: hideCommand(consts.WindowsCmdsFilter),
		}
		sliver.AddCommand(runAsCmd)
		Flags("", false, runAsCmd, func(f *pflag.FlagSet) {
			f.StringP("username", "u", "", "user to impersonate")
			f.StringP("process", "p", "", "process to start")
			f.StringP("args", "a", "", "arguments for the process")
			f.StringP("domain", "d", "", "domain of the user")
			f.StringP("password", "P", "", "password of the user")
			f.BoolP("show-window", "s", false, `
			Log on, but use the specified credentials on the network only. The new process uses the same token as the caller, but the system creates a new logon session within LSA, and the process uses the specified credentials as the default credentials.`)
			f.BoolP("net-only", "n", false, "use ")
			f.Int64P("timeout", "t", 30, "grpc timeout in seconds")
		})

		impersonateCmd := &cobra.Command{
			Use:   consts.ImpersonateStr,
			Short: "Impersonate a logged in user.",
			Long:  help.GetHelpFor([]string{consts.ImpersonateStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				privilege.ImpersonateCmd(cmd, con, args)
			},
			GroupID:     consts.PrivilegesHelpGroup,
			Annotations: hideCommand(consts.WindowsCmdsFilter),
		}
		sliver.AddCommand(impersonateCmd)
		Flags("", false, impersonateCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", 30, "grpc timeout in seconds")
		})
		carapace.Gen(impersonateCmd).PositionalCompletion(carapace.ActionValues().Usage("name of the user account to impersonate"))

		revToSelfCmd := &cobra.Command{
			Use:   consts.RevToSelfStr,
			Short: "Revert to self: lose stolen Windows token",
			Long:  help.GetHelpFor([]string{consts.RevToSelfStr}),
			Run: func(cmd *cobra.Command, args []string) {
				privilege.RevToSelfCmd(cmd, con, args)
			},
			GroupID:     consts.PrivilegesHelpGroup,
			Annotations: hideCommand(consts.WindowsCmdsFilter),
		}
		sliver.AddCommand(revToSelfCmd)
		Flags("", false, revToSelfCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", 30, "grpc timeout in seconds")
		})

		getSystemCmd := &cobra.Command{
			Use:   consts.GetSystemStr,
			Short: "Spawns a new sliver session as the NT AUTHORITY\\SYSTEM user (Windows Only)",
			Long:  help.GetHelpFor([]string{consts.GetSystemStr}),
			Run: func(cmd *cobra.Command, args []string) {
				privilege.GetSystemCmd(cmd, con, args)
			},
			GroupID:     consts.PrivilegesHelpGroup,
			Annotations: hideCommand(consts.WindowsCmdsFilter),
		}
		sliver.AddCommand(getSystemCmd)
		Flags("", false, getSystemCmd, func(f *pflag.FlagSet) {
			f.StringP("process", "p", "spoolsv.exe", "SYSTEM process to inject into")
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		makeTokenCmd := &cobra.Command{
			Use:         consts.MakeTokenStr,
			Short:       "Create a new Logon Session with the specified credentials",
			Long:        help.GetHelpFor([]string{consts.MakeTokenStr}),
			GroupID:     consts.PrivilegesHelpGroup,
			Annotations: hideCommand(consts.WindowsCmdsFilter),
			Run: func(cmd *cobra.Command, args []string) {
				privilege.MakeTokenCmd(cmd, con, args)
			},
		}
		sliver.AddCommand(makeTokenCmd)
		Flags("", false, makeTokenCmd, func(f *pflag.FlagSet) {
			f.StringP("username", "u", "", "username of the user to impersonate")
			f.StringP("password", "p", "", "password of the user to impersonate")
			f.StringP("domain", "d", "", "domain of the user to impersonate")
			f.StringP("logon-type", "T", "LOGON_NEW_CREDENTIALS", "logon type to use")
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		chmodCmd := &cobra.Command{
			Use:   consts.ChmodStr,
			Short: "Change permissions on a file or directory",
			Long:  help.GetHelpFor([]string{consts.ChmodStr}),
			Args:  cobra.ExactArgs(2),
			Run: func(cmd *cobra.Command, args []string) {
				filesystem.ChmodCmd(cmd, con, args)
			},
			GroupID: consts.PrivilegesHelpGroup,
		}
		sliver.AddCommand(chmodCmd)
		Flags("", false, chmodCmd, func(f *pflag.FlagSet) {
			f.BoolP("recursive", "r", false, "recursively change permissions on files")
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		carapace.Gen(chmodCmd).PositionalCompletion(
			carapace.ActionValues().Usage("path to file to change mod perms"),
			carapace.ActionValues().Usage("file permissions in octal (eg. 0644)"),
		)

		chownCmd := &cobra.Command{
			Use:   consts.ChownStr,
			Short: "Change owner on a file or directory",
			Long:  help.GetHelpFor([]string{consts.ChownStr}),
			Args:  cobra.ExactArgs(3),
			Run: func(cmd *cobra.Command, args []string) {
				filesystem.ChownCmd(cmd, con, args)
			},
			GroupID: consts.PrivilegesHelpGroup,
		}
		sliver.AddCommand(chownCmd)
		Flags("", false, chownCmd, func(f *pflag.FlagSet) {
			f.BoolP("recursive", "r", false, "recursively change permissions on files")
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		carapace.Gen(chownCmd).PositionalCompletion(
			carapace.ActionValues().Usage("path to file to change owner for"),
			carapace.ActionValues().Usage("user ID"),
			carapace.ActionValues().Usage("group ID (required)"),
		)

		chtimesCmd := &cobra.Command{
			Use:   consts.ChtimesStr,
			Short: "Change access and modification times on a file (timestomp)",
			Long:  help.GetHelpFor([]string{consts.ChtimesStr}),
			Args:  cobra.ExactArgs(3),
			Run: func(cmd *cobra.Command, args []string) {
				filesystem.ChtimesCmd(cmd, con, args)
			},
			GroupID: consts.PrivilegesHelpGroup,
		}
		sliver.AddCommand(chtimesCmd)
		Flags("", false, chtimesCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		carapace.Gen(chtimesCmd).PositionalCompletion(
			carapace.ActionValues().Usage("path to file to change access timestamps"),
			carapace.ActionValues().Usage("last accessed time in DateTime format, i.e. 2006-01-02 15:04:05"),
			carapace.ActionValues().Usage("last modified time in DateTime format, i.e. 2006-01-02 15:04:05"),
		)

		// [ Screenshot ] ---------------------------------------------

		screenshotCmd := &cobra.Command{
			Use:   consts.ScreenshotStr,
			Short: "Take a screenshot",
			Long:  help.GetHelpFor([]string{consts.ScreenshotStr}),
			Run: func(cmd *cobra.Command, args []string) {
				screenshot.ScreenshotCmd(cmd, con, args)
			},
			GroupID: consts.InfoHelpGroup,
		}
		sliver.AddCommand(screenshotCmd)
		Flags("", false, screenshotCmd, func(f *pflag.FlagSet) {
			f.StringP("save", "s", "", "save to file (will overwrite if exists)")
			f.BoolP("loot", "X", false, "save output as loot")
			f.StringP("name", "n", "", "name to assign loot (optional)")

			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		FlagComps(screenshotCmd, func(comp *carapace.ActionMap) {
			(*comp)["save"] = carapace.ActionFiles()
		})

		// [ Backdoor ] ---------------------------------------------

		backdoorCmd := &cobra.Command{
			Use:         consts.BackdoorStr,
			Short:       "Infect a remote file with a sliver shellcode",
			Long:        help.GetHelpFor([]string{consts.BackdoorStr}),
			Args:        cobra.ExactArgs(1),
			GroupID:     consts.ExecutionHelpGroup,
			Annotations: hideCommand(consts.WindowsCmdsFilter),
			Run: func(cmd *cobra.Command, args []string) {
				backdoor.BackdoorCmd(cmd, con, args)
			},
		}
		sliver.AddCommand(backdoorCmd)
		Flags("", false, backdoorCmd, func(f *pflag.FlagSet) {
			f.StringP("profile", "p", "", "profile to use for service binary")
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		FlagComps(screenshotCmd, func(comp *carapace.ActionMap) {
			(*comp)["profile"] = generate.ProfileNameCompleter(con)
		})
		carapace.Gen(backdoorCmd).PositionalCompletion(carapace.ActionValues().Usage("path to the remote file to backdoor"))

		// // [ DLL Hijack ] -----------------------------------------------------------------

		dllhijackCmd := &cobra.Command{
			Use:         consts.DLLHijackStr,
			Short:       "Plant a DLL for a hijack scenario",
			Long:        help.GetHelpFor([]string{consts.DLLHijackStr}),
			GroupID:     consts.ExecutionHelpGroup,
			Annotations: hideCommand(consts.WindowsCmdsFilter),
			Args:        cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				dllhijack.DllHijackCmd(cmd, con, args)
			},
		}
		sliver.AddCommand(dllhijackCmd)
		Flags("", false, dllhijackCmd, func(f *pflag.FlagSet) {
			f.StringP("reference-path", "r", "", "Path to the reference DLL on the remote system")
			f.StringP("reference-file", "R", "", "Path to the reference DLL on the local system")
			f.StringP("file", "f", "", "Local path to the DLL to plant for the hijack")
			f.StringP("profile", "p", "", "Profile name to use as a base DLL")
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		FlagComps(dllhijackCmd, func(comp *carapace.ActionMap) {
			(*comp)["reference-file"] = carapace.ActionFiles()
			(*comp)["file"] = carapace.ActionFiles()
			(*comp)["profile"] = generate.ProfileNameCompleter(con)
		})
		carapace.Gen(dllhijackCmd).PositionalCompletion(carapace.ActionValues().Usage("Path to upload the DLL to on the remote system"))

		// [ Get Privs ] -----------------------------------------------------------------
		getprivsCmd := &cobra.Command{
			Use:         consts.GetPrivsStr,
			Short:       "Get current privileges (Windows only)",
			Long:        help.GetHelpFor([]string{consts.GetPrivsStr}),
			GroupID:     consts.PrivilegesHelpGroup,
			Annotations: hideCommand(consts.WindowsCmdsFilter),
			Run: func(cmd *cobra.Command, args []string) {
				privilege.GetPrivsCmd(cmd, con, args)
			},
		}
		sliver.AddCommand(getprivsCmd)
		Flags("", false, getprivsCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		//

		// [ Environment ] ---------------------------------------------

		envCmd := &cobra.Command{
			Use:   consts.EnvStr,
			Short: "List environment variables",
			Long:  help.GetHelpFor([]string{consts.EnvStr}),
			Args:  cobra.RangeArgs(0, 1),
			Run: func(cmd *cobra.Command, args []string) {
				environment.EnvGetCmd(cmd, con, args)
			},
			GroupID: consts.InfoHelpGroup,
		}
		sliver.AddCommand(envCmd)
		Flags("", true, envCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		carapace.Gen(envCmd).PositionalCompletion(carapace.ActionValues().Usage("environment variable to fetch (optional)"))

		envSetCmd := &cobra.Command{
			Use:   consts.SetStr,
			Short: "Set environment variables",
			Long:  help.GetHelpFor([]string{consts.EnvStr, consts.SetStr}),
			Args:  cobra.ExactArgs(2),
			Run: func(cmd *cobra.Command, args []string) {
				environment.EnvSetCmd(cmd, con, args)
			},
		}
		envCmd.AddCommand(envSetCmd)
		carapace.Gen(envSetCmd).PositionalCompletion(
			carapace.ActionValues().Usage("environment variable name"),
			carapace.ActionValues().Usage("value to assign"),
		)

		envUnsetCmd := &cobra.Command{
			Use:   consts.UnsetStr,
			Short: "Clear environment variables",
			Long:  help.GetHelpFor([]string{consts.EnvStr, consts.UnsetStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				environment.EnvUnsetCmd(cmd, con, args)
			},
		}
		envCmd.AddCommand(envUnsetCmd)
		carapace.Gen(envUnsetCmd).PositionalCompletion(carapace.ActionValues().Usage("environment variable name"))

		// [ Registry ] ---------------------------------------------

		registryCmd := &cobra.Command{
			Use:         consts.RegistryStr,
			Short:       "Windows registry operations",
			Long:        help.GetHelpFor([]string{consts.RegistryStr}),
			GroupID:     consts.InfoHelpGroup,
			Annotations: hideCommand(consts.WindowsCmdsFilter),
		}
		sliver.AddCommand(registryCmd)
		Flags("registry", true, registryCmd, func(f *pflag.FlagSet) {
			f.IntP("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		registryReadCmd := &cobra.Command{
			Use:   consts.RegistryReadStr,
			Short: "Read values from the Windows registry",
			Long:  help.GetHelpFor([]string{consts.RegistryReadStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				registry.RegReadCmd(cmd, con, args)
			},
		}
		registryCmd.AddCommand(registryReadCmd)
		Flags("", false, registryReadCmd, func(f *pflag.FlagSet) {
			f.StringP("hive", "H", "HKCU", "registry hive")
			f.StringP("hostname", "o", "", "remote host to read values from")
		})
		carapace.Gen(registryReadCmd).PositionalCompletion(carapace.ActionValues().Usage("registry path"))

		registryWriteCmd := &cobra.Command{
			Use:   consts.RegistryWriteStr,
			Short: "Write values to the Windows registry",
			Long:  help.GetHelpFor([]string{consts.RegistryWriteStr}),
			Args:  cobra.ExactArgs(2),
			Run: func(cmd *cobra.Command, args []string) {
				registry.RegWriteCmd(cmd, con, args)
			},
		}
		registryCmd.AddCommand(registryWriteCmd)
		Flags("", false, registryWriteCmd, func(f *pflag.FlagSet) {
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
				registry.RegCreateKeyCmd(cmd, con, args)
			},
		}
		registryCmd.AddCommand(registryCreateKeyCmd)
		Flags("", false, registryCreateKeyCmd, func(f *pflag.FlagSet) {
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
				registry.RegDeleteKeyCmd(cmd, con, args)
			},
		}
		registryCmd.AddCommand(registryDeleteKeyCmd)
		Flags("", false, registryDeleteKeyCmd, func(f *pflag.FlagSet) {
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
				registry.RegListSubKeysCmd(cmd, con, args)
			},
		}
		registryCmd.AddCommand(registryListSubCmd)
		Flags("", false, registryListSubCmd, func(f *pflag.FlagSet) {
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
				registry.RegListValuesCmd(cmd, con, args)
			},
		}
		registryCmd.AddCommand(registryListValuesCmd)
		Flags("", false, registryListValuesCmd, func(f *pflag.FlagSet) {
			f.StringP("hive", "H", "HKCU", "registry hive")
			f.StringP("hostname", "o", "", "remote host to write values to")
		})
		carapace.Gen(registryListValuesCmd).PositionalCompletion(carapace.ActionValues().Usage("registry path"))

		// [ Reverse Port Forwarding ] --------------------------------------------------------------

		rportfwdCmd := &cobra.Command{
			Use:   consts.RportfwdStr,
			Short: "reverse port forwardings",
			Long:  help.GetHelpFor([]string{consts.RportfwdStr}),
			Run: func(cmd *cobra.Command, args []string) {
				rportfwd.RportFwdListenersCmd(cmd, con, args)
			},
			GroupID: consts.NetworkHelpGroup,
		}
		sliver.AddCommand(rportfwdCmd)
		Flags("", true, rportfwdCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		rportfwdAddCmd := &cobra.Command{
			Use:   consts.AddStr,
			Short: "Add and start reverse port forwarding",
			Long:  help.GetHelpFor([]string{consts.RportfwdStr}),
			Run: func(cmd *cobra.Command, args []string) {
				rportfwd.StartRportFwdListenerCmd(cmd, con, args)
			},
		}
		rportfwdCmd.AddCommand(rportfwdAddCmd)
		Flags("", false, rportfwdAddCmd, func(f *pflag.FlagSet) {
			f.StringP("remote", "r", "", "remote address <ip>:<port> connection is forwarded to")
			f.StringP("bind", "b", "", "bind address <ip>:<port> for implants to listen on")
		})
		FlagComps(rportfwdAddCmd, func(comp *carapace.ActionMap) {
			(*comp)["remote"] = completers.ClientInterfacesCompleter()
		})

		rportfwdRmCmd := &cobra.Command{
			Use:   consts.RmStr,
			Short: "Stop and remove reverse port forwarding",
			Long:  help.GetHelpFor([]string{consts.RportfwdStr}),
			Run: func(cmd *cobra.Command, args []string) {
				rportfwd.StopRportFwdListenerCmd(cmd, con, args)
			},
		}
		rportfwdCmd.AddCommand(rportfwdRmCmd)
		Flags("", false, rportfwdRmCmd, func(f *pflag.FlagSet) {
			f.Uint32P("id", "i", 0, "id of portfwd to remove")
		})
		FlagComps(rportfwdRmCmd, func(comp *carapace.ActionMap) {
			(*comp)["id"] = rportfwd.PortfwdIDCompleter(con)
		})

		// [ Pivots ] --------------------------------------------------------------

		pivotsCmd := &cobra.Command{
			Use:   consts.PivotsStr,
			Short: "List pivots for active session",
			Long:  help.GetHelpFor([]string{consts.PivotsStr}),
			Run: func(cmd *cobra.Command, args []string) {
				pivots.PivotsCmd(cmd, con, args)
			},
			GroupID: consts.SliverCoreHelpGroup,
		}
		sliver.AddCommand(pivotsCmd)
		Flags("", true, pivotsCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		namedPipeCmd := &cobra.Command{
			Use:   consts.NamedPipeStr,
			Short: "Start a named pipe pivot listener",
			Long:  help.GetHelpFor([]string{consts.PivotsStr, consts.NamedPipeStr}),
			Run: func(cmd *cobra.Command, args []string) {
				pivots.StartNamedPipeListenerCmd(cmd, con, args)
			},
		}
		pivotsCmd.AddCommand(namedPipeCmd)
		Flags("", false, namedPipeCmd, func(f *pflag.FlagSet) {
			f.StringP("bind", "b", "", "name of the named pipe to bind pivot listener")
			f.BoolP("allow-all", "a", false, "allow all users to connect")
		})

		tcpListenerCmd := &cobra.Command{
			Use:   consts.TCPListenerStr,
			Short: "Start a TCP pivot listener",
			Long:  help.GetHelpFor([]string{consts.PivotsStr, consts.TCPListenerStr}),
			Run: func(cmd *cobra.Command, args []string) {
				pivots.StartTCPListenerCmd(cmd, con, args)
			},
		}
		pivotsCmd.AddCommand(tcpListenerCmd)
		Flags("", false, tcpListenerCmd, func(f *pflag.FlagSet) {
			f.StringP("bind", "b", "", "remote interface to bind pivot listener")
			f.Uint16P("lport", "l", generate.DefaultTCPPivotPort, "tcp pivot listener port")
		})

		pivotStopCmd := &cobra.Command{
			Use:   consts.StopStr,
			Short: "Stop a pivot listener",
			Long:  help.GetHelpFor([]string{consts.PivotsStr, consts.StopStr}),
			Run: func(cmd *cobra.Command, args []string) {
				pivots.StopPivotListenerCmd(cmd, con, args)
			},
		}
		pivotsCmd.AddCommand(pivotStopCmd)
		Flags("", false, pivotStopCmd, func(f *pflag.FlagSet) {
			f.Uint32P("id", "i", 0, "id of the pivot listener to stop")
		})
		FlagComps(pivotStopCmd, func(comp *carapace.ActionMap) {
			(*comp)["id"] = pivots.PivotIDCompleter(con)
		})

		pivotDetailsCmd := &cobra.Command{
			Use:   consts.DetailsStr,
			Short: "Get details of a pivot listener",
			Long:  help.GetHelpFor([]string{consts.PivotsStr, consts.StopStr}),
			Run: func(cmd *cobra.Command, args []string) {
				pivots.PivotDetailsCmd(cmd, con, args)
			},
		}
		pivotsCmd.AddCommand(pivotDetailsCmd)
		Flags("", false, pivotDetailsCmd, func(f *pflag.FlagSet) {
			f.IntP("id", "i", 0, "id of the pivot listener to get details for")
		})
		FlagComps(pivotDetailsCmd, func(comp *carapace.ActionMap) {
			(*comp)["id"] = pivots.PivotIDCompleter(con)
		})

		graphCmd := &cobra.Command{
			Use:   consts.GraphStr,
			Short: "Get pivot listeners graph",
			Long:  help.GetHelpFor([]string{consts.PivotsStr, "graph"}),
			Run: func(cmd *cobra.Command, args []string) {
				pivots.PivotsGraphCmd(cmd, con, args)
			},
		}
		pivotsCmd.AddCommand(graphCmd)

		// [ Portfwd ] --------------------------------------------------------------

		portfwdCmd := &cobra.Command{
			Use:   consts.PortfwdStr,
			Short: "In-band TCP port forwarding",
			Long:  help.GetHelpFor([]string{consts.PortfwdStr}),
			Run: func(cmd *cobra.Command, args []string) {
				portfwd.PortfwdCmd(cmd, con, args)
			},
			GroupID: consts.NetworkHelpGroup,
		}
		sliver.AddCommand(portfwdCmd)
		Flags("", true, portfwdCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		addCmd := &cobra.Command{
			Use:   consts.AddStr,
			Short: "Create a new port forwarding tunnel",
			Long:  help.GetHelpFor([]string{consts.PortfwdStr}),
			Run: func(cmd *cobra.Command, args []string) {
				portfwd.PortfwdAddCmd(cmd, con, args)
			},
		}
		portfwdCmd.AddCommand(addCmd)
		Flags("", false, addCmd, func(f *pflag.FlagSet) {
			f.StringP("remote", "r", "", "remote target host:port (e.g., 10.0.0.1:445)")
			f.StringP("bind", "b", "127.0.0.1:8080", "bind port forward to interface")
		})
		FlagComps(addCmd, func(comp *carapace.ActionMap) {
			(*comp)["bind"] = completers.ClientInterfacesCompleter()
		})

		portfwdRmCmd := &cobra.Command{
			Use:   consts.RmStr,
			Short: "Remove a port forwarding tunnel",
			Long:  help.GetHelpFor([]string{consts.PortfwdStr}),
			Run: func(cmd *cobra.Command, args []string) {
				portfwd.PortfwdRmCmd(cmd, con, args)
			},
		}
		portfwdCmd.AddCommand(portfwdRmCmd)
		Flags("", false, portfwdRmCmd, func(f *pflag.FlagSet) {
			f.IntP("id", "i", 0, "id of portfwd to remove")
		})
		FlagComps(portfwdRmCmd, func(comp *carapace.ActionMap) {
			(*comp)["id"] = portfwd.PortfwdIDCompleter(con)
		})

		// [ Socks ] --------------------------------------------------------------

		socksCmd := &cobra.Command{
			Use:   consts.Socks5Str,
			Short: "In-band SOCKS5 Proxy",
			Long:  help.GetHelpFor([]string{consts.Socks5Str}),
			Run: func(cmd *cobra.Command, args []string) {
				socks.SocksCmd(cmd, con, args)
			},
			GroupID: consts.NetworkHelpGroup,
		}
		sliver.AddCommand(socksCmd)
		Flags("", true, socksCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		socksStartCmd := &cobra.Command{
			Use:   consts.StartStr,
			Short: "Start an in-band SOCKS5 proxy",
			Long:  help.GetHelpFor([]string{consts.Socks5Str}),
			Run: func(cmd *cobra.Command, args []string) {
				socks.SocksStartCmd(cmd, con, args)
			},
		}
		socksCmd.AddCommand(socksStartCmd)
		Flags("", false, socksStartCmd, func(f *pflag.FlagSet) {
			f.StringP("host", "H", "127.0.0.1", "Bind a Socks5 Host")
			f.StringP("port", "P", "1081", "Bind a Socks5 Port")
			f.StringP("user", "u", "", "socks5 auth username (will generate random password)")
		})
		FlagComps(socksStartCmd, func(comp *carapace.ActionMap) {
			(*comp)["host"] = completers.ClientInterfacesCompleter()
		})

		socksStopCmd := &cobra.Command{
			Use:   consts.StopStr,
			Short: "Stop a SOCKS5 proxy",
			Long:  help.GetHelpFor([]string{consts.Socks5Str}),
			Run: func(cmd *cobra.Command, args []string) {
				socks.SocksStopCmd(cmd, con, args)
			},
		}
		socksCmd.AddCommand(socksStopCmd)
		Flags("", false, socksStopCmd, func(f *pflag.FlagSet) {
			f.Uint64P("id", "i", 0, "id of portfwd to remove")
		})
		FlagComps(socksStopCmd, func(comp *carapace.ActionMap) {
			(*comp)["id"] = socks.SocksIDCompleter(con)
		})

		// [ WireGuard ] --------------------------------------------------------------

		wgPortFwdCmd := &cobra.Command{
			Use:   consts.WgPortFwdStr,
			Short: "List ports forwarded by the WireGuard tun interface",
			Long:  help.GetHelpFor([]string{consts.WgPortFwdStr}),
			Run: func(cmd *cobra.Command, args []string) {
				wireguard.WGPortFwdListCmd(cmd, con, args)
			},
			GroupID:     consts.NetworkHelpGroup,
			Annotations: hideCommand(consts.WireguardCmdsFilter),
		}
		Flags("wg portforward", true, wgPortFwdCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		sliver.AddCommand(wgPortFwdCmd)

		wgPortFwdAddCmd := &cobra.Command{
			Use:   consts.AddStr,
			Short: "Add a port forward from the WireGuard tun interface to a host on the target network",
			Long:  help.GetHelpFor([]string{consts.WgPortFwdStr, consts.AddStr}),
			Run: func(cmd *cobra.Command, args []string) {
				wireguard.WGPortFwdAddCmd(cmd, con, args)
			},
		}
		Flags("wg portforward", false, wgPortFwdAddCmd, func(f *pflag.FlagSet) {
			f.Int32P("bind", "b", 1080, "port to listen on the WireGuard tun interface")
			f.StringP("remote", "r", "", "remote target host:port (e.g., 10.0.0.1:445)")
		})
		wgPortFwdCmd.AddCommand(wgPortFwdAddCmd)

		wgPortFwdRmCmd := &cobra.Command{
			Use:   consts.RmStr,
			Short: "Remove a port forward from the WireGuard tun interface",
			Long:  help.GetHelpFor([]string{consts.WgPortFwdStr, consts.RmStr}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				wireguard.WGPortFwdRmCmd(cmd, con, args)
			},
		}
		wgPortFwdCmd.AddCommand(wgPortFwdRmCmd)

		carapace.Gen(wgPortFwdRmCmd).PositionalCompletion(wireguard.PortfwdIDCompleter(con).Usage("forwarder ID"))

		wgSocksCmd := &cobra.Command{
			Use:   consts.WgSocksStr,
			Short: "List socks servers listening on the WireGuard tun interface",
			Long:  help.GetHelpFor([]string{consts.WgSocksStr}),
			Run: func(cmd *cobra.Command, args []string) {
				wireguard.WGSocksListCmd(cmd, con, args)
			},
			GroupID:     consts.NetworkHelpGroup,
			Annotations: hideCommand(consts.WireguardCmdsFilter),
		}
		sliver.AddCommand(wgSocksCmd)
		Flags("wg socks", true, wgSocksCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		wgSocksStartCmd := &cobra.Command{
			Use:   consts.StartStr,
			Short: "Start a socks5 listener on the WireGuard tun interface",
			Long:  help.GetHelpFor([]string{consts.WgSocksStr, consts.StartStr}),
			Run: func(cmd *cobra.Command, args []string) {
				wireguard.WGSocksStartCmd(cmd, con, args)
			},
		}
		wgSocksCmd.AddCommand(wgSocksStartCmd)
		Flags("wg socks", false, wgSocksStartCmd, func(f *pflag.FlagSet) {
			f.Int32P("bind", "b", 3090, "port to listen on the WireGuard tun interface")
		})

		wgSocksStopCmd := &cobra.Command{
			Use:   consts.StopStr,
			Short: "Stop a socks5 listener on the WireGuard tun interface",
			Long:  help.GetHelpFor([]string{consts.WgSocksStr, consts.StopStr}),
			Run: func(cmd *cobra.Command, args []string) {
				wireguard.WGSocksStopCmd(cmd, con, args)
			},
			Args: cobra.ExactArgs(1),
		}
		wgSocksCmd.AddCommand(wgSocksStopCmd)
		carapace.Gen(wgSocksStopCmd).PositionalCompletion(wireguard.SocksIDCompleter(con).Usage("Socks server ID"))

		// [ Curse Commands ] ------------------------------------------------------------

		cursedCmd := &cobra.Command{
			Use:     consts.Cursed,
			Short:   "Chrome/electron post-exploitation tool kit (∩｀-´)⊃━☆ﾟ.*･｡ﾟ",
			Long:    help.GetHelpFor([]string{consts.Cursed}),
			GroupID: consts.ExecutionHelpGroup,
			Run: func(cmd *cobra.Command, args []string) {
				cursed.CursedCmd(cmd, con, args)
			},
		}
		sliver.AddCommand(cursedCmd)
		Flags("", true, cursedCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})

		cursedRmCmd := &cobra.Command{
			Use:   consts.RmStr,
			Short: "Remove a Curse from a process",
			Long:  help.GetHelpFor([]string{consts.Cursed, consts.CursedConsole}),
			Args:  cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				cursed.CursedRmCmd(cmd, con, args)
			},
		}
		cursedCmd.AddCommand(cursedRmCmd)
		Flags("", false, cursedRmCmd, func(f *pflag.FlagSet) {
			f.BoolP("kill", "k", false, "kill the process after removing the curse")
		})
		carapace.Gen(cursedRmCmd).PositionalCompletion(carapace.ActionValues().Usage("bind port of the Cursed process to stop"))

		cursedConsoleCmd := &cobra.Command{
			Use:   consts.CursedConsole,
			Short: "Start a JavaScript console connected to a debug target",
			Long:  help.GetHelpFor([]string{consts.Cursed, consts.CursedConsole}),
			Run: func(cmd *cobra.Command, args []string) {
				cursed.CursedConsoleCmd(cmd, con, args)
			},
		}
		cursedCmd.AddCommand(cursedConsoleCmd)
		Flags("", false, cursedConsoleCmd, func(f *pflag.FlagSet) {
			f.IntP("remote-debugging-port", "r", 0, "remote debugging tcp port (0 = random)`")
		})

		cursedChromeCmd := &cobra.Command{
			Use:   consts.CursedChrome,
			Short: "Automatically inject a Cursed Chrome payload into a remote Chrome extension",
			Long:  help.GetHelpFor([]string{consts.Cursed, consts.CursedChrome}),
			Run: func(cmd *cobra.Command, args []string) {
				cursed.CursedChromeCmd(cmd, con, args)
			},
		}
		cursedCmd.AddCommand(cursedChromeCmd)
		Flags("", false, cursedChromeCmd, func(f *pflag.FlagSet) {
			f.IntP("remote-debugging-port", "r", 0, "remote debugging tcp port (0 = random)")
			f.BoolP("restore", "R", true, "restore the user's session after process termination")
			f.StringP("exe", "e", "", "chrome/chromium browser executable path (blank string = auto)")
			f.StringP("user-data", "u", "", "user data directory (blank string = auto)")
			f.StringP("payload", "p", "", "cursed chrome payload file path (.js)")
			f.BoolP("keep-alive", "k", false, "keeps browser alive after last browser window closes")
			f.BoolP("headless", "H", false, "start browser process in headless mode")
		})
		FlagComps(cursedChromeCmd, func(comp *carapace.ActionMap) {
			(*comp)["payload"] = carapace.ActionFiles("js").Tag("javascript files")
		})
		cursedChromeCmd.Flags().ParseErrorsWhitelist.UnknownFlags = true
		carapace.Gen(cursedChromeCmd).PositionalAnyCompletion(carapace.ActionValues().Usage("additional Chrome CLI arguments"))

		cursedEdgeCmd := &cobra.Command{
			Use:   consts.CursedEdge,
			Short: "Automatically inject a Cursed Chrome payload into a remote Edge extension",
			Long:  help.GetHelpFor([]string{consts.Cursed, consts.CursedEdge}),
			Run: func(cmd *cobra.Command, args []string) {
				cursed.CursedEdgeCmd(cmd, con, args)
			},
		}
		cursedCmd.AddCommand(cursedEdgeCmd)
		Flags("", false, cursedEdgeCmd, func(f *pflag.FlagSet) {
			f.IntP("remote-debugging-port", "r", 0, "remote debugging tcp port (0 = random)")
			f.BoolP("restore", "R", true, "restore the user's session after process termination")
			f.StringP("exe", "e", "", "edge browser executable path (blank string = auto)")
			f.StringP("user-data", "u", "", "user data directory (blank string = auto)")
			f.StringP("payload", "p", "", "cursed chrome payload file path (.js)")
			f.BoolP("keep-alive", "k", false, "keeps browser alive after last browser window closes")
			f.BoolP("headless", "H", false, "start browser process in headless mode")
		})
		FlagComps(cursedEdgeCmd, func(comp *carapace.ActionMap) {
			(*comp)["payload"] = carapace.ActionFiles("js").Tag("javascript files")
		})
		cursedEdgeCmd.Flags().ParseErrorsWhitelist.UnknownFlags = true
		carapace.Gen(cursedEdgeCmd).PositionalAnyCompletion(carapace.ActionValues().Usage("additional Edge CLI arguments"))

		cursedElectronCmd := &cobra.Command{
			Use:   consts.CursedElectron,
			Short: "Curse a remote Electron application",
			Long:  help.GetHelpFor([]string{consts.Cursed, consts.CursedElectron}),
			Run: func(cmd *cobra.Command, args []string) {
				cursed.CursedElectronCmd(cmd, con, args)
			},
		}
		cursedCmd.AddCommand(cursedElectronCmd)
		Flags("", false, cursedElectronCmd, func(f *pflag.FlagSet) {
			f.StringP("exe", "e", "", "remote electron executable absolute path")
			f.IntP("remote-debugging-port", "r", 0, "remote debugging tcp port (0 = random)")
		})
		cursedElectronCmd.Flags().ParseErrorsWhitelist.UnknownFlags = true
		carapace.Gen(cursedElectronCmd).PositionalAnyCompletion(carapace.ActionValues().Usage("additional Electron CLI arguments"))

		CursedCookiesCmd := &cobra.Command{
			Use:   consts.CursedCookies,
			Short: "Dump all cookies from cursed process",
			Long:  help.GetHelpFor([]string{consts.Cursed, consts.CursedCookies}),
			Run: func(cmd *cobra.Command, args []string) {
				cursed.CursedCookiesCmd(cmd, con, args)
			},
		}
		cursedCmd.AddCommand(CursedCookiesCmd)
		Flags("", false, CursedCookiesCmd, func(f *pflag.FlagSet) {
			f.StringP("save", "s", "", "save to file")
		})

		cursedScreenshotCmd := &cobra.Command{
			Use:   consts.ScreenshotStr,
			Short: "Take a screenshot of a cursed process debug target",
			Long:  help.GetHelpFor([]string{consts.Cursed, consts.ScreenshotStr}),
			Run: func(cmd *cobra.Command, args []string) {
				cursed.CursedScreenshotCmd(cmd, con, args)
			},
		}
		cursedCmd.AddCommand(cursedScreenshotCmd)
		Flags("", false, cursedScreenshotCmd, func(f *pflag.FlagSet) {
			f.Int64P("quality", "q", 100, "screenshot quality (1 - 100)")
			f.StringP("save", "s", "", "save to file")
		})

		// [ Wasm ] -----------------------------------------------------------------

		wasmCmd := &cobra.Command{
			Use:     consts.WasmStr,
			Short:   "Execute a Wasm Module Extension",
			Long:    help.GetHelpFor([]string{consts.WasmStr}),
			GroupID: consts.ExecutionHelpGroup,
			Run: func(cmd *cobra.Command, args []string) {
				wasm.WasmCmd(cmd, con, args)
			},
		}
		sliver.AddCommand(wasmCmd)
		Flags("", true, wasmCmd, func(f *pflag.FlagSet) {
			f.Int64P("timeout", "t", defaultTimeout, "grpc timeout in seconds")
		})
		Flags("", false, wasmCmd, func(f *pflag.FlagSet) {
			f.BoolP("pipe", "P", false, "pipe module stdin/stdout/stderr to the current terminal (session only)")
			f.StringP("file", "f", "", "include local file(s) in wasm module's /memfs (glob pattern) ")
			f.StringP("dir", "d", "", "recursively include local directory in wasm module's /memfs (glob pattern)")
			f.BoolP("skip-registration", "s", false, "assume the extension is already registered")
			f.BoolP("loot", "X", false, "save output as loot, incompatible with --pipe")
		})
		FlagComps(wasmCmd, func(comp *carapace.ActionMap) {
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
				wasm.WasmLsCmd(cmd, con, args)
			},
		}
		wasmCmd.AddCommand(wasmLsCmd)

		// [ Post-command declaration setup ]----------------------------------------

		// Everything below this line should preferably not be any command binding
		// (unless you know what you're doing). If there are any final modifications
		// to make to the sliver menu command tree, it time to do them here.

		sliver.InitDefaultHelpCmd()
		sliver.SetHelpCommandGroupID(consts.SliverCoreHelpGroup)

		// Compute which commands should be available based on the current session/beacon.
		con.ExposeCommands()

		return sliver
	}

	return sliverCommands
}
