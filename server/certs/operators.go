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
	return GenerateECCCertificate(OperatorCA, operator, false, true)
}

// GetOperatorCertificate - Generate a certificate signed with a given CA
func GetOperatorCertificate(operator string) ([]byte, []byte, error) {
	return GetECCCertificate(OperatorCA, operator)
}

// ListOperatorCertificates - Get all client certificates
func ListOperatorCertificates() []*x509.Certificate {
	bucket, err := db.GetBucket(OperatorCA)
	if err != nil {
		return []*x509.Certificate{}
	}

	certs := []*x509.Certificate{}
	ls, err := bucket.List("")
	for _, operator := range ls {
		certPEM, err := bucket.Get(operator)
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
