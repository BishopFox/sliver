package windows

import (
	"github.com/jessevdk/go-flags"

	"github.com/bishopfox/sliver/client/constants"
	cctx "github.com/bishopfox/sliver/client/context"
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
)

// BindCommands - Binds Windows-specific commands for Windows-based Sliver session.
func BindCommands(parser *flags.Parser) {

	// The context package checks, handles and reports any error arising from a struct
	// being registered as a command, and saves it in various group related things.
	// The following call is the contextual counterpart of RegisterServerCommand.
	var register = cctx.Commands.RegisterSliverCommand

	// Proc
	m, err := parser.AddCommand(constants.MigrateStr,
		"Migrate into a remote host process", "",
		&Migrate{})
	register(err, m, constants.ProcGroup)

	// Priv
	i, err := parser.AddCommand(constants.ImpersonateStr,
		"Impersonate a logged in user", "",
		&Impersonate{})
	register(err, i, constants.PrivGroup)

	rs, err := parser.AddCommand(constants.RevToSelfStr,
		"Revert to self: lose stolen Windows token", "",
		&Rev2Self{})
	register(err, rs, constants.PrivGroup)

	gs, err := parser.AddCommand(constants.GetSystemStr,
		"Spawns a new sliver session as the NT AUTHORITY\\SYSTEM user ", "",
		&GetSystem{})
	register(err, gs, constants.PrivGroup)

	mt, err := parser.AddCommand(constants.MakeTokenStr,
		"Create a new Logon Session with the specified credentials", "",
		&MakeToken{})
	register(err, mt, constants.PrivGroup)

	// Execution
	ea, err := parser.AddCommand(constants.ExecuteAssemblyStr,
		"Loads and executes a .NET assembly in a child process", "",
		&ExecuteAssembly{})
	register(err, ea, constants.ExecuteGroup)

	sd, err := parser.AddCommand(constants.SpawnDllStr,
		"Load and execute a Reflective DLL in a remote process", "",
		&SpawnDLL{})
	register(err, sd, constants.ExecuteGroup)

	ra, err := parser.AddCommand(constants.RunAsStr,
		"Run a new process in the context of the designated user", "",
		&RunAs{})
	register(err, ra, constants.ExecuteGroup)

	// Persistence
	ss, err := parser.AddCommand(constants.PsExecStr,
		"Start a sliver service on the session target", "",
		&Service{})
	register(err, ss, constants.PersistenceGroup)

	bi, err := parser.AddCommand(constants.BackdoorStr,
		"Infect a remote file with a sliver shellcode", "",
		&Backdoor{})
	register(err, bi, constants.PersistenceGroup)

	reg, err := parser.AddCommand(constants.RegistryStr,
		"Windows Registry management commands",
		help.GetHelpFor(constants.RegistryStr),
		&Registry{})
	register(err, reg, constants.PersistenceGroup)

	if reg != nil {
		_, err := reg.AddCommand(constants.RegistryReadStr,
			"Read values from the Windows Registry",
			help.GetHelpFor(constants.RegistryReadStr),
			&RegistryRead{})
		register(err, nil, constants.PersistenceGroup)

		_, err = reg.AddCommand(constants.RegistryWriteStr,
			"Write values to the Windows Registry",
			help.GetHelpFor(constants.RegistryWriteStr),
			&RegistryWrite{})
		register(err, nil, constants.PersistenceGroup)

		_, err = reg.AddCommand(constants.RegistryCreateKeyStr,
			"Create a Registry key",
			help.GetHelpFor(constants.RegistryCreateKeyStr),
			&RegistryCreateKey{})
		register(err, nil, constants.PersistenceGroup)

	}
	return
}
