package cloudflare

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

var firewallRulePageOpts = PaginationOptions{
	PerPage: 25,
	Page:    1,
}

func TestFirewallRules(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "GET", "Expected method 'GET', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
			"result":[
				{
					"id":"4ae338944d6143378c3cf05a7c77d983",
					"paused":false,
					"description":"allow API traffic without challenge",
					"action":"allow",
					"priority":null,
					"filter":{
						"id":"14217d7bd5ab435e84b1bd468bf4fb9f",
						"expression":"http.request.uri.path matches \"^/api/.*$\"",
						"paused":false,
						"description":"/api"
					}
				},
				{
					"id":"f2d427378e7542acb295380d352e2ebd",
					"paused":false,
					"description":"do not challenge login from office",
					"action":"allow",
					"priority":null,
					"filter":{
						"id":"b7ff25282d394be7b945e23c7106ce8a",
						"expression":"(http.request.uri.path ~ \"^.*/xmlrpc.php$\"",
						"paused":false,
						"description":"wordpress xmlrpc"
					}
				},
				{
					"id":"cbf4b7a5a2a24e59a03044d6d44ceb09",
					"paused":false,
					"description":"challenge login",
					"action":"challenge",
					"priority":null,
					"filter":{
						"id":"c218c536b2bd406f958f278cf0fa8c0f",
						"expression":"(http.request.uri.path ~ \"^.*/wp-login.php$\"",
						"paused":false,
						"description":"Login"
					}
				},
				{
					"id":"52161eb6af4241bb9d4b32394be72fdf",
					"paused":false,
					"description":"JS challenge site",
					"action":"js_challenge",
					"priority":null,
					"filter":{
						"id":"f2a64520581a4209aab12187a0081364",
						"expression":"not http.request.uri.path matches \"^/api/.*$\"",
						"paused":false,
						"description":"not /api"
					}
				}
			],
			"success":true,
			"errors":null,
			"messages":null,
			"result_info":{
				"page":1,
				"per_page":25,
				"count":4,
				"total_count":4
			}
		}
		`)
	}

	mux.HandleFunc("/zones/d56084adb405e0b7e32c52321bf07be6/firewall/rules", handler)
	want := []FirewallRule{
		{
			ID:          "4ae338944d6143378c3cf05a7c77d983",
			Paused:      false,
			Description: "allow API traffic without challenge",
			Action:      "allow",
			Priority:    nil,
			Filter: Filter{
				ID:          "14217d7bd5ab435e84b1bd468bf4fb9f",
				Expression:  "http.request.uri.path matches \"^/api/.*$\"",
				Paused:      false,
				Description: "/api",
			},
		},
		{
			ID:          "f2d427378e7542acb295380d352e2ebd",
			Paused:      false,
			Description: "do not challenge login from office",
			Action:      "allow",
			Priority:    nil,
			Filter: Filter{
				ID:          "b7ff25282d394be7b945e23c7106ce8a",
				Expression:  "(http.request.uri.path ~ \"^.*/xmlrpc.php$\"",
				Paused:      false,
				Description: "wordpress xmlrpc",
			},
		},
		{
			ID:          "cbf4b7a5a2a24e59a03044d6d44ceb09",
			Paused:      false,
			Description: "challenge login",
			Action:      "challenge",
			Priority:    nil,
			Filter: Filter{
				ID:          "c218c536b2bd406f958f278cf0fa8c0f",
				Expression:  "(http.request.uri.path ~ \"^.*/wp-login.php$\"",
				Paused:      false,
				Description: "Login",
			},
		},
		{
			ID:          "52161eb6af4241bb9d4b32394be72fdf",
			Paused:      false,
			Description: "JS challenge site",
			Action:      "js_challenge",
			Priority:    nil,
			Filter: Filter{
				ID:          "f2a64520581a4209aab12187a0081364",
				Expression:  "not http.request.uri.path matches \"^/api/.*$\"",
				Paused:      false,
				Description: "not /api",
			},
		},
	}

	actual, err := client.FirewallRules("d56084adb405e0b7e32c52321bf07be6", firewallRulePageOpts)

	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestFirewallRule(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "GET", "Expected method 'GET', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
			"result":{
				"id":"f2d427378e7542acb295380d352e2ebd",
				"paused":false,
				"description":"do not challenge login from office",
				"action":"allow",
				"priority":null,
				"filter":{
					"id":"b7ff25282d394be7b945e23c7106ce8a",
					"expression":"ip.src in {127.0.0.1} ~ \"^.*/login.php$\")",
					"paused":false,
					"description":"Login from office"
				}
			},
			"success":true,
			"errors":null,
			"messages":null
		}
		`)
	}

	mux.HandleFunc("/zones/d56084adb405e0b7e32c52321bf07be6/firewall/rules/f2d427378e7542acb295380d352e2ebd", handler)
	want := FirewallRule{
		ID:          "f2d427378e7542acb295380d352e2ebd",
		Paused:      false,
		Description: "do not challenge login from office",
		Action:      "allow",
		Priority:    nil,
		Filter: Filter{
			ID:          "b7ff25282d394be7b945e23c7106ce8a",
			Expression:  "ip.src in {127.0.0.1} ~ \"^.*/login.php$\")",
			Paused:      false,
			Description: "Login from office",
		},
	}

	actual, err := client.FirewallRule("d56084adb405e0b7e32c52321bf07be6", "f2d427378e7542acb295380d352e2ebd")

	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestCreateSingleFirewallRule(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "POST", "Expected method 'POST', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
			"result":[
				{
					"id":"f2d427378e7542acb295380d352e2ebd",
					"paused":false,
					"description":"do not challenge login from office",
					"action":"allow",
					"priority":null,
					"filter":{
						"id":"b7ff25282d394be7b945e23c7106ce8a",
						"expression":"ip.src in {127.0.0.0/24}",
						"paused":false,
						"description":"Login from office"
					}
				}
			],
			"success":true,
			"errors":null,
			"messages":null
		}
		`)
	}

	mux.HandleFunc("/zones/d56084adb405e0b7e32c52321bf07be6/firewall/rules", handler)
	want := []FirewallRule{
		{
			ID:          "f2d427378e7542acb295380d352e2ebd",
			Paused:      false,
			Description: "do not challenge login from office",
			Action:      "allow",
			Priority:    nil,
			Filter: Filter{
				ID:          "b7ff25282d394be7b945e23c7106ce8a",
				Expression:  "ip.src in {127.0.0.0/24}",
				Paused:      false,
				Description: "Login from office",
			},
		},
	}

	actual, err := client.CreateFirewallRules("d56084adb405e0b7e32c52321bf07be6", want)

	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestCreateMultipleFirewallRules(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "POST", "Expected method 'POST', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
			"result":[
				{
					"id":"f2d427378e7542acb295380d352e2ebd",
					"paused":false,
					"description":"do not challenge login from office",
					"action":"allow",
					"priority":null,
					"filter":{
						"id":"b7ff25282d394be7b945e23c7106ce8a",
						"expression":"ip.src in {127.0.0.0/24}",
						"paused":false,
						"description":"Login from office"
					}
				},
				{
					"id":"cbf4b7a5a2a24e59a03044d6d44ceb09",
					"paused":false,
					"description":"challenge login",
					"action":"challenge",
					"priority":null,
					"filter":{
						"id":"c218c536b2bd406f958f278cf0fa8c0f",
						"expression":"(http.request.uri.path ~ \"^.*/wp-login.php$\")",
						"paused":false,
						"description":"Login"
					}
				}
			],
			"success":true,
			"errors":null,
			"messages":null
		}
		`)
	}

	mux.HandleFunc("/zones/d56084adb405e0b7e32c52321bf07be6/firewall/rules", handler)
	want := []FirewallRule{
		{
			ID:          "f2d427378e7542acb295380d352e2ebd",
			Paused:      false,
			Description: "do not challenge login from office",
			Action:      "allow",
			Priority:    nil,
			Filter: Filter{
				ID:          "b7ff25282d394be7b945e23c7106ce8a",
				Expression:  "ip.src in {127.0.0.0/24}",
				Paused:      false,
				Description: "Login from office",
			},
		},
		{
			ID:          "cbf4b7a5a2a24e59a03044d6d44ceb09",
			Paused:      false,
			Description: "challenge login",
			Action:      "challenge",
			Priority:    nil,
			Filter: Filter{
				ID:          "c218c536b2bd406f958f278cf0fa8c0f",
				Expression:  "(http.request.uri.path ~ \"^.*/wp-login.php$\")",
				Paused:      false,
				Description: "Login",
			},
		},
	}

	actual, err := client.CreateFirewallRules("d56084adb405e0b7e32c52321bf07be6", want)

	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestUpdateFirewallRuleWithMissingID(t *testing.T) {
	setup()
	defer teardown()

	want := FirewallRule{
		ID:          "",
		Paused:      false,
		Description: "challenge site",
		Action:      "challenge",
		Priority:    nil,
		Filter: Filter{
			ID:          "f2a64520581a4209aab12187a0081364",
			Expression:  "not http.request.uri.path matches \"^/api/.*$\"",
			Paused:      false,
			Description: "not /api",
		},
	}

	_, err := client.UpdateFirewallRule("d56084adb405e0b7e32c52321bf07be6", want)
	assert.EqualError(t, err, "firewall rule ID cannot be empty")
}

func TestUpdateSingleFirewallRule(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "PUT", "Expected method 'PUT', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
			"result":{
				"id":"52161eb6af4241bb9d4b32394be72fdf",
				"paused":false,
				"description":"challenge site",
				"action":"challenge",
				"priority":null,
				"filter":{
					"id":"f2a64520581a4209aab12187a0081364",
					"expression":"not http.request.uri.path matches \"^/api/.*$\"",
					"paused":false,
					"description":"not /api"
				}
			},
			"success":true,
			"errors":null,
			"messages":null
		}
		`)
	}

	mux.HandleFunc("/zones/d56084adb405e0b7e32c52321bf07be6/firewall/rules/52161eb6af4241bb9d4b32394be72fdf", handler)
	want := FirewallRule{
		ID:          "52161eb6af4241bb9d4b32394be72fdf",
		Paused:      false,
		Description: "challenge site",
		Action:      "challenge",
		Priority:    nil,
		Filter: Filter{
			ID:          "f2a64520581a4209aab12187a0081364",
			Expression:  "not http.request.uri.path matches \"^/api/.*$\"",
			Paused:      false,
			Description: "not /api",
		},
	}

	actual, err := client.UpdateFirewallRule("d56084adb405e0b7e32c52321bf07be6", want)

	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestUpdateMultipleFirewallRules(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "PUT", "Expected method 'PUT', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
			"result":[
				{
					"id":"f2d427378e7542acb295380d352e2ebd",
					"paused":false,
					"description":"do not challenge login from office",
					"action":"allow",
					"priority":null,
					"filter":{
						"id":"b7ff25282d394be7b945e23c7106ce8a",
						"expression":"ip.src in {127.0.0.0/24}",
						"paused":false,
						"description":"Login from office"
					}
				},
				{
					"id":"cbf4b7a5a2a24e59a03044d6d44ceb09",
					"paused":false,
					"description":"challenge login",
					"action":"challenge",
					"priority":null,
					"filter":{
						"id":"c218c536b2bd406f958f278cf0fa8c0f",
						"expression":"(http.request.uri.path ~ \"^.*/wp-login.php$\")",
						"paused":false,
						"description":"Login"
					}
				}
			],
			"success":true,
			"errors":null,
			"messages":null
		}
		`)
	}

	mux.HandleFunc("/zones/d56084adb405e0b7e32c52321bf07be6/firewall/rules", handler)
	want := []FirewallRule{
		{
			ID:          "f2d427378e7542acb295380d352e2ebd",
			Paused:      false,
			Description: "do not challenge login from office",
			Action:      "allow",
			Priority:    nil,
			Filter: Filter{
				ID:          "b7ff25282d394be7b945e23c7106ce8a",
				Expression:  "ip.src in {127.0.0.0/24}",
				Paused:      false,
				Description: "Login from office",
			},
		},
		{
			ID:          "cbf4b7a5a2a24e59a03044d6d44ceb09",
			Paused:      false,
			Description: "challenge login",
			Action:      "challenge",
			Priority:    nil,
			Filter: Filter{
				ID:          "c218c536b2bd406f958f278cf0fa8c0f",
				Expression:  "(http.request.uri.path ~ \"^.*/wp-login.php$\")",
				Paused:      false,
				Description: "Login",
			},
		},
	}

	actual, err := client.UpdateFirewallRules("d56084adb405e0b7e32c52321bf07be6", want)

	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestDeleteSingleFirewallRule(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "DELETE", "Expected method 'DELETE', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
			"result": [],
			"success": true,
			"errors": null,
			"messages": null
		}
		`)
	}

	mux.HandleFunc("/zones/d56084adb405e0b7e32c52321bf07be6/firewall/rules/f2d427378e7542acb295380d352e2ebd", handler)

	err := client.DeleteFirewallRule("d56084adb405e0b7e32c52321bf07be6", "f2d427378e7542acb295380d352e2ebd")
	assert.NoError(t, err)
}

func TestDeleteFirewallRuleWithMissingID(t *testing.T) {
	setup()
	defer teardown()

	err := client.DeleteFirewallRule("d56084adb405e0b7e32c52321bf07be6", "")
	assert.EqualError(t, err, "firewall rule ID cannot be empty")
}
