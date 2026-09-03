package transport

import (
	"errors"
	"sync"

	"google.golang.org/grpc"
)

// MultiplayerConnectMode controls how the client reaches multiplayer.
type MultiplayerConnectMode int

const (
	// MultiplayerConnectAuto follows the explicit WireGuard opt-in recorded in
	// the operator config.
	MultiplayerConnectAuto MultiplayerConnectMode = iota
	// MultiplayerConnectEnableWG forces WireGuard for complete configs,
	// including legacy configs that predate the explicit opt-in marker.
	MultiplayerConnectEnableWG
	// MultiplayerConnectDisableWG forces direct multiplayer mTLS.
	MultiplayerConnectDisableWG

	// MultiplayerConnectDirect is retained as a source-compatible alias for
	// callers that explicitly select direct multiplayer mTLS.
	MultiplayerConnectDirect = MultiplayerConnectDisableWG
	// MultiplayerConnectRequireWG is retained for source compatibility.
	// Deprecated: use MultiplayerConnectEnableWG.
	MultiplayerConnectRequireWG = MultiplayerConnectEnableWG
)

type connectionCloser interface {
	Close() error
}

var (
	multiplayerConnectModeMu sync.RWMutex
	multiplayerConnectMode   = MultiplayerConnectAuto
	connClosers              sync.Map // *grpc.ClientConn -> connectionCloser
)

// SetMultiplayerConnectMode configures future multiplayer client dials.
func SetMultiplayerConnectMode(mode MultiplayerConnectMode) {
	multiplayerConnectModeMu.Lock()
	multiplayerConnectMode = mode
	multiplayerConnectModeMu.Unlock()
}

func getMultiplayerConnectMode() MultiplayerConnectMode {
	multiplayerConnectModeMu.RLock()
	mode := multiplayerConnectMode
	multiplayerConnectModeMu.RUnlock()
	return mode
}

func registerConnCloser(conn *grpc.ClientConn, closer connectionCloser) {
	if conn == nil || closer == nil {
		return
	}
	connClosers.Store(conn, closer)
}

func unregisterConnCloser(conn *grpc.ClientConn) connectionCloser {
	if conn == nil {
		return nil
	}
	if closer, ok := connClosers.LoadAndDelete(conn); ok {
		return closer.(connectionCloser)
	}
	return nil
}

// CloseGRPCConnection closes a client connection and any transport-specific
// resources attached to it, such as the multiplayer WireGuard tunnel.
func CloseGRPCConnection(conn *grpc.ClientConn) error {
	if conn == nil {
		return nil
	}

	var errs []error
	errs = append(errs, conn.Close())
	if closer := unregisterConnCloser(conn); closer != nil {
		errs = append(errs, closer.Close())
	}
	return errors.Join(errs...)
}
