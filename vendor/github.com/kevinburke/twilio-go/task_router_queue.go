package twilio

import (
	"context"
	"net/url"
)

const TaskQueuePathPart = "TaskQueues"

type TaskQueueService struct {
	client       *Client
	workspaceSid string
}

type TaskQueue struct {
	Sid                     string     `json:"sid"`
	AccountSid              string     `json:"account_sid"`
	FriendlyName            string     `json:"friendly_name"`
	AssignmentActivityName  string     `json:"assignment_activity_name"`
	AssignmentActivitySid   string     `json:"assignment_activity_sid"`
	ReservationActivityName string     `json:"reservation_activity_name"`
	ReservationActivitySid  string     `json:"reservation_activity_sid"`
	TargetWorkers           string     `json:"target_workers"`
	TaskOrder               string     `json:"task_order"`
	DateCreated             TwilioTime `json:"date_created"`
	DateUpdated             TwilioTime `json:"date_updated"`
	URL                     string     `json:"url"`
	WorkspaceSid            string     `json:"workspace_sid"`
	MaxReservedWorkers      int        `json:"max_reserved_workers"`
}

type TaskQueuePage struct {
	Page
	TaskQueues []*TaskQueue `json:"task_queues"`
}

func (r *TaskQueueService) Get(ctx context.Context, sid string) (*TaskQueue, error) {
	queue := new(TaskQueue)
	err := r.client.GetResource(ctx, "Workspaces/"+r.workspaceSid+"/"+TaskQueuePathPart, sid, queue)
	return queue, err
}

func (r *TaskQueueService) Create(ctx context.Context, data url.Values) (*TaskQueue, error) {
	queue := new(TaskQueue)
	err := r.client.CreateResource(ctx, "Workspaces/"+r.workspaceSid+"/"+TaskQueuePathPart, data, queue)
	return queue, err
}

func (r *TaskQueueService) Delete(ctx context.Context, sid string) error {
	return r.client.DeleteResource(ctx, "Workspaces/"+r.workspaceSid+"/"+TaskQueuePathPart, sid)
}

func (ipn *TaskQueueService) Update(ctx context.Context, sid string, data url.Values) (*TaskQueue, error) {
	queue := new(TaskQueue)
	err := ipn.client.UpdateResource(ctx, "Workspaces/"+ipn.workspaceSid+"/"+TaskQueuePathPart, sid, data, queue)
	return queue, err
}

// GetPage retrieves an TaskQueuePage, filtered by the given data.
func (ins *TaskQueueService) GetPage(ctx context.Context, data url.Values) (*TaskQueuePage, error) {
	iter := ins.GetPageIterator(data)
	return iter.Next(ctx)
}

type TaskQueuePageIterator struct {
	p *PageIterator
}

// GetPageIterator returns an iterator which can be used to retrieve pages.
func (c *TaskQueueService) GetPageIterator(data url.Values) *TaskQueuePageIterator {
	iter := NewPageIterator(c.client, data, "Workspaces/"+c.workspaceSid+"/"+TaskQueuePathPart)
	return &TaskQueuePageIterator{
		p: iter,
	}
}

// Next returns the next page of resources. If there are no more resources,
// NoMoreResults is returned.
func (c *TaskQueuePageIterator) Next(ctx context.Context) (*TaskQueuePage, error) {
	cp := new(TaskQueuePage)
	err := c.p.Next(ctx, cp)
	if err != nil {
		return nil, err
	}
	c.p.SetNextPageURI(cp.NextPageURI)
	return cp, nil
}
