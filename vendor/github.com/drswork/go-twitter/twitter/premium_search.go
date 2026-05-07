package twitter

import (
	"fmt"
	"net/http"

	"github.com/dghubble/sling"
)

// PremiumSearch represents the result of a Tweet search.
// https://developer.twitter.com/en/docs/tweets/search/api-reference/premium-search
type PremiumSearch struct {
	Results           []Tweet            `json:"results"`
	Next              string             `json:"next"`
	RequestParameters *RequestParameters `json:"requestParameters"`
}

// RequestParameters describes a request parameter that was passed to a Premium search API.
type RequestParameters struct {
	MaxResults int    `json:"maxResults"`
	FromDate   string `json:"fromDate"`
	ToDate     string `json:"toDate"`
}

// PremiumSearchCount describes a response of Premium search API's count endpoint.
// https://developer.twitter.com/en/docs/tweets/search/api-reference/premium-search#CountsEndpoint
type PremiumSearchCount struct {
	Results           []TweetCount            `json:"results"`
	TotalCount        int64                   `json:"totalCount"`
	RequestParameters *RequestCountParameters `json:"requestParameters"`
}

// RequestCountParameters describes a request parameter that was passed to a Premium search API.
type RequestCountParameters struct {
	Bucket   string `json:"bucket"`
	FromDate string `json:"fromDate"`
	ToDate   string `json:"toDate"`
}

// TweetCount represents a count of Tweets in the TimePeriod matching a search query.
type TweetCount struct {
	TimePeriod string `json:"timePeriod"`
	Count      int64  `json:"count"`
}

// PremiumSearchService provides methods for accessing Twitter premium search API endpoints.
type PremiumSearchService struct {
	sling *sling.Sling
}

// newSearchService returns a new SearchService.
func newPremiumSearchService(sling *sling.Sling) *PremiumSearchService {
	return &PremiumSearchService{
		sling: sling.Path("tweets/search/"),
	}
}

// PremiumSearchTweetParams are the parameters for PremiumSearchService.SearchFullArchive and Search30Days
type PremiumSearchTweetParams struct {
	Query      string `url:"query,omitempty"`
	Tag        string `url:"tag,omitempty"`
	FromDate   string `url:"fromDate,omitempty"`
	ToDate     string `url:"toDate,omitempty"`
	MaxResults int    `url:"maxResults,omitempty"`
	Next       string `url:"next,omitempty"`
}

// PremiumSearchCountTweetParams are the parameters for PremiumSearchService.CountFullArchive and Count30Days
type PremiumSearchCountTweetParams struct {
	Query    string `url:"query,omitempty"`
	Tag      string `url:"tag,omitempty"`
	FromDate string `url:"fromDate,omitempty"`
	ToDate   string `url:"toDate,omitempty"`
	Bucket   string `url:"bucket,omitempty"`
	Next     string `url:"next,omitempty"`
}

// SearchFullArchive returns a collection of Tweets matching a search query from tweets back to the very first tweet.
// https://developer.twitter.com/en/docs/tweets/search/api-reference/premium-search
func (s *PremiumSearchService) SearchFullArchive(params *PremiumSearchTweetParams, label string) (*PremiumSearch, *http.Response, error) {
	search := new(PremiumSearch)
	apiError := new(APIError)
	path := fmt.Sprintf("fullarchive/%s.json", label)
	resp, err := s.sling.New().Get(path).QueryStruct(params).Receive(search, apiError)
	return search, resp, relevantError(err, *apiError)
}

// Search30Days returns a collection of Tweets matching a search query from Tweets posted within the last 30 days.
// https://developer.twitter.com/en/docs/tweets/search/api-reference/premium-search
func (s *PremiumSearchService) Search30Days(params *PremiumSearchTweetParams, label string) (*PremiumSearch, *http.Response, error) {
	search := new(PremiumSearch)
	apiError := new(APIError)
	path := fmt.Sprintf("30day/%s.json", label)
	resp, err := s.sling.New().Get(path).QueryStruct(params).Receive(search, apiError)
	return search, resp, relevantError(err, *apiError)
}

// CountFullArchive returns a counts of Tweets matching a search query from tweets back to the very first tweet.
// https://developer.twitter.com/en/docs/tweets/search/api-reference/premium-search#CountsEndpoint
func (s *PremiumSearchService) CountFullArchive(params *PremiumSearchCountTweetParams, label string) (*PremiumSearchCount, *http.Response, error) {
	counts := new(PremiumSearchCount)
	apiError := new(APIError)
	path := fmt.Sprintf("fullarchive/%s/counts.json", label)
	resp, err := s.sling.New().Get(path).QueryStruct(params).Receive(counts, apiError)
	return counts, resp, relevantError(err, *apiError)
}

// Count30Days returns a counts of Tweets matching a search query from Tweets posted within the last 30 days.
// https://developer.twitter.com/en/docs/tweets/search/api-reference/premium-search#CountsEndpoint
func (s *PremiumSearchService) Count30Days(params *PremiumSearchCountTweetParams, label string) (*PremiumSearchCount, *http.Response, error) {
	counts := new(PremiumSearchCount)
	apiError := new(APIError)
	path := fmt.Sprintf("30day/%s/counts.json", label)
	resp, err := s.sling.New().Get(path).QueryStruct(params).Receive(counts, apiError)
	return counts, resp, relevantError(err, *apiError)
}
