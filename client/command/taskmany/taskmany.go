package taskmany

/*
	Sliver Implant Framework
	Copyright (C) 2021 Bishop Fox
	Copyright (C) 2023 ActualTrash

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
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/command/help"
	"github.com/bishopfox/sliver/client/console"
	consts "github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
)

func Command(con *console.SliverClient) []*cobra.Command {
	taskmanyCmd := &cobra.Command{
		Use:     consts.TaskmanyStr,
		Short:   "Task many beacons or sessions",
		Long:    help.GetHelpFor([]string{consts.TaskmanyStr}),
		GroupID: consts.SliverHelpGroup,
		Run: func(cmd *cobra.Command, args []string) {
			TaskmanyCmd(cmd, con, args)
		},
	}

	// Add the relevant beacon commands as a subcommand to taskmany
	// Add 相关 beacon 命令作为 taskmany 的子命令
	// taskmanyCmds := map[string]bool{
	// taskmanyCmds := 地图[字符串]布尔{
	// 	consts.ExecuteStr:     true,
	// 	consts.ExecuteStr: 正确，
	// 	consts.LsStr:          true,
	// 	consts.LsStr: 正确，
	// 	consts.CdStr:          true,
	// 	consts.CdStr: 正确，
	// 	consts.MkdirStr:       true,
	// 	consts.MkdirStr: 正确，
	// 	consts.RmStr:          true,
	// 	consts.RmStr: 正确，
	// 	consts.UploadStr:      true,
	// 	consts.UploadStr: 正确，
	// 	consts.DownloadStr:    true,
	// 	consts.DownloadStr: 正确，
	// 	consts.InteractiveStr: true,
	// 	consts.InteractiveStr: 正确，
	// 	consts.ChmodStr:       true,
	// 	consts.ChmodStr: 正确，
	// 	consts.ChownStr:       true,
	// 	consts.ChownStr: 正确，
	// 	consts.ChtimesStr:     true,
	// 	consts.ChtimesStr: 正确，
	// 	consts.PwdStr:         true,
	// 	consts.PwdStr: 正确，
	// 	consts.CatStr:         true,
	// 	consts.CatStr: 正确，
	// 	consts.MvStr:          true,
	// 	consts.MvStr: 正确，
	// 	consts.PingStr:        true,
	// 	consts.PingStr: 正确，
	// 	consts.NetstatStr:     true,
	// 	consts.NetstatStr: 正确，
	// 	consts.PsStr:          true,
	// 	consts.PsStr: 正确，
	// 	consts.IfconfigStr:    true,
	// 	consts.IfconfigStr: 正确，
	// }

	// for _, c := range SliverCommands(con)().Commands() {
	// for _, c := 范围 SliverCommands(con)().Commands() {
	// 	_, ok := taskmanyCmds[c.Use]
	// 	_，好的：= taskmanyCmds[c.Use]
	// 	if ok {
	// 	如果可以的话{
	// 		taskmanyCmd.AddCommand(WrapCommand(c, con))
	// 	}
	// }

	return []*cobra.Command{taskmanyCmd}
}

// TaskmanyCmd - Task many beacons / sessions
// TaskmanyCmd - Task 许多信标/会话
func TaskmanyCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	con.PrintErrorf("Must specify subcommand. See taskmany --help for supported subcommands.\n")
}

// Helper function to wrap grumble commands with taskmany logic
// Helper 函数用 taskmany 逻辑包装 grumble 命令
func WrapCommand(c *cobra.Command, con *console.SliverClient) *cobra.Command {
	wc := &cobra.Command{
		Use:   c.Use,
		Short: c.Short,
		Long:  c.Long,
		Args:  c.Args,
		Run:   wrapFunctionWithTaskmany(con, c.Run),
	}
	wc.Flags().AddFlagSet(c.Flags())
	wc.PersistentFlags().AddFlagSet(c.PersistentFlags())
	return wc
}

// Wrap a function to run it for each beacon / session
// Wrap 一个为每个 beacon / session 运行它的函数
func wrapFunctionWithTaskmany(con *console.SliverClient, f func(cmd *cobra.Command, args []string)) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		defer con.Println()

		sessions, beacons, err := SelectMultipleBeaconsAndSessions(con)
		if err != nil {
			con.Println()
			con.PrintErrorf("%s\n", err)
			return
		}

		con.Println()

		// Save current active beacon or session
		// Save 当前活动 beacon 或 session
		origSession, origBeacon := con.ActiveTarget.Get()

		nB := 0
		nBSkipped := 0
		for _, b := range beacons {
			if !b.IsDead {
				con.ActiveTarget.Set(nil, b)
				f(cmd, args)
				nB += 1
			} else {
				nBSkipped += 1
			}
		}

		nS := 0
		nSSkipped := 0
		for _, s := range sessions {
			if !s.IsDead {
				con.ActiveTarget.Set(s, nil)
				f(cmd, args)
				nS += 1
			} else {
				nSSkipped += 1
			}
		}

		// Restore active session / beacon
		// Restore 活跃 session / beacon
		con.ActiveTarget.Set(origSession, origBeacon)

		con.PrintInfof("Tasked %d sessions and %d beacons >:D\n", nS, nB)
		if nBSkipped > 0 || nSSkipped > 0 {
			con.PrintWarnf("Skipped %d dead sessions and %d dead beacons\n", nSSkipped, nBSkipped)
		}
	}
}

func SelectMultipleBeaconsAndSessions(con *console.SliverClient) ([]*clientpb.Session, []*clientpb.Beacon, error) {
	// Get and sort sessions
	// Get 并对会话进行排序
	sessionsObj, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		return nil, nil, err
	}
	sessions := sessionsObj.Sessions
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ID < sessions[j].ID
	})

	// Get and sort beacons
	// Get 和排序信标
	beaconsObj, err := con.Rpc.GetBeacons(context.Background(), &commonpb.Empty{})
	if err != nil {
		return nil, nil, err
	}
	beacons := beaconsObj.Beacons
	sort.Slice(beacons, func(i, j int) bool {
		return beacons[i].ID < beacons[j].ID
	})

	if len(beacons) == 0 && len(sessions) == 0 {
		return nil, nil, fmt.Errorf("no sessions or beacons 🙁")
	}

	// Render selection table
	// Render选型表
	outputBuf := bytes.NewBufferString("")
	table := tabwriter.NewWriter(outputBuf, 0, 2, 2, ' ', 0)

	sessionOptionMap := map[string]*clientpb.Session{}
	for _, session := range sessions {
		option := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s",
			"SESSION",
			strings.Split(session.ID, "-")[0],
			session.Name,
			session.RemoteAddress,
			session.Hostname,
			session.Username,
			fmt.Sprintf("%s/%s", session.OS, session.Arch),
		)
		fmt.Fprintln(table, option)
		o := strings.ReplaceAll(option, "\t", "")
		sessionOptionMap[o] = session
	}

	beaconOptionMap := map[string]*clientpb.Beacon{}
	for _, beacon := range beacons {
		option := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s",
			"BEACON",
			strings.Split(beacon.ID, "-")[0],
			beacon.Name,
			beacon.RemoteAddress,
			beacon.Hostname,
			beacon.Username,
			fmt.Sprintf("%s/%s", beacon.OS, beacon.Arch),
		)
		fmt.Fprintln(table, option)
		o := strings.ReplaceAll(option, "\t", "")
		beaconOptionMap[o] = beacon
	}
	table.Flush()

	options := strings.Split(outputBuf.String(), "\n")
	options = options[:len(options)-1] // Remove the last empty option
	options = options[:len(options)-1] // Remove 最后一个空选项
	selected := []string{}
	_ = forms.MultiSelect("Select sessions and beacons:", options, &selected)

	if len(selected) == 0 {
		return nil, nil, fmt.Errorf("no sessions or beacons selected 🤔")
	}

	selectedSessions := []*clientpb.Session{}
	selectedBeacons := []*clientpb.Beacon{}
	for _, s := range selected {
		s = strings.ReplaceAll(s, " ", "")
		s = strings.ReplaceAll(s, "\t", "")
		session, ok := sessionOptionMap[s]
		if ok {
			selectedSessions = append(selectedSessions, session)
		}

		beacon, ok := beaconOptionMap[s]
		if ok {
			selectedBeacons = append(selectedBeacons, beacon)
		}
	}

	return selectedSessions, selectedBeacons, nil
}
