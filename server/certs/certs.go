package certs

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"path"
	"sliver/server/log"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	certsLog = log.RootLogger.WithFields(logrus.Fields{
		"pkg":    "certs",
		"stream": "certificates",
	})
)

const (
	validFor       = 365 * 24 * time.Hour
	ClientsCertDir = "clients"
	CertsDirName   = "certs"
	SliversCertDir = "slivers"
	ServersCertDir = "servers"
)

// -------------------
//  LEAF CERTIFICATES
// -------------------

// GenerateSliverCertificate - Generate a certificate signed with a given CA
func GenerateSliverCertificate(rootDir string, sliverName string, save bool) ([]byte, []byte) {
	cert, key := GenerateCertificate(rootDir, sliverName, SliversCertDir, false, true)
	if save {
		SaveCertificate(rootDir, SliversCertDir, sliverName, cert, key)
	}
	return cert, key
}

// GenerateClientCertificate - Generate a certificate signed with a given CA
func GenerateClientCertificate(rootDir string, operator string, save bool) ([]byte, []byte) {
	cert, key := GenerateCertificate(rootDir, operator, ClientsCertDir, false, true)
	if save {
		SaveCertificate(rootDir, ClientsCertDir, operator, cert, key)
	}
	return cert, key
}

// GenerateServerCertificate - Generate a server certificate signed with a given CA
func GenerateServerCertificate(rootDir string, caType string, host string, save bool) ([]byte, []byte) {
	cert, key := GenerateCertificate(rootDir, host, caType, false, false)
	if save {
		SaveCertificate(rootDir, path.Join(ServersCertDir, "ecc"), host, cert, key)
	}
	return cert, key
}

// GenerateServerRSACertificate - Generate a server certificate signed with a given CA
func GenerateServerRSACertificate(rootDir string, caType string, host string, save bool) ([]byte, []byte) {
	cert, key := GenerateRSACertificate(rootDir, caType, host, false, false)
	if save {
		SaveCertificate(rootDir, path.Join(ServersCertDir, "rsa"), host, cert, key)
	}
	return cert, key
}

// GetServerCertificatePEM - Get a server certificate/key pair signed by ca type
func GetServerCertificatePEM(rootDir string, caType string, host string, generateIfNoneExists bool) ([]byte, []byte, error) {

	certsLog.Infof("Getting certificate (ca type = %s) '%s'", caType, host)

	// If not certificate exists for this host we just generate one on the fly
	_, _, err := GetCertificatePEM(rootDir, path.Join(ServersCertDir, "ecc"), host)
	if err != nil {
		if generateIfNoneExists {
			certsLog.Infof("No server certificate, generating ca type = %s '%s'", caType, host)
			GenerateServerCertificate(rootDir, caType, host, true)
		} else {
			return nil, nil, err
		}
	}

	certPEM, keyPEM, err := GetCertificatePEM(rootDir, path.Join(ServersCertDir, "ecc"), host)
	if err != nil {
		certsLog.Infof("Failed to load PEM data %v", err)
		return nil, nil, err
	}

	return certPEM, keyPEM, nil
}

// GetServerRSACertificatePEM - Get a server certificate/key pair signed by ca type
func GetServerRSACertificatePEM(rootDir string, caType string, host string, generateIfNoneExists bool) ([]byte, []byte, error) {

	certsLog.Infof("Getting rsa certificate (ca type = %s) '%s'", caType, host)

	// If not certificate exists for this host we just generate one on the fly
	_, _, err := GetCertificatePEM(rootDir, path.Join(ServersCertDir, "rsa"), host)
	if err != nil {
		if generateIfNoneExists {
			certsLog.Infof("No server certificate, generating ca type = %s '%s'", caType, host)
			GenerateServerRSACertificate(rootDir, caType, host, true)
		} else {
			return nil, nil, err
		}
	}

	certPEM, keyPEM, err := GetCertificatePEM(rootDir, path.Join(ServersCertDir, "rsa"), host)
	if err != nil {
		certsLog.Infof("Failed to load PEM data %v", err)
		return nil, nil, err
	}

	return certPEM, keyPEM, nil
}

// SaveCertificate - Save the certificate and the key to the filesystem
func SaveCertificate(rootDir string, prefix string, host string, cert []byte, key []byte) {

	storageDir := path.Join(rootDir, CertsDirName, prefix)
	if _, err := os.Stat(storageDir); os.IsNotExist(err) {
		os.MkdirAll(storageDir, os.ModePerm)
	}

	host = path.Base(host)
	certFilePath := path.Join(storageDir, fmt.Sprintf("%s-cert.pem", host))
	keyFilePath := path.Join(storageDir, fmt.Sprintf("%s-key.pem", host))

	certsLog.Infof("Saving certificate to: %s", certFilePath)
	err := ioutil.WriteFile(certFilePath, cert, 0600)
	if err != nil {
		certsLog.Fatalf("Failed write certificate data to: %s", certFilePath)
	}

	certsLog.Infof("Saving key to: %s", keyFilePath)
	err = ioutil.WriteFile(keyFilePath, key, 0600)
	if err != nil {
		certsLog.Fatalf("Failed write key data to: %s", keyFilePath)
	}
}

// GetCertificatePEM - Get the PEM encoded certificate & key for a host
func GetCertificatePEM(rootDir string, prefix string, host string) ([]byte, []byte, error) {

	storageDir := path.Join(rootDir, CertsDirName, prefix)
	if _, err := os.Stat(storageDir); os.IsNotExist(err) {
		return nil, nil, err
	}

	host = path.Base(host)
	certFileName := fmt.Sprintf("%s-cert.pem", host)
	keyFileName := fmt.Sprintf("%s-key.pem", host)

	certFilePath := path.Join(storageDir, certFileName)
	keyFilePath := path.Join(storageDir, keyFileName)

	certPEM, err := ioutil.ReadFile(certFilePath)
	if err != nil {
		certsLog.Infof("Failed to load %v", err)
		return nil, nil, err
	}

	keyPEM, err := ioutil.ReadFile(keyFilePath)
	if err != nil {
		certsLog.Infof("Failed to load %v", err)
		return nil, nil, err
	}

	return certPEM, keyPEM, nil
}

// -----------------------
//  CERTIFICATE AUTHORITY
// -----------------------

// GenerateCertificateAuthority - Creates a new CA cert for a given type
func GenerateCertificateAuthority(rootDir string, caType string, save bool) ([]byte, []byte) {
	cert, key := GenerateCertificate(rootDir, "", caType, true, false)
	if save {
		SaveCertificateAuthority(rootDir, caType, cert, key)
	}
	return cert, key
}

// GetCertificateAuthority - Get the current CA certificate
func GetCertificateAuthority(rootDir string, caType string) (*x509.Certificate, *ecdsa.PrivateKey, error) {

	certPEM, keyPEM, err := GetCertificateAuthorityPEM(rootDir, caType)
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
func GetCertificateAuthorityPEM(rootDir string, caType string) ([]byte, []byte, error) {

	caType = path.Base(caType)
	caCertPath := path.Join(rootDir, CertsDirName, fmt.Sprintf("%s-ca-cert.pem", caType))
	caKeyPath := path.Join(rootDir, CertsDirName, fmt.Sprintf("%s-ca-key.pem", caType))

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
func SaveCertificateAuthority(rootDir string, caType string, cert []byte, key []byte) {

	storageDir := path.Join(rootDir, CertsDirName)
	if _, err := os.Stat(storageDir); os.IsNotExist(err) {
		os.MkdirAll(storageDir, os.ModePerm)
	}

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

// --------------------------------
//  Generic Certificates Functions
// --------------------------------

// GenerateCertificate - Generate a TLS certificate with the given parameters
// We choose some reasonable defaults like Curve, Key Size, ValidFor, etc.
// Returns two strings `cert` and `key` (PEM Encoded).
func GenerateCertificate(rootDir string, host string, caType string, isCA bool, isClient bool) ([]byte, []byte) {

	certsLog.Infof("Generating new TLS certificate (ECC) ...")

	var privateKey interface{}
	var err error

	// Generate private key
	privateKey, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		certsLog.Fatalf("Failed to generate private key: %s", err)
	}

	// Valid times
	notBefore := time.Now() // TODO: Randomize
	notAfter := notBefore.Add(validFor)
	certsLog.Infof("Valid from %v to %v", notBefore, notAfter)

	// Serial number
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)
	certsLog.Infof("Serial Number: %d", serialNumber)

	var extKeyUsage []x509.ExtKeyUsage

	if isCA {
		certsLog.Infof("Authority certificate")
		extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageAny}
	} else if isClient {
		certsLog.Infof("Client authentication certificate")
		extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	} else {
		certsLog.Infof("Server authentication certificate")
		extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	}
	certsLog.Infof("ExtKeyUsage = %v", extKeyUsage)

	// Certificate template
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Sliver Hive"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           extKeyUsage,
		BasicConstraintsValid: isCA,
	}

	if !isClient {
		// Host or IP address
		if ip := net.ParseIP(host); ip != nil {
			certsLog.Infof("Certificate authenticates IP address: %v", ip)
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			certsLog.Infof("Certificate authenticates host: %v", host)
			template.DNSNames = append(template.DNSNames, host)
		}
	} else {
		certsLog.Infof("Client certificate authenticates CN: %v", host)
		template.Subject.CommonName = host
	}

	// Sign certificate or self-sign if CA
	var derBytes []byte
	if isCA {
		certsLog.Infof("Ceritificate is an AUTHORITY")
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
		derBytes, err = x509.CreateCertificate(rand.Reader, &template, &template, publicKey(privateKey), privateKey)
	} else {
		caCert, caKey, err := GetCertificateAuthority(rootDir, caType) // Sign the new ceritificate with our CA
		if err != nil {
			certsLog.Fatalf("Invalid ca type (%s): %v", caType, err)
		}
		derBytes, err = x509.CreateCertificate(rand.Reader, &template, caCert, publicKey(privateKey), caKey)
	}
	if err != nil {
		certsLog.Fatalf("Failed to create certificate: %s", err)
	}

	// Encode certificate and key
	certOut := bytes.NewBuffer([]byte{})
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	keyOut := bytes.NewBuffer([]byte{})
	pem.Encode(keyOut, pemBlockForKey(privateKey))

	return certOut.Bytes(), keyOut.Bytes()
}

// GenerateRSACertificate - Generates a 2048 bit RSA Certificate
func GenerateRSACertificate(rootDir string, caType string, host string, isCA bool, isClient bool) ([]byte, []byte) {

	certsLog.Infof("Generating new TLS certificate (RSA) ...")

	var privateKey interface{}
	var err error

	// Generate private key
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		certsLog.Fatalf("Failed to generate private key: %s", err)
	}

	// Valid times
	notBefore := time.Now()
	notAfter := notBefore.Add(validFor)
	certsLog.Infof("Valid from %v to %v", notBefore, notAfter)

	// Serial number
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)
	certsLog.Infof("Serial Number: %d", serialNumber)

	var extKeyUsage []x509.ExtKeyUsage

	if isCA {
		certsLog.Infof("Authority certificate")
		extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageAny}
	} else if isClient {
		certsLog.Infof("Client authentication certificate")
		extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	} else {
		certsLog.Infof("Server authentication certificate")
		extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	}
	certsLog.Infof("ExtKeyUsage = %v", extKeyUsage)

	// Certificate template
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Sliver Hive"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           extKeyUsage,
		BasicConstraintsValid: isCA,
	}

	if !isClient {
		// Host or IP address
		if ip := net.ParseIP(host); ip != nil {
			certsLog.Infof("Certificate authenticates IP address: %v", ip)
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			certsLog.Infof("Certificate authenticates host: %v", host)
			template.DNSNames = append(template.DNSNames, host)
		}
	} else {
		certsLog.Infof("Client certificate authenticates CN: %v", host)
		template.Subject.CommonName = host
	}

	// Sign certificate or self-sign if CA
	var derBytes []byte
	if isCA {
		certsLog.Infof("Ceritificate is an AUTHORITY")
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
		derBytes, err = x509.CreateCertificate(rand.Reader, &template, &template, publicKey(privateKey), privateKey)
	} else {
		caCert, caKey, err := GetCertificateAuthority(rootDir, caType) // Sign the new ceritificate with our CA
		if err != nil {
			certsLog.Fatalf("Invalid ca type (%s): %v", caType, err)
		}
		derBytes, err = x509.CreateCertificate(rand.Reader, &template, caCert, publicKey(privateKey), caKey)
	}
	if err != nil {
		certsLog.Fatalf("Failed to create certificate: %s", err)
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
