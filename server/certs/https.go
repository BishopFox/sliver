package certs

const (
	// HTTPSCA - Directory containing operator certificates
	HTTPSCA = "https"
)

// HTTPSGenerateRSACertificate - Generate a server certificate signed with a given CA
func HTTPSGenerateRSACertificate(host string) ([]byte, []byte, error) {
	cert, key := GenerateRSACertificate(HTTPSCA, host, false, false)
	err := SaveCertificate(HTTPSCA, RSAKey, host, cert, key)
	return cert, key, err
}
