package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/client/assets"
)

func TestSelectConfigFromNonTTYRequiresExplicitConfigWhenMultiple(t *testing.T) {
	configs := map[string]*assets.ClientConfig{
		"first":  {Operator: "first"},
		"second": {Operator: "second"},
	}
	key, config, err := selectConfigFrom(configs, false)
	if err == nil || !strings.Contains(err.Error(), "--config") {
		t.Fatalf("error = %v, want --config guidance", err)
	}
	if key != "" || config != nil {
		t.Fatalf("selected %q (%v) without an explicit config", key, config)
	}
}

func TestSelectConfigFromNonTTYUsesSoleConfig(t *testing.T) {
	want := &assets.ClientConfig{Operator: "only"}
	key, config, err := selectConfigFrom(map[string]*assets.ClientConfig{"only": want}, false)
	if err != nil || key != "only" || config != want {
		t.Fatalf("selection = %q (%v), error = %v", key, config, err)
	}
}

func TestSelectConfigWithExplicitPathOnNonTTY(t *testing.T) {
	want := &assets.ClientConfig{Operator: "selected", LHost: "127.0.0.1"}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "selected.cfg")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	key, config, err := selectConfig(path, false)
	if err != nil {
		t.Fatalf("select explicit config: %v", err)
	}
	if key != "selected.cfg" || config.Operator != want.Operator || config.LHost != want.LHost {
		t.Fatalf("selection = %q (%v)", key, config)
	}
}
