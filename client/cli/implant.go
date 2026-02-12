package cli

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

import (
	"errors"

	"github.com/bishopfox/sliver/client/command"
	"github.com/bishopfox/sliver/client/command/use"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/constants"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func implantCmd(con *console.SliverClient) *cobra.Command {
	con.IsCLI = true

	makeCommands := command.SliverCommands(con)
	cmd := makeCommands()
	cmd.Use = constants.ImplantMenu

	// Flags
	// 参数
	implantFlags := pflag.NewFlagSet(constants.ImplantMenu, pflag.ContinueOnError)
	implantFlags.StringP("use", "s", "", "interact with a session")
	cmd.Flags().AddFlagSet(implantFlags)

	// Pre-runners (console setup, connection, etc)
	// Pre-runners（console 初始化、连接等）
	cmd.PersistentPreRunE, cmd.PersistentPostRunE = makeRunners(cmd, con)

	// Completions
	// 补全
	makeCompleters(cmd, con)

	return cmd
}

func makeRunners(implantCmd *cobra.Command, con *console.SliverClient) (pre, post func(cmd *cobra.Command, args []string) error) {
	startConsole, closeConsole := consoleRunnerCmd(con, false)

	// The pre-run function connects to the server and sets up a "fake" console,
	// pre-run 函数连接 server 并设置一个“fake” console，
	// so we can have access to active sessions/beacons, and other stuff needed.
	// 以便访问活跃的 sessions/beacons 及其他所需信息。
	pre = func(_ *cobra.Command, args []string) error {
		startConsole(implantCmd, args)

		// Set the active target.
		// 设置活跃目标。
		target, _ := implantCmd.Flags().GetString("use")
		if target == "" {
			return errors.New("no target implant to run command on")
		}

		session := con.GetSession(target)
		if session != nil {
			con.ActiveTarget.Set(session, nil)
		}

		return nil
	}

	return pre, closeConsole
}

func makeCompleters(cmd *cobra.Command, con *console.SliverClient) {
	comps := carapace.Gen(cmd)

	comps.PreRun(func(cmd *cobra.Command, args []string) {
		cmd.PersistentPreRunE(cmd, args)
	})

	// Bind completers to flags (wrap them to use the same pre-runners)
	// 将补全器绑定到参数（封装后复用同一组 pre-runners）
	command.BindFlagCompletions(cmd, func(comp *carapace.ActionMap) {
		(*comp)["use"] = carapace.ActionCallback(func(c carapace.Context) carapace.Action {
			cmd.PersistentPreRunE(cmd, c.Args)
			return use.SessionIDCompleter(con)
		})
	})
}
