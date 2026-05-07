package pagerduty

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-querystring/query"
)

// Tag is a way to label user, team and escalation policies in PagerDuty
type Tag struct {
	APIObject
	Label string `json:"label,omitempty"`
}

// ListTagResponse is the structure used when calling the ListTags API endpoint.
type ListTagResponse struct {
	APIListObject
	Tags []*Tag `json:"tags"`
}

// ListUserResponse is the structure used to list users assigned a given tag
type ListUserResponse struct {
	APIListObject
	Users []*APIObject `json:"users,omitempty"`
}

// ListTeamsForTagResponse is the structure used to list teams assigned a given tag
type ListTeamsForTagResponse struct {
	APIListObject
	Teams []*APIObject `json:"teams,omitempty"`
}

// ListEPResponse is the structure used to list escalation policies assigned a given tag
type ListEPResponse struct {
	APIListObject
	EscalationPolicies []*APIObject `json:"escalation_policies,omitempty"`
}

// ListTagOptions are the input parameters used when calling the ListTags API endpoint.
type ListTagOptions struct {
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

	Query string `url:"query,omitempty"`
}

// TagAssignments can be applied teams, users and escalation policies
type TagAssignments struct {
	Add    []*TagAssignment `json:"add,omitempty"`
	Remove []*TagAssignment `json:"remove,omitempty"`
}

// TagAssignment is the structure for assigning tags to an entity
type TagAssignment struct {
	Type  string `json:"type"`
	TagID string `json:"id,omitempty"`
	Label string `json:"label,omitempty"`
}

// ListTags lists tags on your PagerDuty account, optionally filtered by a
// search query. This method currently handles pagination of the response, so
// all tags matched should be present.
//
// Please note that the automatic pagination will be removed in v2 of this
// package, so it's recommended to use ListTagsPaginated() instead.
func (c *Client) ListTags(o ListTagOptions) (*ListTagResponse, error) {
	tags, err := c.ListTagsPaginated(context.Background(), o)
	if err != nil {
		return nil, err
	}

	return &ListTagResponse{Tags: tags}, nil
}

// ListTagsPaginated lists tags on your PagerDuty account, optionally filtered by a search query.
func (c *Client) ListTagsPaginated(ctx context.Context, o ListTagOptions) ([]*Tag, error) {
	tags, err := getTagList(ctx, c, "", "", o)
	if err != nil {
		return nil, err
	}

	return tags, nil
}

// CreateTag creates a new tag.
//
// Deprecated: Use CreateTagWithContext instead.
func (c *Client) CreateTag(t *Tag) (*Tag, error) {
	return c.CreateTagWithContext(context.Background(), t)
}

// CreateTagWithContext creates a new tag.
func (c *Client) CreateTagWithContext(ctx context.Context, t *Tag) (*Tag, error) {
	d := map[string]*Tag{
		"tag": t,
	}

	resp, err := c.post(ctx, "/tags", d, nil)
	return getTagFromResponse(c, resp, err)
}

// DeleteTag removes an existing tag.
//
// Deprecated: Use DeleteTagWithContext instead.
func (c *Client) DeleteTag(id string) error {
	return c.DeleteTagWithContext(context.Background(), id)
}

// DeleteTagWithContext removes an existing tag.
func (c *Client) DeleteTagWithContext(ctx context.Context, id string) error {
	_, err := c.delete(ctx, "/tags/"+id)
	return err
}

// GetTag gets details about an existing tag.
//
// Deprecated: Use GetTagWithContext instead.
func (c *Client) GetTag(id string) (*Tag, error) {
	return c.GetTagWithContext(context.Background(), id)
}

// GetTagWithContext gets details about an existing tag.
func (c *Client) GetTagWithContext(ctx context.Context, id string) (*Tag, error) {
	resp, err := c.get(ctx, "/tags/"+id, nil)
	return getTagFromResponse(c, resp, err)
}

// AssignTags adds and removes tag assignments with entities.
//
// Deprecated: Use AssignTagsWithContext instead.
func (c *Client) AssignTags(e, eid string, a *TagAssignments) error {
	return c.AssignTagsWithContext(context.Background(), e, eid, a)
}

// AssignTagsWithContext adds and removes tag assignments with entities.
// Permitted entity types are users, teams, and escalation_policies.
func (c *Client) AssignTagsWithContext(ctx context.Context, entityType, entityID string, a *TagAssignments) error {
	_, err := c.post(ctx, "/"+entityType+"/"+entityID+"/change_tags", a, nil)
	if err != nil {
		return err
	}

	return nil
}

// GetUsersByTag gets related user references based on the Tag. This method
// currently handles pagination of the response, so all user references with the
// tag should be present.
//
// Please note that the automatic pagination will be removed in v2 of this
// package, so it's recommended to use GetUsersByTagPaginated() instead.
func (c *Client) GetUsersByTag(tid string) (*ListUserResponse, error) {
	objs, err := c.GetUsersByTagPaginated(context.Background(), tid)
	if err != nil {
		return nil, err
	}

	return &ListUserResponse{Users: objs}, nil
}

// GetUsersByTagPaginated gets related user references based on the tag. To get the
// full info of the user, you will need to iterate over the returned slice
// and get that user's details.
func (c *Client) GetUsersByTagPaginated(ctx context.Context, tagID string) ([]*APIObject, error) {
	var users []*APIObject

	// Create a handler closure capable of parsing data from the business_services endpoint
	// and appending resultant business_services to the return slice.
	responseHandler := func(response *http.Response) (APIListObject, error) {
		var result ListUserResponse
		if err := c.decodeJSON(response, &result); err != nil {
			return APIListObject{}, err
		}

		users = append(users, result.Users...)

		// Return stats on the current page. Caller can use this information to
		// adjust for requesting additional pages.
		return APIListObject{
			More:   result.More,
			Offset: result.Offset,
			Limit:  result.Limit,
		}, nil
	}

	// Make call to get all pages associated with the base endpoint.
	if err := c.pagedGet(ctx, "/tags/"+tagID+"/users/", responseHandler); err != nil {
		return nil, err
	}

	return users, nil
}

// GetTeamsByTag gets related teams based on the tag. This method currently
// handles pagination of the response, so all team references with the tag
// should be present.
//
// Please note that the automatic pagination will be removed in v2 of this
// package, so it's recommended to use GetTeamsByTagPaginated() instead.
func (c *Client) GetTeamsByTag(tid string) (*ListTeamsForTagResponse, error) {
	objs, err := c.GetTeamsByTagPaginated(context.Background(), tid)
	if err != nil {
		return nil, err
	}

	return &ListTeamsForTagResponse{Teams: objs}, nil
}

// GetTeamsByTagPaginated gets related teams based on the tag. To get the full
// info of the team, you will need to iterate over the returend slice and get
// that team's details.
func (c *Client) GetTeamsByTagPaginated(ctx context.Context, tagID string) ([]*APIObject, error) {
	var teams []*APIObject

	// Create a handler closure capable of parsing data from the business_services endpoint
	// and appending resultant business_services to the return slice.
	responseHandler := func(response *http.Response) (APIListObject, error) {
		var result ListTeamsForTagResponse
		if err := c.decodeJSON(response, &result); err != nil {
			return APIListObject{}, err
		}

		teams = append(teams, result.Teams...)

		// Return stats on the current page. Caller can use this information to
		// adjust for requesting additional pages.
		return APIListObject{
			More:   result.More,
			Offset: result.Offset,
			Limit:  result.Limit,
		}, nil
	}

	// Make call to get all pages associated with the base endpoint.
	if err := c.pagedGet(ctx, "/tags/"+tagID+"/teams/", responseHandler); err != nil {
		return nil, err
	}

	return teams, nil
}

// GetEscalationPoliciesByTag gets related escalation policies based on the tag.
// This method currently handles pagination of the response, so all escalation
// policy references with the tag should be present.
//
// Please note that the automatic pagination will be removed in v2 of this
// package, so it's recommended to use GetEscalationPoliciesByTagPaginated()
// instead.
func (c *Client) GetEscalationPoliciesByTag(tid string) (*ListEPResponse, error) {
	objs, err := c.GetEscalationPoliciesByTagPaginated(context.Background(), tid)
	if err != nil {
		return nil, err
	}

	return &ListEPResponse{EscalationPolicies: objs}, nil
}

// GetEscalationPoliciesByTagPaginated gets related escalation policies based on
// the tag. To get the full info of the EP, you will need to iterate over the
// returend slice and get that policy's details.
func (c *Client) GetEscalationPoliciesByTagPaginated(ctx context.Context, tagID string) ([]*APIObject, error) {
	var eps []*APIObject

	// Create a handler closure capable of parsing data from the business_services endpoint
	// and appending resultant business_services to the return slice.
	responseHandler := func(response *http.Response) (APIListObject, error) {
		var result ListEPResponse
		if err := c.decodeJSON(response, &result); err != nil {
			return APIListObject{}, err
		}

		eps = append(eps, result.EscalationPolicies...)

		// Return stats on the current page. Caller can use this information to
		// adjust for requesting additional pages.
		return APIListObject{
			More:   result.More,
			Offset: result.Offset,
			Limit:  result.Limit,
		}, nil
	}

	// Make call to get all pages associated with the base endpoint.
	if err := c.pagedGet(ctx, "/tags/"+tagID+"/escalation_policies/", responseHandler); err != nil {
		return nil, err
	}

	return eps, nil
}

// GetTagsForEntity get related tags for Users, Teams or Escalation Policies.
// This method currently handles pagination of the response, so all tags should
// be present.
//
// Please note that the automatic pagination will be removed in v2 of this
// package, so it's recommended to use GetTagsForEntityPaginated() instead.
func (c *Client) GetTagsForEntity(entityType, entityID string, o ListTagOptions) (*ListTagResponse, error) {
	tags, err := c.GetTagsForEntityPaginated(context.Background(), entityType, entityID, o)
	if err != nil {
		return nil, err
	}

	return &ListTagResponse{Tags: tags}, nil
}

// GetTagsForEntityPaginated gets related tags for Users, Teams or Escalation
// Policies.
func (c *Client) GetTagsForEntityPaginated(ctx context.Context, entityType, entityID string, o ListTagOptions) ([]*Tag, error) {
	return getTagList(ctx, c, entityType, entityID, o)
}

func getTagFromResponse(c *Client, resp *http.Response, err error) (*Tag, error) {
	if err != nil {
		return nil, err
	}

	var target map[string]Tag
	if dErr := c.decodeJSON(resp, &target); dErr != nil {
		return nil, fmt.Errorf("Could not decode JSON response: %v", dErr)
	}

	const rootNode = "tag"

	t, nodeOK := target[rootNode]
	if !nodeOK {
		return nil, fmt.Errorf("JSON response does not have %s field", rootNode)
	}

	return &t, nil
}

// getTagList  is a utility function that processes all pages of a ListTagResponse
func getTagList(ctx context.Context, c *Client, entityType, entityID string, o ListTagOptions) ([]*Tag, error) {
	queryParms, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	var tags []*Tag

	// Create a handler closure capable of parsing data from the business_services endpoint
	// and appending resultant business_services to the return slice.
	responseHandler := func(response *http.Response) (APIListObject, error) {
		var result ListTagResponse
		if err := c.decodeJSON(response, &result); err != nil {
			return APIListObject{}, err
		}

		tags = append(tags, result.Tags...)

		// Return stats on the current page. Caller can use this information to
		// adjust for requesting additional pages.
		return APIListObject{
			More:   result.More,
			Offset: result.Offset,
			Limit:  result.Limit,
		}, nil
	}

	path := "/tags"
	if entityType != "" && entityID != "" {
		path = "/" + entityType + "/" + entityID + "/tags"
	}

	// Make call to get all pages associated with the base endpoint.
	if err := c.pagedGet(ctx, path+"?"+queryParms.Encode(), responseHandler); err != nil {
		return nil, err
	}

	return tags, nil
}
