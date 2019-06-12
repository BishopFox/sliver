package cloudflare

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var timestamp, _ = time.Parse(time.RFC3339Nano, "2014-01-01T05:20:00.12345Z")
var expectedCustomPage = CustomPage{
	ID:             "basic_challenge",
	CreatedOn:      timestamp,
	ModifiedOn:     timestamp,
	URL:            "http://www.example.com",
	State:          "default",
	RequiredTokens: []string{"::CAPTCHA_BOX::"},
	PreviewTarget:  "preview:target",
	Description:    "Basic challenge",
}
var updatedCustomPage = CustomPage{
	ID:             "basic_challenge",
	CreatedOn:      timestamp,
	ModifiedOn:     timestamp,
	URL:            "https://mytestexample.com",
	State:          "customized",
	RequiredTokens: []string{"::CAPTCHA_BOX::"},
	PreviewTarget:  "preview:target",
	Description:    "Basic challenge",
}
var defaultCustomPage = CustomPage{
	ID:             "basic_challenge",
	CreatedOn:      timestamp,
	ModifiedOn:     timestamp,
	URL:            nil,
	State:          "default",
	RequiredTokens: []string{"::CAPTCHA_BOX::"},
	PreviewTarget:  "preview:target",
	Description:    "Basic challenge",
}

func TestCustomPagesWithoutZoneIDOrAccountID(t *testing.T) {
	_, err := client.CustomPages(&CustomPageOptions{})
	assert.EqualError(t, err, "either account ID or zone ID must be provided")
}

func TestCustomPagesWithZoneIDAndAccountID(t *testing.T) {
	_, err := client.CustomPages(&CustomPageOptions{ZoneID: "abc123", AccountID: "321cba"})
	assert.EqualError(t, err, "account ID and zone ID are mutually exclusive")
}

func TestCustomPagesForZone(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "GET", "Expected method 'GET', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `
		{
			"success": true,
			"errors": [],
			"messages": [],
			"result": [
				{
					"id": "basic_challenge",
					"created_on": "2014-01-01T05:20:00.12345Z",
					"modified_on": "2014-01-01T05:20:00.12345Z",
					"url": "http://www.example.com",
					"state": "default",
					"required_tokens": [
						"::CAPTCHA_BOX::"
					],
					"preview_target": "preview:target",
					"description": "Basic challenge"
				}
			],
			"result_info": {
				"page": 1,
				"per_page": 20,
				"count": 1,
				"total_count": 2000
			}
		}
		`)
	}

	mux.HandleFunc("/zones/d992d6de698eaf2d8cf8fd53b89b18a4/custom_pages", handler)
	want := []CustomPage{expectedCustomPage}

	pages, err := client.CustomPages(&CustomPageOptions{ZoneID: "d992d6de698eaf2d8cf8fd53b89b18a4"})

	if assert.NoError(t, err) {
		assert.Equal(t, want, pages)
	}
}

func TestCustomPagesForAccount(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "GET", "Expected method 'GET', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `
		{
			"success": true,
			"errors": [],
			"messages": [],
			"result": [
				{
					"id": "basic_challenge",
					"created_on": "2014-01-01T05:20:00.12345Z",
					"modified_on": "2014-01-01T05:20:00.12345Z",
					"url": "http://www.example.com",
					"state": "default",
					"required_tokens": [
						"::CAPTCHA_BOX::"
					],
					"preview_target": "preview:target",
					"description": "Basic challenge"
				}
			],
			"result_info": {
				"page": 1,
				"per_page": 20,
				"count": 1,
				"total_count": 2000
			}
		}
		`)
	}

	mux.HandleFunc("/accounts/01a7362d577a6c3019a474fd6f485823/custom_pages", handler)
	want := []CustomPage{expectedCustomPage}

	pages, err := client.CustomPages(&CustomPageOptions{AccountID: "01a7362d577a6c3019a474fd6f485823"})

	if assert.NoError(t, err) {
		assert.Equal(t, want, pages)
	}
}

func TestCustomPageForZone(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "GET", "Expected method 'GET', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `
		{
			"success": true,
			"errors": [],
			"messages": [],
			"result": {
				"id": "basic_challenge",
				"created_on": "2014-01-01T05:20:00.12345Z",
				"modified_on": "2014-01-01T05:20:00.12345Z",
				"url": "http://www.example.com",
				"state": "default",
				"required_tokens": [
					"::CAPTCHA_BOX::"
				],
				"preview_target": "preview:target",
				"description": "Basic challenge"
			},
			"result_info": {
				"page": 1,
				"per_page": 20,
				"count": 1,
				"total_count": 2000
			}
		}
		`)
	}

	mux.HandleFunc("/zones/d992d6de698eaf2d8cf8fd53b89b18a4/custom_pages/basic_challenge", handler)

	page, err := client.CustomPage(&CustomPageOptions{ZoneID: "d992d6de698eaf2d8cf8fd53b89b18a4"}, "basic_challenge")

	if assert.NoError(t, err) {
		assert.Equal(t, expectedCustomPage, page)
	}
}

func TestCustomPageForAccount(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "GET", "Expected method 'GET', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `
		{
			"success": true,
			"errors": [],
			"messages": [],
			"result": {
				"id": "basic_challenge",
				"created_on": "2014-01-01T05:20:00.12345Z",
				"modified_on": "2014-01-01T05:20:00.12345Z",
				"url": "http://www.example.com",
				"state": "default",
				"required_tokens": [
					"::CAPTCHA_BOX::"
				],
				"preview_target": "preview:target",
				"description": "Basic challenge"
			},
			"result_info": {
				"page": 1,
				"per_page": 20,
				"count": 1,
				"total_count": 2000
			}
		}
		`)
	}

	mux.HandleFunc("/accounts/01a7362d577a6c3019a474fd6f485823/custom_pages/basic_challenge", handler)

	page, err := client.CustomPage(&CustomPageOptions{AccountID: "01a7362d577a6c3019a474fd6f485823"}, "basic_challenge")

	if assert.NoError(t, err) {
		assert.Equal(t, expectedCustomPage, page)
	}
}

func TestUpdateCustomPagesForAccount(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "PUT", "Expected method 'PUT', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `
		{
			"success": true,
			"errors": [],
			"messages": [],
			"result": {
				"id": "basic_challenge",
				"created_on": "2014-01-01T05:20:00.12345Z",
				"modified_on": "2014-01-01T05:20:00.12345Z",
				"url": "https://mytestexample.com",
				"state": "customized",
				"required_tokens": [
					"::CAPTCHA_BOX::"
				],
				"preview_target": "preview:target",
				"description": "Basic challenge"
			},
			"result_info": {
				"page": 1,
				"per_page": 20,
				"count": 1,
				"total_count": 2000
			}
		}
		`)
	}

	mux.HandleFunc("/accounts/01a7362d577a6c3019a474fd6f485823/custom_pages/basic_challenge", handler)
	actual, err := client.UpdateCustomPage(
		&CustomPageOptions{AccountID: "01a7362d577a6c3019a474fd6f485823"},
		"basic_challenge",
		CustomPageParameters{URL: "https://mytestexample.com", State: "customized"},
	)

	if assert.NoError(t, err) {
		assert.Equal(t, updatedCustomPage, actual)
	}
}

func TestUpdateCustomPagesForZone(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "PUT", "Expected method 'PUT', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `
		{
			"success": true,
			"errors": [],
			"messages": [],
			"result": {
				"id": "basic_challenge",
				"created_on": "2014-01-01T05:20:00.12345Z",
				"modified_on": "2014-01-01T05:20:00.12345Z",
				"url": "https://mytestexample.com",
				"state": "customized",
				"required_tokens": [
					"::CAPTCHA_BOX::"
				],
				"preview_target": "preview:target",
				"description": "Basic challenge"
			},
			"result_info": {
				"page": 1,
				"per_page": 20,
				"count": 1,
				"total_count": 2000
			}
		}
		`)
	}

	mux.HandleFunc("/zones/d992d6de698eaf2d8cf8fd53b89b18a4/custom_pages/basic_challenge", handler)
	actual, err := client.UpdateCustomPage(
		&CustomPageOptions{ZoneID: "d992d6de698eaf2d8cf8fd53b89b18a4"},
		"basic_challenge",
		CustomPageParameters{URL: "https://mytestexample.com", State: "customized"},
	)

	if assert.NoError(t, err) {
		assert.Equal(t, updatedCustomPage, actual)
	}
}

func TestUpdateCustomPagesToDefault(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "PUT", "Expected method 'PUT', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `
		{
			"result":{
				"id":"basic_challenge",
				"description":"Basic challenge",
				"required_tokens":[
					"::CAPTCHA_BOX::"
				],
				"preview_target":"preview:target",
				"created_on": "2014-01-01T05:20:00.12345Z",
				"modified_on": "2014-01-01T05:20:00.12345Z",
				"url":null,
				"state":"default"
			},
			"success":true,
			"errors":[],
			"messages":[]
		}
		`)
	}

	mux.HandleFunc("/zones/d992d6de698eaf2d8cf8fd53b89b18a4/custom_pages/basic_challenge", handler)
	actual, err := client.UpdateCustomPage(
		&CustomPageOptions{ZoneID: "d992d6de698eaf2d8cf8fd53b89b18a4"},
		"basic_challenge",
		CustomPageParameters{URL: nil, State: "default"},
	)

	if assert.NoError(t, err) {
		assert.Equal(t, defaultCustomPage, actual)
	}
}
