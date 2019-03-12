// Copyright 2018, Rapid7, Inc.
// License: BSD-3-clause
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
// * Redistributions of source code must retain the above copyright notice,
// this list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright
// notice, this list of conditions and the following disclaimer in the
// documentation and/or other materials provided with the distribution.
// * Neither the name of the copyright holder nor the names of its contributors
// may be used to endorse or promote products derived from this software
// without specific prior written permission.
package proxy

import (
	"errors"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

var dataNewProxy = []struct {
	u         *url.URL
	expectP   Proxy
	expectErr error
}{
	// All input
	{
		&url.URL{Scheme: "https", Host: "testProxy:8999", User: url.UserPassword("user1", "password1")},
		&proxy{protocol: "https", host: "testProxy", port: 8999, user: url.UserPassword("user1", "password1"), src: "Test"}, nil,
	},
	// No password
	{
		&url.URL{Scheme: "https", Host: "testProxy:8999", User: url.User("user1")},
		&proxy{protocol: "https", host: "testProxy", port: 8999, user: url.User("user1"), src: "Test"}, nil,
	},
	// No user
	{
		&url.URL{Scheme: "https", Host: "testProxy:8999"},
		&proxy{protocol: "https", host: "testProxy", port: 8999, user: nil, src: "Test"}, nil,
	},
	// No port
	{
		&url.URL{Scheme: "https", Host: "testProxy"},
		&proxy{protocol: "https", host: "testProxy", port: 8443, user: nil, src: "Test"}, nil,
	},
	// No port - Unknown protocol
	{
		&url.URL{Scheme: "gopher", Host: "testProxy"},
		&proxy{protocol: "gopher", host: "testProxy", port: 8443, user: nil, src: "Test"}, nil,
	},
	// 0 port
	{
		&url.URL{Scheme: "https", Host: "testProxy:0"},
		&proxy{protocol: "https", host: "testProxy", port: 8443, user: nil, src: "Test"}, nil,
	},
	// No URL protocol
	{
		&url.URL{Host: "testProxy"},
		&proxy{protocol: "", host: "testProxy", port: 8443, src: "Test"}, nil,
	},
	// Uppercase expected protocol
	{
		&url.URL{Scheme: "https", Host: "testProxy"},
		&proxy{protocol: "https", host: "testProxy", port: 8443, user: nil, src: "Test"}, nil,
	},
	// Uppercase and whitespace URL protocol
	{
		&url.URL{Scheme: "  HTTPS  ", Host: "testProxy"},
		&proxy{protocol: "https", host: "testProxy", port: 8443, user: nil, src: "Test"}, nil,
	},
	// No expected protocol
	{
		&url.URL{Scheme: "https", Host: "testProxy"},
		&proxy{protocol: "https", host: "testProxy", port: 8443, user: nil, src: "Test"}, nil,
	},
	// Invalid port
	{
		&url.URL{Scheme: "https", Host: "testProxy:testPort"},
		nil, errors.New("SplitHostPort testProxy:testPort: strconv.ParseUint: parsing \"testPort\": invalid syntax"),
	},
	// Negative port
	{
		&url.URL{Scheme: "https", Host: "testProxy:-1"},
		nil, errors.New("SplitHostPort testProxy:-1: strconv.ParseUint: parsing \"-1\": invalid syntax"),
	},
	// Empty host
	{
		&url.URL{Scheme: "https", Host: ""},
		nil, errors.New("empty host"),
	},
	// Nil URL
	{
		nil,
		nil, errors.New("nil URL"),
	},
}

func TestNewProxy(t *testing.T) {
	for _, tt := range dataNewProxy {
		var tName string
		if tt.u == nil {
			tName = "nil"
		} else {
			tName = tt.u.String()
		}
		t.Run(tName, func(t *testing.T) {
			a := assert.New(t)
			p, err := NewProxy(tt.u, "Test")
			if tt.expectP == nil {
				a.Nil(p)
			} else {
				a.Equal(tt.expectP, p)
			}
			if tt.expectErr == nil {
				a.Nil(err)
			} else {
				if a.NotNil(err) {
					a.Equal(tt.expectErr.Error(), err.Error())
				}
			}
		})
	}
}

var dataUsername = []struct {
	u              *url.Userinfo
	expectUsername string
	expectExists   bool
}{
	{
		url.User("user1"), "user1", true,
	},
	{
		url.UserPassword("user1", "password1"), "user1", true,
	},
	{
		url.User(""), "", true,
	},
	{
		nil, "", false,
	},
}

func TestProxy_Username(t *testing.T) {
	for _, tt := range dataUsername {
		var tName string
		if tt.u == nil {
			tName = "nil"
		} else {
			tName = tt.u.String()
		}
		t.Run(tName, func(t *testing.T) {
			a := assert.New(t)
			username, exists := (&proxy{user: tt.u}).Username()
			a.Equal(tt.expectUsername, username)
			a.Equal(tt.expectExists, exists)
		})
	}
}

var dataPassword = []struct {
	u              *url.Userinfo
	expectPassword string
	expectExists   bool
}{
	{
		url.User("user1"), "", false,
	},
	{
		url.UserPassword("user1", "password1"), "password1", true,
	},
	{
		url.UserPassword("user1", ""), "", true,
	},
	{
		url.User(""), "", false,
	},
	{
		nil, "", false,
	},
}

func TestProxy_Password(t *testing.T) {
	for _, tt := range dataPassword {
		var tName string
		if tt.u == nil {
			tName = "nil"
		} else {
			tName = tt.u.String()
		}
		t.Run(tName, func(t *testing.T) {
			a := assert.New(t)
			username, exists := (&proxy{user: tt.u}).Password()
			a.Equal(tt.expectPassword, username)
			a.Equal(tt.expectExists, exists)
		})
	}
}

var dataURL = []struct {
	p         Proxy
	expectU   *url.URL
	expectStr string
}{
	{
		&proxy{protocol: "https", host: "testProxy", port: 8999, user: url.UserPassword("user1", "password1"), src: "Test"},
		&url.URL{Scheme: "https", Host: "testProxy:8999", User: url.UserPassword("user1", "password1")},
		"https://user1:password1@testProxy:8999",
	},
	{
		&proxy{protocol: "https", host: "testProxy", port: 8999, user: url.User("user1"), src: "Test"},
		&url.URL{Scheme: "https", Host: "testProxy:8999", User: url.User("user1")},
		"https://user1@testProxy:8999",
	},
	{
		&proxy{protocol: "https", host: "testProxy", port: 8999},
		&url.URL{Scheme: "https", Host: "testProxy:8999"},
		"https://testProxy:8999",
	},
	{
		&proxy{port: 0},
		&url.URL{Host: ":0"},
		"//:0",
	},
}

func TestProxy_URL(t *testing.T) {
	for _, tt := range dataURL {
		t.Run(tt.p.String(), func(t *testing.T) {
			a := assert.New(t)
			u := tt.p.URL()
			if a.Equal(tt.expectU, u) {
				a.Equal(tt.expectStr, u.String())
			}
		})
	}
}

var dataString = []struct {
	p      Proxy
	expect string
}{
	{
		&proxy{protocol: "https", host: "testProxy", port: 8999, user: url.UserPassword("user1", "password1"), src: "Test"},
		"https://<username>:<password>@testProxy:8999",
	},
	{
		&proxy{protocol: "https", host: "testProxy", port: 8999, user: url.User("user1"), src: "Test"},
		"https://<username>@testProxy:8999",
	},
	{
		&proxy{protocol: "https", host: "testProxy", port: 8999, user: url.User("user1"), src: "Test"},
		"https://<username>@testProxy:8999",
	},
	{
		&proxy{protocol: "https", host: "testProxy", port: 8999, src: "Test"},
		"https://testProxy:8999",
	},
	{
		&proxy{},
		"://:0",
	},
}

func TestProxy_String(t *testing.T) {
	for _, tt := range dataString {
		t.Run(tt.p.String(), func(t *testing.T) {
			a := assert.New(t)
			a.Equal(tt.expect, tt.p.String())
		})
	}
}

var dataProxyMarshalJSON = []struct {
	p      Proxy
	expect string
}{
	{
		&proxy{protocol: "https", host: "testProxy", port: 8999, user: url.UserPassword("user1", "password1"), src: "Test"},
		"{\"host\":\"testProxy\",\"password\":\"password1\",\"port\":8999,\"protocol\":\"https\",\"src\":\"Test\",\"username\":\"user1\"}",
	},
	{
		&proxy{protocol: "https", host: "testProxy", port: 8999, user: url.User("user1"), src: "Test"},
		"{\"host\":\"testProxy\",\"password\":null,\"port\":8999,\"protocol\":\"https\",\"src\":\"Test\",\"username\":\"user1\"}",
	},
	{
		&proxy{},
		"{\"host\":\"\",\"password\":null,\"port\":0,\"protocol\":\"\",\"src\":\"\",\"username\":null}",
	},
}

func TestProxy_MarshalJSON(t *testing.T) {
	for _, tt := range dataProxyMarshalJSON {
		t.Run(tt.p.String(), func(t *testing.T) {
			a := assert.New(t)
			b, err := tt.p.MarshalJSON()
			if a.Nil(err) {
				a.Equal(tt.expect, string(b[:]))
			}
		})
	}
}
