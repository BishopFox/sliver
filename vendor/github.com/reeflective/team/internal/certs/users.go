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
	"encoding/pem"
	"fmt"

	"github.com/reeflective/team/internal/db"
)

const (
	// userCA - Directory containing user certificates.
	userCA = "user"

	clientNamespace  = "client"    // User clients
	serverNamespace  = "server"    // User servers
	userCertHostname = "teamusers" // Hostname used on certificate
)

// UserClientGenerateCertificate - Generate a certificate signed with a given CA.
func (c *Manager) UserClientGenerateCertificate(user string) ([]byte, []byte, error) {
	cert, key := c.GenerateECCCertificate(userCA, user, false, true)
	err := c.saveCertificate(userCA, ECCKey, fmt.Sprintf("%s.%s", clientNamespace, user), cert, key)

	return cert, key, err
}

// UserClientGetCertificate - Helper function to fetch a client cert.
func (c *Manager) UserClientGetCertificate(user string) ([]byte, []byte, error) {
	return c.GetECCCertificate(userCA, fmt.Sprintf("%s.%s", clientNamespace, user))
}

// UserClientRemoveCertificate - Helper function to remove a client cert.
func (c *Manager) UserClientRemoveCertificate(user string) error {
	return c.RemoveCertificate(userCA, ECCKey, fmt.Sprintf("%s.%s", clientNamespace, user))
}

// UserServerGetCertificate - Helper function to fetch a server cert.
func (c *Manager) UserServerGetCertificate() ([]byte, []byte, error) {
	return c.GetECCCertificate(userCA, fmt.Sprintf("%s.%s", serverNamespace, userCertHostname))
}

// UserServerGenerateCertificate - Generate a certificate signed with a given CA.
func (c *Manager) UserServerGenerateCertificate() ([]byte, []byte, error) {
	cert, key := c.GenerateECCCertificate(userCA, userCertHostname, false, false)
	err := c.saveCertificate(userCA, ECCKey, fmt.Sprintf("%s.%s", serverNamespace, userCertHostname), cert, key)

	return cert, key, err
}

// UserClientListCertificates - Get all client certificates.
func (c *Manager) UserClientListCertificates() []*x509.Certificate {
	userCerts := []*db.Certificate{}

	result := c.db().Where(&db.Certificate{CAType: userCA}).Find(&userCerts)
	if result.Error != nil {
		c.log.Error(result.Error)
		return []*x509.Certificate{}
	}

	c.log.Infof("Found %d user certs ...", len(userCerts))

	certs := []*x509.Certificate{}

	for _, user := range userCerts {
		block, _ := pem.Decode([]byte(user.CertificatePEM))
		if block == nil {
			c.log.Warn("failed to parse certificate PEM")
			continue
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			c.log.Warnf("failed to parse x.509 certificate %v", err)
			continue
		}

		certs = append(certs, cert)
	}

	return certs
}
