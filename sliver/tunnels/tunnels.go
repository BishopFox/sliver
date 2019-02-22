package tunnels

import (
	"io"
	"sync"
)

// Tunnel - Duplex byte read/write
type Tunnel struct {
	ID     uint64
	Reader io.ReadCloser
	Writer io.WriteCloser
}

type tunnels struct {
	tunnels *map[uint64]*Tunnel
	mutex   *sync.RWMutex
}

// Tunnels - Holds refs to all tunnels
var Tunnels = tunnels{
	tunnels: &map[uint64]*Tunnel{},
}

func (t *tunnels) start(tun *Tunnel) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	(*t.tunnels)[tun.ID] = tun

	go func() {

	}()
}

func (t *tunnels) Get(ID uint64) *Tunnel {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return (*t.tunnels)[ID]
}

func (t *tunnels) Remove(ID uint64) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	delete(*t.tunnels, ID)
}
