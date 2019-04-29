package c2

import (
	"net/http"
	"net/http/httptest"
	"sliver/server/certs"
	"testing"
)

func TestRsaKeyHandler(t *testing.T) {

	certs.SetupCAs()

	req, err := http.NewRequest("GET", "/test/foo.txt", nil)
	if err != nil {
		t.Fatal(err)
	}

	server := StartHTTPSListener(&HTTPServerConfig{
		Addr: "127.0.0.1:8888",
	})
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(server.rsaKeyHandler)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

}
