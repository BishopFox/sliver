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
	"os"
	"strings"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/spf13/cobra"
)

// ProfilesGenerateCmd - Generate an implant binary based on a profile.
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
		_, err := compile(implantName, profile.Config, nil, save, con)
		if err != nil {
			return
		}
	} else {
		con.PrintErrorf("No profile with name '%s'", name)
	}
}
