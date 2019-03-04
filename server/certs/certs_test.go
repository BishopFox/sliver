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
	GenerateServerCertificate(rootDir, "test3.com", ServersCertDir, false)
	GenerateServerCertificate(rootDir, "test3.com", ServersCertDir, true)
}

func TestGenerateRSACertificate(t *testing.T) {
	rootDir, _ := ioutil.TempDir("", "sliver_test")
	GenerateRSACertificate(rootDir, "test4.com", ServersCertDir, true, false)
	GenerateRSACertificate(rootDir, "test4.com", ServersCertDir, false, true)
}
