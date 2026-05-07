package twitter

import (
	"net/http"

	"github.com/dghubble/sling"
)

// BlockService provides methods for blocking specific user.
type BlockService struct {
	sling *sling.Sling
}

// newBlockService returns a new BlockService.
func newBlockService(sling *sling.Sling) *BlockService {
	return &BlockService{
		sling: sling.Path("blocks/"),
	}
}

// BlockCreateParams are the parameters for BlockService.Create.
type BlockCreateParams struct {
	ScreenName      string `url:"screen_name,omitempty,comma"`
	UserID          int64  `url:"user_id,omitempty,comma"`
	IncludeEntities *bool  `url:"include_entities,omitempty"` // whether 'status' should include entities
	SkipStatus      *bool  `url:"skip_status,omitempty"`
}

// Create a block for specific user, return the user blocked as Entity.
// https://developer.twitter.com/en/docs/accounts-and-users/mute-block-report-users/api-reference/post-blocks-create
func (s *BlockService) Create(params *BlockCreateParams) (User, *http.Response, error) {
	users := new(User)
	apiError := new(APIError)
	resp, err := s.sling.New().Post("create.json").QueryStruct(params).Receive(users, apiError)
	return *users, resp, relevantError(err, *apiError)
}

// BlockDestroyParams are the parameters for BlockService.Destroy.
type BlockDestroyParams struct {
	ScreenName      string `url:"screen_name,omitempty,comma"`
	UserID          int64  `url:"user_id,omitempty,comma"`
	IncludeEntities *bool  `url:"include_entities,omitempty"` // whether 'status' should include entities
	SkipStatus      *bool  `url:"skip_status,omitempty"`
}

// Destroy the block for specific user, return the user unblocked as Entity.
// https://developer.twitter.com/en/docs/accounts-and-users/mute-block-report-users/api-reference/post-blocks-destroy
func (s *BlockService) Destroy(params *BlockDestroyParams) (User, *http.Response, error) {
	users := new(User)
	apiError := new(APIError)
	resp, err := s.sling.New().Post("destroy.json").QueryStruct(params).Receive(users, apiError)
	return *users, resp, relevantError(err, *apiError)
}

// BlockListParams are the parameters for BlockService.List.
type BlockListParams struct {
	IncludeEntities *bool  `url:"include_entities,omitempty"`
	SkipStatus      *bool  `url:"skip_status,omitempty"`
	Cursor          *int64 `url:"cursor,omitempty"`
}

// BlockListResponse is the response from BlockService.List
type BlockListResponse struct {
	PreviousCursor    int64  `json:"previous_cursor"`
	PreviousCursorStr string `json:"previous_cursor_str"`
	NextCursor        int64  `json:"next_cursor"`
	NextCursorStr     string `json:"next_cursor_str"`
	Users             []User `json:"users"`
}

func (s *BlockService) List(params *BlockListParams) (BlockListResponse, *http.Response, error) {
	blr := new(BlockListResponse)
	apiError := new(APIError)
	resp, err := s.sling.New().Post("list.json").QueryStruct(params).Receive(blr, apiError)
	return *blr, resp, relevantError(err, *apiError)
}

// BlockIDsParams are the parameters for BlockService.IDs
type BlockIDsParams struct {
	StringifyIDs *bool  `url:"stringify_ids,omitempty"`
	Cursor       *int64 `url:"cursor,omitempty"`
}

// BlockIDsResponse is the response from BlockService.IDs
type BlockIDsResponse struct {
	NextCursor        int64   `json:"next_cursor"`
	NextCursorStr     string  `json:"next_cursor_str"`
	PreviousCursor    int64   `json:"previous_cusor"`
	PreviousCursorStr string  `json:"previous_cursor_str"`
	IDs               []int64 `json:"ids"`
}

func (s *BlockService) IDs(params *BlockIDsParams) (BlockIDsResponse, *http.Response, error) {
	r := new(BlockIDsResponse)
	apiError := new(APIError)
	resp, err := s.sling.New().Post("ids.json").QueryStruct(params).Receive(r, apiError)
	return *r, resp, relevantError(err, *apiError)
}
