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
	"crypto/x509"
	"fmt"
	"log"
	"os"

	"github.com/reeflective/team/internal/assets"
)

const (
	// DefaultPort is the default team.Server listening port.
	// Should be 31415, but... go to hell with your endless limits.
	DefaultPort = 31416
)

// OpenTLSKeyLogFile returns an open file to the TLS keys log file,
// if the environment variable SSLKEYLOGFILE is defined.
func (c *Manager) OpenTLSKeyLogFile() *os.File {
	keyFilePath, present := os.LookupEnv("SSLKEYLOGFILE")
	if present {
		keyFile, err := os.OpenFile(keyFilePath, assets.FileWriteOpenMode, assets.FileReadPerm)
		if err != nil {
			c.log.Errorf("Failed to open TLS key file %v", err)
			return nil
		}

		c.log.Warnf("NOTICE: TLS Keys logged to '%s'\n", keyFilePath)

		return keyFile
	}

	return nil
}

// RootOnlyVerifyCertificate - Go doesn't provide a method for only skipping hostname validation so
// we have to disable all of the certificate validation and re-implement everything.
// https://github.com/golang/go/issues/21971
func RootOnlyVerifyCertificate(caCertificate string, rawCerts [][]byte) error {
	roots := x509.NewCertPool()

	ok := roots.AppendCertsFromPEM([]byte(caCertificate))
	if !ok {
		fmt.Errorf("Failed to parse root certificate")
	}

	cert, err := x509.ParseCertificate(rawCerts[0]) // We should only get one cert
	if err != nil {
		log.Printf("Failed to parse certificate: " + err.Error())
		return err
	}

	// Basically we only care if the certificate was signed by our authority
	// Go selects sensible defaults for time and EKU, basically we're only
	// skipping the hostname check, I think?
	options := x509.VerifyOptions{
		Roots: roots,
	}

	if options.Roots == nil {
		return fmt.Errorf("Certificate root is nil")
	}

	if _, err := cert.Verify(options); err != nil {
		return fmt.Errorf("Failed to verify certificate: " + err.Error())
	}

	return nil
}
