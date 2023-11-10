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
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

var dataProviderReadConfigFileProxy = []struct {
	content  string
	expected Proxy
}{
	// Typical
	{"{\"https\": \"1.2.3.4:8080\"}", &proxy{protocol: "", host: "1.2.3.4", port: 8080, src: "ConfigurationFile"}},
	// No port
	{"{\"https\": \"1.2.3.4\"}", &proxy{protocol: "", host: "1.2.3.4", port: 8443, src: "ConfigurationFile"}},
	// Protocol
	{"{\"https\": \"http://test\"}", &proxy{protocol: "http", host: "test", port: 8443, src: "ConfigurationFile"}},
	// All caps
	{"{\"HTTPS\": \"http://test\"}", &proxy{protocol: "http", host: "test", port: 8443, src: "ConfigurationFile"}},
	// Multiple - mixed case - uses last entry
	{"{\"https\": \"http://dontPickMe\", \"HTTPS\": \"http://test\"}", &proxy{protocol: "http", host: "test", port: 8443, src: "ConfigurationFile"}},
	// Mismatched protocol on https
	{"{\"https\": \"socks5://test:8080\"}", &proxy{protocol: "socks5", host: "test", port: 8080, src: "ConfigurationFile"}},
	// Invalid URL
	{"{\"https\": \"   \"}", nil},
	// Another protocol
	{"{\"http\": \"1.2.3.4:8080\"}", nil},
	// Invalid json
	{"{ this is not valid json\"", nil},
}

func TestProvider_ReadConfigFileProxy(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "TestReadConfigFileProxy")
	defer os.RemoveAll(tmpDir)
	for _, tt := range dataProviderReadConfigFileProxy {
		t.Run(tt.content, func(t *testing.T) {
			a := assert.New(t)
			if !a.NoError(err) {
				return
			}
			f := filepath.Join(tmpDir, "proxy.config")
			os.Remove(f)
			fp, err := os.OpenFile(f, os.O_CREATE|os.O_WRONLY, 0644)
			if !a.NoError(err) {
				return
			}
			fp.WriteString(tt.content)
			fp.Close()
			p := newTestProvider(f)
			a.Equal(tt.expected, p.readConfigFileProxy("https"))
		})
	}
}

func TestProvider_ReadConfigFileProxy_noFile(t *testing.T) {
	a := assert.New(t)
	tmpDir, err := os.MkdirTemp("", "TestParseConfigFileProxies")
	if !a.NoError(err) {
		return
	}
	os.RemoveAll(tmpDir)
	_, err = os.Stat(tmpDir)
	if !a.False(os.IsExist(err)) {
		return
	}
	p := newTestProvider(tmpDir)
	a.Nil(p.readConfigFileProxy(""))
}

func TestProvider_ParseConfigFileProxies_isDir(t *testing.T) {
	a := assert.New(t)
	tmpDir, err := os.MkdirTemp("", "TestParseConfigFileProxies")
	if !a.NoError(err) {
		return
	}
	defer os.RemoveAll(tmpDir)
	p := newTestProvider(tmpDir)
	a.Nil(p.readConfigFileProxy(""))
}

func TestProvider_ParseConfigFileProxies_emptyFile(t *testing.T) {
	a := assert.New(t)
	tmpDir, err := os.MkdirTemp("", "TestParseConfigFileProxies")
	if !a.NoError(err) {
		return
	}
	defer os.RemoveAll(tmpDir)
	f := filepath.Join(tmpDir, "proxy.config")
	fp, err := os.OpenFile(f, os.O_CREATE|os.O_WRONLY, 0644)
	if !a.NoError(err) {
		return
	}
	fp.Close()
	if _, err := os.Stat(f); !a.NoError(err) {
		return
	}
	p := newTestProvider(f)
	a.Nil(p.readConfigFileProxy(""))
}

func TestProvider_ParseConfigFileProxies_tooLarge(t *testing.T) {
	a := assert.New(t)
	tmpDir, err := os.MkdirTemp("", "TestParseConfigFileProxies")
	if !a.NoError(err) {
		return
	}
	defer os.RemoveAll(tmpDir)
	f := filepath.Join(tmpDir, "proxy.config")
	fp, err := os.OpenFile(f, os.O_CREATE|os.O_WRONLY, 0644)
	if !a.NoError(err) {
		return
	}
	defer fp.Close()
	// Write 1MB+1
	chunk := make([]byte, 1024)
	for i := 0; i < 1024; i++ {
		fp.Write(chunk)
	}
	fp.Write(make([]byte, 1))
	fp.Close()
	stat, err := os.Stat(f)
	if !a.NoError(err) || !a.Equal(int64(1048577), stat.Size()) {
		return
	}
	p := newTestProvider(f)
	a.Nil(p.readConfigFileProxy(""))
}

var dataProviderReadSystemEnvProxiesAll = []struct {
	env       map[string]string
	protocol  string
	targetUrl *url.URL
	expect    Proxy
}{
	// Match upper
	{
		map[string]string{
			"HTTPS_PROXY": "testUpper:8999",
			"https_proxy": "HTTP://testLower",
			"HTTP_PROXY":  "testUpper:8080",
		},
		"https",
		&url.URL{Scheme: "https", Host: "test.endpoint.rapid7.com"},
		newTestProxy("", "testUpper", 8999, nil, "Environment[HTTPS_PROXY]"),
	},
	// Match lower, no proxy does not match
	{
		map[string]string{
			"https_proxy": "HTTP://testLower",
			"HTTP_PROXY":  "testUpper:8080",
			"NO_PROXY":    "someHost",
		},
		"https",
		&url.URL{Scheme: "https", Host: "test.endpoint.rapid7.com"},
		newTestProxy("http", "testLower", 8443, nil, "Environment[https_proxy]"),
	},
	// Match upper, no proxy matches
	{
		map[string]string{
			"HTTPS_PROXY": "http://testUpper",
			"NO_PROXY":    "rapid7.com",
		},
		"https",
		&url.URL{Scheme: "http", Host: "test.endpoint.rapid7.com"},
		nil,
	},
	// Match upper, no proxy matches lower
	{
		map[string]string{
			"HTTPS_PROXY": "http://testUpper",
			"no_proxy":    "rapid7.com",
		},
		"https",
		&url.URL{Scheme: "http", Host: "test.endpoint.rapid7.com"},
		nil,
	},
	// Special case for SOCKS, it is replaced with ALL
	{
		map[string]string{
			"SOCKS_PROXY": "socks://testUpper",
		},
		"socks",
		&url.URL{Scheme: "https", Host: "test.endpoint.rapid7.com"},
		nil,
	},
	// Special case for SOCKS5, it is replaced with ALL
	{
		map[string]string{
			"SOCKS_PROXY": "socks://testUpper",
		},
		"socks5",
		&url.URL{Scheme: "https", Host: "test.endpoint.rapid7.com"},
		nil,
	},
	// Special case for SOCKS, it is replaced with ALL
	{
		map[string]string{
			"ALL_PROXY": "socks://testUpper",
		},
		"socks",
		&url.URL{Scheme: "https", Host: "test.endpoint.rapid7.com"},
		newTestProxy("socks", "testUpper", 8443, nil, "Environment[ALL_PROXY]"),
	},
	{
		map[string]string{}, "https", new(url.URL), nil,
	},
}

func TestProvider_ReadSystemEnvProxy(t *testing.T) {
	for _, tt := range dataProviderReadSystemEnvProxiesAll {
		tName := fmt.Sprintf("%s", tt.env)
		t.Run(tName, func(t *testing.T) {
			a := assert.New(t)
			getEnv := func(key string) string {
				return tt.env[key]
			}
			p := newTestProvider("")
			p.getEnv = getEnv
			a.Equal(tt.expect, p.readSystemEnvProxy(tt.protocol, tt.targetUrl))
		})
	}
}

var dataProviderParseEnvHTTPSProxy = []struct {
	value       string
	expectProxy Proxy
	expectError error
}{
	{"http://test", newTestProxy("http", "test", 8443, nil, "Environment[Key]"), nil},
	{"test:8080", newTestProxy("", "test", 8080, nil, "Environment[Key]"), nil},
	{"1.2.3.4:8080", newTestProxy("", "1.2.3.4", 8080, nil, "Environment[Key]"), nil},
	{"http://username:password@1.2.3.4:8080", newTestProxy("http", "1.2.3.4", 8080, url.UserPassword("username", "password"), "Environment[Key]"), nil},
	{"username:password@1.2.3.4:8080", newTestProxy("", "1.2.3.4", 8080, url.UserPassword("username", "password"), "Environment[Key]"), nil},
	{"", nil, new(notFoundError)},
	{"   ", nil, new(notFoundError)},
	{"HTTPS://test:8080", newTestProxy("https", "test", 8080, nil, "Environment[Key]"), nil},
	{"test:8999", newTestProxy("", "test", 8999, nil, "Environment[Key]"), nil},
	// Invalid
	{"://test:8080", nil, errors.New("parse ://test:8080: missing protocol scheme")},
	// TODO These error cases are introduced after Go 1.7
	//{"https", "https://[test:8080", nil, errors.New("parse https://[test:8080: missing ']' in host")},
	//{"https", "https://username:1412¶45124@test:8080", nil, errors.New("parse https://username:1412¶45124@test:8080: net/url: invalid userinfo")},
}

func TestProvider_ParseEnvProxy(t *testing.T) {
	for _, tt := range dataProviderParseEnvHTTPSProxy {
		t.Run(tt.value, func(t *testing.T) {
			a := assert.New(t)
			getEnv := func(key string) string {
				a.Equal("Key", key)
				return tt.value
			}
			p := newTestProvider("")
			p.getEnv = getEnv
			proxy, err := p.parseEnvProxy("Key")
			if tt.expectProxy == nil {
				a.Nil(proxy)
			} else {
				a.Equal(tt.expectProxy, proxy)
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

var dataProviderParseEnvURL = []struct {
	value       string
	expectUrl   *url.URL
	expectError error
}{
	{"http://test", &url.URL{Scheme: "http", Host: "test"}, nil},
	{"test:8080", &url.URL{Scheme: "", Host: "test:8080"}, nil},
	{"1.2.3.4:8080", &url.URL{Scheme: "", Host: "1.2.3.4:8080"}, nil},
	{"HTTP://username:password@1.2.3.4:8080", &url.URL{Scheme: "http", Host: "1.2.3.4:8080", User: url.UserPassword("username", "password")}, nil},
	{"username:password@1.2.3.4:8080", &url.URL{Scheme: "", Host: "1.2.3.4:8080", User: url.UserPassword("username", "password")}, nil},
	{"", nil, new(notFoundError)},
	{"   ", nil, new(notFoundError)},
	// Invalid
	{"://test:8080", nil, errors.New("parse ://test:8080: missing protocol scheme")},
	// TODO These error cases are introduced after Go 1.7
	//{"https://[test:8080", nil, errors.New("parse https://[test:8080: missing ']' in host")},
	//{"https://username:1412¶45124@test:8080", nil, errors.New("parse https://username:1412¶45124@test:8080: net/url: invalid userinfo")},

}

func TestProvider_ConfigProviderParseEnvURL(t *testing.T) {
	for _, tt := range dataProviderParseEnvURL {
		t.Run(tt.value, func(t *testing.T) {
			a := assert.New(t)
			getEnv := func(key string) string {
				a.Equal("Key", key)
				return tt.value
			}
			p := newTestProvider("")
			p.getEnv = getEnv
			parsedUrl, err := p.parseEnvURL("Key")
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

var dataProviderIsProxyBypass = []struct {
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
	{&url.URL{Host: "test:8080"}, ",", false},
	// Matched
	{&url.URL{Host: "test.endpoint.rapid7.com"}, "rapid7.com", true},
	// Matched - Sub Domain
	{&url.URL{Host: "test.endpoint.rapid7.com:443"}, ".rapid7.com", true},
	// Matched - Wildcard Prefix
	{&url.URL{Host: "test.endpoint.rapid7.com:443"}, "*.rapid7.com", true},
	// Matched - Multiple wildcards
	{&url.URL{Host: "test.endpoint.rapid7.com:443"}, "test.*.*.com", true},
	// Matched - Second value
	{&url.URL{Host: "test.endpoint.rapid7.com"}, "someHost,rapid7.com", true},
	// Matched - Just wildcard
	{&url.URL{Host: "test.endpoint.rapid7.com"}, "*", true},
	// Matched - Wildcard second
	{&url.URL{Host: "test.endpoint.rapid7.com"}, ",*", true},
	// Exact match
	{&url.URL{Host: "test.endpoint.rapid7.com"}, "test.endpoint.rapid7.com", true},
	// Matched - Local Host
	{&url.URL{Host: "localhost"}, "<local>", true},
	// Matched - Local Host second
	{&url.URL{Host: "localhost"}, "someHost,<local>", true},
	// Matched - Local IPv4
	{&url.URL{Host: "[::1]"}, "<local>", true},
	// Matched - Local IPv6
	{&url.URL{Host: "127.0.0.1"}, "<local>", true},
	// Not Matched
	{&url.URL{Host: "test.endpoint.rapid7.com"}, "someHost", false},
	// Not Matched - Not local
	{&url.URL{Host: "test.endpoint.rapid7.com"}, "<local>", false},
}

func TestProvider_IsProxyBypass(t *testing.T) {
	for _, tt := range dataProviderIsProxyBypass {
		var tName string
		if tt.targetUrl == nil {
			tName = "nil"
		} else {
			tName = tt.targetUrl.String()
		}
		tName = tName + " " + tt.proxyBypass
		t.Run(tName, func(t *testing.T) {
			a := assert.New(t)
			p := newTestProvider("")
			a.Equal(tt.expect, p.isProxyBypass(tt.targetUrl, tt.proxyBypass, ","))
		})
	}
}

func newTestProvider(configFile string) *provider {
	c := new(provider)
	c.init(configFile)
	return c
}

func newTestProxy(protocol string, host string, port uint16, user *url.Userinfo, src string) Proxy {
	return &proxy{
		protocol: protocol,
		host:     host,
		port:     port,
		user:     user,
		src:      src,
	}
}
