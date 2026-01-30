package twilio

import (
	"context"
	"fmt"
	"net/url"
)

const EventsPathPart = "Events"

type CallEventsService struct {
	callSid string
	client  *Client
}

type CallEventsPage struct {
	Meta   Meta        `json:"meta"`
	Events []CallEvent `json:"events"`
}

type CallEvent struct {
	AccountSid string     `json:"account_sid"`
	CallSid    string     `json:"call_sid"`
	Edge       string     `json:"edge"`
	Group      string     `json:"group"`
	Level      string     `json:"level"`
	Name       string     `json:"name"`
	Timestamp  TwilioTime `json:"timestamp"`
}

// Returns a list of events for a call. For more information on valid values,
// See https://www.twilio.com/docs/voice/insights/api/call-events-resource#get-call-events
func (s *CallEventsService) GetPage(ctx context.Context, data url.Values) (*CallEventsPage, error) {
	return s.GetPageIterator(data).Next(ctx)
}

type CallEventsPageIterator struct {
	p *PageIterator
}

func (s *CallEventsService) GetPageIterator(data url.Values) *CallEventsPageIterator {
	iter := NewPageIterator(s.client, data, fmt.Sprintf("Voice/%s/%s", s.callSid, EventsPathPart))
	return &CallEventsPageIterator{
		p: iter,
	}
}

func (c *CallEventsPageIterator) Next(ctx context.Context) (*CallEventsPage, error) {
	cp := new(CallEventsPage)
	err := c.p.Next(ctx, cp)
	if err != nil {
		return nil, err
	}
	c.p.SetNextPageURI(cp.Meta.NextPageURL)
	return cp, nil
}
