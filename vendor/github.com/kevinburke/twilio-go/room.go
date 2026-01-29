package twilio

import (
	"context"
	"net/url"
)

const roomPathPart = "Rooms"

type RoomService struct {
	client *Client
}

type Room struct {
	Sid                         string            `json:"sid"`
	AccountSid                  string            `json:"account_sid"`
	Type                        string            `json:"type"`
	EnableTurn                  bool              `json:"enable_turn"`
	UniqueName                  string            `json:"unique_name"`
	StatusCallback              string            `json:"status_callback"`
	StatusCallbackMethod        string            `json:"status_callback_method"`
	MaxParticipants             uint              `json:"max_participants"`
	RecordParticipantsOnConnect bool              `json:"record_participants_on_connect"`
	Duration                    uint              `json:"duration"`
	MediaRegion                 string            `json:"media_region"`
	Status                      Status            `json:"status"`
	DateCreated                 TwilioTime        `json:"date_created"`
	DateUpdated                 TwilioTime        `json:"date_updated"`
	EndTime                     TwilioTime        `json:"end_time"`
	URL                         string            `json:"url"`
	Links                       map[string]string `json:"links"`
}

type RoomPage struct {
	Meta  Meta    `json:"meta"`
	Rooms []*Room `json:"rooms"`
}

type RoomPageIterator struct {
	p *PageIterator
}

// Get finds a single Room resource by its sid or unique name, or returns an error.
func (r *RoomService) Get(ctx context.Context, sidOrUniqueName string) (*Room, error) {
	room := new(Room)
	err := r.client.GetResource(ctx, roomPathPart, sidOrUniqueName, room)
	return room, err
}

// Complete an in-progress Room with the given sid. All connected
// Participants will be immediately disconnected from the Room.
func (r *RoomService) Complete(sid string) (*Room, error) {
	room := new(Room)
	v := url.Values{}
	v.Set("Status", string(StatusCompleted))
	err := r.client.UpdateResource(context.Background(), roomPathPart, sid, v, room)
	return room, err
}

// Create a room with the given url.Values. For more information on valid values,
// see https://www.twilio.com/docs/api/video/rooms-resource#post-parameters or use the
func (r *RoomService) Create(ctx context.Context, data url.Values) (*Room, error) {
	room := new(Room)
	err := r.client.CreateResource(ctx, roomPathPart, data, room)
	return room, err
}

// Returns a list of rooms. For more information on valid values,
// see https://www.twilio.com/docs/api/video/rooms-resource#get-list-resource
func (r *RoomService) GetPage(ctx context.Context, data url.Values) (*RoomPage, error) {
	return r.GetPageIterator(data).Next(ctx)
}

// GetPageIterator returns an iterator which can be used to retrieve pages.
func (r *RoomService) GetPageIterator(data url.Values) *RoomPageIterator {
	iter := NewPageIterator(r.client, data, roomPathPart)
	return &RoomPageIterator{
		p: iter,
	}
}

// Next returns the next page of resources. If there are no more resources,
// NoMoreResults is returned.
func (r *RoomPageIterator) Next(ctx context.Context) (*RoomPage, error) {
	rp := new(RoomPage)
	err := r.p.Next(ctx, rp)
	if err != nil {
		return nil, err
	}
	r.p.SetNextPageURI(rp.Meta.NextPageURL)
	return rp, nil
}
