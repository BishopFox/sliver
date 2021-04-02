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
	"github.com/stretchr/testify/assert"
	"net/url"
	"testing"
)

var dataProviderWindowsParseLpszProxy = []struct {
	lpszProxy string
	expect    string
}{
	{"1.2.3.4:8443", "1.2.3.4:8443"},
	{"1.2.3.4:8443;4.5.6.7:8999", "4.5.6.7:8999"},
	{"1.2.3.4:8443;http=4.5.6.7:8999", "1.2.3.4:8443"},
	{"http=1.2.3.4:8443;https=4.5.6.7:8999", "4.5.6.7:8999"},
	{"", ""},
	{"  ", "  "},
}

func TestProviderWindows_ParseLpszProxy(t *testing.T) {
	for _, tt := range dataProviderWindowsParseLpszProxy {
		t.Run(tt.lpszProxy, func(t *testing.T) {
			a := assert.New(t)
			p := newWindowsTestProvider()
			a.Equal(tt.expect, p.parseLpszProxy("https", tt.lpszProxy))
		})
	}
}

var dataProviderWindowsIsLpszProxyBypass = []struct {
	targetUrl   *url.URL
	proxyBypass string
	expect      bool
}{
	// nil URL will be Host:*
	{nil, "someHost.com", false},
	// Invalid host will be Host:*
	{&url.URL{Host: "test:test"}, "someHost.com", false},
	// Empty proxyBypass values
	{&url.URL{Host: "test:8080"}, "  ", false},
	{&url.URL{Host: "test:8080"}, ";", false},
	// Matched
	{&url.URL{Host: "test.endpoint.rapid7.com"}, "rapid7.com", true},
	// Matched - Sub Domain
	{&url.URL{Host: "test.endpoint.rapid7.com:443"}, ".rapid7.com", true},
	// Matched - Wildcard Prefix
	{&url.URL{Host: "test.endpoint.rapid7.com:443"}, "*.rapid7.com", true},
	// Matched - Multiple wildcards
	{&url.URL{Host: "test.endpoint.rapid7.com:443"}, "test.*.*.com", true},
	// Matched - Second value
	{&url.URL{Host: "test.endpoint.rapid7.com"}, "someHost;rapid7.com", true},
	// Matched - Just wildcard
	{&url.URL{Host: "test.endpoint.rapid7.com"}, "*", true},
	// Matched - Wildcard second
	{&url.URL{Host: "test.endpoint.rapid7.com"}, ";*", true},
	// Exact match
	{&url.URL{Host: "test.endpoint.rapid7.com"}, "test.endpoint.rapid7.com", true},
	// Matched - Local Host
	{&url.URL{Host: "localhost"}, "<local>", true},
	// Matched - Local Host second
	{&url.URL{Host: "localhost"}, "someHost;<local>", true},
	// Matched - Local IPv4
	{&url.URL{Host: "[::1]"}, "<local>", true},
	// Matched - Local IPv6
	{&url.URL{Host: "127.0.0.1"}, "<local>", true},
	// Not Matched
	{&url.URL{Host: "test.endpoint.rapid7.com"}, "someHost", false},
	// Not Matched - Not local
	{&url.URL{Host: "test.endpoint.rapid7.com"}, "<local>", false},
}

func TestProviderWindows_IsLpszProxyBypass(t *testing.T) {
	for _, tt := range dataProviderWindowsIsLpszProxyBypass {
		var tName string
		if tt.targetUrl == nil {
			tName = "nil"
		} else {
			tName = tt.targetUrl.String()
		}
		tName = tName + " " + tt.proxyBypass
		t.Run(tName, func(t *testing.T) {
			a := assert.New(t)
			p := newWindowsTestProvider()
			a.Equal(tt.expect, p.isLpszProxyBypass(tt.targetUrl, tt.proxyBypass))
		})
	}
}

func newWindowsTestProvider() *providerWindows {
	c := new(providerWindows)
	c.init("")
	return c
}
