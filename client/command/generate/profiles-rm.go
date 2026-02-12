package generate

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
	"context"
	"fmt"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/client/forms"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
)

// ProfilesRmCmd - Delete an implant profile.
// ProfilesRmCmd - Delete 和 implant profile.
func ProfilesRmCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var name string
	if len(args) > 0 {
		name = args[0]
	}
	// name := ctx.Args.String("name")
	// 名称 := ctx.Args.String(__PH0__)
	if name == "" {
		con.PrintErrorf("No profile name specified\n")
		return
	}
	profile := GetImplantProfileByName(name, con)
	if profile == nil {
		con.PrintErrorf("No profile found with name '%s'\n", name)
		return
	}
	confirm := false
	_ = forms.Confirm(fmt.Sprintf("Remove '%s' profile?", name), &confirm)
	if !confirm {
		return
	}
	_, err := con.Rpc.DeleteImplantProfile(context.Background(), &clientpb.DeleteReq{
		Name: name,
	})
	if err != nil {
		con.PrintErrorf("Failed to delete profile %s\n", err)
		return
	}
}
