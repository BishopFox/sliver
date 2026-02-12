package help

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox
	Copyright (C) 2019 Bishop Fox

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

// import (
// 进口 （
// 	"fmt"
// 	__PH0__
// 	"sort"
// 	__PH0__
//
// 	"github.com/bishopfox/sliver/client/console"
// 	__PH0__
// 	consts "github.com/bishopfox/sliver/client/constants"
// 	常量__PH0__
//
// 	"github.com/desertbit/columnize"
// 	__PH0__
// 	"github.com/desertbit/grumble"
// 	__PH0__
// )

// HelpCmd - Returns an instance of the 'help' command
// HelpCmd - Returns __PH0__ 命令的实例
// func HelpCmd(con *console.SliverConsole) func(a *grumble.App, shell bool) {
// 	return func(a *grumble.App, shell bool) {
// 	返回 func(a *grumble.App, shell bool) {
// 		printHelp(con)
// 		printHelp（反）
// 	}
// }

// func printHelp(con *console.SliverConsole) {
// 函数 printHelp(con *console.SliverConsole) {
// 	config := columnize.DefaultConfig()
// 	配置：= columnize.DefaultConfig()
// 	config.Delim = "|"
// 	config.Delim = __PH0__
// 	config.Glue = "  "
// 	config.Glue = __PH0__
// 	config.Prefix = "  "
// 	config.Prefix = __PH0__
// 	// Group the commands by their help group if present.
// 	// Group 帮助组的命令 if present.
// 	groups := make(map[string]*grumble.Commands)
// 	组 := make(map[string]*grumble.Commands)
// 	for _, c := range con.App.CurrentMenu().Commands() {
// 	for _, c := 范围 con.App.CurrentMenu().Commands() {
// 		key := c.GroupID
// 		键：= c.GroupID
// 		targetOS := ""
// 		targetOS := __PH0__
// 		session, beacon := con.ActiveTarget.Get()
// 		if session != nil {
// 		如果 session != nil {
// 			targetOS = session.OS
// 		} else if beacon != nil {
// 		} 否则如果 beacon != nil {
// 			targetOS = beacon.OS
// 		}
// 		if beacon != nil || session != nil {
// 		如果 beacon != nil || session != nil {
// 			if targetOS != "windows" && key == consts.SliverWinHelpGroup {
// 			如果 targetOS != __PH0__ && 键 == consts.SliverWinHelpGroup {
// 				continue
// 				继续
// 			}
// 		} else {
// 		} 别的 {
// 			if key == consts.SliverHelpGroup || key == consts.SliverWinHelpGroup || key == consts.AliasHelpGroup || key == consts.ExtensionHelpGroup {
// 			如果键== consts.SliverHelpGroup ||键== consts.SliverWinHelpGroup ||键== consts.AliasHelpGroup ||键== consts.ExtensionHelpGroup {
// 				continue
// 				继续
// 			}
// 		}
// 		if len(key) == 0 {
// 		如果 len(key) == 0 {
// 			key = "Commands:"
// 			键 = __PH0__
// 		}
// 		cc := groups[key]
// 		cc := 组[键]
// 		if cc == nil {
// 		如果 cc == nil {
// 			cc = new(grumble.Commands)
// 			抄送 = 新（grumble.Commands）
// 			groups[key] = cc
// 			组[键] = 抄送
// 		}
// 		cc.Add(c)
// 	}
//
// 	// Sort the map by the keys.
// 	// Sort 由 keys. 映射
// 	var keys []string
// 	var 键 [] 字符串
// 	for k := range groups {
// 	对于 k := 范围组 {
// 		keys = append(keys, k)
// 		键 = 附加（键，k）
// 	}
// 	sort.Strings(keys)
// 	sort.Strings（按键）
//
// 	// Print each commands group.
// 	// Print 每个命令 group.
// 	for _, headline := range keys {
// 	对于 _，标题 := 范围键 {
// 		cc := groups[headline]
// 		cc := 组[标题]
// 		cc.Sort()
//
// 		var output []string
// 		var 输出 [] 字符串
// 		for _, c := range cc.All() {
// 		对于 _, c := 范围 cc.All() {
// 			name := c.Name
// 			名称 := c.Name
// 			for _, a := range c.Aliases {
// 			对于 _, a := 范围 c.Aliases {
// 				name += ", " + a
// 				姓名 += __PH0__ + a
// 			}
// 			output = append(output, fmt.Sprintf("%s | %v", name, c.Help))
// 			输出=附加（输​​出，fmt.Sprintf（__PH0__，名称，c.Help））
// 		}
//
// 		if len(output) > 0 {
// 		如果长度（输出）> 0 {
// 			con.Println()
// 			printHeadline(con.App.Config(), headline, con)
// 			printHeadline(con.App.Config()，标题，骗局)
// 			con.Printf("%s\n", columnize.Format(output, config))
// 			con.Printf(__PH0__, columnize.Format(输出, 配置))
// 			con.Println()
// 		}
// 	}
//
// 	con.Println()
// 	con.Printf("For even more information, please see our wiki: https://github.com/BishopFox/sliver/wiki\n")
// 	con.Printf("For 更多信息，请参阅我们的维基：__PH0__
// 	con.Println()
// }
//
// func headlinePrinter(c *grumble.Config, con *console.SliverConsole) func(v ...interface{}) (int, error) {
// 	if c.NoColor || c.HelpHeadlineColor == nil {
// 	如果 c.NoColor || c.HelpHeadlineColor == 零 {
// 		return con.Println
// 		返回 con.Println
// 	}
// 	return c.HelpHeadlineColor.Println
// 	返回 c.HelpHeadlineColor.Println
// }
//
// func printHeadline(config *grumble.Config, s string, con *console.SliverConsole) {
// func printHeadline(config *grumble.Config, s 字符串, con *console.SliverConsole) {
// 	println := headlinePrinter(config, con)
// 	println := headlinePrinter(配置，con)
// 	if config.HelpHeadlineUnderline {
// 	如果 config.HelpHeadlineUnderline {
// 		println(s)
// 		u := ""
// 		for i := 0; i < len(s); i++ {
// 		对于我：= 0； i < 长度；我++ {
// 			u += "="
// 		}
// 		println(u)
// 		打印（u）
// 	} else {
// 	} 别的 {
// 		println(s)
// 	}
// }
