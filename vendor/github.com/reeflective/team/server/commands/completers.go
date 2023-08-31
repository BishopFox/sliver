package commands

/*
   team - Embedded teamserver for Go programs and CLI applications
   Copyright (C) 2023 Reeflective

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
	"net"
	"strings"

	"github.com/rsteube/carapace"

	"github.com/reeflective/team/client"
	"github.com/reeflective/team/server"
)

// interfacesCompleter completes interface addresses on the client host.
func interfacesCompleter() carapace.Action {
	return carapace.ActionCallback(func(_ carapace.Context) carapace.Action {
		ifaces, err := net.Interfaces()
		if err != nil {
			return carapace.ActionMessage("failed to get net interfaces: %s", err.Error())
		}

		results := make([]string, 0)

		for _, i := range ifaces {
			addrs, err := i.Addrs()
			if err != nil {
				continue
			}

			for _, a := range addrs {
				switch ipType := a.(type) {
				case *net.IPAddr:
					results = append(results, ipType.IP.String())
				case *net.IPNet:
					results = append(results, ipType.IP.String())
				default:
					results = append(results, ipType.String())
				}
			}
		}

		return carapace.ActionValues(results...).Tag("client interfaces").NoSpace(':')
	})
}

// userCompleter completes usernames of the application teamserver.
func userCompleter(client *client.Client, server *server.Server) carapace.CompletionCallback {
	return func(c carapace.Context) carapace.Action {
		users, err := client.Users()
		if err != nil {
			return carapace.ActionMessage("Failed to get users: %s", err)
		}

		results := make([]string, len(users))
		for i, user := range users {
			results[i] = strings.TrimSpace(user.Name)
		}

		if len(results) == 0 {
			return carapace.ActionMessage(fmt.Sprintf("%s teamserver has no users", server.Name()))
		}

		return carapace.ActionValues(results...).Tag(fmt.Sprintf("%s teamserver users", server.Name()))
	}
}

// listenerIDCompleter completes ID for running teamserver listeners.
func listenerIDCompleter(client *client.Client, server *server.Server) carapace.CompletionCallback {
	return func(c carapace.Context) carapace.Action {
		listeners := server.Listeners()
		cfg := server.GetConfig()

		var results []string
		for _, ln := range listeners {
			results = append(results, strings.TrimSpace(formatSmallID(ln.ID)))
			results = append(results, fmt.Sprintf("[%s] (%s)", ln.Description, "Up"))
		}

		var persistents []string
	next:
		for _, saved := range cfg.Listeners {

			for _, ln := range listeners {
				if saved.ID == ln.ID {
					continue next
				}
			}

			persistents = append(persistents, strings.TrimSpace(formatSmallID(saved.ID)))

			host := fmt.Sprintf("%s:%d", saved.Host, saved.Port)
			persistents = append(persistents, fmt.Sprintf("[%s] (%s)", host, "Up"))
		}

		if len(results) == 0 && len(persistents) == 0 {
			return carapace.ActionMessage(fmt.Sprintf("no listeners running/saved for %s teamserver", server.Name()))
		}

		// return carapace.
		return carapace.Batch(
			carapace.ActionValuesDescribed(results...).Tag("active teamserver listeners"),
			carapace.ActionValuesDescribed(persistents...).Tag("saved teamserver listeners"),
		).ToA()
	}
}

// listenerTypeCompleter completes the different types of teamserver listener/handler stacks available.
func listenerTypeCompleter(client *client.Client, server *server.Server) carapace.CompletionCallback {
	return func(c carapace.Context) carapace.Action {
		listeners := server.Handlers()

		var results []string
		for _, ln := range listeners {
			results = append(results, strings.TrimSpace(ln.Name()))
		}

		if len(results) == 0 {
			return carapace.ActionMessage(fmt.Sprintf("no additional listener types for %s teamserver", server.Name()))
		}

		return carapace.ActionValues(results...).Tag(fmt.Sprintf("%s teamserver listener types", server.Name()))
	}
}
