package comm

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

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
	"crypto/sha256"
	"encoding/base64"

	"golang.org/x/crypto/ssh"

	clientAssets "github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/server/certs"
)

// LoadServerLocalCommConfig - The server needs to load some elements that normally belonging
// to the client-server connection config, because some items are used to run a Comm client.
func LoadServerLocalCommConfig() {

	// Declare an Config for the server-as-client, because its Comm system needs a
	// fingerprint value as well for authenticating to itself.
	clientAssets.Config = new(clientAssets.ClientConfig)

	// Make a fingerprint of the implant's private key, for SSH-layer authentication
	_, serverCAKey, _ := certs.GetCertificateAuthorityPEM(certs.OperatorCA)
	signer, _ := ssh.ParsePrivateKey(serverCAKey)
	keyBytes := sha256.Sum256(signer.PublicKey().Marshal())
	fingerprint := base64.StdEncoding.EncodeToString(keyBytes[:])

	// Load only needed fields in the client assets (config) package.
	clientAssets.Config.PrivateKey = string(serverCAKey)
	clientAssets.Config.ServerFingerprint = fingerprint
}
