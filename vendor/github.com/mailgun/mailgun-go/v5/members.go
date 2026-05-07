package mailgun

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/mailgun/mailgun-go/v5/mtypes"
)

type MemberListIterator struct {
	mtypes.MemberListResponse
	mg  Mailgun
	err error
}

func (mg *Client) ListMembers(listAddress string, opts *ListOptions) *MemberListIterator {
	r := newHTTPRequest(generateMemberApiUrl(mg, listsEndpoint, url.QueryEscape(listAddress)) + "/pages")
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	if opts != nil {
		if opts.Limit != 0 {
			r.addParameter("limit", strconv.Itoa(opts.Limit))
		}
	}
	uri, err := r.generateUrlWithParameters()
	return &MemberListIterator{
		mg: mg,
		// TODO(vtopc): why is Next and First both set to the same URL?
		MemberListResponse: mtypes.MemberListResponse{Paging: mtypes.Paging{Next: uri, First: uri}},
		err:                err,
	}
}

// Err if an error occurred during iteration `Err()` will return non nil
func (li *MemberListIterator) Err() error {
	return li.err
}

// Next retrieves the next page of items from the api. Returns false when there are
// no more pages to retrieve or if there was an error. Use `.Err()` to retrieve
// the error
func (li *MemberListIterator) Next(ctx context.Context, items *[]mtypes.Member) bool {
	if li.err != nil {
		return false
	}
	li.err = li.fetch(ctx, li.Paging.Next)
	if li.err != nil {
		return false
	}
	*items = li.Lists

	return len(li.Lists) != 0
}

// First retrieves the first page of items from the api. Returns false if there
// was an error. It also sets the iterator object to the first page.
// Use `.Err()` to retrieve the error.
func (li *MemberListIterator) First(ctx context.Context, items *[]mtypes.Member) bool {
	if li.err != nil {
		return false
	}
	li.err = li.fetch(ctx, li.Paging.First)
	if li.err != nil {
		return false
	}
	*items = li.Lists
	return true
}

// Last retrieves the last page of items from the api.
// Calling Last() is invalid unless you first call First() or Next()
// Returns false if there was an error. It also sets the iterator object
// to the last page. Use `.Err()` to retrieve the error.
func (li *MemberListIterator) Last(ctx context.Context, items *[]mtypes.Member) bool {
	if li.err != nil {
		return false
	}
	li.err = li.fetch(ctx, li.Paging.Last)
	if li.err != nil {
		return false
	}
	*items = li.Lists
	return true
}

// Previous retrieves the previous page of items from the api. Returns false when there
// no more pages to retrieve or if there was an error. Use `.Err()` to retrieve
// the error if any
func (li *MemberListIterator) Previous(ctx context.Context, items *[]mtypes.Member) bool {
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
	*items = li.Lists

	return len(li.Lists) != 0
}

func (li *MemberListIterator) fetch(ctx context.Context, uri string) error {
	li.Lists = nil
	r := newHTTPRequest(uri)
	r.setClient(li.mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, li.mg.APIKey())

	return getResponseFromJSON(ctx, r, &li.MemberListResponse)
}

// GetMember returns a complete Member structure for a member of a mailing list,
// given only their subscription e-mail address.
func (mg *Client) GetMember(ctx context.Context, memberAddress, listAddress string) (mtypes.Member, error) {
	uri := generateMemberApiUrl(mg, listsEndpoint, url.QueryEscape(listAddress)) + "/" + url.QueryEscape(memberAddress)
	r := newHTTPRequest(uri)
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	response, err := makeGetRequest(ctx, r)
	if err != nil {
		return mtypes.Member{}, err
	}

	var resp mtypes.MemberResponse
	err = response.parseFromJSON(&resp)
	return resp.Member, err
}

// CreateMember registers a new member of the indicated mailing list.
// If merge is set to true, then the registration may update an existing Member's settings.
// Otherwise, an error will occur if you attempt to add a member with a duplicate e-mail address.
func (mg *Client) CreateMember(ctx context.Context, merge bool, listAddress string, member mtypes.Member) error {
	vs, err := json.Marshal(member.Vars)
	if err != nil {
		return err
	}

	r := newHTTPRequest(generateMemberApiUrl(mg, listsEndpoint, url.QueryEscape(listAddress)))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	p := NewFormDataPayload()
	p.addValue("upsert", yesNo(merge))
	p.addValue("address", member.Address)
	p.addValue("name", member.Name)
	p.addValue("vars", string(vs))
	if member.Subscribed != nil {
		p.addValue("subscribed", yesNo(*member.Subscribed))
	}
	_, err = makePostRequest(ctx, r, p)
	return err
}

// UpdateMember lets you change certain details about the indicated mailing list member.
// Address, Name, Vars, and Subscribed fields may be changed.
func (mg *Client) UpdateMember(ctx context.Context, memberAddress, listAddress string, member mtypes.Member) (mtypes.Member, error) {
	r := newHTTPRequest(generateMemberApiUrl(mg, listsEndpoint, url.QueryEscape(listAddress)) + "/" + url.QueryEscape(memberAddress))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	p := NewFormDataPayload()
	if member.Address != "" {
		p.addValue("address", member.Address)
	}
	if member.Name != "" {
		p.addValue("name", member.Name)
	}
	if member.Vars != nil {
		vs, err := json.Marshal(member.Vars)
		if err != nil {
			return mtypes.Member{}, err
		}
		p.addValue("vars", string(vs))
	}
	if member.Subscribed != nil {
		p.addValue("subscribed", yesNo(*member.Subscribed))
	}
	response, err := makePutRequest(ctx, r, p)
	if err != nil {
		return mtypes.Member{}, err
	}
	var envelope struct {
		Member mtypes.Member `json:"member"`
	}
	err = response.parseFromJSON(&envelope)
	return envelope.Member, err
}

// DeleteMember removes the member from the list.
func (mg *Client) DeleteMember(ctx context.Context, memberAddress, listAddress string) error {
	r := newHTTPRequest(generateMemberApiUrl(mg, listsEndpoint, url.QueryEscape(listAddress)) + "/" + url.QueryEscape(memberAddress))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	_, err := makeDeleteRequest(ctx, r)
	return err
}

// CreateMemberList registers multiple Members and non-Member members to a single mailing list
// in a single round-trip.
// u indicates if the existing members should be updated or duplicates should be updated.
// Use All to elect not to provide a default.
// The newMembers list can take one of two JSON-encodable forms: a slice of strings, or
// a slice of Member structures.
// If a simple slice of strings is passed, each string refers to the member's e-mail address.
// Otherwise, each Member needs to have at least the Address field filled out.
// Other fields are optional, but may be set according to your needs.
func (mg *Client) CreateMemberList(ctx context.Context, u *bool, listAddress string, newMembers []any) error {
	r := newHTTPRequest(generateMemberApiUrl(mg, listsEndpoint, url.QueryEscape(listAddress)) + ".json")
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	p := NewFormDataPayload()
	if u != nil {
		p.addValue("upsert", yesNo(*u))
	}
	bs, err := json.Marshal(newMembers)
	if err != nil {
		return err
	}
	p.addValue("members", string(bs))
	_, err = makePostRequest(ctx, r, p)
	return err
}
