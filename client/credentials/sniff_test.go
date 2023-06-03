package credentials

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestSniffHashType(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected clientpb.HashType
	}{
		{
			name:     "DCC2 hash",
			input:    "$DCC2$10240#tom#e4e938d12fe5974dc4...",
			expected: clientpb.HashType_DCC2,
		},
		{
			name:     "Kerberos 23 SA-REQ-INIT hash",
			input:    "$krb5pa$23$*user$realm$hash",
			expected: clientpb.HashType_KERBEROS_23_SA_REQ_PREAUTH,
		},
		{
			name:     "Kerberos 23 TGS-REP hash",
			input:    "$krb5tgs$23$",
			expected: clientpb.HashType_KERBEROS_23_TGS_REP,
		},
		{
			name:     "Kerberos 23 AS-REP hash",
			input:    "$krb5asrep$23$",
			expected: clientpb.HashType_KERBEROS_23_AS_REP,
		},
		{
			name:     "Kerberos 17 TGS-REP hash",
			input:    "$krb5tgs$17$",
			expected: clientpb.HashType_KERBEROS_17_TGS_REP,
		},
		{
			name:     "Kerberos 17 PREAUTH hash",
			input:    "$krb5pa$17$",
			expected: clientpb.HashType_KERBEROS_17_PREAUTH,
		},
		{
			name:     "Kerberos 18 TGS-REP hash",
			input:    "$krb5tgs$18$",
			expected: clientpb.HashType_KERBEROS_18_TGS_REP,
		},
		{
			name:     "Kerberos 18 PREAUTH hash",
			input:    "$krb5pa$18$",
			expected: clientpb.HashType_KERBEROS_18_PREAUTH,
		},
		{
			name:     "BCrypt hash",
			input:    "$2a$",
			expected: clientpb.HashType_BCRYPT_UNIX,
		},
		{
			name:     "SHA512-Crypt hash",
			input:    "$6$",
			expected: clientpb.HashType_SHA512_CRYPT_UNIX,
		},
		{
			name:     "SCRYPT hash",
			input:    "SCRYPT:",
			expected: clientpb.HashType_SCRYPT,
		},
		{
			name:     "Invalid hash",
			input:    "invalid_hash",
			expected: clientpb.HashType_INVALID,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := SniffHashType(tc.input)
			if result != tc.expected {
				t.Errorf("expected %v, but got %v", tc.expected, result)
			}
		})
	}
}
