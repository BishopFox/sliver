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
	"github.com/stretchr/testify/assert"
	"net/url"
	"testing"
)

var dataParseURL = []struct {
	rawUrl        string
	defaultScheme string
	expectUrl     *url.URL
	expectError   error
}{
	{"https://test", "", &url.URL{Scheme: "https", Host: "test"}, nil},
	{"https://test:8080", "", &url.URL{Scheme: "https", Host: "test:8080"}, nil},
	{"https://1.2.3.4:8080", "", &url.URL{Scheme: "https", Host: "1.2.3.4:8080"}, nil},
	{"https://1.2.3.4:8080?test123= v1&test456", "", &url.URL{Scheme: "https", Host: "1.2.3.4:8080", RawQuery: "test123= v1&test456"}, nil},
	{"https://1.2.3.4:8080?test123= v1&test456#fragment1", "", &url.URL{Scheme: "https", Host: "1.2.3.4:8080", RawQuery: "test123= v1&test456", Fragment: "fragment1"}, nil},
	{"https://1.2.3.4:8080#fragment1", "", &url.URL{Scheme: "https", Host: "1.2.3.4:8080", Fragment: "fragment1"}, nil},
	{"https://username:password@1.2.3.4:8080", "", &url.URL{Scheme: "https", Host: "1.2.3.4:8080", User: url.UserPassword("username", "password")}, nil},
	{"//test123:8080", "", &url.URL{Scheme: "", Host: "test123:8080"}, nil},
	{"//test123", "", &url.URL{Scheme: "", Host: "test123"}, nil},
	// No scheme
	{"1.2.3.4:8080", "", &url.URL{Host: "1.2.3.4:8080"}, nil},
	// No scheme with default
	{"1.2.3.4:8080", "https", &url.URL{Scheme: "https", Host: "1.2.3.4:8080"}, nil},
	// No scheme with query
	{"1.2.3.4:8080?test123= v1&test456", "", &url.URL{Host: "1.2.3.4:8080", RawQuery: "test123= v1&test456"}, nil},
	// No scheme with query and fragment
	{"1.2.3.4:8080?test123= v1&test456#fragment1", "", &url.URL{Host: "1.2.3.4:8080", RawQuery: "test123= v1&test456", Fragment: "fragment1"}, nil},
	// Whitespace
	{"  https://test  ", "", &url.URL{Scheme: "https", Host: "test"}, nil},
	// Empty
	{"", "", &url.URL{}, nil},
	{"", "https", &url.URL{Scheme: "https"}, nil},
	// Invalid cases
	{"://test:8080", "", nil, errors.New("parse ://test:8080: missing protocol scheme")},
	// TODO These error cases are introduced after Go 1.7
	//{"https://[test:8080", "", nil, errors.New("parse https://[test:8080: missing ']' in host")},
	//{"https://username:1412¶45124@test:8080", "", nil, errors.New("parse https://username:1412¶45124@test:8080: net/url: invalid userinfo")},
}

func TestParseURL(t *testing.T) {
	for _, tt := range dataParseURL {
		t.Run(tt.rawUrl, func(t *testing.T) {
			a := assert.New(t)
			parsedUrl, err := ParseURL(tt.rawUrl, tt.defaultScheme)
			if tt.expectUrl == nil {
				a.Nil(parsedUrl)
			} else {
				a.Equal(tt.expectUrl, parsedUrl)
			}
			if tt.expectError == nil {
				a.Nil(err)
			} else {
				if a.NotNil(err) {
					a.Equal(tt.expectError.Error(), err.Error())
				}
			}
		})
	}
}

var dataParseTargetURL = []struct {
	rawUrl    string
	expectUrl *url.URL
}{
	{"https://test", &url.URL{Scheme: "https", Host: "test"}},
	{"https://test:8080", &url.URL{Scheme: "https", Host: "test:8080"}},
	{"https://1.2.3.4:8080", &url.URL{Scheme: "https", Host: "1.2.3.4:8080"}},
	{"https://1.2.3.4?test123=v1&test456", &url.URL{Scheme: "https", Host: "1.2.3.4"}},
	{"https://1.2.3.4?test123=v1&test456#fragment1", &url.URL{Scheme: "https", Host: "1.2.3.4"}},
	{"https://1.2.3.4#fragment1", &url.URL{Scheme: "https", Host: "1.2.3.4"}},
	{"https://username:password@1.2.3.4:8080", &url.URL{Scheme: "https", Host: "1.2.3.4:8080"}},
	// whitespace
	{"  https://test  ", &url.URL{Scheme: "https", Host: "test"}},
	// host
	{"test", &url.URL{Scheme: "", Host: "test"}},
	// host port
	{"test:8080", &url.URL{Scheme: "", Host: "test:8080"}},
	// host and query params
	{"test:8080?Test123=v1", &url.URL{Scheme: "", Host: "test:8080"}},
	// host, query params, and fragment
	{"test:8080?Test123=v1#fragment1", &url.URL{Scheme: "", Host: "test:8080"}},
	// Invalid cases
	{"", &url.URL{Scheme: "", Host: "*"}},
	{"*", &url.URL{Scheme: "", Host: "*"}},
	{"  *  ", &url.URL{Scheme: "", Host: "*"}},
	{"://test:8080", &url.URL{Scheme: "", Host: "*"}},
	// TODO These error cases are introduced after Go 1.7
	//{"https://[test:8080", "https://[test:8080"},
	//{"https://username:1412¶45124@test:8080", "https://username:1412¶45124@test:8080"},
}

func TestParseTargetURL(t *testing.T) {
	for _, tt := range dataParseTargetURL {
		t.Run(tt.rawUrl, func(t *testing.T) {
			a := assert.New(t)
			a.Equal(tt.expectUrl, ParseTargetURL(tt.rawUrl, ""))
		})
	}
}

var dataPrefixScheme = []struct {
	rawUrl        string
	defaultScheme string
	expectUrlStr  string
}{
	// No scheme
	{"test123", "", "//test123"},
	{"test123", "https", "https://test123"},
	// Existing scheme
	{"https://test123", "", "https://test123"},
	{"https://test123", "someScheme", "https://test123"},
	// Empty scheme
	{"//", "", "//"},
	{"//test123", "", "//test123"},
	// Empty scheme with default
	{"//test123", "https", "https://test123"},
	// Empty
	{"", "", "//"},
	// Empty whitespace
	{"  ", "", "//  "},
}

func TestPrefixScheme(t *testing.T) {
	for _, tt := range dataPrefixScheme {
		t.Run(tt.rawUrl, func(t *testing.T) {
			a := assert.New(t)
			a.Equal(tt.expectUrlStr, prefixScheme(tt.rawUrl, tt.defaultScheme))
		})
	}
}

var dataSplitHostPort = []struct {
	u          *url.URL
	expectHost string
	expectPort uint16
	expectErr  error
}{
	// hostname
	{&url.URL{Host: "test:8080"}, "test", 8080, nil},
	{&url.URL{Host: "test"}, "test", 0, nil},
	// ipv4
	{&url.URL{Host: "1.2.3.4:8080"}, "1.2.3.4", 8080, nil},
	{&url.URL{Host: "1.2.3.4"}, "1.2.3.4", 0, nil},
	// ipv6
	{&url.URL{Host: "[2001:0db8:85a3:0000:0000:8a2e:0370:7333]"}, "[2001:0db8:85a3:0000:0000:8a2e:0370:7333]", 0, nil},
	{&url.URL{Host: "[2001:0db8:85a3:0000:0000:8a2e:0370:7333]:8080"}, "[2001:0db8:85a3:0000:0000:8a2e:0370:7333]", 8080, nil},
	// port Min
	{&url.URL{Host: "test:0"}, "test", 0, nil},
	// port Max
	{&url.URL{Host: "test:65535"}, "test", 65535, nil},
	// Invalid - port NaN
	{&url.URL{Host: "test1:test2"}, "", 0, errors.New("SplitHostPort test1:test2: strconv.ParseUint: parsing \"test2\": invalid syntax")},
	// Invalid - port Exceeded
	{&url.URL{Host: "test1:65536"}, "", 0, errors.New("SplitHostPort test1:65536: strconv.ParseUint: parsing \"65536\": value out of range")},
	// Invalid - port signed
	{&url.URL{Host: "test1:-1"}, "", 0, errors.New("SplitHostPort test1:-1: strconv.ParseUint: parsing \"-1\": invalid syntax")},
	// Invalid - nil URL
	{nil, "", 0, errors.New("SplitHostPort nil: nil URL")},
}

func TestSplitHostPort(t *testing.T) {
	for _, tt := range dataSplitHostPort {
		var tName string
		if tt.u == nil {
			tName = "nil"
		} else {
			tName = tt.u.Host
		}
		t.Run(tName, func(t *testing.T) {
			a := assert.New(t)
			host, port, err := SplitHostPort(tt.u)
			a.Equal(tt.expectHost, host)
			a.Equal(tt.expectPort, port)
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

var dataIsLoopbackHost = []struct {
	host   string
	expect bool
}{
	{"localhost", true},
	{"  localhost  ", true},
	{"127.0.0.1", true},
	{"::1", true},
	{"[::1]", true},
	{"", false},
	{"1.2.3.4", false},
	{"test.endpoint.rapid7.com", false},
	{"test.endpoint.rapid7.com", false},
}

func TestIsLoopbackHost(t *testing.T) {
	for _, tt := range dataIsLoopbackHost {
		t.Run(tt.host, func(t *testing.T) {
			a := assert.New(t)
			a.Equal(tt.expect, IsLoopbackHost(tt.host))
		})
	}
}
