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
	"fmt"
	insecureRand "math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	implantEncoders "github.com/bishopfox/sliver/implant/sliver/encoders"
	implantTransports "github.com/bishopfox/sliver/implant/sliver/transports/httpclient"
	"github.com/bishopfox/sliver/server/certs"
	"github.com/bishopfox/sliver/server/cryptography"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

func TestRsaKeyHandler(t *testing.T) {
	certs.SetupCAs()
	server, err := StartHTTPSListener(&HTTPServerConfig{
		Addr:       "127.0.0.1:8888",
		EnforceOTP: true,
	})
	if err != nil {
		t.Errorf("Listener failed to start %s", err)
		return
	}

	router := server.router()

	// Missing parameters
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test/foo.txt", nil)
	router.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %d want %d", status, http.StatusNotFound)
	}

	// Invalid OTP code
	for i := 0; i < 100; i++ {
		rr = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", fmt.Sprintf("/test/foo.txt?aa=%d", insecureRand.Intn(99999999)), nil)
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %d want %d", status, http.StatusNotFound)
		}
	}

	// Valid OTP code
	client := implantTransports.SliverHTTPClient{}
	baseURL := &url.URL{
		Scheme: "http",
		Host:   "127.0.0.1:8888",
		Path:   "/test/foo.txt",
	}
	nonce, _ := implantEncoders.RandomEncoder()
	testURL := client.NonceQueryArgument(baseURL, nonce)

	now := time.Now().UTC()
	opts := totp.ValidateOpts{
		Digits:    8,
		Algorithm: otp.AlgorithmSHA256,
		Period:    uint(30),
		Skew:      uint(1),
	}
	secret, _ := cryptography.TOTPServerSecret()
	code, _ := totp.GenerateCodeCustom(secret, now, opts)

	testURL = client.OTPQueryArgument(baseURL, code)
	rr = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", testURL.String(), nil)
	router.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %d want %d", status, http.StatusOK)
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
			t.Error(err)
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
			t.Error(err)
		}
		if urlNonce != nonce {
			t.Fatalf("Mismatched encoder nonces %s (%d != %d)", testURL.String(), nonce, urlNonce)
		}
	}
}
