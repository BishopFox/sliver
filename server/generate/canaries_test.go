//go:build sliver_generate

package generate

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
)

// Canary domains must remain visible to a binary scanner after the production
// compiler and obfuscator run. Checking rendered source alone misses both
// literal obfuscation and linker removal of unused type metadata.
func TestCanariesSurviveObfuscation(t *testing.T) {
	const parentDomain = "canary-regression.invalid."
	certs.SetupCAs()
	tests := []struct {
		name         string
		goos         string
		protocol     string
		beacon       bool
		wantCanaries int
	}{
		{name: "windows_http_session", goos: "windows", protocol: "http", wantCanaries: 1},
		{name: "linux_http_beacon", goos: "linux", protocol: "http", beacon: true, wantCanaries: 1},
		{name: "windows_mtls_session", goos: "windows", protocol: "mtls", wantCanaries: 2},
		{name: "linux_wireguard_session", goos: "linux", protocol: "wg", wantCanaries: 2},
		{name: "linux_http_without_canaries", goos: "linux", protocol: "http"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := "canary_" + strings.ReplaceAll(models.NewUUID().String(), "-", "")
			config := &clientpb.ImplantConfig{
				GOOS:             tt.goos,
				GOARCH:           "amd64",
				Debug:            false,
				ObfuscateSymbols: true,
				NetGoEnabled:     true,
				IsBeacon:         tt.beacon,
				C2: []*clientpb.ImplantC2{
					{URL: tt.protocol + "://c2.invalid"},
				},
			}
			if tt.wantCanaries > 0 {
				config.CanaryDomains = []string{parentDomain}
			}
			switch tt.protocol {
			case "http":
				config.IncludeHTTP = true
			case "mtls":
				config.IncludeMTLS = true
			case "wg":
				config.IncludeWG = true
				config.WGKeyExchangePort = 1234
				config.WGTcpCommsPort = 5678
				certs.SetupWGKeys()
			}

			// Register cleanup before generation so failed builds do not leave
			// source trees or canary records behind.
			projectDir := filepath.Join(GetSliversDir(), config.GOOS, config.GOARCH, name)
			t.Cleanup(func() {
				if err := os.RemoveAll(projectDir); err != nil {
					t.Errorf("remove generated project: %v", err)
				}
				if err := db.Session().Where("implant_name = ?", name).Delete(&models.DNSCanary{}).Error; err != nil {
					t.Errorf("remove generated canaries: %v", err)
				}
			})

			build, err := GenerateConfig(name, config)
			if err != nil {
				t.Fatalf("generate implant config: %v", err)
			}
			httpConfig := configs.GenerateDefaultHTTPC2Config()
			binPath, err := SliverExecutable(name, build, config, httpConfig.ImplantConfig)
			if err != nil {
				t.Fatalf("build obfuscated implant: %v", err)
			}
			binary, err := os.ReadFile(binPath)
			if err != nil {
				t.Fatalf("read obfuscated implant: %v", err)
			}
			var canaries []models.DNSCanary
			if err := db.Session().Where("implant_name = ?", name).Find(&canaries).Error; err != nil {
				t.Fatalf("read generated canaries: %v", err)
			}
			assertCanaryBinary(t, binary, canaries, parentDomain, tt.wantCanaries)
			t.Logf("checked %d registered canaries in %s/%s obfuscated binary (%d bytes)", len(canaries), config.GOOS, config.GOARCH, len(binary))
		})
	}
}

func assertCanaryBinary(t *testing.T, binary []byte, canaries []models.DNSCanary, parentDomain string, wantCanaries int) {
	t.Helper()

	if bytes.Contains(binary, []byte("[[GenerateCanary]]")) {
		t.Error("unexpanded canary placeholder remains in binary")
	}
	if len(canaries) != wantCanaries {
		t.Fatalf("registered %d canaries, want %d", len(canaries), wantCanaries)
	}
	if wantCanaries == 0 {
		if bytes.Contains(binary, []byte(parentDomain)) {
			t.Error("build without canaries contains the canary parent domain")
		}
		return
	}
	for _, canary := range canaries {
		if canary.Domain == "" || !strings.HasSuffix(canary.Domain, "."+parentDomain) {
			t.Errorf("unexpected generated canary domain %q", canary.Domain)
			continue
		}
		if !bytes.Contains(binary, []byte(canary.Domain)) {
			t.Errorf("registered canary %q is absent from the obfuscated binary", canary.Domain)
		}
	}
}
