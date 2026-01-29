package pagerduty

import (
	"context"

	"github.com/google/go-querystring/query"
)

// OnCall represents a contiguous unit of time for which a user will be on call for a given escalation policy and escalation rule.
type OnCall struct {
	User             User             `json:"user,omitempty"`
	Schedule         Schedule         `json:"schedule,omitempty"`
	EscalationPolicy EscalationPolicy `json:"escalation_policy,omitempty"`
	EscalationLevel  uint             `json:"escalation_level,omitempty"`
	Start            string           `json:"start,omitempty"`
	End              string           `json:"end,omitempty"`
}

// ListOnCallsResponse is the data structure returned from calling the ListOnCalls API endpoint.
type ListOnCallsResponse struct {
	APIListObject
	OnCalls []OnCall `json:"oncalls"`
}

// ListOnCallOptions is the data structure used when calling the ListOnCalls API endpoint.
type ListOnCallOptions struct {
	// Limit is the pagination parameter that limits the number of results per
	// page. PagerDuty defaults this value to 25 if omitted, and sets an upper
	// bound of 100.
	Limit uint `url:"limit,omitempty"`

	// Offset is the pagination parameter that specifies the offset at which to
	// start pagination results. When trying to request the next page of
	// results, the new Offset value should be currentOffset + Limit.
	Offset uint `url:"offset,omitempty"`

	// Total is the pagination parameter to request that the API return the
	// total count of items in the response. If this field is omitted or set to
	// false, the total number of results will not be sent back from the PagerDuty API.
	//
	// Setting this to true will slow down the API response times, and so it's
	// recommended to omit it unless you've a specific reason for wanting the
	// total count of items in the collection.
	Total bool `url:"total,omitempty"`

	TimeZone            string   `url:"time_zone,omitempty"`
	Includes            []string `url:"include,omitempty,brackets"`
	UserIDs             []string `url:"user_ids,omitempty,brackets"`
	EscalationPolicyIDs []string `url:"escalation_policy_ids,omitempty,brackets"`
	ScheduleIDs         []string `url:"schedule_ids,omitempty,brackets"`
	Since               string   `url:"since,omitempty"`
	Until               string   `url:"until,omitempty"`
	Earliest            bool     `url:"earliest,omitempty"`
}

// ListOnCalls list the on-call entries during a given time range.
//
// Deprecated: Use ListOnCallsWithContext instead.
func (c *Client) ListOnCalls(o ListOnCallOptions) (*ListOnCallsResponse, error) {
	return c.ListOnCallsWithContext(context.Background(), o)
}

// ListOnCallsWithContext list the on-call entries during a given time range.
func (c *Client) ListOnCallsWithContext(ctx context.Context, o ListOnCallOptions) (*ListOnCallsResponse, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, "/oncalls?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var result ListOnCallsResponse
	if err := c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
