package certs

const (
	// SliverCA - Directory containing sliver certificates
	SliverCA = "sliver"
)

// SliverGenerateECCCertificate - Generate a certificate signed with a given CA
func SliverGenerateECCCertificate(sliverName string) ([]byte, []byte, error) {
	cert, key := GenerateECCCertificate(SliverCA, sliverName, false, true)
	err := SaveCertificate(SliverCA, ECCKey, sliverName, cert, key)
	return cert, key, err
}

// SliverGenerateRSACertificate - Generate a certificate signed with a given CA
func SliverGenerateRSACertificate(sliverName string) ([]byte, []byte, error) {
	cert, key := GenerateRSACertificate(SliverCA, sliverName, false, true)
	err := SaveCertificate(SliverCA, RSAKey, sliverName, cert, key)
	return cert, key, err
}
