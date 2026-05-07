package twilio

import (
	"context"
	"net/url"
)

const activityPathPart = "Activities"

type ActivityService struct {
	client       *Client
	workspaceSid string
}

type Activity struct {
	Sid          string     `json:"sid"`
	AccountSid   string     `json:"account_sid"`
	FriendlyName string     `json:"friendly_name"`
	Available    bool       `json:"available"`
	DateCreated  TwilioTime `json:"date_created"`
	DateUpdated  TwilioTime `json:"date_updated"`
	URL          string     `json:"url"`
	WorkspaceSid string     `json:"workspace_sid"`
}

type ActivityPage struct {
	Page
	Activities []*Activity `json:"activities"`
}

// Get retrieves an Activity by its sid.
//
// See https://www.twilio.com/docs/taskrouter/api/activities#action-get for
// more.
func (r *ActivityService) Get(ctx context.Context, sid string) (*Activity, error) {
	activity := new(Activity)
	err := r.client.GetResource(ctx, "Workspaces/"+r.workspaceSid+"/"+activityPathPart, sid, activity)
	return activity, err
}

// Create creates a new Activity.
//
// For a list of valid parameters see
// https://www.twilio.com/docs/taskrouter/api/activities#action-create.
func (r *ActivityService) Create(ctx context.Context, data url.Values) (*Activity, error) {
	activity := new(Activity)
	err := r.client.CreateResource(ctx, "Workspaces/"+r.workspaceSid+"/"+activityPathPart, data, activity)
	return activity, err
}

// Delete deletes an Activity.
//
// See https://www.twilio.com/docs/taskrouter/api/activities#action-delete for
// more.
func (r *ActivityService) Delete(ctx context.Context, sid string) error {
	return r.client.DeleteResource(ctx, "Workspaces/"+r.workspaceSid+"/"+activityPathPart, sid)
}

func (ipn *ActivityService) Update(ctx context.Context, sid string, data url.Values) (*Activity, error) {
	activity := new(Activity)
	err := ipn.client.UpdateResource(ctx, "Workspaces/"+ipn.workspaceSid+"/"+activityPathPart, sid, data, activity)
	return activity, err
}

// GetPage retrieves an ActivityPage, filtered by the given data.
func (ins *ActivityService) GetPage(ctx context.Context, data url.Values) (*ActivityPage, error) {
	iter := ins.GetPageIterator(data)
	return iter.Next(ctx)
}

type ActivityPageIterator struct {
	p *PageIterator
}

// GetPageIterator returns an iterator which can be used to retrieve pages.
func (c *ActivityService) GetPageIterator(data url.Values) *ActivityPageIterator {
	iter := NewPageIterator(c.client, data, "Workspaces/"+c.workspaceSid+"/"+activityPathPart)
	return &ActivityPageIterator{
		p: iter,
	}
}

// Next returns the next page of resources. If there are no more resources,
// NoMoreResults is returned.
func (c *ActivityPageIterator) Next(ctx context.Context) (*ActivityPage, error) {
	cp := new(ActivityPage)
	err := c.p.Next(ctx, cp)
	if err != nil {
		return nil, err
	}
	c.p.SetNextPageURI(cp.NextPageURI)
	return cp, nil
}
