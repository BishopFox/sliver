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
	// C2ServerCA - Directory containing HTTPS server certificates
	C2ServerCA = "c2-server"
)

// C2ServerGenerateECCCertificate - Generate a server certificate signed with a given CA
func C2ServerGenerateECCCertificate(host string) ([]byte, []byte, error) {
	cert, key := GenerateECCCertificate(C2ServerCA, host, false, false)
	err := saveCertificate(C2ServerCA, ECCKey, host, cert, key)
	return cert, key, err
}

// C2ServerGenerateRSACertificate - Generate a server certificate signed with a given CA
func C2ServerGenerateRSACertificate(host string) ([]byte, []byte, error) {
	cert, key := GenerateRSACertificate(C2ServerCA, host, false, false)
	err := saveCertificate(C2ServerCA, RSAKey, host, cert, key)
	return cert, key, err
}

// C2ServerGetRSACertificate - Get a server certificate based on hostname
func C2ServerGetRSACertificate(host string) ([]byte, []byte, error) {
	return GetRSACertificate(C2ServerCA, host)
}
