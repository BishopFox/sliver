package windows

import (
	"github.com/maxlandon/gonsole"

	"github.com/bishopfox/sliver/client/completion"
	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/help"
)

const (
	// ANSI Colors
	normal    = "\033[0m"
	black     = "\033[30m"
	red       = "\033[31m"
	green     = "\033[32m"
	orange    = "\033[33m"
	blue      = "\033[34m"
	purple    = "\033[35m"
	cyan      = "\033[36m"
	gray      = "\033[37m"
	bold      = "\033[1m"
	clearln   = "\r\x1b[2K"
	upN       = "\033[%dA"
	downN     = "\033[%dB"
	underline = "\033[4m"

	// Info - Display colorful information
	Info = bold + cyan + "[*] " + normal
	// Debug - Display debug information
	Debug = bold + purple + "[-] " + normal
	// Error - Notify error to a user
	Error = bold + red + "[!] " + normal
	// Warning - Notify important information, not an error
	Warning = bold + orange + "[!] " + normal
	// Woot - Display success
	Woot = bold + green + "[$] " + normal
)

var (
	// Console Some commands might need to access the current context
	// in the course of the application execution.
	Console *gonsole.Console

	// Most commands just need access to a precise context.
	sliverMenu *gonsole.Menu
)

// BindCommands - Binds Windows-specific commands for Windows-based Sliver session.
func BindCommands(cc *gonsole.Menu) {

	// Keep a reference to this context, command implementations might want to use it.
	sliverMenu = cc

	// Proc -------------------------------------------------------------------------------
	migrate := cc.AddCommand(constants.MigrateStr,
		"Migrate into a remote host process", "",
		constants.ProcGroup,
		[]string{constants.SliverWinHelpGroup},
		func() interface{} { return &Migrate{} })
	migrate.AddArgumentCompletion("PID", completion.SessionProcesses)

	// Priv -------------------------------------------------------------------------------
	cc.AddCommand(constants.ImpersonateStr,
		"Impersonate a logged in user", "",
		constants.PrivGroup,
		[]string{constants.SliverWinHelpGroup},
		func() interface{} { return &Impersonate{} })

	cc.AddCommand(constants.RevToSelfStr,
		"Revert to self: lose stolen Windows token", "",
		constants.PrivGroup,
		[]string{constants.SliverWinHelpGroup},
		func() interface{} { return &Rev2Self{} })

	cc.AddCommand(constants.GetSystemStr,
		"Spawns a new sliver session as the NT AUTHORITY\\SYSTEM user ", "",
		constants.PrivGroup,
		[]string{constants.SliverWinHelpGroup},
		func() interface{} { return &GetSystem{} })

	cc.AddCommand(constants.MakeTokenStr,
		"Create a new Logon Session with the specified credentials", "",
		constants.PrivGroup,
		[]string{constants.SliverWinHelpGroup},
		func() interface{} { return &MakeToken{} })

	// Execution --------------------------------------------------------------------------
	execAssembly := cc.AddCommand(constants.ExecuteAssemblyStr,
		"Loads and executes a .NET assembly in a child process", "",
		constants.ExecuteGroup,
		[]string{constants.SliverWinHelpGroup},
		func() interface{} { return &ExecuteAssembly{} })
	execAssembly.AddArgumentCompletionDynamic("LocalPath", Console.Completer.LocalPathAndFiles)
	execAssembly.AddArgumentCompletionDynamic("Args", completion.CompleteRemotePathAndFiles)
	execAssembly.AddOptionCompletionDynamic("Path", completion.CompleteRemotePathAndFiles)
	execAssembly.AddOptionCompletionDynamic("Save", Console.Completer.LocalPath)
	execAssembly.AddOptionCompletion("Arch", completion.CompleteAssemblyArchs)

	spawnDll := cc.AddCommand(constants.SpawnDllStr,
		"Load and execute a Reflective DLL in a remote process", "",
		constants.ExecuteGroup,
		[]string{constants.SliverWinHelpGroup},
		func() interface{} { return &SpawnDLL{} })
	spawnDll.AddArgumentCompletionDynamic("LocalPath", Console.Completer.LocalPathAndFiles)
	spawnDll.AddArgumentCompletionDynamic("Args", completion.CompleteRemotePathAndFiles)
	spawnDll.AddOptionCompletionDynamic("Save", Console.Completer.LocalPath)

	cc.AddCommand(constants.RunAsStr,
		"Run a new process in the context of the designated user", "",
		constants.ExecuteGroup,
		[]string{constants.SliverWinHelpGroup},
		func() interface{} { return &RunAs{} })

	// Persistence ------------------------------------------------------------------------
	cc.AddCommand(constants.PsExecStr,
		"Start a sliver service on the session target", "",
		constants.PersistenceGroup,
		[]string{constants.SliverWinHelpGroup},
		func() interface{} { return &Service{} })

	backdoor := cc.AddCommand(constants.BackdoorStr,
		"Infect a remote file with a sliver shellcode", "",
		constants.PersistenceGroup,
		[]string{constants.SliverWinHelpGroup},
		func() interface{} { return &Backdoor{} })
	backdoor.AddArgumentCompletionDynamic("RemotePath", completion.CompleteRemotePathAndFiles)

	reg := cc.AddCommand(constants.RegistryStr,
		"Windows Registry management commands",
		help.GetHelpFor(constants.RegistryStr),
		constants.PersistenceGroup,
		[]string{constants.SliverWinHelpGroup},
		func() interface{} { return &Registry{} })

	reg.AddCommand(constants.RegistryReadStr,
		"Read values from the Windows Registry",
		help.GetHelpFor(constants.RegistryReadStr),
		"", []string{constants.SliverWinHelpGroup},
		func() interface{} { return &RegistryRead{} })

	reg.AddCommand(constants.RegistryWriteStr,
		"Write values to the Windows Registry",
		help.GetHelpFor(constants.RegistryWriteStr),
		"", []string{constants.SliverWinHelpGroup},
		func() interface{} { return &RegistryWrite{} })

	reg.AddCommand(constants.RegistryCreateKeyStr,
		"Create a Registry key",
		help.GetHelpFor(constants.RegistryCreateKeyStr),
		"", []string{constants.SliverWinHelpGroup},
		func() interface{} { return &RegistryCreateKey{} })
}
