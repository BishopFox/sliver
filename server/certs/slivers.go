package certs

const (
	// SliverCA - Directory containing sliver certificates
	SliverCA = "sliver"
)

// GenerateECCSliverCertificate - Generate a certificate signed with a given CA
func GenerateECCSliverCertificate(sliverName string) ([]byte, []byte) {
	return GenerateECCCertificate(SliverCA, sliverName, false, true)
}

// GenerateRSASliverCertificate - Generate a certificate signed with a given CA
func GenerateRSASliverCertificate(sliverName string) ([]byte, []byte) {
	return GenerateRSACertificate(SliverCA, sliverName, false, true)
}
