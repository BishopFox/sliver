package cloudflare

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateLoadBalancerPool(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "POST", "Expected method 'POST', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		b, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if assert.NoError(t, err) {
			assert.JSONEq(t, `{
              "description": "Primary data center - Provider XYZ",
              "name": "primary-dc-1",
              "enabled": true,
              "monitor": "f1aba936b94213e5b8dca0c0dbf1f9cc",
              "origins": [
                {
                  "name": "app-server-1",
                  "address": "0.0.0.0",
                  "enabled": true,
                  "weight": 1
                }
              ],
              "notification_email": "someone@example.com",
              "check_regions": [
                "WEU"
              ]
						}`, string(b))
		}
		fmt.Fprint(w, `{
            "success": true,
            "errors": [],
            "messages": [],
            "result": {
              "id": "17b5962d775c646f3f9725cbc7a53df4",
              "created_on": "2014-01-01T05:20:00.12345Z",
              "modified_on": "2014-02-01T05:20:00.12345Z",
              "description": "Primary data center - Provider XYZ",
              "name": "primary-dc-1",
              "enabled": true,
              "minimum_origins": 1,
              "monitor": "f1aba936b94213e5b8dca0c0dbf1f9cc",
              "origins": [
                {
                  "name": "app-server-1",
                  "address": "0.0.0.0",
                  "enabled": true,
                  "weight": 1
                }
              ],
              "notification_email": "someone@example.com",
              "check_regions": [
                "WEU"
              ]
            }
        }`)
	}

	mux.HandleFunc("/user/load_balancers/pools", handler)
	createdOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00.12345Z")
	modifiedOn, _ := time.Parse(time.RFC3339, "2014-02-01T05:20:00.12345Z")
	want := LoadBalancerPool{
		ID:             "17b5962d775c646f3f9725cbc7a53df4",
		CreatedOn:      &createdOn,
		ModifiedOn:     &modifiedOn,
		Description:    "Primary data center - Provider XYZ",
		Name:           "primary-dc-1",
		Enabled:        true,
		MinimumOrigins: 1,
		Monitor:        "f1aba936b94213e5b8dca0c0dbf1f9cc",
		Origins: []LoadBalancerOrigin{
			{
				Name:    "app-server-1",
				Address: "0.0.0.0",
				Enabled: true,
				Weight:  1,
			},
		},
		NotificationEmail: "someone@example.com",
		CheckRegions: []string{
			"WEU",
		},
	}
	request := LoadBalancerPool{
		Description: "Primary data center - Provider XYZ",
		Name:        "primary-dc-1",
		Enabled:     true,
		Monitor:     "f1aba936b94213e5b8dca0c0dbf1f9cc",
		Origins: []LoadBalancerOrigin{
			{
				Name:    "app-server-1",
				Address: "0.0.0.0",
				Enabled: true,
				Weight:  1,
			},
		},
		NotificationEmail: "someone@example.com",
		CheckRegions: []string{
			"WEU",
		},
	}

	actual, err := client.CreateLoadBalancerPool(request)
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestListLoadBalancerPools(t *testing.T) {
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
                    "id": "17b5962d775c646f3f9725cbc7a53df4",
                    "created_on": "2014-01-01T05:20:00.12345Z",
                    "modified_on": "2014-02-01T05:20:00.12345Z",
                    "description": "Primary data center - Provider XYZ",
                    "name": "primary-dc-1",
                    "enabled": true,
                    "monitor": "f1aba936b94213e5b8dca0c0dbf1f9cc",
                    "origins": [
                      {
                        "name": "app-server-1",
                        "address": "0.0.0.0",
                        "enabled": true,
                        "weight": 1
                      }
                    ],
                    "notification_email": "someone@example.com"
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

	mux.HandleFunc("/user/load_balancers/pools", handler)
	createdOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00.12345Z")
	modifiedOn, _ := time.Parse(time.RFC3339, "2014-02-01T05:20:00.12345Z")
	want := []LoadBalancerPool{
		{
			ID:          "17b5962d775c646f3f9725cbc7a53df4",
			CreatedOn:   &createdOn,
			ModifiedOn:  &modifiedOn,
			Description: "Primary data center - Provider XYZ",
			Name:        "primary-dc-1",
			Enabled:     true,
			Monitor:     "f1aba936b94213e5b8dca0c0dbf1f9cc",
			Origins: []LoadBalancerOrigin{
				{
					Name:    "app-server-1",
					Address: "0.0.0.0",
					Enabled: true,
					Weight:  1,
				},
			},
			NotificationEmail: "someone@example.com",
		},
	}

	actual, err := client.ListLoadBalancerPools()
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestLoadBalancerPoolDetails(t *testing.T) {
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
              "id": "17b5962d775c646f3f9725cbc7a53df4",
              "created_on": "2014-01-01T05:20:00.12345Z",
              "modified_on": "2014-02-01T05:20:00.12345Z",
              "description": "Primary data center - Provider XYZ",
              "name": "primary-dc-1",
              "enabled": true,
              "monitor": "f1aba936b94213e5b8dca0c0dbf1f9cc",
              "origins": [
                {
                  "name": "app-server-1",
                  "address": "0.0.0.0",
                  "enabled": true,
                  "weight": 1
                }
              ],
              "notification_email": "someone@example.com"
            }
        }`)
	}

	mux.HandleFunc("/user/load_balancers/pools/17b5962d775c646f3f9725cbc7a53df4", handler)
	createdOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00.12345Z")
	modifiedOn, _ := time.Parse(time.RFC3339, "2014-02-01T05:20:00.12345Z")
	want := LoadBalancerPool{
		ID:          "17b5962d775c646f3f9725cbc7a53df4",
		CreatedOn:   &createdOn,
		ModifiedOn:  &modifiedOn,
		Description: "Primary data center - Provider XYZ",
		Name:        "primary-dc-1",
		Enabled:     true,
		Monitor:     "f1aba936b94213e5b8dca0c0dbf1f9cc",
		Origins: []LoadBalancerOrigin{
			{
				Name:    "app-server-1",
				Address: "0.0.0.0",
				Enabled: true,
				Weight:  1,
			},
		},
		NotificationEmail: "someone@example.com",
	}

	actual, err := client.LoadBalancerPoolDetails("17b5962d775c646f3f9725cbc7a53df4")
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}

	_, err = client.LoadBalancerPoolDetails("bar")
	assert.Error(t, err)
}

func TestDeleteLoadBalancerPool(t *testing.T) {
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
              "id": "17b5962d775c646f3f9725cbc7a53df4"
            }
        }`)
	}

	mux.HandleFunc("/user/load_balancers/pools/17b5962d775c646f3f9725cbc7a53df4", handler)
	assert.NoError(t, client.DeleteLoadBalancerPool("17b5962d775c646f3f9725cbc7a53df4"))
	assert.Error(t, client.DeleteLoadBalancerPool("bar"))
}

func TestModifyLoadBalancerPool(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "PUT", "Expected method 'PUT', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		b, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if assert.NoError(t, err) {
			assert.JSONEq(t, `{
              "id": "17b5962d775c646f3f9725cbc7a53df4",
              "description": "Primary data center - Provider XYZZY",
              "name": "primary-dc-2",
              "enabled": false,
              "origins": [
                {
                  "name": "app-server-2",
                  "address": "0.0.0.1",
                  "enabled": false,
                  "weight": 1
                }
              ],
              "notification_email": "nobody@example.com",
              "check_regions": [
                "WEU"
              ]
						}`, string(b))
		}
		fmt.Fprint(w, `{
            "success": true,
            "errors": [],
            "messages": [],
            "result": {
              "id": "17b5962d775c646f3f9725cbc7a53df4",
              "created_on": "2014-01-01T05:20:00.12345Z",
              "modified_on": "2017-02-01T05:20:00.12345Z",
              "description": "Primary data center - Provider XYZZY",
              "name": "primary-dc-2",
              "enabled": false,
              "origins": [
                {
                  "name": "app-server-2",
                  "address": "0.0.0.1",
                  "enabled": false,
                  "weight": 1
                }
              ],
              "notification_email": "nobody@example.com",
              "check_regions": [
                "WEU"
              ]
            }
        }`)
	}

	mux.HandleFunc("/user/load_balancers/pools/17b5962d775c646f3f9725cbc7a53df4", handler)
	createdOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00.12345Z")
	modifiedOn, _ := time.Parse(time.RFC3339, "2017-02-01T05:20:00.12345Z")
	want := LoadBalancerPool{
		ID:          "17b5962d775c646f3f9725cbc7a53df4",
		CreatedOn:   &createdOn,
		ModifiedOn:  &modifiedOn,
		Description: "Primary data center - Provider XYZZY",
		Name:        "primary-dc-2",
		Enabled:     false,
		Origins: []LoadBalancerOrigin{
			{
				Name:    "app-server-2",
				Address: "0.0.0.1",
				Enabled: false,
				Weight:  1,
			},
		},
		NotificationEmail: "nobody@example.com",
		CheckRegions: []string{
			"WEU",
		},
	}
	request := LoadBalancerPool{
		ID:          "17b5962d775c646f3f9725cbc7a53df4",
		Description: "Primary data center - Provider XYZZY",
		Name:        "primary-dc-2",
		Enabled:     false,
		Origins: []LoadBalancerOrigin{
			{
				Name:    "app-server-2",
				Address: "0.0.0.1",
				Enabled: false,
				Weight:  1,
			},
		},
		NotificationEmail: "nobody@example.com",
		CheckRegions: []string{
			"WEU",
		},
	}

	actual, err := client.ModifyLoadBalancerPool(request)
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestCreateLoadBalancerMonitor(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "POST", "Expected method 'POST', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		b, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if assert.NoError(t, err) {
			assert.JSONEq(t, `{
              "type": "https",
              "description": "Login page monitor",
              "method": "GET",
              "path": "/health",
              "header": {
                "Host": [
                  "example.com"
                ],
                "X-App-ID": [
                  "abc123"
                ]
              },
              "timeout": 3,
              "retries": 0,
              "interval": 90,
              "expected_body": "alive",
              "expected_codes": "2xx",
              "follow_redirects": true,
              "allow_insecure": true,
              "probe_zone": ""
						}`, string(b))
		}
		fmt.Fprint(w, `{
            "success": true,
            "errors": [],
            "messages": [],
            "result": {
                "id": "f1aba936b94213e5b8dca0c0dbf1f9cc",
                "created_on": "2014-01-01T05:20:00.12345Z",
                "modified_on": "2014-02-01T05:20:00.12345Z",
                "type": "https",
                "description": "Login page monitor",
                "method": "GET",
                "path": "/health",
                "header": {
                  "Host": [
                    "example.com"
                  ],
                  "X-App-ID": [
                    "abc123"
                  ]
                },
                "timeout": 3,
                "retries": 0,
                "interval": 90,
                "expected_body": "alive",
                "expected_codes": "2xx",
                "follow_redirects": true,
                "allow_insecure": true,
                "probe_zone": ""
            }
        }`)
	}

	mux.HandleFunc("/user/load_balancers/monitors", handler)
	createdOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00.12345Z")
	modifiedOn, _ := time.Parse(time.RFC3339, "2014-02-01T05:20:00.12345Z")
	want := LoadBalancerMonitor{
		ID:          "f1aba936b94213e5b8dca0c0dbf1f9cc",
		CreatedOn:   &createdOn,
		ModifiedOn:  &modifiedOn,
		Type:        "https",
		Description: "Login page monitor",
		Method:      "GET",
		Path:        "/health",
		Header: map[string][]string{
			"Host":     {"example.com"},
			"X-App-ID": {"abc123"},
		},
		Timeout:       3,
		Retries:       0,
		Interval:      90,
		ExpectedBody:  "alive",
		ExpectedCodes: "2xx",

		FollowRedirects: true,
		AllowInsecure:   true,
	}
	request := LoadBalancerMonitor{
		Type:        "https",
		Description: "Login page monitor",
		Method:      "GET",
		Path:        "/health",
		Header: map[string][]string{
			"Host":     {"example.com"},
			"X-App-ID": {"abc123"},
		},
		Timeout:       3,
		Retries:       0,
		Interval:      90,
		ExpectedBody:  "alive",
		ExpectedCodes: "2xx",

		FollowRedirects: true,
		AllowInsecure:   true,
	}

	actual, err := client.CreateLoadBalancerMonitor(request)
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestListLoadBalancerMonitors(t *testing.T) {
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
                    "id": "f1aba936b94213e5b8dca0c0dbf1f9cc",
                    "created_on": "2014-01-01T05:20:00.12345Z",
                    "modified_on": "2014-02-01T05:20:00.12345Z",
                    "type": "https",
                    "description": "Login page monitor",
                    "method": "GET",
                    "path": "/health",
                    "header": {
                      "Host": [
                        "example.com"
                      ],
                      "X-App-ID": [
                        "abc123"
                      ]
                    },
                    "timeout": 3,
                    "retries": 0,
                    "interval": 90,
                    "expected_body": "alive",
                    "expected_codes": "2xx"
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

	mux.HandleFunc("/user/load_balancers/monitors", handler)
	createdOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00.12345Z")
	modifiedOn, _ := time.Parse(time.RFC3339, "2014-02-01T05:20:00.12345Z")
	want := []LoadBalancerMonitor{
		{
			ID:          "f1aba936b94213e5b8dca0c0dbf1f9cc",
			CreatedOn:   &createdOn,
			ModifiedOn:  &modifiedOn,
			Type:        "https",
			Description: "Login page monitor",
			Method:      "GET",
			Path:        "/health",
			Header: map[string][]string{
				"Host":     {"example.com"},
				"X-App-ID": {"abc123"},
			},
			Timeout:       3,
			Retries:       0,
			Interval:      90,
			ExpectedBody:  "alive",
			ExpectedCodes: "2xx",
		},
	}

	actual, err := client.ListLoadBalancerMonitors()
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestLoadBalancerMonitorDetails(t *testing.T) {
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
                "id": "f1aba936b94213e5b8dca0c0dbf1f9cc",
                "created_on": "2014-01-01T05:20:00.12345Z",
                "modified_on": "2014-02-01T05:20:00.12345Z",
                "type": "https",
                "description": "Login page monitor",
                "method": "GET",
                "path": "/health",
                "header": {
                  "Host": [
                    "example.com"
                  ],
                  "X-App-ID": [
                    "abc123"
                  ]
                },
                "timeout": 3,
                "retries": 0,
                "interval": 90,
                "expected_body": "alive",
                "expected_codes": "2xx",
                "follow_redirects": true,
                "allow_insecure": true,
                "probe_zone": ""
            }
        }`)
	}

	mux.HandleFunc("/user/load_balancers/monitors/f1aba936b94213e5b8dca0c0dbf1f9cc", handler)
	createdOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00.12345Z")
	modifiedOn, _ := time.Parse(time.RFC3339, "2014-02-01T05:20:00.12345Z")
	want := LoadBalancerMonitor{
		ID:          "f1aba936b94213e5b8dca0c0dbf1f9cc",
		CreatedOn:   &createdOn,
		ModifiedOn:  &modifiedOn,
		Type:        "https",
		Description: "Login page monitor",
		Method:      "GET",
		Path:        "/health",
		Header: map[string][]string{
			"Host":     {"example.com"},
			"X-App-ID": {"abc123"},
		},
		Timeout:       3,
		Retries:       0,
		Interval:      90,
		ExpectedBody:  "alive",
		ExpectedCodes: "2xx",

		FollowRedirects: true,
		AllowInsecure:   true,
	}

	actual, err := client.LoadBalancerMonitorDetails("f1aba936b94213e5b8dca0c0dbf1f9cc")
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}

	_, err = client.LoadBalancerMonitorDetails("bar")
	assert.Error(t, err)
}

func TestDeleteLoadBalancerMonitor(t *testing.T) {
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
              "id": "f1aba936b94213e5b8dca0c0dbf1f9cc"
            }
        }`)
	}

	mux.HandleFunc("/user/load_balancers/monitors/f1aba936b94213e5b8dca0c0dbf1f9cc", handler)
	assert.NoError(t, client.DeleteLoadBalancerMonitor("f1aba936b94213e5b8dca0c0dbf1f9cc"))
	assert.Error(t, client.DeleteLoadBalancerMonitor("bar"))
}

func TestModifyLoadBalancerMonitor(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "PUT", "Expected method 'PUT', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		b, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if assert.NoError(t, err) {
			assert.JSONEq(t, `{
                "id": "f1aba936b94213e5b8dca0c0dbf1f9cc",
                "type": "http",
                "description": "Login page monitor",
                "method": "GET",
                "path": "/status",
                "header": {
                  "Host": [
                    "example.com"
                  ],
                  "X-App-ID": [
                    "easy"
                  ]
                },
                "timeout": 3,
                "retries": 0,
                "interval": 90,
                "expected_body": "kicking",
                "expected_codes": "200",
                "follow_redirects": true,
                "allow_insecure": true,
                "probe_zone": ""
						}`, string(b))
		}
		fmt.Fprint(w, `{
            "success": true,
            "errors": [],
            "messages": [],
            "result": {
                "id": "f1aba936b94213e5b8dca0c0dbf1f9cc",
                "created_on": "2014-01-01T05:20:00.12345Z",
                "modified_on": "2017-02-01T05:20:00.12345Z",
                "type": "http",
                "description": "Login page monitor",
                "method": "GET",
                "path": "/status",
                "header": {
                  "Host": [
                    "example.com"
                  ],
                  "X-App-ID": [
                    "easy"
                  ]
                },
                "timeout": 3,
                "retries": 0,
                "interval": 90,
                "expected_body": "kicking",
                "expected_codes": "200",
                "follow_redirects": true,
                "allow_insecure": true,
                "probe_zone": ""
            }
        }`)
	}

	mux.HandleFunc("/user/load_balancers/monitors/f1aba936b94213e5b8dca0c0dbf1f9cc", handler)
	createdOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00.12345Z")
	modifiedOn, _ := time.Parse(time.RFC3339, "2017-02-01T05:20:00.12345Z")
	want := LoadBalancerMonitor{
		ID:          "f1aba936b94213e5b8dca0c0dbf1f9cc",
		CreatedOn:   &createdOn,
		ModifiedOn:  &modifiedOn,
		Type:        "http",
		Description: "Login page monitor",
		Method:      "GET",
		Path:        "/status",
		Header: map[string][]string{
			"Host":     {"example.com"},
			"X-App-ID": {"easy"},
		},
		Timeout:       3,
		Retries:       0,
		Interval:      90,
		ExpectedBody:  "kicking",
		ExpectedCodes: "200",

		FollowRedirects: true,
		AllowInsecure:   true,
	}
	request := LoadBalancerMonitor{
		ID:          "f1aba936b94213e5b8dca0c0dbf1f9cc",
		Type:        "http",
		Description: "Login page monitor",
		Method:      "GET",
		Path:        "/status",
		Header: map[string][]string{
			"Host":     {"example.com"},
			"X-App-ID": {"easy"},
		},
		Timeout:       3,
		Retries:       0,
		Interval:      90,
		ExpectedBody:  "kicking",
		ExpectedCodes: "200",

		FollowRedirects: true,
		AllowInsecure:   true,
	}

	actual, err := client.ModifyLoadBalancerMonitor(request)
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestCreateLoadBalancer(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "POST", "Expected method 'POST', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		b, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if assert.NoError(t, err) {
			assert.JSONEq(t, `{
              "description": "Load Balancer for www.example.com",
              "name": "www.example.com",
              "ttl": 30,
              "fallback_pool": "17b5962d775c646f3f9725cbc7a53df4",
              "default_pools": [
                "de90f38ced07c2e2f4df50b1f61d4194",
                "9290f38c5d07c2e2f4df57b1f61d4196",
                "00920f38ce07c2e2f4df50b1f61d4194"
              ],
              "region_pools": {
                "WNAM": [
                  "de90f38ced07c2e2f4df50b1f61d4194",
                  "9290f38c5d07c2e2f4df57b1f61d4196"
                ],
                "ENAM": [
                  "00920f38ce07c2e2f4df50b1f61d4194"
                ]
              },
              "pop_pools": {
                "LAX": [
                  "de90f38ced07c2e2f4df50b1f61d4194",
                  "9290f38c5d07c2e2f4df57b1f61d4196"
                ],
                "LHR": [
                  "abd90f38ced07c2e2f4df50b1f61d4194",
                  "f9138c5d07c2e2f4df57b1f61d4196"
                ],
                "SJC": [
                  "00920f38ce07c2e2f4df50b1f61d4194"
                ]
              },
              "proxied": true,
              "session_affinity": "cookie",
              "session_affinity_ttl": 5000
						}`, string(b))
		}
		fmt.Fprint(w, `{
            "success": true,
            "errors": [],
            "messages": [],
            "result": {
                "id": "699d98642c564d2e855e9661899b7252",
                "created_on": "2014-01-01T05:20:00.12345Z",
                "modified_on": "2014-02-01T05:20:00.12345Z",
                "description": "Load Balancer for www.example.com",
                "name": "www.example.com",
                "ttl": 30,
                "fallback_pool": "17b5962d775c646f3f9725cbc7a53df4",
                "default_pools": [
                  "de90f38ced07c2e2f4df50b1f61d4194",
                  "9290f38c5d07c2e2f4df57b1f61d4196",
                  "00920f38ce07c2e2f4df50b1f61d4194"
                ],
                "region_pools": {
                  "WNAM": [
                    "de90f38ced07c2e2f4df50b1f61d4194",
                    "9290f38c5d07c2e2f4df57b1f61d4196"
                  ],
                  "ENAM": [
                    "00920f38ce07c2e2f4df50b1f61d4194"
                  ]
                },
                "pop_pools": {
                  "LAX": [
                    "de90f38ced07c2e2f4df50b1f61d4194",
                    "9290f38c5d07c2e2f4df57b1f61d4196"
                  ],
                  "LHR": [
                    "abd90f38ced07c2e2f4df50b1f61d4194",
                    "f9138c5d07c2e2f4df57b1f61d4196"
                  ],
                  "SJC": [
                    "00920f38ce07c2e2f4df50b1f61d4194"
                  ]
                },
                "proxied": true,
                "session_affinity": "cookie",
                "session_affinity_ttl": 5000
            }
        }`)
	}

	mux.HandleFunc("/zones/199d98642c564d2e855e9661899b7252/load_balancers", handler)
	createdOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00.12345Z")
	modifiedOn, _ := time.Parse(time.RFC3339, "2014-02-01T05:20:00.12345Z")
	want := LoadBalancer{
		ID:           "699d98642c564d2e855e9661899b7252",
		CreatedOn:    &createdOn,
		ModifiedOn:   &modifiedOn,
		Description:  "Load Balancer for www.example.com",
		Name:         "www.example.com",
		TTL:          30,
		FallbackPool: "17b5962d775c646f3f9725cbc7a53df4",
		DefaultPools: []string{
			"de90f38ced07c2e2f4df50b1f61d4194",
			"9290f38c5d07c2e2f4df57b1f61d4196",
			"00920f38ce07c2e2f4df50b1f61d4194",
		},
		RegionPools: map[string][]string{
			"WNAM": {
				"de90f38ced07c2e2f4df50b1f61d4194",
				"9290f38c5d07c2e2f4df57b1f61d4196",
			},
			"ENAM": {
				"00920f38ce07c2e2f4df50b1f61d4194",
			},
		},
		PopPools: map[string][]string{
			"LAX": {
				"de90f38ced07c2e2f4df50b1f61d4194",
				"9290f38c5d07c2e2f4df57b1f61d4196",
			},
			"LHR": {
				"abd90f38ced07c2e2f4df50b1f61d4194",
				"f9138c5d07c2e2f4df57b1f61d4196",
			},
			"SJC": {
				"00920f38ce07c2e2f4df50b1f61d4194",
			},
		},
		Proxied:        true,
		Persistence:    "cookie",
		PersistenceTTL: 5000,
	}
	request := LoadBalancer{
		Description:  "Load Balancer for www.example.com",
		Name:         "www.example.com",
		TTL:          30,
		FallbackPool: "17b5962d775c646f3f9725cbc7a53df4",
		DefaultPools: []string{
			"de90f38ced07c2e2f4df50b1f61d4194",
			"9290f38c5d07c2e2f4df57b1f61d4196",
			"00920f38ce07c2e2f4df50b1f61d4194",
		},
		RegionPools: map[string][]string{
			"WNAM": {
				"de90f38ced07c2e2f4df50b1f61d4194",
				"9290f38c5d07c2e2f4df57b1f61d4196",
			},
			"ENAM": {
				"00920f38ce07c2e2f4df50b1f61d4194",
			},
		},
		PopPools: map[string][]string{
			"LAX": {
				"de90f38ced07c2e2f4df50b1f61d4194",
				"9290f38c5d07c2e2f4df57b1f61d4196",
			},
			"LHR": {
				"abd90f38ced07c2e2f4df50b1f61d4194",
				"f9138c5d07c2e2f4df57b1f61d4196",
			},
			"SJC": {
				"00920f38ce07c2e2f4df50b1f61d4194",
			},
		},
		Proxied:        true,
		Persistence:    "cookie",
		PersistenceTTL: 5000,
	}

	actual, err := client.CreateLoadBalancer("199d98642c564d2e855e9661899b7252", request)
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestListLoadBalancers(t *testing.T) {
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
                    "id": "699d98642c564d2e855e9661899b7252",
                    "created_on": "2014-01-01T05:20:00.12345Z",
                    "modified_on": "2014-02-01T05:20:00.12345Z",
                    "description": "Load Balancer for www.example.com",
                    "name": "www.example.com",
                    "ttl": 30,
                    "fallback_pool": "17b5962d775c646f3f9725cbc7a53df4",
                    "default_pools": [
                      "de90f38ced07c2e2f4df50b1f61d4194",
                      "9290f38c5d07c2e2f4df57b1f61d4196",
                      "00920f38ce07c2e2f4df50b1f61d4194"
                    ],
                    "region_pools": {
                      "WNAM": [
                        "de90f38ced07c2e2f4df50b1f61d4194",
                        "9290f38c5d07c2e2f4df57b1f61d4196"
                      ],
                      "ENAM": [
                        "00920f38ce07c2e2f4df50b1f61d4194"
                      ]
                    },
                    "pop_pools": {
                      "LAX": [
                        "de90f38ced07c2e2f4df50b1f61d4194",
                        "9290f38c5d07c2e2f4df57b1f61d4196"
                      ],
                      "LHR": [
                        "abd90f38ced07c2e2f4df50b1f61d4194",
                        "f9138c5d07c2e2f4df57b1f61d4196"
                      ],
                      "SJC": [
                        "00920f38ce07c2e2f4df50b1f61d4194"
                      ]
                    },
                    "proxied": true
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

	mux.HandleFunc("/zones/199d98642c564d2e855e9661899b7252/load_balancers", handler)
	createdOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00.12345Z")
	modifiedOn, _ := time.Parse(time.RFC3339, "2014-02-01T05:20:00.12345Z")
	want := []LoadBalancer{
		{
			ID:           "699d98642c564d2e855e9661899b7252",
			CreatedOn:    &createdOn,
			ModifiedOn:   &modifiedOn,
			Description:  "Load Balancer for www.example.com",
			Name:         "www.example.com",
			TTL:          30,
			FallbackPool: "17b5962d775c646f3f9725cbc7a53df4",
			DefaultPools: []string{
				"de90f38ced07c2e2f4df50b1f61d4194",
				"9290f38c5d07c2e2f4df57b1f61d4196",
				"00920f38ce07c2e2f4df50b1f61d4194",
			},
			RegionPools: map[string][]string{
				"WNAM": {
					"de90f38ced07c2e2f4df50b1f61d4194",
					"9290f38c5d07c2e2f4df57b1f61d4196",
				},
				"ENAM": {
					"00920f38ce07c2e2f4df50b1f61d4194",
				},
			},
			PopPools: map[string][]string{
				"LAX": {
					"de90f38ced07c2e2f4df50b1f61d4194",
					"9290f38c5d07c2e2f4df57b1f61d4196",
				},
				"LHR": {
					"abd90f38ced07c2e2f4df50b1f61d4194",
					"f9138c5d07c2e2f4df57b1f61d4196",
				},
				"SJC": {
					"00920f38ce07c2e2f4df50b1f61d4194",
				},
			},
			Proxied: true,
		},
	}

	actual, err := client.ListLoadBalancers("199d98642c564d2e855e9661899b7252")
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestLoadBalancerDetails(t *testing.T) {
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
                "id": "699d98642c564d2e855e9661899b7252",
                "created_on": "2014-01-01T05:20:00.12345Z",
                "modified_on": "2014-02-01T05:20:00.12345Z",
                "description": "Load Balancer for www.example.com",
                "name": "www.example.com",
                "ttl": 30,
                "fallback_pool": "17b5962d775c646f3f9725cbc7a53df4",
                "default_pools": [
                  "de90f38ced07c2e2f4df50b1f61d4194",
                  "9290f38c5d07c2e2f4df57b1f61d4196",
                  "00920f38ce07c2e2f4df50b1f61d4194"
                ],
                "region_pools": {
                  "WNAM": [
                    "de90f38ced07c2e2f4df50b1f61d4194",
                    "9290f38c5d07c2e2f4df57b1f61d4196"
                  ],
                  "ENAM": [
                    "00920f38ce07c2e2f4df50b1f61d4194"
                  ]
                },
                "pop_pools": {
                  "LAX": [
                    "de90f38ced07c2e2f4df50b1f61d4194",
                    "9290f38c5d07c2e2f4df57b1f61d4196"
                  ],
                  "LHR": [
                    "abd90f38ced07c2e2f4df50b1f61d4194",
                    "f9138c5d07c2e2f4df57b1f61d4196"
                  ],
                  "SJC": [
                    "00920f38ce07c2e2f4df50b1f61d4194"
                  ]
                },
                "proxied": true
            }
        }`)
	}

	mux.HandleFunc("/zones/199d98642c564d2e855e9661899b7252/load_balancers/699d98642c564d2e855e9661899b7252", handler)
	createdOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00.12345Z")
	modifiedOn, _ := time.Parse(time.RFC3339, "2014-02-01T05:20:00.12345Z")
	want := LoadBalancer{
		ID:           "699d98642c564d2e855e9661899b7252",
		CreatedOn:    &createdOn,
		ModifiedOn:   &modifiedOn,
		Description:  "Load Balancer for www.example.com",
		Name:         "www.example.com",
		TTL:          30,
		FallbackPool: "17b5962d775c646f3f9725cbc7a53df4",
		DefaultPools: []string{
			"de90f38ced07c2e2f4df50b1f61d4194",
			"9290f38c5d07c2e2f4df57b1f61d4196",
			"00920f38ce07c2e2f4df50b1f61d4194",
		},
		RegionPools: map[string][]string{
			"WNAM": {
				"de90f38ced07c2e2f4df50b1f61d4194",
				"9290f38c5d07c2e2f4df57b1f61d4196",
			},
			"ENAM": {
				"00920f38ce07c2e2f4df50b1f61d4194",
			},
		},
		PopPools: map[string][]string{
			"LAX": {
				"de90f38ced07c2e2f4df50b1f61d4194",
				"9290f38c5d07c2e2f4df57b1f61d4196",
			},
			"LHR": {
				"abd90f38ced07c2e2f4df50b1f61d4194",
				"f9138c5d07c2e2f4df57b1f61d4196",
			},
			"SJC": {
				"00920f38ce07c2e2f4df50b1f61d4194",
			},
		},
		Proxied: true,
	}

	actual, err := client.LoadBalancerDetails("199d98642c564d2e855e9661899b7252", "699d98642c564d2e855e9661899b7252")
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}

	_, err = client.LoadBalancerDetails("199d98642c564d2e855e9661899b7252", "bar")
	assert.Error(t, err)
}

func TestDeleteLoadBalancer(t *testing.T) {
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
              "id": "699d98642c564d2e855e9661899b7252"
            }
        }`)
	}

	mux.HandleFunc("/zones/199d98642c564d2e855e9661899b7252/load_balancers/699d98642c564d2e855e9661899b7252", handler)
	assert.NoError(t, client.DeleteLoadBalancer("199d98642c564d2e855e9661899b7252", "699d98642c564d2e855e9661899b7252"))
	assert.Error(t, client.DeleteLoadBalancer("199d98642c564d2e855e9661899b7252", "bar"))
}

func TestModifyLoadBalancer(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "PUT", "Expected method 'PUT', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		b, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if assert.NoError(t, err) {
			assert.JSONEq(t, `{
                "id": "699d98642c564d2e855e9661899b7252",
                "description": "Load Balancer for www.example.com",
                "name": "www.example.com",
                "ttl": 30,
                "fallback_pool": "17b5962d775c646f3f9725cbc7a53df4",
                "default_pools": [
                  "00920f38ce07c2e2f4df50b1f61d4194"
                ],
                "region_pools": {
                  "WNAM": [
                    "9290f38c5d07c2e2f4df57b1f61d4196"
                  ],
                  "ENAM": [
                    "00920f38ce07c2e2f4df50b1f61d4194"
                  ]
                },
                "pop_pools": {
                  "LAX": [
                    "9290f38c5d07c2e2f4df57b1f61d4196"
                  ],
                  "LHR": [
                    "f9138c5d07c2e2f4df57b1f61d4196"
                  ],
                  "SJC": [
                    "00920f38ce07c2e2f4df50b1f61d4194"
                  ]
                },
                "proxied": true,
                "session_affinity": "none"
						}`, string(b))
		}
		fmt.Fprint(w, `{
            "success": true,
            "errors": [],
            "messages": [],
            "result": {
                "id": "699d98642c564d2e855e9661899b7252",
                "created_on": "2014-01-01T05:20:00.12345Z",
                "modified_on": "2017-02-01T05:20:00.12345Z",
                "description": "Load Balancer for www.example.com",
                "name": "www.example.com",
                "ttl": 30,
                "fallback_pool": "17b5962d775c646f3f9725cbc7a53df4",
                "default_pools": [
                  "00920f38ce07c2e2f4df50b1f61d4194"
                ],
                "region_pools": {
                  "WNAM": [
                    "9290f38c5d07c2e2f4df57b1f61d4196"
                  ],
                  "ENAM": [
                    "00920f38ce07c2e2f4df50b1f61d4194"
                  ]
                },
                "pop_pools": {
                  "LAX": [
                    "9290f38c5d07c2e2f4df57b1f61d4196"
                  ],
                  "LHR": [
                    "f9138c5d07c2e2f4df57b1f61d4196"
                  ],
                  "SJC": [
                    "00920f38ce07c2e2f4df50b1f61d4194"
                  ]
                },
                "proxied": true,
                "session_affinity": "none"
            }
        }`)
	}

	mux.HandleFunc("/zones/199d98642c564d2e855e9661899b7252/load_balancers/699d98642c564d2e855e9661899b7252", handler)
	createdOn, _ := time.Parse(time.RFC3339, "2014-01-01T05:20:00.12345Z")
	modifiedOn, _ := time.Parse(time.RFC3339, "2017-02-01T05:20:00.12345Z")
	want := LoadBalancer{
		ID:           "699d98642c564d2e855e9661899b7252",
		CreatedOn:    &createdOn,
		ModifiedOn:   &modifiedOn,
		Description:  "Load Balancer for www.example.com",
		Name:         "www.example.com",
		TTL:          30,
		FallbackPool: "17b5962d775c646f3f9725cbc7a53df4",
		DefaultPools: []string{
			"00920f38ce07c2e2f4df50b1f61d4194",
		},
		RegionPools: map[string][]string{
			"WNAM": {
				"9290f38c5d07c2e2f4df57b1f61d4196",
			},
			"ENAM": {
				"00920f38ce07c2e2f4df50b1f61d4194",
			},
		},
		PopPools: map[string][]string{
			"LAX": {
				"9290f38c5d07c2e2f4df57b1f61d4196",
			},
			"LHR": {
				"f9138c5d07c2e2f4df57b1f61d4196",
			},
			"SJC": {
				"00920f38ce07c2e2f4df50b1f61d4194",
			},
		},
		Proxied:     true,
		Persistence: "none",
	}
	request := LoadBalancer{
		ID:           "699d98642c564d2e855e9661899b7252",
		Description:  "Load Balancer for www.example.com",
		Name:         "www.example.com",
		TTL:          30,
		FallbackPool: "17b5962d775c646f3f9725cbc7a53df4",
		DefaultPools: []string{
			"00920f38ce07c2e2f4df50b1f61d4194",
		},
		RegionPools: map[string][]string{
			"WNAM": {
				"9290f38c5d07c2e2f4df57b1f61d4196",
			},
			"ENAM": {
				"00920f38ce07c2e2f4df50b1f61d4194",
			},
		},
		PopPools: map[string][]string{
			"LAX": {
				"9290f38c5d07c2e2f4df57b1f61d4196",
			},
			"LHR": {
				"f9138c5d07c2e2f4df57b1f61d4196",
			},
			"SJC": {
				"00920f38ce07c2e2f4df50b1f61d4194",
			},
		},
		Proxied:     true,
		Persistence: "none",
	}

	actual, err := client.ModifyLoadBalancer("199d98642c564d2e855e9661899b7252", request)
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestLoadBalancerPoolHealthDetails(t *testing.T) {
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
                "pool_id": "699d98642c564d2e855e9661899b7252",
                "pop_health": {
                    "Amsterdam, NL": {
                        "healthy": true,
                        "origins": [
                          {
                            "2001:DB8::5": {
                                "healthy": true,
                                "rtt": "12.1ms",
                                "failure_reason": "No failures",
                                "response_code": 401
                            }
                          }
                        ]
                    }
                }
            }
        }`)
	}

	mux.HandleFunc("/user/load_balancers/pools/699d98642c564d2e855e9661899b7252/health", handler)
	want := LoadBalancerPoolHealth{
		ID: "699d98642c564d2e855e9661899b7252",
		PopHealth: map[string]LoadBalancerPoolPopHealth{
			"Amsterdam, NL": {
				Healthy: true,
				Origins: []map[string]LoadBalancerOriginHealth{
					map[string]LoadBalancerOriginHealth{
						"2001:DB8::5": {
							Healthy:       true,
							RTT:           Duration{12*time.Millisecond + 100*time.Microsecond},
							FailureReason: "No failures",
							ResponseCode:  401,
						},
					},
				},
			},
		},
	}

	actual, err := client.PoolHealthDetails("699d98642c564d2e855e9661899b7252")
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}
