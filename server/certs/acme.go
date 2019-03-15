package certs

import (
	"crypto/tls"
	"path"

	"golang.org/x/crypto/acme/autocert"
)

const (
	// ACMEDirName - Name of dir to store ACME certs
	ACMEDirName = "acme"
)

// GetACMEDir - Dir to store ACME certs
func GetACMEDir(rootDir string) string {
	return path.Join(rootDir, CertsDirName, ACMEDirName)
}

// GetACMECertificate - Get an ACME cert/tls config with the certs
func GetACMECertificate(rootDir string, domain string) *tls.Config {
	acmeDir := GetACMEDir(rootDir)
	manager := &autocert.Manager{
		Cache:      autocert.DirCache(acmeDir),
		HostPolicy: autocert.HostWhitelist(domain),
		Prompt:     autocert.AcceptTOS,
	}
	return manager.TLSConfig()
}
