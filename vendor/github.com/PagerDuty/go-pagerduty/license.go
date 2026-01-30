package pagerduty

import (
	"context"

	"github.com/google/go-querystring/query"
)

type License struct {
	APIObject
	Name                 string   `json:"name"`
	Description          string   `json:"description"`
	ValidRoles           []string `json:"valid_roles"`
	RoleGroup            string   `json:"role_group"`
	Summary              string   `json:"summary"`
	CurrentValue         int      `json:"current_value"`
	AllocationsAvailable int      `json:"allocations_available"`
}

type LicenseAllocated struct {
	APIObject
	Name        string   `json:"name"`
	Description string   `json:"description"`
	ValidRoles  []string `json:"valid_roles"`
	RoleGroup   string   `json:"role_group"`
	Summary     string   `json:"summary"`
}

type LicenseAllocation struct {
	AllocatedAt string `json:"allocated_at"`
	User        APIObject
	License     LicenseAllocated `json:"license"`
}

type ListLicensesResponse struct {
	Licenses []License `json:"licenses"`
}

type ListLicenseAllocationsResponse struct {
	APIListObject
	LicenseAllocations []LicenseAllocation `json:"license_allocations"`
}

type ListLicenseAllocationsOptions struct {
	Limit  int `url:"limit,omitempty"`
	Offset int `url:"offset,omitempty"`
}

func (c *Client) ListLicensesWithContext(ctx context.Context) (*ListLicensesResponse, error) {

	resp, err := c.get(ctx, "/licenses", nil)
	if err != nil {
		return nil, err
	}

	var result ListLicensesResponse
	err = c.decodeJSON(resp, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) ListLicenseAllocationsWithContext(ctx context.Context, o ListLicenseAllocationsOptions) (*ListLicenseAllocationsResponse, error) {

	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, "/license_allocations?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var result ListLicenseAllocationsResponse
	err = c.decodeJSON(resp, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
