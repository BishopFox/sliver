package certs

const (
	// ServerCA - Directory containing server certificates
	ServerCA = "server"
)

// GenerateECCServerCertificate - Generate a server certificate signed with a given CA
func GenerateECCServerCertificate(host string) ([]byte, []byte) {
	return GenerateECCCertificate(ServerCA, host, false, false)
}

// GenerateRSAServerCertificate - Generate a server certificate signed with a given CA
func GenerateRSAServerCertificate(host string) ([]byte, []byte) {
	return GenerateRSACertificate(ServerCA, host, false, false)
}
