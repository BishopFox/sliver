package mailgun

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mailgun/mailgun-go/v5/mtypes"
)

type ListSubaccountsOptions struct {
	Limit     int
	Skip      int
	SortArray string
	Enabled   bool
}

type SubaccountsIterator struct {
	mtypes.ListSubaccountsResponse

	mg        Mailgun
	limit     int
	offset    int
	skip      int
	sortArray string
	enabled   bool
	url       string
	err       error
}

// ListSubaccounts retrieves a set of subaccount linked to the primary Mailgun account.
func (mg *Client) ListSubaccounts(opts *ListSubaccountsOptions) *SubaccountsIterator {
	r := newHTTPRequest(generateSubaccountsApiUrl(mg))
	r.setClient(mg.client)
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	var limit, skip int
	var sortArray string
	var enabled bool
	if opts != nil {
		limit = opts.Limit
		skip = opts.Skip
		sortArray = opts.SortArray
		enabled = opts.Enabled
	}
	if limit == 0 {
		limit = 10
	}

	return &SubaccountsIterator{
		mg:                      mg,
		url:                     generateSubaccountsApiUrl(mg),
		ListSubaccountsResponse: mtypes.ListSubaccountsResponse{Total: -1},
		limit:                   limit,
		skip:                    skip,
		sortArray:               sortArray,
		enabled:                 enabled,
	}
}

// If an error occurred during iteration `Err()` will return non nil
func (ri *SubaccountsIterator) Err() error {
	return ri.err
}

// Offset returns the current offset of the iterator
func (ri *SubaccountsIterator) Offset() int {
	return ri.offset
}

// Next retrieves the next page of items from the api. Returns false when there
// no more pages to retrieve or if there was an error. Use `.Err()` to retrieve
// the error
func (ri *SubaccountsIterator) Next(ctx context.Context, items *[]mtypes.Subaccount) bool {
	if ri.err != nil {
		return false
	}

	ri.err = ri.fetch(ctx, ri.offset, ri.limit)
	if ri.err != nil {
		return false
	}

	cpy := make([]mtypes.Subaccount, len(ri.Items))
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
func (ri *SubaccountsIterator) First(ctx context.Context, items *[]mtypes.Subaccount) bool {
	if ri.err != nil {
		return false
	}
	ri.err = ri.fetch(ctx, 0, ri.limit)
	if ri.err != nil {
		return false
	}
	cpy := make([]mtypes.Subaccount, len(ri.Items))
	copy(cpy, ri.Items)
	*items = cpy
	ri.offset = len(ri.Items)
	return true
}

// Last retrieves the last page of items from the api.
// Calling Last() is invalid unless you first call First() or Next()
// Returns false if there was an error. It also sets the iterator object
// to the last page. Use `.Err()` to retrieve the error.
func (ri *SubaccountsIterator) Last(ctx context.Context, items *[]mtypes.Subaccount) bool {
	if ri.err != nil {
		return false
	}

	if ri.Total == -1 {
		return false
	}

	ri.offset = ri.Total - ri.limit
	if ri.offset < 0 {
		ri.offset = 0
	}

	ri.err = ri.fetch(ctx, ri.offset, ri.limit)
	if ri.err != nil {
		return false
	}
	cpy := make([]mtypes.Subaccount, len(ri.Items))
	copy(cpy, ri.Items)
	*items = cpy
	return true
}

// Previous retrieves the previous page of items from the api. Returns false when there
// no more pages to retrieve or if there was an error. Use `.Err()` to retrieve
// the error if any
func (ri *SubaccountsIterator) Previous(ctx context.Context, items *[]mtypes.Subaccount) bool {
	if ri.err != nil {
		return false
	}

	if ri.Total == -1 {
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
	cpy := make([]mtypes.Subaccount, len(ri.Items))
	copy(cpy, ri.Items)
	*items = cpy

	return len(ri.Items) != 0
}

func (ri *SubaccountsIterator) fetch(ctx context.Context, skip, limit int) error {
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

	return getResponseFromJSON(ctx, r, &ri.ListSubaccountsResponse)
}

// CreateSubaccount instructs Mailgun to create a new account (Subaccount) that is linked to the primary account.
// Subaccounts are child accounts that share the same plan and usage allocations as the primary, but have their own
// assets (sending domains, unique users, API key, SMTP credentials, settings, statistics and site login).
// All you need is the name of the subaccount.
func (mg *Client) CreateSubaccount(ctx context.Context, subaccountName string) (mtypes.SubaccountResponse, error) {
	r := newHTTPRequest(generateSubaccountsApiUrl(mg))
	r.setClient(mg.client)
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	payload := newUrlEncodedPayload()
	payload.addValue("name", subaccountName)
	resp := mtypes.SubaccountResponse{}
	err := postResponseFromJSON(ctx, r, payload, &resp)
	return resp, err
}

// GetSubaccount retrieves detailed information about subaccount using subaccountID.
func (mg *Client) GetSubaccount(ctx context.Context, subaccountID string) (mtypes.SubaccountResponse, error) {
	r := newHTTPRequest(generateSubaccountsApiUrl(mg) + "/" + subaccountID)
	r.setClient(mg.client)
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	var resp mtypes.SubaccountResponse
	err := getResponseFromJSON(ctx, r, &resp)
	return resp, err
}

// EnableSubaccount instructs Mailgun to enable subaccount.
func (mg *Client) EnableSubaccount(ctx context.Context, subaccountId string) (mtypes.SubaccountResponse, error) {
	r := newHTTPRequest(generateSubaccountsApiUrl(mg) + "/" + subaccountId + "/" + "enable")
	r.setClient(mg.client)
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	resp := mtypes.SubaccountResponse{}
	err := postResponseFromJSON(ctx, r, nil, &resp)
	return resp, err
}

// DisableSubaccount instructs Mailgun to disable subaccount.
func (mg *Client) DisableSubaccount(ctx context.Context, subaccountId string) (mtypes.SubaccountResponse, error) {
	r := newHTTPRequest(generateSubaccountsApiUrl(mg) + "/" + subaccountId + "/" + "disable")
	r.setClient(mg.client)
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	resp := mtypes.SubaccountResponse{}
	err := postResponseFromJSON(ctx, r, nil, &resp)
	return resp, err
}

func generateSubaccountsApiUrl(m Mailgun) string {
	return fmt.Sprintf("%s/v5/%s/%s", m.APIBase(), accountsEndpoint, subaccountsEndpoint)
}
