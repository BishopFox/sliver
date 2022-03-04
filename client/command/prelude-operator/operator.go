package operator

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
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/prelude"
	"github.com/desertbit/grumble"
)

func OperatorCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	if prelude.ImplantMapper != nil {
		con.PrintInfof("Connected to Operator at %s\n", prelude.ImplantMapper.GetConfig().OperatorURL)
		return
	}
	con.PrintInfof("Not connected to any Operator server. Use `operator connect` to connect to one.")
}
