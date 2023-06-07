package credentials

/*
	Sliver Implant Framework
	Copyright (C) 2022  Bishop Fox

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
	"strings"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

// See EXAMPLES.md for example hashes, right now we just pick off low
// hanging fruit. Later on we can add length checks, regex, etc.
func SniffHashType(unknownHash string) clientpb.HashType {
	// $DCC2$10240#tom#e4e938d12fe5974dc4...
	if strings.HasPrefix(unknownHash, "$DCC2$") {
		return clientpb.HashType_DCC2
	}
	// $krb5pa$23$*user$realm$hash
	if strings.HasPrefix(unknownHash, "$krb5pa$23$") {
		return clientpb.HashType_KERBEROS_23_SA_REQ_PREAUTH
	}
	// $krb5tgs$23$
	if strings.HasPrefix(unknownHash, "$krb5tgs$23$") {
		return clientpb.HashType_KERBEROS_23_TGS_REP
	}
	// $krb5asrep$23$
	if strings.HasPrefix(unknownHash, "$krb5asrep$23$") {
		return clientpb.HashType_KERBEROS_23_AS_REP
	}
	// $krb5tgs$17$
	if strings.HasPrefix(unknownHash, "$krb5tgs$17$") {
		return clientpb.HashType_KERBEROS_17_TGS_REP
	}
	// $krb5pa$17$
	if strings.HasPrefix(unknownHash, "$krb5pa$17$") {
		return clientpb.HashType_KERBEROS_17_PREAUTH
	}
	// $krb5tgs$18$
	if strings.HasPrefix(unknownHash, "$krb5tgs$18$") {
		return clientpb.HashType_KERBEROS_18_TGS_REP
	}
	// $krb5pa$18$
	if strings.HasPrefix(unknownHash, "$krb5pa$18$") {
		return clientpb.HashType_KERBEROS_18_PREAUTH
	}
	// $2a$
	if strings.HasPrefix(unknownHash, "$2a$") {
		return clientpb.HashType_BCRYPT_UNIX
	}
	// $6$
	if strings.HasPrefix(unknownHash, "$6$") {
		return clientpb.HashType_SHA512_CRYPT_UNIX
	}
	// SCRYPT:...
	if strings.HasPrefix(unknownHash, "SCRYPT:") {
		return clientpb.HashType_SCRYPT
	}
	return clientpb.HashType_INVALID
}
