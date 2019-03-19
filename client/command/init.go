package command

import (
	"fmt"
	consts "sliver/client/constants"
	"sliver/client/core"
	"sliver/client/help"

	"github.com/desertbit/grumble"
)

const (
	defaultMTLSLPort = 8888
)

// Init - Bind commands to a App
func Init(app *grumble.App, server *core.SliverServer) {

	app.SetPrintHelp(helpCmd)
	// [ Jobs ] -----------------------------------------------------------------
	app.AddCommand(&grumble.Command{
		Name:     consts.JobsStr,
		Help:     "Job control",
		LongHelp: help.GetHelpFor(consts.JobsStr),
		Flags: func(f *grumble.Flags) {
			f.Int("k", "kill", -1, "kill a background job")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			jobs(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.MtlsStr,
		Help:     "Start an mTLS listener",
		LongHelp: help.GetHelpFor(consts.MtlsStr),
		Flags: func(f *grumble.Flags) {
			f.String("s", "server", "", "interface to bind server to")
			f.Int("l", "lport", defaultMTLSLPort, "tcp listen port")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			startMTLSListener(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.DnsStr,
		Help:     "Start a DNS listener",
		LongHelp: help.GetHelpFor(consts.DnsStr),
		Flags: func(f *grumble.Flags) {
			f.String("d", "domain", "", "parent domain to use for DNS C2")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			startDNSListener(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.HttpStr,
		Help:     "Start a HTTP listener",
		LongHelp: help.GetHelpFor(consts.HttpStr),
		Flags: func(f *grumble.Flags) {
			f.String("d", "domain", "", "limit responses to specific domain")
			f.Int("l", "lport", 80, "tcp listen port")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			startHTTPListener(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.HttpsStr,
		Help:     "Start a HTTPS listener",
		LongHelp: help.GetHelpFor(consts.HttpsStr),
		Flags: func(f *grumble.Flags) {
			f.String("d", "domain", "", "limit responses to specific domain")
			f.Int("l", "lport", 443, "tcp listen port")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			startHTTPSListener(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.PlayersStr,
		Help:     "List players",
		LongHelp: help.GetHelpFor(consts.PlayersStr),
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			playersCmd(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.MultiplayerHelpGroup,
	})

	// [ Commands ] --------------------------------------------------------------

	app.AddCommand(&grumble.Command{
		Name:     consts.SessionsStr,
		Help:     "Session management",
		LongHelp: help.GetHelpFor(consts.SessionsStr),
		Flags: func(f *grumble.Flags) {
			f.String("i", "interact", "", "interact with a sliver")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			sessions(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.BackgroundStr,
		Help:     "Background an active session",
		LongHelp: help.GetHelpFor(consts.BackgroundStr),
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			background(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:      consts.KillStr,
		Help:      "Kill a remote sliver process",
		LongHelp:  help.GetHelpFor(consts.KillStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			kill(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:      consts.InfoStr,
		Help:      "Get info about sliver",
		LongHelp:  help.GetHelpFor(consts.InfoStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			info(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:      consts.UseStr,
		Help:      "Switch the active sliver",
		LongHelp:  help.GetHelpFor(consts.UseStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			use(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.ShellStr,
		Help:     "Start an interactive shell",
		LongHelp: help.GetHelpFor(consts.ShellStr),
		Flags: func(f *grumble.Flags) {
			f.Bool("y", "no-pty", false, "disable use of pty on macos/linux")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			shell(ctx, server)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.GenerateStr,
		Help:     "Generate a sliver binary",
		LongHelp: help.GetHelpFor(consts.GenerateStr),
		Flags: func(f *grumble.Flags) {
			f.String("o", "os", "windows", "operating system")
			f.String("a", "arch", "amd64", "cpu architecture")
			f.Bool("d", "debug", false, "enable debug features")

			f.String("m", "mtls", "", "mtls connection strings")
			f.String("t", "http", "", "http(s) connection strings")
			f.String("n", "dns", "", "dns connection strings")

			f.Int("j", "reconnect", 60, "attempt to reconnect every n second(s)")
			f.Int("k", "max-errors", 1000, "max number of connection errors")

			f.String("w", "limit-datetime", "", "limit execution to before datetime")
			f.Bool("x", "limit-domainjoined", false, "limit execution to domain joined machines")
			f.String("y", "limit-username", "", "limit execution to specified username")
			f.String("z", "limit-hostname", "", "limit execution to specified hostname")

			f.Bool("r", "shared", false, "Build as a shared library (dll/so/dylib)")

			f.String("s", "save", "", "directory/file to the binary to")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			generate(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.NewProfileStr,
		Help:     "Save a new sliver profile",
		LongHelp: help.GetHelpFor(consts.NewProfileStr),
		Flags: func(f *grumble.Flags) {
			f.String("o", "os", "windows", "operating system")
			f.String("a", "arch", "amd64", "cpu architecture")
			f.Bool("d", "debug", false, "enable debug features")

			f.String("m", "mtls", "", "mtls connection strings")
			f.String("t", "http", "", "http(s) connection strings")
			f.String("n", "dns", "", "dns connection strings")

			f.Int("j", "reconnect", 60, "attempt to reconnect every n second(s)")
			f.Int("k", "max-errors", 1000, "max number of connection errors")

			f.String("w", "limit-datetime", "", "limit execution to before datetime")
			f.Bool("x", "limit-domainjoined", false, "limit execution to domain joined machines")
			f.String("y", "limit-username", "", "limit execution to specified username")
			f.String("z", "limit-hostname", "", "limit execution to specified hostname")

			f.Bool("r", "shared", false, "Build as a shared library (dll/so/dylib)")

			f.String("p", "name", "", "profile name")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			newProfile(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.ProfilesStr,
		Help:     "List existing profiles",
		LongHelp: help.GetHelpFor(consts.ProfilesStr),
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			profiles(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.ProfileGenerateStr,
		Help:     "Generate sliver from profile",
		LongHelp: help.GetHelpFor(consts.ProfileGenerateStr),
		Flags: func(f *grumble.Flags) {
			f.String("p", "name", "", "profile name")
			f.String("s", "save", "", "directory/file to the binary to")
		},
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			profileGenerate(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.GenericHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.MsfStr,
		Help:     "Execute an MSF payload in the current process",
		LongHelp: help.GetHelpFor(consts.MsfStr),
		Flags: func(f *grumble.Flags) {
			f.String("m", "payload", "meterpreter_reverse_https", "msf payload")
			f.String("h", "lhost", "", "listen host")
			f.Int("l", "lport", 4444, "listen port")
			f.String("e", "encoder", "", "msf encoder")
			f.Int("i", "iterations", 1, "iterations of the encoder")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			msf(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.MsfInjectStr,
		Help:     "Inject an MSF payload into a process",
		LongHelp: help.GetHelpFor(consts.MsfInjectStr),
		Flags: func(f *grumble.Flags) {
			f.Int("p", "pid", -1, "pid to inject into")
			f.String("m", "payload", "meterpreter_reverse_https", "msf payload")
			f.String("h", "lhost", "", "listen host")
			f.Int("l", "lport", 4444, "listen port")
			f.String("e", "encoder", "", "msf encoder")
			f.Int("i", "iterations", 1, "iterations of the encoder")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			msfInject(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.PsStr,
		Help:     "List remote processes",
		LongHelp: help.GetHelpFor(consts.PsStr),
		Flags: func(f *grumble.Flags) {
			f.Int("p", "pid", -1, "filter based on pid")
			f.String("e", "exe", "", "filter based on executable name")
			f.String("o", "owner", "", "filter based on owner")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			ps(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:      consts.PingStr,
		Help:      "Test connection to sliver",
		LongHelp:  help.GetHelpFor(consts.PingStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			ping(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.GetPIDStr,
		Help:     "Get sliver pid",
		LongHelp: help.GetHelpFor(consts.GetPIDStr),
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			getPID(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.GetUIDStr,
		Help:     "Get sliver UID",
		LongHelp: help.GetHelpFor(consts.GetUIDStr),
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			getUID(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.GetGIDStr,
		Help:     "Get sliver GID",
		LongHelp: help.GetHelpFor(consts.GetGIDStr),
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			getGID(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.WhoamiStr,
		Help:     "Get sliver user",
		LongHelp: help.GetHelpFor(consts.WhoamiStr),
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			whoami(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.LsStr,
		Help:     "List current directory",
		LongHelp: help.GetHelpFor(consts.LsStr),
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			ls(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:      consts.RmStr,
		Help:      "Remove a file or directory",
		LongHelp:  help.GetHelpFor(consts.RmStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			rm(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:      consts.MkdirStr,
		Help:      "Make a directory",
		LongHelp:  help.GetHelpFor(consts.MkdirStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			mkdir(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:      consts.CdStr,
		Help:      "Change directory",
		LongHelp:  help.GetHelpFor(consts.CdStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			cd(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.PwdStr,
		Help:     "Print working directory",
		LongHelp: help.GetHelpFor(consts.PwdStr),
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			pwd(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:      consts.CatStr,
		Help:      "Dump file to stdout",
		LongHelp:  help.GetHelpFor(consts.CatStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			cat(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:      consts.DownloadStr,
		Help:      "Download a file",
		LongHelp:  help.GetHelpFor(consts.DownloadStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			download(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:      consts.UploadStr,
		Help:      "Upload a file",
		LongHelp:  help.GetHelpFor(consts.UploadStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			upload(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.ProcdumpStr,
		Help:     "Dump process memory",
		LongHelp: help.GetHelpFor(consts.ProcdumpStr),
		Flags: func(f *grumble.Flags) {
			f.Int("p", "pid", -1, "target pid")
			f.String("n", "name", "", "target process name")
			f.Int("t", "timeout", 360, "command timeout in seconds")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			procdump(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.ImpersonateStr,
		Help:     "Run a new process in the context of the designated user",
		LongHelp: help.GetHelpFor(consts.ImpersonateStr),
		Flags: func(f *grumble.Flags) {
			f.String("u", "username", "NT AUTHORITY\\SYSTEM", "user to impersonate")
			f.String("p", "process", "", "process to start")
			f.String("a", "args", "", "arguments for the process")
		},
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			impersonate(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverWinHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.ElevateStr,
		Help:     "Spawns a new sliver session as an elevated process (UAC bypass)",
		LongHelp: help.GetHelpFor(consts.ElevateStr),
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			elevate(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverWinHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:     consts.GetSystemStr,
		Help:     "Spawns a new sliver session as the NT AUTHORITY\\SYSTEM user",
		LongHelp: help.GetHelpFor(consts.GetSystemStr),
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			getsystem(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverWinHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:      consts.ExecuteAssemblyStr,
		Help:      "Load and executes a .NET assembly in a child process",
		LongHelp:  help.GetHelpFor(consts.ExecuteAssemblyStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			executeAssembly(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverWinHelpGroup,
	})

	app.AddCommand(&grumble.Command{
		Name:      consts.ExecuteShellcodeStr,
		Help:      "Executes the given shellcode in the sliver process",
		LongHelp:  help.GetHelpFor(consts.ExecuteShellcodeStr),
		AllowArgs: true,
		Run: func(ctx *grumble.Context) error {
			fmt.Println()
			executeShellcode(ctx, server.RPC)
			fmt.Println()
			return nil
		},
		HelpGroup: consts.SliverHelpGroup,
	})
}
