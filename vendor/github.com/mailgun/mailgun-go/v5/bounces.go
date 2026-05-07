package mailgun

import (
	"context"
	"net/url"
	"strconv"

	"github.com/mailgun/mailgun-go/v5/mtypes"
)

// ListBounces returns a complete set of bounces logged against the sender's domain, if any.
// The results include the total number of bounces (regardless of skip or limit settings),
// and the slice of bounces specified, if successful.
// Note that the length of the slice may be smaller than the total number of bounces.
func (mg *Client) ListBounces(domain string, opts *ListOptions) *BouncesIterator {
	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, bouncesEndpoint, domain))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	if opts != nil {
		if opts.Limit != 0 {
			r.addParameter("limit", strconv.Itoa(opts.Limit))
		}
	}
	uri, err := r.generateUrlWithParameters()
	return &BouncesIterator{
		mg:                  mg,
		BouncesListResponse: mtypes.BouncesListResponse{Paging: mtypes.Paging{Next: uri, First: uri}},
		err:                 err,
	}
}

type BouncesIterator struct {
	mtypes.BouncesListResponse
	mg  Mailgun
	err error
}

// Err if an error occurred during iteration `Err()` will return non nil
func (ci *BouncesIterator) Err() error {
	return ci.err
}

// Next retrieves the next page of items from the api. Returns false when there
// no more pages to retrieve or if there was an error. Use `.Err()` to retrieve
// the error
func (ci *BouncesIterator) Next(ctx context.Context, items *[]mtypes.Bounce) bool {
	if ci.err != nil {
		return false
	}
	ci.err = ci.fetch(ctx, ci.Paging.Next)
	if ci.err != nil {
		return false
	}
	cpy := make([]mtypes.Bounce, len(ci.Items))
	copy(cpy, ci.Items)
	*items = cpy

	return len(ci.Items) != 0
}

// First retrieves the first page of items from the api. Returns false if there
// was an error. It also sets the iterator object to the first page.
// Use `.Err()` to retrieve the error.
func (ci *BouncesIterator) First(ctx context.Context, items *[]mtypes.Bounce) bool {
	if ci.err != nil {
		return false
	}
	ci.err = ci.fetch(ctx, ci.Paging.First)
	if ci.err != nil {
		return false
	}
	cpy := make([]mtypes.Bounce, len(ci.Items))
	copy(cpy, ci.Items)
	*items = cpy
	return true
}

// Last retrieves the last page of items from the api.
// Calling Last() is invalid unless you first call First() or Next()
// Returns false if there was an error. It also sets the iterator object
// to the last page. Use `.Err()` to retrieve the error.
func (ci *BouncesIterator) Last(ctx context.Context, items *[]mtypes.Bounce) bool {
	if ci.err != nil {
		return false
	}
	ci.err = ci.fetch(ctx, ci.Paging.Last)
	if ci.err != nil {
		return false
	}
	cpy := make([]mtypes.Bounce, len(ci.Items))
	copy(cpy, ci.Items)
	*items = cpy
	return true
}

// Previous retrieves the previous page of items from the api. Returns false when there
// no more pages to retrieve or if there was an error. Use `.Err()` to retrieve
// the error if any
func (ci *BouncesIterator) Previous(ctx context.Context, items *[]mtypes.Bounce) bool {
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
	cpy := make([]mtypes.Bounce, len(ci.Items))
	copy(cpy, ci.Items)
	*items = cpy

	return len(ci.Items) != 0
}

func (ci *BouncesIterator) fetch(ctx context.Context, uri string) error {
	ci.Items = nil
	r := newHTTPRequest(uri)
	r.setClient(ci.mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, ci.mg.APIKey())

	return getResponseFromJSON(ctx, r, &ci.BouncesListResponse)
}

// GetBounce retrieves a single bounce record, if any exist, for the given recipient address.
func (mg *Client) GetBounce(ctx context.Context, domain, address string) (mtypes.Bounce, error) {
	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, bouncesEndpoint, domain) + "/" + url.QueryEscape(address))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	var response mtypes.Bounce
	err := getResponseFromJSON(ctx, r, &response)
	return response, err
}

// AddBounce files a bounce report.
// Address identifies the intended recipient of the message that bounced.
// Code corresponds to the numeric response given by the e-mail server which rejected the message.
// Error provides the corresponding human-readable reason for the problem.
// For example,
// here's how the these two fields relate.
// Suppose the SMTP server responds with an error, as below.
// Then, ...
//
//	 550  Requested action not taken: mailbox unavailable
//	\___/\_______________________________________________/
//	  |                         |
//	  `-- Code                  `-- Error
//
// Note that both code and error exist as strings, even though
// code will report as a number.
func (mg *Client) AddBounce(ctx context.Context, domain, address, code, bounceError string) error {
	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, bouncesEndpoint, domain))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	payload := newUrlEncodedPayload()
	payload.addValue("address", address)
	if code != "" {
		payload.addValue("code", code)
	}
	if bounceError != "" {
		payload.addValue("error", bounceError)
	}
	_, err := makePostRequest(ctx, r, payload)
	return err
}

// AddBounces adds a list of bounces to the bounce list
func (mg *Client) AddBounces(ctx context.Context, domain string, bounces []mtypes.Bounce) error {
	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, bouncesEndpoint, domain))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	payload := newJSONEncodedPayload(bounces)

	_, err := makePostRequest(ctx, r, payload)
	return err
}

// DeleteBounce removes all bounces associated with the provided e-mail address.
func (mg *Client) DeleteBounce(ctx context.Context, domain, address string) error {
	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, bouncesEndpoint, domain) + "/" + url.QueryEscape(address))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	_, err := makeDeleteRequest(ctx, r)
	return err
}

// DeleteBounceList removes all bounces in the bounce list
func (mg *Client) DeleteBounceList(ctx context.Context, domain string) error {
	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, bouncesEndpoint, domain))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	_, err := makeDeleteRequest(ctx, r)
	return err
}
