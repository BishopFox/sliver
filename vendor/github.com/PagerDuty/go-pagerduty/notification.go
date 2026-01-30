package pagerduty

import (
	"context"

	"github.com/google/go-querystring/query"
)

// Notification is a message containing the details of the incident.
type Notification struct {
	ID                string    `json:"id"`
	Type              string    `json:"type"`
	StartedAt         string    `json:"started_at"`
	Address           string    `json:"address"`
	User              APIObject `json:"user"`
	ConferenceAddress string    `json:"conferenceAddress"`
	Status            string    `json:"status"`
}

// ListNotificationOptions is the data structure used when calling the ListNotifications API endpoint.
type ListNotificationOptions struct {
	// Limit is the pagination parameter that limits the number of results per
	// page. PagerDuty defaults this value to 25 if omitted, and sets an upper
	// bound of 100.
	Limit uint `url:"limit,omitempty"`

	// Offset is the pagination parameter that specifies the offset at which to
	// start pagination results. When trying to request the next page of
	// results, the new Offset value should be currentOffset + Limit.
	Offset uint `url:"offset,omitempty"`

	// Total is the pagination parameter to request that the API return the
	// total count of items in the response. If this field is omitted or set to
	// false, the total number of results will not be sent back from the PagerDuty API.
	//
	// Setting this to true will slow down the API response times, and so it's
	// recommended to omit it unless you've a specific reason for wanting the
	// total count of items in the collection.
	Total bool `url:"total,omitempty"`

	TimeZone string   `url:"time_zone,omitempty"`
	Since    string   `url:"since,omitempty"`
	Until    string   `url:"until,omitempty"`
	Filter   string   `url:"filter,omitempty"`
	Includes []string `url:"include,omitempty,brackets"`
}

// ListNotificationsResponse is the data structure returned from the ListNotifications API endpoint.
type ListNotificationsResponse struct {
	APIListObject
	Notifications []Notification
}

// ListNotifications lists notifications for a given time range, optionally
// filtered by type (sms_notification, email_notification, phone_notification,
// or push_notification).
//
// Deprecated: Use ListNotificationsWithContext instead.
func (c *Client) ListNotifications(o ListNotificationOptions) (*ListNotificationsResponse, error) {
	return c.ListNotificationsWithContext(context.Background(), o)
}

// ListNotificationsWithContext lists notifications for a given time range,
// optionally filtered by type (sms_notification, email_notification,
// phone_notification, or push_notification).
func (c *Client) ListNotificationsWithContext(ctx context.Context, o ListNotificationOptions) (*ListNotificationsResponse, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, "/notifications?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var result ListNotificationsResponse
	if err := c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
