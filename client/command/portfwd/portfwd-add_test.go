package portfwd

import "testing"

func TestValidatePortfwdRemoteAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		valid   bool
	}{
		{name: "hostname RDP", address: "windows.example:3389", valid: true},
		{name: "IPv4", address: "192.0.2.10:443", valid: true},
		{name: "IPv6", address: "[2001:db8::10]:8443", valid: true},
		{name: "missing port", address: "target.example", valid: false},
		{name: "empty port", address: "target.example:", valid: false},
		{name: "empty host", address: ":3389", valid: false},
		{name: "zero port", address: "target.example:0", valid: false},
		{name: "out of range port", address: "target.example:65536", valid: false},
		{name: "service name", address: "target.example:https", valid: false},
		{name: "unbracketed IPv6", address: "2001:db8::10:8443", valid: false},
		{name: "hostname with space", address: "bad host:80", valid: false},
		{name: "hostname with tab", address: "bad\thost:80", valid: false},
		{name: "hostname with newline", address: "bad\nhost:80", valid: false},
		{name: "IPv6 zone with space", address: "[fe80::1%bad zone]:80", valid: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validatePortfwdRemoteAddress(test.address)
			if test.valid && err != nil {
				t.Fatalf("validatePortfwdRemoteAddress(%q): %v", test.address, err)
			}
			if !test.valid && err == nil {
				t.Fatalf("validatePortfwdRemoteAddress(%q) returned nil", test.address)
			}
		})
	}
}
