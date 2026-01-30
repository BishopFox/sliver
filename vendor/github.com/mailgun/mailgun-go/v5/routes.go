package mailgun

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"time"

	"github.com/mailgun/mailgun-go/v5/mtypes"
)

// ForwardedMessage represents the payload the server will get on match
// You can use ExtractForwardRoute() to extract PostForm into the struct, or you can use only the struct and parse the form manually
type ForwardedMessage struct {
	BodyPlain      string            // body-plain
	From           string            // from
	MessageHeaders map[string]string // message-headers
	Recipient      string            // recipient
	Sender         string            // sender
	Signature      string            // signature
	StrippedHTML   string            // stripped-html
	StrippedText   string            // stripped-text
	Subject        string            // subject
	Timestamp      time.Time         // timestamp
	Token          string            // token
}

// ExtractForwardedMessage extracts the forward route payload values from a parsed PostForm
// Example usage:
//
//	func Handler(w http.ResponseWriter, r *http.Request) {
//	err := r.ParseForm()
//	if err != nil {
//		log.Fatal(err)
//	}
//	forwardRoute := mailgun.ExtractForwardedMessage(r.PostForm)
//	fmt.Printf("Forwarded message: %#v", forwardRoute)
//	}
func ExtractForwardedMessage(formValues url.Values) ForwardedMessage {
	forwardedMessage := ForwardedMessage{}
	forwardedMessage.BodyPlain = formValues.Get("body-plain")
	forwardedMessage.From = formValues.Get("from")
	forwardedMessage.Recipient = formValues.Get("recipient")
	forwardedMessage.Sender = formValues.Get("sender")
	forwardedMessage.Signature = formValues.Get("signature")
	forwardedMessage.StrippedHTML = formValues.Get("stripped-html")
	forwardedMessage.StrippedText = formValues.Get("stripped-text")
	forwardedMessage.Subject = formValues.Get("subject")
	forwardedMessage.Token = formValues.Get("token")

	timestampStr := formValues.Get("timestamp")
	timeInt, err := strconv.Atoi(timestampStr)
	if err != nil {
		timeInt = 0
	}
	forwardedMessage.Timestamp = time.Unix(int64(timeInt), 0)

	headersStr := formValues.Get("message-headers")
	headersParsed := make([][]string, 0)
	messageHeaders := make(map[string]string)
	err = json.Unmarshal([]byte(headersStr), &headersParsed)
	if err == nil {
		for _, header := range headersParsed {
			if len(header) < 2 {
				continue
			}
			messageHeaders[header[0]] = header[1]
		}
	}
	forwardedMessage.MessageHeaders = messageHeaders

	return forwardedMessage
}

// ListRoutes allows you to iterate through a list of routes returned by the API
func (mg *Client) ListRoutes(opts *ListOptions) *RoutesIterator {
	var limit int
	if opts != nil {
		limit = opts.Limit
	}

	if limit == 0 {
		limit = 100
	}

	return &RoutesIterator{
		mg:                 mg,
		url:                generateApiUrl(mg, 3, routesEndpoint),
		RoutesListResponse: mtypes.RoutesListResponse{TotalCount: -1},
		limit:              limit,
	}
}

type RoutesIterator struct {
	mtypes.RoutesListResponse

	limit  int
	mg     Mailgun
	offset int
	url    string
	err    error
}

// If an error occurred during iteration `Err()` will return non nil
func (ri *RoutesIterator) Err() error {
	return ri.err
}

// Offset returns the current offset of the iterator
func (ri *RoutesIterator) Offset() int {
	return ri.offset
}

// Next retrieves the next page of items from the api. Returns false when there
// no more pages to retrieve or if there was an error. Use `.Err()` to retrieve
// the error
func (ri *RoutesIterator) Next(ctx context.Context, items *[]mtypes.Route) bool {
	if ri.err != nil {
		return false
	}

	ri.err = ri.fetch(ctx, ri.offset, ri.limit)
	if ri.err != nil {
		return false
	}

	cpy := make([]mtypes.Route, len(ri.Items))
	copy(cpy, ri.Items)
	*items = cpy
	if len(ri.Items) == 0 {
		return false
	}
	ri.offset += len(ri.Items)
	return true
}

// First retrieves the first page of items from the api. Returns false if there
// was an error. It also sets the iterator object to the first page.
// Use `.Err()` to retrieve the error.
func (ri *RoutesIterator) First(ctx context.Context, items *[]mtypes.Route) bool {
	if ri.err != nil {
		return false
	}
	ri.err = ri.fetch(ctx, 0, ri.limit)
	if ri.err != nil {
		return false
	}
	cpy := make([]mtypes.Route, len(ri.Items))
	copy(cpy, ri.Items)
	*items = cpy
	ri.offset = len(ri.Items)
	return true
}

// Last retrieves the last page of items from the api.
// Calling Last() is invalid unless you first call First() or Next()
// Returns false if there was an error. It also sets the iterator object
// to the last page. Use `.Err()` to retrieve the error.
func (ri *RoutesIterator) Last(ctx context.Context, items *[]mtypes.Route) bool {
	if ri.err != nil {
		return false
	}

	if ri.TotalCount == -1 {
		return false
	}

	ri.offset = ri.TotalCount - ri.limit
	if ri.offset < 0 {
		ri.offset = 0
	}

	ri.err = ri.fetch(ctx, ri.offset, ri.limit)
	if ri.err != nil {
		return false
	}
	cpy := make([]mtypes.Route, len(ri.Items))
	copy(cpy, ri.Items)
	*items = cpy
	return true
}

// Previous retrieves the previous page of items from the api. Returns false when there
// no more pages to retrieve or if there was an error. Use `.Err()` to retrieve
// the error if any
func (ri *RoutesIterator) Previous(ctx context.Context, items *[]mtypes.Route) bool {
	if ri.err != nil {
		return false
	}

	if ri.TotalCount == -1 {
		return false
	}

	ri.offset -= ri.limit * 2
	if ri.offset < 0 {
		ri.offset = 0
	}

	ri.err = ri.fetch(ctx, ri.offset, ri.limit)
	if ri.err != nil {
		return false
	}
	cpy := make([]mtypes.Route, len(ri.Items))
	copy(cpy, ri.Items)
	*items = cpy

	return len(ri.Items) != 0
}

func (ri *RoutesIterator) fetch(ctx context.Context, skip, limit int) error {
	ri.Items = nil
	r := newHTTPRequest(ri.url)
	r.setBasicAuth(basicAuthUser, ri.mg.APIKey())
	r.setClient(ri.mg.HTTPClient())

	if skip != 0 {
		r.addParameter("skip", strconv.Itoa(skip))
	}
	if limit != 0 {
		r.addParameter("limit", strconv.Itoa(limit))
	}

	return getResponseFromJSON(ctx, r, &ri.RoutesListResponse)
}

// CreateRoute installs a new route for your domain.
// The route structure you provide serves as a template, and
// only a subset of the fields influence the operation.
// See the Route structure definition for more details.
func (mg *Client) CreateRoute(ctx context.Context, route mtypes.Route) (_ mtypes.Route, err error) {
	r := newHTTPRequest(generateApiUrl(mg, 3, routesEndpoint))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	p := newUrlEncodedPayload()
	p.addValue("priority", strconv.Itoa(route.Priority))
	p.addValue("description", route.Description)
	p.addValue("expression", route.Expression)
	for _, action := range route.Actions {
		p.addValue("action", action)
	}
	var resp mtypes.CreateRouteResp
	if err := postResponseFromJSON(ctx, r, p, &resp); err != nil {
		return mtypes.Route{}, err
	}

	return resp.Route, nil
}

// DeleteRoute removes the specified route from your domain's configuration.
// To avoid ambiguity, Mailgun identifies the route by unique ID.
// See the Route structure definition and the Mailgun API documentation for more details.
func (mg *Client) DeleteRoute(ctx context.Context, id string) error {
	r := newHTTPRequest(generateApiUrl(mg, 3, routesEndpoint) + "/" + id)
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	_, err := makeDeleteRequest(ctx, r)
	return err
}

// GetRoute retrieves the complete route definition associated with the unique route ID.
func (mg *Client) GetRoute(ctx context.Context, id string) (mtypes.Route, error) {
	r := newHTTPRequest(generateApiUrl(mg, 3, routesEndpoint) + "/" + id)
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	var envelope struct {
		Message       string `json:"message"`
		*mtypes.Route `json:"route"`
	}
	err := getResponseFromJSON(ctx, r, &envelope)
	if err != nil {
		return mtypes.Route{}, err
	}

	return *envelope.Route, err
}

// UpdateRoute provides an "in-place" update of the specified route.
// Only those route fields which are non-zero or non-empty are updated.
// All other fields remain as-is.
func (mg *Client) UpdateRoute(ctx context.Context, id string, route mtypes.Route) (mtypes.Route, error) {
	r := newHTTPRequest(generateApiUrl(mg, 3, routesEndpoint) + "/" + id)
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	p := newUrlEncodedPayload()
	if route.Priority != 0 {
		p.addValue("priority", strconv.Itoa(route.Priority))
	}
	if route.Description != "" {
		p.addValue("description", route.Description)
	}
	if route.Expression != "" {
		p.addValue("expression", route.Expression)
	}
	if route.Actions != nil {
		for _, action := range route.Actions {
			p.addValue("action", action)
		}
	}
	// For some reason, this API function just returns a bare Route on success.
	// Unsure why this is the case; it seems like it ought to be a bug.
	var envelope mtypes.Route
	err := putResponseFromJSON(ctx, r, p, &envelope)
	return envelope, err
}
