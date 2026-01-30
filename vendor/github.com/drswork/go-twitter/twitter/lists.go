package twitter

import (
	"net/http"

	"github.com/dghubble/sling"
)

// List represents a Twitter List.
type List struct {
	Slug            string `json:"slug"`
	Name            string `json:"name"`
	CreatedAt       string `json:"created_at"`
	URI             string `json:"uri"`
	SubscriberCount int    `json:"subscriber_count"`
	IDStr           string `json:"id_str"`
	MemberCount     int    `json:"member_count"`
	Mode            string `json:"mode"`
	ID              int64  `json:"id"`
	FullName        string `json:"full_name"`
	Description     string `json:"description"`
	User            *User  `json:"user"`
	Following       bool   `json:"following"`
}

// Members is a cursored collection of list members.
type Members struct {
	Users             []User `json:"users"`
	NextCursor        int64  `json:"next_cursor"`
	NextCursorStr     string `json:"next_cursor_str"`
	PreviousCursor    int64  `json:"previous_cursor"`
	PreviousCursorStr string `json:"previous_cursor_str"`
}

// Membership is a cursored collection of lists a user is on.
type Membership struct {
	Lists             []List `json:"lists"`
	NextCursor        int64  `json:"next_cursor"`
	NextCursorStr     string `json:"next_cursor_str"`
	PreviousCursor    int64  `json:"previous_cursor"`
	PreviousCursorStr string `json:"previous_cursor_str"`
}

// Ownership is a cursored collection of lists a user owns.
type Ownership struct {
	Lists             []List `json:"lists"`
	NextCursor        int64  `json:"next_cursor"`
	NextCursorStr     string `json:"next_cursor_str"`
	PreviousCursor    int64  `json:"previous_cursor"`
	PreviousCursorStr string `json:"previous_cursor_str"`
}

// Subscribers is a cursored collection of users that subscribe to a list.
type Subscribers struct {
	Users             []User `json:"users"`
	NextCursor        int64  `json:"next_cursor"`
	NextCursorStr     string `json:"next_cursor_str"`
	PreviousCursor    int64  `json:"previous_cursor"`
	PreviousCursorStr string `json:"previous_cursor_str"`
}

// Subscribed is a cursored collection of lists the user is subscribed to.
type Subscribed struct {
	Lists             []List `json:"lists"`
	NextCursor        int64  `json:"next_cursor"`
	NextCursorStr     string `json:"next_cursor_str"`
	PreviousCursor    int64  `json:"previous_cursor"`
	PreviousCursorStr string `json:"previous_cursor_str"`
}

// ListsService provides methods for accessing Twitter lists endpoints.
type ListsService struct {
	sling *sling.Sling
}

// newListService returns a new ListService.
func newListService(sling *sling.Sling) *ListsService {
	return &ListsService{
		sling: sling.Path("lists/"),
	}
}

// ListsListParams are the parameters for ListsService.List
type ListsListParams struct {
	UserID     int64  `url:"user_id,omitempty"`
	ScreenName string `url:"screen_name,omitempty"`
	Reverse    bool   `url:"reverse,omitempty"`
}

// List returns all lists the authenticating or specified user subscribes to, including their own.
// https://developer.twitter.com/en/docs/accounts-and-users/create-manage-lists/api-reference/get-lists-list
func (s *ListsService) List(params *ListsListParams) ([]List, *http.Response, error) {
	list := new([]List)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("list.json").QueryStruct(params).Receive(list, apiError)
	return *list, resp, relevantError(err, *apiError)
}

// ListsMembersParams are the parameters for ListsService.Members
type ListsMembersParams struct {
	ListID          int64  `url:"list_id,omitempty"`
	Slug            string `url:"slug,omitempty"`
	OwnerScreenName string `url:"owner_screen_name,omitempty"`
	OwnerID         int64  `url:"owner_id,omitempty"`
	Count           int    `url:"count,omitempty"`
	Cursor          int64  `url:"cursor,omitempty"`
	IncludeEntities *bool  `url:"include_entities,omitempty"`
	SkipStatus      *bool  `url:"skip_status,omitempty"`
}

// Members returns the members of the specified list
// https://developer.twitter.com/en/docs/accounts-and-users/create-manage-lists/api-reference/get-lists-members
func (s *ListsService) Members(params *ListsMembersParams) (*Members, *http.Response, error) {
	members := new(Members)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("members.json").QueryStruct(params).Receive(members, apiError)
	return members, resp, relevantError(err, *apiError)
}

// ListsMembersShowParams are the parameters for ListsService.MembersShow
type ListsMembersShowParams struct {
	ListID          int64  `url:"list_id,omitempty"`
	Slug            string `url:"slug,omitempty"`
	UserID          int64  `url:"user_id,omitempty"`
	ScreenName      string `url:"screen_name,omitempty"`
	OwnerScreenName string `url:"owner_screen_name,omitempty"`
	OwnerID         int64  `url:"owner_id,omitempty"`
	IncludeEntities *bool  `url:"include_entities,omitempty"`
	SkipStatus      *bool  `url:"skip_status,omitempty"`
}

// MembersShow checks if the specified user is a member of the specified list.
// https://developer.twitter.com/en/docs/accounts-and-users/create-manage-lists/api-reference/get-lists-members-show
func (s *ListsService) MembersShow(params *ListsMembersShowParams) (*User, *http.Response, error) {
	user := new(User)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("members/show.json").QueryStruct(params).Receive(user, apiError)
	return user, resp, relevantError(err, *apiError)
}

// ListsMembershipsParams are the parameters for ListsService.Memberships
type ListsMembershipsParams struct {
	UserID             int64  `url:"user_id,omitempty"`
	ScreenName         string `url:"screen_name,omitempty"`
	Count              int    `url:"count,omitempty"`
	Cursor             int64  `url:"cursor,omitempty"`
	FilterToOwnedLists *bool  `url:"filter_to_owned_lists,omitempty"`
}

// Memberships returns the lists the specified user has been added to.
// https://developer.twitter.com/en/docs/accounts-and-users/create-manage-lists/api-reference/get-lists-memberships
func (s *ListsService) Memberships(params *ListsMembershipsParams) (*Membership, *http.Response, error) {
	membership := new(Membership)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("memberships.json").QueryStruct(params).Receive(membership, apiError)
	return membership, resp, relevantError(err, *apiError)
}

// ListsOwnershipsParams are the parameters for ListsService.Ownerships
type ListsOwnershipsParams struct {
	UserID     int64  `url:"user_id,omitempty"`
	ScreenName string `url:"screen_name,omitempty"`
	Count      int    `url:"count,omitempty"`
	Cursor     int64  `url:"cursor,omitempty"`
}

// Ownerships returns the lists owned by the specified Twitter user.
// https://developer.twitter.com/en/docs/accounts-and-users/create-manage-lists/api-reference/get-lists-ownerships
func (s *ListsService) Ownerships(params *ListsOwnershipsParams) (*Ownership, *http.Response, error) {
	ownership := new(Ownership)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("ownerships.json").QueryStruct(params).Receive(ownership, apiError)
	return ownership, resp, relevantError(err, *apiError)
}

// ListsShowParams are the parameters for ListsService.Show
type ListsShowParams struct {
	ListID          int64  `url:"list_id,omitempty"`
	Slug            string `url:"slug,omitempty"`
	OwnerScreenName string `url:"owner_screen_name,omitempty"`
	OwnerID         int64  `url:"owner_id,omitempty"`
}

// Show returns the specified list.
// https://developer.twitter.com/en/docs/accounts-and-users/create-manage-lists/api-reference/get-lists-show
func (s *ListsService) Show(params *ListsShowParams) (*List, *http.Response, error) {
	list := new(List)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("show.json").QueryStruct(params).Receive(list, apiError)
	return list, resp, relevantError(err, *apiError)
}

// ListsStatusesParams are the parameters for ListsService.Statuses
type ListsStatusesParams struct {
	ListID          int64  `url:"list_id,omitempty"`
	Slug            string `url:"slug,omitempty"`
	OwnerScreenName string `url:"owner_screen_name,omitempty"`
	OwnerID         int64  `url:"owner_id,omitempty"`
	SinceID         int64  `url:"since_id,omitempty"`
	MaxID           int64  `url:"max_id,omitempty"`
	Count           int    `url:"count,omitempty"`
	IncludeEntities *bool  `url:"include_entities,omitempty"`
	IncludeRetweets *bool  `url:"include_rts,omitempty"`
	TweetMode       string `url:"tweet_mode,omitempty"`
}

// Statuses returns a timeline of tweets authored by members of the specified list.
// https://developer.twitter.com/en/docs/accounts-and-users/create-manage-lists/api-reference/get-lists-statuses
func (s *ListsService) Statuses(params *ListsStatusesParams) ([]Tweet, *http.Response, error) {
	tweets := new([]Tweet)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("statuses.json").QueryStruct(params).Receive(tweets, apiError)
	return *tweets, resp, relevantError(err, *apiError)
}

// ListsSubscribersParams are the parameters for ListsService.Subscribers
type ListsSubscribersParams struct {
	ListID          int64  `url:"list_id,omitempty"`
	Slug            string `url:"slug,omitempty"`
	OwnerScreenName string `url:"owner_screen_name,omitempty"`
	OwnerID         int64  `url:"owner_id,omitempty"`
	Count           int    `url:"count,omitempty"`
	Cursor          int64  `url:"cursor,omitempty"`
	IncludeEntities *bool  `url:"include_entities,omitempty"`
	SkipStatus      *bool  `url:"skip_status,omitempty"`
}

// Subscribers returns the subscribers of the specified list.
// https://developer.twitter.com/en/docs/accounts-and-users/create-manage-lists/api-reference/get-lists-subscribers
func (s *ListsService) Subscribers(params *ListsSubscribersParams) (*Subscribers, *http.Response, error) {
	subscribers := new(Subscribers)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("subscribers.json").QueryStruct(params).Receive(subscribers, apiError)
	return subscribers, resp, relevantError(err, *apiError)
}

// ListsSubscribersShowParams are the parameters for ListsService.SubscribersShow
type ListsSubscribersShowParams struct {
	OwnerScreenName string `url:"owner_screen_name,omitempty"`
	OwnerID         int64  `url:"owner_id,omitempty"`
	ListID          int64  `url:"list_id,omitempty"`
	Slug            string `url:"slug,omitempty"`
	UserID          int64  `url:"user_id,omitempty"`
	ScreenName      string `url:"screen_name,omitempty"`
	IncludeEntities *bool  `url:"include_entities,omitempty"`
	SkipStatus      *bool  `url:"skip_status,omitempty"`
}

// SubscribersShow returns the user if they are a subscriber to the list.
// https://developer.twitter.com/en/docs/accounts-and-users/create-manage-lists/api-reference/get-lists-subscribers-show
func (s *ListsService) SubscribersShow(params *ListsSubscribersShowParams) (*User, *http.Response, error) {
	user := new(User)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("subscribers/show.json").QueryStruct(params).Receive(user, apiError)
	return user, resp, relevantError(err, *apiError)
}

// ListsSubscriptionsParams are the parameters for ListsService.Subscriptions
type ListsSubscriptionsParams struct {
	UserID     int64  `url:"user_id,omitempty"`
	ScreenName string `url:"screen_name,omitempty"`
	Count      int    `url:"count,omitempty"`
	Cursor     int64  `url:"cursor,omitempty"`
}

// Subscriptions returns a collection of the lists the specified user is subscribed to.
// https://developer.twitter.com/en/docs/accounts-and-users/create-manage-lists/api-reference/get-lists-subscriptions
func (s *ListsService) Subscriptions(params *ListsSubscriptionsParams) (*Subscribed, *http.Response, error) {
	subscribed := new(Subscribed)
	apiError := new(APIError)
	resp, err := s.sling.New().Get("subscriptions.json").QueryStruct(params).Receive(subscribed, apiError)
	return subscribed, resp, relevantError(err, *apiError)
}

// ListsCreateParams are the parameters for ListsService.Create
type ListsCreateParams struct {
	Name        string `url:"name,omitempty"`
	Mode        string `url:"mode,omitempty"`
	Description string `url:"description,omitempty"`
}

// Create creates a new list for the authenticated user.
// https://developer.twitter.com/en/docs/accounts-and-users/create-manage-lists/api-reference/post-lists-create
func (s *ListsService) Create(name string, params *ListsCreateParams) (*List, *http.Response, error) {
	if params == nil {
		params = &ListsCreateParams{}
	}
	params.Name = name
	list := new(List)
	apiError := new(APIError)
	resp, err := s.sling.New().Post("create.json").BodyForm(params).Receive(list, apiError)
	return list, resp, relevantError(err, *apiError)

}

// ListsDestroyParams are the parameters for ListsService.Destroy
type ListsDestroyParams struct {
	OwnerScreenName string `url:"owner_screen_name,omitempty"`
	OwnerID         int64  `url:"owner_id,omitempty"`
	ListID          int64  `url:"list_id,omitempty"`
	Slug            string `url:"slug,omitempty"`
}

// Destroy deletes the specified list.
// https://developer.twitter.com/en/docs/accounts-and-users/create-manage-lists/api-reference/post-lists-destroy
func (s *ListsService) Destroy(params *ListsDestroyParams) (*List, *http.Response, error) {
	list := new(List)
	apiError := new(APIError)
	resp, err := s.sling.New().Post("destroy.json").BodyForm(params).Receive(list, apiError)
	return list, resp, relevantError(err, *apiError)
}

// ListsMembersCreateParams are the parameters for ListsService.MembersCreate
type ListsMembersCreateParams struct {
	ListID          int64  `url:"list_id,omitempty"`
	Slug            string `url:"slug,omitempty"`
	UserID          int64  `url:"user_id,omitempty"`
	ScreenName      string `url:"screen_name,omitempty"`
	OwnerScreenName string `url:"owner_screen_name,omitempty"`
	OwnerID         int64  `url:"owner_id,omitempty"`
}

// MembersCreate adds a member to a list.
// https://developer.twitter.com/en/docs/accounts-and-users/create-manage-lists/api-reference/post-lists-members-create
func (s *ListsService) MembersCreate(params *ListsMembersCreateParams) (*http.Response, error) {
	apiError := new(APIError)
	resp, err := s.sling.New().Post("members/create.json").BodyForm(params).Receive(nil, apiError)
	return resp, err
}

// ListsMembersCreateAllParams are the parameters for ListsService.MembersCreateAll
type ListsMembersCreateAllParams struct {
	ListID          int64  `url:"list_id,omitempty"`
	Slug            string `url:"slug,omitempty"`
	UserID          string `url:"user_id,omitempty"`
	ScreenName      string `url:"screen_name,omitempty"`
	OwnerScreenName string `url:"owner_screen_name,omitempty"`
	OwnerID         int64  `url:"owner_id,omitempty"`
}

// MembersCreateAll adds multiple members to a list.
// https://developer.twitter.com/en/docs/accounts-and-users/create-manage-lists/api-reference/post-lists-members-create_all
func (s *ListsService) MembersCreateAll(params *ListsMembersCreateAllParams) (*http.Response, error) {
	apiError := new(APIError)
	resp, err := s.sling.New().Post("members/create_all.json").BodyForm(params).Receive(nil, apiError)
	return resp, err
}

// ListsMembersDestroyParams are the parameters for ListsService.MembersDestroy
type ListsMembersDestroyParams struct {
	ListID          int64  `url:"list_id,omitempty"`
	Slug            string `url:"slug,omitempty"`
	UserID          int64  `url:"user_id,omitempty"`
	ScreenName      string `url:"screen_name,omitempty"`
	OwnerScreenName string `url:"owner_screen_name,omitempty"`
	OwnerID         int64  `url:"owner_id,omitempty"`
}

// MembersDestroy removes the specified member from the list.
// https://developer.twitter.com/en/docs/accounts-and-users/create-manage-lists/api-reference/post-lists-members-destroy
func (s *ListsService) MembersDestroy(params *ListsMembersDestroyParams) (*http.Response, error) {
	apiError := new(APIError)
	resp, err := s.sling.New().Post("members/destroy.json").BodyForm(params).Receive(nil, apiError)
	return resp, err
}

// ListsMembersDestroyAllParams are the parameters for ListsService.MembersDestroyAll
type ListsMembersDestroyAllParams struct {
	ListID          int64  `url:"list_id,omitempty"`
	Slug            string `url:"slug,omitempty"`
	UserID          string `url:"user_id,omitempty"`
	ScreenName      string `url:"screen_name,omitempty"`
	OwnerScreenName string `url:"owner_screen_name,omitempty"`
	OwnerID         int64  `url:"owner_id,omitempty"`
}

// MembersDestroyAll removes multiple members from a list.
// https://developer.twitter.com/en/docs/accounts-and-users/create-manage-lists/api-reference/post-lists-members-destroy_all
func (s *ListsService) MembersDestroyAll(params *ListsMembersDestroyAllParams) (*http.Response, error) {
	apiError := new(APIError)
	resp, err := s.sling.New().Post("members/destroy_all.json").BodyForm(params).Receive(nil, apiError)
	return resp, err
}

// ListsSubscribersCreateParams are the parameters for ListsService.SubscribersCreate
type ListsSubscribersCreateParams struct {
	OwnerScreenName string `url:"owner_screen_name,omitempty"`
	OwnerID         int64  `url:"owner_id,omitempty"`
	ListID          int64  `url:"list_id,omitempty"`
	Slug            string `url:"slug,omitempty"`
}

// SubscribersCreate subscribes the authenticated user to the specified list.
// https://developer.twitter.com/en/docs/accounts-and-users/create-manage-lists/api-reference/post-lists-subscribers-create
func (s *ListsService) SubscribersCreate(params *ListsSubscribersCreateParams) (*List, *http.Response, error) {
	list := new(List)
	apiError := new(APIError)
	resp, err := s.sling.New().Post("subscribers/create.json").BodyForm(params).Receive(list, apiError)
	return list, resp, err
}

// ListsSubscribersDestroyParams are the parameters for ListsService.SubscribersDestroy
type ListsSubscribersDestroyParams struct {
	ListID          int64  `url:"list_id,omitempty"`
	Slug            string `url:"slug,omitempty"`
	OwnerScreenName string `url:"owner_screen_name,omitempty"`
	OwnerID         int64  `url:"owner_id,omitempty"`
}

// SubscribersDestroy unsubscribes the authenticated user from the specified list.
// https://developer.twitter.com/en/docs/accounts-and-users/create-manage-lists/api-reference/post-lists-subscribers-destroy
func (s *ListsService) SubscribersDestroy(params *ListsSubscribersDestroyParams) (*http.Response, error) {
	apiError := new(APIError)
	resp, err := s.sling.New().Post("subscribers/destroy.json").BodyForm(params).Receive(nil, apiError)
	return resp, err
}

// ListsUpdateParams are the parameters for ListsService.Update
type ListsUpdateParams struct {
	ListID          int64  `url:"list_id,omitempty"`
	Slug            string `url:"slug,omitempty"`
	Name            string `url:"name,omitempty"`
	Mode            string `url:"mode,omitempty"`
	Description     string `url:"description,omitempty"`
	OwnerScreenName string `url:"owner_screen_name,omitempty"`
	OwnerID         int64  `url:"owner_id,omitempty"`
}

// Update updates the specified list.
// https://developer.twitter.com/en/docs/accounts-and-users/create-manage-lists/api-reference/post-lists-update
func (s *ListsService) Update(params *ListsUpdateParams) (*http.Response, error) {
	apiError := new(APIError)
	resp, err := s.sling.New().Post("update.json").BodyForm(params).Receive(nil, apiError)
	return resp, err
}
