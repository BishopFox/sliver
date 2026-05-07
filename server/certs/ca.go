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
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bishopfox/sliver/server/assets"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
)

// -----------------------
//  CERTIFICATE AUTHORITY
// -----------------------

// SetupCAs - Ensure certificate authorities exist in storage
func SetupCAs() {
	importCertificateAuthoritiesFromDisk()
	GenerateCertificateAuthority(MtlsImplantCA, "")
	GenerateCertificateAuthority(MtlsServerCA, "")
	GenerateCertificateAuthority(OperatorCA, "operators")
	GenerateCertificateAuthority(HTTPSCA, "")
}

func getCertDir() string {
	rootDir := assets.GetRootAppDir()
	return filepath.Join(rootDir, "certs")
}

// GenerateCertificateAuthority - Creates a new CA cert for a given type
func GenerateCertificateAuthority(caType string, commonName string) (*x509.Certificate, *ecdsa.PrivateKey) {
	if !certificateAuthorityExists(caType) {
		if !importCertificateAuthorityFromDisk(caType) {
			certsLog.Infof("Generating certificate authority for '%s'", caType)
			cert, key := GenerateECCCertificate(caType, commonName, true, false, false)
			SaveCertificateAuthority(caType, cert, key)
		}
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
	caType = filepath.Base(caType)
	caModel := &models.CertificateAuthority{}
	dbSession := db.Session()
	result := dbSession.Where(&models.CertificateAuthority{CAType: caType}).First(caModel)
	if result.Error == nil {
		return []byte(caModel.CertificatePEM), []byte(caModel.PrivateKeyPEM), nil
	}
	if !errors.Is(result.Error, db.ErrRecordNotFound) {
		certsLog.Error(result.Error)
		return nil, nil, result.Error
	}

	certPEM, keyPEM, err := loadCertificateAuthorityPEMFromDisk(caType)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			certsLog.Error(err)
		}
		return nil, nil, err
	}

	SaveCertificateAuthority(caType, certPEM, keyPEM)
	return certPEM, keyPEM, nil
}

// SaveCertificateAuthority - Save the certificate and the key to the database
// doesn't return an error because errors are fatal. If we can't generate CAs,
// then we can't secure communication and we should die a horrible death.
func SaveCertificateAuthority(caType string, cert []byte, key []byte) {
	caType = filepath.Base(caType)
	commonName := commonNameFromCertPEM(cert)
	dbSession := db.Session()
	caModel := &models.CertificateAuthority{}
	result := dbSession.Where(&models.CertificateAuthority{CAType: caType}).First(caModel)
	if errors.Is(result.Error, db.ErrRecordNotFound) {
		caModel = &models.CertificateAuthority{
			CommonName:     commonName,
			CAType:         caType,
			CertificatePEM: string(cert),
			PrivateKeyPEM:  string(key),
		}
		if err := dbSession.Create(caModel).Error; err != nil {
			certsLog.Fatalf("Failed to save certificate authority: %s", err)
		}
		return
	}
	if result.Error != nil {
		certsLog.Fatalf("Failed to save certificate authority: %s", result.Error)
	}

	caModel.CommonName = commonName
	caModel.CertificatePEM = string(cert)
	caModel.PrivateKeyPEM = string(key)
	if err := dbSession.Save(caModel).Error; err != nil {
		certsLog.Fatalf("Failed to save certificate authority: %s", err)
	}
}

func certificateAuthorityExists(caType string) bool {
	caType = filepath.Base(caType)
	dbSession := db.Session()
	result := dbSession.Select("id").Where(&models.CertificateAuthority{CAType: caType}).First(&models.CertificateAuthority{})
	if result.Error == nil {
		return true
	}
	if !errors.Is(result.Error, db.ErrRecordNotFound) {
		certsLog.Error(result.Error)
		return true
	}
	return false
}

func importCertificateAuthoritiesFromDisk() {
	certDir := getCertDir()
	fi, err := os.Stat(certDir)
	if err != nil || !fi.IsDir() {
		return
	}

	importCertificateAuthorityFromDisk(MtlsImplantCA)
	importCertificateAuthorityFromDisk(MtlsServerCA)
	importCertificateAuthorityFromDisk(OperatorCA)
	importCertificateAuthorityFromDisk(HTTPSCA)
}

func importCertificateAuthorityFromDisk(caType string) bool {
	if certificateAuthorityExists(caType) {
		return false
	}
	certPEM, keyPEM, err := loadCertificateAuthorityPEMFromDisk(caType)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			certsLog.Error(err)
		}
		return false
	}
	certsLog.Infof("Importing certificate authority for '%s' from disk", caType)
	SaveCertificateAuthority(caType, certPEM, keyPEM)
	return true
}

func loadCertificateAuthorityPEMFromDisk(caType string) ([]byte, []byte, error) {
	caType = filepath.Base(caType)
	certDir := getCertDir()
	fi, err := os.Stat(certDir)
	if err != nil || !fi.IsDir() {
		return nil, nil, os.ErrNotExist
	}

	caCertPath := filepath.Join(certDir, fmt.Sprintf("%s-ca-cert.pem", caType))
	caKeyPath := filepath.Join(certDir, fmt.Sprintf("%s-ca-key.pem", caType))

	certPEM, err := os.ReadFile(caCertPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, os.ErrNotExist
		}
		return nil, nil, err
	}

	keyPEM, err := os.ReadFile(caKeyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, os.ErrNotExist
		}
		return nil, nil, err
	}
	return certPEM, keyPEM, nil
}

func commonNameFromCertPEM(cert []byte) string {
	block, _ := pem.Decode(cert)
	if block == nil {
		return ""
	}
	parsed, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return ""
	}
	return parsed.Subject.CommonName
}
