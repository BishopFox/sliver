package certs

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
