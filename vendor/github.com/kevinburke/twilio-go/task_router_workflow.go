package twilio

import (
	"context"
	"net/url"
)

const WorkflowPathPart = "Workflows"

type WorkflowService struct {
	client       *Client
	workspaceSid string
}

type Workflow struct {
	Sid          string `json:"sid"`
	AccountSid   string `json:"account_sid"`
	FriendlyName string `json:"friendly_name"`
	// A string that contains JSON configuration, for example:
	//
	// `{"task_routing":{"default_filter":{
	//       "queue":"WQ0c1274082082355320d8a41f94eb57aa"
	//  }}}`
	Configuration                 string     `json:"configuration"`
	AssignmentCallbackURL         string     `json:"assignment_callback_url"`
	FallbackAssignmentCallbackURL string     `json:"fallback_assignment_callback_url"`
	TaskReservationTimeout        int        `json:"task_reservation_timeout"`
	DateCreated                   TwilioTime `json:"date_created"`
	DateUpdated                   TwilioTime `json:"date_updated"`
	URL                           string     `json:"url"`
	WorkspaceSid                  string     `json:"workspace_sid"`
}

type WorkflowPage struct {
	Page
	Workflows []*Workflow `json:"workflows"`
}

// Get retrieves a Workflow by its sid.
//
// See https://www.twilio.com/docs/taskrouter/api/workflows#action-get for more.
func (r *WorkflowService) Get(ctx context.Context, sid string) (*Workflow, error) {
	workflow := new(Workflow)
	err := r.client.GetResource(ctx, "Workspaces/"+r.workspaceSid+"/"+WorkflowPathPart, sid, workflow)
	return workflow, err
}

// Create creates a new Workflow.
//
// For a list of valid parameters see
// https://www.twilio.com/docs/taskrouter/api/workflows#action-create.
func (r *WorkflowService) Create(ctx context.Context, data url.Values) (*Workflow, error) {
	workflow := new(Workflow)
	err := r.client.CreateResource(ctx, "Workspaces/"+r.workspaceSid+"/"+WorkflowPathPart, data, workflow)
	return workflow, err
}

// Delete deletes a Workflow.
//
// See https://www.twilio.com/docs/taskrouter/api/workflows#action-delete for more.
func (r *WorkflowService) Delete(ctx context.Context, sid string) error {
	return r.client.DeleteResource(ctx, "Workspaces/"+r.workspaceSid+"/"+WorkflowPathPart, sid)
}

// List all Workflows.
//
// See https://www.twilio.com/docs/taskrouter/api/workflows#action-list for more.
func (ipn *WorkflowService) Update(ctx context.Context, sid string, data url.Values) (*Workflow, error) {
	workflow := new(Workflow)
	err := ipn.client.UpdateResource(ctx, "Workspaces/"+ipn.workspaceSid+"/"+WorkflowPathPart, sid, data, workflow)
	return workflow, err
}

func (ins *WorkflowService) GetPage(ctx context.Context, data url.Values) (*WorkflowPage, error) {
	iter := ins.GetPageIterator(data)
	return iter.Next(ctx)
}

type WorkflowPageIterator struct {
	p *PageIterator
}

func (c *WorkflowService) GetPageIterator(data url.Values) *WorkflowPageIterator {
	iter := NewPageIterator(c.client, data, "Workspaces/"+c.workspaceSid+"/"+WorkflowPathPart)
	return &WorkflowPageIterator{
		p: iter,
	}
}

func (c *WorkflowPageIterator) Next(ctx context.Context) (*WorkflowPage, error) {
	cp := new(WorkflowPage)
	err := c.p.Next(ctx, cp)
	if err != nil {
		return nil, err
	}
	c.p.SetNextPageURI(cp.NextPageURI)
	return cp, nil
}
