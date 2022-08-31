package prelude

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
	"errors"
	"net"
	"sync"

	"github.com/bishopfox/sliver/client/prelude/bridge"
	"github.com/bishopfox/sliver/client/prelude/config"
	"github.com/bishopfox/sliver/client/prelude/implant"
	"github.com/bishopfox/sliver/client/prelude/util"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

var ImplantMapper *OperatorImplantMapper

const (
	defaultImplantSleep = 5
	defaultRange        = "sliver"
)

type OperatorConfig struct {
	Range       string
	OperatorURL string
	RPC         rpcpb.SliverRPCClient
	AESKey      string
}

// OperatorImplantMapper maps an OperatorConfig with
// active Sliver implant sessions/beacons
type OperatorImplantMapper struct {
	implantBridges []*bridge.OperatorImplantBridge
	conf           *OperatorConfig
	sync.Mutex
}

func InitImplantMapper(conf *OperatorConfig) *OperatorImplantMapper {
	if ImplantMapper == nil {
		ImplantMapper = &OperatorImplantMapper{
			implantBridges: make([]*bridge.OperatorImplantBridge, 0),
			conf:           conf,
		}
	}
	return ImplantMapper
}

func (p *OperatorImplantMapper) AddImplant(a implant.ActiveImplant, callback func(string, func(*clientpb.BeaconTask))) error {
	var pwd string
	conn, err := net.Dial("tcp", p.conf.OperatorURL)
	if err != nil {
		return err
	}
	pwdResp, err := p.conf.RPC.Pwd(context.Background(), &sliverpb.PwdReq{
		Request: implant.MakeRequest(a),
	})
	if err != nil {
		return err
	}
	if pwdResp != nil {
		pwd = pwdResp.Path
	}

	// Use a default sleep time for sessions,
	// but respect the one we have for beacons
	sleepTime := defaultImplantSleep
	if b, ok := a.(*clientpb.Beacon); ok {
		sleepTime = int(b.ReconnectInterval)
	}
	if p.conf.Range == "" {
		p.conf.Range = defaultRange
	}

	beacon := config.OperatorBeacon{
		Name:      a.GetID(),
		Target:    p.conf.OperatorURL,
		Hostname:  a.GetHostname(),
		Location:  a.GetFilename(),
		Platform:  a.GetOS(),
		Range:     p.conf.Range,
		Executors: util.DetermineExecutors(a.GetOS(), a.GetArch()),
		Links:     make([]config.Instruction, 0),
		Executing: "",
		Pwd:       pwd,
		Sleep:     int(sleepTime),
	}

	if p.conf.AESKey == "" {
		return errors.New("missing AES key")
	}
	encryptionKey := p.conf.AESKey
	agentConfig := config.AgentConfig{
		Name:      a.GetID(),
		AESKey:    encryptionKey,
		Range:     p.conf.Range,
		Contact:   "tcp",
		Address:   p.conf.OperatorURL,
		Pid:       int(a.GetPID()),
		Executing: make(map[string]config.Instruction),
		Sleep:     int(sleepTime),
	}

	bridge := bridge.NewImplantBridge(&conn, a, p.conf.RPC, beacon, agentConfig, callback, RunCommand)
	p.Lock()
	p.implantBridges = append(p.implantBridges, bridge)
	p.Unlock()
	go bridge.ReceiveLoop()
	return nil
}

func (p *OperatorImplantMapper) RemoveImplant(imp implant.ActiveImplant) (err error) {
	p.Lock()
	for _, bridge := range p.implantBridges {
		if bridge.Implant.GetID() == imp.GetID() {
			if bridge.Conn != nil {
				err = (*bridge.Conn).Close()
			} else {
				err = errors.New("connection is nil")
			}
		}
	}
	p.Unlock()
	return err
}

func (p *OperatorImplantMapper) GetConfig() *OperatorConfig {
	return p.conf
}
