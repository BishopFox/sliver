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
	// MtlsImplantCA - Directory containing HTTPS server certificates
	MtlsImplantCA = "mtls-implant"
	MtlsServerCA  = "mtls-server"
)

// MtlsC2ServerGenerateECCCertificate - Generate a server certificate signed with a given CA
func MtlsC2ServerGenerateECCCertificate(host string) ([]byte, []byte, error) {
	cert, key := GenerateECCCertificate(MtlsServerCA, host, false, false)
	err := saveCertificate(MtlsServerCA, ECCKey, host, cert, key)
	return cert, key, err
}

// MtlsC2ImplantGenerateECCCertificate - Generate a server certificate signed with a given CA
func MtlsC2ImplantGenerateECCCertificate(name string) ([]byte, []byte, error) {
	cert, key := GenerateECCCertificate(MtlsImplantCA, name, false, true)
	err := saveCertificate(MtlsImplantCA, ECCKey, name, cert, key)
	return cert, key, err
}
