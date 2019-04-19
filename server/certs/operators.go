package certs

import (
	"crypto/x509"
	"encoding/pem"
	"sliver/server/db"
)

const (
	// OperatorCA - Directory containing operator certificates
	OperatorCA = "operator"
)

// GenerateOperatorCertificate - Generate a certificate signed with a given CA
func GenerateOperatorCertificate(operator string) ([]byte, []byte) {
	return GenerateCertificate(OperatorCA, operator, false, true)
}

// GetOperatorCertificate - Generate a certificate signed with a given CA
func GetOperatorCertificate(operator string) ([]byte, []byte) {
	return GetCertificate(OperatorCA, operator)
}

// ListOperatorCertificates - Get all client certificates
func ListOperatorCertificates() []*x509.Certificate {
	bucket := db.Bucket(OperatorCA)
	for _, names := range bucket.List("") {

		if err != nil {
			certsLog.Warnf("Failed to read cert file %v", err)
			continue
		}
		block, _ := pem.Decode(certPEM)
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
