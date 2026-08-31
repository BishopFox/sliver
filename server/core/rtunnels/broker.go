package rtunnels

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
)

const DefaultDialTimeout = 10 * time.Second

var ErrNilDialConnection = errors.New("reverse port forward dialer returned a nil connection")

// ContextDialer is the only dependency allowed to create outbound reverse port
// forward connections. It is injectable so authorization-to-dial behavior can
// be tested without opening a real socket.
type ContextDialer interface {
	DialContext(ctx context.Context, network string, address string) (net.Conn, error)
}

// DialContextFunc adapts a function into a ContextDialer for focused tests.
type DialContextFunc func(ctx context.Context, network string, address string) (net.Conn, error)

func (dial DialContextFunc) DialContext(ctx context.Context, network string, address string) (net.Conn, error) {
	return dial(ctx, network, address)
}

// Broker resolves server-owned authorizations and opens their immutable dial
// plans. A caller can provide an implant address only for legacy lookup; that
// value is never passed to the dialer.
type Broker struct {
	registry    *Registry
	dialer      ContextDialer
	dialTimeout time.Duration
}

// NewBroker creates an outbound reverse port forward broker. Nil dependencies
// use production-safe defaults, and non-positive timeouts use DefaultDialTimeout.
func NewBroker(registry *Registry, dialer ContextDialer, dialTimeout time.Duration) *Broker {
	if registry == nil {
		registry = NewRegistry()
	}
	if dialer == nil {
		dialer = &net.Dialer{Timeout: DefaultDialTimeout}
	}
	if dialTimeout <= 0 {
		dialTimeout = DefaultDialTimeout
	}
	return &Broker{
		registry:    registry,
		dialer:      dialer,
		dialTimeout: dialTimeout,
	}
}

// Open resolves an authorization, dials only its stored operator destination,
// applies keepalive policy, and tracks the connection until it is closed or its
// authorization is revoked. Authorization revocation cancels pending dials and
// closes committed connections. The destination-bearing plan remains private;
// callers receive only the resolved capability for tunnel association.
func (broker *Broker) Open(ctx context.Context, sessionID string, id AuthorizationID, legacyAddress string) (net.Conn, AuthorizationID, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	plan, revoked, err := broker.registry.reserve(sessionID, id, legacyAddress)
	if err != nil {
		return nil, "", err
	}
	reservationActive := true
	defer func() {
		if reservationActive {
			broker.registry.releaseReservation(plan.AuthorizationID)
		}
	}()

	dialContext, cancel := context.WithTimeout(ctx, broker.dialTimeout)
	stopRevocationCancel := context.AfterFunc(revoked, cancel)
	connection, err := broker.dialer.DialContext(dialContext, "tcp", plan.Address)
	stopRevocationCancel()
	cancel()
	if err != nil {
		return nil, "", err
	}
	if connection == nil {
		return nil, "", ErrNilDialConnection
	}

	configureKeepAlive(connection, plan.KeepAlive)
	connectionID, err := broker.registry.registerConnection(sessionID, plan.AuthorizationID, connection)
	reservationActive = false
	if err != nil {
		_ = connection.Close()
		return nil, "", err
	}
	return &authorizedConn{
		Conn:          connection,
		registry:      broker.registry,
		authorization: plan.AuthorizationID,
		connectionID:  connectionID,
	}, plan.AuthorizationID, nil
}

func configureKeepAlive(connection net.Conn, keepAlive int32) {
	tcpConnection, ok := connection.(*net.TCPConn)
	if !ok {
		return
	}
	if keepAlive < 0 {
		_ = tcpConnection.SetKeepAlive(false)
		return
	}
	_ = tcpConnection.SetKeepAlive(true)
	if keepAlive == 0 {
		_ = tcpConnection.SetKeepAlivePeriod(30 * time.Second)
		return
	}
	_ = tcpConnection.SetKeepAlivePeriod(time.Duration(keepAlive) * time.Second)
}

type authorizedConn struct {
	net.Conn
	registry      *Registry
	authorization AuthorizationID
	connectionID  uint64
	closeOnce     sync.Once
	closeErr      error
}

func (connection *authorizedConn) Close() error {
	connection.closeOnce.Do(func() {
		connection.registry.unregisterConnection(connection.authorization, connection.connectionID)
		connection.closeErr = connection.Conn.Close()
	})
	return connection.closeErr
}

// DefaultRegistry and DefaultBroker are the production reverse port forward
// authorization components. Tests should prefer instance-owned registries.
var (
	DefaultRegistry = NewRegistry()
	DefaultBroker   = NewBroker(DefaultRegistry, nil, DefaultDialTimeout)
)
