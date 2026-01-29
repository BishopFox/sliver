package twitter

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dghubble/sling"
)

// Tweet represents a Twitter Tweet, previously called a status.
// https://dev.twitter.com/overview/api/tweets
type Tweet struct {
	Coordinates           *Coordinates           `json:"coordinates"`
	CreatedAt             string                 `json:"created_at"`
	CurrentUserRetweet    *TweetIdentifier       `json:"current_user_retweet"`
	Entities              *Entities              `json:"entities"`
	FavoriteCount         int                    `json:"favorite_count"`
	Favorited             bool                   `json:"favorited"`
	FilterLevel           string                 `json:"filter_level"`
	ID                    int64                  `json:"id"`
	IDStr                 string                 `json:"id_str"`
	InReplyToScreenName   string                 `json:"in_reply_to_screen_name"`
	InReplyToStatusID     int64                  `json:"in_reply_to_status_id"`
	InReplyToStatusIDStr  string                 `json:"in_reply_to_status_id_str"`
	InReplyToUserID       int64                  `json:"in_reply_to_user_id"`
	InReplyToUserIDStr    string                 `json:"in_reply_to_user_id_str"`
	Lang                  string                 `json:"lang"`
	PossiblySensitive     bool                   `json:"possibly_sensitive"`
	QuoteCount            int                    `json:"quote_count"`
	ReplyCount            int                    `json:"reply_count"`
	RetweetCount          int                    `json:"retweet_count"`
	Retweeted             bool                   `json:"retweeted"`
	RetweetedStatus       *Tweet                 `json:"retweeted_status"`
	Source                string                 `json:"source"`
	Scopes                map[string]interface{} `json:"scopes"`
	Text                  string                 `json:"text"`
	FullText              string                 `json:"full_text"`
	DisplayTextRange      Indices                `json:"display_text_range"`
	Place                 *Place                 `json:"place"`
	Truncated             bool                   `json:"truncated"`
	User                  *User                  `json:"user"`
	WithheldCopyright     bool                   `json:"withheld_copyright"`
	WithheldInCountries   []string               `json:"withheld_in_countries"`
	WithheldScope         string                 `json:"withheld_scope"`
	ExtendedEntities      *ExtendedEntity        `json:"extended_entities"`
	ExtendedTweet         *ExtendedTweet         `json:"extended_tweet"`
	QuotedStatusID        int64                  `json:"quoted_status_id"`
	QuotedStatusIDStr     string                 `json:"quoted_status_id_str"`
	QuotedStatus          *Tweet                 `json:"quoted_status"`
	IsQuoteStatus         bool                   `json:"is_quote_status"`
	MatchingRules         []Rule                 `json:"matching_rules"`
	QuotedStatusPermalink *QuotedStatusPermalink `json:"quoted_status_permalink"`
}

// Rule represents which rule matched a tweet.
type Rule struct {
	Tag   string `json:"tag"`
	ID    int64  `json:"id"`
	IDStr string `json:"id_str"`
}

// QuotedStatusPermalink holds a permalink for a particular status.
type QuotedStatusPermalink struct {
	URL         string `json:"url"`
	ExpandedURL string `json:"expanded"`
	DisplayURL  string `json:"display"`
}

// CreatedAtTime returns the time a tweet was created.
func (t Tweet) CreatedAtTime() (time.Time, error) {
	return time.Parse(time.RubyDate, t.CreatedAt)
}

// ExtendedTweet represents fields embedded in extended Tweets when served in
// compatibility mode (default).
// https://dev.twitter.com/overview/api/upcoming-changes-to-tweets
type ExtendedTweet struct {
	FullText         string          `json:"full_text"`
	DisplayTextRange Indices         `json:"display_text_range"`
	Entities         *Entities       `json:"entities"`
	ExtendedEntities *ExtendedEntity `json:"extended_entities"`
}

// Place represents a Twitter Place / Location
// https://dev.twitter.com/overview/api/places
type Place struct {
	Attributes  map[string]string `json:"attributes"`
	BoundingBox *BoundingBox      `json:"bounding_box"`
	Country     string            `json:"country"`
	CountryCode string            `json:"country_code"`
	FullName    string            `json:"full_name"`
	Geometry    *BoundingBox      `json:"geometry"`
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	PlaceType   string            `json:"place_type"`
	Polylines   []string          `json:"polylines"`
	URL         string            `json:"url"`
}

// BoundingBox represents the bounding coordinates (longitude, latitutde)
// defining the bounds of a box containing a Place entity.
type BoundingBox struct {
	Coordinates [][][2]float64 `json:"coordinates"`
	Type        string         `json:"type"`
}

// Coordinates are pairs of longitude and latitude locations.
type Coordinates struct {
	Coordinates [2]float64 `json:"coordinates"`
	Type        string     `json:"type"`
}

// TweetIdentifier represents the id by which a Tweet can be identified.
type TweetIdentifier struct {
	ID    int64  `json:"id"`
	IDStr string `json:"id_str"`
}

// StatusService provides methods for accessing Twitter status API endpoints.
type StatusService struct {
	sling *sling.Sling
}

// newStatusService returns a new StatusService.
func newStatusService(sling *sling.Sling) *StatusService {
	return &StatusService{
		sling: sling.Path("statuses/"),
	}
}

// StatusShowParams are the parameters for StatusService.Show
type StatusShowParams struct {
	ID               int64  `url:"id,omitempty"`
	TrimUser         *bool  `url:"trim_user,omitempty"`
	IncludeMyRetweet *bool  `url:"include_my_retweet,omitempty"`
	IncludeEntities  *bool  `url:"include_entities,omitempty"`
	TweetMode        string `url:"tweet_mode,omitempty"`
}

// Show returns the requested Tweet.
// https://dev.twitter.com/rest/reference/get/statuses/show/%3Aid
func (s *StatusService) Show(id int64, params *StatusShowParams) (*Tweet, *http.Response, error) {
	if params == nil {
		params = &StatusShowParams{}
	}
	params.ID = id
	tweet := new(Tweet)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("show.json").QueryStruct(params).Receive(tweet, apiError)
	return tweet, resp, relevantError(err, *apiError)
}

// StatusLookupParams are the parameters for StatusService.Lookup
type StatusLookupParams struct {
	ID              []int64 `url:"id,omitempty,comma"`
	TrimUser        *bool   `url:"trim_user,omitempty"`
	IncludeEntities *bool   `url:"include_entities,omitempty"`
	Map             *bool   `url:"map,omitempty"`
	TweetMode       string  `url:"tweet_mode,omitempty"`
}

// Lookup returns the requested Tweets as a slice. Combines ids from the
// required ids argument and from params.Id.
// https://dev.twitter.com/rest/reference/get/statuses/lookup
func (s *StatusService) Lookup(ids []int64, params *StatusLookupParams) ([]Tweet, *http.Response, error) {
	if params == nil {
		params = &StatusLookupParams{}
	}
	params.ID = append(params.ID, ids...)
	tweets := new([]Tweet)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("lookup.json").QueryStruct(params).Receive(tweets, apiError)
	return *tweets, resp, relevantError(err, *apiError)
}

// StatusUpdateParams are the parameters for StatusService.Update
type StatusUpdateParams struct {
	Status                    string   `url:"status,omitempty"`
	InReplyToStatusID         int64    `url:"in_reply_to_status_id,omitempty"`
	PossiblySensitive         *bool    `url:"possibly_sensitive,omitempty"`
	Lat                       *float64 `url:"lat,omitempty"`
	Long                      *float64 `url:"long,omitempty"`
	PlaceID                   string   `url:"place_id,omitempty"`
	DisplayCoordinates        *bool    `url:"display_coordinates,omitempty"`
	TrimUser                  *bool    `url:"trim_user,omitempty"`
	MediaIds                  []int64  `url:"media_ids,omitempty,comma"`
	TweetMode                 string   `url:"-"`
	AutoPopulateReplyMetadata *bool    `url:"auto_populate_reply_metadata,omitempty"`
	ExcludeReplyUserIDs       []int64  `url:"exclude_reply_user_ids,omitempty,comma"`
	AttachmentURL             string   `url:"attachment_url,omitempty"`
	CardURI                   string   `url:"card_uri,omitempty"`
}

// Update updates the user's status, also known as Tweeting.
// Requires a user auth context.
// https://dev.twitter.com/rest/reference/post/statuses/update
func (s *StatusService) Update(status string, params *StatusUpdateParams) (*Tweet, *http.Response, error) {
	if params == nil {
		params = &StatusUpdateParams{}
	}
	params.Status = status
	tweet := new(Tweet)
	apiError := new(APIError)
	resp, err := s.sling.New().Post("update.json").BodyForm(params).Receive(tweet, apiError)
	return tweet, resp, relevantError(err, *apiError)
}

// StatusRetweetParams are the parameters for StatusService.Retweet
type StatusRetweetParams struct {
	ID        int64  `url:"id,omitempty"`
	TrimUser  *bool  `url:"trim_user,omitempty"`
	TweetMode string `url:"tweet_mode,omitempty"`
}

// Retweet retweets the Tweet with the given id and returns the original Tweet
// with embedded retweet details.
// Requires a user auth context.
// https://dev.twitter.com/rest/reference/post/statuses/retweet/%3Aid
func (s *StatusService) Retweet(id int64, params *StatusRetweetParams) (*Tweet, *http.Response, error) {
	if params == nil {
		params = &StatusRetweetParams{}
	}
	params.ID = id
	tweet := new(Tweet)
	apiError := new(APIError)
	path := fmt.Sprintf("retweet/%d.json", params.ID)
	resp, err := s.sling.New().Post(path).BodyForm(params).Receive(tweet, apiError)
	return tweet, resp, relevantError(err, *apiError)
}

// StatusUnretweetParams are the parameters for StatusService.Unretweet
type StatusUnretweetParams struct {
	ID        int64  `url:"id,omitempty"`
	TrimUser  *bool  `url:"trim_user,omitempty"`
	TweetMode string `url:"tweet_mode,omitempty"`
}

// Unretweet unretweets the Tweet with the given id and returns the original Tweet.
// Requires a user auth context.
// https://dev.twitter.com/rest/reference/post/statuses/unretweet/%3Aid
func (s *StatusService) Unretweet(id int64, params *StatusUnretweetParams) (*Tweet, *http.Response, error) {
	if params == nil {
		params = &StatusUnretweetParams{}
	}
	params.ID = id
	tweet := new(Tweet)
	apiError := new(APIError)
	path := fmt.Sprintf("unretweet/%d.json", params.ID)
	resp, err := s.sling.New().Post(path).BodyForm(params).Receive(tweet, apiError)
	return tweet, resp, relevantError(err, *apiError)
}

// StatusRetweetsParams are the parameters for StatusService.Retweets
type StatusRetweetsParams struct {
	ID        int64  `url:"id,omitempty"`
	Count     int    `url:"count,omitempty"`
	TrimUser  *bool  `url:"trim_user,omitempty"`
	TweetMode string `url:"tweet_mode,omitempty"`
}

// Retweets returns the most recent retweets of the Tweet with the given id.
// https://dev.twitter.com/rest/reference/get/statuses/retweets/%3Aid
func (s *StatusService) Retweets(id int64, params *StatusRetweetsParams) ([]Tweet, *http.Response, error) {
	if params == nil {
		params = &StatusRetweetsParams{}
	}
	params.ID = id
	tweets := new([]Tweet)
	apiError := new(APIError)
	path := fmt.Sprintf("retweets/%d.json", params.ID)
	resp, err := s.sling.New().Get(path).QueryStruct(params).Receive(tweets, apiError)
	return *tweets, resp, relevantError(err, *apiError)
}

// StatusDestroyParams are the parameters for StatusService.Destroy
type StatusDestroyParams struct {
	ID        int64  `url:"id,omitempty"`
	TrimUser  *bool  `url:"trim_user,omitempty"`
	TweetMode string `url:"tweet_mode,omitempty"`
}

// Destroy deletes the Tweet with the given id and returns it if successful.
// Requires a user auth context.
// https://dev.twitter.com/rest/reference/post/statuses/destroy/%3Aid
func (s *StatusService) Destroy(id int64, params *StatusDestroyParams) (*Tweet, *http.Response, error) {
	if params == nil {
		params = &StatusDestroyParams{}
	}
	params.ID = id
	tweet := new(Tweet)
	apiError := new(APIError)
	path := fmt.Sprintf("destroy/%d.json", params.ID)
	resp, err := s.sling.New().Post(path).BodyForm(params).Receive(tweet, apiError)
	return tweet, resp, relevantError(err, *apiError)
}

// OEmbedTweet represents a Tweet in oEmbed format.
type OEmbedTweet struct {
	URL          string `json:"url"`
	ProviderURL  string `json:"provider_url"`
	ProviderName string `json:"provider_name"`
	AuthorName   string `json:"author_name"`
	Version      string `json:"version"`
	AuthorURL    string `json:"author_url"`
	Type         string `json:"type"`
	HTML         string `json:"html"`
	Height       int64  `json:"height"`
	Width        int64  `json:"width"`
	CacheAge     string `json:"cache_age"`
}

// StatusOEmbedParams are the parameters for StatusService.OEmbed
type StatusOEmbedParams struct {
	ID         int64  `url:"id,omitempty"`
	URL        string `url:"url,omitempty"`
	Align      string `url:"align,omitempty"`
	MaxWidth   int64  `url:"maxwidth,omitempty"`
	HideMedia  *bool  `url:"hide_media,omitempty"`
	HideThread *bool  `url:"hide_thread,omitempty"`
	OmitScript *bool  `url:"omit_script,omitempty"`
	WidgetType string `url:"widget_type,omitempty"`
	HideTweet  *bool  `url:"hide_tweet,omitempty"`
}

// OEmbed returns the requested Tweet in oEmbed format.
// https://dev.twitter.com/rest/reference/get/statuses/oembed
func (s *StatusService) OEmbed(params *StatusOEmbedParams) (*OEmbedTweet, *http.Response, error) {
	oEmbedTweet := new(OEmbedTweet)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("oembed.json").QueryStruct(params).Receive(oEmbedTweet, apiError)
	return oEmbedTweet, resp, relevantError(err, *apiError)
}

// StatusRetweeterParams are the parameters for StatusService.Retweeters.
type StatusRetweeterParams struct {
	ID           int64  `url:"id,omitempty"`
	Cursor       int64  `url:"cursor,omitempty"`
	Count        int    `url:"count,omitempty"`
	StringifyIDs string `url:"stringify_ids,omitempty"`
}

// Retweeter represents a Tweet in oEmbed format.
type Retweeter struct {
	PreviousCursor    int64    `json:"previous_cursor"`
	PreviousCursorStr string   `json:"previous_cursor_str"`
	IDs               []string `json:"ids"`
	NextCursor        int64    `json:"next_cursor"`
	NextCursorStr     string   `json:"next_cursor_str"`
}

// Retweeters return the retweeters of a specific tweet.
// https://developer.twitter.com/en/docs/tweets/post-and-engage/api-reference/get-statuses-retweeters-ids
func (s *StatusService) Retweeters(params *StatusRetweeterParams) (*Retweeter, *http.Response, error) {
	retweeters := new(Retweeter)
	apiError := new(APIError)
	resp, err := s.sling.Get("retweeter/ids.json").QueryStruct(params).Receive(retweeters, apiError)
	return retweeters, resp, relevantError(err, *apiError)
}
