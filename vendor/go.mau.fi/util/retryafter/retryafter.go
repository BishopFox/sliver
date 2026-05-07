// Copyright (c) 2021 Dillon Dixon
// Copyright (c) 2023 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package retryafter contains a utility function for parsing the Retry-After HTTP header.
package retryafter

import (
	"net/http"
	"strconv"
	"time"
)

var now = time.Now

// Parse parses the backoff time specified in the Retry-After header if present.
// See https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Retry-After.
//
// The second parameter is the fallback duration to use if the header is not present or invalid.
//
// Example:
//
//	time.Sleep(retryafter.Parse(resp.Header.Get("Retry-After"), 5*time.Second))
func Parse(retryAfter string, fallback time.Duration) time.Duration {
	if retryAfter == "" {
		return fallback
	} else if t, err := time.Parse(http.TimeFormat, retryAfter); err == nil {
		return t.Sub(now())
	} else if seconds, err := strconv.Atoi(retryAfter); err == nil {
		return time.Duration(seconds) * time.Second
	}

	return fallback
}

// Should returns true if the given status code indicates that the request should be retried.
//
//	if retryafter.Should(resp.StatusCode, true) {
//		time.Sleep(retryafter.Parse(resp.Header.Get("Retry-After"), 5*time.Second))
//	}
func Should(statusCode int, retryOnRateLimit bool) bool {
	switch statusCode {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	case http.StatusTooManyRequests:
		return retryOnRateLimit
	default:
		return false
	}
}
