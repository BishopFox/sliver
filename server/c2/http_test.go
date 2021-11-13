package c2

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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	insecureRand "math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	implantCrypto "github.com/bishopfox/sliver/implant/sliver/cryptography"
	implantEncoders "github.com/bishopfox/sliver/implant/sliver/encoders"
	implantTransports "github.com/bishopfox/sliver/implant/sliver/transports/httpclient"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/configs"
	"github.com/bishopfox/sliver/server/cryptography"
	"github.com/bishopfox/sliver/server/db"
	"github.com/bishopfox/sliver/server/db/models"
	"google.golang.org/protobuf/proto"
)

var (
	serverECCKeyPair  *cryptography.ECCKeyPair
	implantECCKeyPair *cryptography.ECCKeyPair
	totpSecret        string
)

func TestMain(m *testing.M) {
	implantConfig := setup()
	code := m.Run()
	cleanup(implantConfig)
	os.Exit(code)
}

func setup() *models.ImplantConfig {
	var err error
	certs.SetupCAs()
	serverECCKeyPair = cryptography.ECCServerKeyPair()
	implantECCKeyPair, err = cryptography.RandomECCKeyPair()
	if err != nil {
		panic(err)
	}
	totpSecret, err := cryptography.TOTPServerSecret()
	if err != nil {
		panic(err)
	}
	implantCrypto.SetSecrets(
		implantECCKeyPair.PublicBase64(),
		implantECCKeyPair.PrivateBase64(),
		"",
		serverECCKeyPair.PublicBase64(),
		totpSecret,
	)
	digest := sha256.Sum256(implantECCKeyPair.Public[:])
	implantConfig := &models.ImplantConfig{
		ECCPublicKey:       implantECCKeyPair.PublicBase64(),
		ECCPrivateKey:      implantECCKeyPair.PrivateBase64(),
		ECCPublicKeyDigest: hex.EncodeToString(digest[:]),
		ECCServerPublicKey: serverECCKeyPair.PublicBase64(),
	}
	err = db.Session().Create(implantConfig).Error
	if err != nil {
		panic(err)
	}
	return implantConfig
}

func cleanup(implantConfig *models.ImplantConfig) {
	db.Session().Delete(implantConfig)
}

func TestStartSessionHandler(t *testing.T) {
	server, err := StartHTTPSListener(&HTTPServerConfig{
		Addr:       "127.0.0.1:8888",
		Secure:     false,
		EnforceOTP: true,
	})
	if err != nil {
		t.Fatalf("Listener failed to start %s", err)
		return
	}

	c2Config := configs.GetHTTPC2Config()
	client := implantTransports.SliverHTTPClient{}
	baseURL := &url.URL{
		Scheme: "http",
		Host:   "127.0.0.1:8888",
		Path:   fmt.Sprintf("/test/foo.%s", c2Config.ImplantConfig.StartSessionFileExt),
	}
	nonce, encoder := implantEncoders.RandomEncoder()
	testURL := client.NonceQueryArgument(baseURL, nonce)
	testURL = client.OTPQueryArgument(testURL, implantCrypto.GetOTPCode())

	// Generate key exchange request
	sKey := cryptography.RandomKey()
	httpSessionInit := &sliverpb.HTTPSessionInit{Key: sKey[:]}
	data, _ := proto.Marshal(httpSessionInit)
	encryptedSessionInit, err := implantCrypto.ECCEncryptToServer(data)
	if err != nil {
		t.Fatalf("Failed to encrypt session init %s", err)
	}
	payload := encoder.Encode(encryptedSessionInit)
	body := bytes.NewReader(payload)

	validReq := httptest.NewRequest(http.MethodPost, testURL.String(), body)
	t.Logf("[http] req request uri: '%v'", validReq.RequestURI)
	rr := httptest.NewRecorder()
	server.HTTPServer.Handler.ServeHTTP(rr, validReq)
	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %d want %d", status, http.StatusOK)
	}

}

func TestGetOTPFromURL(t *testing.T) {
	client := implantTransports.SliverHTTPClient{}

	for i := 0; i < 100; i++ {
		baseURL := &url.URL{
			Scheme: "http",
			Host:   "127.0.0.1:8888",
			Path:   "/test/foo.txt",
		}
		value := fmt.Sprintf("%d", insecureRand.Intn(99999999))
		testURL := client.OTPQueryArgument(baseURL, value)
		urlValue, err := getOTPFromURL(testURL)
		if err != nil {
			t.Fatal(err)
		}
		if urlValue != value {
			t.Fatalf("Mismatched OTP values %s (%s != %s)", testURL.String(), value, urlValue)
		}
	}
}

func TestGetNonceFromURL(t *testing.T) {
	client := implantTransports.SliverHTTPClient{}
	for i := 0; i < 100; i++ {
		baseURL := &url.URL{
			Scheme: "http",
			Host:   "127.0.0.1:8888",
			Path:   "/test/foo.txt",
		}
		nonce, _ := implantEncoders.RandomEncoder()
		testURL := client.NonceQueryArgument(baseURL, nonce)
		urlNonce, err := getNonceFromURL(testURL)
		if err != nil {
			t.Fatal(err)
		}
		if urlNonce != nonce {
			t.Fatalf("Mismatched encoder nonces %s (%d != %d)", testURL.String(), nonce, urlNonce)
		}
	}
}
