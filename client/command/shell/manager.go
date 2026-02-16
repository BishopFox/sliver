package shell

import (
	"io"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/bishopfox/sliver/client/core"
)

type shellManager struct {
	mu     sync.RWMutex
	nextID int
	shells map[int]*managedShell
}

func newShellManager() *shellManager {
	return &shellManager{
		nextID: 1,
		shells: map[int]*managedShell{},
	}
}

var shells = newShellManager()

type shellState string

const (
	shellStateAttached shellState = "attached"
	shellStateDetached shellState = "detached"
	shellStateClosing  shellState = "closing"
)

type managedShell struct {
	ID          int
	SessionID   string
	SessionName string
	Path        string
	Pid         uint32
	TunnelID    uint64
	EnablePTY   bool
	CreatedAt   time.Time

	mu       sync.RWMutex
	state    shellState
	tunnel   *core.TunnelIO
	output   *swapWriter
	readerWG sync.WaitGroup
}

func (s *managedShell) State() shellState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func (s *managedShell) setState(state shellState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = state
}

func (s *managedShell) Tunnel() *core.TunnelIO {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tunnel
}

func (s *managedShell) SetOutput(w io.Writer) {
	s.output.Set(w)
}

func (s *managedShell) startReader(remove func()) {
	s.readerWG.Add(1)
	go func() {
		defer s.readerWG.Done()
		_, err := io.Copy(s.output, s.tunnel)
		log.Printf("Shell tunnel reader (id=%d tunnel=%d) exited: %v", s.ID, s.TunnelID, err)
		remove()
	}()
}

// swapWriter is an io.Writer whose underlying destination can be swapped at runtime.
// This is used so detached shells can keep draining output without clobbering the REPL.
type swapWriter struct {
	mu sync.RWMutex
	w  io.Writer
}

func newSwapWriter(w io.Writer) *swapWriter {
	return &swapWriter{w: w}
}

func (s *swapWriter) Set(w io.Writer) {
	s.mu.Lock()
	s.w = w
	s.mu.Unlock()
}

func (s *swapWriter) Write(p []byte) (int, error) {
	s.mu.RLock()
	w := s.w
	s.mu.RUnlock()
	if w == nil {
		return len(p), nil
	}
	return w.Write(p)
}

func (m *shellManager) Add(sh *managedShell) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := m.nextID
	m.nextID++

	sh.ID = id
	m.shells[id] = sh

	return id
}

func (m *shellManager) Remove(id int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.shells, id)
}

func (m *shellManager) Get(id int) (*managedShell, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sh, ok := m.shells[id]
	return sh, ok
}

func (m *shellManager) List() []*managedShell {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make([]*managedShell, 0, len(m.shells))
	for _, sh := range m.shells {
		results = append(results, sh)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].ID < results[j].ID
	})
	return results
}
