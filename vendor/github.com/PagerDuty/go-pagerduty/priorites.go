package pagerduty

import (
	"context"

	"github.com/google/go-querystring/query"
)

// PriorityProperty is the original type name and is retained as an alias for API
// compatibility.
//
// Deprecated: Use type Priority instead; will be removed in V2
type PriorityProperty = Priority

// ListPrioritiesResponse repreents the API response from PagerDuty when listing
// the configured priorities.
type ListPrioritiesResponse struct {
	APIListObject
	Priorities []Priority `json:"priorities"`
}

// Priorities is the original type name and is retained as an alias for API
// compatibility.
//
// Deprecated: Use type ListPrioritiesResponse instead; will be removed in V2
type Priorities = ListPrioritiesResponse

// ListPrioritiesOptions is the data structure used when calling the
// ListPriorities API endpoint.
type ListPrioritiesOptions struct {
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
}

// ListPriorities lists existing priorities.
//
// Deprecated: Use ListPrioritiesWithContext instead.
func (c *Client) ListPriorities() (*ListPrioritiesResponse, error) {
	return c.ListPrioritiesWithContext(context.Background(), ListPrioritiesOptions{})
}

// ListPrioritiesWithContext lists existing priorities.
func (c *Client) ListPrioritiesWithContext(ctx context.Context, o ListPrioritiesOptions) (*ListPrioritiesResponse, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, "/priorities?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var p ListPrioritiesResponse
	if err := c.decodeJSON(resp, &p); err != nil {
		return nil, err
	}

	return &p, nil
}
