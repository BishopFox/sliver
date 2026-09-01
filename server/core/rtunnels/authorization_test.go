package rtunnels

import (
	"bytes"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanonicalizeAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "IPv4", input: "127.0.0.1:8080", expected: "127.0.0.1:8080"},
		{name: "mapped IPv4", input: "[::ffff:127.0.0.1]:080", expected: "127.0.0.1:80"},
		{name: "IPv6", input: "[2001:0db8::1]:443", expected: "[2001:db8::1]:443"},
		{name: "IPv6 zone", input: "[fe80::1%en0]:22", expected: "[fe80::1%en0]:22"},
		{name: "hostname", input: "Example.COM.:00443", expected: "example.com:443"},
		{name: "trim", input: " localhost:80 ", expected: "localhost:80"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := canonicalizeAddress(test.input)
			must.NoError(t, err)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestCanonicalizeAddressRejectsInvalidTargets(t *testing.T) {
	t.Parallel()

	for _, address := range []string{
		"",
		"localhost",
		":80",
		"localhost:0",
		"localhost:65536",
		"localhost:http",
		"bad host:80",
		"-invalid.example:80",
		"invalid..example:80",
		"127.0.0.1:80/path",
	} {
		t.Run(address, func(t *testing.T) {
			_, err := canonicalizeAddress(address)
			must.ErrorIs(t, err, ErrInvalidForwardAddress)
		})
	}
}

func TestRegistryBeginSpecCreatesImmutableStartingAuthorization(t *testing.T) {
	t.Parallel()

	random := bytes.NewReader(bytes.Repeat([]byte{0x42}, authorizationIDBytes))
	registry := newRegistry(random)
	id, err := registry.BeginSpec("session-a", "0.0.0.0:8443", "Example.COM.:080", 47)
	must.NoError(t, err)
	must.NotEmpty(t, id)
	assert.NotContains(t, id.String(), "+")
	assert.NotContains(t, id.String(), "/")

	authorization, ok := registry.Lookup("session-a", id)
	must.True(t, ok)
	assert.Equal(t, AuthorizationStarting, authorization.State)
	assert.Equal(t, "0.0.0.0:8443", authorization.BindAddress)
	assert.Equal(t, "example.com:80", authorization.Address)
	assert.Equal(t, int32(47), authorization.KeepAlive)
	assert.False(t, authorization.HasListenerID)

	// Snapshots and dial plans are values. Mutating either must not alter the
	// registry's immutable operator-derived destination.
	authorization.Address = "127.0.0.1:1"
	plan, err := acquirePlanForTest(registry, "session-a", id)
	must.NoError(t, err)
	plan.Address = "127.0.0.1:2"
	plan, err = acquirePlanForTest(registry, "session-a", id)
	must.NoError(t, err)
	assert.Equal(t, "example.com:80", plan.Address)
}

func TestRegistryBeginValidationAndRandomFailure(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	_, err := registry.Begin("", "127.0.0.1:80", 0)
	must.ErrorIs(t, err, ErrInvalidSessionID)
	_, err = registry.Begin("session", "127.0.0.1", 0)
	must.ErrorIs(t, err, ErrInvalidForwardAddress)

	registry = newRegistry(errorReader{})
	_, err = registry.Begin("session", "127.0.0.1:80", 0)
	must.ErrorIs(t, err, ErrAuthorizationIDGeneration)
}

func TestRegistryActivationDuplicateListenerFailsClosed(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	firstID, err := registry.Begin("session", "127.0.0.1:8001", 10)
	must.NoError(t, err)
	must.NoError(t, registry.Activate("session", firstID, 7))
	must.NoError(t, registry.Activate("session", firstID, 7), "same transition is idempotent")

	secondID, err := registry.Begin("session", "127.0.0.1:8002", 20)
	must.NoError(t, err)
	err = registry.Activate("session", secondID, 7)
	must.ErrorIs(t, err, ErrDuplicateListenerID)

	first, ok := registry.LookupListener("session", 7)
	must.True(t, ok)
	assert.Equal(t, firstID, first.AuthorizationID)
	assert.Equal(t, "127.0.0.1:8001", first.Address)
	_, ok = registry.Lookup("session", secondID)
	assert.False(t, ok, "rejected candidate must be purged rather than retained as a tombstone")
	_, err = acquirePlanForTest(registry, "session", secondID)
	must.ErrorIs(t, err, ErrUnknownAuthorization)

	listed := registry.List("session")
	must.Len(t, listed, 1)
	assert.Equal(t, firstID, listed[0].AuthorizationID)
}

func TestRegistryAuthorizationIsSessionScoped(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	id, err := registry.Begin("owner", "127.0.0.1:9000", 0)
	must.NoError(t, err)
	_, err = acquirePlanForTest(registry, "attacker", id)
	must.ErrorIs(t, err, ErrAuthorizationSession)
	err = registry.Activate("attacker", id, 1)
	must.ErrorIs(t, err, ErrAuthorizationSession)
	assert.False(t, registry.Revoke("attacker", id))

	plan, err := acquirePlanForTest(registry, "owner", id)
	must.NoError(t, err)
	assert.Equal(t, "127.0.0.1:9000", plan.Address)
}

func TestRegistryLegacyLookupReturnsStoredCanonicalDestination(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	firstID, err := registry.Begin("session", "Example.COM.:080", 10)
	must.NoError(t, err)
	secondID, err := registry.Begin("session", "example.com:80", 20)
	must.NoError(t, err)

	plan, err := acquireLegacyPlanForTest(registry, "session", "EXAMPLE.COM.:00080")
	must.NoError(t, err)
	assert.Equal(t, secondID, plan.AuthorizationID)
	assert.Equal(t, "example.com:80", plan.Address, "wire spelling must never become the dial target")
	assert.Equal(t, int32(20), plan.KeepAlive)

	must.True(t, registry.Revoke("session", secondID))
	plan, err = acquireLegacyPlanForTest(registry, "session", "example.com:80")
	must.NoError(t, err)
	assert.Equal(t, firstID, plan.AuthorizationID)
	assert.Equal(t, int32(10), plan.KeepAlive)

	_, err = acquireLegacyPlanForTest(registry, "session", "127.0.0.1:80")
	must.ErrorIs(t, err, ErrUnknownAuthorization)
}

func TestRegistryNegotiatedAuthorizationIDDisablesLegacyLookup(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	id, err := registry.Begin("session", "Example.COM.:443", 0)
	must.NoError(t, err)
	must.NoError(t, registry.ActivateProtocol("session", id, 7, true))

	authorization, ok := registry.LookupListener("session", 7)
	must.True(t, ok)
	assert.True(t, authorization.RequiresAuthorizationID)

	_, err = acquireLegacyPlanForTest(registry, "session", "example.com:443")
	must.ErrorIs(t, err, ErrAuthorizationIDRequired)
	plan, err := acquirePlanForTest(registry, "session", id)
	must.NoError(t, err)
	assert.Equal(t, "example.com:443", plan.Address)
}

func TestRegistryAuthorizationConnectionLimitIncludesPendingAndActive(t *testing.T) {
	assert.Equal(t, 64, maxConnectionsPerAuthorization)
	assert.Equal(t, 256, maxConnectionsPerSession)
	assert.Equal(t, 2048, maxConnectionsGlobal)

	registry := NewRegistry()
	id, err := registry.Begin("session", "127.0.0.1:8080", 0)
	must.NoError(t, err)

	reservations := make([]*connectionReservation, 0, maxConnectionsPerAuthorization)
	for range maxConnectionsPerAuthorization {
		_, _, reservation, reserveErr := registry.reserve("session", id, "")
		must.NoError(t, reserveErr)
		reservations = append(reservations, reservation)
	}

	clients := make([]net.Conn, 0, maxConnectionsPerAuthorization/2)
	peers := make([]net.Conn, 0, maxConnectionsPerAuthorization/2)
	connectionIDs := make([]uint64, 0, maxConnectionsPerAuthorization/2)
	for index := 0; index < maxConnectionsPerAuthorization/2; index++ {
		client, peer := net.Pipe()
		connectionID, registerErr := registry.registerConnection("session", id, reservations[index], client)
		must.NoError(t, registerErr)
		clients = append(clients, client)
		peers = append(peers, peer)
		connectionIDs = append(connectionIDs, connectionID)
	}
	t.Cleanup(func() {
		for _, connection := range clients {
			_ = connection.Close()
		}
		for _, connection := range peers {
			_ = connection.Close()
		}
	})

	assertRegistryConnectionCounts(t, registry, 64, map[string]uint64{"session": 64})
	registry.mu.RLock()
	record := registry.byID[id]
	if record == nil {
		registry.mu.RUnlock()
		t.Fatal("authorization record disappeared before revocation")
	}
	assert.Equal(t, uint64(32), record.pendingConnections)
	assert.Len(t, record.connections, 32)
	registry.mu.RUnlock()

	_, _, _, err = registry.reserve("session", id, "")
	must.ErrorIs(t, err, ErrAuthorizationConnectionLimit)

	registry.releaseReservation(reservations[32])
	_, _, replacementPending, err := registry.reserve("session", id, "")
	must.NoError(t, err)
	registry.releaseReservation(replacementPending)
	assertRegistryConnectionCounts(t, registry, 63, map[string]uint64{"session": 63})

	registry.unregisterConnection(record, connectionIDs[0])
	_ = clients[0].Close()
	_, _, replacementActive, err := registry.reserve("session", id, "")
	must.NoError(t, err)
	registry.releaseReservation(replacementActive)
	assertRegistryConnectionCounts(t, registry, 62, map[string]uint64{"session": 62})

	must.True(t, registry.Revoke("session", id))
	for _, reservation := range reservations {
		registry.releaseReservation(reservation)
	}
	assertRegistryConnectionCounts(t, registry, 0, nil)
}

func TestRegistrySessionConnectionLimitIsScopedAndReclaimable(t *testing.T) {
	registry := NewRegistry()
	reservations := make([]*connectionReservation, 0, maxConnectionsPerSession)
	for authIndex := 0; authIndex < 8; authIndex++ {
		id, err := registry.Begin("full-session", "127.0.0.1:8080", 0)
		must.NoError(t, err)
		for range 32 {
			_, _, reservation, reserveErr := registry.reserve("full-session", id, "")
			must.NoError(t, reserveErr)
			reservations = append(reservations, reservation)
		}
	}
	assertRegistryConnectionCounts(t, registry, 256, map[string]uint64{"full-session": 256})

	overflowID, err := registry.Begin("full-session", "127.0.0.1:8081", 0)
	must.NoError(t, err)
	_, _, _, err = registry.reserve("full-session", overflowID, "")
	must.ErrorIs(t, err, ErrSessionConnectionLimit)

	otherID, err := registry.Begin("other-session", "127.0.0.1:8082", 0)
	must.NoError(t, err)
	_, _, otherReservation, err := registry.reserve("other-session", otherID, "")
	must.NoError(t, err)
	assertRegistryConnectionCounts(t, registry, 257, map[string]uint64{"full-session": 256, "other-session": 1})
	registry.releaseReservation(otherReservation)

	registry.releaseReservation(reservations[0])
	_, _, reclaimed, err := registry.reserve("full-session", overflowID, "")
	must.NoError(t, err)
	registry.releaseReservation(reclaimed)
	assertRegistryConnectionCounts(t, registry, 255, map[string]uint64{"full-session": 255})

	assert.Equal(t, 9, registry.RevokeSession("full-session"))
	registry.RevokeSession("other-session")
	assertRegistryConnectionCounts(t, registry, 0, nil)
}

func TestRegistryGlobalConnectionLimitIsReclaimable(t *testing.T) {
	registry := NewRegistry()
	filledSessions := make([]string, 0, 8)
	var firstReservation *connectionReservation
	for sessionIndex := 0; sessionIndex < 8; sessionIndex++ {
		sessionID := "global-session-" + string(rune('a'+sessionIndex))
		filledSessions = append(filledSessions, sessionID)
		for authIndex := 0; authIndex < 4; authIndex++ {
			id, err := registry.Begin(sessionID, "127.0.0.1:8080", 0)
			must.NoError(t, err)
			for range maxConnectionsPerAuthorization {
				_, _, reservation, reserveErr := registry.reserve(sessionID, id, "")
				must.NoError(t, reserveErr)
				if firstReservation == nil {
					firstReservation = reservation
				}
			}
		}
	}
	wantSessions := map[string]uint64{}
	for _, sessionID := range filledSessions {
		wantSessions[sessionID] = maxConnectionsPerSession
	}
	assertRegistryConnectionCounts(t, registry, 2048, wantSessions)

	overflowID, err := registry.Begin("overflow-session", "127.0.0.1:8081", 0)
	must.NoError(t, err)
	_, _, _, err = registry.reserve("overflow-session", overflowID, "")
	must.ErrorIs(t, err, ErrGlobalConnectionLimit)

	registry.releaseReservation(firstReservation)
	_, _, reclaimed, err := registry.reserve("overflow-session", overflowID, "")
	must.NoError(t, err)
	registry.releaseReservation(reclaimed)
	assertRegistryConnectionCounts(t, registry, maxConnectionsGlobal-1, map[string]uint64{
		"global-session-a": maxConnectionsPerSession - 1,
		"global-session-b": maxConnectionsPerSession,
		"global-session-c": maxConnectionsPerSession,
		"global-session-d": maxConnectionsPerSession,
		"global-session-e": maxConnectionsPerSession,
		"global-session-f": maxConnectionsPerSession,
		"global-session-g": maxConnectionsPerSession,
		"global-session-h": maxConnectionsPerSession,
	})

	for _, sessionID := range filledSessions {
		registry.RevokeSession(sessionID)
	}
	registry.RevokeSession("overflow-session")
	assertRegistryConnectionCounts(t, registry, 0, nil)
}

func TestRegistryRegisterAndRevokeRaceReclaimsConnectionQuota(t *testing.T) {
	for iteration := 0; iteration < 100; iteration++ {
		registry := NewRegistry()
		id, err := registry.Begin("session", "127.0.0.1:8080", 0)
		must.NoError(t, err)
		_, _, reservation, err := registry.reserve("session", id, "")
		must.NoError(t, err)

		client, peer := net.Pipe()
		start := make(chan struct{})
		var wait sync.WaitGroup
		wait.Add(2)
		go func() {
			defer wait.Done()
			<-start
			_, _ = registry.registerConnection("session", id, reservation, client)
		}()
		go func() {
			defer wait.Done()
			<-start
			registry.Revoke("session", id)
		}()
		close(start)
		wait.Wait()
		registry.releaseReservation(reservation)
		_ = client.Close()
		_ = peer.Close()
		assertRegistryConnectionCounts(t, registry, 0, nil)
	}
}

func TestRegistryLateReservationReleaseCannotAffectReusedAuthorizationID(t *testing.T) {
	random := bytes.NewReader(bytes.Repeat([]byte{0x42}, authorizationIDBytes*2))
	registry := newRegistry(random)
	oldID, err := registry.Begin("session", "127.0.0.1:8080", 0)
	must.NoError(t, err)
	_, _, oldReservation, err := registry.reserve("session", oldID, "")
	must.NoError(t, err)
	must.True(t, registry.Revoke("session", oldID))

	newID, err := registry.Begin("session", "127.0.0.1:8081", 0)
	must.NoError(t, err)
	assert.Equal(t, oldID, newID)
	_, _, newReservation, err := registry.reserve("session", newID, "")
	must.NoError(t, err)
	registry.releaseReservation(oldReservation)
	assertRegistryConnectionCounts(t, registry, 1, map[string]uint64{"session": 1})
	registry.releaseReservation(newReservation)
	assertRegistryConnectionCounts(t, registry, 0, nil)
}

func TestRegistryRevokeAndActivateRaceFailsClosed(t *testing.T) {
	t.Parallel()

	for iteration := 0; iteration < 100; iteration++ {
		registry := NewRegistry()
		id, err := registry.Begin("session", "127.0.0.1:9000", 0)
		must.NoError(t, err)

		start := make(chan struct{})
		var activateErr error
		var wait sync.WaitGroup
		wait.Add(2)
		go func() {
			defer wait.Done()
			<-start
			activateErr = registry.Activate("session", id, 1)
		}()
		go func() {
			defer wait.Done()
			<-start
			registry.Revoke("session", id)
		}()
		close(start)
		wait.Wait()

		if activateErr != nil {
			if !errors.Is(activateErr, ErrAuthorizationRevoked) && !errors.Is(activateErr, ErrUnknownAuthorization) {
				t.Fatalf("Activate() race error = %v, want revoked or unknown", activateErr)
			}
		}
		_, ok := registry.Lookup("session", id)
		assert.False(t, ok)
		_, err = acquirePlanForTest(registry, "session", id)
		must.ErrorIs(t, err, ErrUnknownAuthorization)
	}
}

func TestRegistryRevokeSessionIsScopedAndIdempotent(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	firstID, err := registry.Begin("first", "127.0.0.1:8001", 0)
	must.NoError(t, err)
	secondID, err := registry.Begin("first", "127.0.0.1:8002", 0)
	must.NoError(t, err)
	otherID, err := registry.Begin("other", "127.0.0.1:8003", 0)
	must.NoError(t, err)
	must.NoError(t, registry.Activate("first", firstID, 1))
	must.NoError(t, registry.Activate("first", secondID, 2))
	must.NoError(t, registry.Activate("other", otherID, 1))

	assert.Equal(t, 2, registry.RevokeSession("first"))
	assert.Equal(t, 0, registry.RevokeSession("first"))
	_, ok := registry.Lookup("first", firstID)
	assert.False(t, ok)
	assert.Empty(t, registry.List("first"))
	plan, err := acquirePlanForTest(registry, "other", otherID)
	must.NoError(t, err)
	assert.Equal(t, "127.0.0.1:8003", plan.Address)
}

func TestRegistryGeneratesUniqueOpaqueIDs(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	seen := map[AuthorizationID]struct{}{}
	for index := 0; index < 256; index++ {
		id, err := registry.Begin("session", "127.0.0.1:8080", 0)
		must.NoError(t, err)
		assert.Len(t, id, 43)
		assert.False(t, strings.ContainsAny(id.String(), "+/="))
		_, duplicate := seen[id]
		assert.False(t, duplicate)
		seen[id] = struct{}{}
	}
}

type errorReader struct{}

func (errorReader) Read(_ []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func TestAuthorizationErrorsAreComparable(t *testing.T) {
	t.Parallel()

	_, err := canonicalizeAddress("invalid")
	assert.True(t, errors.Is(err, ErrInvalidForwardAddress))
}

func acquirePlanForTest(registry *Registry, sessionID string, id AuthorizationID) (dialPlan, error) {
	plan, _, reservation, err := registry.reserve(sessionID, id, "")
	if err == nil {
		registry.releaseReservation(reservation)
	}
	return plan, err
}

func acquireLegacyPlanForTest(registry *Registry, sessionID string, address string) (dialPlan, error) {
	plan, _, reservation, err := registry.reserve(sessionID, "", address)
	if err == nil {
		registry.releaseReservation(reservation)
	}
	return plan, err
}

func assertRegistryConnectionCounts(t *testing.T, registry *Registry, total uint64, sessions map[string]uint64) {
	t.Helper()
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	assert.Equal(t, total, registry.totalConnectionCount)
	if sessions == nil {
		assert.Empty(t, registry.sessionConnectionCounts)
		return
	}
	assert.Equal(t, sessions, registry.sessionConnectionCounts)
}
