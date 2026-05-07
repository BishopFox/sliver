package mailgun

import (
	"context"
	"net/url"
	"strconv"

	"github.com/mailgun/mailgun-go/v5/mtypes"
)

const (
	complaintsEndpoint = "complaints"
)

// ListComplaints returns a set of spam complaints registered against your domain.
// Recipients of your messages can click on a link which sends feedback to Mailgun
// indicating that the message they received is, to them, spam.
func (mg *Client) ListComplaints(domain string, opts *ListOptions) *ComplaintsIterator {
	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, complaintsEndpoint, domain))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	if opts != nil {
		if opts.Limit != 0 {
			r.addParameter("limit", strconv.Itoa(opts.Limit))
		}
	}
	uri, err := r.generateUrlWithParameters()
	return &ComplaintsIterator{
		mg:                 mg,
		ComplaintsResponse: mtypes.ComplaintsResponse{Paging: mtypes.Paging{Next: uri, First: uri}},
		err:                err,
	}
}

type ComplaintsIterator struct {
	mtypes.ComplaintsResponse
	mg  Mailgun
	err error
}

// If an error occurred during iteration `Err()` will return non nil
func (ci *ComplaintsIterator) Err() error {
	return ci.err
}

// Next retrieves the next page of items from the api. Returns false when there
// no more pages to retrieve or if there was an error. Use `.Err()` to retrieve
// the error
func (ci *ComplaintsIterator) Next(ctx context.Context, items *[]mtypes.Complaint) bool {
	if ci.err != nil {
		return false
	}
	ci.err = ci.fetch(ctx, ci.Paging.Next)
	if ci.err != nil {
		return false
	}
	cpy := make([]mtypes.Complaint, len(ci.Items))
	copy(cpy, ci.Items)
	*items = cpy

	return len(ci.Items) != 0
}

// First retrieves the first page of items from the api. Returns false if there
// was an error. It also sets the iterator object to the first page.
// Use `.Err()` to retrieve the error.
func (ci *ComplaintsIterator) First(ctx context.Context, items *[]mtypes.Complaint) bool {
	if ci.err != nil {
		return false
	}
	ci.err = ci.fetch(ctx, ci.Paging.First)
	if ci.err != nil {
		return false
	}
	cpy := make([]mtypes.Complaint, len(ci.Items))
	copy(cpy, ci.Items)
	*items = cpy
	return true
}

// Last retrieves the last page of items from the api.
// Calling Last() is invalid unless you first call First() or Next()
// Returns false if there was an error. It also sets the iterator object
// to the last page. Use `.Err()` to retrieve the error.
func (ci *ComplaintsIterator) Last(ctx context.Context, items *[]mtypes.Complaint) bool {
	if ci.err != nil {
		return false
	}
	ci.err = ci.fetch(ctx, ci.Paging.Last)
	if ci.err != nil {
		return false
	}
	cpy := make([]mtypes.Complaint, len(ci.Items))
	copy(cpy, ci.Items)
	*items = cpy
	return true
}

// Previous retrieves the previous page of items from the api. Returns false when there
// no more pages to retrieve or if there was an error. Use `.Err()` to retrieve
// the error if any
func (ci *ComplaintsIterator) Previous(ctx context.Context, items *[]mtypes.Complaint) bool {
	if ci.err != nil {
		return false
	}
	if ci.Paging.Previous == "" {
		return false
	}
	ci.err = ci.fetch(ctx, ci.Paging.Previous)
	if ci.err != nil {
		return false
	}
	cpy := make([]mtypes.Complaint, len(ci.Items))
	copy(cpy, ci.Items)
	*items = cpy

	return len(ci.Items) != 0
}

func (ci *ComplaintsIterator) fetch(ctx context.Context, uri string) error {
	ci.Items = nil
	r := newHTTPRequest(uri)
	r.setClient(ci.mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, ci.mg.APIKey())

	return getResponseFromJSON(ctx, r, &ci.ComplaintsResponse)
}

// GetComplaint returns a single complaint record filed by a recipient at the email address provided.
// If no complaint exists, the Complaint instance returned will be empty.
func (mg *Client) GetComplaint(ctx context.Context, domain, address string) (mtypes.Complaint, error) {
	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, complaintsEndpoint, domain) + "/" + url.QueryEscape(address))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	var c mtypes.Complaint
	err := getResponseFromJSON(ctx, r, &c)
	return c, err
}

// CreateComplaint registers the specified address as a recipient who has complained of receiving spam
// from your domain.
func (mg *Client) CreateComplaint(ctx context.Context, domain, address string) error {
	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, complaintsEndpoint, domain))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	p := newUrlEncodedPayload()
	p.addValue("address", address)
	_, err := makePostRequest(ctx, r, p)
	return err
}

func (mg *Client) CreateComplaints(ctx context.Context, domain string, addresses []string) error {
	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, complaintsEndpoint, domain))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	body := make([]map[string]string, len(addresses))
	for i, addr := range addresses {
		body[i] = map[string]string{"address": addr}
	}

	payload := newJSONEncodedPayload(body)

	_, err := makePostRequest(ctx, r, payload)
	return err
}

// DeleteComplaint removes a previously registered e-mail address from the list of people who complained
// of receiving spam from your domain.
func (mg *Client) DeleteComplaint(ctx context.Context, domain, address string) error {
	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, complaintsEndpoint, domain) + "/" + url.QueryEscape(address))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	_, err := makeDeleteRequest(ctx, r)
	return err
}
