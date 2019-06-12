package cloudflare

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCustomHostname_DeleteCustomHostname(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/zones/foo/custom_hostnames/bar", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method, "Expected method 'DELETE', got %s", r.Method)

		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `
{
  "id": "bar"
}`)
	})

	err := client.DeleteCustomHostname("foo", "bar")

	assert.NoError(t, err)
}

func TestCustomHostname_CreateCustomHostname(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/zones/foo/custom_hostnames", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method, "Expected method 'POST', got %s", r.Method)

		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `
{
  "success": true,
  "errors": [],
  "messages": [],
  "result": {
    "id": "0d89c70d-ad9f-4843-b99f-6cc0252067e9",
    "hostname": "app.example.com",
    "ssl": {
      "status": "pending_validation",
      "method": "cname",
      "type": "dv",
      "cname_target": "dcv.digicert.com",
      "cname": "810b7d5f01154524b961ba0cd578acc2.app.example.com",
      "settings": {
        "http2": "on"
      }
    }
  }
}`)
	})

	response, err := client.CreateCustomHostname("foo", CustomHostname{Hostname: "app.example.com", SSL: CustomHostnameSSL{Method: "cname", Type: "dv"}})

	want := &CustomHostnameResponse{
		Result: CustomHostname{
			ID:       "0d89c70d-ad9f-4843-b99f-6cc0252067e9",
			Hostname: "app.example.com",
			SSL: CustomHostnameSSL{
				Type:        "dv",
				Method:      "cname",
				Status:      "pending_validation",
				CnameTarget: "dcv.digicert.com",
				CnameName:   "810b7d5f01154524b961ba0cd578acc2.app.example.com",
				Settings: CustomHostnameSSLSettings{
					HTTP2: "on",
				},
			},
		},
		Response: Response{Success: true, Errors: []ResponseInfo{}, Messages: []ResponseInfo{}},
	}

	if assert.NoError(t, err) {
		assert.Equal(t, want, response)
	}
}

func TestCustomHostname_CustomHostnames(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/zones/foo/custom_hostnames", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method, "Expected method 'GET', got %s", r.Method)

		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
"success": true,
"result": [
    {
      "id": "custom_host_1",
      "hostname": "custom.host.one",
			"ssl": {
        "type": "dv",
        "method": "cname",
        "status": "pending_validation",
        "cname_target": "dcv.digicert.com",
        "cname": "810b7d5f01154524b961ba0cd578acc2.app.example.com"
      },
      "custom_metadata": {
				"a_random_field": "random field value"
      }
    }
],
"result_info": {
	  "page": 1,
    "per_page": 20,
    "count": 5,
    "total_count": 5
}
}`)
	})

	customHostnames, _, err := client.CustomHostnames("foo", 1, CustomHostname{})

	want := []CustomHostname{
		{
			ID:       "custom_host_1",
			Hostname: "custom.host.one",
			SSL: CustomHostnameSSL{
				Type:        "dv",
				Method:      "cname",
				Status:      "pending_validation",
				CnameTarget: "dcv.digicert.com",
				CnameName:   "810b7d5f01154524b961ba0cd578acc2.app.example.com",
			},
			CustomMetadata: CustomMetadata{"a_random_field": "random field value"},
		},
	}

	if assert.NoError(t, err) {
		assert.Equal(t, want, customHostnames)
	}
}

func TestCustomHostname_CustomHostname(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/zones/foo/custom_hostnames/bar", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method, "Expected method 'GET', got %s", r.Method)

		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
"success": true,
"result": {
    "id": "bar",
    "hostname": "foo.bar.com",
    "ssl": {
      "type": "dv",
      "method": "http",
      "status": "active",
      "settings": {
        "ciphers": ["ECDHE-RSA-AES128-GCM-SHA256","AES128-SHA"],
        "http2": "on",
        "min_tls_version": "1.2"
      }
    },
    "custom_metadata": {
      "origin": "a.custom.origin"
    }
  }
}`)
	})

	customHostname, err := client.CustomHostname("foo", "bar")

	want := CustomHostname{
		ID:       "bar",
		Hostname: "foo.bar.com",
		SSL: CustomHostnameSSL{
			Status: "active",
			Method: "http",
			Type:   "dv",
			Settings: CustomHostnameSSLSettings{
				HTTP2:         "on",
				MinTLSVersion: "1.2",
				Ciphers:       []string{"ECDHE-RSA-AES128-GCM-SHA256", "AES128-SHA"},
			},
		},
		CustomMetadata: CustomMetadata{"origin": "a.custom.origin"},
	}

	if assert.NoError(t, err) {
		assert.Equal(t, want, customHostname)
	}
}
