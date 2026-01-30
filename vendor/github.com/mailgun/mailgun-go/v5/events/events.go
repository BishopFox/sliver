package events

import (
	"strings"
	"time"
)

// An EventName is a struct with the event name.
type EventName struct {
	Name string `json:"event"`
}

// GetName returns the name of the event.
func (e *EventName) GetName() string {
	return strings.ToLower(e.Name)
}

func (e *EventName) SetName(name string) {
	e.Name = strings.ToLower(name)
}

type Generic struct {
	EventName
	Timestamp float64 `json:"timestamp"`
	ID        string  `json:"id"`
}

func (g *Generic) GetTimestamp() time.Time {
	return time.Unix(0, int64(g.Timestamp*float64(time.Second))).UTC()
}

func (g *Generic) SetTimestamp(t time.Time) {
	g.Timestamp = float64(t.Unix()) + (float64(t.Nanosecond()/int(time.Microsecond)) / float64(1000000))
}

func (g *Generic) GetID() string {
	return g.ID
}

func (g *Generic) SetID(id string) {
	g.ID = id
}

//
// Message Events
//

type Accepted struct {
	Generic

	Envelope Envelope `json:"envelope"`
	Message  Message  `json:"message"`
	Flags    Flags    `json:"flags"`

	Recipient       string     `json:"recipient"`
	RecipientDomain string     `json:"recipient-domain"`
	Method          string     `json:"method"`
	OriginatingIP   string     `json:"originating-ip"`
	Tags            []string   `json:"tags"`
	Campaigns       []Campaign `json:"campaigns"`
	UserVariables   any        `json:"user-variables"`
	Storage         Storage    `json:"storage"`
}

type Rejected struct {
	Generic

	Reject struct {
		Reason      string `json:"reason"`
		Description string `json:"description"`
	} `json:"reject"`

	Message Message `json:"message"`
	Storage Storage `json:"storage"`
	Flags   Flags   `json:"flags"`

	Tags          []string   `json:"tags"`
	Campaigns     []Campaign `json:"campaigns"`
	UserVariables any        `json:"user-variables"`
}

type Delivered struct {
	Generic

	Envelope Envelope `json:"envelope"`
	Message  Message  `json:"message"`
	Flags    Flags    `json:"flags"`

	Recipient       string     `json:"recipient"`
	RecipientDomain string     `json:"recipient-domain"`
	Method          string     `json:"method"`
	Tags            []string   `json:"tags"`
	Campaigns       []Campaign `json:"campaigns"`
	Storage         Storage    `json:"storage"`

	DeliveryStatus DeliveryStatus `json:"delivery-status"`
	UserVariables  any            `json:"user-variables"`
}

// Failed - Mailgun could not deliver the email to the recipient email server.
// Use for permanent_fail and temporary_fail webhooks.
type Failed struct {
	Generic

	Envelope Envelope `json:"envelope"`
	Message  Message  `json:"message"`
	Flags    Flags    `json:"flags"`

	Recipient       string     `json:"recipient"`
	RecipientDomain string     `json:"recipient-domain"`
	Method          string     `json:"method"`
	Tags            []string   `json:"tags"`
	Campaigns       []Campaign `json:"campaigns"`
	Storage         Storage    `json:"storage"`

	DeliveryStatus DeliveryStatus `json:"delivery-status"`

	// Severity:
	//
	// - permanent when a message is not delivered;
	//
	// - temporary when a message is temporarily rejected by an ESP.
	Severity      string `json:"severity"`
	Reason        string `json:"reason"`
	UserVariables any    `json:"user-variables"`
}

type Stored struct {
	Generic

	Message Message `json:"message"`
	Storage Storage `json:"storage"`
	Flags   Flags   `json:"flags"`

	Tags          []string   `json:"tags"`
	Campaigns     []Campaign `json:"campaigns"`
	UserVariables any        `json:"user-variables"`
}

//
// Message Events (User)
//

type Opened struct {
	Generic

	Message     Message     `json:"message"`
	Campaigns   []Campaign  `json:"campaigns"`
	MailingList MailingList `json:"mailing-list"`

	Recipient       string   `json:"recipient"`
	RecipientDomain string   `json:"recipient-domain"`
	Tags            []string `json:"tags"`

	IP          string      `json:"ip"`
	ClientInfo  ClientInfo  `json:"client-info"`
	GeoLocation GeoLocation `json:"geolocation"`

	UserVariables any `json:"user-variables"`
}

type Clicked struct {
	Generic

	Url string `json:"url"`

	Message     Message     `json:"message"`
	Campaigns   []Campaign  `json:"campaigns"`
	MailingList MailingList `json:"mailing-list"`

	Recipient       string   `json:"recipient"`
	RecipientDomain string   `json:"recipient-domain"`
	Tags            []string `json:"tags"`

	IP          string      `json:"ip"`
	ClientInfo  ClientInfo  `json:"client-info"`
	GeoLocation GeoLocation `json:"geolocation"`

	UserVariables any `json:"user-variables"`
}

type Unsubscribed struct {
	Generic

	Message     Message     `json:"message"`
	Campaigns   []Campaign  `json:"campaigns"`
	MailingList MailingList `json:"mailing-list"`

	Recipient       string   `json:"recipient"`
	RecipientDomain string   `json:"recipient-domain"`
	Tags            []string `json:"tags"`

	IP          string      `json:"ip"`
	ClientInfo  ClientInfo  `json:"client-info"`
	GeoLocation GeoLocation `json:"geolocation"`

	UserVariables any `json:"user-variables"`
}

type Complained struct {
	Generic

	Message   Message    `json:"message"`
	Campaigns []Campaign `json:"campaigns"`

	Recipient     string   `json:"recipient"`
	Tags          []string `json:"tags"`
	UserVariables any      `json:"user-variables"`
}

//
// Mailing List Events
//

type MailingListMember struct {
	Subscribed bool
	Address    string
	Name       string
	Vars       []string
}

type MailingListError struct {
	Message string
}

type ListMemberUploaded struct {
	Generic
	MailingList MailingList       `json:"mailing-list"`
	Member      MailingListMember `json:"member"`
	TaskID      string            `json:"task-id"`
}

type ListMemberUploadError struct {
	Generic
	MailingList       MailingList      `json:"mailing-list"`
	TaskID            string           `json:"task-id"`
	Format            string           `json:"format"`
	MemberDescription string           `json:"member-description"`
	Error             MailingListError `json:"error"`
}

type ListUploaded struct {
	Generic
	MailingList   MailingList       `json:"mailing-list"`
	IsUpsert      bool              `json:"is-upsert"`
	Format        string            `json:"format"`
	UpsertedCount int               `json:"upserted-count"`
	FailedCount   int               `json:"failed-count"`
	Member        MailingListMember `json:"member"`
	Subscribed    bool              `json:"subscribed"`
	TaskID        string            `json:"task-id"`
}

type Paging struct {
	First    string `json:"first,omitempty"`
	Next     string `json:"next,omitempty"`
	Previous string `json:"previous,omitempty"`
	Last     string `json:"last,omitempty"`
}

type RawJSON []byte

func (v *RawJSON) UnmarshalJSON(data []byte) error {
	*v = data
	return nil
}

type Response struct {
	Items  []RawJSON `json:"items"`
	Paging Paging    `json:"paging"`
}
