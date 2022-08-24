package sgn

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
	"encoding/hex"
	"io/ioutil"
	"path/filepath"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

// ShikataGaNaiCmd - Command wrapper for the Shikata Ga Nai shellcode encoder
func ShikataGaNaiCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	shellcodeFile := ctx.Args.String("shellcode")
	rawShellcode, err := ioutil.ReadFile(shellcodeFile)
	if err != nil {
		con.PrintErrorf("Failed to read shellcode file: %s", err)
		return
	}

	arch := ctx.Flags.String("arch")
	iterations := ctx.Flags.Int("iterations")
	rawBadChars := ctx.Flags.String("bad-chars")
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
		Request:      con.ActiveTarget.Request(ctx),
	})
	if err != nil {
		con.PrintErrorf("%s\n", err)
		return
	}
	if shellcodeResp.Response != nil && shellcodeResp.Response.Err != "" {
		con.PrintErrorf("%s\n", shellcodeResp.Response.Err)
		return
	}

	outputFile := ctx.Flags.String("save")
	if outputFile == "" {
		outputFile = filepath.Base(shellcodeFile)
		outputFile += ".sgn"
	}

	err = ioutil.WriteFile(outputFile, shellcodeResp.Data, 0644)
	if err != nil {
		con.PrintErrorf("Failed to write shellcode file: %s", err)
		return
	}
	con.PrintInfof("Shellcode written to %s (%d bytes)\n", outputFile, len(shellcodeResp.Data))
}
