package core

import "sync"

var (
	// Portfwds - Holds refs to all port forwards
	Portfwds = portForwards{
		forwards: map[uint64]Portfwd{},
		mutex:    &sync.RWMutex{},
	}
)

type portForwards struct {
	forwards map[uint64]Portfwd
	mutex    *sync.RWMutex
}

func (p *portForwards) Add(tunnelID uint64, portfwd Portfwd) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.forwards[tunnelID] = portfwd
}

func (p *portForwards) Remove(tunnelID uint64) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	delete(p.forwards, tunnelID)
}

func (p *portForwards) List() []Portfwd {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	forwards := make([]Portfwd, 0, len(p.forwards))
	for _, forward := range p.forwards {
		forwards = append(forwards, forward)
	}
	return forwards
}

// Portfwd - Holds metadata about the portfwd
type Portfwd struct {
	TunnelID uint64
	Port     uint32
	Protocol string
}
