package transports

import (
	"net/url"
	"sync"

	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
)

type Connection struct {
	Send    chan *pb.Envelope
	Recv    chan *pb.Envelope
	IsOpen  bool
	ctrl    chan struct{}
	cleanup func()
	once    *sync.Once
	tunnels *map[uint64]*Tunnel
	mutex   *sync.RWMutex

	uri      *url.URL
	proxyURL *url.URL

	Start Start
	Stop  Stop
}

// URL - Get the c2 URL of the connection
func (c *Connection) URL() string {
	if c.uri == nil {
		return ""
	}
	return c.uri.String()
}

// ProxyURL - Get the c2 URL of the connection
func (c *Connection) ProxyURL() string {
	if c.proxyURL == nil {
		return ""
	}
	return c.proxyURL.String()
}

// Cleanup - Execute cleanup once
func (c *Connection) Cleanup() {
	c.once.Do(func() {
		c.cleanup()
		c.IsOpen = false
	})
}

// Tunnel - Add tunnel to mapping
func (c *Connection) Tunnel(ID uint64) *Tunnel {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return (*c.tunnels)[ID]
}

// AddTunnel - Add tunnel to mapping
func (c *Connection) AddTunnel(tun *Tunnel) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	(*c.tunnels)[tun.ID] = tun
}

// RemoveTunnel - Add tunnel to mapping
func (c *Connection) RemoveTunnel(ID uint64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(*c.tunnels, ID)
}

func (c *Connection) RequestResend(data []byte) {
	c.Send <- &pb.Envelope{
		Type: pb.MsgTunnelData,
		Data: data,
	}
}
