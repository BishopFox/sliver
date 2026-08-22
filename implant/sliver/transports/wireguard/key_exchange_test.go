//go:build windows || darwin || linux

package wireguard

import (
	"net"
	"strings"
	"testing"
)

func exchangeResponse(t *testing.T, response string) (string, string, string, error) {
	t.Helper()
	client, server := net.Pipe()
	go func() {
		_, _ = server.Write([]byte(response))
		_ = server.Close()
	}()
	defer client.Close()
	return doKeyExchange(client)
}

func TestDoKeyExchangeParsesTextTunnelIP(t *testing.T) {
	privateKey := strings.Repeat("a", 64)
	publicKey := strings.Repeat("b", 64)
	gotPrivate, gotPublic, gotIP, err := exchangeResponse(t, privateKey+"|"+publicKey+"|100.64.0.23")
	if err != nil {
		t.Fatal(err)
	}
	if gotPrivate != privateKey || gotPublic != publicKey || gotIP != "100.64.0.23" {
		t.Fatalf("unexpected exchange result: private=%q public=%q ip=%q", gotPrivate, gotPublic, gotIP)
	}
}

func TestDoKeyExchangeRejectsMalformedResponses(t *testing.T) {
	key := strings.Repeat("a", 64)
	tests := map[string]string{
		"missing field":     key + "|" + key,
		"short key":         "aa|" + key + "|100.64.0.2",
		"non-hex key":       strings.Repeat("z", 64) + "|" + key + "|100.64.0.2",
		"invalid IP":        key + "|" + key + "|not-an-ip",
		"IPv6 tunnel IP":    key + "|" + key + "|fd00::2",
		"oversized payload": strings.Repeat("x", 257),
	}
	for name, response := range tests {
		t.Run(name, func(t *testing.T) {
			_, _, _, err := exchangeResponse(t, response)
			if err == nil {
				t.Fatal("expected malformed key exchange response to fail")
			}
		})
	}
}
