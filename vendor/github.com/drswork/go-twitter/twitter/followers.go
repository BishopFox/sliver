package twitter

import (
	"net/http"

	"github.com/dghubble/sling"
)

// FollowerIDs is a cursored collection of follower ids.
type FollowerIDs struct {
	IDs               []int64 `json:"ids"`
	NextCursor        int64   `json:"next_cursor"`
	NextCursorStr     string  `json:"next_cursor_str"`
	PreviousCursor    int64   `json:"previous_cursor"`
	PreviousCursorStr string  `json:"previous_cursor_str"`
}

// Followers is a cursored collection of followers.
type Followers struct {
	Users             []User `json:"users"`
	NextCursor        int64  `json:"next_cursor"`
	NextCursorStr     string `json:"next_cursor_str"`
	PreviousCursor    int64  `json:"previous_cursor"`
	PreviousCursorStr string `json:"previous_cursor_str"`
}

// FollowerService provides methods for accessing Twitter followers endpoints.
type FollowerService struct {
	sling *sling.Sling
}

// newFollowerService returns a new FollowerService.
func newFollowerService(sling *sling.Sling) *FollowerService {
	return &FollowerService{
		sling: sling.Path("followers/"),
	}
}

// FollowerIDParams are the parameters for FollowerService.Ids
type FollowerIDParams struct {
	UserID     int64  `url:"user_id,omitempty"`
	ScreenName string `url:"screen_name,omitempty"`
	Cursor     int64  `url:"cursor,omitempty"`
	Count      int    `url:"count,omitempty"`
}

// IDs returns a cursored collection of user ids following the specified user.
// https://dev.twitter.com/rest/reference/get/followers/ids
func (s *FollowerService) IDs(params *FollowerIDParams) (*FollowerIDs, *http.Response, error) {
	ids := new(FollowerIDs)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("ids.json").QueryStruct(params).Receive(ids, apiError)
	return ids, resp, relevantError(err, *apiError)
}

// FollowerListParams are the parameters for FollowerService.List
type FollowerListParams struct {
	UserID              int64  `url:"user_id,omitempty"`
	ScreenName          string `url:"screen_name,omitempty"`
	Cursor              int64  `url:"cursor,omitempty"`
	Count               int    `url:"count,omitempty"`
	SkipStatus          *bool  `url:"skip_status,omitempty"`
	IncludeUserEntities *bool  `url:"include_user_entities,omitempty"`
}

// List returns a cursored collection of Users following the specified user.
// https://dev.twitter.com/rest/reference/get/followers/list
func (s *FollowerService) List(params *FollowerListParams) (*Followers, *http.Response, error) {
	followers := new(Followers)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("list.json").QueryStruct(params).Receive(followers, apiError)
	return followers, resp, relevantError(err, *apiError)
}
