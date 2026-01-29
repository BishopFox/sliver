package reddit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/go-querystring/query"
)

// MessageService handles communication with the message
// related methods of the Reddit API.
//
// Reddit API docs: https://www.reddit.com/dev/api/#section_messages
type MessageService struct {
	client *Client
}

// Message is a message.
type Message struct {
	ID      string     `json:"id"`
	FullID  string     `json:"name"`
	Created *Timestamp `json:"created_utc"`

	Subject  string `json:"subject"`
	Text     string `json:"body"`
	ParentID string `json:"parent_id"`

	Author string `json:"author"`
	To     string `json:"dest"`

	IsComment bool `json:"was_comment"`
}

type inboxThing struct {
	Kind string   `json:"kind"`
	Data *Message `json:"data"`
}

type inboxListing struct {
	inboxThings
	after string
}

func (l *inboxListing) After() string {
	return l.after
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (l *inboxListing) UnmarshalJSON(b []byte) error {
	root := new(struct {
		Data struct {
			Things inboxThings `json:"children"`
			After  string      `json:"after"`
		} `json:"data"`
	})

	err := json.Unmarshal(b, root)
	if err != nil {
		return err
	}

	l.inboxThings = root.Data.Things
	l.after = root.Data.After

	return nil
}

type inboxThings struct {
	Comments []*Message
	Messages []*Message
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (t *inboxThings) UnmarshalJSON(b []byte) error {
	var things []inboxThing
	if err := json.Unmarshal(b, &things); err != nil {
		return err
	}

	t.add(things...)
	return nil
}

func (t *inboxThings) add(things ...inboxThing) {
	for _, thing := range things {
		switch thing.Kind {
		case kindComment:
			t.Comments = append(t.Comments, thing.Data)
		case kindMessage:
			t.Messages = append(t.Messages, thing.Data)
		}
	}
}

// SendMessageRequest represents a request to send a message.
type SendMessageRequest struct {
	// Username, or /r/name for that subreddit's moderators.
	To      string `url:"to"`
	Subject string `url:"subject"`
	Text    string `url:"text"`
	// Optional. If specified, the message will look like it came from the subreddit.
	FromSubreddit string `url:"from_sr,omitempty"`
}

// ReadAll marks all messages/comments as read. It queues up the task on Reddit's end.
// A successful response returns 202 to acknowledge acceptance of the request.
// This endpoint is heavily rate limited.
func (s *MessageService) ReadAll(ctx context.Context) (*Response, error) {
	path := "api/read_all_messages"
	req, err := s.client.NewRequest(http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

// Read marks a message/comment as read via its full ID.
func (s *MessageService) Read(ctx context.Context, ids ...string) (*Response, error) {
	if len(ids) == 0 {
		return nil, errors.New("must provide at least 1 id")
	}

	path := "api/read_message"

	form := url.Values{}
	form.Set("id", strings.Join(ids, ","))

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Unread marks a message/comment as unread via its full ID.
func (s *MessageService) Unread(ctx context.Context, ids ...string) (*Response, error) {
	if len(ids) == 0 {
		return nil, errors.New("must provide at least 1 id")
	}

	path := "api/unread_message"

	form := url.Values{}
	form.Set("id", strings.Join(ids, ","))

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Block the author of a post, comment or message via its full ID.
func (s *MessageService) Block(ctx context.Context, id string) (*Response, error) {
	path := "api/block"

	form := url.Values{}
	form.Set("id", id)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Collapse messages.
func (s *MessageService) Collapse(ctx context.Context, ids ...string) (*Response, error) {
	if len(ids) == 0 {
		return nil, errors.New("must provide at least 1 id")
	}

	path := "api/collapse_message"

	form := url.Values{}
	form.Set("id", strings.Join(ids, ","))

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Uncollapse messages.
func (s *MessageService) Uncollapse(ctx context.Context, ids ...string) (*Response, error) {
	if len(ids) == 0 {
		return nil, errors.New("must provide at least 1 id")
	}

	path := "api/uncollapse_message"

	form := url.Values{}
	form.Set("id", strings.Join(ids, ","))

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Delete a message.
func (s *MessageService) Delete(ctx context.Context, id string) (*Response, error) {
	path := "api/del_msg"

	form := url.Values{}
	form.Set("id", id)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Send a message.
func (s *MessageService) Send(ctx context.Context, sendRequest *SendMessageRequest) (*Response, error) {
	if sendRequest == nil {
		return nil, errors.New("*SendMessageRequest: cannot be nil")
	}

	path := "api/compose"

	form, err := query.Values(sendRequest)
	if err != nil {
		return nil, err
	}
	form.Set("api_type", "json")

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Inbox returns comments and messages that appear in your inbox, respectively.
func (s *MessageService) Inbox(ctx context.Context, opts *ListOptions) ([]*Message, []*Message, *Response, error) {
	root, resp, err := s.inbox(ctx, "message/inbox", opts)
	if err != nil {
		return nil, nil, resp, err
	}
	return root.Comments, root.Messages, resp, nil
}

// InboxUnread returns unread comments and messages that appear in your inbox, respectively.
func (s *MessageService) InboxUnread(ctx context.Context, opts *ListOptions) ([]*Message, []*Message, *Response, error) {
	root, resp, err := s.inbox(ctx, "message/unread", opts)
	if err != nil {
		return nil, nil, resp, err
	}
	return root.Comments, root.Messages, resp, nil
}

// Sent returns messages that you've sent.
func (s *MessageService) Sent(ctx context.Context, opts *ListOptions) ([]*Message, *Response, error) {
	root, resp, err := s.inbox(ctx, "message/sent", opts)
	if err != nil {
		return nil, resp, err
	}
	return root.Messages, resp, nil
}

func (s *MessageService) inbox(ctx context.Context, path string, opts *ListOptions) (*inboxListing, *Response, error) {
	path, err := addOptions(path, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(inboxListing)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, nil, err
	}

	return root, resp, nil
}
