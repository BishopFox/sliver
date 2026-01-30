package plivo

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"time"
)

const baseUrlString = "https://api.plivo.com/"

var HttpsScheme = "https"
var voiceBaseUrlString = "api.plivo.com"
var voiceBaseUrlStringFallback1 = "api.plivo.com"
var voiceBaseUrlStringFallback2 = "api.plivo.com"

const baseRequestString = "/v1/Account/%s/"

type Client struct {
	BaseClient
	Messages                    *MessageService
	Accounts                    *AccountService
	Subaccounts                 *SubaccountService
	Applications                *ApplicationService
	Endpoints                   *EndpointService
	Numbers                     *NumberService
	PhoneNumbers                *PhoneNumberService
	Pricing                     *PricingService // TODO Rename?
	Recordings                  *RecordingService
	Transcription               *TranscriptionService
	Calls                       *CallService
	Token                       *TokenService
	LiveCalls                   *LiveCallService
	QueuedCalls                 *QueuedCallService
	Conferences                 *ConferenceService
	CallFeedback                *CallFeedbackService
	Powerpack                   *PowerpackService
	Media                       *MediaService
	Lookup                      *LookupService
	EndUsers                    *EndUserService
	ComplianceDocuments         *ComplianceDocumentService
	ComplianceDocumentTypes     *ComplianceDocumentTypeService
	ComplianceRequirements      *ComplianceRequirementService
	ComplianceApplications      *ComplianceApplicationService
	MultiPartyCall              *MultiPartyCallService
	Brand                       *BrandService
	Profile                     *ProfileService
	Campaign                    *CampaignService
	MaskingSession              *MaskingSessionService
	VerifySession               *VerifyService
	VerifyCallerId              *VerifyCallerIdService
	TollFreeRequestVerification *TollfreeVerificationService
}

/*
To set a proxy for all requests, configure the Transport for the HttpClient passed in:

		&http.Client{
	 		Transport: &http.Transport{
	 			Proxy: http.ProxyURL("http//your.proxy.here"),
	 		},
	 	}

Similarly, to configure the timeout, set it on the HttpClient passed in:

		&http.Client{
	 		Timeout: time.Minute,
	 	}
*/
func NewClient(authId, authToken string, options *ClientOptions) (client *Client, err error) {

	client = &Client{}

	if len(authId) == 0 {
		authId = os.Getenv("PLIVO_AUTH_ID")
	}
	if len(authToken) == 0 {
		authToken = os.Getenv("PLIVO_AUTH_TOKEN")
	}
	client.AuthId = authId
	client.AuthToken = authToken
	client.userAgent = fmt.Sprintf("%s/%s (Go: %s)", "plivo-go", sdkVersion, runtime.Version())

	baseUrl, err := url.Parse(baseUrlString) // Todo: handle error case?

	client.BaseUrl = baseUrl
	client.httpClient = &http.Client{
		Timeout: time.Second * 10,
	}

	if options.HttpClient != nil {
		client.httpClient = options.HttpClient
	}

	client.Messages = &MessageService{client: client}
	client.Accounts = &AccountService{client: client}
	client.Subaccounts = &SubaccountService{client: client}
	client.Applications = &ApplicationService{client: client}
	client.Endpoints = &EndpointService{client: client}
	client.Numbers = &NumberService{client: client}
	client.PhoneNumbers = &PhoneNumberService{client: client}
	client.Pricing = &PricingService{client: client}
	client.Recordings = &RecordingService{client: client}
	client.Transcription = &TranscriptionService{client: client}
	client.Calls = &CallService{client: client}
	client.Token = &TokenService{client: client}
	client.LiveCalls = &LiveCallService{client: client}
	client.QueuedCalls = &QueuedCallService{client: client}
	client.Conferences = &ConferenceService{client: client}
	client.CallFeedback = &CallFeedbackService{client: client}
	client.Powerpack = &PowerpackService{client: client}
	client.Media = &MediaService{client: client}
	client.EndUsers = &EndUserService{client: client}
	client.ComplianceDocuments = &ComplianceDocumentService{client: client}
	client.ComplianceDocumentTypes = &ComplianceDocumentTypeService{client: client}
	client.ComplianceRequirements = &ComplianceRequirementService{client: client}
	client.ComplianceApplications = &ComplianceApplicationService{client: client}
	client.Lookup = &LookupService{client: client}
	client.MultiPartyCall = &MultiPartyCallService{client: client}
	client.Brand = &BrandService{client: client}
	client.Campaign = &CampaignService{client: client}
	client.Profile = &ProfileService{client: client}
	client.MaskingSession = &MaskingSessionService{client: client}
	client.VerifySession = &VerifyService{client: client}
	client.TollFreeRequestVerification = &TollfreeVerificationService{client: client}
	client.VerifyCallerId = &VerifyCallerIdService{client: client}
	return
}

func (client *Client) NewRequest(method string, params interface{}, formatString string,
	formatParams ...interface{}) (*http.Request, error) {
	formatParams = append([]interface{}{client.AuthId}, formatParams...)
	formatString = fmt.Sprintf("%s/%s", "%s", formatString)
	return client.BaseClient.NewRequest(method, params, baseRequestString, formatString, formatParams...)
}
