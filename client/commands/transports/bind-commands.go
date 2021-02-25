package transports

import (
	"fmt"

	"github.com/jessevdk/go-flags"
	"github.com/bishopfox/sliver/client/readline"
	"gopkg.in/AlecAivazis/survey.v1"

	"github.com/bishopfox/sliver/client/constants"
	cctx "github.com/bishopfox/sliver/client/context"
	"github.com/bishopfox/sliver/client/help"
	"github.com/bishopfox/sliver/client/util"
)

// registerError - Format an error arising from binding a command. These errors generally
// happen because badly formatted metadata has been added in the command struct.
func registerError(command string, err error) (errs []error) {
	if err != nil {
		nerr := fmt.Errorf(util.CommandError+" %s%s%s - %s\n",
			readline.BOLD, command, readline.RESET, err.Error())
		errs = append(errs, nerr)
	}
	return errs
}

// BindCommands - All C2/transports related commands, are bound in a given context
// when the implant supports such commands: ex, if an implant is able to use MTLS
// pivot listeners, the mtls command will also appear in the Sliver context
func BindCommands(parser *flags.Parser, registerGroup func(err error, cmd *flags.Command, group string)) {

	// Commands in both contexts
	m, err := parser.AddCommand(constants.MtlsStr,
		"Start an mTLS listener on the server, or on a routed session",
		help.GetHelpFor(constants.MtlsStr),
		&MTLSListener{})
	registerGroup(err, m, constants.TransportsGroup)

	switch cctx.Context.Menu {

	case cctx.Server:
		// C2 listeners -----------------------------------------------------------------
		d, err := parser.AddCommand(constants.DnsStr,
			"Start a DNS listener on the server",
			help.GetHelpFor(constants.DnsStr),
			&DNSListener{})
		registerGroup(err, d, constants.TransportsGroup)

		hs, err := parser.AddCommand(constants.HttpsStr,
			"Start an HTTP(S) listener on the server",
			help.GetHelpFor(constants.HttpsStr),
			&HTTPSListener{})
		registerGroup(err, hs, constants.TransportsGroup)

		h, err := parser.AddCommand(constants.HttpStr,
			"Start an HTTP listener on the server",
			help.GetHelpFor(constants.HttpStr),
			&HTTPListener{})
		registerGroup(err, h, constants.TransportsGroup)

		s, err := parser.AddCommand(constants.StageListenerStr,
			"Start a staging listener (TCP/HTTP/HTTPS), bound to a Sliver profile",
			help.GetHelpFor(constants.StageListenerStr),
			&StageListener{})
		registerGroup(err, s, constants.TransportsGroup)

		// Websites -----------------------------------------------------------------
		ws, err := parser.AddCommand(constants.WebsitesStr,
			"Manage websites (used with HTTP C2) (prints website name argument by default)", "",
			&Websites{})
		registerGroup(err, ws, constants.TransportsGroup)

		if ws != nil {
			ws.SubcommandsOptional = true

			_, err = ws.AddCommand(constants.RmStr,
				"Remove an entire website", "",
				&WebsitesDelete{})
			registerGroup(err, nil, constants.TransportsGroup)

			_, err = ws.AddCommand(constants.AddWebContentStr,
				"Add content to a website", "",
				&WebsitesAddContent{})
			registerGroup(err, nil, constants.TransportsGroup)

			_, err = ws.AddCommand(constants.RmWebContentStr,
				"Remove content from a website", "",
				&WebsitesDeleteContent{})
			registerGroup(err, nil, constants.TransportsGroup)

			_, err = ws.AddCommand(constants.WebUpdateStr,
				"Update a website's content type", "",
				&WebsiteType{})
			registerGroup(err, nil, constants.TransportsGroup)
		}

	case cctx.Sliver:
		tp, err := parser.AddCommand(constants.TCPListenerStr,
			"Start a TCP pivot listener (unencrypted!)", "",
			&TCPPivot{})
		registerGroup(err, tp, constants.TransportsGroup)

		// add Named Pipes if the sliver host OS is Windows
		if cctx.Context.Sliver.OS == "windows" {
			np, err := parser.AddCommand(constants.NamedPipeStr,
				"Start a named pipe pivot listener", "",
				&NamedPipePivot{})
			registerGroup(err, np, constants.TransportsGroup)
		}
	}

	return
}

// This should be called for any dangerous (OPSEC-wise) functions
func isUserAnAdult() bool {
	confirm := false
	prompt := &survey.Confirm{Message: "This action is bad OPSEC, are you an adult?"}
	survey.AskOne(prompt, &confirm, nil)
	return confirm
}
