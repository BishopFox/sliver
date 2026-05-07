package pagerduty

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-querystring/query"
)

const (
	escPath = "/escalation_policies"
)

// EscalationRule is a rule for an escalation policy to trigger.
type EscalationRule struct {
	ID      string      `json:"id,omitempty"`
	Delay   uint        `json:"escalation_delay_in_minutes,omitempty"`
	Targets []APIObject `json:"targets"`
}

// EscalationPolicy is a collection of escalation rules.
type EscalationPolicy struct {
	APIObject
	Name                       string           `json:"name,omitempty"`
	EscalationRules            []EscalationRule `json:"escalation_rules,omitempty"`
	Services                   []APIObject      `json:"services,omitempty"`
	NumLoops                   uint             `json:"num_loops,omitempty"`
	Teams                      []APIReference   `json:"teams"`
	Description                string           `json:"description,omitempty"`
	OnCallHandoffNotifications string           `json:"on_call_handoff_notifications,omitempty"`
}

// ListEscalationPoliciesResponse is the data structure returned from calling the ListEscalationPolicies API endpoint.
type ListEscalationPoliciesResponse struct {
	APIListObject
	EscalationPolicies []EscalationPolicy `json:"escalation_policies"`
}

// ListEscalationRulesResponse represents the data structure returned when
// calling the ListEscalationRules API endpoint.
type ListEscalationRulesResponse struct {
	APIListObject
	EscalationRules []EscalationRule `json:"escalation_rules"`
}

// ListEscalationPoliciesOptions is the data structure used when calling the ListEscalationPolicies API endpoint.
type ListEscalationPoliciesOptions struct {
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

	Query    string   `url:"query,omitempty"`
	UserIDs  []string `url:"user_ids,omitempty,brackets"`
	TeamIDs  []string `url:"team_ids,omitempty,brackets"`
	Includes []string `url:"include,omitempty,brackets"`
	SortBy   string   `url:"sort_by,omitempty"`
}

// GetEscalationRuleOptions is the data structure used when calling the GetEscalationRule API endpoint.
type GetEscalationRuleOptions struct {
	Includes []string `url:"include,omitempty,brackets"`
}

// ListEscalationPolicies lists all of the existing escalation policies.
//
// Deprecated: Use ListEscalationPoliciesWithContext instead.
func (c *Client) ListEscalationPolicies(o ListEscalationPoliciesOptions) (*ListEscalationPoliciesResponse, error) {
	return c.ListEscalationPoliciesWithContext(context.Background(), o)
}

// ListEscalationPoliciesWithContext lists all of the existing escalation policies.
func (c *Client) ListEscalationPoliciesWithContext(ctx context.Context, o ListEscalationPoliciesOptions) (*ListEscalationPoliciesResponse, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, escPath+"?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var result ListEscalationPoliciesResponse
	if err = c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CreateEscalationPolicy creates a new escalation policy.
//
// Deprecated: Use CreateEscalationPolicyWithContext instead.
func (c *Client) CreateEscalationPolicy(e EscalationPolicy) (*EscalationPolicy, error) {
	return c.CreateEscalationPolicyWithContext(context.Background(), e)
}

// CreateEscalationPolicyWithContext creates a new escalation policy.
func (c *Client) CreateEscalationPolicyWithContext(ctx context.Context, e EscalationPolicy) (*EscalationPolicy, error) {
	d := map[string]EscalationPolicy{
		"escalation_policy": e,
	}

	resp, err := c.post(ctx, escPath, d, nil)
	return getEscalationPolicyFromResponse(c, resp, err)
}

// DeleteEscalationPolicy deletes an existing escalation policy and rules.
//
// Deprecated: Use DeleteEscalationPolicyWithContext instead.
func (c *Client) DeleteEscalationPolicy(id string) error {
	return c.DeleteEscalationPolicyWithContext(context.Background(), id)
}

// DeleteEscalationPolicyWithContext deletes an existing escalation policy and rules.
func (c *Client) DeleteEscalationPolicyWithContext(ctx context.Context, id string) error {
	_, err := c.delete(ctx, escPath+"/"+id)
	return err
}

// GetEscalationPolicyOptions is the data structure used when calling the GetEscalationPolicy API endpoint.
type GetEscalationPolicyOptions struct {
	Includes []string `url:"include,omitempty,brackets"`
}

// GetEscalationPolicy gets information about an existing escalation policy and
// its rules.
//
// Deprecated: Use GetEscalationPolicyWithContext instead.
func (c *Client) GetEscalationPolicy(id string, o *GetEscalationPolicyOptions) (*EscalationPolicy, error) {
	return c.GetEscalationPolicyWithContext(context.Background(), id, o)
}

// GetEscalationPolicyWithContext gets information about an existing escalation
// policy and its rules.
func (c *Client) GetEscalationPolicyWithContext(ctx context.Context, id string, o *GetEscalationPolicyOptions) (*EscalationPolicy, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, escPath+"/"+id+"?"+v.Encode(), nil)
	return getEscalationPolicyFromResponse(c, resp, err)
}

// UpdateEscalationPolicy updates an existing escalation policy and its rules.
//
// Deprecated: Use UpdateEscalationPolicyWithContext instead.
func (c *Client) UpdateEscalationPolicy(id string, e *EscalationPolicy) (*EscalationPolicy, error) {
	return c.UpdateEscalationPolicyWithContext(context.Background(), id, *e)
}

// UpdateEscalationPolicyWithContext updates an existing escalation policy and its rules.
func (c *Client) UpdateEscalationPolicyWithContext(ctx context.Context, id string, e EscalationPolicy) (*EscalationPolicy, error) {
	d := map[string]EscalationPolicy{
		"escalation_policy": e,
	}

	resp, err := c.put(ctx, escPath+"/"+id, d, nil)
	return getEscalationPolicyFromResponse(c, resp, err)
}

// CreateEscalationRule creates a new escalation rule for an escalation policy
// and appends it to the end of the existing escalation rules.
//
// Deprecated: Use CreateEscalationRuleWithContext instead.
func (c *Client) CreateEscalationRule(escID string, e EscalationRule) (*EscalationRule, error) {
	return c.CreateEscalationRuleWithContext(context.Background(), escID, e)
}

// CreateEscalationRuleWithContext creates a new escalation rule for an escalation policy
// and appends it to the end of the existing escalation rules.
func (c *Client) CreateEscalationRuleWithContext(ctx context.Context, escID string, e EscalationRule) (*EscalationRule, error) {
	d := map[string]EscalationRule{
		"escalation_rule": e,
	}

	resp, err := c.post(ctx, escPath+"/"+escID+"/escalation_rules", d, nil)
	return getEscalationRuleFromResponse(c, resp, err)
}

// GetEscalationRule gets information about an existing escalation rule.
//
// Deprecated: Use GetEscalationRuleWithContext instead.
func (c *Client) GetEscalationRule(escID string, id string, o *GetEscalationRuleOptions) (*EscalationRule, error) {
	return c.GetEscalationRuleWithContext(context.Background(), escID, id, o)
}

// GetEscalationRuleWithContext gets information about an existing escalation rule.
func (c *Client) GetEscalationRuleWithContext(ctx context.Context, escID string, id string, o *GetEscalationRuleOptions) (*EscalationRule, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, escPath+"/"+escID+"/escalation_rules/"+id+"?"+v.Encode(), nil)
	return getEscalationRuleFromResponse(c, resp, err)
}

// DeleteEscalationRule deletes an existing escalation rule.
//
// Deprecated: Use DeleteEscalationRuleWithContext instead.
func (c *Client) DeleteEscalationRule(escID string, id string) error {
	return c.DeleteEscalationRuleWithContext(context.Background(), escID, id)
}

// DeleteEscalationRuleWithContext deletes an existing escalation rule.
func (c *Client) DeleteEscalationRuleWithContext(ctx context.Context, escID string, id string) error {
	_, err := c.delete(ctx, escPath+"/"+escID+"/escalation_rules/"+id)
	return err
}

// UpdateEscalationRule updates an existing escalation rule.
//
// Deprecated: Use UpdateEscalationRuleWithContext instead.
func (c *Client) UpdateEscalationRule(escID string, id string, e *EscalationRule) (*EscalationRule, error) {
	return c.UpdateEscalationRuleWithContext(context.Background(), escID, id, *e)
}

// UpdateEscalationRuleWithContext updates an existing escalation rule.
func (c *Client) UpdateEscalationRuleWithContext(ctx context.Context, escID string, id string, e EscalationRule) (*EscalationRule, error) {
	d := map[string]EscalationRule{
		"escalation_rule": e,
	}

	resp, err := c.put(ctx, escPath+"/"+escID+"/escalation_rules/"+id, d, nil)
	return getEscalationRuleFromResponse(c, resp, err)
}

// ListEscalationRules lists all of the escalation rules for an existing
// escalation policy.
//
// Deprecated: Use ListEscalationRulesWithContext instead.
func (c *Client) ListEscalationRules(escID string) (*ListEscalationRulesResponse, error) {
	return c.ListEscalationRulesWithContext(context.Background(), escID)
}

// ListEscalationRulesWithContext lists all of the escalation rules for an existing escalation policy.
func (c *Client) ListEscalationRulesWithContext(ctx context.Context, escID string) (*ListEscalationRulesResponse, error) {
	resp, err := c.get(ctx, escPath+"/"+escID+"/escalation_rules", nil)
	if err != nil {
		return nil, err
	}

	var result ListEscalationRulesResponse
	if err = c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func getEscalationRuleFromResponse(c *Client, resp *http.Response, err error) (*EscalationRule, error) {
	if err != nil {
		return nil, err
	}

	var target map[string]EscalationRule
	if dErr := c.decodeJSON(resp, &target); dErr != nil {
		return nil, fmt.Errorf("Could not decode JSON response: %v", dErr)
	}

	const rootNode = "escalation_rule"

	t, nodeOK := target[rootNode]
	if !nodeOK {
		return nil, fmt.Errorf("JSON response does not have %s field", rootNode)
	}

	return &t, nil
}

func getEscalationPolicyFromResponse(c *Client, resp *http.Response, err error) (*EscalationPolicy, error) {
	if err != nil {
		return nil, err
	}

	var target map[string]EscalationPolicy
	if dErr := c.decodeJSON(resp, &target); dErr != nil {
		return nil, fmt.Errorf("Could not decode JSON response: %v", dErr)
	}

	const rootNode = "escalation_policy"

	t, nodeOK := target[rootNode]
	if !nodeOK {
		return nil, fmt.Errorf("JSON response does not have %s field", rootNode)
	}

	return &t, nil
}
