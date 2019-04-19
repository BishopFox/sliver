package certs

const (
	// SliverCA - Directory containing sliver certificates
	SliverCA = "sliver"
)

// GenerateECCSliverCertificate - Generate a certificate signed with a given CA
func GenerateECCSliverCertificate(sliverName string) ([]byte, []byte) {
	cert, key := GenerateCertificate(rootDir, sliverName, SliversCert, false, true)
	return cert, key
}

// GenerateRSASliverCertificate - Generate a certificate signed with a given CA
func GenerateRSASliverCertificate(sliverName string) ([]byte, []byte) {
	cert, key := GenerateCertificate(rootDir, sliverName, SliversCert, false, true)
	return cert, key
}
