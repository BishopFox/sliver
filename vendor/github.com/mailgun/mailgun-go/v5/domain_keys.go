package mailgun

// TODO(vtopc): split file into domain_keys and (dkim|all|account)_keys files.

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/mailgun/mailgun-go/v5/mtypes"
)

type ListAllDomainsKeysOptions struct {
	Limit int // NOTE: currently ignored by Mailgun API
}

type CreateDomainKeyOptions struct {
	Bits int
	PEM  string
}

// AllDomainsKeysIterator is a list iterator for
// https://documentation.mailgun.com/docs/mailgun/api-reference/send/mailgun/domain-keys/get-v1-dkim-keys
type AllDomainsKeysIterator struct {
	mtypes.ListAllDomainsKeysResponse

	limit int
	mg    Mailgun
	url   string
	err   error
}

// DomainKeysIterator is a list iterator for
// https://documentation.mailgun.com/docs/mailgun/api-reference/send/mailgun/domain-keys/get-v4-domains--authority-name--keys
//
// TODO(v6): domain can have at most 5 domain keys, so it makes no sense to have an iterator here.
type DomainKeysIterator struct {
	mtypes.ListDomainKeysResponse

	mg      Mailgun
	uri     string
	domain  string
	err     error
	isFirst bool
}

// ListAllDomainsKeys retrieves a set of domain keys from Mailgun.
// https://documentation.mailgun.com/docs/mailgun/api-reference/send/mailgun/domain-keys/get-v1-dkim-keys
func (mg *Client) ListAllDomainsKeys(opts *ListAllDomainsKeysOptions) *AllDomainsKeysIterator {
	var limit int
	if opts != nil {
		limit = opts.Limit
	}

	if limit == 0 {
		limit = 100
	}
	return &AllDomainsKeysIterator{
		mg:                         mg,
		url:                        generateApiUrl(mg, 1, dkimEndpoint),
		ListAllDomainsKeysResponse: mtypes.ListAllDomainsKeysResponse{TotalCount: -1},
		limit:                      limit,
	}
}

// Err if an error occurred during iteration `Err()` will return non nil
func (ri *AllDomainsKeysIterator) Err() error {
	return ri.err
}

// Next retrieves the next page of items from the api. Returns false when there
// are no more pages to retrieve or if there was an error. Use `.Err()` to retrieve
// the error
func (ri *AllDomainsKeysIterator) Next(ctx context.Context, items *[]mtypes.DomainKey) bool {
	if ri.err != nil {
		return false
	}

	ri.err = ri.fetch(ctx, ri.Paging.Next, ri.limit)
	if ri.err != nil {
		return false
	}

	cpy := make([]mtypes.DomainKey, len(ri.Items))
	copy(cpy, ri.Items)
	*items = cpy
	return len(ri.Items) != 0
}

// First retrieves the first page of items from the api. Returns false if there
// was an error. It also sets the iterator object to the first page.
// Use `.Err()` to retrieve the error.
func (ri *AllDomainsKeysIterator) First(ctx context.Context, items *[]mtypes.DomainKey) bool {
	if ri.err != nil {
		return false
	}
	ri.err = ri.fetch(ctx, ri.Paging.First, ri.limit)
	if ri.err != nil {
		return false
	}
	cpy := make([]mtypes.DomainKey, len(ri.Items))
	copy(cpy, ri.Items)
	*items = cpy
	return true
}

// Last retrieves the last page of items from the api.
// Calling Last() is invalid unless you first call First() or Next()
// Returns false if there was an error. It also sets the iterator object
// to the last page. Use `.Err()` to retrieve the error.
func (ri *AllDomainsKeysIterator) Last(ctx context.Context, items *[]mtypes.DomainKey) bool {
	if ri.err != nil {
		return false
	}

	if ri.TotalCount == -1 {
		return false
	}

	ri.err = ri.fetch(ctx, ri.Paging.Last, ri.limit)
	if ri.err != nil {
		return false
	}
	cpy := make([]mtypes.DomainKey, len(ri.Items))
	copy(cpy, ri.Items)
	*items = cpy
	return true
}

// Previous retrieves the previous page of items from the api. Returns false when there
// are no more pages to retrieve or if there was an error. Use `.Err()` to retrieve
// the error if any
func (ri *AllDomainsKeysIterator) Previous(ctx context.Context, items *[]mtypes.DomainKey) bool {
	if ri.err != nil {
		return false
	}

	if ri.TotalCount == -1 {
		return false
	}

	ri.err = ri.fetch(ctx, ri.Paging.Previous, ri.limit)
	if ri.err != nil {
		return false
	}
	cpy := make([]mtypes.DomainKey, len(ri.Items))
	copy(cpy, ri.Items)
	*items = cpy

	return len(ri.Items) != 0
}

func (ri *AllDomainsKeysIterator) fetch(ctx context.Context, pageUrl string, limit int) error {
	ri.Items = nil
	uri := ri.url
	if pageUrl != "" {
		uri = pageUrl
	}
	r := newHTTPRequest(uri)

	r.setBasicAuth(basicAuthUser, ri.mg.APIKey())
	r.setClient(ri.mg.HTTPClient())

	if limit != 0 {
		r.addParameter("limit", strconv.Itoa(limit))
	}

	return getResponseFromJSON(ctx, r, &ri.ListAllDomainsKeysResponse)
}

// CreateDomainKey creates a domain key for the given domain
func (mg *Client) CreateDomainKey(ctx context.Context, domain, dkimSelector string, opts *CreateDomainKeyOptions) (mtypes.DomainKey, error) {
	r := newHTTPRequest(generateApiUrl(mg, 1, dkimEndpoint))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	// TODO(vtopc): should be "multipart/form-data" (NewFormDataPayload) according to the docs:
	// https://documentation.mailgun.com/docs/inboxready/api-reference/optimize/mailgun/domain-keys/post-v1-dkim-keys
	payload := newUrlEncodedPayload()
	payload.addValue("signing_domain", domain)
	payload.addValue("selector", dkimSelector)

	if opts.Bits != 0 {
		payload.addValue("bits", strconv.Itoa(opts.Bits))
	}

	if opts.PEM != "" {
		payload.addValue("pem", opts.PEM)
	}

	var resp mtypes.DomainKey
	err := postResponseFromJSON(ctx, r, payload, &resp)
	return resp, err
}

// DeleteDomainKey deletes a domain key from the given domain
func (mg *Client) DeleteDomainKey(ctx context.Context, domain, dkimSelector string) error {
	r := newHTTPRequest(generateApiUrl(mg, 1, dkimEndpoint))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	r.addParameter("signing_domain", domain)
	r.addParameter("selector", dkimSelector)

	_, err := makeDeleteRequest(ctx, r)
	return err
}

// ActivateDomainKey deactivates a domain key for the given domain
func (mg *Client) ActivateDomainKey(ctx context.Context, domain, dkimSelector string) error {
	uri := generateActivateDomainKeyApiUrl(domainsEndpoint, domain, dkimSelector)

	r := newHTTPRequest(generateApiUrl(mg, 4, uri))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	// TODO(vtopc): why newUrlEncodedPayload()?
	_, err := makePutRequest(ctx, r, newUrlEncodedPayload())
	return err
}

// ListDomainKeys retrieves a set of domain keys from Mailgun.
// https://documentation.mailgun.com/docs/mailgun/api-reference/send/mailgun/domain-keys/get-v4-domains--authority-name--keys
//
// TODO(v6): domain can have at most 5 domain keys, so it makes no sense to have an iterator here,
//
//	Just return []mtypes.DomainKey.
func (mg *Client) ListDomainKeys(domain string) *DomainKeysIterator {
	uri := generateListDomainKeysApiUrl(domainsEndpoint, domain)

	return &DomainKeysIterator{
		ListDomainKeysResponse: mtypes.ListDomainKeysResponse{},
		mg:                     mg,
		uri:                    generateApiUrl(mg, 4, uri),
		domain:                 domain,
		isFirst:                true,
	}
}

// Err if an error occurred during iteration `Err()` will return non nil
func (iter *DomainKeysIterator) Err() error {
	return iter.err
}

// Next retrieves the next(or first) page of items from the API.
// Returns false when there are no more pages to retrieve or if there was an error.
// Use `.Err()` to retrieve the error.
// Domain can have at most 5 domain keys, so this will always return false on the first call.
func (iter *DomainKeysIterator) Next(ctx context.Context, items *[]mtypes.DomainKey) bool {
	if iter.err != nil {
		return false
	}

	var pageURI string
	if iter.isFirst {
		pageURI = iter.uri
	} else {
		pageURI = iter.Paging.Next
	}

	if pageURI == "" {
		return false
	}

	iter.err = iter.fetch(ctx, pageURI)
	if iter.err != nil {
		return false
	}

	cpy := make([]mtypes.DomainKey, len(iter.Items))
	copy(cpy, iter.Items)
	*items = cpy
	iter.isFirst = false

	if iter.Paging.Next == "" { // NOTE: always empty on API as of now.
		return false
	}

	return len(iter.Items) != 0
}

// First retrieves the first page of items from the API.
// Returns false if there was an error.
// Use `.Err()` to retrieve the error.
func (iter *DomainKeysIterator) First(ctx context.Context, items *[]mtypes.DomainKey) bool {
	if iter.err != nil {
		return false
	}

	uri := generateListDomainKeysApiUrl(domainsEndpoint, iter.domain)
	iter.err = iter.fetch(ctx, generateApiUrl(iter.mg, 4, uri))
	if iter.err != nil {
		return false
	}

	cpy := make([]mtypes.DomainKey, len(iter.Items))
	copy(cpy, iter.Items)
	*items = cpy
	iter.isFirst = false

	return true
}

// Last - not implemented on API. Use Next() instead.
func (iter *DomainKeysIterator) Last(_ context.Context, _ *[]mtypes.DomainKey) bool {
	iter.err = errors.New("not implemented on API; use Next() instead")

	return false
}

// Previous - not implemented on API. Use Next() instead.
func (iter *DomainKeysIterator) Previous(_ context.Context, _ *[]mtypes.DomainKey) bool {
	iter.err = errors.New("not implemented on API; use Next() instead")

	return false
}

func (iter *DomainKeysIterator) fetch(ctx context.Context, uri string) error {
	iter.Items = nil
	r := newHTTPRequest(uri)

	r.setBasicAuth(basicAuthUser, iter.mg.APIKey())
	r.setClient(iter.mg.HTTPClient())

	return getResponseFromJSON(ctx, r, &iter.ListDomainKeysResponse)
}

// DeactivateDomainKey deactivates a domain key for the given domain
func (mg *Client) DeactivateDomainKey(ctx context.Context, domain, dkimSelector string) error {
	uri := generateDeactivateDomainKeyApiUrl(domainsEndpoint, domain, dkimSelector)

	r := newHTTPRequest(generateApiUrl(mg, 4, uri))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	// TODO(vtopc): why newUrlEncodedPayload()?
	_, err := makePutRequest(ctx, r, newUrlEncodedPayload())
	return err
}

func (mg *Client) UpdateDomainDkimAuthority(ctx context.Context, domain string, self bool) (mtypes.UpdateDomainDkimAuthorityResponse, error) {
	r := newHTTPRequest(generateApiUrl(mg, 3, domainsEndpoint) + "/" + domain + "/dkim_authority")
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	// TODO(vtopc): should be "multipart/form-data" (NewFormDataPayload) according to the docs:
	// https://documentation.mailgun.com/docs/inboxready/api-reference/optimize/mailgun/domain-keys/put-v3-domains--name--dkim-authority
	payload := newUrlEncodedPayload()
	payload.addValue("self", boolToString(self))

	var resp mtypes.UpdateDomainDkimAuthorityResponse

	err := putResponseFromJSON(ctx, r, payload, &resp)

	return resp, err
}

// UpdateDomainDkimSelector updates the DKIM selector for a domain
func (mg *Client) UpdateDomainDkimSelector(ctx context.Context, domain, dkimSelector string) error {
	r := newHTTPRequest(generateApiUrl(mg, 3, domainsEndpoint) + "/" + domain + "/dkim_selector")
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	// TODO(vtopc): should be "multipart/form-data" (NewFormDataPayload) according to the docs:
	// https://documentation.mailgun.com/docs/inboxready/api-reference/optimize/mailgun/domain-keys/put-v3-domains--name--dkim-selector
	payload := newUrlEncodedPayload()
	payload.addValue("dkim_selector", dkimSelector)
	_, err := makePutRequest(ctx, r, payload)
	return err
}

// generateActivateDomainKeyApiUrl renders a URL fragment relevant for deactivating a domain key
func generateActivateDomainKeyApiUrl(endpoint, domain, dkimSelector string) string {
	return fmt.Sprintf("%s/%s/keys/%s/activate", endpoint, domain, dkimSelector)
}

// generateListDomainKeysApiUrl renders a URL fragment relevant for listing a domain's keys
func generateListDomainKeysApiUrl(endpoint, domain string) string {
	return fmt.Sprintf("%s/%s/keys", endpoint, domain)
}

// generateDeactivateDomainKeyApiUrl renders a URL fragment relevant for deactivating a domain key
func generateDeactivateDomainKeyApiUrl(endpoint, domain, dkimSelector string) string {
	return fmt.Sprintf("%s/%s/keys/%s/deactivate", endpoint, domain, dkimSelector)
}
