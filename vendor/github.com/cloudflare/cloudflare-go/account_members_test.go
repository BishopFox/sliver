package cloudflare

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

var expectedAccountMemberStruct = AccountMember{
	ID:   "4536bcfad5faccb111b47003c79917fa",
	Code: "05dd05cce12bbed97c0d87cd78e89bc2fd41a6cee72f27f6fc84af2e45c0fac0",
	User: AccountMemberUserDetails{
		ID:                             "7c5dae5552338874e5053f2534d2767a",
		FirstName:                      "John",
		LastName:                       "Appleseed",
		Email:                          "user@example.com",
		TwoFactorAuthenticationEnabled: false,
	},
	Status: "accepted",
	Roles: []AccountRole{
		{
			ID:          "3536bcfad5faccb999b47003c79917fb",
			Name:        "Account Administrator",
			Description: "Administrative access to the entire Account",
			Permissions: map[string]AccountRolePermission{
				"analytics": {Read: true, Edit: true},
				"billing":   {Read: true, Edit: false},
			},
		},
	},
}

var expectedNewAccountMemberStruct = AccountMember{
	ID:   "4536bcfad5faccb111b47003c79917fa",
	Code: "05dd05cce12bbed97c0d87cd78e89bc2fd41a6cee72f27f6fc84af2e45c0fac0",
	User: AccountMemberUserDetails{
		Email:                          "user@example.com",
		TwoFactorAuthenticationEnabled: false,
	},
	Status: "pending",
	Roles: []AccountRole{
		{
			ID:          "3536bcfad5faccb999b47003c79917fb",
			Name:        "Account Administrator",
			Description: "Administrative access to the entire Account",
			Permissions: map[string]AccountRolePermission{
				"analytics": {Read: true, Edit: true},
				"billing":   {Read: true, Edit: true},
			},
		},
	},
}

var newUpdatedAccountMemberStruct = AccountMember{
	ID:   "4536bcfad5faccb111b47003c79917fa",
	Code: "05dd05cce12bbed97c0d87cd78e89bc2fd41a6cee72f27f6fc84af2e45c0fac0",
	User: AccountMemberUserDetails{
		ID:                             "7c5dae5552338874e5053f2534d2767a",
		FirstName:                      "John",
		LastName:                       "Appleseeds",
		Email:                          "new-user@example.com",
		TwoFactorAuthenticationEnabled: false,
	},
	Status: "accepted",
	Roles: []AccountRole{
		{
			ID:          "3536bcfad5faccb999b47003c79917fb",
			Name:        "Account Administrator",
			Description: "Administrative access to the entire Account",
			Permissions: map[string]AccountRolePermission{
				"analytics": {Read: true, Edit: true},
				"billing":   {Read: true, Edit: true},
			},
		},
	},
}

func TestAccountMembers(t *testing.T) {
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
					"id": "4536bcfad5faccb111b47003c79917fa",
					"code": "05dd05cce12bbed97c0d87cd78e89bc2fd41a6cee72f27f6fc84af2e45c0fac0",
					"user": {
						"id": "7c5dae5552338874e5053f2534d2767a",
						"first_name": "John",
						"last_name": "Appleseed",
						"email": "user@example.com",
						"two_factor_authentication_enabled": false
					},
					"status": "accepted",
					"roles": [
						{
							"id": "3536bcfad5faccb999b47003c79917fb",
							"name": "Account Administrator",
							"description": "Administrative access to the entire Account",
							"permissions": {
								"analytics": {
									"read": true,
									"edit": true
								},
								"billing": {
									"read": true,
									"edit": false
								}
							}
						}
					]
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

	mux.HandleFunc("/accounts/01a7362d577a6c3019a474fd6f485823/members", handler)
	want := []AccountMember{expectedAccountMemberStruct}

	actual, _, err := client.AccountMembers("01a7362d577a6c3019a474fd6f485823", PaginationOptions{})

	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestAccountMembersWithoutAccountID(t *testing.T) {
	setup()
	defer teardown()

	_, _, err := client.AccountMembers("", PaginationOptions{})

	if assert.Error(t, err) {
		assert.Equal(t, err.Error(), errMissingAccountID)
	}
}

func TestCreateAccountMember(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "POST", "Expected method 'POST', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
			"success": true,
			"errors": [],
			"messages": [],
			"result": {
				"id": "4536bcfad5faccb111b47003c79917fa",
				"code": "05dd05cce12bbed97c0d87cd78e89bc2fd41a6cee72f27f6fc84af2e45c0fac0",
				"user": {
					"id": null,
					"first_name": null,
					"last_name": null,
					"email": "user@example.com",
					"two_factor_authentication_enabled": false
				},
				"status": "pending",
				"roles": [{
					"id": "3536bcfad5faccb999b47003c79917fb",
					"name": "Account Administrator",
					"description": "Administrative access to the entire Account",
					"permissions": {
						"analytics": {
							"read": true,
							"edit": true
						},
						"billing": {
							"read": true,
							"edit": true
						}
					}
				}]
			}
		}
		`)
	}

	mux.HandleFunc("/accounts/01a7362d577a6c3019a474fd6f485823/members", handler)

	actual, err := client.CreateAccountMember(
		"01a7362d577a6c3019a474fd6f485823",
		"user@example.com",
		[]string{"3536bcfad5faccb999b47003c79917fb"})

	if assert.NoError(t, err) {
		assert.Equal(t, expectedNewAccountMemberStruct, actual)
	}
}

func TestCreateAccountMemberWithoutAccountID(t *testing.T) {
	setup()
	defer teardown()

	_, err := client.CreateAccountMember(
		"",
		"user@example.com",
		[]string{"3536bcfad5faccb999b47003c79917fb"})

	if assert.Error(t, err) {
		assert.Equal(t, err.Error(), errMissingAccountID)
	}
}

func TestUpdateAccountMember(t *testing.T) {
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
				"id": "4536bcfad5faccb111b47003c79917fa",
				"code": "05dd05cce12bbed97c0d87cd78e89bc2fd41a6cee72f27f6fc84af2e45c0fac0",
				"user": {
					"id": "7c5dae5552338874e5053f2534d2767a",
					"first_name": "John",
					"last_name": "Appleseeds",
					"email": "new-user@example.com",
					"two_factor_authentication_enabled": false
				},
				"status": "accepted",
				"roles": [{
					"id": "3536bcfad5faccb999b47003c79917fb",
					"name": "Account Administrator",
					"description": "Administrative access to the entire Account",
					"permissions": {
						"analytics": {
							"read": true,
							"edit": true
						},
						"billing": {
							"read": true,
							"edit": true
						}
					}
				}]
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

	mux.HandleFunc("/accounts/01a7362d577a6c3019a474fd6f485823/members/4536bcfad5faccb111b47003c79917fa", handler)

	actual, err := client.UpdateAccountMember(
		"01a7362d577a6c3019a474fd6f485823",
		"4536bcfad5faccb111b47003c79917fa",
		newUpdatedAccountMemberStruct,
	)

	if assert.NoError(t, err) {
		assert.Equal(t, newUpdatedAccountMemberStruct, actual)
	}
}

func TestUpdateAccountMemberWithoutAccountID(t *testing.T) {
	setup()
	defer teardown()

	_, err := client.UpdateAccountMember(
		"",
		"4536bcfad5faccb111b47003c79917fa",
		newUpdatedAccountMemberStruct,
	)

	if assert.Error(t, err) {
		assert.Equal(t, err.Error(), errMissingAccountID)
	}
}

func TestAccountMember(t *testing.T) {
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
				"id": "4536bcfad5faccb111b47003c79917fa",
				"code": "05dd05cce12bbed97c0d87cd78e89bc2fd41a6cee72f27f6fc84af2e45c0fac0",
				"user": {
					"id": "7c5dae5552338874e5053f2534d2767a",
					"first_name": "John",
					"last_name": "Appleseed",
					"email": "user@example.com",
					"two_factor_authentication_enabled": false
				},
				"status": "accepted",
				"roles": [
					{
						"id": "3536bcfad5faccb999b47003c79917fb",
						"name": "Account Administrator",
						"description": "Administrative access to the entire Account",
						"permissions": {
							"analytics": {
								"read": true,
								"edit": true
							},
							"billing": {
								"read": true,
								"edit": false
							}
						}
					}
				]
			}
		}
		`)
	}

	mux.HandleFunc("/accounts/01a7362d577a6c3019a474fd6f485823/members/4536bcfad5faccb111b47003c79917fa", handler)

	actual, err := client.AccountMember("01a7362d577a6c3019a474fd6f485823", "4536bcfad5faccb111b47003c79917fa")

	if assert.NoError(t, err) {
		assert.Equal(t, expectedAccountMemberStruct, actual)
	}
}

func TestAccountMemberWithoutAccountID(t *testing.T) {
	setup()
	defer teardown()

	_, err := client.AccountMember("", "4536bcfad5faccb111b47003c79917fa")

	if assert.Error(t, err) {
		assert.Equal(t, err.Error(), errMissingAccountID)
	}
}
