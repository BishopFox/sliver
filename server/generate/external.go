package generate

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
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/cryptography"
	"github.com/bishopfox/sliver/server/db/models"
)

// SliverExternal - Generates the cryptographic keys for the implant but compiles no code
func SliverExternal(name string, config *models.ImplantConfig) (*clientpb.ExternalImplantConfig, error) {
	if config.Format != clientpb.OutputFormat_EXTERNAL {
		return nil, fmt.Errorf("invalid format: %s", config.Format)
	}
	config.MTLSc2Enabled = isC2Enabled([]string{"mtls"}, config.C2)
	config.WGc2Enabled = isC2Enabled([]string{"wg"}, config.C2)
	config.HTTPc2Enabled = isC2Enabled([]string{"http", "https"}, config.C2)
	config.DNSc2Enabled = isC2Enabled([]string{"dns"}, config.C2)
	config.NamePipec2Enabled = isC2Enabled([]string{"namedpipe"}, config.C2)
	config.TCPPivotc2Enabled = isC2Enabled([]string{"tcppivot"}, config.C2)

	// Cert PEM encoded certificates
	serverCACert, _, _ := certs.GetCertificateAuthorityPEM(certs.MtlsServerCA)
	sliverCert, sliverKey, err := certs.MtlsC2ImplantGenerateECCCertificate(name)
	if err != nil {
		return nil, err
	}

	// ECC keys
	implantKeyPair, err := cryptography.RandomECCKeyPair()
	if err != nil {
		return nil, err
	}
	serverKeyPair := cryptography.ECCServerKeyPair()
	digest := sha256.Sum256((*implantKeyPair.Public)[:])
	config.ECCPublicKey = implantKeyPair.PublicBase64()
	config.ECCPublicKeyDigest = hex.EncodeToString(digest[:])
	config.ECCPrivateKey = implantKeyPair.PrivateBase64()
	config.ECCPublicKeySignature = cryptography.MinisignServerSign(implantKeyPair.Public[:])
	config.ECCServerPublicKey = serverKeyPair.PublicBase64()
	config.MinisignServerPublicKey = cryptography.MinisignServerPublicKey()

	// MTLS keys
	if config.MTLSc2Enabled {
		config.MtlsCACert = string(serverCACert)
		config.MtlsCert = string(sliverCert)
		config.MtlsKey = string(sliverKey)
	}

	otpSecret, err := cryptography.TOTPServerSecret()
	if err != nil {
		return nil, err
	}

	// Generate wg Keys as needed
	if config.WGc2Enabled {
		implantPrivKey, _, err := certs.ImplantGenerateWGKeys(config.WGPeerTunIP)
		if err != nil {
			return nil, err
		}
		_, serverPubKey, err := certs.GetWGServerKeys()
		if err != nil {
			return nil, fmt.Errorf("failed to embed implant wg keys: %s", err)
		}
		config.WGImplantPrivKey = implantPrivKey
		config.WGServerPubKey = serverPubKey
	}
	err = ImplantConfigSave(config)
	if err != nil {
		return nil, err
	}
	return &clientpb.ExternalImplantConfig{
		Name:      name,
		Config:    config.ToProtobuf(),
		OTPSecret: otpSecret,
	}, nil
}
