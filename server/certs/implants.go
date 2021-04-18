package certs

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

const (
	// ImplantCA - Directory containing sliver certificates
	ImplantCA = "sliver"
)

// ImplantGenerateWGKeys - Generate WG keys for implant
func ImplantGenerateWGKeys(wgPeerTunIP string) (string, string, error) {
	isPeer := true
	privKey, pubKey, err := GenerateWGKeys(isPeer, wgPeerTunIP)

	if err != nil {
		wgKeysLog.Errorf("Error generating wg keys for peer: %s", err)
		wgKeysLog.Errorf("priv:  %s", privKey)
		wgKeysLog.Errorf("pub:  %s", pubKey)
		return "", "", err
	}

	return privKey, pubKey, nil
}

// ImplantGenerateECCCertificate - Generate a certificate signed with a given CA
func ImplantGenerateECCCertificate(sliverName string) ([]byte, []byte, error) {
	cert, key := GenerateECCCertificate(ImplantCA, sliverName, false, true)
	err := saveCertificate(ImplantCA, ECCKey, sliverName, cert, key)
	return cert, key, err
}

// ImplantGenerateRSACertificate - Generate a certificate signed with a given CA
func ImplantGenerateRSACertificate(sliverName string) ([]byte, []byte, error) {
	cert, key := GenerateRSACertificate(ImplantCA, sliverName, false, true)
	err := saveCertificate(ImplantCA, RSAKey, sliverName, cert, key)
	return cert, key, err
}
