package assets

import (
	"encoding/json"
	"testing"
)

func TestClientWGConfigEnabledIsPositiveOptIn(t *testing.T) {
	legacyJSON := []byte(`{"wg":{"server_pub_key":"server-pub","client_private_key":"client-priv","client_ip":"100.64.0.2"}}`)
	legacy := &ClientConfig{}
	if err := json.Unmarshal(legacyJSON, legacy); err != nil {
		t.Fatalf("unmarshal legacy config: %v", err)
	}
	if legacy.WG == nil {
		t.Fatal("expected legacy WireGuard config")
	}
	if legacy.WG.Enabled {
		t.Fatal("expected an absent enabled marker to remain disabled")
	}

	enabledJSON := []byte(`{"wg":{"enabled":true,"server_pub_key":"server-pub","client_private_key":"client-priv","client_ip":"100.64.0.2"}}`)
	enabled := &ClientConfig{}
	if err := json.Unmarshal(enabledJSON, enabled); err != nil {
		t.Fatalf("unmarshal enabled config: %v", err)
	}
	if enabled.WG == nil || !enabled.WG.Enabled {
		t.Fatal("expected enabled marker to opt in to WireGuard")
	}
}
