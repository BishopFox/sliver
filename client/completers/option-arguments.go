package completers

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
	"strings"

	"github.com/jessevdk/go-flags"

	"github.com/bishopfox/sliver/client/readline"

	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/context"
)

// completeOptionArguments - Completes all values for arguments to a command. Arguments here are different from command options (--option).
// Many categories, from multiple sources in multiple contexts
func completeOptionArguments(cmd *flags.Command, opt *flags.Option, lastWord string) (prefix string, completions []*readline.CompletionGroup) {

	// By default the last word is the prefix
	prefix = lastWord

	var comp *readline.CompletionGroup // This group is used as a buffer, to add groups to final completions

	// First of all: some options, no matter their contexts and subject, have default values.
	// When we have such an option, we don't bother analyzing context, we just build completions and return.
	if len(opt.Choices) > 0 {
		comp = &readline.CompletionGroup{
			Name:        opt.ValueName, // Value names are specified in struct metadata fields
			DisplayType: readline.TabDisplayGrid,
		}
		for _, choice := range opt.Choices {
			if strings.HasPrefix(choice, lastWord) {
				comp.Suggestions = append(comp.Suggestions, choice)
			}
		}
		completions = append(completions, comp)
		return
	}

	// Depends first on the current menu context.
	// We have a different problem here than for command arguments: options
	// may pertain to different contexts, no matter the context in which we
	// are when using this option (module options, for instance).
	// Therefore the filtering is a bit different, involved.
	// We have 3 words, potentially different, with which we can filter:
	//
	// 1) '--option-name' is the string typed as input.
	// 2) 'OptionName' is the name of the struct/type for this option.
	// 3) 'ValueName' is the name of the value we expect.
	var match = func(name string) bool {
		if strings.Contains(opt.Field().Name, name) {
			return true
		}
		return false
	}

	switch context.Context.Menu {
	case context.Server:
		// Sessions
		if match("ImplantID") || match("SessionID") {
			completions = append(completions, sessionIDs(lastWord))
		}

		// Any arguments with a path name. Often we "save" files that need paths, certificates, etc
		if match("Path") || match("Save") || match("Certificate") || match("PrivateKey") {
			switch cmd.Name {
			case constants.WebContentTypeStr, constants.WebUpdateStr, constants.AddWebContentStr, constants.RmWebContentStr:
				// Make an exception for WebPath option in websites commands.
			default:
				switch opt.ValueName {
				case "local-path", "path":
					prefix, comp = completeLocalPath(lastWord)
					completions = append(completions, comp)
				case "local-file", "file":
					prefix, comp = completeLocalPathAndFiles(lastWord)
					completions = append(completions, comp)
				default:
					// We always have a default searching for files, locally
					prefix, comp = completeLocalPathAndFiles(lastWord)
					completions = append(completions, comp)
				}

			}
		}

		// Local host: client/server interfaces, routed implant interfaces
		// We include generate --c2 protocol strings (ex: --mtls)
		if match("LHost") {
			// Locally, we add IPv4/IPv6 interfaces
			completions = append(completions, clientInterfaceAddrs(lastWord, true))

			// All implant host addresses reachable through a route.
			completions = append(completions, routedSessionIfaceAddrs(lastWord, 0, true)...)
		}
		if match("MTLS") || match("HTTP") || match("DNS") || match("NamedPipe") {
			// Locally, we add IPv4/IPv6 interfaces
			completions = append(completions, clientInterfaceAddrs(lastWord, false))

			// All implant host addresses reachable through a route.
			completions = append(completions, routedSessionIfaceAddrs(lastWord, 0, false)...)
		}

		// Remote hosts: server interfaces, implant local and public
		if match("RHost") {
			completions = append(completions, allSessionIfaceAddrs(lastWord, 0, true)...)
		}

		// Network/CIDR:
		if match("Network") || match("CIDR") || match("Subnet") {
			completions = append(completions, allSessionsIfaceNetworks(lastWord, 0, true)...)
		}

		// Websites
		if match("Content") {
			prefix, comp = completeLocalPathAndFiles(lastWord)
			completions = append(completions, comp)
		}

		// Implant profiles
		if match("Profile") && cmd.Name != constants.NewProfileStr {
			completions = append(completions, implantProfiles(lastWord))
		}

		// URL completer (we only pass valid schemes for a given command)
		if match("URL") && cmd.Name != constants.StageListenerStr {
			urlPrefix, comps := completeURL(lastWord, false, urlShemes)
			completions = append(completions, comps...)

			// We return this prefix because it is aware of paths lengths, etc...
			return urlPrefix, completions

		} else if match("URL") && cmd.Name == constants.StageListenerStr {
			urlPrefix, comps := completeURL(lastWord, false, stageListenerProtocols)
			completions = append(completions, comps...)

			// We return this prefix because it is aware of paths lengths, etc...
			return urlPrefix, completions
		}

	case context.Sliver:
		sliverID := context.Context.Sliver.Session.ID

		// Any arguments with a path name. Often we "save" files that need paths, certificates, etc
		if match("Save") {
			switch cmd.Name {
			case constants.WebContentTypeStr, constants.WebUpdateStr, constants.AddWebContentStr, constants.RmWebContentStr:
				// Make an exception for WebPath option in websites commands.
			default:
				switch opt.ValueName {
				case "local-path", "path":
					prefix, comp = completeLocalPath(lastWord)
					completions = append(completions, comp)
				case "local-file", "file":
					prefix, comp = completeLocalPathAndFiles(lastWord)
					completions = append(completions, comp)
				default:
					// We always have a default searching for files, locally
					prefix, comp = completeLocalPathAndFiles(lastWord)
					completions = append(completions, comp)
				}
			}
		}

		// Remote paths
		if match("RemotePath") {
			switch cmd.Name {
			case constants.ExecuteShellcodeStr:
				prefix, comp = completeRemotePathAndFiles(lastWord)
				completions = append(completions, comp)
			default:
				prefix, comp = completeRemotePathAndFiles(lastWord)
				completions = append(completions, comp)
			}
		}

		// Sessions
		if match("ImplantID") || match("SessionID") {
			completions = append(completions, sessionIDs(lastWord))
		}

		if match("LHost") {
			// Client IPv4/IPv6 interfaces for port forwarders
			if cmd.Name == constants.PortfwdOpenStr {
				completions = append(completions, clientInterfaceAddrs(lastWord, true))
			} else {
				// Else we are starting a handler, or something like. Separate group for active session
				completions = append(completions, sessionIfaceAddrs(lastWord, context.Context.Sliver.Session, true))
			}

			// Remaining routed implants
			completions = append(completions, routedSessionIfaceAddrs(lastWord, sliverID, true)...)
		}

		if match("RHost") {
			// Separate group for active session
			completions = append(completions, sessionIfaceAddrs(lastWord, context.Context.Sliver.Session, true))

			// This might be used by port forwarders (need local loopbacks)
			completions = append(completions, allSessionIfaceAddrs(lastWord, sliverID, true)...)
		}

		// Implant builds & profiles
		if match("Profile") {
			completions = append(completions, implantProfiles(lastWord))
		}

		// Network/CIDR:
		if match("Network") || match("CIDR") || match("Subnet") {
			completions = append(completions, allSessionsIfaceNetworks(lastWord, 0, true)...)
		}

		// Implant profiles
		if match("Profile") && cmd.Name != constants.NewProfileStr {
			completions = append(completions, implantProfiles(lastWord))
		}

		// URL completer
		if match("URL") {
			urlPrefix, comps := completeURL(lastWord, true, urlShemes)
			completions = append(completions, comps...)

			// We return this prefix because it is aware of paths lengths, etc...
			return urlPrefix, completions
		}

		// Processes
		if match("PID") {
			completions = append(completions, processes(lastWord))
		}

	}
	return
}
