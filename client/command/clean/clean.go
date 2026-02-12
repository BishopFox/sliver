package clean

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox
	Copyright (C) 2023 Bishop Fox

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
	"context"

	"github.com/bishopfox/sliver/client/command/flags"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/constants"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/spf13/cobra"
)

// CleanCmd - Remove all profiles, beacons, sessions and implant builds (Builds and logs will still exist on disk in .sliver)
// CleanCmd - Remove 所有配置文件、信标、会话和 implant 版本（Builds 和日志仍将存在于 .sliver 中的磁盘上）
func CleanCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	con.Printf("This command will kill and remove all sessions, beacons and profiles \n")
	confirm := false
	_ = forms.Confirm("Are you sure you want to destroy everything?", &confirm)
	if !confirm {
		return
	}
	err := removeSessionsAndBeacons(con)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	err = removeImplantBuilds(con)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	err = removeProfiles(con)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	con.Printf("All done !\n")
}

func removeImplantBuilds(con *console.SliverClient) error {
	builds, err := con.Rpc.ImplantBuilds(context.Background(), &commonpb.Empty{})
	if err != nil {
		return err
	}
	for name, _ := range builds.Configs {
		_, err := con.Rpc.DeleteImplantBuild(context.Background(), &clientpb.DeleteReq{
			Name: name,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func removeProfiles(con *console.SliverClient) error {
	profiles, err := con.Rpc.ImplantProfiles(context.Background(), &commonpb.Empty{})
	if err != nil {
		return err
	}

	for _, profile := range profiles.Profiles {
		_, err := con.Rpc.DeleteImplantProfile(context.Background(), &clientpb.DeleteReq{
			Name: profile.Name,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func removeSessionsAndBeacons(con *console.SliverClient) error {
	sessions, err := con.Rpc.GetSessions(context.Background(), &commonpb.Empty{})
	if err != nil {
		return err
	}

	for _, session := range sessions.Sessions {
		_, err := con.Rpc.Kill(context.Background(), &sliverpb.KillReq{
			Request: &commonpb.Request{
				SessionID: session.ID,
				Timeout:   flags.DefaultTimeout,
			},
			Force: false,
		})
		if err != nil {
			return err
		}
	}

	beacons, err := con.Rpc.GetBeacons(context.Background(), &commonpb.Empty{})
	if err != nil {
		return err
	}

	for _, beacon := range beacons.Beacons {
		_, err = con.Rpc.RmBeacon(context.Background(), &clientpb.Beacon{ID: beacon.ID})
		if err != nil {
			return err
		}
	}
	return nil
}

// Commands returns the `exit` command.
// Commands 返回 __PH0__ command.
func Command(con *console.SliverClient) []*cobra.Command {
	return []*cobra.Command{{
		Use:   "clean",
		Short: "Remove all profiles, beacons, sessions, implant builds and HTTP profiles (Builds and logs will still exist on disk in .sliver)",
		Run: func(cmd *cobra.Command, args []string) {
			CleanCmd(cmd, con, args)
		},
		GroupID: constants.GenericHelpGroup,
	}}
}
