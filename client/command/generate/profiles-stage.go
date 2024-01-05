package generate

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

import (
	"context"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

// ProfilesStageCmd - Generate an encrypted/compressed implant binary based on a profile
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
