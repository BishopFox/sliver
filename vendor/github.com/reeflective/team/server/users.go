package server

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
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/reeflective/team/client"
	"github.com/reeflective/team/internal/certs"
	"github.com/reeflective/team/internal/db"
)

var namePattern = regexp.MustCompile("^[a-zA-Z0-9_-]*$") // Only allow alphanumeric chars

// UserCreate creates a new teamserver user, with all cryptographic material and server remote
// endpoints needed by this user to connect to us.
//
// Certificate files and the API authentication token are saved into the teamserver database,
// conformingly to its configured backend/filesystem (can be in-memory or on filesystem).
func (ts *Server) UserCreate(name string, lhost string, lport uint16) (*client.Config, error) {
	if err := ts.initDatabase(); err != nil {
		return nil, ts.errorf("%w: %w", ErrDatabase, err)
	}

	if !namePattern.MatchString(name) {
		return nil, ts.errorf("%w: invalid user name (alphanumerics only)", ErrUserConfig)
	}

	if name == "" {
		return nil, ts.errorf("%w: user name required ", ErrUserConfig)
	}

	if lhost == "" {
		return nil, ts.errorf("%w: invalid team server host (empty)", ErrUserConfig)
	}

	if lport == blankPort {
		lport = uint16(ts.opts.config.DaemonMode.Port)
	}

	rawToken, err := ts.newUserToken()
	if err != nil {
		return nil, ts.errorf("%w: %w", ErrUserConfig, err)
	}

	digest := sha256.Sum256([]byte(rawToken))
	dbuser := &db.User{
		Name:  name,
		Token: hex.EncodeToString(digest[:]),
	}

	err = ts.dbSession().Save(dbuser).Error
	if err != nil {
		return nil, ts.errorf("%w: %w", ErrDatabase, err)
	}

	publicKey, privateKey, err := ts.certs.UserClientGenerateCertificate(name)
	if err != nil {
		return nil, ts.errorf("%w: failed to generate certificate %w", ErrCertificate, err)
	}

	caCertPEM, _, _ := ts.certs.GetUsersCAPEM()
	config := client.Config{
		User:          name,
		Token:         rawToken,
		Host:          lhost,
		Port:          int(lport),
		CACertificate: string(caCertPEM),
		PrivateKey:    string(privateKey),
		Certificate:   string(publicKey),
	}

	return &config, nil
}

// UserDelete deletes a user and its cryptographic materials from
// the teamserver database, clearing the API auth tokens cache.
//
// WARN: This function has two very precise effects/consequences:
//  1. The server-side Mutual TLS configuration obtained with server.GetUserTLSConfig()
//     will refuse all connections using the deleted user TLS credentials, returning
//     an authentication failure.
//  2. The server.AuthenticateUser(token) method will always return an ErrUnauthenticated
//     error from the call, because the delete user is not in the database anymore.
//
// Thus, it is up to the users of this library to use the builting teamserver TLS
// configurations in their teamserver listener / teamclient dialer implementations.
//
// Certificate files, API authentication token are deleted from the teamserver database,
// conformingly to its configured backend/filesystem (can be in-memory or on filesystem).
func (ts *Server) UserDelete(name string) error {
	if err := ts.initDatabase(); err != nil {
		return ts.errorf("%w: %w", ErrDatabase, err)
	}

	err := ts.dbSession().Where(&db.User{
		Name: name,
	}).Delete(&db.User{}).Error
	if err != nil {
		return err
	}

	// Clear the token cache so that all requests from
	// connected clients of this user are now refused.
	ts.userTokens = &sync.Map{}

	return ts.certs.UserClientRemoveCertificate(name)
}

// UserAuthenticate accepts a raw 128-bits long API Authentication token belonging to the
// user of a connected/connecting teamclient. The token is hashed and checked against the
// teamserver users database for the matching user.
// This function shall alternatively return:
//   - The name of the authenticated user, true for authenticated and no error.
//   - No name, false for authenticated, and an ErrUnauthenticated error.
//   - No name, false for authenticated, and a database error, if was ignited now.
//
// This call updates the last time the user has been seen by the server.
func (ts *Server) UserAuthenticate(rawToken string) (name string, authorized bool, err error) {
	if err := ts.initDatabase(); err != nil {
		return "", false, ts.errorf("%w: %w", ErrDatabase, err)
	}

	log := ts.NamedLogger("server", "auth")
	log.Debugf("Authorization-checking user token ...")

	// Check auth cache
	digest := sha256.Sum256([]byte(rawToken))
	token := hex.EncodeToString(digest[:])

	if name, ok := ts.userTokens.Load(token); ok {
		log.Debugf("Token in cache!")
		ts.updateLastSeen(name.(string))
		return name.(string), true, nil
	}

	user, err := ts.userByToken(token)
	if err != nil || user == nil {
		return "", false, ts.errorf("%w: %w", ErrUnauthenticated, err)
	}

	ts.updateLastSeen(user.Name)

	log.Debugf("Valid user token for %s", user.Name)
	ts.userTokens.Store(token, user.Name)

	return user.Name, true, nil
}

// UsersTLSConfig returns a server-side Mutual TLS configuration struct, ready to run.
// The configuration performs all and every verifications that the teamserver should do,
// and peer TLS clients (teamclient.Config) are not allowed to choose any TLS parameters.
//
// This should be used by team/server.Listeners at the net.Listener/net.Conn level.
// As for all errors of the teamserver API, any error returned here is defered-logged.
func (ts *Server) UsersTLSConfig() (*tls.Config, error) {
	log := ts.NamedLogger("certs", "mtls")

	if err := ts.initDatabase(); err != nil {
		return nil, ts.errorf("%w: %w", ErrDatabase, err)
	}

	caCertPtr, _, err := ts.certs.GetUsersCA()
	if err != nil {
		return nil, ts.errorWith(log, "%w: failed to get users certificate authority: %w", ErrCertificate, err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCertPtr)

	_, _, err = ts.certs.UserServerGetCertificate()
	if errors.Is(err, certs.ErrCertDoesNotExist) {
		if _, _, err := ts.certs.UserServerGenerateCertificate(); err != nil {
			return nil, ts.errorWith(log, err.Error())
		}
	}

	certPEM, keyPEM, err := ts.certs.UserServerGetCertificate()
	if err != nil {
		return nil, ts.errorWith(log, "%w: failed to generated or fetch user certificate: %w", ErrCertificate, err)
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, ts.errorWith(log, "%w: failed to load server certificate: %w", ErrCertificate, err)
	}

	tlsConfig := &tls.Config{
		RootCAs:      caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}

	if keyLogger := ts.certs.OpenTLSKeyLogFile(); keyLogger != nil {
		tlsConfig.KeyLogWriter = ts.certs.OpenTLSKeyLogFile()
	}

	return tlsConfig, nil
}

// UsersGetCA returns the bytes of a PEM-encoded certificate authority,
// which contains certificates of all users of this teamserver.
func (ts *Server) UsersGetCA() ([]byte, []byte, error) {
	if err := ts.initDatabase(); err != nil {
		return nil, nil, ts.errorf("%w: %w", ErrDatabase, err)
	}

	return ts.certs.GetUsersCAPEM()
}

// UsersSaveCA accepts the public and private parts of a Certificate
// Authority containing one or more users to add to the teamserver.
func (ts *Server) UsersSaveCA(cert, key []byte) {
	if err := ts.initDatabase(); err != nil {
		return
	}

	ts.certs.SaveUsersCA(cert, key)
}

// newUserToken - Generate a new user authentication token.
func (ts *Server) newUserToken() (string, error) {
	buf := make([]byte, tokenLength)

	n, err := rand.Read(buf)
	if err != nil || n != len(buf) {
		return "", fmt.Errorf("%w: %w", ErrSecureRandFailed, err)
	} else if n != len(buf) {
		return "", ErrSecureRandFailed
	}

	return hex.EncodeToString(buf), nil
}

// userByToken - Select a teamserver user by token value.
func (ts *Server) userByToken(value string) (*db.User, error) {
	if len(value) < 1 {
		return nil, db.ErrRecordNotFound
	}

	user := &db.User{}
	err := ts.dbSession().Where(&db.User{
		Token: value,
	}).First(user).Error

	return user, err
}

func (ts *Server) updateLastSeen(name string) {
	lastSeen := time.Now().Round(1 * time.Second)
	ts.dbSession().Model(&db.User{}).Where("name", name).Update("LastSeen", lastSeen)
}

// func TestRootOnlyVerifyCertificate(t *testing.T) {
// 	certs.SetupCAs()
//
// 	data, err := NewOperatorConfig("zerocool", "localhost", uint16(1337))
// 	if err != nil {
// 		t.Fatalf("failed to generate test player profile %s", err)
// 	}
// 	config := &ClientConfig{}
// 	err = json.Unmarshal(data, config)
// 	if err != nil {
// 		t.Fatalf("failed to parse client config %s", err)
// 	}
//
// 	_, _, err = certs.OperatorServerGetCertificate("localhost")
// 	if err == certs.ErrCertDoesNotExist {
// 		certs.OperatorServerGenerateCertificate("localhost")
// 	}
//
// 	// Test with a valid certificate
// 	certPEM, _, _ := certs.OperatorServerGetCertificate("localhost")
// 	block, _ := pem.Decode(certPEM)
// 	err = clienttransport.RootOnlyVerifyCertificate(config.CACertificate, [][]byte{block.Bytes})
// 	if err != nil {
// 		t.Fatalf("root only verify certificate error: %s", err)
// 	}
//
// 	// Test with wrong CA
// 	wrongCert, _ := certs.GenerateECCCertificate(certs.HTTPSCA, "foobar", false, false)
// 	block, _ = pem.Decode(wrongCert)
// 	err = clienttransport.RootOnlyVerifyCertificate(config.CACertificate, [][]byte{block.Bytes})
// 	if err == nil {
// 		t.Fatal("root only verify cert verified a certificate with invalid ca!")
// 	}
//
// }

// func TestOperatorGenerateCertificate(t *testing.T) {
// 	GenerateCertificateAuthority(OperatorCA, "")
// 	cert1, key1, err := OperatorClientGenerateCertificate("test3")
// 	if err != nil {
// 		t.Errorf("Failed to store ecc certificate %v", err)
// 		return
// 	}
//
// 	cert2, key2, err := OperatorClientGetCertificate("test3")
// 	if err != nil {
// 		t.Errorf("Failed to get ecc certificate %v", err)
// 		return
// 	}
//
// 	if !bytes.Equal(cert1, cert2) || !bytes.Equal(key1, key2) {
// 		t.Errorf("Stored ecc cert/key does match generated cert/key: %v != %v", cert1, cert2)
// 		return
// 	}
// }
