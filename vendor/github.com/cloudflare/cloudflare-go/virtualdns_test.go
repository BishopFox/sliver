package cloudflare

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func float64Ptr(v float64) *float64 {
	return &v
}

func int64Ptr(v int64) *int64 {
	return &v
}

func TestVirtualDNSUserAnalytics(t *testing.T) {
	setup()
	defer teardown()

	since := time.Now().Add(-1 * time.Hour)
	until := time.Now()

	handler := func(w http.ResponseWriter, r *http.Request) {
		expectedMetrics := "queryCount,uncachedCount,staleCount,responseTimeAvg,responseTimeMedia,responseTime90th,responseTime99th"

		assert.Equal(t, r.Method, "GET", "Expected method 'GET'")
		assert.Equal(t, expectedMetrics, r.URL.Query().Get("metrics"), "Expected many metrics in URL parameter")
		assert.Equal(t, since.Format(time.RFC3339), r.URL.Query().Get("since"), "Expected since parameter in URL")
		assert.Equal(t, until.Format(time.RFC3339), r.URL.Query().Get("until"), "Expected until parameter in URL")

		w.Header().Set("content-type", "application/json")
		fmt.Fprint(w, `{
		  "result": {
			"totals":{
				"queryCount": 5,
				"uncachedCount":6,
				"staleCount":7,
				"responseTimeAvg":1.0,
				"responseTimeMedian":2.0,
				"responseTime90th":3.0,
				"responseTime99th":4.0
			  }
		  },
		  "success": true,
		  "errors": null,
		  "messages": null
		}`)
	}

	mux.HandleFunc("/user/virtual_dns/12345/dns_analytics/report", handler)
	want := VirtualDNSAnalytics{
		Totals: VirtualDNSAnalyticsMetrics{
			QueryCount:         int64Ptr(5),
			UncachedCount:      int64Ptr(6),
			StaleCount:         int64Ptr(7),
			ResponseTimeAvg:    float64Ptr(1.0),
			ResponseTimeMedian: float64Ptr(2.0),
			ResponseTime90th:   float64Ptr(3.0),
			ResponseTime99th:   float64Ptr(4.0),
		},
	}

	params := VirtualDNSUserAnalyticsOptions{
		Metrics: []string{
			"queryCount",
			"uncachedCount",
			"staleCount",
			"responseTimeAvg",
			"responseTimeMedia",
			"responseTime90th",
			"responseTime99th",
		},
		Since: &since,
		Until: &until,
	}
	actual, err := client.VirtualDNSUserAnalytics("12345", params)
	if assert.NoError(t, err) {
		assert.Equal(t, want, actual)
	}
}
