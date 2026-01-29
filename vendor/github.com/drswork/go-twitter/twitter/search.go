package twitter

import (
	"net/http"

	"github.com/dghubble/sling"
)

// Search represents the result of a Tweet search.
type Search struct {
	Statuses []Tweet         `json:"statuses"`
	Metadata *SearchMetadata `json:"search_metadata"`
}

// SearchMetadata describes a Search result.
type SearchMetadata struct {
	Count       int     `json:"count"`
	SinceID     int64   `json:"since_id"`
	SinceIDStr  string  `json:"since_id_str"`
	MaxID       int64   `json:"max_id"`
	MaxIDStr    string  `json:"max_id_str"`
	RefreshURL  string  `json:"refresh_url"`
	NextResults string  `json:"next_results"`
	CompletedIn float64 `json:"completed_in"`
	Query       string  `json:"query"`
}

// SearchService provides methods for accessing Twitter search API endpoints.
type SearchService struct {
	sling *sling.Sling
}

// newSearchService returns a new SearchService.
func newSearchService(sling *sling.Sling) *SearchService {
	return &SearchService{
		sling: sling.Path("search/"),
	}
}

// SearchTweetParams are the parameters for SearchService.Tweets
type SearchTweetParams struct {
	Query           string `url:"q,omitempty"`
	Geocode         string `url:"geocode,omitempty"`
	Lang            string `url:"lang,omitempty"`
	Locale          string `url:"locale,omitempty"`
	ResultType      string `url:"result_type,omitempty"`
	Count           int    `url:"count,omitempty"`
	SinceID         int64  `url:"since_id,omitempty"`
	MaxID           int64  `url:"max_id,omitempty"`
	Until           string `url:"until,omitempty"`
	Since           string `url:"since,omitempty"`
	Filter          string `url:"filter,omitempty"`
	IncludeEntities *bool  `url:"include_entities,omitempty"`
	TweetMode       string `url:"tweet_mode,omitempty"`
}

// Tweets returns a collection of Tweets matching a search query.
// https://dev.twitter.com/rest/reference/get/search/tweets
func (s *SearchService) Tweets(params *SearchTweetParams) (*Search, *http.Response, error) {
	search := new(Search)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("tweets.json").QueryStruct(params).Receive(search, apiError)
	return search, resp, relevantError(err, *apiError)
}
