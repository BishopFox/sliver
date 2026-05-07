package pagerduty

import (
	"context"

	"github.com/google/go-querystring/query"
)

const (
	standardPath = "/standards"
)

// Standard defines a PagerDuty resource standard.
type Standard struct {
	Active       bool                         `json:"active"`
	Description  string                       `json:"description,omitempty"`
	Exclusions   []StandardInclusionExclusion `json:"exclusions,omitempty"`
	ID           string                       `json:"id,omitempty"`
	Inclusions   []StandardInclusionExclusion `json:"inclusions,omitempty"`
	Name         string                       `json:"name,omitempty"`
	ResourceType string                       `json:"resource_type,omitempty"`
	Type         string                       `json:"type,omitempty"`
}

type StandardInclusionExclusion struct {
	Type string `json:"type,omitempty"`
	ID   string `json:"id,omitempty"`
}

// ListStandardsResponse is the data structure returned from calling the ListStandards API endpoint.
type ListStandardsResponse struct {
	Standards []Standard `json:"standards"`
}

// ListStandardsOptions is the data structure used when calling the ListStandards API endpoint.
type ListStandardsOptions struct {
	Active bool `url:"active,omitempty"`

	// ResourceType query for a specific resource type.
	//  Allowed value: technical_service
	ResourceType string `url:"resource_type,omitempty"`
}

type ResourceStandardScore struct {
	ResourceID   string             `json:"resource_id,omitempty"`
	ResourceType string             `json:"resource_type,omitempty"`
	Score        *ResourceScore     `json:"score,omitempty"`
	Standards    []ResourceStandard `json:"standards,omitempty"`
}

type ResourceScore struct {
	Passing int `json:"passing,omitempty"`
	Total   int `json:"total,omitempty"`
}

type ResourceStandard struct {
	Active      bool   `json:"active"`
	Description string `json:"description,omitempty"`
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Pass        bool   `json:"pass"`
	Type        string `json:"type,omitempty"`
}

type ListMultiResourcesStandardScoresResponse struct {
	Resources []ResourceStandardScore `json:"resources,omitempty"`
}

type ListMultiResourcesStandardScoresOptions struct {
	// Ids of resources to apply the standards. Maximum of 100 items
	IDs []string `url:"ids,omitempty,brackets"`
}

// ListStandards lists all the existing standards.
func (c *Client) ListStandards(ctx context.Context, o ListStandardsOptions) (*ListStandardsResponse, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, standardPath+"?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var result ListStandardsResponse
	if err = c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateStandard updates an existing standard.
func (c *Client) UpdateStandard(ctx context.Context, id string, s Standard) (*Standard, error) {
	resp, err := c.put(ctx, standardPath+"/"+id, s, nil)
	if err != nil {
		return nil, err
	}

	var result Standard
	if err = c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListResourceStandardScores
//
//	rt - Resource type
//	Allowed values: technical_services
func (c *Client) ListResourceStandardScores(ctx context.Context, id string, rt string) (*ResourceStandardScore, error) {
	resp, err := c.get(ctx, standardPath+"/scores/"+rt+"/"+id, nil)
	if err != nil {
		return nil, err
	}

	var result ResourceStandardScore
	if err = c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListMultiResourcesStandardScores
//
//	rt - Resource type
//	Allowed values: technical_services
func (c *Client) ListMultiResourcesStandardScores(ctx context.Context, rt string, o ListMultiResourcesStandardScoresOptions) (*ListMultiResourcesStandardScoresResponse, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, standardPath+"/scores/"+rt+"?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var result ListMultiResourcesStandardScoresResponse
	if err = c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
