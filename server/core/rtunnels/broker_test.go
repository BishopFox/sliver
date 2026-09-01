package rtunnels

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBrokerOpenDialsOnlyStoredOperatorAddress(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	id, err := registry.Begin("session", "Operator.Example.:080", 19)
	must.NoError(t, err)

	var network string
	var address string
	client, peer := net.Pipe()
	t.Cleanup(func() {
		_ = peer.Close()
	})
	dialer := DialContextFunc(func(_ context.Context, dialNetwork string, dialAddress string) (net.Conn, error) {
		network = dialNetwork
		address = dialAddress
		return client, nil
	})
	broker := NewBroker(registry, dialer, time.Second)

	connection, resolvedID, err := broker.Open(context.Background(), "session", id, "127.0.0.1:1")
	must.NoError(t, err)
	t.Cleanup(func() {
		_ = connection.Close()
	})
	assert.Equal(t, "tcp", network)
	assert.Equal(t, "operator.example:80", address)
	assert.Equal(t, id, resolvedID)
}

func TestBrokerLegacyOpenUsesWireAddressOnlyAsLookupKey(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	id, err := registry.Begin("session", "Example.COM.:080", 0)
	must.NoError(t, err)

	client, peer := net.Pipe()
	t.Cleanup(func() {
		_ = peer.Close()
	})
	var dialed string
	broker := NewBroker(registry, DialContextFunc(func(_ context.Context, _ string, address string) (net.Conn, error) {
		dialed = address
		return client, nil
	}), time.Second)

	connection, resolvedID, err := broker.Open(context.Background(), "session", "", "EXAMPLE.COM.:00080")
	must.NoError(t, err)
	t.Cleanup(func() {
		_ = connection.Close()
	})
	assert.Equal(t, id, resolvedID)
	assert.Equal(t, "example.com:80", dialed)
	assert.NotEqual(t, "EXAMPLE.COM.:00080", dialed)
}

func TestBrokerRejectsUnknownAndCrossSessionIDsWithoutDialing(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	id, err := registry.Begin("owner", "127.0.0.1:8080", 0)
	must.NoError(t, err)
	dials := 0
	broker := NewBroker(registry, DialContextFunc(func(_ context.Context, _ string, _ string) (net.Conn, error) {
		dials++
		return nil, errors.New("unexpected dial")
	}), time.Second)

	_, _, err = broker.Open(context.Background(), "owner", AuthorizationID("unknown"), "127.0.0.1:1")
	must.ErrorIs(t, err, ErrUnknownAuthorization)
	_, _, err = broker.Open(context.Background(), "attacker", id, "127.0.0.1:1")
	must.ErrorIs(t, err, ErrAuthorizationSession)
	_, _, err = broker.Open(context.Background(), "owner", "", "127.0.0.1:1")
	must.ErrorIs(t, err, ErrUnknownAuthorization)
	assert.Zero(t, dials)
}

func TestBrokerAppliesBoundedDialContext(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	id, err := registry.Begin("session", "127.0.0.1:8080", 0)
	must.NoError(t, err)
	timeout := 250 * time.Millisecond
	var remaining time.Duration
	broker := NewBroker(registry, DialContextFunc(func(ctx context.Context, _ string, _ string) (net.Conn, error) {
		deadline, ok := ctx.Deadline()
		must.True(t, ok)
		remaining = time.Until(deadline)
		return nil, context.DeadlineExceeded
	}), timeout)

	_, _, err = broker.Open(context.Background(), "session", id, "")
	must.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Greater(t, remaining, time.Duration(0))
	assert.LessOrEqual(t, remaining, timeout)
}

func TestBrokerRevokeCancelsPendingDial(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	id, err := registry.Begin("session", "127.0.0.1:8080", 0)
	must.NoError(t, err)
	started := make(chan struct{})
	broker := NewBroker(registry, DialContextFunc(func(ctx context.Context, _ string, _ string) (net.Conn, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	}), time.Minute)

	result := make(chan error, 1)
	go func() {
		_, _, openErr := broker.Open(context.Background(), "session", id, "")
		result <- openErr
	}()
	<-started
	must.True(t, registry.Revoke("session", id))

	select {
	case openErr := <-result:
		must.ErrorIs(t, openErr, ErrAuthorizationRevoked)
	case <-time.After(time.Second):
		t.Fatal("pending dial was not canceled by revocation")
	}
}

func TestBrokerRevokeClosesActiveConnections(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	id, err := registry.Begin("session", "127.0.0.1:8080", 0)
	must.NoError(t, err)
	client, peer := net.Pipe()
	t.Cleanup(func() {
		_ = peer.Close()
	})
	broker := NewBroker(registry, DialContextFunc(func(_ context.Context, _ string, _ string) (net.Conn, error) {
		return client, nil
	}), time.Second)

	connection, _, err := broker.Open(context.Background(), "session", id, "")
	must.NoError(t, err)
	assert.NotNil(t, connection)
	must.True(t, registry.Revoke("session", id))

	_ = peer.SetReadDeadline(time.Now().Add(time.Second))
	_, err = peer.Read(make([]byte, 1))
	must.ErrorIs(t, err, io.EOF)
	_, _, err = broker.Open(context.Background(), "session", id, "")
	must.ErrorIs(t, err, ErrUnknownAuthorization)
}

func TestBrokerListenerAndSessionRevocationCloseOnlyOwnedConnections(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	listenerID, err := registry.Begin("session", "127.0.0.1:8001", 0)
	must.NoError(t, err)
	must.NoError(t, registry.Activate("session", listenerID, 17))
	sessionID, err := registry.Begin("session", "127.0.0.1:8002", 0)
	must.NoError(t, err)
	otherID, err := registry.Begin("other", "127.0.0.1:8003", 0)
	must.NoError(t, err)

	peers := make(chan net.Conn, 3)
	broker := NewBroker(registry, DialContextFunc(func(_ context.Context, _ string, _ string) (net.Conn, error) {
		client, peer := net.Pipe()
		peers <- peer
		return client, nil
	}), time.Second)
	_, _, err = broker.Open(context.Background(), "session", listenerID, "")
	must.NoError(t, err)
	listenerPeer := <-peers
	sessionConnection, _, err := broker.Open(context.Background(), "session", sessionID, "")
	must.NoError(t, err)
	sessionPeer := <-peers
	otherConnection, _, err := broker.Open(context.Background(), "other", otherID, "")
	must.NoError(t, err)
	otherPeer := <-peers
	t.Cleanup(func() {
		_ = listenerPeer.Close()
		_ = sessionPeer.Close()
		_ = otherPeer.Close()
		_ = sessionConnection.Close()
		_ = otherConnection.Close()
	})

	must.True(t, registry.RevokeListener("session", 17))
	assertConnectionClosed(t, listenerPeer)
	assertConnectionOpen(t, sessionPeer, sessionConnection)

	assert.Equal(t, 1, registry.RevokeSession("session"))
	assertConnectionClosed(t, sessionPeer)
	assertConnectionOpen(t, otherPeer, otherConnection)
}

func TestBrokerDialCompletingAfterRevokeIsClosedAndRejected(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	id, err := registry.Begin("session", "127.0.0.1:8080", 0)
	must.NoError(t, err)
	started := make(chan struct{})
	release := make(chan struct{})
	peerReady := make(chan net.Conn, 1)
	broker := NewBroker(registry, DialContextFunc(func(_ context.Context, _ string, _ string) (net.Conn, error) {
		close(started)
		<-release // Deliberately ignore context cancellation to exercise commit.
		client, peer := net.Pipe()
		peerReady <- peer
		return client, nil
	}), time.Minute)

	result := make(chan error, 1)
	go func() {
		_, _, openErr := broker.Open(context.Background(), "session", id, "")
		result <- openErr
	}()
	<-started
	must.True(t, registry.Revoke("session", id))
	close(release)
	peer := <-peerReady
	t.Cleanup(func() {
		_ = peer.Close()
	})

	select {
	case openErr := <-result:
		must.ErrorIs(t, openErr, ErrAuthorizationRevoked)
	case <-time.After(time.Second):
		t.Fatal("dial completion did not fail closed after revocation")
	}
	_ = peer.SetReadDeadline(time.Now().Add(time.Second))
	_, err = peer.Read(make([]byte, 1))
	must.ErrorIs(t, err, io.EOF)
}

func TestBrokerAllowsMultipleConnectionsUntilRevocation(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	id, err := registry.Begin("session", "127.0.0.1:8080", 0)
	must.NoError(t, err)
	var mutex sync.Mutex
	peers := make([]net.Conn, 0, 2)
	broker := NewBroker(registry, DialContextFunc(func(_ context.Context, _ string, _ string) (net.Conn, error) {
		client, peer := net.Pipe()
		mutex.Lock()
		peers = append(peers, peer)
		mutex.Unlock()
		return client, nil
	}), time.Second)

	first, _, err := broker.Open(context.Background(), "session", id, "")
	must.NoError(t, err)
	second, _, err := broker.Open(context.Background(), "session", id, "")
	must.NoError(t, err)
	assert.NotNil(t, first)
	assert.NotNil(t, second)
	must.True(t, registry.Revoke("session", id))

	mutex.Lock()
	defer mutex.Unlock()
	must.Len(t, peers, 2)
	for _, peer := range peers {
		_ = peer.SetReadDeadline(time.Now().Add(time.Second))
		_, readErr := peer.Read(make([]byte, 1))
		must.ErrorIs(t, readErr, io.EOF)
		_ = peer.Close()
	}
}

func TestBrokerRejectsNilConnection(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	id, err := registry.Begin("session", "127.0.0.1:8080", 0)
	must.NoError(t, err)
	broker := NewBroker(registry, DialContextFunc(func(_ context.Context, _ string, _ string) (net.Conn, error) {
		return nil, nil
	}), time.Second)

	_, _, err = broker.Open(context.Background(), "session", id, "")
	must.ErrorIs(t, err, ErrNilDialConnection)
}

func TestBrokerOpenFailuresReleaseConnectionReservation(t *testing.T) {
	tests := []struct {
		name    string
		dialErr error
		wantErr error
	}{
		{name: "dial error", dialErr: errors.New("dial failed"), wantErr: errors.New("dial failed")},
		{name: "nil connection", wantErr: ErrNilDialConnection},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry := NewRegistry()
			id, err := registry.Begin("session", "127.0.0.1:8080", 0)
			must.NoError(t, err)
			broker := NewBroker(registry, DialContextFunc(func(_ context.Context, _ string, _ string) (net.Conn, error) {
				return nil, test.dialErr
			}), time.Second)

			for range maxConnectionsPerAuthorization + 1 {
				_, _, openErr := broker.Open(context.Background(), "session", id, "")
				if test.dialErr != nil {
					assert.EqualError(t, openErr, test.dialErr.Error())
				} else {
					must.ErrorIs(t, openErr, test.wantErr)
				}
				assertRegistryConnectionCounts(t, registry, 0, nil)
			}
		})
	}
}

func TestBrokerRegisterErrorClosesConnectionAndReleasesReservation(t *testing.T) {
	registry := NewRegistry()
	id, err := registry.Begin("session", "127.0.0.1:8080", 0)
	must.NoError(t, err)
	registerErr := errors.New("register failed")
	peers := make(chan net.Conn, maxConnectionsPerAuthorization+1)
	broker := NewBroker(registry, DialContextFunc(func(_ context.Context, _ string, _ string) (net.Conn, error) {
		client, peer := net.Pipe()
		peers <- peer
		return client, nil
	}), time.Second)
	broker.registerConnection = func(string, AuthorizationID, *connectionReservation, net.Conn) (uint64, error) {
		return 0, registerErr
	}

	for range maxConnectionsPerAuthorization + 1 {
		_, _, openErr := broker.Open(context.Background(), "session", id, "")
		must.ErrorIs(t, openErr, registerErr)
		peer := <-peers
		assertConnectionClosed(t, peer)
		_ = peer.Close()
		assertRegistryConnectionCounts(t, registry, 0, nil)
	}
}

func TestBrokerRejectsConnectionReturnedAfterDialContextCancellation(t *testing.T) {
	registry := NewRegistry()
	id, err := registry.Begin("session", "127.0.0.1:8080", 0)
	must.NoError(t, err)
	started := make(chan struct{})
	peerReady := make(chan net.Conn, 1)
	broker := NewBroker(registry, DialContextFunc(func(ctx context.Context, _ string, _ string) (net.Conn, error) {
		close(started)
		<-ctx.Done()
		client, peer := net.Pipe()
		peerReady <- peer
		return client, nil
	}), time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, _, openErr := broker.Open(ctx, "session", id, "")
		result <- openErr
	}()
	<-started
	cancel()
	peer := <-peerReady
	defer peer.Close()
	select {
	case openErr := <-result:
		must.ErrorIs(t, openErr, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("Open did not reject a connection returned after cancellation")
	}
	assertConnectionClosed(t, peer)
	assertRegistryConnectionCounts(t, registry, 0, nil)
}

func assertConnectionClosed(t *testing.T, connection net.Conn) {
	t.Helper()
	_ = connection.SetReadDeadline(time.Now().Add(time.Second))
	_, err := connection.Read(make([]byte, 1))
	must.ErrorIs(t, err, io.EOF)
}

func assertConnectionOpen(t *testing.T, peer net.Conn, connection net.Conn) {
	t.Helper()
	_ = peer.SetReadDeadline(time.Now().Add(time.Second))
	writeDone := make(chan error, 1)
	go func() {
		_, err := connection.Write([]byte{0x42})
		writeDone <- err
	}()
	buffer := make([]byte, 1)
	_, err := peer.Read(buffer)
	must.NoError(t, err)
	assert.Equal(t, byte(0x42), buffer[0])
	must.NoError(t, <-writeDone)
}
