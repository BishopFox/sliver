package pagerduty

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-querystring/query"
)

// BusinessService represents a business service.
type BusinessService struct {
	ID             string               `json:"id,omitempty"`
	Name           string               `json:"name,omitempty"`
	Type           string               `json:"type,omitempty"`
	Summary        string               `json:"summary,omitempty"`
	Self           string               `json:"self,omitempty"`
	PointOfContact string               `json:"point_of_contact,omitempty"`
	HTMLUrl        string               `json:"html_url,omitempty"`
	Description    string               `json:"description,omitempty"`
	Team           *BusinessServiceTeam `json:"team,omitempty"`
}

// BusinessServiceTeam represents a team object in a business service
type BusinessServiceTeam struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
	Self string `json:"self,omitempty"`
}

// BusinessServicePayload represents payload with a business service object
type BusinessServicePayload struct {
	BusinessService *BusinessService `json:"business_service,omitempty"`
}

// ListBusinessServicesResponse represents a list response of business services.
type ListBusinessServicesResponse struct {
	Total            uint               `json:"total,omitempty"`
	BusinessServices []*BusinessService `json:"business_services,omitempty"`
	Offset           uint               `json:"offset,omitempty"`
	More             bool               `json:"more,omitempty"`
	Limit            uint               `json:"limit,omitempty"`
}

// ListBusinessServiceOptions is the data structure used when calling the ListBusinessServices API endpoint.
type ListBusinessServiceOptions struct {
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

// ListBusinessServices lists existing business services. This method currently
// handles pagination of the response, so all business services should be
// present.
//
// Please note that the automatic pagination will be removed in v2 of this
// package, so it's recommended to use ListBusinessServicesPaginated instead.
func (c *Client) ListBusinessServices(o ListBusinessServiceOptions) (*ListBusinessServicesResponse, error) {
	bss, err := c.ListBusinessServicesPaginated(context.Background(), o)
	if err != nil {
		return nil, err
	}

	return &ListBusinessServicesResponse{BusinessServices: bss}, nil
}

// ListBusinessServicesPaginated lists existing business services, automatically
// handling pagination and returning the full collection.
func (c *Client) ListBusinessServicesPaginated(ctx context.Context, o ListBusinessServiceOptions) ([]*BusinessService, error) {
	queryParms, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	var businessServices []*BusinessService

	// Create a handler closure capable of parsing data from the business_services endpoint
	// and appending resultant business_services to the return slice.
	responseHandler := func(response *http.Response) (APIListObject, error) {
		var result ListBusinessServicesResponse
		if err := c.decodeJSON(response, &result); err != nil {
			return APIListObject{}, err
		}

		businessServices = append(businessServices, result.BusinessServices...)

		// Return stats on the current page. Caller can use this information to
		// adjust for requesting additional pages.
		return APIListObject{
			More:   result.More,
			Offset: result.Offset,
			Limit:  result.Limit,
		}, nil
	}

	// Make call to get all pages associated with the base endpoint.
	if err := c.pagedGet(ctx, "/business_services?"+queryParms.Encode(), responseHandler); err != nil {
		return nil, err
	}

	return businessServices, nil
}

// CreateBusinessService creates a new business service.
//
// Deprecated: Use CreateBusinessServiceWithContext instead
func (c *Client) CreateBusinessService(b *BusinessService) (*BusinessService, error) {
	return c.CreateBusinessServiceWithContext(context.Background(), b)
}

// CreateBusinessServiceWithContext creates a new business service.
func (c *Client) CreateBusinessServiceWithContext(ctx context.Context, b *BusinessService) (*BusinessService, error) {
	d := map[string]*BusinessService{
		"business_service": b,
	}

	resp, err := c.post(ctx, "/business_services", d, nil)
	return getBusinessServiceFromResponse(resp, err, c.decodeJSON)
}

// GetBusinessService gets details about a business service.
//
// Deprecated: Use GetBusinessServiceWithContext instead.
func (c *Client) GetBusinessService(id string) (*BusinessService, error) {
	return c.GetBusinessServiceWithContext(context.Background(), id)
}

// GetBusinessServiceWithContext gets details about a business service.
func (c *Client) GetBusinessServiceWithContext(ctx context.Context, id string) (*BusinessService, error) {
	resp, err := c.get(ctx, "/business_services/"+id, nil)
	return getBusinessServiceFromResponse(resp, err, c.decodeJSON)
}

// DeleteBusinessService deletes a business_service.
//
// Deprecated: Use DeleteBusinessServiceWithContext instead.
func (c *Client) DeleteBusinessService(id string) error {
	return c.DeleteBusinessServiceWithContext(context.Background(), id)
}

// DeleteBusinessServiceWithContext deletes a business_service.
func (c *Client) DeleteBusinessServiceWithContext(ctx context.Context, id string) error {
	_, err := c.delete(ctx, "/business_services/"+id)
	return err
}

// UpdateBusinessService updates a business_service.
//
// Deprecated: Use UpdateBusinessServiceWithContext instead.
func (c *Client) UpdateBusinessService(b *BusinessService) (*BusinessService, error) {
	return c.UpdateBusinessServiceWithContext(context.Background(), b)
}

// UpdateBusinessServiceWithContext updates a business_service.
func (c *Client) UpdateBusinessServiceWithContext(ctx context.Context, b *BusinessService) (*BusinessService, error) {
	id := b.ID
	b.ID = ""

	d := map[string]*BusinessService{
		"business_service": b,
	}

	resp, err := c.put(ctx, "/business_services/"+id, d, nil)
	return getBusinessServiceFromResponse(resp, err, c.decodeJSON)
}

func getBusinessServiceFromResponse(resp *http.Response, err error, decodeFn func(*http.Response, interface{}) error) (*BusinessService, error) {
	if err != nil {
		return nil, err
	}

	var target map[string]BusinessService
	if dErr := decodeFn(resp, &target); dErr != nil {
		return nil, fmt.Errorf("Could not decode JSON response: %v", dErr)
	}

	const rootNode = "business_service"

	t, nodeOK := target[rootNode]
	if !nodeOK {
		return nil, fmt.Errorf("JSON response does not have %s field", rootNode)
	}

	return &t, nil
}
