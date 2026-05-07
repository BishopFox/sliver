package twitter

import (
	"net/http"

	"github.com/dghubble/sling"
)

// TimelineService provides methods for accessing Twitter status timeline
// API endpoints.
type TimelineService struct {
	sling *sling.Sling
}

// newTimelineService returns a new TimelineService.
func newTimelineService(sling *sling.Sling) *TimelineService {
	return &TimelineService{
		sling: sling.Path("statuses/"),
	}
}

// UserTimelineParams are the parameters for TimelineService.UserTimeline.
type UserTimelineParams struct {
	UserID          int64  `url:"user_id,omitempty"`
	ScreenName      string `url:"screen_name,omitempty"`
	Count           int    `url:"count,omitempty"`
	SinceID         int64  `url:"since_id,omitempty"`
	MaxID           int64  `url:"max_id,omitempty"`
	TrimUser        *bool  `url:"trim_user,omitempty"`
	ExcludeReplies  *bool  `url:"exclude_replies,omitempty"`
	IncludeRetweets *bool  `url:"include_rts,omitempty"`
	TweetMode       string `url:"tweet_mode,omitempty"`
}

// UserTimeline returns recent Tweets from the specified user.
// https://dev.twitter.com/rest/reference/get/statuses/user_timeline
func (s *TimelineService) UserTimeline(params *UserTimelineParams) ([]Tweet, *http.Response, error) {
	tweets := new([]Tweet)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("user_timeline.json").QueryStruct(params).Receive(tweets, apiError)
	return *tweets, resp, relevantError(err, *apiError)
}

// HomeTimelineParams are the parameters for TimelineService.HomeTimeline.
type HomeTimelineParams struct {
	Count              int    `url:"count,omitempty"`
	SinceID            int64  `url:"since_id,omitempty"`
	MaxID              int64  `url:"max_id,omitempty"`
	TrimUser           *bool  `url:"trim_user,omitempty"`
	ExcludeReplies     *bool  `url:"exclude_replies,omitempty"`
	ContributorDetails *bool  `url:"contributor_details,omitempty"`
	IncludeEntities    *bool  `url:"include_entities,omitempty"`
	TweetMode          string `url:"tweet_mode,omitempty"`
}

// HomeTimeline returns recent Tweets and retweets from the user and those
// users they follow.
// Requires a user auth context.
// https://dev.twitter.com/rest/reference/get/statuses/home_timeline
func (s *TimelineService) HomeTimeline(params *HomeTimelineParams) ([]Tweet, *http.Response, error) {
	tweets := new([]Tweet)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("home_timeline.json").QueryStruct(params).Receive(tweets, apiError)
	return *tweets, resp, relevantError(err, *apiError)
}

// MentionTimelineParams are the parameters for TimelineService.MentionTimeline.
type MentionTimelineParams struct {
	Count              int    `url:"count,omitempty"`
	SinceID            int64  `url:"since_id,omitempty"`
	MaxID              int64  `url:"max_id,omitempty"`
	TrimUser           *bool  `url:"trim_user,omitempty"`
	ContributorDetails *bool  `url:"contributor_details,omitempty"`
	IncludeEntities    *bool  `url:"include_entities,omitempty"`
	TweetMode          string `url:"tweet_mode,omitempty"`
}

// MentionTimeline returns recent Tweet mentions of the authenticated user.
// Requires a user auth context.
// https://dev.twitter.com/rest/reference/get/statuses/mentions_timeline
func (s *TimelineService) MentionTimeline(params *MentionTimelineParams) ([]Tweet, *http.Response, error) {
	tweets := new([]Tweet)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("mentions_timeline.json").QueryStruct(params).Receive(tweets, apiError)
	return *tweets, resp, relevantError(err, *apiError)
}

// RetweetsOfMeTimelineParams are the parameters for
// TimelineService.RetweetsOfMeTimeline.
type RetweetsOfMeTimelineParams struct {
	Count               int    `url:"count,omitempty"`
	SinceID             int64  `url:"since_id,omitempty"`
	MaxID               int64  `url:"max_id,omitempty"`
	TrimUser            *bool  `url:"trim_user,omitempty"`
	IncludeEntities     *bool  `url:"include_entities,omitempty"`
	IncludeUserEntities *bool  `url:"include_user_entities"`
	TweetMode           string `url:"tweet_mode,omitempty"`
}

// RetweetsOfMeTimeline returns the most recent Tweets by the authenticated
// user that have been retweeted by others.
// Requires a user auth context.
// https://dev.twitter.com/rest/reference/get/statuses/retweets_of_me
func (s *TimelineService) RetweetsOfMeTimeline(params *RetweetsOfMeTimelineParams) ([]Tweet, *http.Response, error) {
	tweets := new([]Tweet)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("retweets_of_me.json").QueryStruct(params).Receive(tweets, apiError)
	return *tweets, resp, relevantError(err, *apiError)
}
