package core

import (
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"sync"
)

var (
	// TunSocksTunnelsnels - Interating with duplex SocksTunnels
	SocksTunnels = tcpTunnel{
		tunnels: map[uint64]*TcpTunnel{},
		mutex:   &sync.Mutex{},
	}

)
type TcpTunnel struct {
	ID        uint64
	SessionID uint32
	ToImplantSequence uint64
	ToImplantMux sync.Mutex

	FromImplant         chan *sliverpb.SocksData
	FromImplantSequence uint64
	Client rpcpb.SliverRPC_SocksProxyServer
}
type tcpTunnel struct {
	tunnels map[uint64]*TcpTunnel
	mutex   *sync.Mutex
}

func (t *tcpTunnel)Create(sessionID uint32) *TcpTunnel {
	tunnelID := NewTunnelID()
	session := Sessions.Get(sessionID)
	tunnel := &TcpTunnel{
		ID:          tunnelID,
		SessionID:   session.ID,
		//ToImplant:   make(chan []byte),
		FromImplant: make(chan *sliverpb.SocksData),
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.tunnels[tunnel.ID] = tunnel

	return tunnel
}

func (t *tcpTunnel)Close(tunnelID uint64) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	tunnel := t.tunnels[tunnelID]
	if tunnel == nil {
		return ErrInvalidTunnelID
	}
	delete(t.tunnels, tunnelID)
	//close(tunnel.ToImplant)
	close(tunnel.FromImplant)
	return nil
}

func (t *tcpTunnel)Get(tunnelID uint64)  *TcpTunnel {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.tunnels[tunnelID]
}