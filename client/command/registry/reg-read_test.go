package registry

import (
	"testing"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

func TestFormatRegRead(t *testing.T) {
	tests := []struct {
		name     string
		response *sliverpb.RegistryRead
		want     string
	}{
		{
			name:     "legacy response",
			response: &sliverpb.RegistryRead{Value: "legacy value"},
			want:     "legacy value",
		},
		{
			name:     "string",
			response: &sliverpb.RegistryRead{Type: sliverpb.RegistryType_String, Value: "string value"},
			want:     "string value",
		},
		{
			name:     "binary",
			response: &sliverpb.RegistryRead{Type: sliverpb.RegistryType_Binary, Binary: []byte{0x00, 0x7f, 0x80, 0xff}},
			want:     "[0 127 128 255]",
		},
		{
			name:     "dword",
			response: &sliverpb.RegistryRead{Type: sliverpb.RegistryType_DWORD, Binary: []byte{0xde, 0xc0, 0x17, 0x5a}},
			want:     "0x5a17c0de",
		},
		{
			name:     "qword",
			response: &sliverpb.RegistryRead{Type: sliverpb.RegistryType_QWORD, Binary: []byte{0xef, 0xcd, 0xab, 0x89, 0x67, 0x45, 0x23, 0x01}},
			want:     "0x123456789abcdef",
		},
		{
			name:     "malformed dword",
			response: &sliverpb.RegistryRead{Type: sliverpb.RegistryType_DWORD, Binary: []byte{0x01}},
			want:     "[1]",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := formatRegRead(test.response); got != test.want {
				t.Fatalf("formatRegRead() = %q, want %q", got, test.want)
			}
		})
	}
}
