package core

import (
	"context"
	"fmt"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/log"
	"github.com/bishopfox/sliver/util/leaky"
	"io"
	"net"
	"sync"
	"time"
)

var (
	// SocksProxys - Struct instance that holds all the portfwds
	SocksProxys = socksProxy{
		tcpProxys: map[int]*SocksProxy{},
		mutex:    &sync.RWMutex{},
	}
	SocksConnPool = sync.Map{}
	SocksProxyID = 0
)
var coreLog = log.NamedLogger("ClientCore", "sessions")
// PortfwdMeta - Metadata about a portfwd listener
type SocksProxyMeta struct {
	ID         int
	SessionID  uint32
	BindAddr   string
	Username string
	Password string
}
type TcpProxy struct {
	Rpc     rpcpb.SliverRPCClient
	Session *clientpb.Session

	Username string
	Password string
	BindAddr        string
	Listener net.Listener
	stopChan bool
	KeepAlivePeriod time.Duration
	DialTimeout     time.Duration
}
// SocksProxy - Tracks portfwd<->tcpproxy
type SocksProxy struct {
	ID           int
	ChannelProxy *TcpProxy
}

// GetMetadata - Get metadata about the portfwd
func (p *SocksProxy) GetMetadata() *SocksProxyMeta {
	return &SocksProxyMeta{
		ID:         p.ID,
		SessionID:  p.ChannelProxy.Session.ID,
		BindAddr:   p.ChannelProxy.BindAddr,
		Username: p.ChannelProxy.Username,
		Password: p.ChannelProxy.Password,
	}
}

type socksProxy struct {
	tcpProxys map[int]*SocksProxy
	mutex    *sync.RWMutex
}

// Add - Add a TCP proxy instance
func (f *socksProxy) Add(tcpProxy *TcpProxy) *SocksProxy {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	Sockser := &SocksProxy{
		ID:       nextSocksProxyID(),
		ChannelProxy: tcpProxy,
	}
	f.tcpProxys[Sockser.ID] = Sockser

	return Sockser
}


func (f *socksProxy)Start(tcpProxy *TcpProxy)error {
	proxy, err := tcpProxy.Rpc.SocksProxy(context.Background())
	if err != nil {
		return err
	}
	go func() {
		for !tcpProxy.stopChan {
			FromImplantSequence:=0
			p, err := proxy.Recv()
			if err != nil {
				return
			}

			if v,ok:=SocksConnPool.Load(p.TunnelID);ok{
				n:=v.(net.Conn)
				if p.CloseConn{
					n.Close()
					SocksConnPool.Delete(p.TunnelID)
					continue
				}
				coreLog.Debugf("[socks] agent to Server To (Client to User) Data Sequence %d , Data Size %d \n",FromImplantSequence,len(p.Data))
				//fmt.Printf("recv data len %d \n", len(p.Data))
				_, err := n.Write(p.Data)
				if err != nil {
					continue
				}
				FromImplantSequence++
			}
		}
	}()
	for !tcpProxy.stopChan {
		l, err := tcpProxy.Listener.Accept()
		if err != nil {
			return err
		}
		rpcSocks, err := tcpProxy.Rpc.CreateSocks(context.Background(), &sliverpb.Socks{
			SessionID: tcpProxy.Session.ID,
		})
		if err !=nil{
			fmt.Println(err)
			return err
		}

		go connect(l,proxy,&sliverpb.SocksData{
			Username: tcpProxy.Username,
			Password: tcpProxy.Password,
			TunnelID: rpcSocks.TunnelID,
			Request:  &commonpb.Request{SessionID: rpcSocks.SessionID},
		})
	}
	fmt.Printf("Socks Stop -> %s\n", tcpProxy.BindAddr)
	tcpProxy.Listener.Close()
	proxy.CloseSend()
	return nil
}

// Remove - Remove a TCP proxy instance
func (f *socksProxy) Remove(socksId int) bool {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if _, ok := f.tcpProxys[socksId]; ok {
		f.tcpProxys[socksId].ChannelProxy.stopChan = true
		delete(f.tcpProxys, socksId)
		return true
	}
	return false
}

// List - List all TCP proxy instances
func (f *socksProxy) List() []*SocksProxyMeta {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	socksProxy := []*SocksProxyMeta{}
	for _, socks := range f.tcpProxys {
		socksProxy = append(socksProxy, socks.GetMetadata())
	}
	return socksProxy
}

func nextSocksProxyID() int {
	SocksProxyID++
	return SocksProxyID
}


const leakyBufSize = 4108 // data.len(2) + hmacsha1(10) + data(4096)

var leakyBuf = leaky.NewLeakyBuf(2048, leakyBufSize)

func connect(conn net.Conn,stream rpcpb.SliverRPC_SocksProxyClient,frame *sliverpb.SocksData) {

	SocksConnPool.Store(frame.TunnelID,conn)
	//defer fmt.Printf("tcp close %q<--><-->%q", conn.LocalAddr(), conn.RemoteAddr())
	fmt.Printf("tcp conn %q<--><-->%q \n", conn.LocalAddr(), conn.RemoteAddr())

	buff := leakyBuf.Get()
	defer leakyBuf.Put(buff)
	var ToImplantSequence uint64 =0
	for {
		n, err := conn.Read(buff)
		if err != nil {
			if err == io.EOF {
				return
			}
			continue
		}
		if n > 0 {
			frame.Data = buff[:n]
			frame.Sequence = ToImplantSequence
			coreLog.Debugf("[socks] (User to Client) to Server to agent  Data Sequence %d , Data Size %d \n",ToImplantSequence,len(frame.Data))
			err := stream.Send(frame)
			if err != nil {
				return
			}
			ToImplantSequence++

		}

	}
}
