package twitter

import (
	"net/http"

	"github.com/dghubble/sling"
)

// TrendsService provides methods for accessing Twitter trends API endpoints.
type TrendsService struct {
	sling *sling.Sling
}

// newTrendsService returns a new TrendsService.
func newTrendsService(sling *sling.Sling) *TrendsService {
	return &TrendsService{
		sling: sling.Path("trends/"),
	}
}

// PlaceType represents a twitter trends PlaceType.
type PlaceType struct {
	Code int    `json:"code"`
	Name string `json:"name"`
}

// Location represents a twitter Location.
type Location struct {
	Country     string    `json:"country"`
	CountryCode string    `json:"countryCode"`
	Name        string    `json:"name"`
	ParentID    int       `json:"parentid"`
	PlaceType   PlaceType `json:"placeType"`
	URL         string    `json:"url"`
	WOEID       int64     `json:"woeid"`
}

// Available returns the locations that Twitter has trending topic information for.
// https://dev.twitter.com/rest/reference/get/trends/available
func (s *TrendsService) Available() ([]Location, *http.Response, error) {
	locations := new([]Location)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("available.json").Receive(locations, apiError)
	return *locations, resp, relevantError(err, *apiError)
}

// Trend represents a twitter trend.
type Trend struct {
	Name            string `json:"name"`
	URL             string `json:"url"`
	PromotedContent string `json:"promoted_content"`
	Query           string `json:"query"`
	TweetVolume     int64  `json:"tweet_volume"`
}

// TrendsList represents a list of twitter trends.
type TrendsList struct {
	Trends    []Trend          `json:"trends"`
	AsOf      string           `json:"as_of"`
	CreatedAt string           `json:"created_at"`
	Locations []TrendsLocation `json:"locations"`
}

// TrendsLocation represents a twitter trend location.
type TrendsLocation struct {
	Name  string `json:"name"`
	WOEID int64  `json:"woeid"`
}

// TrendsPlaceParams are the parameters for Trends.Place.
type TrendsPlaceParams struct {
	WOEID   int64  `url:"id,omitempty"`
	Exclude string `url:"exclude,omitempty"`
}

// Place returns the top 50 trending topics for a specific WOEID.
// https://dev.twitter.com/rest/reference/get/trends/place
func (s *TrendsService) Place(woeid int64, params *TrendsPlaceParams) ([]TrendsList, *http.Response, error) {
	if params == nil {
		params = &TrendsPlaceParams{}
	}
	trendsList := new([]TrendsList)
	params.WOEID = woeid
	apiError := new(APIError)
	resp, err := s.sling.New().Get("place.json").QueryStruct(params).Receive(trendsList, apiError)
	return *trendsList, resp, relevantError(err, *apiError)
}

// ClosestParams are the parameters for Trends.Closest.
type ClosestParams struct {
	Lat  float64 `url:"lat"`
	Long float64 `url:"long"`
}

// Closest returns the locations that Twitter has trending topic information for, closest to a specified location.
// https://dev.twitter.com/rest/reference/get/trends/closest
func (s *TrendsService) Closest(params *ClosestParams) ([]Location, *http.Response, error) {
	locations := new([]Location)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("closest.json").QueryStruct(params).Receive(locations, apiError)
	return *locations, resp, relevantError(err, *apiError)
}
