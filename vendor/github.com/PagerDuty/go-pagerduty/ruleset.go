package pagerduty

import (
	"context"
	"fmt"
	"net/http"
)

// Ruleset represents a ruleset.
type Ruleset struct {
	ID          string         `json:"id,omitempty"`
	Name        string         `json:"name,omitempty"`
	Type        string         `json:"type,omitempty"`
	Self        string         `json:"self,omitempty"`
	RoutingKeys []string       `json:"routing_keys,omitempty"`
	CreatedAt   string         `json:"created_at"`
	Creator     *RulesetObject `json:"creator,omitempty"`
	UpdatedAt   string         `json:"updated_at"`
	Updater     *RulesetObject `json:"updater,omitempty"`
	Team        *RulesetObject `json:"team,omitempty"`
}

// RulesetObject represents a generic object that is common within a ruleset object
type RulesetObject struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
	Self string `json:"self,omitempty"`
}

// RulesetPayload represents payload with a ruleset object
type RulesetPayload struct {
	Ruleset *Ruleset `json:"ruleset,omitempty"`
}

// ListRulesetsResponse represents a list response of rulesets.
type ListRulesetsResponse struct {
	Total    uint       `json:"total,omitempty"`
	Rulesets []*Ruleset `json:"rulesets,omitempty"`
	Offset   uint       `json:"offset,omitempty"`
	More     bool       `json:"more,omitempty"`
	Limit    uint       `json:"limit,omitempty"`
}

// RulesetRule represents a Ruleset rule
type RulesetRule struct {
	ID         string          `json:"id,omitempty"`
	Self       string          `json:"self,omitempty"`
	Position   *int            `json:"position,omitempty"`
	Disabled   bool            `json:"disabled,omitempty"`
	Conditions *RuleConditions `json:"conditions,omitempty"`
	Actions    *RuleActions    `json:"actions,omitempty"`
	Ruleset    *APIObject      `json:"ruleset,omitempty"`
	CatchAll   bool            `json:"catch_all,omitempty"`
	TimeFrame  *RuleTimeFrame  `json:"time_frame,omitempty"`
}

// RulesetRulePayload represents a payload for ruleset rules
type RulesetRulePayload struct {
	Rule *RulesetRule `json:"rule,omitempty"`
}

// RuleConditions represents the conditions field for a Ruleset
type RuleConditions struct {
	Operator          string              `json:"operator,omitempty"`
	RuleSubconditions []*RuleSubcondition `json:"subconditions,omitempty"`
}

// RuleSubcondition represents a subcondition of a ruleset condition
type RuleSubcondition struct {
	Operator   string              `json:"operator,omitempty"`
	Parameters *ConditionParameter `json:"parameters,omitempty"`
}

// ConditionParameter represents  parameters in a rule condition
type ConditionParameter struct {
	Path  string `json:"path,omitempty"`
	Value string `json:"value,omitempty"`
}

// RuleTimeFrame represents a time_frame object on the rule object
type RuleTimeFrame struct {
	ScheduledWeekly *ScheduledWeekly `json:"scheduled_weekly,omitempty"`
	ActiveBetween   *ActiveBetween   `json:"active_between,omitempty"`
}

// ScheduledWeekly represents a time_frame object for scheduling rules weekly
type ScheduledWeekly struct {
	// Weekdays is a 0 indexed slice of days, where 0 is Sunday and 6 is
	// Saturday, when the window is scheduled for.
	Weekdays []int `json:"weekdays,omitempty"`

	Timezone string `json:"timezone,omitempty"`

	// StartTime is the number of milliseconds into the day at which the window
	// starts.
	StartTime int `json:"start_time,omitempty"`

	// Duration is the window duration in milliseconds.
	Duration int `json:"duration,omitempty"`
}

// ActiveBetween represents an active_between object for setting a timeline for rules
type ActiveBetween struct {
	// StartTime is in the number of milliseconds into the day at which the
	// window starts.
	StartTime int `json:"start_time,omitempty"`

	// EndTime is the number of milliseconds into the day at which the window
	// ends.
	EndTime int `json:"end_time,omitempty"`
}

// ListRulesetRulesResponse represents a list of rules in a ruleset
type ListRulesetRulesResponse struct {
	Total  uint           `json:"total,omitempty"`
	Rules  []*RulesetRule `json:"rules,omitempty"`
	Offset uint           `json:"offset,omitempty"`
	More   bool           `json:"more,omitempty"`
	Limit  uint           `json:"limit,omitempty"`
}

// RuleActions represents a rule action
type RuleActions struct {
	Annotate    *RuleActionParameter    `json:"annotate,omitempty"`
	EventAction *RuleActionParameter    `json:"event_action,omitempty"`
	Extractions []*RuleActionExtraction `json:"extractions,omitempty"`
	Priority    *RuleActionParameter    `json:"priority,omitempty"`
	Severity    *RuleActionParameter    `json:"severity,omitempty"`
	Suppress    *RuleActionSuppress     `json:"suppress,omitempty"`
	Suspend     *RuleActionSuspend      `json:"suspend,omitempty"`
	Route       *RuleActionParameter    `json:"route"`
}

// RuleActionParameter represents a generic parameter object on a rule action
type RuleActionParameter struct {
	Value string `json:"value,omitempty"`
}

// RuleActionSuppress represents a rule suppress action object
type RuleActionSuppress struct {
	Value               bool   `json:"value"`
	ThresholdValue      int    `json:"threshold_value,omitempty"`
	ThresholdTimeUnit   string `json:"threshold_time_unit,omitempty"`
	ThresholdTimeAmount int    `json:"threshold_time_amount,omitempty"`
}

// RuleActionSuspend represents a rule suspend action object
type RuleActionSuspend struct {
	// Value specifies for how long to suspend the alert in seconds.
	Value int `json:"value,omitempty"`
}

// RuleActionExtraction represents a rule extraction action object
type RuleActionExtraction struct {
	Target string `json:"target,omitempty"`
	Source string `json:"source,omitempty"`
	Regex  string `json:"regex,omitempty"`
}

// ListRulesets gets all rulesets. This method currently handles pagination of
// the response, so all rulesets should be present.
//
// Please note that the automatic pagination will be removed in v2 of this
// package, so it's recommended to use ListRulesetsPaginated instead.
func (c *Client) ListRulesets() (*ListRulesetsResponse, error) {
	rs, err := c.ListRulesetsPaginated(context.Background())
	if err != nil {
		return nil, err
	}

	return &ListRulesetsResponse{Rulesets: rs}, nil
}

// ListRulesetsPaginated gets all rulesets.
func (c *Client) ListRulesetsPaginated(ctx context.Context) ([]*Ruleset, error) {
	var rulesets []*Ruleset

	// Create a handler closure capable of parsing data from the rulesets endpoint
	// and appending resultant rulesets to the return slice.
	responseHandler := func(response *http.Response) (APIListObject, error) {
		var result ListRulesetsResponse
		if err := c.decodeJSON(response, &result); err != nil {
			return APIListObject{}, err
		}

		rulesets = append(rulesets, result.Rulesets...)

		// Return stats on the current page. Caller can use this information to
		// adjust for requesting additional pages.
		return APIListObject{
			More:   result.More,
			Offset: result.Offset,
			Limit:  result.Limit,
		}, nil
	}

	// Make call to get all pages associated with the base endpoint.
	if err := c.pagedGet(ctx, "/rulesets/", responseHandler); err != nil {
		return nil, err
	}

	return rulesets, nil
}

// CreateRuleset creates a new ruleset.
//
// Deprecated: Use CreateRulesetWithContext instead.
func (c *Client) CreateRuleset(r *Ruleset) (*Ruleset, error) {
	return c.CreateRulesetWithContext(context.Background(), r)
}

// CreateRulesetWithContext creates a new ruleset.
func (c *Client) CreateRulesetWithContext(ctx context.Context, r *Ruleset) (*Ruleset, error) {
	d := map[string]*Ruleset{
		"ruleset": r,
	}

	resp, err := c.post(ctx, "/rulesets", d, nil)
	return getRulesetFromResponse(c, resp, err)
}

// DeleteRuleset deletes a ruleset.
//
// Deprecated: Use DeleteRulesetWithContext instead.
func (c *Client) DeleteRuleset(id string) error {
	return c.DeleteRulesetWithContext(context.Background(), id)
}

// DeleteRulesetWithContext deletes a ruleset.
func (c *Client) DeleteRulesetWithContext(ctx context.Context, id string) error {
	_, err := c.delete(ctx, "/rulesets/"+id)
	return err
}

// GetRuleset gets details about a ruleset.
//
// Deprecated: Use GetRulesetWithContext instead.
func (c *Client) GetRuleset(id string) (*Ruleset, error) {
	return c.GetRulesetWithContext(context.Background(), id)
}

// GetRulesetWithContext gets details about a ruleset.
func (c *Client) GetRulesetWithContext(ctx context.Context, id string) (*Ruleset, error) {
	resp, err := c.get(ctx, "/rulesets/"+id, nil)
	return getRulesetFromResponse(c, resp, err)
}

// UpdateRuleset updates a ruleset.
//
// Deprecated: Use UpdateRulesetWithContext instead.
func (c *Client) UpdateRuleset(r *Ruleset) (*Ruleset, error) {
	return c.UpdateRulesetWithContext(context.Background(), r)
}

// UpdateRulesetWithContext updates a ruleset.
func (c *Client) UpdateRulesetWithContext(ctx context.Context, r *Ruleset) (*Ruleset, error) {
	d := map[string]*Ruleset{
		"ruleset": r,
	}

	resp, err := c.put(ctx, "/rulesets/"+r.ID, d, nil)
	return getRulesetFromResponse(c, resp, err)
}

func getRulesetFromResponse(c *Client, resp *http.Response, err error) (*Ruleset, error) {
	if err != nil {
		return nil, err
	}

	var target map[string]Ruleset
	if dErr := c.decodeJSON(resp, &target); dErr != nil {
		return nil, fmt.Errorf("Could not decode JSON response: %v", dErr)
	}

	t, nodeOK := target["ruleset"]
	if !nodeOK {
		return nil, fmt.Errorf("JSON response does not have ruleset field")
	}

	return &t, nil
}

// ListRulesetRules gets all rules for a ruleset. This method currently handles pagination of
// the response, so all RuleseRule should be present.
//
// Please note that the automatic pagination will be removed in v2 of this
// package, so it's recommended to use ListRulesetRulesPaginated instead.
func (c *Client) ListRulesetRules(rulesetID string) (*ListRulesetRulesResponse, error) {
	rsr, err := c.ListRulesetRulesPaginated(context.Background(), rulesetID)
	if err != nil {
		return nil, err
	}

	return &ListRulesetRulesResponse{Rules: rsr}, nil
}

// ListRulesetRulesPaginated gets all rules for a ruleset.
func (c *Client) ListRulesetRulesPaginated(ctx context.Context, rulesetID string) ([]*RulesetRule, error) {
	var rules []*RulesetRule

	// Create a handler closure capable of parsing data from the ruleset rules endpoint
	// and appending resultant ruleset rules to the return slice.
	responseHandler := func(response *http.Response) (APIListObject, error) {
		var result ListRulesetRulesResponse

		if err := c.decodeJSON(response, &result); err != nil {
			return APIListObject{}, err
		}

		rules = append(rules, result.Rules...)

		// Return stats on the current page. Caller can use this information to
		// adjust for requesting additional pages.
		return APIListObject{
			More:   result.More,
			Offset: result.Offset,
			Limit:  result.Limit,
		}, nil
	}

	// Make call to get all pages associated with the base endpoint.
	if err := c.pagedGet(ctx, "/rulesets/"+rulesetID+"/rules", responseHandler); err != nil {
		return nil, err
	}

	return rules, nil
}

// GetRulesetRule gets an event rule.
//
// Deprecated: Use GetRulesetRuleWithContext instead.
func (c *Client) GetRulesetRule(rulesetID, ruleID string) (*RulesetRule, error) {
	return c.GetRulesetRuleWithContext(context.Background(), rulesetID, ruleID)
}

// GetRulesetRuleWithContext gets an event rule
func (c *Client) GetRulesetRuleWithContext(ctx context.Context, rulesetID, ruleID string) (*RulesetRule, error) {
	resp, err := c.get(ctx, "/rulesets/"+rulesetID+"/rules/"+ruleID, nil)
	return getRuleFromResponse(c, resp, err)
}

// DeleteRulesetRule deletes a rule.
//
// Deprecated: Use DeleteRulesetRuleWithContext instead.
func (c *Client) DeleteRulesetRule(rulesetID, ruleID string) error {
	return c.DeleteRulesetRuleWithContext(context.Background(), rulesetID, ruleID)
}

// DeleteRulesetRuleWithContext deletes a rule.
func (c *Client) DeleteRulesetRuleWithContext(ctx context.Context, rulesetID, ruleID string) error {
	_, err := c.delete(ctx, "/rulesets/"+rulesetID+"/rules/"+ruleID)
	return err
}

// CreateRulesetRule creates a new rule for a ruleset.
//
// Deprecated: Use CreateRulesetRuleWithContext instead.
func (c *Client) CreateRulesetRule(rulesetID string, rule *RulesetRule) (*RulesetRule, error) {
	return c.CreateRulesetRuleWithContext(context.Background(), rulesetID, rule)
}

// CreateRulesetRuleWithContext creates a new rule for a ruleset.
func (c *Client) CreateRulesetRuleWithContext(ctx context.Context, rulesetID string, rule *RulesetRule) (*RulesetRule, error) {
	d := map[string]*RulesetRule{
		"rule": rule,
	}

	resp, err := c.post(ctx, "/rulesets/"+rulesetID+"/rules/", d, nil)
	return getRuleFromResponse(c, resp, err)
}

// UpdateRulesetRule updates a rule.
//
// Deprecated: Use UpdateRulesetRuleWithContext instead.
func (c *Client) UpdateRulesetRule(rulesetID, ruleID string, r *RulesetRule) (*RulesetRule, error) {
	return c.UpdateRulesetRuleWithContext(context.Background(), rulesetID, ruleID, r)
}

// UpdateRulesetRuleWithContext updates a rule.
func (c *Client) UpdateRulesetRuleWithContext(ctx context.Context, rulesetID, ruleID string, r *RulesetRule) (*RulesetRule, error) {
	d := map[string]*RulesetRule{
		"rule": r,
	}

	resp, err := c.put(ctx, "/rulesets/"+rulesetID+"/rules/"+ruleID, d, nil)
	return getRuleFromResponse(c, resp, err)
}

func getRuleFromResponse(c *Client, resp *http.Response, err error) (*RulesetRule, error) {
	if err != nil {
		return nil, err
	}

	var target map[string]RulesetRule
	if dErr := c.decodeJSON(resp, &target); dErr != nil {
		return nil, fmt.Errorf("Could not decode JSON response: %v", dErr)
	}

	const rootNode = "rule"

	t, nodeOK := target[rootNode]
	if !nodeOK {
		return nil, fmt.Errorf("JSON response does not have %s field", rootNode)
	}

	return &t, nil
}
