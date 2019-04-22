package certs

import (
	"testing"
)

func TestGenerateSliverCertificate(t *testing.T) {
	GenerateCertificateAuthority(SliverCA)
	GenerateECCSliverCertificate("test1")
	GenerateRSASliverCertificate("test2")
}

func TestGenerateClientCertificate(t *testing.T) {
	GenerateCertificateAuthority(OperatorCA)
	GenerateOperatorCertificate("test2")
}

func TestGenerateServerCertificate(t *testing.T) {
	GenerateCertificateAuthority(ServerCA)
	GenerateECCServerCertificate("test3.com")
	GenerateRSAServerCertificate("test3.com")
}
