package cloudflare

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateRailgun(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "POST", "Expected method 'POST', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		b, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if assert.NoError(t, err) {
			assert.JSONEq(t, `{"name":"My Railgun"}`, string(b))
		}
		fmt.Fprint(w, `{
            "success": true,
            "errors": [],
            "messages": [],
            "result": {
                "id": "e928d310693a83094309acf9ead50448",
                "name": "My Railgun",
                "status": "active",
                "enabled": true,
                "zones_connected": 2,
                "build": "b1234",
                "version": "2.1",
                "revision": "123",
                "activation_key": "e4edc00281cb56ebac22c81be9bac8f3",
                "activated_on": "2014-01-02T02:20:00Z",
                "created_on": "2014-01-01T05:20:00Z",
                "modified_on": "2014-01-01T05:20:00Z"
            }
        }`)
	}

	mux.HandleFunc("/railguns", handler)
	activatedOn, _ := time.Parse(time.RFC3339, "2014-01-02T02:20:00Z")
	createdOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00Z")
	modifiedOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00Z")
	want := Railgun{
		ID:             "e928d310693a83094309acf9ead50448",
		Name:           "My Railgun",
		Status:         "active",
		Enabled:        true,
		ZonesConnected: 2,
		Build:          "b1234",
		Version:        "2.1",
		Revision:       "123",
		ActivationKey:  "e4edc00281cb56ebac22c81be9bac8f3",
		ActivatedOn:    activatedOn,
		CreatedOn:      createdOn,
		ModifiedOn:     modifiedOn,
	}

	actual, err := client.CreateRailgun("My Railgun")
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestListRailguns(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "GET", "Expected method 'GET', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{
            "success": true,
            "errors": [],
            "messages": [],
            "result": [
                {
                    "id": "e928d310693a83094309acf9ead50448",
                    "name": "My Railgun",
                    "status": "active",
                    "enabled": true,
                    "zones_connected": 2,
                    "build": "b1234",
                    "version": "2.1",
                    "revision": "123",
                    "activation_key": "e4edc00281cb56ebac22c81be9bac8f3",
                    "activated_on": "2014-01-02T02:20:00Z",
                    "created_on": "2014-01-01T05:20:00Z",
                    "modified_on": "2014-01-01T05:20:00Z"
                }
            ],
            "result_info": {
                "page": 1,
                "per_page": 20,
                "count": 1,
                "total_count": 2000
            }
        }`)
	}

	mux.HandleFunc("/railguns", handler)
	activatedOn, _ := time.Parse(time.RFC3339, "2014-01-02T02:20:00Z")
	createdOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00Z")
	modifiedOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00Z")
	want := []Railgun{
		{
			ID:             "e928d310693a83094309acf9ead50448",
			Name:           "My Railgun",
			Status:         "active",
			Enabled:        true,
			ZonesConnected: 2,
			Build:          "b1234",
			Version:        "2.1",
			Revision:       "123",
			ActivationKey:  "e4edc00281cb56ebac22c81be9bac8f3",
			ActivatedOn:    activatedOn,
			CreatedOn:      createdOn,
			ModifiedOn:     modifiedOn,
		},
	}

	actual, err := client.ListRailguns(RailgunListOptions{Direction: "desc"})
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestRailgunDetails(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "GET", "Expected method 'GET', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{
            "success": true,
            "errors": [],
            "messages": [],
            "result": {
                "id": "e928d310693a83094309acf9ead50448",
                "name": "My Railgun",
                "status": "active",
                "enabled": true,
                "zones_connected": 2,
                "build": "b1234",
                "version": "2.1",
                "revision": "123",
                "activation_key": "e4edc00281cb56ebac22c81be9bac8f3",
                "activated_on": "2014-01-02T02:20:00Z",
                "created_on": "2014-01-01T05:20:00Z",
                "modified_on": "2014-01-01T05:20:00Z"
            }
        }`)
	}

	mux.HandleFunc("/railguns/e928d310693a83094309acf9ead50448", handler)
	activatedOn, _ := time.Parse(time.RFC3339, "2014-01-02T02:20:00Z")
	createdOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00Z")
	modifiedOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00Z")
	want := Railgun{
		ID:             "e928d310693a83094309acf9ead50448",
		Name:           "My Railgun",
		Status:         "active",
		Enabled:        true,
		ZonesConnected: 2,
		Build:          "b1234",
		Version:        "2.1",
		Revision:       "123",
		ActivationKey:  "e4edc00281cb56ebac22c81be9bac8f3",
		ActivatedOn:    activatedOn,
		CreatedOn:      createdOn,
		ModifiedOn:     modifiedOn,
	}

	actual, err := client.RailgunDetails("e928d310693a83094309acf9ead50448")
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}

	_, err = client.RailgunDetails("bar")
	assert.Error(t, err)
}

func TestRailgunZones(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "GET", "Expected method 'GET', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{
            "success": true,
            "errors": [],
            "messages": [],
            "result": [
                {
                    "id": "023e105f4ecef8ad9ca31a8372d0c353",
                    "name": "example.com",
                    "development_mode": 7200,
                    "original_name_servers": [
                        "ns1.originaldnshost.com",
                        "ns2.originaldnshost.com"
                    ],
                    "original_registrar": "GoDaddy",
                    "original_dnshost": "NameCheap",
                    "created_on": "2014-01-01T05:20:00.12345Z",
                    "modified_on": "2014-01-01T05:20:00.12345Z"
                }
            ],
            "result_info": {
                "page": 1,
                "per_page": 20,
                "count": 1,
                "total_count": 2000
            }
        }`)
	}

	mux.HandleFunc("/railguns/e928d310693a83094309acf9ead50448/zones", handler)
	createdOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00.12345Z")
	modifiedOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00.12345Z")
	want := []Zone{
		{
			ID:                "023e105f4ecef8ad9ca31a8372d0c353",
			Name:              "example.com",
			DevMode:           7200,
			OriginalNS:        []string{"ns1.originaldnshost.com", "ns2.originaldnshost.com"},
			OriginalRegistrar: "GoDaddy",
			OriginalDNSHost:   "NameCheap",
			CreatedOn:         createdOn,
			ModifiedOn:        modifiedOn,
		},
	}

	actual, err := client.RailgunZones("e928d310693a83094309acf9ead50448")
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}

	_, err = client.RailgunZones("bar")
	assert.Error(t, err)
}

func TestEnableRailgun(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "PATCH", "Expected method 'PATCH', got %s", r.Method)
		b, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if assert.NoError(t, err) {
			assert.JSONEq(t, `{"enabled":true}`, string(b))
		}
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{
            "success": true,
            "errors": [],
            "messages": [],
            "result": {
                "id": "e928d310693a83094309acf9ead50448",
                "name": "My Railgun",
                "status": "active",
                "enabled": true,
                "zones_connected": 2,
                "build": "b1234",
                "version": "2.1",
                "revision": "123",
                "activation_key": "e4edc00281cb56ebac22c81be9bac8f3",
                "activated_on": "2014-01-02T02:20:00Z",
                "created_on": "2014-01-01T05:20:00Z",
                "modified_on": "2014-01-01T05:20:00Z"
            }
        }`)
	}

	mux.HandleFunc("/railguns/e928d310693a83094309acf9ead50448", handler)
	activatedOn, _ := time.Parse(time.RFC3339, "2014-01-02T02:20:00Z")
	createdOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00Z")
	modifiedOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00Z")
	want := Railgun{
		ID:             "e928d310693a83094309acf9ead50448",
		Name:           "My Railgun",
		Status:         "active",
		Enabled:        true,
		ZonesConnected: 2,
		Build:          "b1234",
		Version:        "2.1",
		Revision:       "123",
		ActivationKey:  "e4edc00281cb56ebac22c81be9bac8f3",
		ActivatedOn:    activatedOn,
		CreatedOn:      createdOn,
		ModifiedOn:     modifiedOn,
	}

	actual, err := client.EnableRailgun("e928d310693a83094309acf9ead50448")
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}

	_, err = client.EnableRailgun("bar")
	assert.Error(t, err)
}

func TestDisbleRailgun(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "PATCH", "Expected method 'PATCH', got %s", r.Method)
		b, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if assert.NoError(t, err) {
			assert.JSONEq(t, `{"enabled":false}`, string(b))
		}
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{
            "success": true,
            "errors": [],
            "messages": [],
            "result": {
                "id": "e928d310693a83094309acf9ead50448",
                "name": "My Railgun",
                "status": "active",
                "enabled": false,
                "zones_connected": 2,
                "build": "b1234",
                "version": "2.1",
                "revision": "123",
                "activation_key": "e4edc00281cb56ebac22c81be9bac8f3",
                "activated_on": "2014-01-02T02:20:00Z",
                "created_on": "2014-01-01T05:20:00Z",
                "modified_on": "2014-01-01T05:20:00Z"
            }
        }`)
	}

	mux.HandleFunc("/railguns/e928d310693a83094309acf9ead50448", handler)
	activatedOn, _ := time.Parse(time.RFC3339, "2014-01-02T02:20:00Z")
	createdOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00Z")
	modifiedOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00Z")
	want := Railgun{
		ID:             "e928d310693a83094309acf9ead50448",
		Name:           "My Railgun",
		Status:         "active",
		Enabled:        false,
		ZonesConnected: 2,
		Build:          "b1234",
		Version:        "2.1",
		Revision:       "123",
		ActivationKey:  "e4edc00281cb56ebac22c81be9bac8f3",
		ActivatedOn:    activatedOn,
		CreatedOn:      createdOn,
		ModifiedOn:     modifiedOn,
	}

	actual, err := client.DisableRailgun("e928d310693a83094309acf9ead50448")
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}

	_, err = client.DisableRailgun("bar")
	assert.Error(t, err)
}

func TestDeleteRailgun(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "DELETE", "Expected method 'DELETE', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{
            "success": true,
            "errors": [],
            "messages": [],
            "result": {
                "id": "e928d310693a83094309acf9ead50448"
            }
        }`)
	}

	mux.HandleFunc("/railguns/e928d310693a83094309acf9ead50448", handler)
	assert.NoError(t, client.DeleteRailgun("e928d310693a83094309acf9ead50448"))
	assert.Error(t, client.DeleteRailgun("bar"))
}

func TestZoneRailguns(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "GET", "Expected method 'GET', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{
            "success": true,
            "errors": [],
            "messages": [],
            "result": [
                {
                    "id": "e928d310693a83094309acf9ead50448",
                    "name": "My Railgun",
                    "enabled": true,
                    "connected": true
                }
            ],
            "result_info": {
                "page": 1,
                "per_page": 20,
                "count": 1,
                "total_count": 2000
            }
        }`)
	}

	mux.HandleFunc("/zones/023e105f4ecef8ad9ca31a8372d0c353/railguns", handler)
	want := []ZoneRailgun{
		{
			ID:        "e928d310693a83094309acf9ead50448",
			Name:      "My Railgun",
			Enabled:   true,
			Connected: true,
		},
	}

	actual, err := client.ZoneRailguns("023e105f4ecef8ad9ca31a8372d0c353")
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}

	_, err = client.ZoneRailguns("bar")
	assert.Error(t, err)
}

func TestZoneRailgunDetails(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "GET", "Expected method 'GET', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{
            "success": true,
            "errors": [],
            "messages": [],
            "result": {
                "id": "e928d310693a83094309acf9ead50448",
                "name": "My Railgun",
                "enabled": true,
                "connected": true
            }
        }`)
	}

	mux.HandleFunc("/zones/023e105f4ecef8ad9ca31a8372d0c353/railguns/e928d310693a83094309acf9ead50448", handler)
	want := ZoneRailgun{
		ID:        "e928d310693a83094309acf9ead50448",
		Name:      "My Railgun",
		Enabled:   true,
		Connected: true,
	}

	actual, err := client.ZoneRailgunDetails("023e105f4ecef8ad9ca31a8372d0c353", "e928d310693a83094309acf9ead50448")
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}

	_, err = client.ZoneRailgunDetails("bar", "baz")
	assert.Error(t, err)
}

func TestTestRailgunConnection(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "GET", "Expected method 'GET', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{
            "success": true,
            "errors": [],
            "messages": [],
            "result": {
                "method": "GET",
                "host_name": "www.example.com",
                "http_status": 200,
                "railgun": "on",
                "url": "https://www.cloudflare.com",
                "response_status": "200 OK",
                "protocol": "HTTP/1.1",
                "elapsed_time": "0.239013s",
                "body_size": "63910 bytes",
                "body_hash": "be27f2429421e12f200cab1da43ba301bdc70e1d",
                "missing_headers": "No Content-Length or Transfer-Encoding",
                "connection_close": false,
                "cloudflare": "on",
                "cf-ray": "1ddd7570575207d9-LAX",
                "cf-wan-error": null,
                "cf-cache-status": null
            }
        }`)
	}

	mux.HandleFunc("/zones/023e105f4ecef8ad9ca31a8372d0c353/railguns/e928d310693a83094309acf9ead50448/diagnose", handler)
	want := RailgunDiagnosis{
		Method:          "GET",
		HostName:        "www.example.com",
		HTTPStatus:      200,
		Railgun:         "on",
		URL:             "https://www.cloudflare.com",
		ResponseStatus:  "200 OK",
		Protocol:        "HTTP/1.1",
		ElapsedTime:     "0.239013s",
		BodySize:        "63910 bytes",
		BodyHash:        "be27f2429421e12f200cab1da43ba301bdc70e1d",
		MissingHeaders:  "No Content-Length or Transfer-Encoding",
		ConnectionClose: false,
		Cloudflare:      "on",
		CFRay:           "1ddd7570575207d9-LAX",
		CFWANError:      "",
		CFCacheStatus:   "",
	}

	actual, err := client.TestRailgunConnection("023e105f4ecef8ad9ca31a8372d0c353", "e928d310693a83094309acf9ead50448")
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}

	_, err = client.TestRailgunConnection("bar", "baz")
	assert.Error(t, err)
}

func TestConnectRailgun(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "PATCH", "Expected method 'PATCH', got %s", r.Method)
		b, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if assert.NoError(t, err) {
			assert.JSONEq(t, `{"connected":true}`, string(b))
		}
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{
            "success": true,
            "errors": [],
            "messages": [],
            "result": {
                "id": "e928d310693a83094309acf9ead50448",
                "name": "My Railgun",
                "enabled": true,
                "connected": true
            }
        }`)
	}

	mux.HandleFunc("/zones/023e105f4ecef8ad9ca31a8372d0c353/railguns/e928d310693a83094309acf9ead50448", handler)
	want := ZoneRailgun{
		ID:        "e928d310693a83094309acf9ead50448",
		Name:      "My Railgun",
		Enabled:   true,
		Connected: true,
	}

	actual, err := client.ConnectZoneRailgun("023e105f4ecef8ad9ca31a8372d0c353", "e928d310693a83094309acf9ead50448")
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}

	_, err = client.ConnectZoneRailgun("bar", "baz")
	assert.Error(t, err)
}

func TestDisconnectRailgun(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "PATCH", "Expected method 'PATCH', got %s", r.Method)
		b, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if assert.NoError(t, err) {
			assert.JSONEq(t, `{"connected":false}`, string(b))
		}
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{
            "success": true,
            "errors": [],
            "messages": [],
            "result": {
                "id": "e928d310693a83094309acf9ead50448",
                "name": "My Railgun",
                "enabled": true,
                "connected": false
            }
        }`)
	}

	mux.HandleFunc("/zones/023e105f4ecef8ad9ca31a8372d0c353/railguns/e928d310693a83094309acf9ead50448", handler)
	want := ZoneRailgun{
		ID:        "e928d310693a83094309acf9ead50448",
		Name:      "My Railgun",
		Enabled:   true,
		Connected: false,
	}

	actual, err := client.DisconnectZoneRailgun("023e105f4ecef8ad9ca31a8372d0c353", "e928d310693a83094309acf9ead50448")
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}

	_, err = client.DisconnectZoneRailgun("bar", "baz")
	assert.Error(t, err)
}
