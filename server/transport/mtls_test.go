package transport

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/bishopfox/sliver/server/certs"
)

func TestOperatorClientCertificateValidation(t *testing.T) {
	certs.SetupCAs()

	t.Run("accepts stored operator certificate", func(t *testing.T) {
		operatorName := uniqueOperatorName(t, "stored")
		certPEM, keyPEM, err := certs.OperatorClientGenerateCertificate(operatorName)
		if err != nil {
			t.Fatalf("generate operator certificate: %v", err)
		}

		err = performMutualTLSHandshake(getOperatorServerTLSConfig("multiplayer"), newClientTLSConfig(t, certPEM, keyPEM))
		if err != nil {
			t.Fatalf("expected stored operator certificate to be accepted, got %v", err)
		}
	})

	t.Run("rejects deleted operator certificate", func(t *testing.T) {
		operatorName := uniqueOperatorName(t, "deleted")
		certPEM, keyPEM, err := certs.OperatorClientGenerateCertificate(operatorName)
		if err != nil {
			t.Fatalf("generate operator certificate: %v", err)
		}
		if err := certs.OperatorClientRemoveCertificate(operatorName); err != nil {
			t.Fatalf("remove operator certificate: %v", err)
		}

		err = performMutualTLSHandshake(getOperatorServerTLSConfig("multiplayer"), newClientTLSConfig(t, certPEM, keyPEM))
		if err == nil {
			t.Fatal("expected deleted operator certificate to be rejected")
		}
		if !strings.Contains(err.Error(), certs.ErrOperatorClientCertificateNotFound.Error()) {
			t.Fatalf("expected database rejection error, got %v", err)
		}
	})

	t.Run("rejects rotated-out certificate but accepts current one", func(t *testing.T) {
		operatorName := uniqueOperatorName(t, "rotated")
		oldCertPEM, oldKeyPEM, err := certs.OperatorClientGenerateCertificate(operatorName)
		if err != nil {
			t.Fatalf("generate original operator certificate: %v", err)
		}
		if err := certs.OperatorClientRemoveCertificate(operatorName); err != nil {
			t.Fatalf("remove original operator certificate: %v", err)
		}
		newCertPEM, newKeyPEM, err := certs.OperatorClientGenerateCertificate(operatorName)
		if err != nil {
			t.Fatalf("generate replacement operator certificate: %v", err)
		}

		err = performMutualTLSHandshake(getOperatorServerTLSConfig("multiplayer"), newClientTLSConfig(t, oldCertPEM, oldKeyPEM))
		if err == nil {
			t.Fatal("expected rotated-out operator certificate to be rejected")
		}
		if !strings.Contains(err.Error(), certs.ErrOperatorClientCertificateNotFound.Error()) {
			t.Fatalf("expected rotated certificate to fail database validation, got %v", err)
		}

		err = performMutualTLSHandshake(getOperatorServerTLSConfig("multiplayer"), newClientTLSConfig(t, newCertPEM, newKeyPEM))
		if err != nil {
			t.Fatalf("expected replacement operator certificate to be accepted, got %v", err)
		}
	})

	t.Run("rejects certificate from another authority", func(t *testing.T) {
		certPEM, keyPEM := certs.GenerateECCCertificate(certs.HTTPSCA, uniqueOperatorName(t, "wrong-ca"), false, true, false)

		err := performMutualTLSHandshake(getOperatorServerTLSConfig("multiplayer"), newClientTLSConfig(t, certPEM, keyPEM))
		if err == nil {
			t.Fatal("expected certificate from another authority to be rejected")
		}
		if strings.Contains(err.Error(), certs.ErrOperatorClientCertificateNotFound.Error()) {
			t.Fatalf("expected trust-chain rejection before database lookup, got %v", err)
		}
	})
}

func newClientTLSConfig(t *testing.T, certPEM []byte, keyPEM []byte) *tls.Config {
	t.Helper()

	clientCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("parse client certificate: %v", err)
	}

	return &tls.Config{
		Certificates:       []tls.Certificate{clientCert},
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS13,
	}
}

func performMutualTLSHandshake(serverConfig *tls.Config, clientConfig *tls.Config) error {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	deadline := time.Now().Add(2 * time.Second)
	_ = serverConn.SetDeadline(deadline)
	_ = clientConn.SetDeadline(deadline)

	serverTLS := tls.Server(serverConn, serverConfig)
	clientTLS := tls.Client(clientConn, clientConfig)
	defer serverTLS.Close()
	defer clientTLS.Close()

	type handshakeResult struct {
		side string
		err  error
	}

	results := make(chan handshakeResult, 2)
	go func() {
		results <- handshakeResult{side: "server", err: serverTLS.Handshake()}
	}()
	go func() {
		results <- handshakeResult{side: "client", err: clientTLS.Handshake()}
	}()

	var errs []string
	for i := 0; i < 2; i++ {
		result := <-results
		if result.err == nil {
			continue
		}
		if errors.Is(result.err, net.ErrClosed) {
			continue
		}
		errs = append(errs, fmt.Sprintf("%s: %v", result.side, result.err))
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "; "))
}

func uniqueOperatorName(t *testing.T, prefix string) string {
	t.Helper()
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
