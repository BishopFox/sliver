package transport

import (
	"testing"
	"time"

	"github.com/bishopfox/sliver/client/assets"
)

func TestWireGuardTunnelCacheKeyIncludesConfigMaterial(t *testing.T) {
	config := &assets.ClientConfig{
		LHost: "127.0.0.1",
		LPort: 31337,
		WG: &assets.ClientWGConfig{
			ServerPubKey:     "server-pub",
			ClientPrivateKey: "client-priv",
			ClientIP:         "100.65.0.2",
			ServerIP:         "100.65.0.1",
		},
	}

	key1, err := wireGuardTunnelCacheKey(config)
	if err != nil {
		t.Fatalf("cache key: %v", err)
	}

	config.WG.ClientPrivateKey = "other-client-priv"
	key2, err := wireGuardTunnelCacheKey(config)
	if err != nil {
		t.Fatalf("cache key after config change: %v", err)
	}

	if key1 == key2 {
		t.Fatal("expected cache key to change when wireguard config changes")
	}
}

func TestCacheIdleWireGuardTunnelRemovesIdleTunnel(t *testing.T) {
	resetWireGuardTunnelCacheForTest(t)

	previousIdleTimeout := multiplayerWireGuardIdleTimeout
	multiplayerWireGuardIdleTimeout = 10 * time.Millisecond
	t.Cleanup(func() {
		multiplayerWireGuardIdleTimeout = previousIdleTimeout
	})

	const cacheKey = "test-cache-key"
	cacheIdleWireGuardTunnel(cacheKey, &wireGuardTunnel{}, "100.65.0.1:31337")

	deadline := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(deadline) {
		wireGuardTunnelCacheMu.Lock()
		_, exists := wireGuardTunnelCache[cacheKey]
		wireGuardTunnelCacheMu.Unlock()
		if !exists {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}

	t.Fatal("expected idle tunnel cache entry to be removed")
}

func resetWireGuardTunnelCacheForTest(t *testing.T) {
	t.Helper()

	wireGuardTunnelCacheMu.Lock()
	for key, shared := range wireGuardTunnelCache {
		if shared != nil && shared.timer != nil {
			shared.timer.Stop()
		}
		delete(wireGuardTunnelCache, key)
	}
	wireGuardTunnelCacheMu.Unlock()

	t.Cleanup(func() {
		wireGuardTunnelCacheMu.Lock()
		for key, shared := range wireGuardTunnelCache {
			if shared != nil && shared.timer != nil {
				shared.timer.Stop()
			}
			delete(wireGuardTunnelCache, key)
		}
		wireGuardTunnelCacheMu.Unlock()
	})
}
