package twitter

import (
	"net/http"
	"time"

	"github.com/dghubble/sling"
)

// DirectMessageEvents lists Direct Message events.
type DirectMessageEvents struct {
	Events     []DirectMessageEvent `json:"events"`
	NextCursor string               `json:"next_cursor"`
}

// DirectMessageEvent is a single Direct Message sent or received.
type DirectMessageEvent struct {
	CreatedAt string                     `json:"created_timestamp,omitempty"`
	ID        string                     `json:"id,omitempty"`
	Type      string                     `json:"type"`
	Message   *DirectMessageEventMessage `json:"message_create"`
}

// DirectMessageEventMessage contains message contents, along with sender and
// target recipient.
type DirectMessageEventMessage struct {
	SenderID string               `json:"sender_id,omitempty"`
	Target   *DirectMessageTarget `json:"target"`
	Data     *DirectMessageData   `json:"message_data"`
}

// DirectMessageTarget specifies the recipient of a Direct Message event.
type DirectMessageTarget struct {
	RecipientID string `json:"recipient_id"`
}

// DirectMessageData is the message data contained in a Direct Message event.
type DirectMessageData struct {
	Text               string                           `json:"text"`
	Entities           *Entities                        `json:"entities,omitempty"`
	Attachment         *DirectMessageDataAttachment     `json:"attachment,omitempty"`
	QuickReply         *DirectMessageQuickReply         `json:"quick_reply,omitempty"`
	QuickReplyResponse *DirectMessageQuickReplyResponse `json:"quick_reply_response,omitempty"`
	CTAs               []DirectMessageCTA               `json:"ctas,omitempty"`
}

// DirectMessageQuickReplyResponse contains the response from QuickReply.
type DirectMessageQuickReplyResponse struct {
	Type     string `json:"type"`
	Metadata string `json:"metadata"`
}

// DirectMessageDataAttachment contains message data attachments for a Direct
// Message event.
type DirectMessageDataAttachment struct {
	Type  string      `json:"type"`
	Media MediaEntity `json:"media"`
}

// DirectMessageQuickReply contains quick reply data for a Direct Message
// event.
type DirectMessageQuickReply struct {
	Type    string                          `json:"type"`
	Options []DirectMessageQuickReplyOption `json:"options"`
}

// DirectMessageQuickReplyOption represents Option object for
// a Direct Message's Quick Reply.
type DirectMessageQuickReplyOption struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Metadata    string `json:"metadata,omitempty"`
}

// DirectMessageCTA contains CTA data for a Direct Message event.
type DirectMessageCTA struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	URL   string `json:"url"`
}

// DirectMessageService provides methods for accessing Twitter direct message
// API endpoints.
type DirectMessageService struct {
	baseSling *sling.Sling
	sling     *sling.Sling
}

// newDirectMessageService returns a new DirectMessageService.
func newDirectMessageService(sling *sling.Sling) *DirectMessageService {
	return &DirectMessageService{
		baseSling: sling.New(),
		sling:     sling.Path("direct_messages/"),
	}
}

// DirectMessageEventsNewParams are the parameters for
// DirectMessageService.EventsNew
type DirectMessageEventsNewParams struct {
	Event *DirectMessageEvent `json:"event"`
}

// EventsNew publishes a new Direct Message event and returns the event.
// Requires a user auth context with DM scope.
// https://developer.twitter.com/en/docs/direct-messages/sending-and-receiving/api-reference/new-event
func (s *DirectMessageService) EventsNew(params *DirectMessageEventsNewParams) (*DirectMessageEvent, *http.Response, error) {
	// Twitter API wraps the event response
	wrap := &struct {
		Event *DirectMessageEvent `json:"event"`
	}{}
	apiError := new(APIError)
	resp, err := s.sling.New().Post("events/new.json").BodyJSON(params).Receive(wrap, apiError)
	return wrap.Event, resp, relevantError(err, *apiError)
}

// DirectMessageEventsShowParams are the parameters for
// DirectMessageService.EventsShow
type DirectMessageEventsShowParams struct {
	ID string `url:"id,omitempty"`
}

// EventsShow returns a single Direct Message event by id.
// Requires a user auth context with DM scope.
// https://developer.twitter.com/en/docs/direct-messages/sending-and-receiving/api-reference/get-event
func (s *DirectMessageService) EventsShow(id string, params *DirectMessageEventsShowParams) (*DirectMessageEvent, *http.Response, error) {
	if params == nil {
		params = &DirectMessageEventsShowParams{}
	}
	params.ID = id
	// Twitter API wraps the event response
	wrap := &struct {
		Event *DirectMessageEvent `json:"event"`
	}{}
	apiError := new(APIError)
	resp, err := s.sling.New().Get("events/show.json").QueryStruct(params).Receive(wrap, apiError)
	return wrap.Event, resp, relevantError(err, *apiError)
}

// DirectMessageEventsListParams are the parameters for
// DirectMessageService.EventsList
type DirectMessageEventsListParams struct {
	Cursor string `url:"cursor,omitempty"`
	Count  int    `url:"count,omitempty"`
}

// EventsList returns Direct Message events (both sent and received) within
// the last 30 days in reverse chronological order.
// Requires a user auth context with DM scope.
// https://developer.twitter.com/en/docs/direct-messages/sending-and-receiving/api-reference/list-events
func (s *DirectMessageService) EventsList(params *DirectMessageEventsListParams) (*DirectMessageEvents, *http.Response, error) {
	events := new(DirectMessageEvents)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("events/list.json").QueryStruct(params).Receive(events, apiError)
	return events, resp, relevantError(err, *apiError)
}

// EventsDestroy deletes the Direct Message event by id.
// Requires a user auth context with DM scope.
// https://developer.twitter.com/en/docs/direct-messages/sending-and-receiving/api-reference/delete-message-event
func (s *DirectMessageService) EventsDestroy(id string) (*http.Response, error) {
	params := struct {
		ID string `url:"id,omitempty"`
	}{id}
	apiError := new(APIError)
	resp, err := s.sling.New().Delete("events/destroy.json").QueryStruct(params).Receive(nil, apiError)
	return resp, relevantError(err, *apiError)
}

// DEPRECATED

// DirectMessage is a direct message to a single recipient (DEPRECATED).
type DirectMessage struct {
	CreatedAt           string    `json:"created_at"`
	Entities            *Entities `json:"entities"`
	ID                  int64     `json:"id"`
	IDStr               string    `json:"id_str"`
	Recipient           *User     `json:"recipient"`
	RecipientID         int64     `json:"recipient_id"`
	RecipientScreenName string    `json:"recipient_screen_name"`
	Sender              *User     `json:"sender"`
	SenderID            int64     `json:"sender_id"`
	SenderScreenName    string    `json:"sender_screen_name"`
	Text                string    `json:"text"`
}

// CreatedAtTime returns the time a Direct Message was created (DEPRECATED).
func (d DirectMessage) CreatedAtTime() (time.Time, error) {
	return time.Parse(time.RubyDate, d.CreatedAt)
}

// directMessageShowParams are the parameters for DirectMessageService.Show
type directMessageShowParams struct {
	ID int64 `url:"id,omitempty"`
}

// Show returns the requested Direct Message (DEPRECATED).
// Requires a user auth context with DM scope.
// https://dev.twitter.com/rest/reference/get/direct_messages/show
func (s *DirectMessageService) Show(id int64) (*DirectMessage, *http.Response, error) {
	params := &directMessageShowParams{ID: id}
	dm := new(DirectMessage)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("show.json").QueryStruct(params).Receive(dm, apiError)
	return dm, resp, relevantError(err, *apiError)
}

// DirectMessageGetParams are the parameters for DirectMessageService.Get
// (DEPRECATED).
type DirectMessageGetParams struct {
	SinceID         int64 `url:"since_id,omitempty"`
	MaxID           int64 `url:"max_id,omitempty"`
	Count           int   `url:"count,omitempty"`
	IncludeEntities *bool `url:"include_entities,omitempty"`
	SkipStatus      *bool `url:"skip_status,omitempty"`
}

// Get returns recent Direct Messages received by the authenticated user
// (DEPRECATED).
// Requires a user auth context with DM scope.
// https://dev.twitter.com/rest/reference/get/direct_messages
func (s *DirectMessageService) Get(params *DirectMessageGetParams) ([]DirectMessage, *http.Response, error) {
	dms := new([]DirectMessage)
	apiError := new(APIError)
	resp, err := s.baseSling.New().Get("direct_messages.json").QueryStruct(params).Receive(dms, apiError)
	return *dms, resp, relevantError(err, *apiError)
}

// DirectMessageSentParams are the parameters for DirectMessageService.Sent
// (DEPRECATED).
type DirectMessageSentParams struct {
	SinceID         int64 `url:"since_id,omitempty"`
	MaxID           int64 `url:"max_id,omitempty"`
	Count           int   `url:"count,omitempty"`
	Page            int   `url:"page,omitempty"`
	IncludeEntities *bool `url:"include_entities,omitempty"`
}

// Sent returns recent Direct Messages sent by the authenticated user
// (DEPRECATED).
// Requires a user auth context with DM scope.
// https://dev.twitter.com/rest/reference/get/direct_messages/sent
func (s *DirectMessageService) Sent(params *DirectMessageSentParams) ([]DirectMessage, *http.Response, error) {
	dms := new([]DirectMessage)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("sent.json").QueryStruct(params).Receive(dms, apiError)
	return *dms, resp, relevantError(err, *apiError)
}

// DirectMessageNewParams are the parameters for DirectMessageService.New
// (DEPRECATED).
type DirectMessageNewParams struct {
	UserID     int64  `url:"user_id,omitempty"`
	ScreenName string `url:"screen_name,omitempty"`
	Text       string `url:"text"`
}

// New sends a new Direct Message to a specified user as the authenticated
// user (DEPRECATED).
// Requires a user auth context with DM scope.
// https://dev.twitter.com/rest/reference/post/direct_messages/new
func (s *DirectMessageService) New(params *DirectMessageNewParams) (*DirectMessage, *http.Response, error) {
	dm := new(DirectMessage)
	apiError := new(APIError)
	resp, err := s.sling.New().Post("new.json").BodyForm(params).Receive(dm, apiError)
	return dm, resp, relevantError(err, *apiError)
}

// DirectMessageDestroyParams are the parameters for DirectMessageService.Destroy
// (DEPRECATED).
type DirectMessageDestroyParams struct {
	ID              int64 `url:"id,omitempty"`
	IncludeEntities *bool `url:"include_entities,omitempty"`
}

// Destroy deletes the Direct Message with the given id and returns it if
// successful (DEPRECATED).
// Requires a user auth context with DM scope.
// https://dev.twitter.com/rest/reference/post/direct_messages/destroy
func (s *DirectMessageService) Destroy(id int64, params *DirectMessageDestroyParams) (*DirectMessage, *http.Response, error) {
	if params == nil {
		params = &DirectMessageDestroyParams{}
	}
	params.ID = id
	dm := new(DirectMessage)
	apiError := new(APIError)
	resp, err := s.sling.New().Post("destroy.json").BodyForm(params).Receive(dm, apiError)
	return dm, resp, relevantError(err, *apiError)
}
