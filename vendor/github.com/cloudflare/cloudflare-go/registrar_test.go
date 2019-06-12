package cloudflare

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	registrarDomainPayload = `{
		"id": "ea95132c15732412d22c1476fa83f27a",
		"available": false,
		"supported_tld": true,
		"can_register": false,
		"transfer_in": {
			"unlock_domain": "ok",
			"disable_privacy": "ok",
			"enter_auth_code": "needed",
			"approve_transfer": "unknown",
			"accept_foa": "needed",
			"can_cancel_transfer": true
		},
		"current_registrar": "Cloudflare",
		"expires_at": "2019-08-28T23:59:59Z",
		"registry_statuses": "ok,serverTransferProhibited",
		"locked": false,
		"created_at": "2018-08-28T17:26:26Z",
		"updated_at": "2018-08-28T17:26:26Z",
		"registrant_contact": {
			"id": "ea95132c15732412d22c1476fa83f27a",
			"first_name": "John",
			"last_name": "Appleseed",
			"organization": "Cloudflare, Inc.",
			"address": "123 Sesame St.",
			"address2": "Suite 430",
			"city": "Austin",
			"state": "TX",
			"zip": "12345",
			"country": "US",
			"phone": "+1 123-123-1234",
			"email": "user@example.com",
			"fax": "123-867-5309"
		}
	}
`
)

var (
	createdAndModifiedTimestamp, _ = time.Parse(time.RFC3339, "2018-08-28T17:26:26Z")
	expiresAtTimestamp, _          = time.Parse(time.RFC3339, "2019-08-28T23:59:59Z")
	expectedRegistrarTransferIn    = RegistrarTransferIn{
		UnlockDomain:      "ok",
		DisablePrivacy:    "ok",
		EnterAuthCode:     "needed",
		ApproveTransfer:   "unknown",
		AcceptFoa:         "needed",
		CanCancelTransfer: true,
	}
	expectedRegistrarContact = RegistrantContact{
		ID:           "ea95132c15732412d22c1476fa83f27a",
		FirstName:    "John",
		LastName:     "Appleseed",
		Organization: "Cloudflare, Inc.",
		Address:      "123 Sesame St.",
		Address2:     "Suite 430",
		City:         "Austin",
		State:        "TX",
		Zip:          "12345",
		Country:      "US",
		Phone:        "+1 123-123-1234",
		Email:        "user@example.com",
		Fax:          "123-867-5309",
	}
	expectedRegistrarDomain = RegistrarDomain{
		ID:                "ea95132c15732412d22c1476fa83f27a",
		Available:         false,
		SupportedTLD:      true,
		CanRegister:       false,
		TransferIn:        expectedRegistrarTransferIn,
		CurrentRegistrar:  "Cloudflare",
		ExpiresAt:         expiresAtTimestamp,
		RegistryStatuses:  "ok,serverTransferProhibited",
		Locked:            false,
		CreatedAt:         createdAndModifiedTimestamp,
		UpdatedAt:         createdAndModifiedTimestamp,
		RegistrantContact: expectedRegistrarContact,
	}
)

func TestRegistrarDomain(t *testing.T) {
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
				"id": "ea95132c15732412d22c1476fa83f27a",
				"available": false,
				"supported_tld": true,
				"can_register": false,
				"transfer_in": {
					"unlock_domain": "ok",
					"disable_privacy": "ok",
					"enter_auth_code": "needed",
					"approve_transfer": "unknown",
					"accept_foa": "needed",
					"can_cancel_transfer": true
				},
				"current_registrar": "Cloudflare",
				"expires_at": "2019-08-28T23:59:59Z",
				"registry_statuses": "ok,serverTransferProhibited",
				"locked": false,
				"created_at": "2018-08-28T17:26:26Z",
				"updated_at": "2018-08-28T17:26:26Z",
				"registrant_contact": {
					"id": "ea95132c15732412d22c1476fa83f27a",
					"first_name": "John",
					"last_name": "Appleseed",
					"organization": "Cloudflare, Inc.",
					"address": "123 Sesame St.",
					"address2": "Suite 430",
					"city": "Austin",
					"state": "TX",
					"zip": "12345",
					"country": "US",
					"phone": "+1 123-123-1234",
					"email": "user@example.com",
					"fax": "123-867-5309"
				}
			}
		}
		`)
	}

	mux.HandleFunc("/accounts/01a7362d577a6c3019a474fd6f485823/registrar/domains/cloudflare.com", handler)

	actual, err := client.RegistrarDomain("01a7362d577a6c3019a474fd6f485823", "cloudflare.com")

	if assert.NoError(t, err) {
		assert.Equal(t, expectedRegistrarDomain, actual)
	}
}

func TestRegistrarDomains(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "POST", "Expected method 'POST', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
			"success": true,
			"errors": [],
			"messages": [],
			"result": [
				{
					"id": "ea95132c15732412d22c1476fa83f27a",
					"available": false,
					"supported_tld": true,
					"can_register": false,
					"transfer_in": {
						"unlock_domain": "ok",
						"disable_privacy": "ok",
						"enter_auth_code": "needed",
						"approve_transfer": "unknown",
						"accept_foa": "needed",
						"can_cancel_transfer": true
					},
					"current_registrar": "Cloudflare",
					"expires_at": "2019-08-28T23:59:59Z",
					"registry_statuses": "ok,serverTransferProhibited",
					"locked": false,
					"created_at": "2018-08-28T17:26:26Z",
					"updated_at": "2018-08-28T17:26:26Z",
					"registrant_contact": {
						"id": "ea95132c15732412d22c1476fa83f27a",
						"first_name": "John",
						"last_name": "Appleseed",
						"organization": "Cloudflare, Inc.",
						"address": "123 Sesame St.",
						"address2": "Suite 430",
						"city": "Austin",
						"state": "TX",
						"zip": "12345",
						"country": "US",
						"phone": "+1 123-123-1234",
						"email": "user@example.com",
						"fax": "123-867-5309"
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

	mux.HandleFunc("/accounts/01a7362d577a6c3019a474fd6f485823/registrar/domains", handler)

	actual, err := client.RegistrarDomains("01a7362d577a6c3019a474fd6f485823")

	if assert.NoError(t, err) {
		assert.Equal(t, []RegistrarDomain{expectedRegistrarDomain}, actual)
	}
}

func TestTransferRegistrarDomain(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "POST", "Expected method 'POST', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
			"success": true,
			"errors": [],
			"messages": [],
			"result": [
				{
					"id": "ea95132c15732412d22c1476fa83f27a",
					"available": false,
					"supported_tld": true,
					"can_register": false,
					"transfer_in": {
						"unlock_domain": "ok",
						"disable_privacy": "ok",
						"enter_auth_code": "needed",
						"approve_transfer": "unknown",
						"accept_foa": "needed",
						"can_cancel_transfer": true
					},
					"current_registrar": "Cloudflare",
					"expires_at": "2019-08-28T23:59:59Z",
					"registry_statuses": "ok,serverTransferProhibited",
					"locked": false,
					"created_at": "2018-08-28T17:26:26Z",
					"updated_at": "2018-08-28T17:26:26Z",
					"registrant_contact": {
						"id": "ea95132c15732412d22c1476fa83f27a",
						"first_name": "John",
						"last_name": "Appleseed",
						"organization": "Cloudflare, Inc.",
						"address": "123 Sesame St.",
						"address2": "Suite 430",
						"city": "Austin",
						"state": "TX",
						"zip": "12345",
						"country": "US",
						"phone": "+1 123-123-1234",
						"email": "user@example.com",
						"fax": "123-867-5309"
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

	mux.HandleFunc("/accounts/01a7362d577a6c3019a474fd6f485823/registrar/domains/cloudflare.com/transfer", handler)

	actual, err := client.TransferRegistrarDomain("01a7362d577a6c3019a474fd6f485823", "cloudflare.com")

	if assert.NoError(t, err) {
		assert.Equal(t, []RegistrarDomain{expectedRegistrarDomain}, actual)
	}
}

func TestCancelRegistrarDomainTransfer(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "POST", "Expected method 'POST', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
			"success": true,
			"errors": [],
			"messages": [],
			"result": [
				{
					"id": "ea95132c15732412d22c1476fa83f27a",
					"available": false,
					"supported_tld": true,
					"can_register": false,
					"transfer_in": {
						"unlock_domain": "ok",
						"disable_privacy": "ok",
						"enter_auth_code": "needed",
						"approve_transfer": "unknown",
						"accept_foa": "needed",
						"can_cancel_transfer": true
					},
					"current_registrar": "Cloudflare",
					"expires_at": "2019-08-28T23:59:59Z",
					"registry_statuses": "ok,serverTransferProhibited",
					"locked": false,
					"created_at": "2018-08-28T17:26:26Z",
					"updated_at": "2018-08-28T17:26:26Z",
					"registrant_contact": {
						"id": "ea95132c15732412d22c1476fa83f27a",
						"first_name": "John",
						"last_name": "Appleseed",
						"organization": "Cloudflare, Inc.",
						"address": "123 Sesame St.",
						"address2": "Suite 430",
						"city": "Austin",
						"state": "TX",
						"zip": "12345",
						"country": "US",
						"phone": "+1 123-123-1234",
						"email": "user@example.com",
						"fax": "123-867-5309"
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

	mux.HandleFunc("/accounts/01a7362d577a6c3019a474fd6f485823/registrar/domains/cloudflare.com/cancel_transfer", handler)

	actual, err := client.CancelRegistrarDomainTransfer("01a7362d577a6c3019a474fd6f485823", "cloudflare.com")

	if assert.NoError(t, err) {
		assert.Equal(t, []RegistrarDomain{expectedRegistrarDomain}, actual)
	}
}

func TestUpdateRegistrarDomain(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "PUT", "Expected method 'PUT', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
			"success": true,
			"errors": [],
			"messages": [],
			"result": {
				"id": "ea95132c15732412d22c1476fa83f27a",
				"available": false,
				"supported_tld": true,
				"can_register": false,
				"transfer_in": {
					"unlock_domain": "ok",
					"disable_privacy": "ok",
					"enter_auth_code": "needed",
					"approve_transfer": "unknown",
					"accept_foa": "needed",
					"can_cancel_transfer": true
				},
				"current_registrar": "Cloudflare",
				"expires_at": "2019-08-28T23:59:59Z",
				"registry_statuses": "ok,serverTransferProhibited",
				"locked": false,
				"created_at": "2018-08-28T17:26:26Z",
				"updated_at": "2018-08-28T17:26:26Z",
				"registrant_contact": {
					"id": "ea95132c15732412d22c1476fa83f27a",
					"first_name": "John",
					"last_name": "Appleseed",
					"organization": "Cloudflare, Inc.",
					"address": "123 Sesame St.",
					"address2": "Suite 430",
					"city": "Austin",
					"state": "TX",
					"zip": "12345",
					"country": "US",
					"phone": "+1 123-123-1234",
					"email": "user@example.com",
					"fax": "123-867-5309"
				}
			}
		}
		`)
	}

	mux.HandleFunc("/accounts/01a7362d577a6c3019a474fd6f485823/registrar/domains/cloudflare.com", handler)

	actual, err := client.UpdateRegistrarDomain(
		"01a7362d577a6c3019a474fd6f485823",
		"cloudflare.com",
		RegistrarDomainConfiguration{
			NameServers: []string{"ns1.cloudflare.com", "ns2.cloudflare.com"},
			Locked:      false,
		},
	)

	if assert.NoError(t, err) {
		assert.Equal(t, expectedRegistrarDomain, actual)
	}
}
