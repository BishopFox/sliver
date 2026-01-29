// Package mailgun provides methods for interacting with the Mailgun API.  It
// automates the HTTP request/response cycle, encodings, and other details
// needed by the API.  This SDK lets you do everything the API lets you, in a
// more Go-friendly way.
//
// For further information please see the Mailgun documentation at
// http://documentation.mailgun.com/
//
//	Original Author: Michael Banzon
//	Contributions:   Samuel A. Falvo II <sam.falvo %at% rackspace.com>
//	                 Derrick J. Wippler <thrawn01 %at% gmail.com>
//
// # Examples
//
// All functions and method have a corresponding test, so if you don't find an
// example for a function you'd like to know more about, please check for a
// corresponding test. Of course, contributions to the documentation are always
// welcome as well. Feel free to submit a pull request or open a Github issue
// if you cannot find an example to suit your needs.
//
// # List iterators
//
// Most methods that begin with `List` return an iterator which simplfies
// paging through large result sets returned by the mailgun API. Most `List`
// methods allow you to specify a `Limit` parameter which as you'd expect,
// limits the number of items returned per page.  Note that, at present,
// Mailgun imposes its own cap of 100 items per page, for all API endpoints.
//
// For example, the following iterates over all pages of events 100 items at a time
//
//	mg := mailgun.NewMailgun("your-api-key")
//	it := mg.ListEvents(&mailgun.ListEventOptions{Limit: 100})
//
//	// The entire operation should not take longer than 30 seconds
//	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
//	defer cancel()
//
//	// For each page of 100 events
//	var page []mailgun.Event
//	for it.Next(ctx, &page) {
//	  for _, e := range page {
//	    // Do something with 'e'
//	  }
//	}
//
// # License
//
// Copyright (c) 2013-2019, Michael Banzon.
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// * Redistributions of source code must retain the above copyright notice, this
// list of conditions and the following disclaimer.
//
// * Redistributions in binary form must reproduce the above copyright notice, this
// list of conditions and the following disclaimer in the documentation and/or
// other materials provided with the distribution.
//
// * Neither the names of Mailgun, Michael Banzon, nor the names of their
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON
// ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package mailgun

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mailgun/mailgun-go/v5/mtypes"
)

// Debug set true to write the HTTP requests in curl for to stdout
var Debug = false

const (
	// APIBase - base URL the library uses to contact mailgun. Use SetAPIBase() to override
	APIBase   = "https://api.mailgun.net"
	APIBaseUS = APIBase
	APIBaseEU = "https://api.eu.mailgun.net"

	OnBehalfOfHeader = "X-Mailgun-On-Behalf-Of"

	basicAuthUser = "api"
)

// Mailgun Send endpoints
const (
	messagesEndpoint     = "messages"
	mimeMessagesEndpoint = "messages.mime"
	bouncesEndpoint      = "bounces"
	metricsEndpoint      = "analytics/metrics"
	domainsEndpoint      = "domains"
	tagsEndpoint         = "tags"
	eventsEndpoint       = "events"
	unsubscribesEndpoint = "unsubscribes"
	routesEndpoint       = "routes"
	ipsEndpoint          = "ips"
	ipWarmupsEndpoint    = "ip_warmups"
	exportsEndpoint      = "exports"
	webhooksEndpoint     = "webhooks"
	listsEndpoint        = "lists"
	templatesEndpoint    = "templates"
	accountsEndpoint     = "accounts"
	subaccountsEndpoint  = "subaccounts"
)

// Mailgun defines the supported subset of the Mailgun API.
// The Mailgun API may contain additional features which have been deprecated since writing this SDK.
// This SDK only covers currently supported interface endpoints.
//
// Note that Mailgun reserves the right to deprecate endpoints.
// Some endpoints listed in this interface may, at any time, become obsolete.
// Always double-check with the Mailgun API Documentation to
// determine the currently supported feature set.
type Mailgun interface {
	APIBase() string
	APIKey() string
	HTTPClient() *http.Client
	SetHTTPClient(client *http.Client)
	SetAPIBase(url string) error
	AddOverrideHeader(k string, v string)

	// Send attempts to queue a message (see CommonMessage, NewMessage, and its methods) for delivery.
	Send(ctx context.Context, m Message) (mtypes.SendMessageResponse, error)
	ReSend(ctx context.Context, url string, recipients ...string) (mtypes.SendMessageResponse, error)

	ListBounces(domain string, opts *ListOptions) *BouncesIterator
	GetBounce(ctx context.Context, domain, address string) (mtypes.Bounce, error)
	AddBounce(ctx context.Context, domain, address, code, err string) error
	DeleteBounce(ctx context.Context, domain, address string) error
	DeleteBounceList(ctx context.Context, domain string) error

	ListMetrics(opts MetricsOptions) (*MetricsIterator, error)

	GetTag(ctx context.Context, domain, tag string) (mtypes.Tag, error)
	DeleteTag(ctx context.Context, domain, tag string) error
	ListTags(domain string, opts *ListTagOptions) *TagIterator

	ListDomains(opts *ListDomainsOptions) *DomainsIterator
	GetDomain(ctx context.Context, domain string, opts *GetDomainOptions) (mtypes.GetDomainResponse, error)
	CreateDomain(ctx context.Context, domain string, opts *CreateDomainOptions) (mtypes.GetDomainResponse, error)
	VerifyDomain(ctx context.Context, domain string) (mtypes.GetDomainResponse, error)
	UpdateDomain(ctx context.Context, domain string, opts *UpdateDomainOptions) error
	DeleteDomain(ctx context.Context, domain string) error
	// Deprecated: use VerifyDomain instead.
	VerifyAndReturnDomain(ctx context.Context, domain string) (mtypes.GetDomainResponse, error)
	ListIPDomains(ip string, opts *ListIPDomainOptions) *IPDomainsIterator

	// Deprecated: use UpdateDomain instead
	UpdateDomainConnection(ctx context.Context, domain string, dc mtypes.DomainConnection) error
	// Deprecated: use GetDomain instead
	GetDomainConnection(ctx context.Context, domain string) (mtypes.DomainConnection, error)

	GetDomainTracking(ctx context.Context, domain string) (mtypes.DomainTracking, error)
	UpdateClickTracking(ctx context.Context, domain, active string) error
	UpdateUnsubscribeTracking(ctx context.Context, domain, active, htmlFooter, textFooter string) error
	UpdateOpenTracking(ctx context.Context, domain, active string) error

	UpdateDomainDkimSelector(ctx context.Context, domain, dkimSelector string) error

	GetStoredMessage(ctx context.Context, url string) (mtypes.StoredMessage, error)
	GetStoredMessageRaw(ctx context.Context, id string) (mtypes.StoredMessageRaw, error)
	GetStoredAttachment(ctx context.Context, url string) ([]byte, error)

	ListCredentials(domain string, opts *ListOptions) *CredentialsIterator
	CreateCredential(ctx context.Context, domain, login, password string) error
	ChangeCredentialPassword(ctx context.Context, domain, login, password string) error
	DeleteCredential(ctx context.Context, domain, login string) error

	ListUnsubscribes(domain string, opts *ListOptions) *UnsubscribesIterator
	GetUnsubscribe(ctx context.Context, domain, address string) (mtypes.Unsubscribe, error)
	CreateUnsubscribe(ctx context.Context, domain, address, tag string) error
	CreateUnsubscribes(ctx context.Context, domain string, unsubscribes []mtypes.Unsubscribe) error
	DeleteUnsubscribe(ctx context.Context, domain, address string) error
	DeleteUnsubscribeWithTag(ctx context.Context, domain, address, tag string) error

	ListComplaints(domain string, opts *ListOptions) *ComplaintsIterator
	GetComplaint(ctx context.Context, domain, address string) (mtypes.Complaint, error)
	CreateComplaint(ctx context.Context, domain, address string) error
	CreateComplaints(ctx context.Context, domain string, addresses []string) error
	DeleteComplaint(ctx context.Context, domain, address string) error

	ListRoutes(opts *ListOptions) *RoutesIterator
	GetRoute(ctx context.Context, id string) (mtypes.Route, error)
	CreateRoute(ctx context.Context, route mtypes.Route) (mtypes.Route, error)
	DeleteRoute(ctx context.Context, id string) error
	UpdateRoute(ctx context.Context, id string, r mtypes.Route) (mtypes.Route, error)

	ListWebhooks(ctx context.Context, domain string) (map[string][]string, error)
	CreateWebhook(ctx context.Context, domain, kind string, url []string) error
	DeleteWebhook(ctx context.Context, domain, kind string) error
	GetWebhook(ctx context.Context, domain, kind string) ([]string, error)
	UpdateWebhook(ctx context.Context, domain, kind string, url []string) error
	VerifyWebhookSignature(sig mtypes.Signature) (verified bool, err error)

	ListMailingLists(opts *ListOptions) *ListsIterator
	CreateMailingList(ctx context.Context, ml mtypes.MailingList) (mtypes.MailingList, error)
	DeleteMailingList(ctx context.Context, address string) error
	GetMailingList(ctx context.Context, address string) (mtypes.MailingList, error)
	UpdateMailingList(ctx context.Context, address string, ml mtypes.MailingList) (mtypes.MailingList, error)

	ListMembers(listAddress string, opts *ListOptions) *MemberListIterator
	GetMember(ctx context.Context, memberAddress, listAddress string) (mtypes.Member, error)
	CreateMember(ctx context.Context, merge bool, listAddress string, member mtypes.Member) error
	CreateMemberList(ctx context.Context, subscribed *bool, listAddress string, newMembers []any) error
	UpdateMember(ctx context.Context, memberAddress, listAddress string, member mtypes.Member) (mtypes.Member, error)
	DeleteMember(ctx context.Context, memberAddress, listAddress string) error

	ListEvents(domain string, opts *ListEventOptions) *EventIterator
	PollEvents(domain string, opts *ListEventOptions) *EventPoller

	ListIPs(ctx context.Context, dedicated, enabled bool) ([]mtypes.IPAddress, error)
	GetIP(ctx context.Context, ip string) (mtypes.IPAddress, error)
	ListDomainIPs(ctx context.Context, domain string) ([]mtypes.IPAddress, error)
	AddDomainIP(ctx context.Context, domain, ip string) error
	DeleteDomainIP(ctx context.Context, domain, ip string) error

	ListIPWarmups() *IPWarmupsIterator
	GetIPWarmup(ctx context.Context, ip string) (mtypes.IPWarmupDetails, error)
	CreateIPWarmup(ctx context.Context, ip string) error
	DeleteIPWarmup(ctx context.Context, ip string) error

	ListExports(ctx context.Context, url string) ([]mtypes.Export, error)
	GetExport(ctx context.Context, id string) (mtypes.Export, error)
	GetExportLink(ctx context.Context, id string) (string, error)
	CreateExport(ctx context.Context, url string) error

	GetTagLimits(ctx context.Context, domain string) (mtypes.TagLimits, error)

	CreateTemplate(ctx context.Context, domain string, template *mtypes.Template) error
	GetTemplate(ctx context.Context, domain, name string) (mtypes.Template, error)
	UpdateTemplate(ctx context.Context, domain string, template *mtypes.Template) error
	DeleteTemplate(ctx context.Context, domain, name string) error
	ListTemplates(domain string, opts *ListTemplateOptions) *TemplatesIterator

	AddTemplateVersion(ctx context.Context, domain, templateName string, version *mtypes.TemplateVersion) error
	GetTemplateVersion(ctx context.Context, domain, templateName, tag string) (mtypes.TemplateVersion, error)
	UpdateTemplateVersion(ctx context.Context, domain, templateName string, version *mtypes.TemplateVersion) error
	DeleteTemplateVersion(ctx context.Context, domain, templateName, tag string) error
	ListTemplateVersions(domain, templateName string, opts *ListOptions) *TemplateVersionsIterator

	ValidateEmail(ctx context.Context, email string, mailBoxVerify bool) (mtypes.ValidateEmailResponse, error)

	ListAlertsEvents(context.Context, *ListAlertsEventsOptions) (*mtypes.AlertsEventsResponse, error)
	ListAlerts(context.Context, *ListAlertsOptions) (*mtypes.AlertsSettingsResponse, error)
	AddAlert(context.Context, mtypes.AlertsEventSettingRequest) (*mtypes.AlertsEventSettingResponse, error)
	DeleteAlert(ctx context.Context, id uuid.UUID) error

	ListMonitoredDomains(opts ListMonitoredDomainsOptions) (*MonitoredDomainsIterator, error)

	ListSubaccounts(opts *ListSubaccountsOptions) *SubaccountsIterator
	CreateSubaccount(ctx context.Context, subaccountName string) (mtypes.SubaccountResponse, error)
	GetSubaccount(ctx context.Context, subaccountID string) (mtypes.SubaccountResponse, error)
	EnableSubaccount(ctx context.Context, subaccountID string) (mtypes.SubaccountResponse, error)
	DisableSubaccount(ctx context.Context, subaccountID string) (mtypes.SubaccountResponse, error)

	SetOnBehalfOfSubaccount(subaccountID string)
	RemoveOnBehalfOfSubaccount()

	ListAPIKeys(ctx context.Context, opts *ListAPIKeysOptions) ([]mtypes.APIKey, error)
	CreateAPIKey(ctx context.Context, role string, opts *CreateAPIKeyOptions) (mtypes.APIKey, error)
	DeleteAPIKey(ctx context.Context, id string) error
	RegeneratePublicAPIKey(ctx context.Context) (mtypes.RegeneratePublicAPIKeyResponse, error)
}

// Client bundles data needed by a large number of methods in order to interact with the Mailgun API.
type Client struct {
	apiBase           string
	apiKey            string
	webhookSigningKey string
	client            *http.Client
	overrideHeaders   map[string]string
}

// NewMailgun creates a new client instance.
func NewMailgun(apiKey string) *Client {
	return &Client{
		apiBase: APIBase,
		apiKey:  apiKey,
		client:  http.DefaultClient,
	}
}

// NewMailgunFromEnv returns a new Mailgun client using the environment variables
// MG_API_KEY, MG_URL, and MG_WEBHOOK_SIGNING_KEY
func NewMailgunFromEnv() (*Client, error) {
	apiKey := os.Getenv("MG_API_KEY")
	if apiKey == "" {
		return nil, errors.New("required environment variable MG_API_KEY not defined")
	}

	mg := NewMailgun(apiKey)

	url := os.Getenv("MG_URL")
	if url != "" {
		err := mg.SetAPIBase(url)
		if err != nil {
			return nil, err
		}
	}

	webhookSigningKey := os.Getenv("MG_WEBHOOK_SIGNING_KEY")
	if webhookSigningKey != "" {
		mg.SetWebhookSigningKey(webhookSigningKey)
	}

	return mg, nil
}

// APIBase returns the API Base URL configured for this client.
func (mg *Client) APIBase() string {
	return mg.apiBase
}

// APIKey returns the API key configured for this client.
func (mg *Client) APIKey() string {
	return mg.apiKey
}

// HTTPClient returns the HTTP client configured for this client.
func (mg *Client) HTTPClient() *http.Client {
	return mg.client
}

// SetHTTPClient updates the HTTP client for this client.
func (mg *Client) SetHTTPClient(c *http.Client) {
	mg.client = c
}

// WebhookSigningKey returns the webhook signing key configured for this client
func (mg *Client) WebhookSigningKey() string {
	return mg.webhookSigningKey
}

// SetWebhookSigningKey updates the webhook signing key for this client
func (mg *Client) SetWebhookSigningKey(webhookSigningKey string) {
	mg.webhookSigningKey = webhookSigningKey
}

// SetOnBehalfOfSubaccount sets X-Mailgun-On-Behalf-Of header to SUBACCOUNT_ACCOUNT_ID in order to perform API request
// on behalf of subaccount.
func (mg *Client) SetOnBehalfOfSubaccount(subaccountId string) {
	mg.AddOverrideHeader(OnBehalfOfHeader, subaccountId)
}

// RemoveOnBehalfOfSubaccount remove X-Mailgun-On-Behalf-Of header for primary usage.
func (mg *Client) RemoveOnBehalfOfSubaccount() {
	delete(mg.overrideHeaders, OnBehalfOfHeader)
}

// SetAPIBase updates the API Base URL for this client.
//
//	// For EU Customers
//	mg.SetAPIBase(mailgun.APIBaseEU)
//
//	// For US Customers
//	mg.SetAPIBase(mailgun.APIBaseUS)
//
//	// Set a custom base API
//	mg.SetAPIBase("https://localhost")
func (mg *Client) SetAPIBase(address string) error {
	if invalidURL.MatchString(address) {
		return errors.New(`APIBase must not contain a version; SetAPIBase("https://host")`)
	}

	mg.apiBase = address
	return nil
}

// AddOverrideHeader allows the user to specify additional headers that will be included in the HTTP request
// This is mostly useful for testing the Mailgun API hosted at a different endpoint.
func (mg *Client) AddOverrideHeader(k, v string) {
	if mg.overrideHeaders == nil {
		mg.overrideHeaders = make(map[string]string)
	}
	mg.overrideHeaders[k] = v
}

// ListOptions used by List methods to specify what list parameters to send to the mailgun API
type ListOptions struct {
	Limit int
}

// TimeToFloat given time.Time{} return a float64 as given in mailgun event timestamps
func TimeToFloat(t time.Time) float64 {
	return float64(t.Unix()) + (float64(t.Nanosecond()/int(time.Microsecond)) / float64(1000000))
}

// TODO(vtopc): sort all these generate URL functions(some are generateApi...Url, other are generate...ApiUrl)

func generateApiUrlWithDomain(m Mailgun, version int, endpoint, domain string) string {
	return fmt.Sprintf("%s/v%d/%s/%s", m.APIBase(), version, domain, endpoint)
}

// generateApiV3UrlWithDomain renders a URL for an API endpoint using the domain and endpoint name.
func generateApiV3UrlWithDomain(m Mailgun, endpoint, domain string) string {
	return generateApiUrlWithDomain(m, 3, endpoint, domain)
}

// generateMemberApiUrl renders a URL relevant for specifying mailing list members.
// The address parameter refers to the mailing list in question.
func generateMemberApiUrl(m Mailgun, endpoint, address string) string {
	return fmt.Sprintf("%s/v3/%s/%s/members", m.APIBase(), endpoint, address)
}

// generateApiV3UrlWithTarget works as generateApiV3UrlWithDomain
// but consumes an additional resource parameter called 'target'.
func generateApiV3UrlWithTarget(m Mailgun, endpoint, domain, target string) string {
	tail := ""
	if target != "" {
		tail = fmt.Sprintf("/%s", target)
	}
	return fmt.Sprintf("%s%s", generateApiV3UrlWithDomain(m, endpoint, domain), tail)
}

// generateV3DomainsApiUrl renders a URL as generateApiV3UrlWithDomain, but
// addresses a family of functions which have a non-standard URL structure.
// Most URLs consume a domain in the 2nd position, but some endpoints
// require the word "domains" to be there instead.
func generateV3DomainsApiUrl(m Mailgun, endpoint, domain string) string {
	return fmt.Sprintf("%s/v3/domains/%s/%s", m.APIBase(), domain, endpoint)
}

// generateCredentialsUrl renders a URL as generateV3DomainsApiUrl,
// but focuses on the SMTP credentials family of API functions.
func generateCredentialsUrl(m Mailgun, domain, login string) string {
	tail := ""
	if login != "" {
		tail = fmt.Sprintf("/%s", login)
	}
	return generateV3DomainsApiUrl(m, fmt.Sprintf("credentials%s", tail), domain)
}

// generateApiUrl returns domain agnostic URL.
func generateApiUrl(m Mailgun, version int, endpoint string) string {
	return fmt.Sprintf("%s/v%d/%s", m.APIBase(), version, endpoint)
}

// formatMailgunTime translates a timestamp into a human-readable form.
func formatMailgunTime(t time.Time) string {
	return t.Format("Mon, 2 Jan 2006 15:04:05 -0700")
}

func ptr[T any](v T) *T {
	return &v
}

// TODO(vtopc): remove boolToString and use strconv.FormatBool() directly.
func boolToString(b bool) string {
	return strconv.FormatBool(b)
}
