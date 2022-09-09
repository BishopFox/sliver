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
	"time"

	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"github.com/bishopfox/sliver/server/log"
)

const (
	// ECCKey - Namespace for ECC keys
	ECCKey = "ecc"

	// RSAKey - Namespace for RSA keys
	RSAKey = "rsa"
)

var (
	certsLog = log.NamedLogger("certs", "certificates")

	// ErrCertDoesNotExist - Returned if a GetCertificate() is called for a cert/cn that does not exist
	ErrCertDoesNotExist = errors.New("Certificate does not exist")
)

// saveCertificate - Save the certificate and the key to the filesystem
func saveCertificate(caType string, keyType string, commonName string, cert []byte, key []byte) error {

	if keyType != ECCKey && keyType != RSAKey {
		return fmt.Errorf("Invalid key type '%s'", keyType)
	}

	certsLog.Infof("Saving certificate for cn = '%s'", commonName)

	certModel := &models.Certificate{
		CommonName:     commonName,
		CAType:         caType,
		KeyType:        keyType,
		CertificatePEM: string(cert),
		PrivateKeyPEM:  string(key),
	}

	dbSession := db.Session()
	result := dbSession.Create(&certModel)

	return result.Error
}

// GetECCCertificate - Get an ECC certificate
func GetECCCertificate(caType string, commonName string) ([]byte, []byte, error) {
	return GetCertificate(caType, ECCKey, commonName)
}

// GetRSACertificate - Get an RSA certificate
func GetRSACertificate(caType string, commonName string) ([]byte, []byte, error) {
	return GetCertificate(caType, RSAKey, commonName)
}

// GetCertificate - Get the PEM encoded certificate & key for a host
func GetCertificate(caType string, keyType string, commonName string) ([]byte, []byte, error) {

	if keyType != ECCKey && keyType != RSAKey {
		return nil, nil, fmt.Errorf("Invalid key type '%s'", keyType)
	}

	certsLog.Infof("Getting certificate ca type = %s, cn = '%s'", caType, commonName)

	certModel := models.Certificate{}
	dbSession := db.Session()
	result := dbSession.Where(&models.Certificate{
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

// RemoveCertificate - Remove a certificate from the cert store
func RemoveCertificate(caType string, keyType string, commonName string) error {
	if keyType != ECCKey && keyType != RSAKey {
		return fmt.Errorf("Invalid key type '%s'", keyType)
	}
	dbSession := db.Session()
	err := dbSession.Where(&models.Certificate{
		CAType:     caType,
		KeyType:    keyType,
		CommonName: commonName,
	}).Delete(&models.Certificate{}).Error
	return err
}

// --------------------------------
//  Generic Certificate Functions
// --------------------------------

// GenerateECCCertificate - Generate a TLS certificate with the given parameters
// We choose some reasonable defaults like Curve, Key Size, ValidFor, etc.
// Returns two strings `cert` and `key` (PEM Encoded).
func GenerateECCCertificate(caType string, commonName string, isCA bool, isClient bool) ([]byte, []byte) {

	certsLog.Infof("Generating TLS certificate (ECC) for '%s' ...", commonName)

	var privateKey interface{}
	var err error

	// Generate private key
	curves := []elliptic.Curve{elliptic.P521(), elliptic.P384(), elliptic.P256()}
	curve := curves[randomInt(len(curves))]
	privateKey, err = ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		certsLog.Fatalf("Failed to generate private key: %s", err)
	}
	subject := pkix.Name{
		CommonName: commonName,
	}
	return generateCertificate(caType, subject, isCA, isClient, privateKey)
}

// GenerateRSACertificate - Generates an RSA Certificate
func GenerateRSACertificate(caType string, commonName string, isCA bool, isClient bool) ([]byte, []byte) {

	certsLog.Debugf("Generating TLS certificate (RSA) for '%s' ...", commonName)

	var privateKey interface{}
	var err error

	// Generate private key
	privateKey, err = rsa.GenerateKey(rand.Reader, rsaKeySize())
	if err != nil {
		certsLog.Fatalf("Failed to generate private key %s", err)
	}
	subject := pkix.Name{
		CommonName: commonName,
	}
	return generateCertificate(caType, subject, isCA, isClient, privateKey)
}

func generateCertificate(caType string, subject pkix.Name, isCA bool, isClient bool, privateKey interface{}) ([]byte, []byte) {

	// Valid times, subtract random days from .Now()
	notBefore := time.Now()
	days := randomInt(365) * -1 // Within -1 year
	notBefore = notBefore.AddDate(0, 0, days)
	notAfter := notBefore.Add(randomValidFor())
	certsLog.Debugf("Valid from %v to %v", notBefore, notAfter)

	// Serial number
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)
	certsLog.Debugf("Serial Number: %d", serialNumber)

	var keyUsage x509.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
	var extKeyUsage []x509.ExtKeyUsage

	if isCA {
		certsLog.Debugf("Authority certificate")
		keyUsage = x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
		extKeyUsage = []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		}
	} else if isClient {
		certsLog.Debugf("Client authentication certificate")
		extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	} else {
		certsLog.Debugf("Server authentication certificate")
		extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	}
	certsLog.Debugf("ExtKeyUsage = %v", extKeyUsage)

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
			certsLog.Debugf("Certificate authenticates IP address: %v", ip)
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			certsLog.Debugf("Certificate authenticates host: %v", subject.CommonName)
			template.DNSNames = append(template.DNSNames, subject.CommonName)
		}
	} else {
		certsLog.Debugf("Client certificate authenticates CN: %v", subject.CommonName)
	}

	// Sign certificate or self-sign if CA
	var certErr error
	var derBytes []byte
	if isCA {
		certsLog.Debugf("Certificate is an AUTHORITY")
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
		derBytes, certErr = x509.CreateCertificate(rand.Reader, &template, &template, publicKey(privateKey), privateKey)
	} else {
		caCert, caKey, err := GetCertificateAuthority(caType) // Sign the new certificate with our CA
		if err != nil {
			certsLog.Fatalf("Invalid ca type (%s): %v", caType, err)
		}
		derBytes, certErr = x509.CreateCertificate(rand.Reader, &template, caCert, publicKey(privateKey), caKey)
	}
	if certErr != nil {
		// We maybe don't want this to be fatal, but it should basically never happen afaik
		certsLog.Fatalf("Failed to create certificate: %s", certErr)
	}

	// Encode certificate and key
	certOut := bytes.NewBuffer([]byte{})
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	keyOut := bytes.NewBuffer([]byte{})
	pem.Encode(keyOut, pemBlockForKey(privateKey))

	return certOut.Bytes(), keyOut.Bytes()
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

func pemBlockForKey(priv interface{}) *pem.Block {
	switch key := priv.(type) {
	case *rsa.PrivateKey:
		data := x509.MarshalPKCS1PrivateKey(key)
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: data}
	case *ecdsa.PrivateKey:
		data, err := x509.MarshalECPrivateKey(key)
		if err != nil {
			certsLog.Fatalf("Unable to marshal ECDSA private key: %v", err)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: data}
	default:
		return nil
	}
}

func randomInt(max int) int {
	buf := make([]byte, 4)
	rand.Read(buf)
	i := binary.LittleEndian.Uint32(buf)
	return int(i) % max
}

func randomValidFor() time.Duration {
	validFor := 3 * (365 * 24 * time.Hour)
	switch insecureRand.Intn(2) {
	case 0:
		validFor = 2 * (365 * 24 * time.Hour)
	case 1:
		validFor = 3 * (365 * 24 * time.Hour)
	}
	return validFor
}

func rsaKeySize() int {
	rsaKeySizes := []int{4096, 2048}
	return rsaKeySizes[randomInt(len(rsaKeySizes))]
}
