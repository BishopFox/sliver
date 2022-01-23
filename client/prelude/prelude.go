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
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/bishopfox/sliver/client/prelude/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

var SessionMapper *PreludeSessionMapper

const defaultImplantSleep = 5

type OperatorConfig struct {
	Range       string
	OperatorURL string
	RPC         rpcpb.SliverRPCClient
	AESKey      string
}

type PreludeSessionMapper struct {
	sessions []*AgentSession
	conf     *OperatorConfig
	sync.Mutex
}

func InitSessionMapper(conf *OperatorConfig) *PreludeSessionMapper {
	if SessionMapper == nil {
		SessionMapper = &PreludeSessionMapper{
			sessions: make([]*AgentSession, 0),
			conf:     conf,
		}
	}
	return SessionMapper
}

func (p *PreludeSessionMapper) AddSession(s *clientpb.Session) error {
	var pwd string
	conn, err := net.Dial("tcp", p.conf.OperatorURL)
	if err != nil {
		return err
	}
	pwdResp, err := p.conf.RPC.Pwd(context.Background(), &sliverpb.PwdReq{
		Request: MakeRequest(s),
	})
	if err != nil {
		return err
	}
	if pwdResp != nil {
		pwd = pwdResp.Path
	}
	// Operator implants have embedded static IDs, but we don't,
	// so to avoid having multiple sessions showing as one on the Operator
	// GUI, we need to have a unique name for them.
	// Plus, having the ID in the name will help the user to make the
	// correlation.
	sessName := fmt.Sprintf("%s-%s", s.Name, s.ID)
	beacon := Beacon{
		Name:      sessName,
		Target:    p.conf.OperatorURL,
		Hostname:  s.Hostname,
		Location:  s.Filename,
		Platform:  s.OS,
		Range:     p.conf.Range,
		Executors: util.DetermineExecutors(s.OS, s.Arch),
		Links:     make([]Instruction, 0),
		Executing: "",
		Pwd:       pwd,
		Sleep:     defaultImplantSleep,
	}

	if p.conf.AESKey == "" {
		return errors.New("missing AES key")
	}
	encryptionKey := p.conf.AESKey
	agentConfig := AgentConfig{
		Name:      sessName,
		AESKey:    encryptionKey,
		Range:     p.conf.Range,
		Contact:   "tcp",
		Address:   p.conf.OperatorURL,
		Pid:       int(s.PID),
		Executing: make(map[string]Instruction),
		Sleep:     defaultImplantSleep,
	}
	util.EncryptionKey = &agentConfig.AESKey
	agentSession := NewAgentSession(&conn, s, p.conf.RPC, beacon, agentConfig)
	p.Lock()
	p.sessions = append(p.sessions, agentSession)
	p.Unlock()
	go agentSession.ReceiveLoop()
	return nil
}

func (p *PreludeSessionMapper) RemoveSession(s *clientpb.Session) (err error) {
	p.Lock()
	for _, agentSession := range p.sessions {
		if agentSession.Session.ID == s.ID {
			if agentSession.Conn != nil {
				err = (*agentSession.Conn).Close()
			} else {
				err = errors.New("connection is nil")
			}
		}
	}
	p.Unlock()
	return err
}

func (p *PreludeSessionMapper) GetConfig() *OperatorConfig {
	return p.conf
}
