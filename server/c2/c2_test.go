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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bishopfox/sliver/server/certs"
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
