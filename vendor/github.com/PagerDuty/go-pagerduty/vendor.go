package pagerduty

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-querystring/query"
)

// Vendor represents a specific type of integration. AWS Cloudwatch, Splunk, Datadog, etc are all examples of vendors that can be integrated in PagerDuty by making an integration.
type Vendor struct {
	APIObject
	Name                  string `json:"name,omitempty"`
	LogoURL               string `json:"logo_url,omitempty"`
	LongName              string `json:"long_name,omitempty"`
	WebsiteURL            string `json:"website_url,omitempty"`
	Description           string `json:"description,omitempty"`
	Connectable           bool   `json:"connectable,omitempty"`
	ThumbnailURL          string `json:"thumbnail_url,omitempty"`
	GenericServiceType    string `json:"generic_service_type,omitempty"`
	IntegrationGuideURL   string `json:"integration_guide_url,omitempty"`
	AlertCreationDefault  string `json:"alert_creation_default,omitempty"`
	AlertCreationEditable bool   `json:"alert_creation_editable,omitempty"`
	IsPDCEF               bool   `json:"is_pd_cef,omitempty"`
}

// ListVendorResponse is the data structure returned from calling the ListVendors API endpoint.
type ListVendorResponse struct {
	APIListObject
	Vendors []Vendor
}

// ListVendorOptions is the data structure used when calling the ListVendors API endpoint.
type ListVendorOptions struct {
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

// ListVendors lists existing vendors.
//
// Deprecated: Use ListVendorsWithContext instead.
func (c *Client) ListVendors(o ListVendorOptions) (*ListVendorResponse, error) {
	return c.ListVendorsWithContext(context.Background(), o)
}

// ListVendorsWithContext lists existing vendors.
func (c *Client) ListVendorsWithContext(ctx context.Context, o ListVendorOptions) (*ListVendorResponse, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, "/vendors?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var result ListVendorResponse
	if err = c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetVendor gets details about an existing vendor.
//
// Deprecated: Use GetVendorWithContext instead.
func (c *Client) GetVendor(id string) (*Vendor, error) {
	return c.GetVendorWithContext(context.Background(), id)
}

// GetVendorWithContext gets details about an existing vendor.
func (c *Client) GetVendorWithContext(ctx context.Context, id string) (*Vendor, error) {
	resp, err := c.get(ctx, "/vendors/"+id, nil)
	return getVendorFromResponse(c, resp, err)
}

func getVendorFromResponse(c *Client, resp *http.Response, err error) (*Vendor, error) {
	if err != nil {
		return nil, err
	}

	var target map[string]Vendor
	if dErr := c.decodeJSON(resp, &target); dErr != nil {
		return nil, fmt.Errorf("Could not decode JSON response: %v", dErr)
	}

	const rootNode = "vendor"

	t, nodeOK := target[rootNode]
	if !nodeOK {
		return nil, fmt.Errorf("JSON response does not have %s field", rootNode)
	}

	return &t, nil
}
