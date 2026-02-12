package sgn

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox
	Copyright (C) 2022 Bishop Fox

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
	"encoding/hex"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

// ShikataGaNaiCmd - Command wrapper for the Shikata Ga Nai shellcode encoder
// ShikataGaNaiCmd - Command Shikata Ga Nai shellcode 编码器的包装器
func ShikataGaNaiCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	shellcodeFile := args[0]
	rawShellcode, err := os.ReadFile(shellcodeFile)
	if err != nil {
		con.PrintErrorf("Failed to read shellcode file: %s", err)
		return
	}

	arch, _ := cmd.Flags().GetString("arch")
	iterations, _ := cmd.Flags().GetInt("iterations")
	rawBadChars, _ := cmd.Flags().GetString("bad-chars")
	badChars, err := hex.DecodeString(rawBadChars)
	if err != nil {
		con.PrintErrorf("Failed to decode bad chars: %s", err)
		return
	}

	con.PrintInfof("Encoding shellcode with %d iterations and %d bad chars\n", iterations, len(badChars))

	shellcodeResp, err := con.Rpc.ShellcodeEncoder(context.Background(), &clientpb.ShellcodeEncodeReq{
		Encoder:      clientpb.ShellcodeEncoder_SHIKATA_GA_NAI,
		Architecture: arch,
		Iterations:   uint32(iterations),
		BadChars:     badChars,
		Data:         rawShellcode,
		Request:      con.ActiveTarget.Request(cmd),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if shellcodeResp.Response != nil && shellcodeResp.Response.Err != "" {
		con.PrintErrorf("%s\n", shellcodeResp.Response.Err)
		return
	}

	outputFile, _ := cmd.Flags().GetString("save")
	if outputFile == "" {
		outputFile = filepath.Base(shellcodeFile)
		outputFile += ".sgn"
	}

	err = os.WriteFile(outputFile, shellcodeResp.Data, 0o644)
	if err != nil {
		con.PrintErrorf("Failed to write shellcode file: %s", err)
		return
	}
	con.PrintInfof("Shellcode written to %s (%d bytes)\n", outputFile, len(shellcodeResp.Data))
}
