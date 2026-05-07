package mailgun

import (
	"context"
	"strconv"

	"github.com/mailgun/mailgun-go/v5/mtypes"
)

// AddTemplateVersion adds a template version to a template
func (mg *Client) AddTemplateVersion(ctx context.Context, domain, templateName string, version *mtypes.TemplateVersion) error {
	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, templatesEndpoint, domain) + "/" + templateName + "/versions")
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	payload := newUrlEncodedPayload()
	payload.addValue("template", version.Template)

	if version.Tag != "" {
		payload.addValue("tag", string(version.Tag))
	}
	if version.Engine != "" {
		payload.addValue("engine", string(version.Engine))
	}
	if version.Comment != "" {
		payload.addValue("comment", version.Comment)
	}
	if version.Active {
		payload.addValue("active", boolToString(version.Active))
	}

	var resp mtypes.TemplateResp
	if err := postResponseFromJSON(ctx, r, payload, &resp); err != nil {
		return err
	}
	*version = resp.Item.Version
	return nil
}

// GetTemplateVersion gets a specific version of a template
func (mg *Client) GetTemplateVersion(ctx context.Context, domain, templateName, tag string) (mtypes.TemplateVersion, error) {
	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, templatesEndpoint, domain) + "/" + templateName + "/versions/" + tag)
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	var resp mtypes.TemplateResp
	err := getResponseFromJSON(ctx, r, &resp)
	if err != nil {
		return mtypes.TemplateVersion{}, err
	}
	return resp.Item.Version, nil
}

// Update the comment and mark a version of a template active
func (mg *Client) UpdateTemplateVersion(ctx context.Context, domain, templateName string, version *mtypes.TemplateVersion) error {
	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, templatesEndpoint, domain) + "/" + templateName + "/versions/" + version.Tag)
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	p := newUrlEncodedPayload()

	if version.Comment != "" {
		p.addValue("comment", version.Comment)
	}
	if version.Active {
		p.addValue("active", boolToString(version.Active))
	}
	if version.Template != "" {
		p.addValue("template", version.Template)
	}

	var resp mtypes.TemplateResp
	err := putResponseFromJSON(ctx, r, p, &resp)
	if err != nil {
		return err
	}
	*version = resp.Item.Version
	return nil
}

// Delete a specific version of a template
func (mg *Client) DeleteTemplateVersion(ctx context.Context, domain, templateName, tag string) error {
	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, templatesEndpoint, domain) + "/" + templateName + "/versions/" + tag)
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	_, err := makeDeleteRequest(ctx, r)
	return err
}

type TemplateVersionsIterator struct {
	mtypes.TemplateVersionListResp
	mg  Mailgun
	err error
}

// ListTemplateVersions lists all the versions of a specific template
func (mg *Client) ListTemplateVersions(domain, templateName string, opts *ListOptions) *TemplateVersionsIterator {
	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, templatesEndpoint, domain) + "/" + templateName + "/versions")
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	if opts != nil {
		if opts.Limit != 0 {
			r.addParameter("limit", strconv.Itoa(opts.Limit))
		}
	}
	url, err := r.generateUrlWithParameters()
	return &TemplateVersionsIterator{
		mg:                      mg,
		TemplateVersionListResp: mtypes.TemplateVersionListResp{Paging: mtypes.Paging{Next: url, First: url}},
		err:                     err,
	}
}

// If an error occurred during iteration `Err()` will return non nil
func (li *TemplateVersionsIterator) Err() error {
	return li.err
}

// Next retrieves the next page of items from the api. Returns false when there
// no more pages to retrieve or if there was an error. Use `.Err()` to retrieve
// the error
func (li *TemplateVersionsIterator) Next(ctx context.Context, items *[]mtypes.TemplateVersion) bool {
	if li.err != nil {
		return false
	}
	li.err = li.fetch(ctx, li.Paging.Next)
	if li.err != nil {
		return false
	}
	cpy := make([]mtypes.TemplateVersion, len(li.Template.Versions))
	copy(cpy, li.Template.Versions)
	*items = cpy

	return len(li.Template.Versions) != 0
}

// First retrieves the first page of items from the api. Returns false if there
// was an error. It also sets the iterator object to the first page.
// Use `.Err()` to retrieve the error.
func (li *TemplateVersionsIterator) First(ctx context.Context, items *[]mtypes.TemplateVersion) bool {
	if li.err != nil {
		return false
	}
	li.err = li.fetch(ctx, li.Paging.First)
	if li.err != nil {
		return false
	}
	cpy := make([]mtypes.TemplateVersion, len(li.Template.Versions))
	copy(cpy, li.Template.Versions)
	*items = cpy
	return true
}

// Last retrieves the last page of items from the api.
// Calling Last() is invalid unless you first call First() or Next()
// Returns false if there was an error. It also sets the iterator object
// to the last page. Use `.Err()` to retrieve the error.
func (li *TemplateVersionsIterator) Last(ctx context.Context, items *[]mtypes.TemplateVersion) bool {
	if li.err != nil {
		return false
	}
	li.err = li.fetch(ctx, li.Paging.Last)
	if li.err != nil {
		return false
	}
	cpy := make([]mtypes.TemplateVersion, len(li.Template.Versions))
	copy(cpy, li.Template.Versions)
	*items = cpy
	return true
}

// Previous retrieves the previous page of items from the api. Returns false when there
// no more pages to retrieve or if there was an error. Use `.Err()` to retrieve
// the error if any
func (li *TemplateVersionsIterator) Previous(ctx context.Context, items *[]mtypes.TemplateVersion) bool {
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
	cpy := make([]mtypes.TemplateVersion, len(li.Template.Versions))
	copy(cpy, li.Template.Versions)
	*items = cpy

	return len(li.Template.Versions) != 0
}

func (li *TemplateVersionsIterator) fetch(ctx context.Context, url string) error {
	li.Template.Versions = nil
	r := newHTTPRequest(url)
	r.setClient(li.mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, li.mg.APIKey())

	return getResponseFromJSON(ctx, r, &li.TemplateVersionListResp)
}
