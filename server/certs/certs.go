package certs

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net"
	"sliver/server/db"
	"sliver/server/log"
	"time"
)

var (
	certsLog = log.NamedLogger("certs", "certificates")
)

const (
	// Certs are valid for ~3 Years, minus up to 1 year from Now()
	validFor = 3 * (365 * 24 * time.Hour)
)

type CertificateKeyPair struct {
	KeyType     string `json:"key_type"`
	Certificate []byte `json:"certificate"`
	PrivateKey  []byte `json:"private_key"`
}

// SaveCertificate - Save the certificate and the key to the filesystem
func SaveCertificate(caType string, keyType string, commonName string, cert []byte, key []byte) error {
	bucket := db.Bucket(caType)
	bucket.Log.Infof("Saving certificate for %s", commonName)
	keyPair, err := json.Marshal(CertificateKeyPair{
		KeyType:     keyType,
		Certificate: cert,
		PrivateKey:  key,
	})
	if err != nil {
		bucket.Log.Errorf("Failed to marshal key pair %s", err)
		return err
	}
	return bucket.Set(commonName, keyPair)
}

// GetCertificatePEM - Get the PEM encoded certificate & key for a host
func GetCertificatePEM(caType string, commonName string) ([]byte, []byte, error) {
	bucket := db.Bucket(caType)
	rawKeyPair, err := bucket.Get(commonName)
	if err != nil {
		return nil, nil, err
	}
	keyPair, err := json.Unmarshal(rawKeyPair)
	if err != nil {
		return nil, nil, err
	}
	return keyPair.Ceritificate, keyPair.PrivateKey, nil
}

// --------------------------------
//  Generic Certificates Functions
// --------------------------------

// GenerateECCCertificate - Generate a TLS certificate with the given parameters
// We choose some reasonable defaults like Curve, Key Size, ValidFor, etc.
// Returns two strings `cert` and `key` (PEM Encoded).
func GenerateECCCertificate(caType string, commonName string, isCA bool, isClient bool) ([]byte, []byte) {

	certsLog.Infof("Generating TLS certificate (ECC) ...")

	var privateKey interface{}
	var err error

	// Generate private key
	privateKey, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		certsLog.Fatalf("Failed to generate private key: %s", err)
	}

	return generateCertificate(caType, commonName, isCa, isClient, privateKey)
}

// GenerateRSACertificate - Generates a 2048 bit RSA Certificate
func GenerateRSACertificate(caType string, commonName string, isCA bool, isClient bool) ([]byte, []byte) {

	certsLog.Infof("Generating TLS certificate (RSA) ...")

	var privateKey interface{}
	var err error

	// Generate private key
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		certsLog.Fatalf("Failed to generate private key %s", err)
	}
	return generateCertificate(caType, commonName, isCa, isClient, privateKey)
}

func generateCertificate(caType string, commonName string, isCA bool, isClient bool, privateKey interface{}) ([]byte, []byte) {

	// Valid times, subtract random days from .Now()
	notBefore := time.Now()
	days := randomInt(365) * -1 // Within -1 year
	notBefore = notBefore.AddDate(0, 0, days)
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
			Organization: []string{""},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           extKeyUsage,
		BasicConstraintsValid: isCA,
	}

	if !isClient {
		// Host or IP address
		if ip := net.ParseIP(commonName); ip != nil {
			certsLog.Infof("Certificate authenticates IP address: %v", ip)
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			certsLog.Infof("Certificate authenticates host: %v", commonName)
			template.DNSNames = append(template.DNSNames, commonName)
		}
	} else {
		certsLog.Infof("Client certificate authenticates CN: %v", commonName)
		template.Subject.CommonName = commonName
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

func randomInt(max int) int {
	buf := make([]byte, 4)
	rand.Read(buf)
	i := binary.LittleEndian.Uint32(buf)
	return int(i) % max
}
