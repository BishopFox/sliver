package certs

import (
	"path"
	"sliver/server/db"
)

const (
	// ServerCA - Directory containing server certificates
	ServerCA = "server"
)

// GenerateECCServerCertificate - Generate a server certificate signed with a given CA
func GenerateECCServerCertificate(host string) ([]byte, []byte) {
	bucket = db.GetBucket(ServersCert)
	cert, key := GenerateECCCertificate(bucket, host, caType, false, false)

	return cert, key
}

// GenerateRSAServerCertificate - Generate a server certificate signed with a given CA
func GenerateRSAServerCertificate(host string) ([]byte, []byte) {
	bucket = db.GetBucket(ServersCert)
	cert, key := GenerateRSACertificate(bucket, caType, host, false, false)
	return cert, key
}

// GetServerECCCertificate - Get a server certificate/key pair signed by ca type
func GetServerECCCertificate(rootDir string, caType string, host string, generateIfNoneExists bool) ([]byte, []byte, error) {

	certsLog.Infof("Getting certificate (ca type = %s) '%s'", caType, host)

	// If not certificate exists for this host we just generate one on the fly
	_, _, err := GetCertificatePEM(rootDir, path.Join(caType, "ecc"), host)
	if err != nil {
		if generateIfNoneExists {
			certsLog.Infof("No server certificate, generating ca type = %s '%s'", caType, host)
			GenerateServerCertificate(rootDir, caType, host, true)
		} else {
			return nil, nil, err
		}
	}

	certPEM, keyPEM, err := GetCertificatePEM(rootDir, path.Join(caType, "ecc"), host)
	if err != nil {
		certsLog.Infof("Failed to load PEM data %v", err)
		return nil, nil, err
	}

	return certPEM, keyPEM, nil
}

// GetServerRSACertificate - Get a server certificate/key pair signed by ca type
func GetServerRSACertificate(rootDir string, caType string, host string, generateIfNoneExists bool) ([]byte, []byte, error) {

	certsLog.Infof("Getting rsa certificate (ca type = %s) '%s'", caType, host)

	// If not certificate exists for this host we just generate one on the fly
	_, _, err := GetCertificatePEM(rootDir, path.Join(caType, "rsa"), host)
	if err != nil {
		if generateIfNoneExists {
			certsLog.Infof("No server certificate, generating ca type = %s '%s'", caType, host)
			GenerateServerRSACertificate(rootDir, caType, host, true)
		} else {
			return nil, nil, err
		}
	}

	certPEM, keyPEM, err := GetCertificatePEM(rootDir, path.Join(caType, "rsa"), host)
	if err != nil {
		certsLog.Infof("Failed to load PEM data %v", err)
		return nil, nil, err
	}

	return certPEM, keyPEM, nil
}
