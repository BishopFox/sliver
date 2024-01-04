package completers

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"net"

	"github.com/rsteube/carapace"
)

// ClientInterfacesCompleter completes interface addresses on the client host.
func ClientInterfacesCompleter() carapace.Action {
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
				switch v := a.(type) {
				case *net.IPAddr:
					results = append(results, v.IP.String())
				case *net.IPNet:
					results = append(results, v.IP.String())
				default:
					results = append(results, v.String())
				}
			}
		}

		return carapace.ActionValues(results...).Tag("client interfaces").NoSpace(':')
	})
}

// LocalProxyCompleter gives URL completion to all flags/arguments that accept a client proxy address.
func LocalProxyCompleter() carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		prefix := ""

		hostPort := carapace.ActionMultiParts(":", func(c carapace.Context) carapace.Action {
			switch len(c.Parts) {
			case 0:
				return ClientInterfacesCompleter()
			case 1:
				return carapace.ActionMessage("server port")
			default:
				return carapace.ActionValues()
			}
		})

		return carapace.ActionMultiParts("://", func(c carapace.Context) carapace.Action {
			switch len(c.Parts) {
			case 0:
				return carapace.ActionValues("http", "https").Tag("proxy protocols").Suffix("://")
			case 1:
				return hostPort
			default:
				return carapace.ActionValues()
			}
		}).Invoke(c).Prefix(prefix).ToA()
	})
}
