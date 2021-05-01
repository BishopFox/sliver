package c2

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

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
	"github.com/maxlandon/gonsole"

	"github.com/bishopfox/sliver/client/completion"
	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/help"
)

var (
	// Console Some commands might need to access the current context
	// in the course of the application execution.
	Console *gonsole.Console

	// Most commands just need access to a precise context.
	serverMenu *gonsole.Menu
)

// BindCommands - C2 transports might be available in either or both contexts.
// For now, there is a clear seggregation, and server listeners can only be spawned from the server context.
func BindCommands(cc *gonsole.Menu) {

	switch cc.Name {
	// ----------------------------------------------------------------------------------------------
	// All C2 transports that can listen on/ dial from the server.
	// ----------------------------------------------------------------------------------------------
	case constants.ServerMenu:
		// C2 listeners -----------------------------------------------------------------
		mtls := cc.AddCommand(constants.MtlsStr,
			"Start an mTLS listener on the server, or on a routed session",
			help.GetHelpFor(constants.MtlsStr),
			constants.TransportsGroup,
			[]string{""},
			func() interface{} { return &MTLSListener{} })
		mtls.AddOptionCompletion("LHost", completion.ServerInterfaceAddrs)

		cc.AddCommand(constants.WGStr,
			"Start a WireGuard listener",
			help.GetHelpFor(constants.WGStr),
			constants.TransportsGroup,
			[]string{""},
			func() interface{} { return &WireGuardListener{} })

		cc.AddCommand(constants.DnsStr,
			"Start a DNS listener on the server",
			help.GetHelpFor(constants.DnsStr),
			constants.TransportsGroup,
			[]string{""},
			func() interface{} { return &DNSListener{} })

		https := cc.AddCommand(constants.HttpsStr,
			"Start an HTTP(S) listener on the server",
			help.GetHelpFor(constants.HttpsStr),
			constants.TransportsGroup,
			[]string{""},
			func() interface{} { return &HTTPSListener{} })
		https.AddOptionCompletion("Domain", completion.ServerInterfaceAddrs)
		https.AddOptionCompletionDynamic("Certificate", Console.Completer.LocalPathAndFiles)
		https.AddOptionCompletionDynamic("PrivateKey", Console.Completer.LocalPathAndFiles)

		http := cc.AddCommand(constants.HttpStr,
			"Start an HTTP listener on the server",
			help.GetHelpFor(constants.HttpStr),
			constants.TransportsGroup,
			[]string{""},
			func() interface{} { return &HTTPListener{} })
		http.AddOptionCompletion("LHost", completion.ServerInterfaceAddrs)

		stager := cc.AddCommand(constants.StageListenerStr,
			"Start a staging listener (TCP/HTTP/HTTPS), bound to a Sliver profile",
			help.GetHelpFor(constants.StageListenerStr),
			constants.TransportsGroup,
			[]string{""},
			func() interface{} { return &StageListener{} })
		stager.AddOptionCompletionDynamic("URL", completion.NewURLCompleterStager().CompleteURL)
		stager.AddOptionCompletionDynamic("Certificate", Console.Completer.LocalPathAndFiles)
		stager.AddOptionCompletionDynamic("PrivateKey", Console.Completer.LocalPathAndFiles)
		stager.AddOptionCompletion("Profile", completion.ImplantProfiles)

		// Websites -----------------------------------------------------------------
		ws := cc.AddCommand(constants.WebsitesStr,
			"Manage websites (used with HTTP C2) (prints website name argument by default)",
			"",
			constants.TransportsGroup,
			[]string{""},
			func() interface{} { return &Websites{} })

		ws.SubcommandsOptional = true

		ws.AddCommand(constants.WebsitesShowStr,
			"Print the contents of a website",
			"",
			constants.TransportsGroup,
			[]string{""},
			func() interface{} { return &WebsitesShow{} })

		ws.AddCommand(constants.RmStr,
			"Remove an entire website",
			"", "", []string{""},
			func() interface{} { return &WebsitesDelete{} })

		wa := ws.AddCommand(constants.AddWebContentStr,
			"Add content to a website",
			"", "", []string{""},
			func() interface{} { return &WebsitesAddContent{} })
		wa.AddOptionCompletionDynamic("Content", Console.Completer.LocalPathAndFiles)

		wd := ws.AddCommand(constants.RmWebContentStr,
			"Remove content from a website",
			"", "", []string{""},
			func() interface{} { return &WebsitesDeleteContent{} })
		wd.AddOptionCompletionDynamic("Content", Console.Completer.LocalPathAndFiles)

		wu := ws.AddCommand(constants.WebUpdateStr,
			"Update a website's content type",
			"", "", []string{""},
			func() interface{} { return &WebsiteType{} })
		wu.AddOptionCompletionDynamic("Content", Console.Completer.LocalPathAndFiles)

	// ----------------------------------------------------------------------------------------------
	// All C2 transports that can listen on/ dial from the implant.
	// ----------------------------------------------------------------------------------------------
	case constants.SliverMenu:
		// C2 listeners -----------------------------------------------------------------
		tcp := cc.AddCommand(constants.TCPListenerStr,
			"Start a TCP pivot listener (unencrypted!)",
			"",
			constants.TransportsGroup,
			[]string{""},
			func() interface{} { return &TCPPivot{} })
		tcp.AddOptionCompletion("LHost", completion.ActiveSessionIfaceAddrs)

		cc.AddCommand(constants.NamedPipeStr,
			"Start a named pipe pivot listener",
			"",
			constants.TransportsGroup,
			[]string{"windows"}, // Command is only available if the sliver host OS is Windows
			func() interface{} { return &NamedPipePivot{} })
	}
}
