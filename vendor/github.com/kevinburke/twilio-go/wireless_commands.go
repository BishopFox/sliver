package twilio

import (
	"context"
	"net/url"
)

const commandPathPart = "Commands"

type CommandService struct {
	client *Client
}

// Command represents a Command resource.
type Command struct {
	Sid       string    `json:"sid"`
	SimSid    string    `json:"sim_sid"`
	Command   string    `json:"command"`
	Direction Direction `json:"direction"`
	// A string representing which mode to send the SMS message using.
	// May be "text" or "binary". If omitted, the default SMS mode is text.
	CommandMode string `json:"command_mode"`
	Status      Status `json:"status"`

	DateCreated TwilioTime `json:"date_created"`
	DateUpdated TwilioTime `json:"date_updated"`
	AccountSid  string     `json:"account_sid"`
	URL         string     `json:"url"`
}

func (c *CommandService) Get(ctx context.Context, sid string) (*Command, error) {
	cmd := new(Command)
	err := c.client.GetResource(ctx, commandPathPart, sid, cmd)
	return cmd, err
}

// CommandPage represents a page of Commands.
type CommandPage struct {
	Meta     Meta       `json:"meta"`
	Commands []*Command `json:"commands"`
}

// GetPage returns a single Page of resources, filtered by data.
//
// See https://www.twilio.com/docs/api/wireless/rest-api/command#list-get
func (f *CommandService) GetPage(ctx context.Context, data url.Values) (*CommandPage, error) {
	return f.GetPageIterator(data).Next(ctx)
}

// GetPageIterator returns a CommandPageIterator with the given page
// filters. Call iterator.Next() to get the first page of resources (and again
// to retrieve subsequent pages).
func (f *CommandService) GetPageIterator(data url.Values) CommandPageIterator {
	iter := NewPageIterator(f.client, data, commandPathPart)
	return &commandPageIterator{
		p: iter,
	}
}

type CommandPageIterator interface {
	// Next returns the next page of resources. If there are no more resources,
	// NoMoreResults is returned.
	Next(context.Context) (*CommandPage, error)
}

type commandPageIterator struct {
	p *PageIterator
}

func (i *commandPageIterator) Next(ctx context.Context) (*CommandPage, error) {
	ap := new(CommandPage)
	err := i.p.Next(ctx, ap)
	if err != nil {
		return nil, err
	}
	i.p.SetNextPageURI(ap.Meta.NextPageURL)
	return ap, nil
}

// Create a new command. Command creation is asynchronous.
func (c *CommandService) Create(ctx context.Context, data url.Values) (*Command, error) {
	cmd := new(Command)
	err := c.client.CreateResource(ctx, commandPathPart, data, cmd)
	return cmd, err
}

// Send a command to a device identified by sid. This is a wrapper around
// Create(); to provide optional parameters, use Create directly.
func (c *CommandService) Send(ctx context.Context, sid string, text string) (*Command, error) {
	v := url.Values{}
	v.Set("Sim", sid)
	v.Set("Command", text)
	return c.Create(context.Background(), v)
}
