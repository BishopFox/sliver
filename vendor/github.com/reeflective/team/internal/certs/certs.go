package certs

/*
   team - Embedded teamserver for Go programs and CLI applications
   Copyright (C) 2023 Reeflective

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
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	insecureRand "math/rand"
	"net"
	"path/filepath"
	"time"

	"github.com/reeflective/team/internal/assets"
	"github.com/reeflective/team/internal/db"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	// ECCKey - Namespace for ECC keys.
	ECCKey = "ecc"

	// RSAKey - Namespace for RSA keys.
	RSAKey = "rsa"

	// Internal constants.
	daysInYear      = 365
	hoursInDay      = 24
	validForYears   = 3
	serialNumberLen = 128
)

// ErrCertDoesNotExist - Returned if a GetCertificate() is called for a cert/cn that does not exist.
var ErrCertDoesNotExist = errors.New("Certificate does not exist")

// Manager is used to manage the certificate infrastructure for a given teamserver.
// Has access to a given database for storage, a logger and an abstract filesystem.
type Manager struct {
	appName  string
	appDir   string
	log      *logrus.Entry
	database *gorm.DB
	fs       *assets.FS
}

// NewManager initializes and returns a certificate manager for a given teamserver.
// The returned manager will have ensured that all certificate authorities are initialized
// and working, or will create them if needed.
// Any critical error happening at initialization time will send a log.Fatal event to the
// provided logger. If the latter has no modified log.ExitFunc, this will make the server
// panic and exit.
func NewManager(filesystem *assets.FS, db *gorm.DB, logger *logrus.Entry, appName, appDir string) *Manager {
	certs := &Manager{
		appName:  appName,
		appDir:   appDir,
		log:      logger,
		database: db,
		fs:       filesystem,
	}

	certs.generateCA(userCA, "teamusers")

	return certs
}

func (c *Manager) db() *gorm.DB {
	return c.database.Session(&gorm.Session{
		FullSaveAssociations: true,
	})
}

// GetECCCertificate - Get an ECC certificate.
func (c *Manager) GetECCCertificate(caType string, commonName string) ([]byte, []byte, error) {
	return c.GetCertificate(caType, ECCKey, commonName)
}

// GetRSACertificate - Get an RSA certificate.
func (c *Manager) GetRSACertificate(caType string, commonName string) ([]byte, []byte, error) {
	return c.GetCertificate(caType, RSAKey, commonName)
}

// GetCertificate - Get the PEM encoded certificate & key for a host.
func (c *Manager) GetCertificate(caType string, keyType string, commonName string) ([]byte, []byte, error) {
	if keyType != ECCKey && keyType != RSAKey {
		return nil, nil, fmt.Errorf("Invalid key type '%s'", keyType)
	}

	c.log.Infof("Getting certificate ca type = %s, cn = '%s'", caType, commonName)

	certModel := db.Certificate{}
	result := c.db().Where(&db.Certificate{
		CAType:     caType,
		KeyType:    keyType,
		CommonName: commonName,
	}).First(&certModel)

	if errors.Is(result.Error, db.ErrRecordNotFound) {
		return nil, nil, ErrCertDoesNotExist
	}

	if result.Error != nil {
		return nil, nil, result.Error
	}

	return []byte(certModel.CertificatePEM), []byte(certModel.PrivateKeyPEM), nil
}

// RemoveCertificate - Remove a certificate from the cert store.
func (c *Manager) RemoveCertificate(caType string, keyType string, commonName string) error {
	if keyType != ECCKey && keyType != RSAKey {
		return fmt.Errorf("Invalid key type '%s'", keyType)
	}

	c.log.Infof("Deleting certificate for cn = '%s'", commonName)

	err := c.db().Where(&db.Certificate{
		CAType:     caType,
		KeyType:    keyType,
		CommonName: commonName,
	}).Delete(&db.Certificate{}).Error

	return err
}

// --------------------------------
//  Generic Certificate Functions
// --------------------------------

// GenerateECCCertificate - Generate a TLS certificate with the given parameters
// We choose some reasonable defaults like Curve, Key Size, ValidFor, etc.
// Returns two strings `cert` and `key` (PEM Encoded).
func (c *Manager) GenerateECCCertificate(caType string, commonName string, isCA bool, isClient bool) ([]byte, []byte) {
	c.log.Infof("Generating TLS certificate (ECC) for '%s' ...", commonName)

	var privateKey interface{}
	var err error

	// Generate private key
	curves := []elliptic.Curve{elliptic.P521(), elliptic.P384(), elliptic.P256()}
	curve := curves[randomInt(len(curves))]

	privateKey, err = ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		c.log.Fatalf("Failed to generate private key: %s", err)
	}

	subject := pkix.Name{
		CommonName: commonName,
	}

	return c.generateCertificate(caType, subject, isCA, isClient, privateKey)
}

// GenerateRSACertificate - Generates an RSA Certificate.
func (c *Manager) GenerateRSACertificate(caType string, commonName string, isCA bool, isClient bool) ([]byte, []byte) {
	c.log.Debugf("Generating TLS certificate (RSA) for '%s' ...", commonName)

	var privateKey interface{}
	var err error

	// Generate private key
	privateKey, err = rsa.GenerateKey(rand.Reader, rsaKeySize())
	if err != nil {
		c.log.Fatalf("Failed to generate private key: %s", err)
	}

	subject := pkix.Name{
		CommonName: commonName,
	}

	return c.generateCertificate(caType, subject, isCA, isClient, privateKey)
}

func (c *Manager) generateCertificate(caType string, subject pkix.Name, isCA bool, isClient bool, privateKey interface{}) ([]byte, []byte) {
	// Valid times, subtract random days from .Now()
	notBefore := time.Now()
	days := randomInt(daysInYear) * -1 // Within -1 year
	notBefore = notBefore.AddDate(0, 0, days)
	notAfter := notBefore.Add(randomValidFor())
	c.log.Debugf("Valid from %v to %v", notBefore, notAfter)

	// Serial number
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), serialNumberLen)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)
	c.log.Debugf("Serial Number: %d", serialNumber)

	keyUsage := x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
	var extKeyUsage []x509.ExtKeyUsage

	switch {
	case isCA:
		c.log.Debugf("Authority certificate")

		keyUsage = x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
		extKeyUsage = []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		}
	case isClient:
		c.log.Debugf("Client authentication certificate")

		extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	default:
		c.log.Debugf("Server authentication certificate")

		extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	}

	c.log.Debugf("ExtKeyUsage = %v", extKeyUsage)

	// Certificate template
	template := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               subject,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              keyUsage,
		ExtKeyUsage:           extKeyUsage,
		BasicConstraintsValid: isCA,
	}

	if !isClient {
		// Host or IP address
		if ip := net.ParseIP(subject.CommonName); ip != nil {
			c.log.Debugf("Certificate authenticates IP address: %v", ip)
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			c.log.Debugf("Certificate authenticates host: %v", subject.CommonName)
			template.DNSNames = append(template.DNSNames, subject.CommonName)
		}
	} else {
		c.log.Debugf("Client certificate authenticates CN: %v", subject.CommonName)
	}

	// Sign certificate or self-sign if CA
	var certErr error
	var derBytes []byte

	if isCA {
		c.log.Debugf("Certificate is an AUTHORITY")

		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
		derBytes, certErr = x509.CreateCertificate(rand.Reader, &template, &template, publicKey(privateKey), privateKey)
	} else {
		caCert, caKey, err := c.getCA(caType) // Sign the new certificate with our CA
		if err != nil {
			c.log.Fatalf("Invalid ca type (%s): %s", caType, err)
		}
		derBytes, certErr = x509.CreateCertificate(rand.Reader, &template, caCert, publicKey(privateKey), caKey)
	}

	if certErr != nil {
		// We maybe don't want this to be fatal, but it should basically never happen afaik
		c.log.Fatalf("Failed to create certificate: %s", certErr)
	}

	// Encode certificate and key
	certOut := bytes.NewBuffer([]byte{})
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	keyOut := bytes.NewBuffer([]byte{})
	pem.Encode(keyOut, c.pemBlockForKey(privateKey))

	return certOut.Bytes(), keyOut.Bytes()
}

func (c *Manager) saveCertificate(caType string, keyType string, commonName string, cert []byte, key []byte) error {
	if keyType != ECCKey && keyType != RSAKey {
		return fmt.Errorf("Invalid key type '%s'", keyType)
	}

	c.log.Infof("Saving certificate for cn = '%s'", commonName)

	certModel := &db.Certificate{
		CommonName:     commonName,
		CAType:         caType,
		KeyType:        keyType,
		CertificatePEM: string(cert),
		PrivateKeyPEM:  string(key),
	}

	result := c.db().Create(&certModel)

	return result.Error
}

// getCertDir returns the directory (and makes it if needed) for writing certificate backups.
func (c *Manager) getCertDir() string {
	rootDir := c.appDir
	certDir := filepath.Join(rootDir, "certs")

	err := c.fs.MkdirAll(certDir, assets.DirPerm)
	if err != nil {
		c.log.Fatalf("Failed to create cert dir: %s", err)
	}

	return certDir
}

func (c *Manager) pemBlockForKey(priv interface{}) *pem.Block {
	switch key := priv.(type) {
	case *rsa.PrivateKey:
		data := x509.MarshalPKCS1PrivateKey(key)
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: data}
	case *ecdsa.PrivateKey:
		data, err := x509.MarshalECPrivateKey(key)
		if err != nil {
			c.log.Fatalf("Unable to marshal ECDSA private key: %v", err)
		}

		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: data}
	default:
		return nil
	}
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func randomInt(max int) int {
	intLen := 4
	buf := make([]byte, intLen)
	rand.Read(buf)
	i := binary.LittleEndian.Uint32(buf)

	return int(i) % max
}

func randomValidFor() time.Duration {
	validFor := validForYears * (daysInYear * hoursInDay * time.Hour)

	switch insecureRand.Intn(2) {
	case 0:
		validFor = (validForYears - 1) * (daysInYear * hoursInDay * time.Hour)
	case 1:
		validFor = validForYears * (daysInYear * hoursInDay * time.Hour)
	}

	return validFor
}

func rsaKeySize() int {
	rsaKeySizes := []int{4096, 2048}
	return rsaKeySizes[randomInt(len(rsaKeySizes))]
}
