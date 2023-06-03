package help

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

// import (
// 	"fmt"
// 	"sort"
//
// 	"github.com/bishopfox/sliver/client/console"
// 	consts "github.com/bishopfox/sliver/client/constants"
//
// 	"github.com/desertbit/columnize"
// 	"github.com/desertbit/grumble"
// )

// HelpCmd - Returns an instance of the 'help' command
// func HelpCmd(con *console.SliverConsole) func(a *grumble.App, shell bool) {
// 	return func(a *grumble.App, shell bool) {
// 		printHelp(con)
// 	}
// }

// func printHelp(con *console.SliverConsole) {
// 	config := columnize.DefaultConfig()
// 	config.Delim = "|"
// 	config.Glue = "  "
// 	config.Prefix = "  "
// 	// Group the commands by their help group if present.
// 	groups := make(map[string]*grumble.Commands)
// 	for _, c := range con.App.CurrentMenu().Commands() {
// 		key := c.GroupID
// 		targetOS := ""
// 		session, beacon := con.ActiveTarget.Get()
// 		if session != nil {
// 			targetOS = session.OS
// 		} else if beacon != nil {
// 			targetOS = beacon.OS
// 		}
// 		if beacon != nil || session != nil {
// 			if targetOS != "windows" && key == consts.SliverWinHelpGroup {
// 				continue
// 			}
// 		} else {
// 			if key == consts.SliverHelpGroup || key == consts.SliverWinHelpGroup || key == consts.AliasHelpGroup || key == consts.ExtensionHelpGroup {
// 				continue
// 			}
// 		}
// 		if len(key) == 0 {
// 			key = "Commands:"
// 		}
// 		cc := groups[key]
// 		if cc == nil {
// 			cc = new(grumble.Commands)
// 			groups[key] = cc
// 		}
// 		cc.Add(c)
// 	}
//
// 	// Sort the map by the keys.
// 	var keys []string
// 	for k := range groups {
// 		keys = append(keys, k)
// 	}
// 	sort.Strings(keys)
//
// 	// Print each commands group.
// 	for _, headline := range keys {
// 		cc := groups[headline]
// 		cc.Sort()
//
// 		var output []string
// 		for _, c := range cc.All() {
// 			name := c.Name
// 			for _, a := range c.Aliases {
// 				name += ", " + a
// 			}
// 			output = append(output, fmt.Sprintf("%s | %v", name, c.Help))
// 		}
//
// 		if len(output) > 0 {
// 			con.Println()
// 			printHeadline(con.App.Config(), headline, con)
// 			con.Printf("%s\n", columnize.Format(output, config))
// 			con.Println()
// 		}
// 	}
//
// 	con.Println()
// 	con.Printf("For even more information, please see our wiki: https://github.com/BishopFox/sliver/wiki\n")
// 	con.Println()
// }
//
// func headlinePrinter(c *grumble.Config, con *console.SliverConsole) func(v ...interface{}) (int, error) {
// 	if c.NoColor || c.HelpHeadlineColor == nil {
// 		return con.Println
// 	}
// 	return c.HelpHeadlineColor.Println
// }
//
// func printHeadline(config *grumble.Config, s string, con *console.SliverConsole) {
// 	println := headlinePrinter(config, con)
// 	if config.HelpHeadlineUnderline {
// 		println(s)
// 		u := ""
// 		for i := 0; i < len(s); i++ {
// 			u += "="
// 		}
// 		println(u)
// 	} else {
// 		println(s)
// 	}
// }
