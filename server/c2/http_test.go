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

	implantEncoders "github.com/bishopfox/sliver/implant/sliver/encoders"
	implantTransports "github.com/bishopfox/sliver/implant/sliver/transports"
	"github.com/bishopfox/sliver/server/certs"
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
	handler := http.HandlerFunc(server.rsaKeyHandler)

	// Missing parameters
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test/foo.txt", nil)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %d want %d", status, http.StatusNotFound)
	}

	// Invalid OTP code
	rr = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", fmt.Sprintf("/test/foo.txt?aa=%d", insecureRand.Intn(99999999)), nil)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %d want %d", status, http.StatusNotFound)
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
