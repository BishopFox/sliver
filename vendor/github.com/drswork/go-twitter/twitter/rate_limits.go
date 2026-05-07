package twitter

import (
	"net/http"

	"github.com/dghubble/sling"
)

// RateLimitService provides methods for accessing Twitter rate limits
// API endpoints.
type RateLimitService struct {
	sling *sling.Sling
}

// newRateLimitService returns a new RateLimitService.
func newRateLimitService(sling *sling.Sling) *RateLimitService {
	return &RateLimitService{
		sling: sling.Path("application/"),
	}
}

// RateLimit summarizes current rate limits of resource families.
type RateLimit struct {
	RateLimitContext *RateLimitContext   `json:"rate_limit_context"`
	Resources        *RateLimitResources `json:"resources"`
}

// RateLimitContext contains auth context
type RateLimitContext struct {
	AccessToken string `json:"access_token"`
}

// RateLimitResources contains all limit status data for endpoints group by resources
type RateLimitResources struct {
	Application map[string]*RateLimitResource `json:"application"`
	Favorites   map[string]*RateLimitResource `json:"favorites"`
	Followers   map[string]*RateLimitResource `json:"followers"`
	Friends     map[string]*RateLimitResource `json:"friends"`
	Friendships map[string]*RateLimitResource `json:"friendships"`
	Geo         map[string]*RateLimitResource `json:"geo"`
	Help        map[string]*RateLimitResource `json:"help"`
	Lists       map[string]*RateLimitResource `json:"lists"`
	Search      map[string]*RateLimitResource `json:"search"`
	Statuses    map[string]*RateLimitResource `json:"statuses"`
	Trends      map[string]*RateLimitResource `json:"trends"`
	Users       map[string]*RateLimitResource `json:"users"`
}

// RateLimitResource contains limit status data for a single endpoint
type RateLimitResource struct {
	Limit     int `json:"limit"`
	Remaining int `json:"remaining"`
	Reset     int `json:"reset"`
}

// RateLimitParams are the parameters for RateLimitService.Status.
type RateLimitParams struct {
	Resources []string `url:"resources,omitempty,comma"`
}

// Status summarizes the current rate limits of specified resource families.
// https://developer.twitter.com/en/docs/developer-utilities/rate-limit-status/api-reference/get-application-rate_limit_status
func (s *RateLimitService) Status(params *RateLimitParams) (*RateLimit, *http.Response, error) {
	rateLimit := new(RateLimit)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("rate_limit_status.json").QueryStruct(params).Receive(rateLimit, apiError)
	return rateLimit, resp, relevantError(err, *apiError)
}
