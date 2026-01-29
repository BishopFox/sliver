package mailgun

// TODO(vtopc): deprecate tags API in favor of new /v1/analytics/tags

import (
	"context"
	"net/url"
	"strconv"

	"github.com/mailgun/mailgun-go/v5/mtypes"
)

type ListTagOptions struct {
	// Restrict the page size to this limit
	Limit int
	// Return only the tags starting with the given prefix
	Prefix string
}

// DeleteTag removes all counters for a particular tag, including the tag itself.
func (mg *Client) DeleteTag(ctx context.Context, domain, tag string) error {
	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, tagsEndpoint, domain) + "/" + tag)
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	_, err := makeDeleteRequest(ctx, r)
	return err
}

// GetTag retrieves metadata about the tag from the api
func (mg *Client) GetTag(ctx context.Context, domain, tag string) (mtypes.Tag, error) {
	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, tagsEndpoint, domain) + "/" + tag)
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	var tagItem mtypes.Tag
	err := getResponseFromJSON(ctx, r, &tagItem)
	return tagItem, err
}

// ListTags returns a cursor used to iterate through a list of tags
//
//	it := mg.ListTags(nil)
//	var page []mailgun.Tag
//	for it.Next(&page) {
//		for _, tag := range(page) {
//			// Do stuff with tags
//		}
//	}
//	if it.Err() != nil {
//		log.Fatal(it.Err())
//	}
func (mg *Client) ListTags(domain string, opts *ListTagOptions) *TagIterator {
	req := newHTTPRequest(generateApiV3UrlWithDomain(mg, tagsEndpoint, domain))
	if opts != nil {
		if opts.Limit != 0 {
			req.addParameter("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Prefix != "" {
			req.addParameter("prefix", opts.Prefix)
		}
	}

	uri, err := req.generateUrlWithParameters()
	return &TagIterator{
		TagsResponse: mtypes.TagsResponse{Paging: mtypes.Paging{Next: uri, First: uri}},
		err:          err,
		mg:           mg,
	}
}

type TagIterator struct {
	mtypes.TagsResponse
	mg  Mailgun
	err error
}

// Next returns the next page in the list of tags
func (ti *TagIterator) Next(ctx context.Context, items *[]mtypes.Tag) bool {
	if ti.err != nil {
		return false
	}

	if !canFetchPage(ti.Paging.Next) {
		return false
	}

	ti.err = ti.fetch(ctx, ti.Paging.Next)
	if ti.err != nil {
		return false
	}
	*items = ti.Items

	return len(ti.Items) != 0
}

// Previous returns the previous page in the list of tags
func (ti *TagIterator) Previous(ctx context.Context, items *[]mtypes.Tag) bool {
	if ti.err != nil {
		return false
	}

	if ti.Paging.Previous == "" {
		return false
	}

	if !canFetchPage(ti.Paging.Previous) {
		return false
	}

	ti.err = ti.fetch(ctx, ti.Paging.Previous)
	if ti.err != nil {
		return false
	}
	*items = ti.Items

	return len(ti.Items) != 0
}

// First returns the first page in the list of tags
func (ti *TagIterator) First(ctx context.Context, items *[]mtypes.Tag) bool {
	if ti.err != nil {
		return false
	}
	ti.err = ti.fetch(ctx, ti.Paging.First)
	if ti.err != nil {
		return false
	}
	*items = ti.Items
	return true
}

// Last returns the last page in the list of tags
func (ti *TagIterator) Last(ctx context.Context, items *[]mtypes.Tag) bool {
	if ti.err != nil {
		return false
	}
	ti.err = ti.fetch(ctx, ti.Paging.Last)
	if ti.err != nil {
		return false
	}
	*items = ti.Items
	return true
}

// Err returns any error if one occurred
func (ti *TagIterator) Err() error {
	return ti.err
}

func (ti *TagIterator) fetch(ctx context.Context, uri string) error {
	ti.Items = nil
	req := newHTTPRequest(uri)
	req.setClient(ti.mg.HTTPClient())
	req.setBasicAuth(basicAuthUser, ti.mg.APIKey())
	return getResponseFromJSON(ctx, req, &ti.TagsResponse)
}

func canFetchPage(slug string) bool {
	parts, err := url.Parse(slug)
	if err != nil {
		return false
	}
	params, err := url.ParseQuery(parts.RawQuery)
	if err != nil {
		return false
	}
	value, ok := params["tag"]
	// If tags doesn't exist, it's our first time fetching pages
	if !ok {
		return true
	}
	// If tags has no value, there are no more pages to fetch
	return len(value) == 0
}
