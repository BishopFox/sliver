package proxy

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	ScutilDataHttpsHttp = "ScutilDataHttpsHttp"
	ScutilDataHttps     = "ScutilDataHttps"
	ScutilDataHttp      = "ScutilDataHttp"
)

var (
	ScutilBypassTest1 = "localhost"
	ScutilBypassTest2 = "myorg1.com"
	ScutilBypassTest3 = "endpoint.myorg2.com"
	ScutilBypassTest4 = ".myorg3.com"
	ScutilBypassTest5 = "*.myorg4.com"
	ScutilBypassTest6 = "test.endpoint.myorg5.com"
	ScutilBypassTest7 = "test.test.*.*.com"
)

var providerDarwinTestCasesNoBypass = []struct {
	testType string
	test     string
}{
	{ScutilDataHttpsHttp,
		"<dictionary> {\n  HTTPSEnable : 1\n HTTPSPort : 1234\n  HTTPSProxy : 1.2.3.4\n HTTPEnable : 1\n  " +
			"HTTPPort : 1234\n  HTTPProxy : 1.2.3.4\n}"},
	{ScutilDataHttpsHttp,
		"         <dictionary> {   \n  HTTPSEnable:1    \nHTTPSPort :  1234\n  HTTPSProxy :  1.2.3.4\n " +
			"HTTPEnable : 1   \n  HTTPPort :      1234\nHTTPProxy: 1.2.3.4   \n    }"},
	{ScutilDataHttps,
		"<dictionary> {\n  HTTPEnable : 0\n  HTTPSEnable : 1\n  HTTPSPort : 1234\n  HTTPSProxy : 1.2.3.4\n}"},
	{ScutilDataHttps,
		"<dictionary> {\n   HTTPEnable: 0\n  HTTPSEnable: 1\nHTTPSPort :      1234\n     HTTPSProxy:   1.2.3.4\n}"},
	{ScutilDataHttp,
		"<dictionary> {\n  HTTPSEnable : 0\n  HTTPEnable : 1\n  HTTPPort : 1234\n  HTTPProxy : 1.2.3.4\n}"},
	{ScutilDataHttp,
		"<dictionary> {\n     HTTPSEnable:0\n  HTTPEnable: 1\nHTTPPort :       1234\n     HTTPProxy:   1.2.3.4\n}"},
}

var bypassProxySettingsHostsDomains = map[string][]map[string]bool{
	ScutilBypassTest1: { // TODO: for MAC, we need to support "*.local"
		{"localhost": true},
		{"localhost:1234": true},
		{"localhost:8080": true},
		{"http://localhost:8080": true},
		{"local": false},
	},
	ScutilBypassTest2: {
		{"myorg1.com": true},
		{"myorg1.com:443": true},
		{"www.myorg1.com": true},
		{"https://myorg1.com": true},
		{"https://www.myorg1.com": true},
		{"https://www.myorg1.com:443": true},
		{"https://www.myorg2.com": false},
		{"myorg2.com": false},
		{"myorg1:myorg1": false},
	},
	ScutilBypassTest3: {
		{"endpoint.myorg2.com": true},
		{"myorg2.com": false},
		{"myorg2.com:1234": false},
		{"endpoint.myorg2.com:443": true},
		{"https://endpoint.myorg2.com": true},
		{"https://endpoint.myorg2.com:8443": true},
		{"test.endpoint.myorg2.com": true},
		{"test.endpoint.myorg2.com:1234": true},
		{"https://test.endpoint.myorg2.com:1234": true},
		{"test.endpoint.com": false},
	},
	ScutilBypassTest4: {
		{"endpoint.myorg3.com": true},
		{"myorg3.com": false},
		{"myorg3.com:1234": false},
		{"endpoint.myorg3.com:443": true},
		{"https://endpoint.myorg3.com": true},
		{"https://endpoint.myorg3.com:8443": true},
		{"test.endpoint.myorg3.com": true},
		{"test.endpoint.myorg3.com:1234": true},
		{"https://test.endpoint.myorg3.com:1234": true},
		{"test.endpoint.com": false},
	},
	ScutilBypassTest5: {
		{"endpoint.myorg4.com": true},
		{"myorg4.com": false},
		{"myorg4.com:1234": false},
		{"endpoint.myorg4.com:443": true},
		{"https://endpoint.myorg4.com": true},
		{"https://endpoint.myorg4.com:8443": true},
		{"test.endpoint.myorg4.com": true},
		{"test.endpoint.myorg4.com:1234": true},
		{"https://test.endpoint.myorg4.com:1234": true},
		{"test.endpoint.com": false},
	},
	ScutilBypassTest6: {
		{"test.endpoint.myorg5.com": true},
		{"myorg5.com": false},
		{"myorg5.com:1234": false},
		{"endpoint.myorg5.com": false},
		{"endpoint.myorg5.com:443": false},
		{"https://test.endpoint.myorg5.com": true},
		{"https://test.endpoint.myorg5.com:8443": true},
		{"test1.test.endpoint.myorg5.com": true},
		{"test1.test.endpoint.myorg5.com:1234": true},
		{"https://test1.test.endpoint.myorg5.com:1234": true},
		{"test.endpoint.com": false},
	},
}

var providerDarwinTestCaseBypass = fmt.Sprintf("<dictionary> {\n  ExceptionsList : <array> {\n    0 : %s\n    1 : %s\n    2 : %s\n    "+
	"3 : %s\n    4: %s\n    5: %s\n    6 : %s\n  }\n  HTTPEnable : 0\n  HTTPSEnable : 1\n  HTTPSPort : 1234\n "+
	"HTTPSProxy : 1.2.3.4\n}\n", ScutilBypassTest1, ScutilBypassTest2, ScutilBypassTest3, ScutilBypassTest4,
	ScutilBypassTest5, ScutilBypassTest6, ScutilBypassTest7)

func getDarwinProviderTestsNoBypass(key string) []string {
	var s []string
	for _, v := range providerDarwinTestCasesNoBypass {
		if v.testType == key {
			s = append(s, v.test)
		}
	}
	return s
}

/*
Below tests cover cases when both https and http proxies are present.
following tests are being performed:
- Test https and http proxies are not nil,
- Test https and http proxies match expected,
- Test for lower case and upper case, i.e. https/HTTPS/...,
- Test no errors are returned
*/
func TestParseScutildata_Read_HTTPS_HTTP(t *testing.T) {
	a := assert.New(t)

	c := newDarwinTestProvider()

	commands := getDarwinProviderTestsNoBypass(ScutilDataHttpsHttp)

	targetUrl := ParseTargetURL("", "")

	protocols := [4]string{"http", "https", "HTTP", "HTTPS"}
	for _, protocol := range protocols {
		for _, command := range commands {
			expectedProxy, err := c.parseScutildata(protocol, targetUrl, "echo", command)
			// test error is nil
			a.Nil(err)
			// test https and https proxies are not nil
			a.NotNil(expectedProxy)
			// test expected https proxy matches hardcoded proxy, test lowercase
			a.Equal(&proxy{src: "State:/Network/Global/Proxies", host: "1.2.3.4", port: 1234},
				expectedProxy)
		}
	}
}

/*
Below tests cover cases when only https proxy is present.
following tests are being performed:
- Test https proxy is not nil,
- Test http proxy is nil,
- Test https proxy match expected,
- Test for lower case and upper case, i.e. https/HTTPS/...,
- Test no errors are returned
*/
func TestParseScutildata_Read_HTTPS(t *testing.T) {
	a := assert.New(t)

	c := newDarwinTestProvider()

	commands := getDarwinProviderTestsNoBypass(ScutilDataHttps)

	targetUrl := ParseTargetURL("", "")

	protocols := [4]string{"http", "https", "HTTP", "HTTPS"}
	for _, protocol := range protocols {
		for _, command := range commands {
			expectedProxy, err := c.parseScutildata(protocol, targetUrl, "echo", command)
			if strings.ToLower(protocol) == "https" {
				// test error is nil
				a.Nil(err)
				// test https proxy is not nil
				a.NotNil(expectedProxy)
				// test expected https proxy matches hardcoded proxy
				a.Equal(&proxy{src: "State:/Network/Global/Proxies", host: "1.2.3.4", port: 1234},
					expectedProxy)
			} else {
				// test http proxy is nil
				a.Nil(expectedProxy)
				a.Equal(isNotFound(err), true)
			}
		}
	}
}

/*
Below tests cover cases when only http proxy is present.
following tests are being performed:
- Test http proxy is not nil,
- Test https proxy is nil,
- Test http proxy match expected,
- Test for lower case and upper case, i.e. https/HTTPS/...,
- Test no errors are returned
*/
func TestParseScutildata_Read_HTTP(t *testing.T) {
	a := assert.New(t)

	c := newDarwinTestProvider()

	commands := getDarwinProviderTestsNoBypass(ScutilDataHttp)

	targetUrl := ParseTargetURL("", "")

	protocols := [4]string{"http", "https", "HTTP", "HTTPS"}
	for _, protocol := range protocols {
		for _, command := range commands {
			expectedProxy, err := c.parseScutildata(protocol, targetUrl, "echo", command)
			if strings.ToLower(protocol) == "http" {
				// test error is nil
				a.Nil(err)
				// test http proxy is not nil
				a.NotNil(expectedProxy)
				// test expected http proxy matches hardcoded proxy
				a.Equal(&proxy{src: "State:/Network/Global/Proxies", host: "1.2.3.4", port: 1234},
					expectedProxy)
			} else {
				// test https proxy is nil
				a.Nil(expectedProxy)
				a.Equal(isNotFound(err), true)
			}
		}
	}
}

/*
Tests whether the timeout property functions as expected
*/
func TestExecCommandsHandledProperly(t *testing.T) {
	a := assert.New(t)

	c := newDarwinTestProvider()

	targetUrl := ParseTargetURL("", "")
	expectedProxy, err := c.parseScutildata("", targetUrl, "exit", "")

	a.Equal(isTimedOut(err), true)
	a.Equal(expectedProxy, nil)
}

/*
Below tests cover cases when only https proxy is present, and there is some bypass settings. The tests are performed for
different target URLs.
following tests are being performed:
- Test https proxy is returned as expected based on the configured bypass settings,
- Test http proxy is nil,
- Test no errors are returned
*/
func TestParseScutildata_Read_HTTPS_BYPASS_TARGET_URL(t *testing.T) {
	a := assert.New(t)

	c := newDarwinTestProvider()

	command := providerDarwinTestCaseBypass

	protocol := "https"

	// test empty target URL
	targetUrl := ParseTargetURL("", "")
	expectedProxy, err := c.parseScutildata(protocol, targetUrl, "echo", command)
	// test error is nil
	a.Nil(err)
	// test expected proxy matches hardcoded proxy
	a.Equal(&proxy{src: "State:/Network/Global/Proxies", host: "1.2.3.4", port: 1234},
		expectedProxy)
	// test proxy is not nil
	a.NotNil(c.parseScutildata(protocol, targetUrl, "echo", command))

	// tests https
	for _, tests := range bypassProxySettingsHostsDomains {
		for _, test := range tests {
			for urlStr, result := range test {
				targetUrl := ParseTargetURL(urlStr, "")
				expectedProxy, err := c.parseScutildata(protocol, targetUrl, "echo", command)
				// test error is nil
				a.Nil(err)
				if result == true {
					// test https proxy is nil
					a.Nil(expectedProxy)
				} else {
					// test expected https proxy matches hardcoded proxy
					a.Equal(&proxy{src: "State:/Network/Global/Proxies", host: "1.2.3.4", port: 1234},
						expectedProxy)
				}
			}
		}
	}

	//tests http
	protocol = "http"
	for _, tests := range bypassProxySettingsHostsDomains {
		for _, test := range tests {
			for urlStr := range test {
				targetUrl := ParseTargetURL(urlStr, "")
				expectedProxy, err := c.parseScutildata(protocol, targetUrl, "echo", command)
				// test error is returned correctly
				a.Equal(isNotFound(err), true)
				// test proxy is nil
				a.Nil(expectedProxy)
			}
		}
	}
}

func newDarwinTestProvider() *providerDarwin {
	Cmd := func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, name, args...)
	}
	c := new(providerDarwin)
	c.proc = Cmd
	return c
}
