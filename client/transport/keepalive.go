package transport

import (
	"sync"

	"google.golang.org/grpc/keepalive"
)

var (
	kaMu     sync.RWMutex
	kaParams *keepalive.ClientParameters
)

// SetKeepaliveParams configures gRPC keepalive pings for future MTLSConnect dials.
// Behavior is unchanged unless this is called.
func SetKeepaliveParams(p keepalive.ClientParameters) {
	kaMu.Lock()
	kaParams = &p
	kaMu.Unlock()
}

func getKeepaliveParams() (keepalive.ClientParameters, bool) {
	kaMu.RLock()
	p := kaParams
	kaMu.RUnlock()
	if p == nil {
		return keepalive.ClientParameters{}, false
	}
	return *p, true
}
