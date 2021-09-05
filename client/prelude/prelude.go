package prelude

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
	// Operator implants have embeded static IDs, but we don't,
	// so to avoid having multiple sessions showing as one on the Operator
	// GUI, we need to have a unique name for them.
	// Plus, having the ID in the name will help the user to make the
	// correlation.
	sessName := fmt.Sprintf("%s-%d", s.Name, s.ID)
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
		Sleep:     3,
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
