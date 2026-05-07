package mailgun

import (
	"context"
	"slices"
	"strconv"

	"github.com/mailgun/mailgun-go/v5/mtypes"
)

// ListIPs returns a list of IPs assigned to your account, including their warmup and assignable to pools status if applicable.
func (mg *Client) ListIPs(ctx context.Context, dedicated, enabled bool) ([]mtypes.IPAddress, error) {
	r := newHTTPRequest(generateApiUrl(mg, 3, ipsEndpoint))
	r.setClient(mg.HTTPClient())
	if dedicated {
		r.addParameter("dedicated", "true")
	}
	if enabled {
		r.addParameter("enabled", "true")
	}
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	var resp mtypes.IPAddressListResponse
	if err := getResponseFromJSON(ctx, r, &resp); err != nil {
		return nil, err
	}
	var result []mtypes.IPAddress
	for _, ip := range resp.Items {
		assignableToPools := slices.Index(resp.AssignableToPools, ip) != -1
		detailsIndex := slices.IndexFunc(resp.Details, func(d mtypes.IPAddressListResponseDetail) bool { return d.IP == ip })
		isOnWarmup := resp.Details[detailsIndex].IsOnWarmup
		ipState := mtypes.IPAddress{IP: ip, AssignableToPools: assignableToPools, IsOnWarmup: isOnWarmup}
		result = append(result, ipState)
	}
	return result, nil
}

// GetIP returns information about the specified IP
func (mg *Client) GetIP(ctx context.Context, ip string) (mtypes.IPAddress, error) {
	r := newHTTPRequest(generateApiUrl(mg, 3, ipsEndpoint) + "/" + ip)
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	var resp mtypes.IPAddress
	err := getResponseFromJSON(ctx, r, &resp)
	return resp, err
}

// ListDomainIPs returns a list of IPs currently assigned to the specified domain.
func (mg *Client) ListDomainIPs(ctx context.Context, domain string) ([]mtypes.IPAddress, error) {
	r := newHTTPRequest(generateApiUrl(mg, 3, domainsEndpoint) + "/" + domain + "/ips")
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	var resp mtypes.IPAddressListResponse
	if err := getResponseFromJSON(ctx, r, &resp); err != nil {
		return nil, err
	}
	var result []mtypes.IPAddress
	for _, ip := range resp.Items {
		result = append(result, mtypes.IPAddress{IP: ip})
	}
	return result, nil
}

// AddDomainIP assign a dedicated IP to the domain specified.
func (mg *Client) AddDomainIP(ctx context.Context, domain, ip string) error {
	r := newHTTPRequest(generateApiUrl(mg, 3, domainsEndpoint) + "/" + domain + "/ips")
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	payload := newUrlEncodedPayload()
	payload.addValue("ip", ip)
	_, err := makePostRequest(ctx, r, payload)
	return err
}

// DeleteDomainIP unassign an IP from the domain specified.
func (mg *Client) DeleteDomainIP(ctx context.Context, domain, ip string) error {
	r := newHTTPRequest(generateApiUrl(mg, 3, domainsEndpoint) + "/" + domain + "/ips/" + ip)
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	_, err := makeDeleteRequest(ctx, r)
	return err
}

type IPDomainsIterator struct {
	mtypes.ListIPDomainsResponse

	limit  int
	mg     Mailgun
	offset int
	url    string
	err    error
}

// Err if an error occurred during iteration `Err()` will return non nil
func (ri *IPDomainsIterator) Err() error {
	return ri.err
}

// Offset returns the current offset of the iterator
func (ri *IPDomainsIterator) Offset() int {
	return ri.offset
}

// Next retrieves the next page of items from the api. Returns false when there
// no more pages to retrieve or if there was an error. Use `.Err()` to retrieve
// the error
func (ri *IPDomainsIterator) Next(ctx context.Context, items *[]mtypes.DomainIPs) bool {
	if ri.err != nil {
		return false
	}

	ri.err = ri.fetch(ctx, ri.offset, ri.limit)
	if ri.err != nil {
		return false
	}

	cpy := make([]mtypes.DomainIPs, len(ri.Items))
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
func (ri *IPDomainsIterator) First(ctx context.Context, items *[]mtypes.DomainIPs) bool {
	if ri.err != nil {
		return false
	}
	ri.err = ri.fetch(ctx, 0, ri.limit)
	if ri.err != nil {
		return false
	}
	cpy := make([]mtypes.DomainIPs, len(ri.Items))
	copy(cpy, ri.Items)
	*items = cpy
	ri.offset = len(ri.Items)
	return true
}

// Last retrieves the last page of items from the api.
// Calling Last() is invalid unless you first call First() or Next()
// Returns false if there was an error. It also sets the iterator object
// to the last page. Use `.Err()` to retrieve the error.
func (ri *IPDomainsIterator) Last(ctx context.Context, items *[]mtypes.DomainIPs) bool {
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
	cpy := make([]mtypes.DomainIPs, len(ri.Items))
	copy(cpy, ri.Items)
	*items = cpy
	return true
}

// Previous retrieves the previous page of items from the api. Returns false when there
// no more pages to retrieve or if there was an error. Use `.Err()` to retrieve
// the error if any
func (ri *IPDomainsIterator) Previous(ctx context.Context, items *[]mtypes.DomainIPs) bool {
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
	cpy := make([]mtypes.DomainIPs, len(ri.Items))
	copy(cpy, ri.Items)
	*items = cpy

	return len(ri.Items) != 0
}

func (ri *IPDomainsIterator) fetch(ctx context.Context, skip, limit int) error {
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

	return getResponseFromJSON(ctx, r, &ri.ListIPDomainsResponse)
}

type ListIPDomainOptions struct {
	Limit int
}

// ListIPDomains retrieves a list of domains for the specified IP address.
func (mg *Client) ListIPDomains(ip string, opts *ListIPDomainOptions) *IPDomainsIterator {
	var limit int
	if opts != nil {
		limit = opts.Limit
	}

	if limit == 0 {
		limit = 100
	}
	return &IPDomainsIterator{
		mg:                    mg,
		url:                   generateApiUrl(mg, 3, ipsEndpoint) + "/" + ip + "/domains",
		ListIPDomainsResponse: mtypes.ListIPDomainsResponse{TotalCount: -1},
		limit:                 limit,
	}
}
