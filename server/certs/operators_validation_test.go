package certs

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
)

func TestValidateOperatorClientCertificateAdversarial(t *testing.T) {
	SetupCAs()

	t.Run("accepts stored operator client leaf", func(t *testing.T) {
		operatorName := uniqueOperatorCertificateName(t, "stored")
		certPEM, _, err := OperatorClientGenerateCertificate(operatorName)
		if err != nil {
			t.Fatalf("generate operator client certificate: %v", err)
		}

		err = ValidateOperatorClientCertificate([]*x509.Certificate{mustParseCertificate(t, certPEM)})
		if err != nil {
			t.Fatalf("expected stored operator client certificate to be accepted, got %v", err)
		}
	})

	t.Run("rejects valid stored certificate hidden behind an unstored first certificate", func(t *testing.T) {
		operatorName := uniqueOperatorCertificateName(t, "hidden")
		storedCertPEM, _, err := OperatorClientGenerateCertificate(operatorName)
		if err != nil {
			t.Fatalf("generate stored operator certificate: %v", err)
		}
		rogueCertPEM, _ := GenerateECCCertificate(OperatorCA, operatorName, false, true, true)

		err = ValidateOperatorClientCertificate([]*x509.Certificate{
			mustParseCertificate(t, rogueCertPEM),
			mustParseCertificate(t, storedCertPEM),
		})
		if !errors.Is(err, ErrOperatorClientCertificateNotFound) {
			t.Fatalf("expected first unstored certificate to be rejected, got %v", err)
		}
	})

	t.Run("rejects same-common-name different-leaf certificate", func(t *testing.T) {
		operatorName := uniqueOperatorCertificateName(t, "same-cn")
		_, _, err := OperatorClientGenerateCertificate(operatorName)
		if err != nil {
			t.Fatalf("generate stored operator certificate: %v", err)
		}
		rogueCertPEM, _ := GenerateECCCertificate(OperatorCA, operatorName, false, true, true)

		err = ValidateOperatorClientCertificate([]*x509.Certificate{mustParseCertificate(t, rogueCertPEM)})
		if !errors.Is(err, ErrOperatorClientCertificateNotFound) {
			t.Fatalf("expected rotated or forged same-CN leaf to be rejected, got %v", err)
		}
	})

	t.Run("rejects stored operator server certificate", func(t *testing.T) {
		hostName := uniqueOperatorCertificateName(t, "server")
		serverCertPEM, _, err := OperatorServerGenerateCertificate(hostName)
		if err != nil {
			t.Fatalf("generate operator server certificate: %v", err)
		}

		err = ValidateOperatorClientCertificate([]*x509.Certificate{mustParseCertificate(t, serverCertPEM)})
		if !errors.Is(err, ErrInvalidOperatorClientCertificate) {
			t.Fatalf("expected stored operator server certificate to be rejected, got %v", err)
		}
	})

	t.Run("rejects operator CA certificate even if inserted into the certificates table", func(t *testing.T) {
		caCertPEM, caKeyPEM, err := GetCertificateAuthorityPEM(OperatorCA)
		if err != nil {
			t.Fatalf("get operator CA: %v", err)
		}
		caCert := mustParseCertificate(t, caCertPEM)

		record := &models.Certificate{
			CommonName:     fmt.Sprintf("%s.%s", clientNamespace, caCert.Subject.CommonName),
			CAType:         OperatorCA,
			KeyType:        ECCKey,
			CertificatePEM: string(caCertPEM),
			PrivateKeyPEM:  string(caKeyPEM),
		}
		if err := db.Session().Create(record).Error; err != nil {
			t.Fatalf("insert CA cert into certificates table: %v", err)
		}

		err = ValidateOperatorClientCertificate([]*x509.Certificate{caCert})
		if !errors.Is(err, ErrInvalidOperatorClientCertificate) {
			t.Fatalf("expected CA certificate to be rejected, got %v", err)
		}
	})
}

func mustParseCertificate(t *testing.T, certPEM []byte) *x509.Certificate {
	t.Helper()

	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatal("failed to decode certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse x509 certificate: %v", err)
	}
	return cert
}

func uniqueOperatorCertificateName(t *testing.T, prefix string) string {
	t.Helper()
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
