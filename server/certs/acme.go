package certs

import (
	"os"
	"path"
	"sliver/server/log"

	"golang.org/x/crypto/acme/autocert"
)

const (
	// ACMEDirName - Name of dir to store ACME certs
	ACMEDirName = "acme"
)

var (
	acmeLog = log.NamedLogger("certs", "acme")
)

// GetACMEDir - Dir to store ACME certs
func GetACMEDir(rootDir string) string {
	acmePath := path.Join(rootDir, CertsDirName, ACMEDirName)
	if _, err := os.Stat(acmePath); os.IsNotExist(err) {
		acmeLog.Infof("[mkdir] %s", acmePath)
		os.MkdirAll(acmePath, os.ModePerm)
	}
	return acmePath
}

// GetACMEManager - Get an ACME cert/tls config with the certs
func GetACMEManager(rootDir string, domain string) *autocert.Manager {
	acmeDir := GetACMEDir(rootDir)
	return &autocert.Manager{
		Cache:  autocert.DirCache(acmeDir),
		Prompt: autocert.AcceptTOS,
	}
}
