package cloudflare

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListWAFPackages(t *testing.T) {
	setup()
	defer teardown()

	testZoneID := "abcd123"

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method, "Expected method 'GET', got %s", r.Method)

		w.Header().Set("content-type", "application/json")
		// JSON data from: https://api.cloudflare.com/#waf-rule-packages-properties
		fmt.Fprintf(w, `{
			"success": true,
			"errors": [],
			"messages": [],
			"result": [
				{
				"id": "a25a9a7e9c00afc1fb2e0245519d725b",
				"name": "WordPress rules",
				"description": "Common WordPress exploit protections",
				"detection_mode": "traditional",
				"zone_id": "023e105f4ecef8ad9ca31a8372d0c353",
				"status": "active"
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

	mux.HandleFunc("/zones/"+testZoneID+"/firewall/waf/packages", handler)

	want := []WAFPackage{
		{
			ID:            "a25a9a7e9c00afc1fb2e0245519d725b",
			Name:          "WordPress rules",
			Description:   "Common WordPress exploit protections",
			ZoneID:        "023e105f4ecef8ad9ca31a8372d0c353",
			DetectionMode: "traditional",
			Sensitivity:   "",
			ActionMode:    "",
		},
	}

	d, err := client.ListWAFPackages(testZoneID)

	if assert.NoError(t, err) {
		assert.Equal(t, want, d)
	}

	_, err = client.ListWAFRules(testZoneID, "123")
	assert.Error(t, err)
}

func TestListWAFRules(t *testing.T) {
	setup()
	defer teardown()

	testZoneID := "abcd123"

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method, "Expected method 'GET', got %s", r.Method)

		w.Header().Set("content-type", "application/json")
		// JSON data from: https://api.cloudflare.com/#waf-rules-properties
		fmt.Fprintf(w, `{
			"success": true,
			"errors": [],
			"messages": [],
			"result": [
				{
				"id": "f939de3be84e66e757adcdcb87908023",
				"description": "SQL injection prevention for SELECT statements",
				"priority": "5",
				"group": {
					"id": "de677e5818985db1285d0e80225f06e5",
					"name": "Project Honey Pot"
				},
				"package_id": "a25a9a7e9c00afc1fb2e0245519d725b",
				"allowed_modes": [
					"on",
					"off"
				],
				"mode": "on"
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

	mux.HandleFunc("/zones/"+testZoneID+"/firewall/waf/packages/a25a9a7e9c00afc1fb2e0245519d725b/rules", handler)

	want := []WAFRule{
		{
			ID:          "f939de3be84e66e757adcdcb87908023",
			Description: "SQL injection prevention for SELECT statements",
			Priority:    "5",
			PackageID:   "a25a9a7e9c00afc1fb2e0245519d725b",
			Group: struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{
				ID:   "de677e5818985db1285d0e80225f06e5",
				Name: "Project Honey Pot",
			},
			Mode:         "on",
			DefaultMode:  "",
			AllowedModes: []string{"on", "off"},
		},
	}

	d, err := client.ListWAFRules(testZoneID, "a25a9a7e9c00afc1fb2e0245519d725b")

	if assert.NoError(t, err) {
		assert.Equal(t, want, d)
	}

	_, err = client.ListWAFRules(testZoneID, "123")
	assert.Error(t, err)
}

func TestWAFRule(t *testing.T) {
	setup()
	defer teardown()

	testZoneID := "abcd123"

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method, "Expected method 'GET', got %s", r.Method)

		w.Header().Set("content-type", "application/json")
		// JSON data from: https://api.cloudflare.com/#waf-rules-properties
		fmt.Fprintf(w, `{
			"success": true,
			"errors": [],
			"messages": [],
			"result": {
				"id": "f939de3be84e66e757adcdcb87908023",
				"description": "SQL injection prevention for SELECT statements",
				"priority": "5",
				"group": {
					"id": "de677e5818985db1285d0e80225f06e5",
					"name": "Project Honey Pot"
				},
				"package_id": "a25a9a7e9c00afc1fb2e0245519d725b",
				"allowed_modes": [
					"on",
					"off"
				],
				"mode": "on"
			}
		}`)
	}

	mux.HandleFunc("/zones/"+testZoneID+"/firewall/waf/packages/a25a9a7e9c00afc1fb2e0245519d725b/rules/f939de3be84e66e757adcdcb87908023", handler)

	want := WAFRule{
		ID:          "f939de3be84e66e757adcdcb87908023",
		Description: "SQL injection prevention for SELECT statements",
		Priority:    "5",
		PackageID:   "a25a9a7e9c00afc1fb2e0245519d725b",
		Group: struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}{
			ID:   "de677e5818985db1285d0e80225f06e5",
			Name: "Project Honey Pot",
		},
		Mode:         "on",
		DefaultMode:  "",
		AllowedModes: []string{"on", "off"},
	}

	d, err := client.WAFRule(testZoneID, "a25a9a7e9c00afc1fb2e0245519d725b", "f939de3be84e66e757adcdcb87908023")

	if assert.NoError(t, err) {
		assert.Equal(t, want, d)
	}

	_, err = client.ListWAFRules(testZoneID, "123")
	assert.Error(t, err)
}

func TestUpdateWAFRule(t *testing.T) {
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

		assert.Equal(t, `{"mode":"on"}`, string(body), "Expected method '{\"mode\":\"on\"}', got %s", string(body))

		w.Header().Set("content-type", "application/json")
		// JSON data from: https://api.cloudflare.com/#waf-rules-properties
		fmt.Fprintf(w, `{
			"success": true,
			"errors": [],
			"messages": [],
			"result": {
				"id": "f939de3be84e66e757adcdcb87908023",
				"description": "SQL injection prevention for SELECT statements",
				"priority": "5",
				"group": {
					"id": "de677e5818985db1285d0e80225f06e5",
					"name": "Project Honey Pot"
				},
				"package_id": "a25a9a7e9c00afc1fb2e0245519d725b",
				"allowed_modes": [
					"on",
					"off"
				],
				"mode": "on"
			}
		}`)
	}

	mux.HandleFunc("/zones/"+testZoneID+"/firewall/waf/packages/a25a9a7e9c00afc1fb2e0245519d725b/rules/f939de3be84e66e757adcdcb87908023", handler)

	want := WAFRule{
		ID:          "f939de3be84e66e757adcdcb87908023",
		Description: "SQL injection prevention for SELECT statements",
		Priority:    "5",
		PackageID:   "a25a9a7e9c00afc1fb2e0245519d725b",
		Group: struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}{
			ID:   "de677e5818985db1285d0e80225f06e5",
			Name: "Project Honey Pot",
		},
		Mode:         "on",
		DefaultMode:  "",
		AllowedModes: []string{"on", "off"},
	}

	d, err := client.UpdateWAFRule(testZoneID, "a25a9a7e9c00afc1fb2e0245519d725b", "f939de3be84e66e757adcdcb87908023", "on")

	if assert.NoError(t, err) {
		assert.Equal(t, want, d)
	}

	_, err = client.ListWAFRules(testZoneID, "123")
	assert.Error(t, err)
}
