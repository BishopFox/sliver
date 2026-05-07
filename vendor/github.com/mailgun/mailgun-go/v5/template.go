package mailgun

import (
	"context"
	"errors"
	"strconv"

	"github.com/mailgun/mailgun-go/v5/mtypes"
)

// Create a new template which can be used to attach template versions to
func (mg *Client) CreateTemplate(ctx context.Context, domain string, template *mtypes.Template) error {
	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, templatesEndpoint, domain))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	payload := newUrlEncodedPayload()

	if template.Name != "" {
		payload.addValue("name", template.Name)
	}
	if template.Description != "" {
		payload.addValue("description", template.Description)
	}

	if template.Version.Engine != "" {
		payload.addValue("engine", string(template.Version.Engine))
	}
	if template.Version.Template != "" {
		payload.addValue("template", template.Version.Template)
	}
	if template.Version.Comment != "" {
		payload.addValue("comment", template.Version.Comment)
	}
	if template.Version.Tag != "" {
		payload.addValue("tag", template.Version.Tag)
	}

	var resp mtypes.TemplateResp
	if err := postResponseFromJSON(ctx, r, payload, &resp); err != nil {
		return err
	}
	*template = resp.Item
	return nil
}

// GetTemplate gets a template given the template name
func (mg *Client) GetTemplate(ctx context.Context, domain, name string) (mtypes.Template, error) {
	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, templatesEndpoint, domain) + "/" + name)
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	r.addParameter("active", "yes")

	var resp mtypes.TemplateResp
	err := getResponseFromJSON(ctx, r, &resp)
	if err != nil {
		return mtypes.Template{}, err
	}
	return resp.Item, nil
}

// Update the name and description of a template
func (mg *Client) UpdateTemplate(ctx context.Context, domain string, template *mtypes.Template) error {
	if template.Name == "" {
		return errors.New("UpdateTemplate() Template.Name cannot be empty")
	}

	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, templatesEndpoint, domain) + "/" + template.Name)
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	p := newUrlEncodedPayload()

	if template.Name != "" {
		p.addValue("name", template.Name)
	}
	if template.Description != "" {
		p.addValue("description", template.Description)
	}

	var resp mtypes.TemplateResp
	err := putResponseFromJSON(ctx, r, p, &resp)
	if err != nil {
		return err
	}
	*template = resp.Item
	return nil
}

// Delete a template given a template name
func (mg *Client) DeleteTemplate(ctx context.Context, domain, name string) error {
	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, templatesEndpoint, domain) + "/" + name)
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	_, err := makeDeleteRequest(ctx, r)
	return err
}

type TemplatesIterator struct {
	mtypes.ListTemplateResp
	mg  Mailgun
	err error
}

type ListTemplateOptions struct {
	Limit  int
	Active bool
}

// List all available templates
func (mg *Client) ListTemplates(domain string, opts *ListTemplateOptions) *TemplatesIterator {
	r := newHTTPRequest(generateApiV3UrlWithDomain(mg, templatesEndpoint, domain))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	if opts != nil {
		if opts.Limit != 0 {
			r.addParameter("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Active {
			r.addParameter("active", "yes")
		}
	}
	url, err := r.generateUrlWithParameters()
	return &TemplatesIterator{
		mg:               mg,
		ListTemplateResp: mtypes.ListTemplateResp{Paging: mtypes.Paging{Next: url, First: url}},
		err:              err,
	}
}

// If an error occurred during iteration `Err()` will return non nil
func (ti *TemplatesIterator) Err() error {
	return ti.err
}

// Next retrieves the next page of items from the api. Returns false when there
// no more pages to retrieve or if there was an error. Use `.Err()` to retrieve
// the error
func (ti *TemplatesIterator) Next(ctx context.Context, items *[]mtypes.Template) bool {
	if ti.err != nil {
		return false
	}
	ti.err = ti.fetch(ctx, ti.Paging.Next)
	if ti.err != nil {
		return false
	}
	cpy := make([]mtypes.Template, len(ti.Items))
	copy(cpy, ti.Items)
	*items = cpy

	return len(ti.Items) != 0
}

// First retrieves the first page of items from the api. Returns false if there
// was an error. It also sets the iterator object to the first page.
// Use `.Err()` to retrieve the error.
func (ti *TemplatesIterator) First(ctx context.Context, items *[]mtypes.Template) bool {
	if ti.err != nil {
		return false
	}
	ti.err = ti.fetch(ctx, ti.Paging.First)
	if ti.err != nil {
		return false
	}
	cpy := make([]mtypes.Template, len(ti.Items))
	copy(cpy, ti.Items)
	*items = cpy
	return true
}

// Last retrieves the last page of items from the api.
// Calling Last() is invalid unless you first call First() or Next()
// Returns false if there was an error. It also sets the iterator object
// to the last page. Use `.Err()` to retrieve the error.
func (ti *TemplatesIterator) Last(ctx context.Context, items *[]mtypes.Template) bool {
	if ti.err != nil {
		return false
	}
	ti.err = ti.fetch(ctx, ti.Paging.Last)
	if ti.err != nil {
		return false
	}
	cpy := make([]mtypes.Template, len(ti.Items))
	copy(cpy, ti.Items)
	*items = cpy
	return true
}

// Previous retrieves the previous page of items from the api. Returns false when there
// no more pages to retrieve or if there was an error. Use `.Err()` to retrieve
// the error if any
func (ti *TemplatesIterator) Previous(ctx context.Context, items *[]mtypes.Template) bool {
	if ti.err != nil {
		return false
	}
	if ti.Paging.Previous == "" {
		return false
	}
	ti.err = ti.fetch(ctx, ti.Paging.Previous)
	if ti.err != nil {
		return false
	}
	cpy := make([]mtypes.Template, len(ti.Items))
	copy(cpy, ti.Items)
	*items = cpy

	return len(ti.Items) != 0
}

func (ti *TemplatesIterator) fetch(ctx context.Context, url string) error {
	ti.Items = nil
	r := newHTTPRequest(url)
	r.setClient(ti.mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, ti.mg.APIKey())

	return getResponseFromJSON(ctx, r, &ti.ListTemplateResp)
}
