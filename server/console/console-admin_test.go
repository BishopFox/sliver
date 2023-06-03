package console

import (
	"encoding/json"
	"encoding/pem"
	"testing"

	clienttransport "github.com/bishopfox/sliver/client/transport"
	"github.com/bishopfox/sliver/server/certs"
)

func TestRootOnlyVerifyCertificate(t *testing.T) {
	certs.SetupCAs()

	data, err := NewOperatorConfig("zerocool", "localhost", uint16(1337))
	if err != nil {
		t.Fatalf("failed to generate test player profile %s", err)
	}
	config := &ClientConfig{}
	err = json.Unmarshal(data, config)
	if err != nil {
		t.Fatalf("failed to parse client config %s", err)
	}

	_, _, err = certs.OperatorServerGetCertificate("localhost")
	if err == certs.ErrCertDoesNotExist {
		certs.OperatorServerGenerateCertificate("localhost")
	}

	// Test with a valid certificate
	certPEM, _, _ := certs.OperatorServerGetCertificate("localhost")
	block, _ := pem.Decode(certPEM)
	err = clienttransport.RootOnlyVerifyCertificate(config.CACertificate, [][]byte{block.Bytes})
	if err != nil {
		t.Fatalf("root only verify certificate error: %s", err)
	}

	// Test with wrong CA
	wrongCert, _ := certs.GenerateECCCertificate(certs.HTTPSCA, "foobar", false, false)
	block, _ = pem.Decode(wrongCert)
	err = clienttransport.RootOnlyVerifyCertificate(config.CACertificate, [][]byte{block.Bytes})
	if err == nil {
		t.Fatal("root only verify cert verified a certificate with invalid ca!")
	}

}
