//go:build windows || darwin || linux

package wireguard

import "testing"

func TestFormatWGEndpoint_IPv6(t *testing.T) {
	got := formatWGEndpoint("::1", 8989)
	if got != "[::1]:8989" {
		t.Fatalf("unexpected endpoint: got=%q want=%q", got, "[::1]:8989")
	}
}

func TestFormatWGEndpoint_IPv4(t *testing.T) {
	got := formatWGEndpoint("127.0.0.1", 53)
	if got != "127.0.0.1:53" {
		t.Fatalf("unexpected endpoint: got=%q want=%q", got, "127.0.0.1:53")
	}
}

func TestFormatWGEndpoint_Hostname(t *testing.T) {
	got := formatWGEndpoint("example.com", 1337)
	if got != "example.com:1337" {
		t.Fatalf("unexpected endpoint: got=%q want=%q", got, "example.com:1337")
	}
}
