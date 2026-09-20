package mailgun

import (
	"context"
	"strconv"
	"strings"

	"github.com/mailgun/mailgun-go/v5/mtypes"
)

type ListDomainsOptions struct {
	Limit int

	// Get only domains with a specific state.
	State *mtypes.DomainState

	// If sorting is not specified domains are returned in reverse creation date order.
	Sort *string

	// Get only domains with a specific authority.
	Authority *string

	// Search domains by the given partial or complete name. Does not support wildcards.
	Search *string

	// Search on every domain that belongs to any subaccounts under this account.
	IncludeSubaccounts *bool
}

// ListDomains retrieves a set of domains from Mailgun.
// https://documentation.mailgun.com/docs/mailgun/api-reference/send/mailgun/domains/get-v4-domains
func (mg *Client) ListDomains(opts *ListDomainsOptions) *DomainsIterator {
	var limit int
	if opts != nil {
		limit = opts.Limit
	}

	if limit == 0 {
		limit = 100
	}
	return &DomainsIterator{
		mg:                  mg,
		url:                 generateApiUrl(mg, 4, domainsEndpoint),
		ListDomainsResponse: mtypes.ListDomainsResponse{TotalCount: -1},
		limit:               limit,
		opts:                opts,
	}
}

type DomainsIterator struct {
	mtypes.ListDomainsResponse

	limit  int
	opts   *ListDomainsOptions
	mg     Mailgun
	offset int
	url    string
	err    error
}

// Err if an error occurred during iteration `Err()` will return non nil
func (ri *DomainsIterator) Err() error {
	return ri.err
}

// Offset returns the current offset of the iterator
func (ri *DomainsIterator) Offset() int {
	return ri.offset
}

// Next retrieves the next page of items from the api. Returns false when there
// no more pages to retrieve or if there was an error. Use `.Err()` to retrieve
// the error
func (ri *DomainsIterator) Next(ctx context.Context, items *[]mtypes.Domain) bool {
	if ri.err != nil {
		return false
	}

	ri.err = ri.fetch(ctx, ri.offset, ri.limit)
	if ri.err != nil {
		return false
	}

	cpy := make([]mtypes.Domain, len(ri.Items))
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
func (ri *DomainsIterator) First(ctx context.Context, items *[]mtypes.Domain) bool {
	if ri.err != nil {
		return false
	}
	ri.err = ri.fetch(ctx, 0, ri.limit)
	if ri.err != nil {
		return false
	}
	cpy := make([]mtypes.Domain, len(ri.Items))
	copy(cpy, ri.Items)
	*items = cpy
	ri.offset = len(ri.Items)
	return true
}

// Last retrieves the last page of items from the api.
// Calling Last() is invalid unless you first call First() or Next()
// Returns false if there was an error. It also sets the iterator object
// to the last page. Use `.Err()` to retrieve the error.
func (ri *DomainsIterator) Last(ctx context.Context, items *[]mtypes.Domain) bool {
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
	cpy := make([]mtypes.Domain, len(ri.Items))
	copy(cpy, ri.Items)
	*items = cpy
	return true
}

// Previous retrieves the previous page of items from the api. Returns false when there
// no more pages to retrieve or if there was an error. Use `.Err()` to retrieve
// the error if any
func (ri *DomainsIterator) Previous(ctx context.Context, items *[]mtypes.Domain) bool {
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
	cpy := make([]mtypes.Domain, len(ri.Items))
	copy(cpy, ri.Items)
	*items = cpy

	return len(ri.Items) != 0
}

func (ri *DomainsIterator) fetch(ctx context.Context, skip, limit int) error {
	ri.Items = nil
	r := newHTTPRequest(ri.url)
	r.setBasicAuth(basicAuthUser, ri.mg.APIKey())
	r.setClient(ri.mg.HTTPClient())

	if skip != 0 {
		r.addParameter("skip", strconv.Itoa(skip))
	}

	// TODO(vtopc): switch to opts.Limit:
	if limit != 0 {
		r.addParameter("limit", strconv.Itoa(limit))
	}

	if ri.opts != nil {
		if ri.opts.State != nil {
			r.addParameter("state", string(*ri.opts.State))
		}
		if ri.opts.Sort != nil {
			r.addParameter("sort", *ri.opts.Sort)
		}
		if ri.opts.Authority != nil {
			r.addParameter("authority", *ri.opts.Authority)
		}
		if ri.opts.Search != nil {
			r.addParameter("search", *ri.opts.Search)
		}
		if ri.opts.IncludeSubaccounts != nil {
			r.addParameter("include_subaccounts", strconv.FormatBool(*ri.opts.IncludeSubaccounts))
		}
	}

	return getResponseFromJSON(ctx, r, &ri.ListDomainsResponse)
}

type GetDomainOptions struct {
	// If set to true, domain payload will include dkim_host, mailfrom_host and pod
	Extended *bool

	// Domain payload will include sending and receiving dns records payload
	WithDNS *bool
}

// GetDomain retrieves detailed information about the named domain.
// https://documentation.mailgun.com/docs/mailgun/api-reference/send/mailgun/domains/get-v4-domains--name-
func (mg *Client) GetDomain(ctx context.Context, domain string, opts *GetDomainOptions) (mtypes.GetDomainResponse, error) {
	r := newHTTPRequest(generateApiUrl(mg, 4, domainsEndpoint) + "/" + domain)
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	if opts != nil {
		if opts.Extended != nil {
			r.addParameter("h:extended", strconv.FormatBool(*opts.Extended))
		}

		if opts.WithDNS != nil {
			r.addParameter("h:with_dns", strconv.FormatBool(*opts.WithDNS))
		}
	}

	var resp mtypes.GetDomainResponse
	err := getResponseFromJSON(ctx, r, &resp)
	return resp, err
}

// VerifyDomain verifies the domains DNS records (includes A, CNAME, SPF,
// DKIM and MX records) to ensure the domain is ready and able to send.
func (mg *Client) VerifyDomain(ctx context.Context, domain string) (mtypes.GetDomainResponse, error) {
	r := newHTTPRequest(generateApiUrl(mg, 4, domainsEndpoint) + "/" + domain + "/verify")
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	// TODO(vtopc): why newUrlEncodedPayload()?
	payload := newUrlEncodedPayload()
	var resp mtypes.GetDomainResponse
	err := putResponseFromJSON(ctx, r, payload, &resp)
	return resp, err
}

// VerifyAndReturnDomain verifies the domains DNS records (includes A, CNAME, SPF,
// DKIM and MX records) to ensure the domain is ready and able to send.
//
// Deprecated: use VerifyDomain instead.
//
// TODO(v6): remove this method
func (mg *Client) VerifyAndReturnDomain(ctx context.Context, domain string) (mtypes.GetDomainResponse, error) {
	return mg.VerifyDomain(ctx, domain)
}

// CreateDomainOptions - optional parameters when creating a domain
// https://documentation.mailgun.com/docs/mailgun/api-reference/openapi-final/tag/Domains/#tag/Domains/operation/POST-v4-domains
type CreateDomainOptions struct {
	Password   string
	SpamAction mtypes.SpamAction
	// Wildcard parameter instructs Mailgun to treat all subdomains of this domain uniformly if true,
	// and as different domains if false.
	Wildcard                   bool
	ForceDKIMAuthority         bool
	DKIMKeySize                int
	IPs                        []string
	WebScheme                  string
	UseAutomaticSenderSecurity bool
	ArchiveTo                  string
	DKIMHostName               string
	DKIMSelector               string
	ForceRootDKIMHost          bool
	EncryptIncomingMessage     bool
	PoolID                     string
	RequireTLS                 bool
	SkipVerification           bool
	WebPrefix                  string
	MessageTTL                 int
}

// CreateDomain instructs Mailgun to create a new domain for your account.
// The domain parameter identifies the domain.
// https://documentation.mailgun.com/docs/mailgun/api-reference/send/mailgun/domains/post-v4-domains
func (mg *Client) CreateDomain(ctx context.Context, domain string, opts *CreateDomainOptions) (mtypes.GetDomainResponse, error) {
	r := newHTTPRequest(generateApiUrl(mg, 4, domainsEndpoint))
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	payload := NewFormDataPayload()
	payload.addValue("name", domain)

	if opts != nil {
		if opts.SpamAction != "" {
			payload.addValue("spam_action", string(opts.SpamAction))
		}
		if opts.Wildcard {
			payload.addValue("wildcard", boolToString(opts.Wildcard))
		}
		if opts.ForceDKIMAuthority {
			payload.addValue("force_dkim_authority", boolToString(opts.ForceDKIMAuthority))
		}
		if opts.DKIMKeySize != 0 {
			payload.addValue("dkim_key_size", strconv.Itoa(opts.DKIMKeySize))
		}
		if len(opts.IPs) != 0 {
			payload.addValue("ips", strings.Join(opts.IPs, ","))
		}
		if opts.Password != "" {
			payload.addValue("smtp_password", opts.Password)
		}
		if opts.WebScheme != "" {
			payload.addValue("web_scheme", opts.WebScheme)
		}
		if opts.UseAutomaticSenderSecurity {
			payload.addValue("use_automatic_sender_security", boolToString(opts.UseAutomaticSenderSecurity))
		}
		if opts.ArchiveTo != "" {
			payload.addValue("archive_to", opts.ArchiveTo)
		}
		if opts.DKIMHostName != "" {
			payload.addValue("dkim_host_name", opts.DKIMHostName)
		}
		if opts.DKIMSelector != "" {
			payload.addValue("dkim_selector", opts.DKIMSelector)
		}
		if opts.ForceRootDKIMHost {
			payload.addValue("force_root_dkim_host", boolToString(opts.ForceRootDKIMHost))
		}
		if opts.EncryptIncomingMessage {
			payload.addValue("encrypt_incoming_message", boolToString(opts.EncryptIncomingMessage))
		}
		if opts.PoolID != "" {
			payload.addValue("pool_id", opts.PoolID)
		}
		if opts.RequireTLS {
			payload.addValue("require_tls", boolToString(opts.RequireTLS))
		}
		if opts.SkipVerification {
			payload.addValue("skip_verification", boolToString(opts.SkipVerification))
		}
		if opts.WebPrefix != "" {
			payload.addValue("web_prefix", opts.WebPrefix)
		}
		if opts.MessageTTL != 0 {
			payload.addValue("message_ttl", strconv.Itoa(opts.MessageTTL))
		}
	}
	var resp mtypes.GetDomainResponse
	err := postResponseFromJSON(ctx, r, payload, &resp)
	return resp, err
}

// DeleteDomain instructs Mailgun to dispose of the named domain name
func (mg *Client) DeleteDomain(ctx context.Context, domain string) error {
	r := newHTTPRequest(generateApiUrl(mg, 3, domainsEndpoint) + "/" + domain)
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())
	_, err := makeDeleteRequest(ctx, r)
	return err
}

// UpdateDomainOptions options for updating a domain
type UpdateDomainOptions struct {
	Password                   string
	SpamAction                 mtypes.SpamAction
	Wildcard                   *bool
	WebScheme                  string
	WebPrefix                  string
	RequireTLS                 *bool
	SkipVerification           *bool
	UseAutomaticSenderSecurity *bool
	ArchiveTo                  string
	MailFromHost               string
	MessageTTL                 *int
}

// UpdateDomain updates a domain's attributes.
func (mg *Client) UpdateDomain(ctx context.Context, domain string, opts *UpdateDomainOptions) error {
	r := newHTTPRequest(generateApiUrl(mg, 4, domainsEndpoint) + "/" + domain)
	r.setClient(mg.HTTPClient())
	r.setBasicAuth(basicAuthUser, mg.APIKey())

	payload := NewFormDataPayload()

	if opts != nil {
		if opts.Password != "" {
			payload.addValue("smtp_password", opts.Password)
		}
		if opts.SpamAction != "" {
			payload.addValue("spam_action", string(opts.SpamAction))
		}
		if opts.Wildcard != nil {
			payload.addValue("wildcard", boolToString(*opts.Wildcard))
		}
		if opts.WebScheme != "" {
			payload.addValue("web_scheme", opts.WebScheme)
		}
		if opts.WebPrefix != "" {
			payload.addValue("web_prefix", opts.WebPrefix)
		}
		if opts.RequireTLS != nil {
			payload.addValue("require_tls", boolToString(*opts.RequireTLS))
		}
		if opts.SkipVerification != nil {
			payload.addValue("skip_verification", boolToString(*opts.SkipVerification))
		}
		if opts.UseAutomaticSenderSecurity != nil {
			payload.addValue("use_automatic_sender_security", boolToString(*opts.UseAutomaticSenderSecurity))
		}
		if opts.ArchiveTo != "" {
			payload.addValue("archive_to", opts.ArchiveTo)
		}
		if opts.MailFromHost != "" {
			payload.addValue("mailfrom_host", opts.MailFromHost)
		}
		if opts.MessageTTL != nil {
			payload.addValue("message_ttl", strconv.Itoa(*opts.MessageTTL))
		}
	}

	_, err := makePutRequest(ctx, r, payload)

	return err
}
