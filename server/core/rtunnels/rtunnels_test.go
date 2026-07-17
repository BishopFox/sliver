package rtunnels

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestKeepAliveState(t *testing.T) {
	sessionID := "test-session"
	connStr := "127.0.0.1:8080"
	listenerID := uint32(1)
	keepAlive := int32(42)

	// Test TrackListener
	TrackListener(sessionID, listenerID, connStr, keepAlive)

	assert.True(t, Check(sessionID, connStr))
	assert.Equal(t, keepAlive, GetKeepAlive(sessionID, connStr))

	// Test multiple listeners for same address
	TrackListener(sessionID, 2, connStr, 60)
	assert.Equal(t, int32(60), GetKeepAlive(sessionID, connStr))
	assert.True(t, Check(sessionID, connStr))

	// Test UntrackListener
	UntrackListener(sessionID, 1)
	assert.True(t, Check(sessionID, connStr)) // Still has listener 2
	assert.Equal(t, int32(60), GetKeepAlive(sessionID, connStr))

	UntrackListener(sessionID, 2)
	assert.False(t, Check(sessionID, connStr))
	assert.Equal(t, int32(0), GetKeepAlive(sessionID, connStr))
}

func TestAddDeletePending(t *testing.T) {
	sessionID := "pending-session"
	connStr := "127.0.0.1:9090"

	AddPending(sessionID, connStr)
	assert.True(t, Check(sessionID, connStr))
	assert.Equal(t, int32(0), GetKeepAlive(sessionID, connStr))

	DeletePending(sessionID, connStr)
	assert.False(t, Check(sessionID, connStr))
}
