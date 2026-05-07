package mailgun

import (
	"context"
	"net/url"
	"strconv"

	"github.com/mailgun/mailgun-go/v5/mtypes"
)

type ListsIterator struct {
	mtypes.ListMailingListsResponse
	mg  Mailgun
	err error
}

// ListMailingLists returns the specified set of mailing lists administered by your account.
func (mg *Client) ListMailingLists(opts *ListOptions) *ListsIterator {
	r := newHTTPRequest(generateApiUrl(mg, 3, listsEndpoint) + "/pages")
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	if opts != nil {
		if opts.Limit != 0 {
			r.addParameter("limit", strconv.Itoa(opts.Limit))
		}
	}
	uri, err := r.generateUrlWithParameters()
	return &ListsIterator{
		mg: mg,
		// TODO(vtopc): why is Next and First both set to the same URL?
		ListMailingListsResponse: mtypes.ListMailingListsResponse{Paging: mtypes.Paging{Next: uri, First: uri}},
		err:                      err,
	}
}

// Err if an error occurred during iteration `Err()` will return non nil
func (li *ListsIterator) Err() error {
	return li.err
}

// Next retrieves the next page of items from the api. Returns false when there
// no more pages to retrieve or if there was an error. Use `.Err()` to retrieve
// the error
func (li *ListsIterator) Next(ctx context.Context, items *[]mtypes.MailingList) bool {
	if li.err != nil {
		return false
	}
	li.err = li.fetch(ctx, li.Paging.Next)
	if li.err != nil {
		return false
	}
	cpy := make([]mtypes.MailingList, len(li.Items))
	copy(cpy, li.Items)
	*items = cpy

	return len(li.Items) != 0
}

// First retrieves the first page of items from the api. Returns false if there
// was an error. It also sets the iterator object to the first page.
// Use `.Err()` to retrieve the error.
func (li *ListsIterator) First(ctx context.Context, items *[]mtypes.MailingList) bool {
	if li.err != nil {
		return false
	}
	li.err = li.fetch(ctx, li.Paging.First)
	if li.err != nil {
		return false
	}
	cpy := make([]mtypes.MailingList, len(li.Items))
	copy(cpy, li.Items)
	*items = cpy
	return true
}

// Last retrieves the last page of items from the api.
// Calling Last() is invalid unless you first call First() or Next()
// Returns false if there was an error. It also sets the iterator object
// to the last page. Use `.Err()` to retrieve the error.
func (li *ListsIterator) Last(ctx context.Context, items *[]mtypes.MailingList) bool {
	if li.err != nil {
		return false
	}
	li.err = li.fetch(ctx, li.Paging.Last)
	if li.err != nil {
		return false
	}
	cpy := make([]mtypes.MailingList, len(li.Items))
	copy(cpy, li.Items)
	*items = cpy
	return true
}

// Previous retrieves the previous page of items from the api. Returns false when there
// no more pages to retrieve or if there was an error. Use `.Err()` to retrieve
// the error if any
func (li *ListsIterator) Previous(ctx context.Context, items *[]mtypes.MailingList) bool {
	if li.err != nil {
		return false
	}
	if li.Paging.Previous == "" {
		return false
	}
	li.err = li.fetch(ctx, li.Paging.Previous)
	if li.err != nil {
		return false
	}
	cpy := make([]mtypes.MailingList, len(li.Items))
	copy(cpy, li.Items)
	*items = cpy

	return len(li.Items) != 0
}

func (li *ListsIterator) fetch(ctx context.Context, uri string) error {
	li.Items = nil
	r := newHTTPRequest(uri)
	r.setClient(li.mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, li.mg.APIKey())

	return getResponseFromJSON(ctx, r, &li.ListMailingListsResponse)
}

// CreateMailingList creates a new mailing list under your Mailgun account.
// You need specify only the Address and Name members of the prototype;
// Description, AccessLevel and ReplyPreference are optional.
// If unspecified, the Description remains blank,
// while AccessLevel defaults to Everyone
// and ReplyPreference defaults to List.
func (mg *Client) CreateMailingList(ctx context.Context, list mtypes.MailingList) (mtypes.MailingList, error) {
	r := newHTTPRequest(generateApiUrl(mg, 3, listsEndpoint))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	p := newUrlEncodedPayload()
	if list.Address != "" {
		p.addValue("address", list.Address)
	}
	if list.Name != "" {
		p.addValue("name", list.Name)
	}
	if list.Description != "" {
		p.addValue("description", list.Description)
	}
	if list.AccessLevel != "" {
		p.addValue("access_level", string(list.AccessLevel))
	}
	if list.ReplyPreference != "" {
		p.addValue("reply_preference", string(list.ReplyPreference))
	}
	response, err := makePostRequest(ctx, r, p)
	if err != nil {
		return mtypes.MailingList{}, err
	}
	var l mtypes.MailingList
	err = response.parseFromJSON(&l)
	return l, err
}

// DeleteMailingList removes all current members of the list, then removes the list itself.
// Attempts to send e-mail to the list will fail subsequent to this call.
func (mg *Client) DeleteMailingList(ctx context.Context, address string) error {
	r := newHTTPRequest(generateApiUrl(mg, 3, listsEndpoint) + "/" + url.QueryEscape(address))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	_, err := makeDeleteRequest(ctx, r)
	return err
}

// GetMailingList allows your application to recover the complete List structure
// representing a mailing list, so long as you have its e-mail address.
// TODO(v6): rename to GetMailingListByAddress to be more explicit.
func (mg *Client) GetMailingList(ctx context.Context, address string) (mtypes.MailingList, error) {
	r := newHTTPRequest(generateApiUrl(mg, 3, listsEndpoint) + "/" + url.QueryEscape(address))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	response, err := makeGetRequest(ctx, r)
	if err != nil {
		return mtypes.MailingList{}, err
	}

	var resp mtypes.GetMailingListResponse
	err = response.parseFromJSON(&resp)
	return resp.MailingList, err
}

// UpdateMailingList allows you to change various attributes of a list.
// Address, Name, Description, AccessLevel and ReplyPreference are all optional;
// only those fields which are set in the prototype will change.
//
// Be careful!  If changing the address of a mailing list,
// e-mail sent to the old address will not succeed.
// Make sure you account for the change accordingly.
func (mg *Client) UpdateMailingList(ctx context.Context, address string, list mtypes.MailingList) (mtypes.MailingList, error) {
	r := newHTTPRequest(generateApiUrl(mg, 3, listsEndpoint) + "/" + url.QueryEscape(address))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	p := newUrlEncodedPayload()
	if list.Address != "" {
		p.addValue("address", list.Address)
	}
	if list.Name != "" {
		p.addValue("name", list.Name)
	}
	if list.Description != "" {
		p.addValue("description", list.Description)
	}
	if list.AccessLevel != "" {
		p.addValue("access_level", string(list.AccessLevel))
	}
	if list.ReplyPreference != "" {
		p.addValue("reply_preference", string(list.ReplyPreference))
	}
	var l mtypes.MailingList
	response, err := makePutRequest(ctx, r, p)
	if err != nil {
		return l, err
	}
	err = response.parseFromJSON(&l)
	return l, err
}
