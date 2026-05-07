package pagerduty

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-querystring/query"
)

// ResponsePlay represents the API object for a response object:
//
// https://developer.pagerduty.com/api-reference/b3A6Mjc0ODE2Ng-create-a-response-play
type ResponsePlay struct {
	ID                 string          `json:"id,omitempty"`
	Type               string          `json:"type,omitempty"`
	Summary            string          `json:"summary,omitempty"`
	Self               string          `json:"self,omitempty"`
	HTMLURL            string          `json:"html_url,omitempty"`
	Name               string          `json:"name,omitempty"`
	Description        string          `json:"description"`
	Team               *APIReference   `json:"team,omitempty"`
	Subscribers        []*APIReference `json:"subscribers,omitempty"`
	SubscribersMessage string          `json:"subscribers_message"`
	Responders         []*APIReference `json:"responders,omitempty"`
	RespondersMessage  string          `json:"responders_message"`
	Runnability        *string         `json:"runnability"`
	ConferenceNumber   *string         `json:"conference_number"`
	ConferenceURL      *string         `json:"conference_url"`
	ConferenceType     *string         `json:"conference_type"`
}

// ListResponsePlaysResponse represents the list of response plays.
type ListResponsePlaysResponse struct {
	ResponsePlays []ResponsePlay `json:"response_plays"`
}

// ListResponsePlaysOptions are the options for listing response plays.
type ListResponsePlaysOptions struct {
	// FilterForManualRun limits results to show only response plays that can be
	// invoked manually.
	FilterForManualRun bool `url:"filter_for_manual_run,omitempty"`

	Query string `url:"query,omitempty"`

	From string
}

// ListResponsePlays lists existing response plays.
func (c *Client) ListResponsePlays(ctx context.Context, o ListResponsePlaysOptions) ([]ResponsePlay, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	h := map[string]string{
		"From": o.From,
	}

	resp, err := c.get(ctx, "/response_plays?"+v.Encode(), h)
	if err != nil {
		return nil, err
	}

	var result ListResponsePlaysResponse
	if err = c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return result.ResponsePlays, nil
}

// CreateResponsePlay creates a new response play.
func (c *Client) CreateResponsePlay(ctx context.Context, rp ResponsePlay) (ResponsePlay, error) {
	d := map[string]ResponsePlay{
		"response_play": rp,
	}

	resp, err := c.post(ctx, "/response_plays", d, nil)
	return getResponsePlayFromResponse(c, resp, err)
}

// GetResponsePlay gets details about an existing response play.
func (c *Client) GetResponsePlay(ctx context.Context, id string) (ResponsePlay, error) {
	resp, err := c.get(ctx, "/response_plays/"+id, nil)
	return getResponsePlayFromResponse(c, resp, err)
}

// UpdateResponsePlay updates an existing response play.
func (c *Client) UpdateResponsePlay(ctx context.Context, rp ResponsePlay) (ResponsePlay, error) {
	d := map[string]ResponsePlay{
		"response_play": rp,
	}

	resp, err := c.put(ctx, "/response_plays/"+rp.ID, d, nil)
	return getResponsePlayFromResponse(c, resp, err)
}

// DeleteResponsePlay deletes an existing response play.
func (c *Client) DeleteResponsePlay(ctx context.Context, id string) error {
	_, err := c.delete(ctx, "/response_plays/"+id)
	return err
}

// RunResponsePlay runs a response play on a given incident.
func (c *Client) RunResponsePlay(ctx context.Context, from string, responsePlayID string, incidentID string) error {
	d := map[string]APIReference{
		"incident": {
			ID:   incidentID,
			Type: "incident_reference",
		},
	}

	h := map[string]string{
		"From": from,
	}

	resp, err := c.post(ctx, "/response_plays/"+responsePlayID+"/run", d, h)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to run response play %s on incident %s (status code: %d)", responsePlayID, incidentID, resp.StatusCode)
	}

	return nil
}

func getResponsePlayFromResponse(c *Client, resp *http.Response, err error) (ResponsePlay, error) {
	if err != nil {
		return ResponsePlay{}, err
	}

	var target map[string]ResponsePlay
	if dErr := c.decodeJSON(resp, &target); dErr != nil {
		return ResponsePlay{}, fmt.Errorf("Could not decode JSON response: %v", dErr)
	}

	const rootNode = "response_play"

	t, nodeOK := target[rootNode]
	if !nodeOK {
		return ResponsePlay{}, fmt.Errorf("JSON response does not have %s field", rootNode)
	}

	return t, nil
}
