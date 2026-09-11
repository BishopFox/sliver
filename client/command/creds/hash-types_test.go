package creds

import (
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestHashcatV7BenchmarkHashTypes(t *testing.T) {
	tests := []struct {
		mode        int32
		name        string
		description string
	}{
		{13763, "VERACRYPT_SHA256_XTS_1536_BIT_BOOT_MODE_LEGACY", "VeraCrypt SHA256 + XTS 1536 bit + boot-mode (legacy)"},
		{13771, "VERACRYPT_STREEBOG_512_XTS_512_BIT_LEGACY", "VeraCrypt Streebog-512 + XTS 512 bit (legacy)"},
		{13772, "VERACRYPT_STREEBOG_512_XTS_1024_BIT_LEGACY", "VeraCrypt Streebog-512 + XTS 1024 bit (legacy)"},
		{13773, "VERACRYPT_STREEBOG_512_XTS_1536_BIT_LEGACY", "VeraCrypt Streebog-512 + XTS 1536 bit (legacy)"},
		{13781, "VERACRYPT_STREEBOG_512_XTS_512_BIT_BOOT_MODE_LEGACY", "VeraCrypt Streebog-512 + XTS 512 bit + boot-mode (legacy)"},
		{13782, "VERACRYPT_STREEBOG_512_XTS_1024_BIT_BOOT_MODE_LEGACY", "VeraCrypt Streebog-512 + XTS 1024 bit + boot-mode (legacy)"},
		{13783, "VERACRYPT_STREEBOG_512_XTS_1536_BIT_BOOT_MODE_LEGACY", "VeraCrypt Streebog-512 + XTS 1536 bit + boot-mode (legacy)"},
		{13800, "WINDOWS_PHONE", "Windows Phone 8+ PIN/Password"},
		{13900, "OPENCART", "OpenCart"},
		{14000, "DES", "DES (PT = $salt, key = $pass)"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got, ok := clientpb.HashType_name[test.mode]; !ok || got != test.name {
				t.Errorf("HashType_name[%d] = %q, %v; want %q, true", test.mode, got, ok, test.name)
			}
			if got, ok := clientpb.HashType_value[test.name]; !ok || got != test.mode {
				t.Errorf("HashType_value[%q] = %d, %v; want %d, true", test.name, got, ok, test.mode)
			}
			if got, ok := hashTypes[test.name]; !ok || got != test.description {
				t.Errorf("hashTypes[%q] = %q, %v; want %q, true", test.name, got, ok, test.description)
			}
		})
	}
}
