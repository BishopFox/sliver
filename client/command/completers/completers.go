package completers

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox
	Copyright (C) 2021 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	This 程序是免费软件：您可以重新分发它 and/or 修改
	it under the terms of the GNU General Public License as published by
	它根据 GNU General Public License 发布的条款
	the Free Software Foundation, either version 3 of the License, or
	Free Software Foundation，License 的版本 3，或
	(at your option) any later version.
	（由您选择）稍后 version.

	This program is distributed in the hope that it will be useful,
	This 程序被分发，希望它有用，
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	但是WITHOUT ANY WARRANTY；甚至没有默示保证
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	MERCHANTABILITY 或 FITNESS FOR A PARTICULAR PURPOSE. See
	GNU General Public License for more details.
	GNU General Public License 更多 details.

	You should have received a copy of the GNU General Public License
	You 应已收到 GNU General Public License 的副本
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	与此 program. If 不一起，请参见 <__PH0__
*/

import (
	"net"

	"github.com/rsteube/carapace"
)

// ClientInterfacesCompleter completes interface addresses on the client host.
// ClientInterfacesCompleter 完成客户端上的接口地址 host.
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
// LocalProxyCompleter 为所有接受客户端代理 address. 的 flags/arguments 提供 URL 补全
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
