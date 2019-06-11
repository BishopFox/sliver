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
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"

	"github.com/bishopfox/sliver/server/db"
)

const (
	// OperatorCA - Directory containing operator certificates
	OperatorCA = "operator"

	clientNamespace = "client"
	serverNamespace = "server"
)

// OperatorClientGenerateCertificate - Generate a certificate signed with a given CA
func OperatorClientGenerateCertificate(operator string) ([]byte, []byte, error) {
	cert, key := GenerateECCCertificate(OperatorCA, operator, false, true)
	err := SaveCertificate(OperatorCA, ECCKey, fmt.Sprintf("%s.%s", clientNamespace, operator), cert, key)
	return cert, key, err
}

// OperatorClientGetCertificate - Helper function to fetch a client cert
func OperatorClientGetCertificate(operator string) ([]byte, []byte, error) {
	return GetCertificate(OperatorCA, ECCKey, fmt.Sprintf("%s.%s", clientNamespace, operator))
}

// OperatorServerGetCertificate - Helper function to fetch a client cert
func OperatorServerGetCertificate(operator string) ([]byte, []byte, error) {
	return GetCertificate(OperatorCA, ECCKey, fmt.Sprintf("%s.%s", serverNamespace, operator))
}

// OperatorServerGenerateCertificate - Generate a certificate signed with a given CA
func OperatorServerGenerateCertificate(hostname string) ([]byte, []byte, error) {
	cert, key := GenerateECCCertificate(OperatorCA, hostname, false, false)
	err := SaveCertificate(OperatorCA, ECCKey, fmt.Sprintf("%s.%s", serverNamespace, hostname), cert, key)
	return cert, key, err
}

// OperatorClientListCertificates - Get all client certificates
func OperatorClientListCertificates() []*x509.Certificate {
	bucket, err := db.GetBucket(OperatorCA)
	if err != nil {
		return []*x509.Certificate{}
	}

	// The key structure is: <key type>_<namespace>.<operator name>
	operators, err := bucket.List(fmt.Sprintf("%s_%s", ECCKey, clientNamespace))
	if err != nil {
		return []*x509.Certificate{}
	}
	certsLog.Infof("Found %d operator certs ...", len(operators))

	certs := []*x509.Certificate{}
	for _, operator := range operators {

		certsLog.Infof("Operator = %v", operator)
		keypairRaw, err := bucket.Get(operator)
		if err != nil {
			certsLog.Warnf("Failed to fetch operator keypair %v", err)
			continue
		}
		keypair := &CertificateKeyPair{}
		json.Unmarshal(keypairRaw, keypair)

		block, _ := pem.Decode(keypair.Certificate)
		if block == nil {
			certsLog.Warn("failed to parse certificate PEM")
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			certsLog.Warnf("failed to parse x.509 certificate %v", err)
			continue
		}
		certs = append(certs, cert)
	}
	return certs
}
