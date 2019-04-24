package certs

import (
	"bytes"
	"testing"
)

func TestGenerateECCSliverCertificate(t *testing.T) {
	GenerateCertificateAuthority(SliverCA)
	eccCert1, eccKey1, err := GenerateECCSliverCertificate("test1")
	if err != nil {
		t.Errorf("Failed to generate ecc certificate %v", err)
		return
	}
	eccCert2, eccKey2, err := GetCertificate(SliverCA, ECCKey, "test1")
	if err != nil {
		t.Errorf("Failed to get certificate %v", err)
		return
	}
	if !bytes.Equal(eccCert1, eccCert2) || !bytes.Equal(eccKey1, eccKey2) {
		t.Errorf("Stored ecc cert/key does match generated cert/key")
		return
	}

}

func TestGenerateRSASliverCertificate(t *testing.T) {
	rsaCert1, rsaKey1, err := GenerateRSASliverCertificate("test2")
	if err != nil {
		t.Errorf("Failed to generate rsa certificate %v", err)
		return
	}
	rsaCert2, rsaKey2, err := GetCertificate(SliverCA, RSAKey, "test2")
	if err != nil {
		t.Errorf("Failed to get certificate %v", err)
		return
	}
	if !bytes.Equal(rsaCert1, rsaCert2) || !bytes.Equal(rsaKey1, rsaKey2) {
		t.Errorf("Stored rsa cert/key does match generated cert/key")
		return
	}
}

func TestGenerateOperatorCertificate(t *testing.T) {
	GenerateCertificateAuthority(OperatorCA)
	cert1, key1, err := GenerateOperatorCertificate("test3")
	if err != nil {
		t.Errorf("Failed to store ecc certificate %v", err)
	}

	cert2, key2, err := GetCertificate(OperatorCA, ECCKey, "test3")
	if err != nil {
		t.Errorf("Failed to get ecc certificate %v", err)
	}

	if !bytes.Equal(cert1, cert2) || !bytes.Equal(key1, key2) {
		t.Errorf("Stored ecc cert/key does match generated cert/key")
		return
	}
}

func TestGenerateServerCertificate(t *testing.T) {
	GenerateCertificateAuthority(ServerCA)

	GenerateECCServerCertificate("test3.com")
	GenerateRSAServerCertificate("test3.com")

}
