// Copyright (c) 2024 Sumner Evans
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package exhttp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
)

type ErrorBodies struct {
	NotFound         json.RawMessage
	MethodNotAllowed json.RawMessage
}

func HandleErrors(gen ErrorBodies) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(&bodyOverrider{
				ResponseWriter:             w,
				statusNotFoundBody:         gen.NotFound,
				statusMethodNotAllowedBody: gen.MethodNotAllowed,
			}, r)
		})
	}
}

type bodyOverrider struct {
	http.ResponseWriter

	code     int
	override bool
	written  bool

	hijacked bool

	statusNotFoundBody         json.RawMessage
	statusMethodNotAllowedBody json.RawMessage
}

var (
	_ http.ResponseWriter = (*bodyOverrider)(nil)
	_ http.Flusher        = (*bodyOverrider)(nil)
	_ http.Hijacker       = (*bodyOverrider)(nil)
)

func (b *bodyOverrider) WriteHeader(code int) {
	if !b.hijacked &&
		b.Header().Get("Content-Type") == "text/plain; charset=utf-8" &&
		(code == http.StatusNotFound || code == http.StatusMethodNotAllowed) {

		b.Header().Set("Content-Type", "application/json")
		b.override = true
	}

	b.code = code
	b.ResponseWriter.WriteHeader(code)
}

func (b *bodyOverrider) Write(body []byte) (n int, err error) {
	if b.override {
		n = len(body)
		if !b.written {
			switch b.code {
			case http.StatusNotFound:
				_, err = b.ResponseWriter.Write(b.statusNotFoundBody)
			case http.StatusMethodNotAllowed:
				_, err = b.ResponseWriter.Write(b.statusMethodNotAllowedBody)
			}
		}
		b.written = true
		return
	}

	return b.ResponseWriter.Write(body)
}

func (b *bodyOverrider) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := b.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("HandleErrors: %T does not implement http.Hijacker", b.ResponseWriter)
	}
	b.hijacked = true
	return hijacker.Hijack()
}

func (b *bodyOverrider) Flush() {
	flusher, ok := b.ResponseWriter.(http.Flusher)
	if ok {
		flusher.Flush()
	}
}
