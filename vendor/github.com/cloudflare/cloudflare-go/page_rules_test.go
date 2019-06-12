package cloudflare

import (
	"fmt"
	"net/http"
	"testing"

	"time"

	"github.com/stretchr/testify/assert"
)

const (
	pageRuleID                = "15dae2fc158942f2adb1dd2a3d4273bc"
	serverPageRuleDescription = `{
    "id": "%s",
    "targets": [
      {
        "target": "url",
        "constraint": {
          "operator": "matches",
          "value": "example.%s"
        }
      }
    ],
    "actions": [
      {
        "id": "always_online",
        "value": "on"
      },
      {
        "id": "ssl",
        "value": "flexible"
      }
    ],
    "priority": 1,
    "status": "active",
    "created_on": "%[3]s",
    "modified_on": "%[3]s"
  }
`
)

var testTimestamp = time.Now().UTC()
var expectedPageRuleStruct = PageRule{
	ID: pageRuleID,
	Actions: []PageRuleAction{
		{
			ID:    "always_online",
			Value: "on",
		},
		{
			ID:    "ssl",
			Value: "flexible",
		},
	},
	Targets: []PageRuleTarget{
		{
			Target: "url",
			Constraint: struct {
				Operator string "json:\"operator\""
				Value    string "json:\"value\""
			}{Operator: "matches", Value: fmt.Sprintf("example.%s", testZoneID)},
		},
	},
	Priority:   1,
	Status:     "active",
	CreatedOn:  testTimestamp,
	ModifiedOn: testTimestamp,
}

func TestListPageRules(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "GET", "Expected method 'GET', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
		  "result": [
			%s
		  ],
		  "success": true,
		  "errors": null,
		  "messages": null,
		  "result_info": {
			"page": 1,
			"per_page": 25,
			"count": 1,
			"total_count": 1
		  }
		}
		`, fmt.Sprintf(serverPageRuleDescription, pageRuleID, testZoneID, testTimestamp.Format(time.RFC3339Nano)))
	}

	mux.HandleFunc("/zones/"+testZoneID+"/pagerules", handler)
	want := []PageRule{expectedPageRuleStruct}

	actual, err := client.ListPageRules(testZoneID)
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestGetPageRule(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "GET", "Expected method 'GET', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
		  "result": %s,
		  "success": true,
		  "errors": null,
		  "messages": null
		}
		`, fmt.Sprintf(serverPageRuleDescription, pageRuleID, testZoneID, testTimestamp.Format(time.RFC3339Nano)))
	}

	mux.HandleFunc("/zones/"+testZoneID+"/pagerules/"+pageRuleID, handler)
	want := expectedPageRuleStruct

	actual, err := client.PageRule(testZoneID, pageRuleID)
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestCreatePageRule(t *testing.T) {
	setup()
	defer teardown()
	newPageRule := PageRule{
		Actions: []PageRuleAction{
			{
				ID:    "always_online",
				Value: "on",
			},
			{
				ID:    "ssl",
				Value: "flexible",
			},
		},
		Targets: []PageRuleTarget{
			{
				Target: "url",
				Constraint: struct {
					Operator string "json:\"operator\""
					Value    string "json:\"value\""
				}{Operator: "matches", Value: fmt.Sprintf("example.%s", testZoneID)},
			},
		},
		Priority: 1,
		Status:   "active",
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "POST", "Expected method 'POST', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprintf(w, `{
		  "result": %s,
		  "success": true,
		  "errors": null,
		  "messages": null
		}
		`, fmt.Sprintf(serverPageRuleDescription, pageRuleID, testZoneID, testTimestamp.Format(time.RFC3339Nano)))
	}

	mux.HandleFunc("/zones/"+testZoneID+"/pagerules", handler)
	want := &expectedPageRuleStruct

	actual, err := client.CreatePageRule(testZoneID, newPageRule)
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}

func TestDeletePageRule(t *testing.T) {
	setup()
	defer teardown()

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.Method, "DELETE", "Expected method 'DELETE', got %s", r.Method)
		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{
		  "result": null,
		  "success": true,
		  "errors": null,
		  "messages": null
		}
		`)
	}

	mux.HandleFunc("/zones/"+testZoneID+"/pagerules/"+pageRuleID, handler)

	err := client.DeletePageRule(testZoneID, pageRuleID)
	assert.NoError(t, err)
}
