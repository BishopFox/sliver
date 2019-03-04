package certs

import (
	"io/ioutil"
	"testing"
)

func TestGenerateSliverCertificate(t *testing.T) {
	rootDir, _ := ioutil.TempDir("", "sliver_test")
	GenerateCertificateAuthority(rootDir, SliversCertDir, true)
	GenerateSliverCertificate(rootDir, "test1", false)
	GenerateSliverCertificate(rootDir, "test1", true)
}

func TestGenerateClientCertificate(t *testing.T) {
	rootDir, _ := ioutil.TempDir("", "sliver_test")
	GenerateCertificateAuthority(rootDir, ClientsCertDir, true)
	GenerateClientCertificate(rootDir, "test2", false)
	GenerateClientCertificate(rootDir, "test2", true)
}

func TestGenerateServerCertificate(t *testing.T) {
	rootDir, _ := ioutil.TempDir("", "sliver_test")
	GenerateCertificateAuthority(rootDir, ServersCertDir, true)
	GenerateServerCertificate(rootDir, ServersCertDir, "test3.com", false)
	GenerateServerCertificate(rootDir, ServersCertDir, "test3.com", true)
}

func TestGenerateRSACertificate(t *testing.T) {
	rootDir, _ := ioutil.TempDir("", "sliver_test")
	GenerateCertificateAuthority(rootDir, ServersCertDir, true)
	GenerateRSACertificate(rootDir, ServersCertDir, "test4.com", true, false)
	GenerateRSACertificate(rootDir, ServersCertDir, "test4.com", false, true)
}
