package pagerduty

import (
	"context"
	"fmt"
)

const analyticsBaseURL = "/analytics/metrics/incidents"

// AnalyticsRequest represents the request to be sent to PagerDuty when you want
// aggregated analytics.
type AnalyticsRequest struct {
	Filters       *AnalyticsFilter `json:"filters,omitempty"`
	AggregateUnit string           `json:"aggregate_unit,omitempty"`
	TimeZone      string           `json:"time_zone,omitempty"`
}

// AnalyticsResponse represents the response from the PagerDuty API.
type AnalyticsResponse struct {
	Data          []AnalyticsData  `json:"data,omitempty"`
	Filters       *AnalyticsFilter `json:"filters,omitempty"`
	AggregateUnit string           `json:"aggregate_unit,omitempty"`
	TimeZone      string           `json:"time_zone,omitempty"`
}

// AnalyticsFilter represents the set of filters as part of the request to PagerDuty when
// requesting analytics.
type AnalyticsFilter struct {
	CreatedAtStart string   `json:"created_at_start,omitempty"`
	CreatedAtEnd   string   `json:"created_at_end,omitempty"`
	Urgency        string   `json:"urgency,omitempty"`
	Major          bool     `json:"major,omitempty"`
	ServiceIDs     []string `json:"service_ids,omitempty"`
	TeamIDs        []string `json:"team_ids,omitempty"`
	PriorityIDs    []string `json:"priority_ids,omitempty"`
	PriorityNames  []string `json:"priority_names,omitempty"`
}

// AnalyticsData represents the structure of the analytics we have available.
type AnalyticsData struct {
	ServiceID                      string  `json:"service_id,omitempty"`
	ServiceName                    string  `json:"service_name,omitempty"`
	TeamID                         string  `json:"team_id,omitempty"`
	TeamName                       string  `json:"team_name,omitempty"`
	MeanSecondsToResolve           int     `json:"mean_seconds_to_resolve,omitempty"`
	MeanSecondsToFirstAck          int     `json:"mean_seconds_to_first_ack,omitempty"`
	MeanSecondsToEngage            int     `json:"mean_seconds_to_engage,omitempty"`
	MeanSecondsToMobilize          int     `json:"mean_seconds_to_mobilize,omitempty"`
	MeanEngagedSeconds             int     `json:"mean_engaged_seconds,omitempty"`
	MeanEngagedUserCount           int     `json:"mean_engaged_user_count,omitempty"`
	TotalEscalationCount           int     `json:"total_escalation_count,omitempty"`
	MeanAssignmentCount            int     `json:"mean_assignment_count,omitempty"`
	TotalBusinessHourInterruptions int     `json:"total_business_hour_interruptions,omitempty"`
	TotalSleepHourInterruptions    int     `json:"total_sleep_hour_interruptions,omitempty"`
	TotalOffHourInterruptions      int     `json:"total_off_hour_interruptions,omitempty"`
	TotalSnoozedSeconds            int     `json:"total_snoozed_seconds,omitempty"`
	TotalEngagedSeconds            int     `json:"total_engaged_seconds,omitempty"`
	TotalIncidentCount             int     `json:"total_incident_count,omitempty"`
	UpTimePct                      float64 `json:"up_time_pct,omitempty"`
	UserDefinedEffortSeconds       int     `json:"user_defined_effort_seconds,omitempty"`
	RangeStart                     string  `json:"range_start,omitempty"`
}

// GetAggregatedIncidentData gets the aggregated incident analytics for the requested data.
func (c *Client) GetAggregatedIncidentData(ctx context.Context, analytics AnalyticsRequest) (AnalyticsResponse, error) {
	return c.getAggregatedData(ctx, analytics, "all")
}

// GetAggregatedServiceData gets the aggregated service analytics for the requested data.
func (c *Client) GetAggregatedServiceData(ctx context.Context, analytics AnalyticsRequest) (AnalyticsResponse, error) {
	return c.getAggregatedData(ctx, analytics, "services")
}

// GetAggregatedTeamData gets the aggregated team analytics for the requested data.
func (c *Client) GetAggregatedTeamData(ctx context.Context, analytics AnalyticsRequest) (AnalyticsResponse, error) {
	return c.getAggregatedData(ctx, analytics, "teams")
}

func (c *Client) getAggregatedData(ctx context.Context, analytics AnalyticsRequest, endpoint string) (AnalyticsResponse, error) {
	h := map[string]string{
		"X-EARLY-ACCESS": "analytics-v2",
	}

	u := fmt.Sprintf("%s/%s", analyticsBaseURL, endpoint)
	resp, err := c.post(ctx, u, analytics, h)
	if err != nil {
		return AnalyticsResponse{}, err
	}

	var analyticsResponse AnalyticsResponse
	if err = c.decodeJSON(resp, &analyticsResponse); err != nil {
		return AnalyticsResponse{}, err
	}

	return analyticsResponse, nil
}
