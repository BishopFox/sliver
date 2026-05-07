package mailgun

import (
	"context"

	"github.com/mailgun/mailgun-go/v5/mtypes"
)

type IPWarmupsIterator struct {
	mtypes.ListIPWarmupsResponse
	mg  Mailgun
	err error
}

// Err if an error occurred during iteration `Err()` will return non nil
func (ri *IPWarmupsIterator) Err() error {
	return ri.err
}

// Next retrieves the next page of items from the api. Returns false when there
// no more pages to retrieve or if there was an error. Use `.Err()` to retrieve
// the error
func (ri *IPWarmupsIterator) Next(ctx context.Context, items *[]mtypes.IPWarmup) bool {
	if ri.err != nil {
		return false
	}

	ri.err = ri.fetch(ctx, ri.Paging.Next)
	if ri.err != nil {
		return false
	}

	cpy := make([]mtypes.IPWarmup, len(ri.Items))
	copy(cpy, ri.Items)
	*items = cpy
	return len(ri.Items) != 0
}

// First retrieves the first page of items from the api. Returns false if there
// was an error. It also sets the iterator object to the first page.
// Use `.Err()` to retrieve the error.
func (ri *IPWarmupsIterator) First(ctx context.Context, items *[]mtypes.IPWarmup) bool {
	if ri.err != nil {
		return false
	}
	ri.err = ri.fetch(ctx, ri.Paging.First)
	if ri.err != nil {
		return false
	}
	cpy := make([]mtypes.IPWarmup, len(ri.Items))
	copy(cpy, ri.Items)
	*items = cpy
	return true
}

func (ri *IPWarmupsIterator) fetch(ctx context.Context, url string) error {
	ri.Items = nil
	r := newHTTPRequest(url)
	r.setBasicAuth(basicAuthUser, ri.mg.APIKey())
	r.setClient(ri.mg.HTTPClient())

	return getResponseFromJSON(ctx, r, &ri.ListIPWarmupsResponse)
}

// ListIPWarmups retrieves a list of warmups in progress in the account
func (mg *Client) ListIPWarmups() *IPWarmupsIterator {
	url := generateApiUrl(mg, 3, ipWarmupsEndpoint)
	return &IPWarmupsIterator{
		mg:                    mg,
		ListIPWarmupsResponse: mtypes.ListIPWarmupsResponse{Paging: mtypes.Paging{Next: url, First: url}},
	}
}

// GetIPWarmup retrieves the details of a warmup in progress for the specified IP address
func (mg *Client) GetIPWarmup(ctx context.Context, ip string) (mtypes.IPWarmupDetails, error) {
	url := generateApiUrl(mg, 3, ipWarmupsEndpoint) + "/" + ip
	r := newHTTPRequest(url)
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	r.setClient(mg.HTTPClient())
	var resp mtypes.IPWarmupDetailsResponse
	if err := getResponseFromJSON(ctx, r, &resp); err != nil {
		return resp.Details, err
	}
	return resp.Details, nil
}

func (mg *Client) CreateIPWarmup(ctx context.Context, ip string) error {
	url := generateApiUrl(mg, 3, ipWarmupsEndpoint) + "/" + ip
	r := newHTTPRequest(url)
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	r.setClient(mg.HTTPClient())
	_, err := makePostRequest(ctx, r, nil)
	return err
}

func (mg *Client) DeleteIPWarmup(ctx context.Context, ip string) error {
	url := generateApiUrl(mg, 3, ipWarmupsEndpoint) + "/" + ip
	r := newHTTPRequest(url)
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	r.setClient(mg.HTTPClient())
	_, err := makeDeleteRequest(ctx, r)
	return err
}
