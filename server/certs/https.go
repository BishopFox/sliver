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

import (
	"crypto/rand"
	"crypto/rsa"
)

const (
	// HTTPSCA - Directory containing operator certificates
	HTTPSCA = "https"
)

// HTTPSGenerateRSACertificate - Generate a server certificate signed with a given CA
func HTTPSGenerateRSACertificate(host string) ([]byte, []byte, error) {

	certsLog.Debugf("Generating TLS certificate (RSA) for '%s' ...", host)

	var privateKey interface{}
	var err error

	// Generate private key
	privateKey, err = rsa.GenerateKey(rand.Reader, rsaKeySize())
	if err != nil {
		certsLog.Fatalf("Failed to generate private key %s", err)
		return nil, nil, err
	}
	subject := randomSubject(host)
	cert, key := generateCertificate(HTTPSCA, (*subject), false, false, privateKey)
	err = saveCertificate(HTTPSCA, RSAKey, host, cert, key)
	return cert, key, err
}
