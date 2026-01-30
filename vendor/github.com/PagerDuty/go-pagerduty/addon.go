package pagerduty

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-querystring/query"
)

// Addon is a third-party add-on to PagerDuty's UI.
type Addon struct {
	APIObject
	Name     string      `json:"name,omitempty"`
	Src      string      `json:"src,omitempty"`
	Services []APIObject `json:"services,omitempty"`
}

// ListAddonOptions are the options available when calling the ListAddons API endpoint.
type ListAddonOptions struct {
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

	Includes   []string `url:"include,omitempty,brackets"`
	ServiceIDs []string `url:"service_ids,omitempty,brackets"`
	Filter     string   `url:"filter,omitempty"`
}

// ListAddonResponse is the response when calling the ListAddons API endpoint.
type ListAddonResponse struct {
	APIListObject
	Addons []Addon `json:"addons"`
}

// ListAddons lists all of the add-ons installed on your account.
//
// Deprecated: Use ListAddonsWithContext instead.
func (c *Client) ListAddons(o ListAddonOptions) (*ListAddonResponse, error) {
	return c.ListAddonsWithContext(context.Background(), o)
}

// ListAddonsWithContext lists all of the add-ons installed on your account.
func (c *Client) ListAddonsWithContext(ctx context.Context, o ListAddonOptions) (*ListAddonResponse, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, "/addons?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var result ListAddonResponse
	if err = c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// InstallAddon installs an add-on for your account.
//
// Deprecated: Use InstallAddonWithContext instead.
func (c *Client) InstallAddon(a Addon) (*Addon, error) {
	return c.InstallAddonWithContext(context.Background(), a)
}

// InstallAddonWithContext installs an add-on for your account.
func (c *Client) InstallAddonWithContext(ctx context.Context, a Addon) (*Addon, error) {
	d := map[string]Addon{
		"addon": a,
	}

	resp, err := c.post(ctx, "/addons", d, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("Failed to create. HTTP Status code: %d", resp.StatusCode)
	}

	return getAddonFromResponse(c, resp)
}

// DeleteAddon deletes an add-on from your account.
//
// Deprecated: Use DeleteAddonWithContext instead.
func (c *Client) DeleteAddon(id string) error {
	return c.DeleteAddonWithContext(context.Background(), id)
}

// DeleteAddonWithContext deletes an add-on from your account.
func (c *Client) DeleteAddonWithContext(ctx context.Context, id string) error {
	_, err := c.delete(ctx, "/addons/"+id)
	return err
}

// GetAddon gets details about an existing add-on.
//
// Deprecated: Use GetAddonWithContext instead.
func (c *Client) GetAddon(id string) (*Addon, error) {
	return c.GetAddonWithContext(context.Background(), id)
}

// GetAddonWithContext gets details about an existing add-on.
func (c *Client) GetAddonWithContext(ctx context.Context, id string) (*Addon, error) {
	resp, err := c.get(ctx, "/addons/"+id, nil)
	if err != nil {
		return nil, err
	}

	return getAddonFromResponse(c, resp)
}

// UpdateAddon updates an existing add-on.
//
// Deprecated: Use UpdateAddonWithContext instead.
func (c *Client) UpdateAddon(id string, a Addon) (*Addon, error) {
	return c.UpdateAddonWithContext(context.Background(), id, a)
}

// UpdateAddonWithContext updates an existing add-on.
func (c *Client) UpdateAddonWithContext(ctx context.Context, id string, a Addon) (*Addon, error) {
	d := map[string]Addon{
		"addon": a,
	}

	resp, err := c.put(ctx, "/addons/"+id, d, nil)
	if err != nil {
		return nil, err
	}

	return getAddonFromResponse(c, resp)
}

func getAddonFromResponse(c *Client, resp *http.Response) (*Addon, error) {
	var result map[string]Addon
	if err := c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	const rootNode = "addon"

	a, ok := result[rootNode]
	if !ok {
		return nil, fmt.Errorf("JSON response does not have %s field", rootNode)
	}

	return &a, nil
}
