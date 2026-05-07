package twitter

import (
	"net/http"

	"github.com/dghubble/sling"
)

// FriendIDs is a cursored collection of friend ids.
type FriendIDs struct {
	IDs               []int64 `json:"ids"`
	NextCursor        int64   `json:"next_cursor"`
	NextCursorStr     string  `json:"next_cursor_str"`
	PreviousCursor    int64   `json:"previous_cursor"`
	PreviousCursorStr string  `json:"previous_cursor_str"`
}

// Friends is a cursored collection of friends.
type Friends struct {
	Users             []User `json:"users"`
	NextCursor        int64  `json:"next_cursor"`
	NextCursorStr     string `json:"next_cursor_str"`
	PreviousCursor    int64  `json:"previous_cursor"`
	PreviousCursorStr string `json:"previous_cursor_str"`
}

// FriendService provides methods for accessing Twitter friends endpoints.
type FriendService struct {
	sling *sling.Sling
}

// newFriendService returns a new FriendService.
func newFriendService(sling *sling.Sling) *FriendService {
	return &FriendService{
		sling: sling.Path("friends/"),
	}
}

// FriendIDParams are the parameters for FriendService.Ids
type FriendIDParams struct {
	UserID     int64  `url:"user_id,omitempty"`
	ScreenName string `url:"screen_name,omitempty"`
	Cursor     int64  `url:"cursor,omitempty"`
	Count      int    `url:"count,omitempty"`
}

// IDs returns a cursored collection of user ids that the specified user is following.
// https://dev.twitter.com/rest/reference/get/friends/ids
func (s *FriendService) IDs(params *FriendIDParams) (*FriendIDs, *http.Response, error) {
	ids := new(FriendIDs)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("ids.json").QueryStruct(params).Receive(ids, apiError)
	return ids, resp, relevantError(err, *apiError)
}

// FriendListParams are the parameters for FriendService.List
type FriendListParams struct {
	UserID              int64  `url:"user_id,omitempty"`
	ScreenName          string `url:"screen_name,omitempty"`
	Cursor              int64  `url:"cursor,omitempty"`
	Count               int    `url:"count,omitempty"`
	SkipStatus          *bool  `url:"skip_status,omitempty"`
	IncludeUserEntities *bool  `url:"include_user_entities,omitempty"`
}

// List returns a cursored collection of Users that the specified user is following.
// https://dev.twitter.com/rest/reference/get/friends/list
func (s *FriendService) List(params *FriendListParams) (*Friends, *http.Response, error) {
	friends := new(Friends)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("list.json").QueryStruct(params).Receive(friends, apiError)
	return friends, resp, relevantError(err, *apiError)
}
