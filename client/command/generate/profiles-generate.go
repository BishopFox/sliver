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
	"os"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
)

// ProfilesGenerateCmd - Generate an implant binary based on a profile.
// ProfilesGenerateCmd - Generate 一个基于 profile. 的 implant 二进制文件
func ProfilesGenerateCmd(cmd *cobra.Command, con *console.SliverClient, args []string) {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	if name == "" {
		con.PrintErrorf("No profile name specified\n")
		return
	}

	implantName, _ := cmd.Flags().GetString("name")
	save, _ := cmd.Flags().GetString("save")
	if save == "" {
		save, _ = os.Getwd()
	}
	profile := GetImplantProfileByName(name, con)
	if profile != nil {
		// Override shellcode encoder if explicitly requested.
		// 如果明确 requested. 则为 Override shellcode 编码器
		if cmd.Flags().Changed("shellcode-encoder") {
			rawEncoder, _ := cmd.Flags().GetString("shellcode-encoder")
			rawEncoder = strings.TrimSpace(rawEncoder)
			if rawEncoder == "" {
				con.PrintErrorf("shellcode-encoder cannot be empty; use 'none' to disable encoding\n")
				return
			}

			normalized := normalizeShellcodeEncoderName(rawEncoder)
			if normalized == "none" {
				profile.Config.ShellcodeEncoder = clientpb.ShellcodeEncoder_NONE
				profile.Config.SGNEnabled = false
			} else if profile.Config.Format != clientpb.OutputFormat_SHELLCODE {
				con.PrintWarnf("Shellcode encoder only applies when using `--format shellcode`, ignoring.\n")
			} else {
				encoderMap, err := fetchShellcodeEncoderMap(con)
				if err != nil {
					con.PrintErrorf("Failed to fetch shellcode encoders: %s\n", err)
					return
				}
				encoder, ok := shellcodeEncoderEnumForArch(encoderMap, profile.Config.GOARCH, rawEncoder)
				if !ok {
					compatible := compatibleShellcodeEncoderNames(encoderMap, profile.Config.GOARCH)
					if len(compatible) == 0 {
						con.PrintErrorf("No shellcode encoders are available for arch %s\n", normalizeShellcodeArch(profile.Config.GOARCH))
					} else {
						con.PrintErrorf("Unsupported shellcode encoder %q for arch %s (valid: %s)\n", rawEncoder, normalizeShellcodeArch(profile.Config.GOARCH), strings.Join(compatible, ", "))
					}
					return
				}
				profile.Config.ShellcodeEncoder = encoder
				profile.Config.SGNEnabled = encoder == clientpb.ShellcodeEncoder_SHIKATA_GA_NAI
			}
		}
		_, err := compile(implantName, profile.Config, save, con)
		if err != nil {
			return
		}
	} else {
		con.PrintErrorf("No profile with name '%s'", name)
	}
}
