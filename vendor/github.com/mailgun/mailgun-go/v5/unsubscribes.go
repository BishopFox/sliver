package mailgun

import (
	"context"
	"net/url"
	"strconv"

	"github.com/mailgun/mailgun-go/v5/mtypes"
)

// ListUnsubscribes fetches the list of unsubscribes
func (mg *Client) ListUnsubscribes(domain string, opts *ListOptions) *UnsubscribesIterator {
	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, unsubscribesEndpoint, domain))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	if opts != nil {
		if opts.Limit != 0 {
			r.addParameter("limit", strconv.Itoa(opts.Limit))
		}
	}
	uri, err := r.generateUrlWithParameters()
	return &UnsubscribesIterator{
		mg: mg,
		// TODO(vtopc): why is Next and First both set to the same URL?
		ListUnsubscribesResponse: mtypes.ListUnsubscribesResponse{Paging: mtypes.Paging{Next: uri, First: uri}},
		err:                      err,
	}
}

type UnsubscribesIterator struct {
	mtypes.ListUnsubscribesResponse
	mg  Mailgun
	err error
}

// Err if an error occurred during iteration `Err()` will return non nil
func (ci *UnsubscribesIterator) Err() error {
	return ci.err
}

// Next retrieves the next page of items from the api. Returns false when there are
// no more pages to retrieve or if there was an error. Use `.Err()` to retrieve
// the error
func (ci *UnsubscribesIterator) Next(ctx context.Context, items *[]mtypes.Unsubscribe) bool {
	if ci.err != nil {
		return false
	}
	ci.err = ci.fetch(ctx, ci.Paging.Next)
	if ci.err != nil {
		return false
	}
	cpy := make([]mtypes.Unsubscribe, len(ci.Items))
	copy(cpy, ci.Items)
	*items = cpy

	return len(ci.Items) != 0
}

// First retrieves the first page of items from the api. Returns false if there
// was an error. It also sets the iterator object to the first page.
// Use `.Err()` to retrieve the error.
func (ci *UnsubscribesIterator) First(ctx context.Context, items *[]mtypes.Unsubscribe) bool {
	if ci.err != nil {
		return false
	}
	ci.err = ci.fetch(ctx, ci.Paging.First)
	if ci.err != nil {
		return false
	}
	cpy := make([]mtypes.Unsubscribe, len(ci.Items))
	copy(cpy, ci.Items)
	*items = cpy
	return true
}

// Last retrieves the last page of items from the api.
// Calling Last() is invalid unless you first call First() or Next()
// Returns false if there was an error. It also sets the iterator object
// to the last page. Use `.Err()` to retrieve the error.
func (ci *UnsubscribesIterator) Last(ctx context.Context, items *[]mtypes.Unsubscribe) bool {
	if ci.err != nil {
		return false
	}
	ci.err = ci.fetch(ctx, ci.Paging.Last)
	if ci.err != nil {
		return false
	}
	cpy := make([]mtypes.Unsubscribe, len(ci.Items))
	copy(cpy, ci.Items)
	*items = cpy
	return true
}

// Previous retrieves the previous page of items from the api. Returns false when there
// no more pages to retrieve or if there was an error. Use `.Err()` to retrieve
// the error if any
func (ci *UnsubscribesIterator) Previous(ctx context.Context, items *[]mtypes.Unsubscribe) bool {
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
	cpy := make([]mtypes.Unsubscribe, len(ci.Items))
	copy(cpy, ci.Items)
	*items = cpy

	return len(ci.Items) != 0
}

func (ci *UnsubscribesIterator) fetch(ctx context.Context, uri string) error {
	ci.Items = nil
	r := newHTTPRequest(uri)
	r.setClient(ci.mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, ci.mg.APIKey())

	return getResponseFromJSON(ctx, r, &ci.ListUnsubscribesResponse)
}

// GetUnsubscribe retrieves a single unsubscribe record.
// Can be used to check if a given address is present in the list of unsubscribed users.
func (mg *Client) GetUnsubscribe(ctx context.Context, domain, address string) (mtypes.Unsubscribe, error) {
	r := newHTTPRequest(generateApiV3UrlWithTarget(mg, unsubscribesEndpoint, domain, url.QueryEscape(address)))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	envelope := mtypes.Unsubscribe{}
	err := getResponseFromJSON(ctx, r, &envelope)

	return envelope, err
}

// CreateUnsubscribe adds an e-mail address to the domain's unsubscription table.
func (mg *Client) CreateUnsubscribe(ctx context.Context, domain, address, tag string) error {
	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, unsubscribesEndpoint, domain))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	p := newUrlEncodedPayload()
	p.addValue("address", address)
	p.addValue("tag", tag)
	_, err := makePostRequest(ctx, r, p)
	return err
}

// CreateUnsubscribes adds multiple e-mail addresses to the domain's unsubscription table.
// TODO(vtopc): Doc says it's domain ID, not name. Rename arg to clarify.
//
//	https://documentation.mailgun.com/docs/mailgun/api-reference/send/mailgun/unsubscribe
func (mg *Client) CreateUnsubscribes(ctx context.Context, domain string, unsubscribes []mtypes.Unsubscribe) error {
	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, unsubscribesEndpoint, domain))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	r.addHeader("Content-Type", "application/json")

	p := newJSONEncodedPayload(unsubscribes)
	_, err := makePostRequest(ctx, r, p)
	return err
}

// DeleteUnsubscribe removes the e-mail address given from the domain's unsubscription table.
// If passing in an ID (discoverable from, e.g., ListUnsubscribes()), the e-mail address associated
// with the given ID will be removed.
func (mg *Client) DeleteUnsubscribe(ctx context.Context, domain, address string) error {
	r := newHTTPRequest(generateApiV3UrlWithTarget(mg, unsubscribesEndpoint, domain, url.QueryEscape(address)))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	_, err := makeDeleteRequest(ctx, r)
	return err
}

// DeleteUnsubscribeWithTag removes the e-mail address given from the domain's unsubscription table with a matching tag.
// If passing in an ID (discoverable from, e.g., ListUnsubscribes()), the e-mail address associated
// with the given ID will be removed.
func (mg *Client) DeleteUnsubscribeWithTag(ctx context.Context, domain, address, tag string) error {
	r := newHTTPRequest(generateApiV3UrlWithTarget(mg, unsubscribesEndpoint, domain, url.QueryEscape(address)))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	r.addParameter("tag", tag)
	_, err := makeDeleteRequest(ctx, r)
	return err
}
