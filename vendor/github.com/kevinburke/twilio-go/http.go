package twilio

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kevinburke/rest/restclient"
	"github.com/kevinburke/rest/resterror"
)

// The twilio-go version. Run "make release" to bump this number.
const Version = "2.9.0"
const userAgent = "twilio-go/" + Version

// The base URL serving the API. Override this for testing.
var BaseURL = "https://api.twilio.com"

// The base URL for Twilio Monitor.
var MonitorBaseURL = "https://monitor.twilio.com"

// Version of the Twilio Monitor API.
const MonitorVersion = "v1"

// The base URL for Twilio Pricing.
var PricingBaseURL = "https://pricing.twilio.com"

// Version of the Twilio Pricing API.
const PricingVersion = "v2"

var FaxBaseURL = "https://fax.twilio.com"

const FaxVersion = "v1"

// The base URL for Twilio Wireless.
var WirelessBaseURL = "https://wireless.twilio.com"

// Version of the Twilio Wireless API.
const WirelessVersion = "v1"

// The APIVersion to use. Your mileage may vary using other values for the
// APIVersion; the resource representations may not match.
const APIVersion = "2010-04-01"

const NotifyBaseURL = "https://notify.twilio.com"
const NotifyVersion = "v1"

// Lookup service
const LookupBaseURL = "https://lookups.twilio.com"
const LookupVersion = "v1"

// Verify service
var VerifyBaseURL = "https://verify.twilio.com"

const VerifyVersion = "v2"

// Video service
var VideoBaseUrl = "https://video.twilio.com"

const VideoVersion = "v1"

var TaskRouterBaseUrl = "https://taskrouter.twilio.com"

const TaskRouterVersion = "v1"

// Voice Insights service
var InsightsBaseUrl = "https://insights.twilio.com"

const InsightsVersion = "v1"

type Client struct {
	*restclient.Client
	Monitor    *Client
	Pricing    *Client
	Fax        *Client
	Wireless   *Client
	Notify     *Client
	Lookup     *Client
	Verify     *Client
	Video      *Client
	TaskRouter *Client
	Insights   *Client

	// FullPath takes a path part (e.g. "Messages") and
	// returns the full API path, including the version (e.g.
	// "/2010-04-01/Accounts/AC123/Messages").
	FullPath func(pathPart string) string
	// The API version.
	APIVersion string

	AccountSid string
	AuthToken  string

	// The API Client uses these resources
	Accounts          *AccountService
	Applications      *ApplicationService
	Calls             *CallService
	Conferences       *ConferenceService
	IncomingNumbers   *IncomingNumberService
	Keys              *KeyService
	Media             *MediaService
	Messages          *MessageService
	OutgoingCallerIDs *OutgoingCallerIDService
	Queues            *QueueService
	Recordings        *RecordingService
	Transcriptions    *TranscriptionService
	AvailableNumbers  *AvailableNumberService

	// NewMonitorClient initializes these services
	Alerts *AlertService

	// NewPricingClient initializes these services
	Voice        *VoicePriceService
	Messaging    *MessagingPriceService
	PhoneNumbers *PhoneNumberPriceService

	// NewFaxClient initializes these services
	Faxes *FaxService

	// NewWirelessClient initializes these services
	Sims     *SimService
	Commands *CommandService

	// NewNotifyClient initializes these services
	Credentials *NotifyCredentialsService

	// NewLookupClient initializes these services
	LookupPhoneNumbers *LookupPhoneNumbersService

	// NewVerifyClient initializes these services
	Verifications *VerifyPhoneNumberService
	AccessTokens  *VerifyAccessTokenService
	Challenges    *VerifyChallengeService

	// NewVideoClient initializes these services
	Rooms           *RoomService
	VideoRecordings *VideoRecordingService

	// NewTaskRouterClient initializes these services
	Workspace func(sid string) *WorkspaceService

	// NewInsightsClient initializes these services
	VoiceInsights func(sid string) *VoiceInsightsService
}

const defaultTimeout = 30*time.Second + 500*time.Millisecond

var defaultHttpClient *http.Client

func init() {
	defaultHttpClient = &http.Client{
		Timeout:   defaultTimeout,
		Transport: restclient.DefaultTransport,
	}
}

// An error returned by the Twilio API. We don't want to expose this - let's
// try to standardize on the fields in the HTTP problem spec instead.
type twilioError struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`
	MoreInfo string `json:"more_info"`
	// This will be ignored in favor of the actual HTTP status code
	Status int `json:"status"`
}

func parseTwilioError(resp *http.Response) error {
	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := resp.Body.Close(); err != nil {
		return err
	}
	rerr := new(twilioError)
	err = json.Unmarshal(resBody, rerr)
	if err != nil {
		return fmt.Errorf("invalid response body: %s", string(resBody))
	}
	if rerr.Message == "" {
		return fmt.Errorf("invalid response body: %s", string(resBody))
	}
	return &resterror.Error{
		Title:  rerr.Message,
		Type:   rerr.MoreInfo,
		ID:     strconv.Itoa(rerr.Code),
		Status: resp.StatusCode,
	}
}

// NewFaxClient returns a Client for use with the Twilio Fax API.
func NewFaxClient(accountSid string, authToken string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = defaultHttpClient
	}
	restClient := restclient.New(accountSid, authToken, FaxBaseURL)
	restClient.Client = httpClient
	restClient.UploadType = restclient.FormURLEncoded
	restClient.ErrorParser = parseTwilioError
	c := &Client{
		Client:     restClient,
		AccountSid: accountSid,
		AuthToken:  authToken,
	}
	c.FullPath = func(pathPart string) string {
		return "/" + c.APIVersion + "/" + pathPart
	}
	c.APIVersion = FaxVersion
	c.Faxes = &FaxService{client: c}
	return c
}

func newNewClient(sid, token, baseURL string, client *http.Client) *Client {
	if client == nil {
		client = defaultHttpClient
	}
	restClient := restclient.New(sid, token, baseURL)
	restClient.Client = client
	restClient.UploadType = restclient.FormURLEncoded
	restClient.ErrorParser = parseTwilioError
	c := &Client{
		Client:     restClient,
		AccountSid: sid,
		AuthToken:  token,
	}
	c.FullPath = func(pathPart string) string {
		return "/" + c.APIVersion + "/" + pathPart
	}
	return c
}

// NewWirelessClient returns a Client for use with the Twilio Wireless API.
func NewWirelessClient(accountSid string, authToken string, httpClient *http.Client) *Client {
	c := newNewClient(accountSid, authToken, WirelessBaseURL, httpClient)
	c.APIVersion = WirelessVersion
	c.Sims = &SimService{client: c}
	c.Commands = &CommandService{client: c}
	return c
}

// NewMonitorClient returns a Client for use with the Twilio Monitor API.
func NewMonitorClient(accountSid string, authToken string, httpClient *http.Client) *Client {
	c := newNewClient(accountSid, authToken, MonitorBaseURL, httpClient)
	c.APIVersion = MonitorVersion
	c.Alerts = &AlertService{client: c}
	return c
}

// NewTaskRouterClient returns a Client for use with the Twilio TaskRouter API.
func NewTaskRouterClient(accountSid string, authToken string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	c := newNewClient(accountSid, authToken, TaskRouterBaseUrl, httpClient)
	c.APIVersion = TaskRouterVersion
	c.Workspace = func(sid string) *WorkspaceService {
		return &WorkspaceService{
			Activities: &ActivityService{
				workspaceSid: sid,
				client:       c,
			},
			Queues: &TaskQueueService{
				workspaceSid: sid,
				client:       c,
			},
			Workflows: &WorkflowService{
				workspaceSid: sid,
				client:       c,
			},
			Workers: &WorkerService{
				workspaceSid: sid,
				client:       c,
			},
		}
	}
	return c
}

func NewInsightsClient(accountSid string, authToken string, httpClient *http.Client) *Client {
	c := newNewClient(accountSid, authToken, InsightsBaseUrl, httpClient)
	c.APIVersion = InsightsVersion
	c.VoiceInsights = func(callSid string) *VoiceInsightsService {
		return &VoiceInsightsService{
			Summary: &CallSummaryService{
				callSid: callSid,
				client:  c,
			},
			Metrics: &CallMetricsService{
				callSid: callSid,
				client:  c,
			},
			Events: &CallEventsService{
				callSid: callSid,
				client:  c,
			},
		}
	}
	return c
}

// NewPricingClient returns a new Client to use the pricing API
func NewPricingClient(accountSid string, authToken string, httpClient *http.Client) *Client {
	c := newNewClient(accountSid, authToken, PricingBaseURL, httpClient)
	c.APIVersion = PricingVersion
	c.Voice = &VoicePriceService{
		Countries: &CountryVoicePriceService{client: c},
		Numbers:   &NumberVoicePriceService{client: c},
	}
	c.Messaging = &MessagingPriceService{
		Countries: &CountryMessagingPriceService{client: c},
	}
	c.PhoneNumbers = &PhoneNumberPriceService{
		Countries: &CountryPhoneNumberPriceService{client: c},
	}
	return c
}

func NewNotifyClient(accountSid string, authToken string, httpClient *http.Client) *Client {
	c := newNewClient(accountSid, authToken, NotifyBaseURL, httpClient)
	c.APIVersion = NotifyVersion
	c.Credentials = &NotifyCredentialsService{client: c}
	return c
}

// NewLookupClient returns a new Client to use the lookups API
func NewLookupClient(accountSid string, authToken string, httpClient *http.Client) *Client {
	c := newNewClient(accountSid, authToken, LookupBaseURL, httpClient)
	c.APIVersion = LookupVersion
	c.LookupPhoneNumbers = &LookupPhoneNumbersService{client: c}
	return c
}

// NewVerifyClient returns a new Client to use the verify API
func NewVerifyClient(accountSid string, authToken string, httpClient *http.Client) *Client {
	c := newNewClient(accountSid, authToken, VerifyBaseURL, httpClient)
	c.APIVersion = VerifyVersion
	c.Verifications = &VerifyPhoneNumberService{client: c}
	c.AccessTokens = &VerifyAccessTokenService{client: c}
	c.Challenges = &VerifyChallengeService{client: c}
	return c
}

// NewVideoClient returns a new Client to use the video API
func NewVideoClient(accountSid string, authToken string, httpClient *http.Client) *Client {
	c := newNewClient(accountSid, authToken, VideoBaseUrl, httpClient)
	c.APIVersion = VideoVersion
	c.Rooms = &RoomService{client: c}
	c.VideoRecordings = &VideoRecordingService{client: c}
	return c
}

// NewClient creates a Client for interacting with the Twilio API. This is the
// main entrypoint for API interactions; view the methods on the subresources
// for more information.
func NewClient(accountSid string, authToken string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = defaultHttpClient
	}
	restClient := restclient.New(accountSid, authToken, BaseURL)
	restClient.Client = httpClient
	restClient.UploadType = restclient.FormURLEncoded
	restClient.ErrorParser = parseTwilioError

	c := &Client{Client: restClient, AccountSid: accountSid, AuthToken: authToken}
	c.APIVersion = APIVersion

	c.FullPath = func(pathPart string) string {
		return "/" + strings.Join([]string{c.APIVersion, "Accounts", c.AccountSid, pathPart + ".json"}, "/")
	}
	c.Monitor = NewMonitorClient(accountSid, authToken, httpClient)
	c.Pricing = NewPricingClient(accountSid, authToken, httpClient)
	c.Fax = NewFaxClient(accountSid, authToken, httpClient)
	c.Wireless = NewWirelessClient(accountSid, authToken, httpClient)
	c.Notify = NewNotifyClient(accountSid, authToken, httpClient)
	c.Lookup = NewLookupClient(accountSid, authToken, httpClient)
	c.Verify = NewVerifyClient(accountSid, authToken, httpClient)
	c.Video = NewVideoClient(accountSid, authToken, httpClient)
	c.TaskRouter = NewTaskRouterClient(accountSid, authToken, httpClient)
	c.Insights = NewInsightsClient(accountSid, authToken, httpClient)

	c.Accounts = &AccountService{client: c}
	c.Applications = &ApplicationService{client: c}
	c.Calls = &CallService{client: c}
	c.Conferences = &ConferenceService{client: c}
	c.Keys = &KeyService{client: c}
	c.Media = &MediaService{client: c}
	c.Messages = &MessageService{client: c}
	c.OutgoingCallerIDs = &OutgoingCallerIDService{client: c}
	c.Queues = &QueueService{client: c}
	c.Recordings = &RecordingService{client: c}
	c.Transcriptions = &TranscriptionService{client: c}

	c.IncomingNumbers = &IncomingNumberService{
		NumberPurchasingService: &NumberPurchasingService{
			client:   c,
			pathPart: "",
		},
		client: c,
		Local: &NumberPurchasingService{
			client:   c,
			pathPart: "Local",
		},
		TollFree: &NumberPurchasingService{
			client:   c,
			pathPart: "TollFree",
		},
	}

	c.AvailableNumbers = &AvailableNumberService{
		Local: &AvailableNumberBase{
			client:   c,
			pathPart: "Local",
		},
		Mobile: &AvailableNumberBase{
			client:   c,
			pathPart: "Mobile",
		},

		TollFree: &AvailableNumberBase{
			client:   c,
			pathPart: "TollFree",
		},
		SupportedCountries: &SupportedCountriesService{
			client: c,
		},
	}

	return c
}

// RequestOnBehalfOf will make all future client requests using the same
// Account Sid and Auth Token for Basic Auth, but will use the provided
// subaccountSid in the URL. Use this to make requests on behalf of a
// subaccount, using the parent account's credentials.
//
// RequestOnBehalfOf is *not* thread safe, and modifies the Client's behavior
// for all requests going forward.
//
// RequestOnBehalfOf should only be used with api.twilio.com, not (for example)
// Twilio Monitor - the newer API's do not include the account sid in the URI.
//
// To authenticate using a subaccount sid / auth token, create a new Client
// using that account's credentials.
func (c *Client) RequestOnBehalfOf(subaccountSid string) {
	c.FullPath = func(pathPart string) string {
		return "/" + strings.Join([]string{c.APIVersion, "Accounts", subaccountSid, pathPart + ".json"}, "/")
	}
}

// UseSecretKey will use the provided secret key to authenticate to the API
// (instead of the AccountSid).
//
// For more information about secret keys, see
// https://www.twilio.com/docs/api/rest/keys.
func (c *Client) UseSecretKey(key string) {
	c.Client.ID = key
	if c.Monitor != nil {
		c.Monitor.UseSecretKey(key)
	}
	if c.Pricing != nil {
		c.Pricing.UseSecretKey(key)
	}
	if c.Fax != nil {
		c.Fax.UseSecretKey(key)
	}
	if c.Wireless != nil {
		c.Wireless.UseSecretKey(key)
	}
	if c.Insights != nil {
		c.Insights.UseSecretKey(key)
	}
}

// GetResource retrieves an instance resource with the given path part (e.g.
// "/Messages") and sid (e.g. "MM123").
func (c *Client) GetResource(ctx context.Context, pathPart string, sid string, v interface{}) error {
	sidPart := strings.Join([]string{pathPart, sid}, "/")
	return c.MakeRequest(ctx, "GET", sidPart, nil, v)
}

// CreateResource makes a POST request to the given resource.
func (c *Client) CreateResource(ctx context.Context, pathPart string, data url.Values, v interface{}) error {
	return c.MakeRequest(ctx, "POST", pathPart, data, v)
}

func (c *Client) UpdateResource(ctx context.Context, pathPart string, sid string, data url.Values, v interface{}) error {
	sidPart := strings.Join([]string{pathPart, sid}, "/")
	return c.MakeRequest(ctx, "POST", sidPart, data, v)
}

func (c *Client) DeleteResource(ctx context.Context, pathPart string, sid string) error {
	sidPart := strings.Join([]string{pathPart, sid}, "/")
	err := c.MakeRequest(ctx, "DELETE", sidPart, nil, nil)
	if err == nil {
		return nil
	}
	rerr, ok := err.(*resterror.Error)
	if ok && rerr.Status == http.StatusNotFound {
		return nil
	}
	return err
}

func (c *Client) ListResource(ctx context.Context, pathPart string, data url.Values, v interface{}) error {
	return c.MakeRequest(ctx, "GET", pathPart, data, v)
}

// GetNextPage fetches the Page at fullUri and decodes it into v. fullUri
// should be a next_page_uri returned in the response to a paging request, and
// should be the full path, eg "/2010-04-01/.../Messages?Page=1&PageToken=..."
func (c *Client) GetNextPage(ctx context.Context, fullUri string, v interface{}) error {
	// for monitor etc.
	fullUri = strings.TrimPrefix(fullUri, c.Base)
	return c.MakeRequest(ctx, "GET", fullUri, nil, v)
}

// Make a request to the Twilio API.
func (c *Client) MakeRequest(ctx context.Context, method string, pathPart string, data url.Values, v interface{}) error {
	if !strings.HasPrefix(pathPart, "/"+c.APIVersion) {
		pathPart = c.FullPath(pathPart)
	}
	rb := new(strings.Reader)
	if data != nil && (method == "POST" || method == "PUT") {
		rb = strings.NewReader(data.Encode())
	}
	if method == "GET" && data != nil {
		pathPart = pathPart + "?" + data.Encode()
	}
	req, err := c.NewRequest(method, pathPart, rb)
	if err != nil {
		return err
	}
	req = withContext(req, ctx)
	if ua := req.Header.Get("User-Agent"); ua == "" {
		req.Header.Set("User-Agent", userAgent)
	} else {
		req.Header.Set("User-Agent", userAgent+" "+ua)
	}
	return c.Do(req, &v)
}
