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
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
)

const (
	// OperatorCA - Directory containing operator certificates
	OperatorCA = "operator"

	clientNamespace = "client" // Operator clients
	serverNamespace = "server" // Operator servers
)

var (
	// ErrOperatorClientCertificateNotFound indicates that the presented operator
	// client certificate is no longer trusted because it is not present in the
	// certificate store.
	ErrOperatorClientCertificateNotFound = errors.New("operator client certificate not found in database")

	// ErrInvalidOperatorClientCertificate indicates that the presented
	// certificate is not shaped like an operator client leaf certificate.
	ErrInvalidOperatorClientCertificate = errors.New("invalid operator client certificate")
)

// OperatorClientGenerateCertificate - Generate a certificate signed with a given CA
func OperatorClientGenerateCertificate(operator string) ([]byte, []byte, error) {
	cert, key := GenerateECCCertificate(OperatorCA, operator, false, true, true)
	err := saveCertificate(OperatorCA, ECCKey, fmt.Sprintf("%s.%s", clientNamespace, operator), cert, key)
	return cert, key, err
}

// OperatorClientGetCertificate - Helper function to fetch a client cert
func OperatorClientGetCertificate(operator string) ([]byte, []byte, error) {
	return GetECCCertificate(OperatorCA, fmt.Sprintf("%s.%s", clientNamespace, operator))
}

// OperatorClientRemoveCertificate - Helper function to remove a client cert
func OperatorClientRemoveCertificate(operator string) error {
	return RemoveCertificate(OperatorCA, ECCKey, fmt.Sprintf("%s.%s", clientNamespace, operator))
}

// ValidateOperatorClientCertificate ensures that the presented operator client
// certificate is still present in the database. A valid chain alone is not
// enough; the exact leaf certificate must still exist in storage.
func ValidateOperatorClientCertificate(peerCertificates []*x509.Certificate) error {
	if len(peerCertificates) == 0 || peerCertificates[0] == nil {
		return ErrInvalidOperatorClientCertificate
	}

	leaf := peerCertificates[0]
	if leaf.IsCA || leaf.Subject.CommonName == "" || !hasExtKeyUsage(leaf, x509.ExtKeyUsageClientAuth) {
		return ErrInvalidOperatorClientCertificate
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: leaf.Raw,
	})
	if len(pemBytes) == 0 {
		return ErrInvalidOperatorClientCertificate
	}

	record := &models.Certificate{}
	result := db.Session().Select("id").Where(&models.Certificate{
		CommonName:     fmt.Sprintf("%s.%s", clientNamespace, leaf.Subject.CommonName),
		CAType:         OperatorCA,
		KeyType:        ECCKey,
		CertificatePEM: string(pemBytes),
	}).First(record)
	if result.Error == nil {
		return nil
	}
	if errors.Is(result.Error, db.ErrRecordNotFound) {
		return ErrOperatorClientCertificateNotFound
	}
	return result.Error
}

func hasExtKeyUsage(cert *x509.Certificate, usage x509.ExtKeyUsage) bool {
	for _, extKeyUsage := range cert.ExtKeyUsage {
		if extKeyUsage == usage {
			return true
		}
	}
	return false
}

// OperatorServerGetCertificate - Helper function to fetch a server cert
func OperatorServerGetCertificate(hostname string) ([]byte, []byte, error) {
	return GetECCCertificate(OperatorCA, fmt.Sprintf("%s.%s", serverNamespace, hostname))
}

// OperatorServerGenerateCertificate - Generate a certificate signed with a given CA
func OperatorServerGenerateCertificate(hostname string) ([]byte, []byte, error) {
	cert, key := GenerateECCCertificate(OperatorCA, hostname, false, false, true)
	err := saveCertificate(OperatorCA, ECCKey, fmt.Sprintf("%s.%s", serverNamespace, hostname), cert, key)
	return cert, key, err
}

// OperatorClientListCertificates - Get all client certificates
func OperatorClientListCertificates() []*x509.Certificate {
	operatorCerts := []*models.Certificate{}
	dbSession := db.Session()
	result := dbSession.Where(&models.Certificate{CAType: OperatorCA}).Find(&operatorCerts)
	if result.Error != nil {
		certsLog.Error(result.Error)
		return []*x509.Certificate{}
	}

	certsLog.Infof("Found %d operator certs ...", len(operatorCerts))

	certs := []*x509.Certificate{}
	for _, operator := range operatorCerts {
		block, _ := pem.Decode([]byte(operator.CertificatePEM))
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
