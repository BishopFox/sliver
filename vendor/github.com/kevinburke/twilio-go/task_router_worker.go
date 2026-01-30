package twilio

import (
	"context"
	"net/url"
)

const WorkersPathPart = "Workers"

type WorkerService struct {
	client       *Client
	workspaceSid string
}

type Worker struct {
	Sid          string `json:"sid"`
	AccountSid   string `json:"account_sid"`
	FriendlyName string `json:"friendly_name"`
	// A string that contains JSON attributes, for example:
	// `{"type": "support"}`
	Attributes   string     `json:"attributes"`
	ActivityName string     `json:"activity_name"`
	ActivitySid  string     `json:"activity_sid"`
	Available    bool       `json:"available"`
	DateCreated  TwilioTime `json:"date_created"`
	DateUpdated  TwilioTime `json:"date_updated"`
	URL          string     `json:"url"`
	WorkspaceSid string     `json:"workspace_sid"`
}

type WorkerPage struct {
	Page
	Workers []*Worker `json:"workers"`
}

// Get retrieves a Worker by its sid.
//
// See https://www.twilio.com/docs/taskrouter/api/workers#action-get for more.
func (r *WorkerService) Get(ctx context.Context, sid string) (*Worker, error) {
	worker := new(Worker)
	err := r.client.GetResource(ctx, "Workspaces/"+r.workspaceSid+"/"+WorkersPathPart, sid, worker)
	return worker, err
}

// Create creates a new Worker.
//
// For a list of valid parameters see
// https://www.twilio.com/docs/taskrouter/api/workers#action-create.
func (r *WorkerService) Create(ctx context.Context, data url.Values) (*Worker, error) {
	worker := new(Worker)
	err := r.client.CreateResource(ctx, "Workspaces/"+r.workspaceSid+"/"+WorkersPathPart, data, worker)
	return worker, err
}

// Delete deletes a Worker.
//
// See https://www.twilio.com/docs/taskrouter/api/workers#action-delete for more.
func (r *WorkerService) Delete(ctx context.Context, sid string) error {
	return r.client.DeleteResource(ctx, "Workspaces/"+r.workspaceSid+"/"+WorkersPathPart, sid)
}

// Update updates a Workers.
//
// See https://www.twilio.com/docs/taskrouter/api/workers#update-a-worker for more.
func (ipn *WorkerService) Update(ctx context.Context, sid string, data url.Values) (*Worker, error) {
	worker := new(Worker)
	err := ipn.client.UpdateResource(ctx, "Workspaces/"+ipn.workspaceSid+"/"+WorkersPathPart, sid, data, worker)
	return worker, err
}

func (ins *WorkerService) GetPage(ctx context.Context, data url.Values) (*WorkerPage, error) {
	iter := ins.GetPageIterator(data)
	return iter.Next(ctx)
}

type WorkerPageIterator struct {
	p *PageIterator
}

func (c *WorkerService) GetPageIterator(data url.Values) *WorkerPageIterator {
	iter := NewPageIterator(c.client, data, "Workspaces/"+c.workspaceSid+"/"+WorkersPathPart)
	return &WorkerPageIterator{
		p: iter,
	}
}

func (c *WorkerPageIterator) Next(ctx context.Context) (*WorkerPage, error) {
	cp := new(WorkerPage)
	err := c.p.Next(ctx, cp)
	if err != nil {
		return nil, err
	}
	c.p.SetNextPageURI(cp.NextPageURI)
	return cp, nil
}
