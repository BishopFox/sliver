package transports

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

	---
	This package contains wrappers around Golang's crypto package that make it easier to use
	we manage things like the nonces, key gen, etc.
*/

import (
	"crypto/aes"
	"crypto/cipher"
	secureRand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"os"

	// {{if .Config.Debug}}
	"log"
	// {{end}}
)

const (
	// AESKeySize - Always use 256 bit keys
	AESKeySize = 16

	// GCMNonceSize - 96 bit nonces for GCM
	GCMNonceSize = 12
)

// AESKey - 128 bit key
type AESKey [AESKeySize]byte

// FromBytes - Creates an AESKey from bytes
func (AESKey) FromBytes(data []byte) AESKey {
	var key AESKey
	copy(key[:], data[:AESKeySize])
	return key
}

// AESIV - 128 bit IV
type AESIV [aes.BlockSize]byte

// RandomAESKey - Generate random ID of randomIDSize bytes
func RandomAESKey() AESKey {
	randBuf := make([]byte, 64)
	n, err := secureRand.Read(randBuf)
	if n != 64 || err != nil {
		panic("[[GenerateCanary]]") // If we can't securely generate keys then we die
	}
	digest := sha256.Sum256(randBuf)
	var key AESKey
	copy(key[:], digest[:AESKeySize])
	return key
}

// RandomAESIV - 128 bit Random IV
func RandomAESIV() AESIV {
	data := RandomAESKey()
	var iv AESIV
	copy(iv[:], data[:16])
	return iv
}

// RSAEncrypt - Encrypt a msg with a public rsa key
func RSAEncrypt(msg []byte, pub *rsa.PublicKey) ([]byte, error) {
	hash := sha256.New()
	ciphertext, err := rsa.EncryptOAEP(hash, secureRand.Reader, pub, msg, nil)
	if err != nil {
		return nil, err
	}
	return ciphertext, nil
}

// RSADecrypt - Decrypt ciphertext with rsa private key
func RSADecrypt(ciphertext []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	hash := sha256.New()
	plaintext, err := rsa.DecryptOAEP(hash, secureRand.Reader, privateKey, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

// GCMEncrypt - Encrypt using AES GCM
func GCMEncrypt(key AESKey, plaintext []byte) ([]byte, error) {
	block, _ := aes.NewCipher(key[:])
	nonce := make([]byte, GCMNonceSize)
	if _, err := io.ReadFull(secureRand.Reader, nonce); err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)

	// Prepend nonce to ciphertext
	ciphertext = append(nonce, ciphertext...)
	return ciphertext, nil
}

// GCMDecrypt - Decrypt GCM ciphertext
func GCMDecrypt(key AESKey, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < GCMNonceSize+1 {
		return nil, errors.New("[[GenerateCanary]]")
	}
	block, _ := aes.NewCipher(key[:])
	aesgcm, _ := cipher.NewGCM(block)
	plaintext, err := aesgcm.Open(nil, ciphertext[:GCMNonceSize], ciphertext[GCMNonceSize:], nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

// tlsConfig - A wrapper around several elements needed to produce a TLS config for either
// a server or a client, depending on the direction of the connection to the implant.
type tlsConfig struct {
	ca   *x509.CertPool
	cert tls.Certificate
	key  []byte
}

// newCredentialsTLS - Generates a new custom tlsConfig loaded with the Slivers Certificate Authority.
// It may thus load and export any TLS configuration for talking with an implant, bind or reverse.
func newCredentialsTLS() (creds *tlsConfig) {

	// Load CA cert
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(caCertPEM))

	cert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Error loading server certificate: %v", err)
		// {{end}}
		os.Exit(5)
	}

	creds = &tlsConfig{
		ca:   caCertPool,
		cert: cert,
	}

	return creds
}

// ClientConfig - TLS config used when we dial an implant over Mutual TLS.
// This makes use of a custom function for skipping (only) hostname validation,
// because the tlsConfig verifies only against its own Certificate Authority.
func (t *tlsConfig) ClientConfig(host string) (c *tls.Config) {

	// Client config with custom certificate validation routine
	c = &tls.Config{
		Certificates:          []tls.Certificate{t.cert},
		RootCAs:               t.ca,
		InsecureSkipVerify:    true, // Don't worry I sorta know what I'm doing
		VerifyPeerCertificate: rootOnlyVerifyCertificate,
	}
	c.BuildNameToCertificate()

	return c
}

// ServerConfig - TLS config used when we listen for incoming Mutual TLS implant connections.
func (t *tlsConfig) ServerConfig(host string) (c *tls.Config) {

	// Server configuration
	c = &tls.Config{
		RootCAs:                  t.ca,
		ClientAuth:               tls.RequireAndVerifyClientCert,
		ClientCAs:                t.ca,
		Certificates:             []tls.Certificate{t.cert},
		CipherSuites:             []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384},
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS12,
	}
	c.BuildNameToCertificate()

	return
}

// rootOnlyVerifyCertificate - Go doesn't provide a method for only skipping hostname validation so
// we have to disable all of the fucking certificate validation and re-implement everything.
// https://github.com/golang/go/issues/21971
func rootOnlyVerifyCertificate(rawCerts [][]byte, _ [][]*x509.Certificate) error {

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(caCertPEM))
	if !ok {
		// {{if .Config.Debug}}
		log.Printf("Failed to parse root certificate")
		// {{end}}
		os.Exit(3)
	}

	cert, err := x509.ParseCertificate(rawCerts[0]) // We should only get one cert
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to parse certificate: " + err.Error())
		// {{end}}
		return err
	}

	// Basically we only care if the certificate was signed by our authority
	// Go selects sensible defaults for time and EKU, basically we're only
	// skipping the hostname check, I think?
	options := x509.VerifyOptions{
		Roots: roots,
	}
	if _, err := cert.Verify(options); err != nil {
		// {{if .Config.Debug}}
		log.Printf("Failed to verify certificate: " + err.Error())
		// {{end}}
		return err
	}

	return nil
}
