package generate

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
	"context"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

// ProfilesStageCmd - Generate an encrypted/compressed implant binary based on a profile
// ProfilesStageCmd - Generate 和 encrypted/compressed implant 基于配置文件的二进制文件
func ProfilesStageCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	name, _ := cmd.Flags().GetString("name")
	profileName := args[0]
	aesEncryptKey, _ := cmd.Flags().GetString("aes-encrypt-key")
	aesEncryptIv, _ := cmd.Flags().GetString("aes-encrypt-iv")
	rc4EncryptKey, _ := cmd.Flags().GetString("rc4-encrypt-key")
	prependSize, _ := cmd.Flags().GetBool("prepend-size")
	compressF, _ := cmd.Flags().GetString("compress")
	compress := strings.ToLower(compressF)
	save, _ := cmd.Flags().GetString("save")

	profile := GetImplantProfileByName(profileName, con)
	if profile == nil {
		con.PrintErrorf("Profile not found\n")
		return
	}

	ctrl := make(chan bool)
	con.SpinUntil("Compiling, please wait ...", ctrl)
	stage2, err := con.Rpc.GenerateStage(context.Background(), &clientpb.GenerateStageReq{
		Profile:       profileName,
		Name:          name,
		AESEncryptKey: aesEncryptKey,
		AESEncryptIv:  aesEncryptIv,
		RC4EncryptKey: rc4EncryptKey,
		PrependSize:   prependSize,
		Compress:      compress,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	saveTo, err := saveLocation(save, stage2.File.Name, con)
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}

	err = os.WriteFile(saveTo, stage2.File.Data, 0o700)
	if err != nil {
		con.PrintErrorf("Failed to write to: %s\n", saveTo)
		return
	}
	con.PrintInfof("Implant saved to %s\n", saveTo)
}
