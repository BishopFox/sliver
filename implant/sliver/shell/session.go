package shell

import (
	"sync"
	"time"
)

// Session tracks the process associated with an interactive-shell tunnel.
// Tunnel-close messages are handled independently from shell requests, so the
// registry lets teardown stop the owning process instead of only closing pipes.
type Session struct {
	Shell    *Shell
	stopOnce sync.Once
}

// NewSession associates a system shell with its tunnel lifecycle state.
func NewSession(systemShell *Shell) *Session {
	return &Session{Shell: systemShell}
}

// Stop closes shell input and cancels the process exactly once.
func (s *Session) Stop() {
	if s == nil || s.Shell == nil {
		return
	}

	s.stopOnce.Do(func() {
		if s.Shell.Stdin != nil {
			_ = s.Shell.Stdin.Close()
		}
		s.Shell.Stop()
	})
}

var sessions = struct {
	mutex                sync.RWMutex
	items                map[uint64]*Session
	closedBeforeRegister map[uint64]time.Time
}{
	items:                map[uint64]*Session{},
	closedBeforeRegister: map[uint64]time.Time{},
}

const closedBeforeRegisterTTL = time.Minute

// RegisterSession atomically publishes a started shell. It returns false and
// stops the process if a tunnel-close handler ran before registration.
func RegisterSession(tunnelID uint64, session *Session) bool {
	if session == nil {
		return false
	}

	sessions.mutex.Lock()
	pruneClosedBeforeRegister(time.Now())
	if _, closed := sessions.closedBeforeRegister[tunnelID]; closed {
		delete(sessions.closedBeforeRegister, tunnelID)
		sessions.mutex.Unlock()
		session.Stop()
		return false
	}
	sessions.items[tunnelID] = session
	sessions.mutex.Unlock()
	return true
}

// UnregisterSession removes the session and startup tombstone for tunnelID.
func UnregisterSession(tunnelID uint64) {
	sessions.mutex.Lock()
	delete(sessions.items, tunnelID)
	delete(sessions.closedBeforeRegister, tunnelID)
	sessions.mutex.Unlock()
}

// StopSession requests termination of the shell owned by tunnelID.
func StopSession(tunnelID uint64) bool {
	sessions.mutex.Lock()
	session := sessions.items[tunnelID]
	if session == nil {
		now := time.Now()
		pruneClosedBeforeRegister(now)
		sessions.closedBeforeRegister[tunnelID] = now
		sessions.mutex.Unlock()
		return false
	}
	sessions.mutex.Unlock()

	session.Stop()
	return true
}

// pruneClosedBeforeRegister runs with sessions.mutex held. Tombstones only
// bridge the short concurrent-handler startup window and must not accumulate
// for the lifetime of an implant.
func pruneClosedBeforeRegister(now time.Time) {
	for tunnelID, closedAt := range sessions.closedBeforeRegister {
		if now.Sub(closedAt) >= closedBeforeRegisterTTL {
			delete(sessions.closedBeforeRegister, tunnelID)
		}
	}
}
