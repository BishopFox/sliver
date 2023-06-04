package generate

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

// GenerateStagerCmd - Generate a stager using Metasploit
func GenerateStagerCmd(cmd *cobra.Command, con *console.SliverConsoleClient, args []string) {
	var stageProto clientpb.StageProtocol
	lhost, _ := cmd.Flags().GetString("lhost")
	if lhost == "" {
		con.PrintErrorf("Please specify a listening host")
		return
	}
	match, err := regexp.MatchString(`^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$`, lhost)
	if err != nil {
		return
	}
	if !match {
		addr, err := net.LookupHost(lhost)
		if err != nil {
			con.PrintErrorf("Error resolving %s: %v\n", lhost, err)
			return
		}
		if len(addr) > 1 {
			prompt := &survey.Select{
				Message: "Select an address",
				Options: addr,
			}
			err := survey.AskOne(prompt, &lhost)
			if err != nil {
				con.PrintErrorf("Error: %v\n", err)
				return
			}
		} else {
			lhost = addr[0]
		}
	}
	lport, _ := cmd.Flags().GetUint32("lport")
	stageOS, _ := cmd.Flags().GetString("os")
	arch, _ := cmd.Flags().GetString("arch")
	proto, _ := cmd.Flags().GetString("protocol")
	format, _ := cmd.Flags().GetString("format")
	badChars, _ := cmd.Flags().GetString("badchars")
	save, _ := cmd.Flags().GetString("save")

	bChars := make([]string, 0)
	if len(badChars) > 0 {
		for _, b := range strings.Split(badChars, " ") {
			bChars = append(bChars, fmt.Sprintf("\\x%s", b))
		}
	}

	switch proto {
	case "tcp":
		stageProto = clientpb.StageProtocol_TCP
	case "http":
		stageProto = clientpb.StageProtocol_HTTP
	case "https":
		stageProto = clientpb.StageProtocol_HTTPS
	default:
		con.PrintErrorf("%s staging protocol not supported\n", proto)
		return
	}

	ctrl := make(chan bool)
	con.SpinUntil("Generating stager, please wait ...", ctrl)
	stageFile, err := con.Rpc.MsfStage(context.Background(), &clientpb.MsfStagerReq{
		Arch:     arch,
		BadChars: bChars,
		Format:   format,
		Host:     lhost,
		Port:     lport,
		Protocol: stageProto,
		OS:       stageOS,
	})
	ctrl <- true
	<-ctrl

	if err != nil {
		con.PrintErrorf("Error: %v - Please make sure Metasploit framework >= v6.2 is installed and msfvenom/msfconsole are in your PATH", err)
		return
	}

	if save != "" || format == "raw" {
		saveTo, err := saveLocation(save, stageFile.GetFile().GetName(), con)
		if err != nil {
			return
		}

		err = os.WriteFile(saveTo, stageFile.GetFile().GetData(), 0o700)
		if err != nil {
			con.PrintErrorf("Failed to write to: %s\n", saveTo)
			return
		}
		con.PrintInfof("Sliver implant stager saved to: %s\n", saveTo)
	} else {
		con.PrintInfof("Here's your stager:")
		con.Println(string(stageFile.GetFile().GetData()))
	}
}
