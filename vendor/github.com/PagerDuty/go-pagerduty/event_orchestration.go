package pagerduty

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-querystring/query"
)

const (
	eoPath = "/event_orchestrations"
)

// Orchestration defines a global orchestration to route events the same source
// to different services.
type Orchestration struct {
	APIObject
	Name         string                      `json:"name,omitempty"`
	Description  string                      `json:"description,omitempty"`
	Team         *APIReference               `json:"team,omitempty"`
	Integrations []*OrchestrationIntegration `json:"integrations,omitempty"`
	Routes       uint                        `json:"routes,omitempty"`
	CreatedAt    string                      `json:"created_at,omitempty"`
	CreatedBy    *APIReference               `json:"created_by,omitempty"`
	UpdatedAt    string                      `json:"updated_at,omitempty"`
	UpdatedBy    *APIReference               `json:"updated_by,omitempty"`
	Version      string                      `json:"version,omitempty"`
}

// OrchestrationIntegration is a route into an orchestration.
type OrchestrationIntegration struct {
	ID         string                              `json:"id,omitempty"`
	Parameters *OrchestrationIntegrationParameters `json:"parameters,omitempty"`
}

type OrchestrationIntegrationParameters struct {
	RoutingKey string `json:"routing_key,omitempty"`
	Type       string `json:"type,omitempty"`
}

// ListOrchestrationsResponse is the data structure returned from calling the ListOrchestrations API endpoint.
type ListOrchestrationsResponse struct {
	APIListObject
	Orchestrations []Orchestration `json:"orchestrations"`
}

// ListOrchestrationsOptions is the data structure used when calling the ListOrchestrations API endpoint.
type ListOrchestrationsOptions struct {
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

	SortBy string `url:"sort_by,omitempty"`
}

// ListOrchestrationsWithContext lists all the existing event orchestrations.
func (c *Client) ListOrchestrationsWithContext(ctx context.Context, o ListOrchestrationsOptions) (*ListOrchestrationsResponse, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, eoPath+"?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var result ListOrchestrationsResponse
	if err = c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CreateOrchestrationWithContext creates a new event orchestration.
func (c *Client) CreateOrchestrationWithContext(ctx context.Context, e Orchestration) (*Orchestration, error) {
	d := map[string]Orchestration{
		"orchestration": e,
	}

	resp, err := c.post(ctx, eoPath, d, nil)
	return getOrchestrationFromResponse(c, resp, err)
}

// DeleteOrchestrationWithContext deletes an existing event orchestration.
func (c *Client) DeleteOrchestrationWithContext(ctx context.Context, id string) error {
	_, err := c.delete(ctx, eoPath+"/"+id)
	return err
}

// GetOrchestrationOptions is the data structure used when calling the GetOrchestration API endpoint.
type GetOrchestrationOptions struct {
}

// GetOrchestrationWithContext gets information about an event orchestration.
func (c *Client) GetOrchestrationWithContext(ctx context.Context, id string, o *GetOrchestrationOptions) (*Orchestration, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, eoPath+"/"+id+"?"+v.Encode(), nil)
	return getOrchestrationFromResponse(c, resp, err)
}

// UpdateOrchestrationWithContext updates an existing event orchestration.
func (c *Client) UpdateOrchestrationWithContext(ctx context.Context, id string, e Orchestration) (*Orchestration, error) {
	d := map[string]Orchestration{
		"orchestration": e,
	}

	resp, err := c.put(ctx, eoPath+"/"+id, d, nil)
	return getOrchestrationFromResponse(c, resp, err)
}

func getOrchestrationFromResponse(c *Client, resp *http.Response, err error) (*Orchestration, error) {
	if err != nil {
		return nil, err
	}

	var target map[string]Orchestration
	if dErr := c.decodeJSON(resp, &target); dErr != nil {
		return nil, fmt.Errorf("could not decode JSON response: %v", dErr)
	}

	const rootNode = "orchestration"

	t, nodeOK := target[rootNode]
	if !nodeOK {
		return nil, fmt.Errorf("JSON response does not have %s field", rootNode)
	}

	return &t, nil
}

// OrchestrationRouter is an event router.
type OrchestrationRouter struct {
	Type      string                           `json:"type,omitempty"`
	Parent    *APIReference                    `json:"parent,omitempty"`
	Sets      []*OrchestrationRouterRuleSet    `json:"sets,omitempty"`
	CatchAll  *OrchestrationRouterCatchAllRule `json:"catch_all,omitempty"`
	CreatedAt string                           `json:"created_at,omitempty"`
	CreatedBy *APIReference                    `json:"created_by,omitempty"`
	UpdatedAt string                           `json:"updated_at,omitempty"`
	UpdatedBy *APIReference                    `json:"updated_by,omitempty"`
	Version   string                           `json:"version,omitempty"`
}

type OrchestrationRouterRuleSet struct {
	ID    string                     `json:"id,omitempty"`
	Rules []*OrchestrationRouterRule `json:"rules,omitempty"`
}

type OrchestrationRouterRule struct {
	ID         string                              `json:"id,omitempty"`
	Label      string                              `json:"label,omitempty"`
	Conditions []*OrchestrationRouterRuleCondition `json:"conditions,omitempty"`
	Actions    *OrchestrationRouterActions         `json:"actions,omitempty"`
	Disabled   bool                                `json:"disabled,omitempty"`
}

type OrchestrationRouterRuleCondition struct {
	Expression string `json:"expression,omitempty"`
}

// OrchestrationRouterCatchAllRule routes an event when none of the rules match an event.
type OrchestrationRouterCatchAllRule struct {
	Actions *OrchestrationRouterActions `json:"actions,omitempty"`
}

// OrchestrationRouterActions are the actions that will be taken to change the resulting alert and incident.
type OrchestrationRouterActions struct {
	RouteTo string `json:"route_to,omitempty"`
}

// GetOrchestrationRouterOptions is the data structure used when calling the GetOrchestrationRouter API endpoint.
type GetOrchestrationRouterOptions struct {
}

// GetOrchestrationRouterWithContext gets information about an event orchestration.
func (c *Client) GetOrchestrationRouterWithContext(ctx context.Context, id string, o *GetOrchestrationRouterOptions) (*OrchestrationRouter, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, eoPath+"/"+id+"/router"+"?"+v.Encode(), nil)
	return getOrchestrationRouterFromResponse(c, resp, err)
}

// UpdateOrchestrationRouterWithContext updates the routing rules of an existing event orchestration.
func (c *Client) UpdateOrchestrationRouterWithContext(ctx context.Context, id string, e OrchestrationRouter) (*OrchestrationRouter, error) {
	d := map[string]OrchestrationRouter{
		"orchestration_path": e,
	}

	resp, err := c.put(ctx, eoPath+"/"+id+"/router", d, nil)
	return getOrchestrationRouterFromResponse(c, resp, err)
}

func getOrchestrationRouterFromResponse(c *Client, resp *http.Response, err error) (*OrchestrationRouter, error) {
	if err != nil {
		return nil, err
	}

	var target map[string]OrchestrationRouter
	if dErr := c.decodeJSON(resp, &target); dErr != nil {
		return nil, fmt.Errorf("could not decode JSON response: %v", dErr)
	}

	const rootNode = "orchestration_path"

	t, nodeOK := target[rootNode]
	if !nodeOK {
		return nil, fmt.Errorf("JSON response does not have %s field", rootNode)
	}

	return &t, nil
}

// ServiceOrchestration defines sets of rules belonging to a service.
type ServiceOrchestration struct {
	Type     string                            `json:"type,omitempty"`
	Parent   *APIReference                     `json:"parent,omitempty"`
	Sets     []*ServiceOrchestrationRuleSet    `json:"sets,omitempty"`
	CatchAll *ServiceOrchestrationCatchAllRule `json:"catch_all,omitempty"`

	CreatedAt string        `json:"created_at,omitempty"`
	CreatedBy *APIReference `json:"created_by,omitempty"`
	UpdatedAt string        `json:"updated_at,omitempty"`
	UpdatedBy *APIReference `json:"updated_by,omitempty"`
	Version   string        `json:"version,omitempty"`
}

type ServiceOrchestrationCatchAllRule struct {
	Actions *ServiceOrchestrationRuleActions `json:"actions,omitempty"`
}

type ServiceOrchestrationRuleSet struct {
	ID    string                      `json:"id,omitempty"`
	Rules []*ServiceOrchestrationRule `json:"rules,omitempty"`
}

type ServiceOrchestrationRule struct {
	ID         string                               `json:"id,omitempty"`
	Label      string                               `json:"label,omitempty"`
	Conditions []*ServiceOrchestrationRuleCondition `json:"conditions,omitempty"`
	Actions    *ServiceOrchestrationRuleActions     `json:"actions,omitempty"`
	Disabled   bool                                 `json:"disabled,omitempty"`
}

type ServiceOrchestrationRuleCondition struct {
	Expression string `json:"expression,omitempty"`
}

// ServiceOrchestrationRuleActions are the actions that will be taken to change the resulting alert and incident.
type ServiceOrchestrationRuleActions struct {
	RouteTo                    string                       `json:"route_to,omitempty"`
	Suppress                   bool                         `json:"suppress,omitempty"`
	Suspend                    uint                         `json:"suspend,omitempty"`
	Priority                   string                       `json:"priority,omitempty"`
	Annotate                   string                       `json:"annotate,omitempty"`
	PagerDutyAutomationActions []*PagerDutyAutomationAction `json:"pagerduty_automation_actions,omitempty"`
	AutomationActions          []*AutomationAction          `json:"automation_actions,omitempty"`
	Severity                   string                       `json:"severity,omitempty"`
	EventAction                string                       `json:"event_action,omitempty"`
	Variables                  []*OrchestrationVariable     `json:"variables,omitempty"`
	Extractions                []*OrchestrationExtraction   `json:"extractions,omitempty"`
}

type ServiceOrchestrationActive struct {
	Active bool `json:"active,omitempty"`
}

// GetServiceOrchestrationOptions is the data structure used when calling the GetServiceOrchestration API endpoint.
type GetServiceOrchestrationOptions struct {
}

// GetServiceOrchestrationWithContext gets information about a service orchestration.
func (c *Client) GetServiceOrchestrationWithContext(ctx context.Context, id string, o *GetServiceOrchestrationOptions) (*ServiceOrchestration, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, eoPath+"/services/"+id+"?"+v.Encode(), nil)
	return getServiceOrchestrationFromResponse(c, resp, err)
}

// UpdateServiceOrchestrationWithContext updates the routing rules of a service orchestration.
func (c *Client) UpdateServiceOrchestrationWithContext(ctx context.Context, id string, e ServiceOrchestration) (*ServiceOrchestration, error) {
	d := map[string]ServiceOrchestration{
		"orchestration_path": e,
	}

	resp, err := c.put(ctx, eoPath+"/services/"+id, d, nil)
	return getServiceOrchestrationFromResponse(c, resp, err)
}

// GetServiceOrchestrationActiveWithContext gets a service orchestration's active status.
func (c *Client) GetServiceOrchestrationActiveWithContext(ctx context.Context, id string) (*ServiceOrchestrationActive, error) {
	resp, err := c.get(ctx, eoPath+"/services/"+id+"/active", nil)
	return getServiceOrchestrationActiveFromResponse(c, resp, err)
}

// UpdateServiceOrchestrationActiveWithContext updates a service orchestration's active status.
func (c *Client) UpdateServiceOrchestrationActiveWithContext(ctx context.Context, id string, e ServiceOrchestrationActive) (*ServiceOrchestrationActive, error) {
	resp, err := c.put(ctx, eoPath+"/services/"+id+"/active", e, nil)
	return getServiceOrchestrationActiveFromResponse(c, resp, err)
}

func getServiceOrchestrationFromResponse(c *Client, resp *http.Response, err error) (*ServiceOrchestration, error) {
	if err != nil {
		return nil, err
	}

	var target map[string]ServiceOrchestration
	if dErr := c.decodeJSON(resp, &target); dErr != nil {
		return nil, fmt.Errorf("could not decode JSON response: %v", dErr)
	}

	const rootNode = "orchestration_path"

	t, nodeOK := target[rootNode]
	if !nodeOK {
		return nil, fmt.Errorf("JSON response does not have %s field", rootNode)
	}

	return &t, nil
}

func getServiceOrchestrationActiveFromResponse(c *Client, resp *http.Response, err error) (*ServiceOrchestrationActive, error) {
	if err != nil {
		return nil, err
	}

	var target ServiceOrchestrationActive
	if dErr := c.decodeJSON(resp, &target); dErr != nil {
		return nil, fmt.Errorf("could not decode JSON response: %v", dErr)
	}

	return &target, nil
}

// OrchestrationUnrouted defines sets of rules to be applied to unrouted events.
type OrchestrationUnrouted struct {
	Type      string                             `json:"type,omitempty"`
	Parent    *APIReference                      `json:"parent,omitempty"`
	Sets      []*ServiceOrchestrationRuleSet     `json:"sets,omitempty"`
	CatchAll  *OrchestrationUnroutedCatchAllRule `json:"catch_all,omitempty"`
	CreatedAt string                             `json:"created_at,omitempty"`
	CreatedBy *APIReference                      `json:"created_by,omitempty"`
	UpdatedAt string                             `json:"updated_at,omitempty"`
	UpdatedBy *APIReference                      `json:"updated_by,omitempty"`
	Version   string                             `json:"version,omitempty"`
}

type OrchestrationUnroutedCatchAllRule struct {
	Actions *OrchestrationUnroutedRuleActions `json:"actions,omitempty"`
}

type OrchestrationUnroutedRuleSet struct {
	ID    string                       `json:"id,omitempty"`
	Rules []*OrchestrationUnroutedRule `json:"rules,omitempty"`
}

type OrchestrationUnroutedRule struct {
	ID         string                                `json:"id,omitempty"`
	Label      string                                `json:"label,omitempty"`
	Conditions []*OrchestrationUnroutedRuleCondition `json:"conditions,omitempty"`
	Actions    *OrchestrationUnroutedRuleActions     `json:"actions,omitempty"`
	Disabled   bool                                  `json:"disabled,omitempty"`
}

type OrchestrationUnroutedRuleCondition struct {
	Expression string `json:"expression,omitempty"`
}

// OrchestrationUnroutedRuleActions are the actions that will be taken to change the resulting alert and incident.
type OrchestrationUnroutedRuleActions struct {
	RouteTo     string                     `json:"route_to,omitempty"`
	Severity    string                     `json:"severity,omitempty"`
	EventAction string                     `json:"event_action,omitempty"`
	Variables   []*OrchestrationVariable   `json:"variables,omitempty"`
	Extractions []*OrchestrationExtraction `json:"extractions,omitempty"`
}

// GetOrchestrationUnroutedOptions is the data structure used when calling the GetOrchestrationUnrouted API endpoint.
type GetOrchestrationUnroutedOptions struct {
}

// GetOrchestrationUnroutedWithContext gets the routing rules for unrouted events.
func (c *Client) GetOrchestrationUnroutedWithContext(ctx context.Context, id string, o *GetOrchestrationUnroutedOptions) (*OrchestrationUnrouted, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, eoPath+"/"+id+"/unrouted"+"?"+v.Encode(), nil)
	return getOrchestrationUnroutedFromResponse(c, resp, err)
}

// UpdateOrchestrationUnroutedWithContext updates the routing rules for unrouted events.
func (c *Client) UpdateOrchestrationUnroutedWithContext(ctx context.Context, id string, e OrchestrationUnrouted) (*OrchestrationUnrouted, error) {
	d := map[string]OrchestrationUnrouted{
		"orchestration_path": e,
	}

	resp, err := c.put(ctx, eoPath+"/"+id+"/unrouted", d, nil)
	return getOrchestrationUnroutedFromResponse(c, resp, err)
}

func getOrchestrationUnroutedFromResponse(c *Client, resp *http.Response, err error) (*OrchestrationUnrouted, error) {
	if err != nil {
		return nil, err
	}

	var target map[string]OrchestrationUnrouted
	if dErr := c.decodeJSON(resp, &target); dErr != nil {
		return nil, fmt.Errorf("could not decode JSON response: %v", dErr)
	}

	const rootNode = "orchestration_path"

	t, nodeOK := target[rootNode]
	if !nodeOK {
		return nil, fmt.Errorf("JSON response does not have %s field", rootNode)
	}

	return &t, nil
}

type PagerDutyAutomationAction struct {
	ActionID string `json:"action_id,omitempty"`
}

type AutomationAction struct {
	Name       string                    `json:"name,omitempty"`
	URL        string                    `json:"url,omitempty"`
	AutoSend   bool                      `json:"auto_send,omitempty"`
	Headers    []*OrchestrationHeader    `json:"headers,omitempty"`
	Parameters []*OrchestrationParameter `json:"parameters,omitempty"`
}

type OrchestrationHeader struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type OrchestrationParameter struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// OrchestrationExtraction defines a value extraction in an orchestration rule.
type OrchestrationExtraction struct {
	Target   string `json:"target,omitempty"`
	Regex    string `json:"regex,omitempty"`
	Source   string `json:"source,omitempty"`
	Template string `json:"template,omitempty"`
}

// OrchestrationVariable defines a variable in an orchestration rule.
type OrchestrationVariable struct {
	Name  string `json:"name,omitempty"`
	Path  string `json:"path,omitempty"`
	Type  string `json:"type,omitempty"`
	Value string `json:"value,omitempty"`
}
