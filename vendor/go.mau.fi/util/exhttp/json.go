// Copyright (c) 2024 Sumner Evans
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package exhttp

import (
	"encoding/json"
	"net/http"
)

var AutoAllowCORS = true

func WriteJSONResponse(w http.ResponseWriter, httpStatusCode int, jsonData any) {
	if AutoAllowCORS {
		AddCORSHeaders(w)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusCode)
	_ = json.NewEncoder(w).Encode(jsonData)
}

func WriteJSONData(w http.ResponseWriter, httpStatusCode int, data []byte) {
	if AutoAllowCORS {
		AddCORSHeaders(w)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusCode)
	_, _ = w.Write(data)
}

func WriteEmptyJSONResponse(w http.ResponseWriter, httpStatusCode int) {
	WriteJSONData(w, httpStatusCode, []byte("{}"))
}
