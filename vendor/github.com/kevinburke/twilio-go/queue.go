package twilio

import (
	"context"
	"net/url"
)

const queuePathPart = "Queues"

type QueueService struct {
	client *Client
}

type Queue struct {
	Sid             string     `json:"sid"`
	AverageWaitTime uint       `json:"average_wait_time"`
	CurrentSize     uint       `json:"current_size"`
	FriendlyName    string     `json:"friendly_name"`
	MaxSize         uint       `json:"max_size"`
	DateCreated     TwilioTime `json:"date_created"`
	DateUpdated     TwilioTime `json:"date_updated"`
	AccountSid      string     `json:"account_sid"`
	URI             string     `json:"uri"`
}

type QueuePage struct {
	Page
	Queues []*Queue
}

// Get returns a single Queue or an error.
func (c *QueueService) Get(ctx context.Context, sid string) (*Queue, error) {
	queue := new(Queue)
	err := c.client.GetResource(ctx, queuePathPart, sid, queue)
	return queue, err
}

// Create a new Queue.
func (c *QueueService) Create(ctx context.Context, data url.Values) (*Queue, error) {
	queue := new(Queue)
	err := c.client.CreateResource(ctx, queuePathPart, data, queue)
	return queue, err
}

// Delete the Queue with the given sid. If the Queue has
// already been deleted, or does not exist, Delete returns nil. If another
// error or a timeout occurs, the error is returned.
func (c *QueueService) Delete(ctx context.Context, sid string) error {
	return c.client.DeleteResource(ctx, queuePathPart, sid)
}

func (c *QueueService) GetPage(ctx context.Context, data url.Values) (*QueuePage, error) {
	iter := c.GetPageIterator(data)
	return iter.Next(ctx)
}

type QueuePageIterator struct {
	p *PageIterator
}

// GetPageIterator returns a QueuePageIterator with the given page filters.
// Call iterator.Next() to get the first page of resources (and again to
// retrieve subsequent pages).
func (c *QueueService) GetPageIterator(data url.Values) *QueuePageIterator {
	iter := NewPageIterator(c.client, data, queuePathPart)
	return &QueuePageIterator{
		p: iter,
	}
}

// Next returns the next page of resources. If there are no more resources,
// NoMoreResults is returned.
func (c *QueuePageIterator) Next(ctx context.Context) (*QueuePage, error) {
	qp := new(QueuePage)
	err := c.p.Next(ctx, qp)
	if err != nil {
		return nil, err
	}
	c.p.SetNextPageURI(qp.NextPageURI)
	return qp, nil
}
