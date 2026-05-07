package pagerduty

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/go-querystring/query"
)

// Agent is the actor who carried out the action.
type Agent APIObject

// Channel is the means by which the action was carried out.
type Channel struct {
	Type string
	Raw  map[string]interface{}
}

// Context are to be included with the trigger such as links to graphs or images.
type Context struct {
	Alt  string
	Href string
	Src  string
	Text string
	Type string
}

// CommonLogEntryField is the list of shared log entry between Incident and LogEntry
type CommonLogEntryField struct {
	APIObject
	CreatedAt              string            `json:"created_at,omitempty"`
	Agent                  Agent             `json:"agent,omitempty"`
	Channel                Channel           `json:"channel,omitempty"`
	Teams                  []Team            `json:"teams,omitempty"`
	Contexts               []Context         `json:"contexts,omitempty"`
	AcknowledgementTimeout int               `json:"acknowledgement_timeout"`
	EventDetails           map[string]string `json:"event_details,omitempty"`
	Assignees              []APIObject       `json:"assignees,omitempty"`
}

// LogEntry is a list of all of the events that happened to an incident.
type LogEntry struct {
	CommonLogEntryField
	Incident Incident  `json:"incident"`
	Service  APIObject `json:"service"`
	User     APIObject `json:"user"`
}

// ListLogEntryResponse is the response data when calling the ListLogEntry API endpoint.
type ListLogEntryResponse struct {
	APIListObject
	LogEntries []LogEntry `json:"log_entries"`
}

// ListLogEntriesOptions is the data structure used when calling the ListLogEntry API endpoint.
type ListLogEntriesOptions struct {
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

	TimeZone   string   `url:"time_zone,omitempty"`
	Since      string   `url:"since,omitempty"`
	Until      string   `url:"until,omitempty"`
	IsOverview bool     `url:"is_overview,omitempty"`
	Includes   []string `url:"include,omitempty,brackets"`
	TeamIDs    []string `url:"team_ids,omitempty,brackets"`
}

// ListLogEntries lists all of the incident log entries across the entire
// account.
//
// Deprecated: Use ListLogEntriesWithContext instead.
func (c *Client) ListLogEntries(o ListLogEntriesOptions) (*ListLogEntryResponse, error) {
	return c.ListLogEntriesWithContext(context.Background(), o)
}

// ListLogEntriesWithContext lists all of the incident log entries across the entire account.
func (c *Client) ListLogEntriesWithContext(ctx context.Context, o ListLogEntriesOptions) (*ListLogEntryResponse, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, "/log_entries?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var result ListLogEntryResponse
	if err = c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &result, err
}

// GetLogEntryOptions is the data structure used when calling the GetLogEntry API endpoint.
type GetLogEntryOptions struct {
	TimeZone string   `url:"time_zone,omitempty"`
	Includes []string `url:"include,omitempty,brackets"`
}

// GetLogEntry list log entries for the specified incident.
//
// Deprecated: Use GetLogEntryWithContext instead.
func (c *Client) GetLogEntry(id string, o GetLogEntryOptions) (*LogEntry, error) {
	return c.GetLogEntryWithContext(context.Background(), id, o)
}

// GetLogEntryWithContext list log entries for the specified incident.
func (c *Client) GetLogEntryWithContext(ctx context.Context, id string, o GetLogEntryOptions) (*LogEntry, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, "/log_entries/"+id+"?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var result map[string]LogEntry
	if err := c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	le, ok := result["log_entry"]
	if !ok {
		return nil, fmt.Errorf("JSON response does not have log_entry field")
	}

	return &le, nil
}

// UnmarshalJSON Expands the LogEntry.Channel object to parse out a raw value
func (c *Channel) UnmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	ct, ok := raw["type"]
	if ok {
		c.Type = ct.(string)
		c.Raw = raw
	}

	return nil
}

// MarshalJSON Expands the LogEntry.Channel object to correctly marshal it back
func (c *Channel) MarshalJSON() ([]byte, error) {
	raw := map[string]interface{}{}
	if c != nil && c.Type != "" {
		for k, v := range c.Raw {
			raw[k] = v
		}
	}

	return json.Marshal(raw)
}
