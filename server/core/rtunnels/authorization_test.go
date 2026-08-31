package rtunnels

import (
	"bytes"
	"errors"
	"io"
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

func TestRegistryBoundsPendingAndActiveConnectionsPerAuthorization(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	id, err := registry.Begin("session", "127.0.0.1:8080", 0)
	must.NoError(t, err)
	for range maxConnectionsPerAuthorization {
		_, _, err := registry.reserve("session", id, "")
		must.NoError(t, err)
	}
	_, _, err = registry.reserve("session", id, "")
	must.ErrorIs(t, err, ErrAuthorizationConnectionLimit)
	for range maxConnectionsPerAuthorization {
		registry.releaseReservation(id)
	}
	plan, _, err := registry.reserve("session", id, "")
	must.NoError(t, err)
	registry.releaseReservation(plan.AuthorizationID)
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
	plan, _, err := registry.reserve(sessionID, id, "")
	if err == nil {
		registry.releaseReservation(plan.AuthorizationID)
	}
	return plan, err
}

func acquireLegacyPlanForTest(registry *Registry, sessionID string, address string) (dialPlan, error) {
	plan, _, err := registry.reserve(sessionID, "", address)
	if err == nil {
		registry.releaseReservation(plan.AuthorizationID)
	}
	return plan, err
}
