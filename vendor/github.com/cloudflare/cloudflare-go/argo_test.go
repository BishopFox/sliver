package cloudflare

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var argoTimestamp, _ = time.Parse(time.RFC3339Nano, "2019-02-20T22:37:07.107449Z")

func TestArgoSmartRouting(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "GET", "Expected method 'GET', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
			"success": true,
			"errors": [],
			"messages": [],
			"result": {
				"id": "smart_routing",
				"value": "on",
				"editable": true,
				"modified_on": "2019-02-20T22:37:07.107449Z"
			}
		}
		`)
	}

	mux.HandleFunc("/zones/01a7362d577a6c3019a474fd6f485823/argo/smart_routing", handler)
	want := ArgoFeatureSetting{
		ID:         "smart_routing",
		Value:      "on",
		Editable:   true,
		ModifiedOn: argoTimestamp,
	}

	actual, err := client.ArgoSmartRouting("01a7362d577a6c3019a474fd6f485823")

	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestUpdateArgoSmartRouting(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "PATCH", "Expected method 'PATCH', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
			"success": true,
			"errors": [],
			"messages": [],
			"result": {
				"id": "smart_routing",
				"value": "off",
				"editable": true,
				"modified_on": "2019-02-20T22:37:07.107449Z"
			}
		}
		`)
	}

	mux.HandleFunc("/zones/01a7362d577a6c3019a474fd6f485823/argo/smart_routing", handler)
	want := ArgoFeatureSetting{
		ID:         "smart_routing",
		Value:      "off",
		Editable:   true,
		ModifiedOn: argoTimestamp,
	}

	actual, err := client.UpdateArgoSmartRouting("01a7362d577a6c3019a474fd6f485823", "off")

	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestUpdateArgoSmartRoutingWithInvalidValue(t *testing.T) {
	setup()
	defer teardown()

	_, err := client.UpdateArgoSmartRouting("01a7362d577a6c3019a474fd6f485823", "notreal")

	if assert.Error(t, err) {
		assert.Equal(t, "invalid setting value 'notreal'. must be 'on' or 'off'", err.Error())
	}
}

func TestArgoTieredCaching(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "GET", "Expected method 'GET', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
			"success": true,
			"errors": [],
			"messages": [],
			"result": {
				"id": "tiered_caching",
				"value": "on",
				"editable": true,
				"modified_on": "2019-02-20T22:37:07.107449Z"
			}
		}
		`)
	}

	mux.HandleFunc("/zones/01a7362d577a6c3019a474fd6f485823/argo/tiered_caching", handler)
	want := ArgoFeatureSetting{
		ID:         "tiered_caching",
		Value:      "on",
		Editable:   true,
		ModifiedOn: argoTimestamp,
	}

	actual, err := client.ArgoTieredCaching("01a7362d577a6c3019a474fd6f485823")

	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestUpdateArgoTieredCaching(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "PATCH", "Expected method 'PATCH', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
			"success": true,
			"errors": [],
			"messages": [],
			"result": {
				"id": "tiered_caching",
				"value": "off",
				"editable": true,
				"modified_on": "2019-02-20T22:37:07.107449Z"
			}
		}
		`)
	}

	mux.HandleFunc("/zones/01a7362d577a6c3019a474fd6f485823/argo/tiered_caching", handler)
	want := ArgoFeatureSetting{
		ID:         "tiered_caching",
		Value:      "off",
		Editable:   true,
		ModifiedOn: argoTimestamp,
	}

	actual, err := client.UpdateArgoTieredCaching("01a7362d577a6c3019a474fd6f485823", "off")

	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestUpdateArgoTieredCachingWithInvalidValue(t *testing.T) {
	setup()
	defer teardown()

	_, err := client.UpdateArgoTieredCaching("01a7362d577a6c3019a474fd6f485823", "notreal")

	if assert.Error(t, err) {
		assert.Equal(t, "invalid setting value 'notreal'. must be 'on' or 'off'", err.Error())
	}
}
