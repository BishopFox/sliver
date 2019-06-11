package certs

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/bishopfox/sliver/server/assets"
)

// -----------------------
//  CERTIFICATE AUTHORITY
// -----------------------

// SetupCAs - Creates directories for certs
func SetupCAs() {
	GenerateCertificateAuthority(ServerCA)
	GenerateCertificateAuthority(SliverCA)
	GenerateCertificateAuthority(OperatorCA)
	GenerateCertificateAuthority(HTTPSCA)
}

func getCertDir() string {
	rootDir := assets.GetRootAppDir()
	certDir := path.Join(rootDir, "certs")
	if _, err := os.Stat(certDir); os.IsNotExist(err) {
		err := os.MkdirAll(certDir, os.ModePerm)
		if err != nil {
			certsLog.Fatalf("Failed to create cert dir %s", err)
		}
	}
	return certDir
}

// GenerateCertificateAuthority - Creates a new CA cert for a given type
func GenerateCertificateAuthority(caType string) (*x509.Certificate, *ecdsa.PrivateKey) {
	storageDir := getCertDir()
	certFilePath := path.Join(storageDir, fmt.Sprintf("%s-ca-cert.pem", caType))
	if _, err := os.Stat(certFilePath); os.IsNotExist(err) {
		certsLog.Infof("Generating certificate authority for '%s'", caType)
		cert, key := GenerateECCCertificate(caType, "", true, false)
		SaveCertificateAuthority(caType, cert, key)
	}
	cert, key, err := GetCertificateAuthority(caType)
	if err != nil {
		certsLog.Fatalf("Failed to load CA %s", err)
	}
	return cert, key
}

// GetCertificateAuthority - Get the current CA certificate
func GetCertificateAuthority(caType string) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	certPEM, keyPEM, err := GetCertificateAuthorityPEM(caType)
	if err != nil {
		return nil, nil, err
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		certsLog.Error("Failed to parse certificate PEM")
		return nil, nil, err
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		certsLog.Error("Failed to parse certificate: " + err.Error())
		return nil, nil, err
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		certsLog.Error("Failed to parse certificate PEM")
		return nil, nil, err
	}
	key, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		certsLog.Error(err)
		return nil, nil, err
	}

	return cert, key, nil
}

// GetCertificateAuthorityPEM - Get PEM encoded CA cert/key
func GetCertificateAuthorityPEM(caType string) ([]byte, []byte, error) {
	caType = path.Base(caType)
	caCertPath := path.Join(getCertDir(), fmt.Sprintf("%s-ca-cert.pem", caType))
	caKeyPath := path.Join(getCertDir(), fmt.Sprintf("%s-ca-key.pem", caType))

	certPEM, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		certsLog.Error(err)
		return nil, nil, err
	}

	keyPEM, err := ioutil.ReadFile(caKeyPath)
	if err != nil {
		certsLog.Error(err)
		return nil, nil, err
	}
	return certPEM, keyPEM, nil
}

// SaveCertificateAuthority - Save the certificate and the key to the filesystem
// doesn't return an error because errors are fatal. If we can't generate CAs,
// then we can't secure comms and we should die a horrible death.
func SaveCertificateAuthority(caType string, cert []byte, key []byte) {

	storageDir := getCertDir()
	if _, err := os.Stat(storageDir); os.IsNotExist(err) {
		os.MkdirAll(storageDir, os.ModePerm)
	}

	// CAs get written to the filesystem since we control the names and makes them
	// easier to move around/backup
	certFilePath := path.Join(storageDir, fmt.Sprintf("%s-ca-cert.pem", caType))
	keyFilePath := path.Join(storageDir, fmt.Sprintf("%s-ca-key.pem", caType))

	err := ioutil.WriteFile(certFilePath, cert, 0600)
	if err != nil {
		certsLog.Fatalf("Failed write certificate data to: %s", certFilePath)
	}

	err = ioutil.WriteFile(keyFilePath, key, 0600)
	if err != nil {
		certsLog.Fatalf("Failed write certificate data to: %s", keyFilePath)
	}
}
