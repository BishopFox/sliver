package cloudflare

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUniversalSSLSettingDetails(t *testing.T) {
	setup()
	defer teardown()

	testZoneID := "abcd123"

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "GET", "Expected method 'GET', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
			"success": true,
			"errors": [],
			"messages": [],
			"result": {
			  "enabled": true
			}
		  }`)
	}

	mux.HandleFunc("/zones/"+testZoneID+"/ssl/universal/settings", handler)

	want := UniversalSSLSetting{
		Enabled: true,
	}

	got, err := client.UniversalSSLSettingDetails(testZoneID)
	if assert.NoError(t, err) {
		assert.Equal(t, want, got)
	}
}

func TestEditUniversalSSLSetting(t *testing.T) {
	setup()
	defer teardown()

	testZoneID := "abcd123"

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method, "Expected method 'PATCH', got %s", r.Method)
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		defer r.Body.Close()

		assert.Equal(t, `{"enabled":true}`, string(body))

		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
			"success": true,
			"errors": [],
			"messages": [],
			"result": {
			  "enabled": true
			}
		  }`)
	}

	mux.HandleFunc("/zones/"+testZoneID+"/ssl/universal/settings", handler)

	want := UniversalSSLSetting{
		Enabled: true,
	}

	got, err := client.EditUniversalSSLSetting(testZoneID, want)
	if assert.NoError(t, err) {
		assert.Equal(t, want, got)
	}
}
