package certs

const (
	// ServerCA - Directory containing server certificates
	ServerCA = "server"
)

// ServerGenerateECCCertificate - Generate a server certificate signed with a given CA
func ServerGenerateECCCertificate(host string) ([]byte, []byte) {
	return GenerateECCCertificate(ServerCA, host, false, false)
}

// ServerGenerateRSACertificate - Generate a server certificate signed with a given CA
func ServerGenerateRSACertificate(host string) ([]byte, []byte, error) {
	cert, key := GenerateRSACertificate(ServerCA, host, false, false)
	err := SaveCertificate(ServerCA, RSAKey, host, cert, key)
	return cert, key, err
}
