package exec

import (
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/bishopfox/sliver/client/command/completers"
	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/command/generate"
	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
)

// Commands returns the “ command and its subcommands.
func Commands(con *console.SliverClient) []*cobra.Command {
	executeCmd := &cobra.Command{
		Use:   consts.ExecuteStr,
		Short: "Execute a program on the remote system",
		Long:  help.GetHelpFor([]string{consts.ExecuteStr}),
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExecuteCmd(cmd, con, args)
		},
		GroupID: consts.ExecutionHelpGroup,
	}
	flags.Bind("", false, executeCmd, func(f *pflag.FlagSet) {
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

		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
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
			ExecuteAssemblyCmd(cmd, con, args)
		},
		GroupID:     consts.ExecutionHelpGroup,
		Annotations: flags.RestrictTargets(consts.WindowsCmdsFilter),
	}
	flags.Bind("", false, executeAssemblyCmd, func(f *pflag.FlagSet) {
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

		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
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
			ExecuteShellcodeCmd(cmd, con, args)
		},
		GroupID: consts.ExecutionHelpGroup,
	}
	flags.Bind("", false, executeShellcodeCmd, func(f *pflag.FlagSet) {
		f.BoolP("rwx-pages", "r", false, "Use RWX permissions for memory pages")
		f.Uint32P("pid", "p", 0, "Pid of process to inject into (0 means injection into ourselves)")
		f.StringP("process", "n", `c:\windows\system32\notepad.exe`, "Process to inject into when running in interactive mode")
		f.BoolP("interactive", "i", false, "Inject into a new process and interact with it")
		f.BoolP("shikata-ga-nai", "S", false, "encode shellcode using shikata ga nai prior to execution")
		f.StringP("architecture", "A", "amd64", "architecture of the shellcode: 386, amd64 (used with --shikata-ga-nai flag)")
		f.Uint32P("iterations", "I", 1, "number of encoding iterations (used with --shikata-ga-nai flag)")

		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	completers.NewFlagCompsFor(executeShellcodeCmd, func(comp *carapace.ActionMap) {
		(*comp)["shikata-ga-nai"] = carapace.ActionValues("386", "amd64").Tag("shikata-ga-nai architectures")
	})
	carapace.Gen(executeShellcodeCmd).PositionalCompletion(carapace.ActionFiles().Usage("path to shellcode file (required)"))

	sideloadCmd := &cobra.Command{
		Use:   consts.SideloadStr,
		Short: "Load and execute a shared object (shared library/DLL) in a remote process",
		Long:  help.GetHelpFor([]string{consts.SideloadStr}),
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			SideloadCmd(cmd, con, args)
		},
		GroupID: consts.ExecutionHelpGroup,
	}
	flags.Bind("", false, sideloadCmd, func(f *pflag.FlagSet) {
		f.StringP("entry-point", "e", "", "Entrypoint for the DLL (Windows only)")
		f.StringP("process", "p", `c:\windows\system32\notepad.exe`, "Path to process to host the shellcode")
		f.BoolP("unicode", "w", false, "Command line is passed to unmanaged DLL function in UNICODE format. (default is ANSI)")
		f.BoolP("save", "s", false, "save output to file")
		f.BoolP("loot", "X", false, "save output as loot")
		f.StringP("name", "n", "", "name to assign loot (optional)")
		f.BoolP("keep-alive", "k", false, "don't terminate host process once the execution completes")
		f.Uint32P("ppid", "P", 0, "parent process id (optional)")
		f.StringP("process-arguments", "A", "", "arguments to pass to the hosting process")

		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
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
			SpawnDllCmd(cmd, con, args)
		},
		GroupID:     consts.ExecutionHelpGroup,
		Annotations: flags.RestrictTargets(consts.WindowsCmdsFilter),
	}
	flags.Bind("", false, spawnDllCmd, func(f *pflag.FlagSet) {
		f.StringP("process", "p", `c:\windows\system32\notepad.exe`, "Path to process to host the shellcode")
		f.StringP("export", "e", "ReflectiveLoader", "Entrypoint of the Reflective DLL")
		f.BoolP("save", "s", false, "save output to file")
		f.BoolP("loot", "X", false, "save output as loot")
		f.StringP("name", "n", "", "name to assign loot (optional)")
		f.BoolP("keep-alive", "k", false, "don't terminate host process once the execution completes")
		f.UintP("ppid", "P", 0, "parent process id (optional)")
		f.StringP("process-arguments", "A", "", "arguments to pass to the hosting process")

		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
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
			MigrateCmd(cmd, con, args)
		},
		GroupID:     consts.ExecutionHelpGroup,
		Annotations: flags.RestrictTargets(consts.WindowsCmdsFilter),
	}
	flags.Bind("", false, migrateCmd, func(f *pflag.FlagSet) {
		f.BoolP("disable-sgn", "S", true, "disable shikata ga nai shellcode encoder")
		f.Uint32P("pid", "p", 0, "process id to migrate into")
		f.StringP("process-name", "n", "", "name of the process to migrate into")
		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	carapace.Gen(migrateCmd).PositionalCompletion(carapace.ActionValues().Usage("PID of process to migrate into"))

	psExecCmd := &cobra.Command{
		Use:   consts.PsExecStr,
		Short: "Start a sliver service on a remote target",
		Long:  help.GetHelpFor([]string{consts.PsExecStr}),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			PsExecCmd(cmd, con, args)
		},
		GroupID:     consts.ExecutionHelpGroup,
		Annotations: flags.RestrictTargets(consts.WindowsCmdsFilter),
	}
	flags.Bind("", false, psExecCmd, func(f *pflag.FlagSet) {
		f.StringP("service-name", "s", "Sliver", "name that will be used to register the service")
		f.StringP("service-description", "d", "Sliver implant", "description of the service")
		f.StringP("profile", "p", "", "profile to use for service binary")
		f.StringP("binpath", "b", "c:\\windows\\temp", "directory to which the executable will be uploaded")
		f.StringP("custom-exe", "c", "", "custom service executable to use instead of generating a new Sliver")

		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	completers.NewFlagCompsFor(psExecCmd, func(comp *carapace.ActionMap) {
		(*comp)["custom-exe"] = carapace.ActionFiles()
		(*comp)["profile"] = generate.ProfileNameCompleter(con)
	})
	carapace.Gen(psExecCmd).PositionalCompletion(carapace.ActionValues().Usage("hostname (required)"))

	sshCmd := &cobra.Command{
		Use:   consts.SSHStr,
		Short: "Run a SSH command on a remote host",
		Long:  help.GetHelpFor([]string{consts.SSHStr}),
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			SSHCmd(cmd, con, args)
		},
		GroupID: consts.ExecutionHelpGroup,
	}
	flags.Bind("", false, sshCmd, func(f *pflag.FlagSet) {
		f.UintP("port", "p", 22, "SSH port")
		f.StringP("private-key", "i", "", "path to private key file")
		f.StringP("password", "P", "", "SSH user password")
		f.StringP("login", "l", "", "username to use to connect")
		f.BoolP("skip-loot", "s", false, "skip the prompt to use loot credentials")
		f.StringP("kerberos-config", "c", "/etc/krb5.conf", "path to remote Kerberos config file")
		f.StringP("kerberos-keytab", "k", "", "path to Kerberos keytab file")
		f.StringP("kerberos-realm", "r", "", "Kerberos realm")

		f.Int64P("timeout", "t", flags.DefaultTimeout, "grpc timeout in seconds")
	})
	sshCmd.Flags().ParseErrorsWhitelist.UnknownFlags = true

	completers.NewFlagCompsFor(sshCmd, func(comp *carapace.ActionMap) {
		(*comp)["private-key"] = carapace.ActionFiles()
		(*comp)["kerberos-keytab"] = carapace.ActionFiles()
	})

	carapace.Gen(sshCmd).PositionalCompletion(carapace.ActionValues().Usage("remote host to SSH to (required)"))
	carapace.Gen(sshCmd).PositionalAnyCompletion(carapace.ActionValues().Usage("command line with arguments"))

	return []*cobra.Command{executeCmd, executeAssemblyCmd, executeShellcodeCmd, sideloadCmd, spawnDllCmd, migrateCmd, psExecCmd, sshCmd}
}
