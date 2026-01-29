// Copyright (c) 2024 Sumner Evans
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package exhttp

import "net/http"

func AddCORSHeaders(w http.ResponseWriter) {
	// Recommended CORS headers can be found in https://spec.matrix.org/v1.3/client-server-api/#web-browser-clients
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "X-Requested-With, Content-Type, Authorization")
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'; script-src 'none'; plugin-types application/pdf; style-src 'unsafe-inline'; object-src 'self';")
	// Allow browsers to cache above for 1 day
	w.Header().Set("Access-Control-Max-Age", "86400")
}

// CORSMiddleware adds CORS headers to the response and handles OPTIONS
// requests by returning 200 OK immediately.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		AddCORSHeaders(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
