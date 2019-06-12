package cloudflare

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

var expectedAccountRole = AccountRole{
	ID:          "3536bcfad5faccb999b47003c79917fb",
	Name:        "Account Administrator",
	Description: "Administrative access to the entire Account",
	Permissions: map[string]AccountRolePermission{
		"dns_records": {Read: true, Edit: true},
		"lb":          {Read: true, Edit: false},
	},
}

func TestAccountRoles(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "GET", "Expected method 'GET', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
			"success": true,
			"errors": [],
			"messages": [],
			"result": [
				{
					"id": "3536bcfad5faccb999b47003c79917fb",
					"name": "Account Administrator",
					"description": "Administrative access to the entire Account",
					"permissions": {
						"dns_records": {
							"read": true,
							"edit": true
						},
						"lb": {
							"read": true,
							"edit": false
						}
					}
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

	mux.HandleFunc("/accounts/01a7362d577a6c3019a474fd6f485823/roles", handler)
	want := []AccountRole{expectedAccountRole}

	actual, err := client.AccountRoles("01a7362d577a6c3019a474fd6f485823")

	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestAccountRole(t *testing.T) {
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
				"id": "3536bcfad5faccb999b47003c79917fb",
				"name": "Account Administrator",
				"description": "Administrative access to the entire Account",
				"permissions": {
					"dns_records": {
						"read": true,
						"edit": true
					},
					"lb": {
						"read": true,
						"edit": false
					}
				}
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

	mux.HandleFunc("/accounts/01a7362d577a6c3019a474fd6f485823/roles/3536bcfad5faccb999b47003c79917fb", handler)

	actual, err := client.AccountRole("01a7362d577a6c3019a474fd6f485823", "3536bcfad5faccb999b47003c79917fb")

	if assert.NoError(t, err) {
		assert.Equal(t, expectedAccountRole, actual)
	}
}
