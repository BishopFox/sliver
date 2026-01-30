package reddit

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// UserService handles communication with the user
// related methods of the Reddit API.
//
// Reddit API docs: https://www.reddit.com/dev/api/#section_users
type UserService struct {
	client *Client
}

// User represents a Reddit user.
type User struct {
	// this is not the full ID, watch out.
	ID      string     `json:"id,omitempty"`
	Name    string     `json:"name,omitempty"`
	Created *Timestamp `json:"created_utc,omitempty"`

	PostKarma    int `json:"link_karma"`
	CommentKarma int `json:"comment_karma"`

	IsFriend         bool `json:"is_friend"`
	IsEmployee       bool `json:"is_employee"`
	HasVerifiedEmail bool `json:"has_verified_email"`
	NSFW             bool `json:"over_18"`
	IsSuspended      bool `json:"is_suspended"`
}

// UserSummary represents a Reddit user, but
// contains fewer pieces of information.
type UserSummary struct {
	Name    string     `json:"name,omitempty"`
	Created *Timestamp `json:"created_utc,omitempty"`

	PostKarma    int `json:"link_karma"`
	CommentKarma int `json:"comment_karma"`

	NSFW bool `json:"profile_over_18"`
}

// Blocked represents a blocked relationship.
type Blocked struct {
	Blocked   string     `json:"name,omitempty"`
	BlockedID string     `json:"id,omitempty"`
	Created   *Timestamp `json:"date,omitempty"`
}

// Trophy is a Reddit award.
type Trophy struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Get returns information about the user.
func (s *UserService) Get(ctx context.Context, username string) (*User, *Response, error) {
	path := fmt.Sprintf("user/%s/about", username)
	t, resp, err := s.client.getThing(ctx, path, nil)
	if err != nil {
		return nil, resp, err
	}
	user, _ := t.User()
	return user, resp, nil
}

// GetMultipleByID returns multiple users from their full IDs.
// The response body is a map where the keys are the IDs (if they exist), and the value is the user.
func (s *UserService) GetMultipleByID(ctx context.Context, ids ...string) (map[string]*UserSummary, *Response, error) {
	params := struct {
		IDs []string `url:"ids,omitempty,comma"`
	}{ids}

	path := "api/user_data_by_account_ids"
	path, err := addOptions(path, params)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := make(map[string]*UserSummary)
	resp, err := s.client.Do(ctx, req, &root)
	if err != nil {
		return nil, resp, err
	}

	return root, resp, nil
}

// UsernameAvailable checks whether a username is available for registration.
func (s *UserService) UsernameAvailable(ctx context.Context, username string) (bool, *Response, error) {
	params := struct {
		User string `url:"user"`
	}{username}

	path := "api/username_available"
	path, err := addOptions(path, params)
	if err != nil {
		return false, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return false, nil, err
	}

	root := new(bool)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return false, resp, err
	}

	return *root, resp, nil
}

// Overview returns a list of your posts and comments.
func (s *UserService) Overview(ctx context.Context, opts *ListUserOverviewOptions) ([]*Post, []*Comment, *Response, error) {
	return s.OverviewOf(ctx, s.client.Username, opts)
}

// OverviewOf returns a list of the user's posts and comments.
func (s *UserService) OverviewOf(ctx context.Context, username string, opts *ListUserOverviewOptions) ([]*Post, []*Comment, *Response, error) {
	path := fmt.Sprintf("user/%s/overview", username)
	l, resp, err := s.client.getListing(ctx, path, opts)
	if err != nil {
		return nil, nil, resp, err
	}
	return l.Posts(), l.Comments(), resp, nil
}

// Posts returns a list of your posts.
func (s *UserService) Posts(ctx context.Context, opts *ListUserOverviewOptions) ([]*Post, *Response, error) {
	return s.PostsOf(ctx, s.client.Username, opts)
}

// PostsOf returns a list of the user's posts.
func (s *UserService) PostsOf(ctx context.Context, username string, opts *ListUserOverviewOptions) ([]*Post, *Response, error) {
	path := fmt.Sprintf("user/%s/submitted", username)
	l, resp, err := s.client.getListing(ctx, path, opts)
	if err != nil {
		return nil, resp, err
	}
	return l.Posts(), resp, nil
}

// Comments returns a list of your comments.
func (s *UserService) Comments(ctx context.Context, opts *ListUserOverviewOptions) ([]*Comment, *Response, error) {
	return s.CommentsOf(ctx, s.client.Username, opts)
}

// CommentsOf returns a list of the user's comments.
func (s *UserService) CommentsOf(ctx context.Context, username string, opts *ListUserOverviewOptions) ([]*Comment, *Response, error) {
	path := fmt.Sprintf("user/%s/comments", username)
	l, resp, err := s.client.getListing(ctx, path, opts)
	if err != nil {
		return nil, resp, err
	}
	return l.Comments(), resp, nil
}

// Saved returns a list of the user's saved posts and comments.
func (s *UserService) Saved(ctx context.Context, opts *ListUserOverviewOptions) ([]*Post, []*Comment, *Response, error) {
	path := fmt.Sprintf("user/%s/saved", s.client.Username)
	l, resp, err := s.client.getListing(ctx, path, opts)
	if err != nil {
		return nil, nil, resp, err
	}
	return l.Posts(), l.Comments(), resp, nil
}

// Upvoted returns a list of your upvoted posts.
func (s *UserService) Upvoted(ctx context.Context, opts *ListUserOverviewOptions) ([]*Post, *Response, error) {
	return s.UpvotedOf(ctx, s.client.Username, opts)
}

// UpvotedOf returns a list of the user's upvoted posts.
// The user's votes must be public for this to work (unless the user is you).
func (s *UserService) UpvotedOf(ctx context.Context, username string, opts *ListUserOverviewOptions) ([]*Post, *Response, error) {
	path := fmt.Sprintf("user/%s/upvoted", username)
	l, resp, err := s.client.getListing(ctx, path, opts)
	if err != nil {
		return nil, resp, err
	}
	return l.Posts(), resp, nil
}

// Downvoted returns a list of your downvoted posts.
func (s *UserService) Downvoted(ctx context.Context, opts *ListUserOverviewOptions) ([]*Post, *Response, error) {
	return s.DownvotedOf(ctx, s.client.Username, opts)
}

// DownvotedOf returns a list of the user's downvoted posts.
// The user's votes must be public for this to work (unless the user is you).
func (s *UserService) DownvotedOf(ctx context.Context, username string, opts *ListUserOverviewOptions) ([]*Post, *Response, error) {
	path := fmt.Sprintf("user/%s/downvoted", username)
	l, resp, err := s.client.getListing(ctx, path, opts)
	if err != nil {
		return nil, resp, err
	}
	return l.Posts(), resp, nil
}

// Hidden returns a list of the user's hidden posts.
func (s *UserService) Hidden(ctx context.Context, opts *ListUserOverviewOptions) ([]*Post, *Response, error) {
	path := fmt.Sprintf("user/%s/hidden", s.client.Username)
	l, resp, err := s.client.getListing(ctx, path, opts)
	if err != nil {
		return nil, resp, err
	}
	return l.Posts(), resp, nil
}

// Gilded returns a list of the user's gilded posts.
func (s *UserService) Gilded(ctx context.Context, opts *ListUserOverviewOptions) ([]*Post, *Response, error) {
	path := fmt.Sprintf("user/%s/gilded", s.client.Username)
	l, resp, err := s.client.getListing(ctx, path, opts)
	if err != nil {
		return nil, resp, err
	}
	return l.Posts(), resp, nil
}

// GetFriendship returns relationship details with the specified user.
// If the user is not your friend, it will return an error.
func (s *UserService) GetFriendship(ctx context.Context, username string) (*Relationship, *Response, error) {
	path := fmt.Sprintf("api/v1/me/friends/%s", username)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(Relationship)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root, resp, nil
}

// Friend a user.
func (s *UserService) Friend(ctx context.Context, username string) (*Relationship, *Response, error) {
	body := struct {
		Username string `json:"name"`
	}{username}

	path := fmt.Sprintf("api/v1/me/friends/%s", username)
	req, err := s.client.NewJSONRequest(http.MethodPut, path, body)
	if err != nil {
		return nil, nil, err
	}

	root := new(Relationship)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root, resp, nil
}

// Unfriend a user.
func (s *UserService) Unfriend(ctx context.Context, username string) (*Response, error) {
	path := fmt.Sprintf("api/v1/me/friends/%s", username)
	req, err := s.client.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

// Block a user.
func (s *UserService) Block(ctx context.Context, username string) (*Blocked, *Response, error) {
	path := "api/block_user"

	form := url.Values{}
	form.Set("name", username)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, nil, err
	}

	root := new(Blocked)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root, resp, nil
}

// BlockByID blocks a user via their full id.
func (s *UserService) BlockByID(ctx context.Context, id string) (*Blocked, *Response, error) {
	path := "api/block_user"

	form := url.Values{}
	form.Set("account_id", id)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, nil, err
	}

	root := new(Blocked)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root, resp, nil
}

// Unblock a user.
func (s *UserService) Unblock(ctx context.Context, username string) (*Response, error) {
	selfID, resp, err := s.client.id(ctx)
	if err != nil {
		return resp, err
	}

	path := "api/unfriend"

	form := url.Values{}
	form.Set("name", username)
	form.Set("type", "enemy")
	form.Set("container", selfID)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// UnblockByID unblocks a user via their full id.
func (s *UserService) UnblockByID(ctx context.Context, id string) (*Response, error) {
	selfID, resp, err := s.client.id(ctx)
	if err != nil {
		return resp, err
	}

	path := "api/unfriend"

	form := url.Values{}
	form.Set("id", id)
	form.Set("type", "enemy")
	form.Set("container", selfID)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Trophies returns a list of your trophies.
func (s *UserService) Trophies(ctx context.Context) ([]*Trophy, *Response, error) {
	return s.TrophiesOf(ctx, s.client.Username)
}

// TrophiesOf returns a list of the specified user's trophies.
func (s *UserService) TrophiesOf(ctx context.Context, username string) ([]*Trophy, *Response, error) {
	path := fmt.Sprintf("api/v1/user/%s/trophies", username)
	t, resp, err := s.client.getThing(ctx, path, nil)
	if err != nil {
		return nil, resp, err
	}
	trophies, _ := t.TrophyList()
	return trophies, resp, nil
}

// Popular gets the user subreddits with the most activity.
func (s *UserService) Popular(ctx context.Context, opts *ListOptions) ([]*Subreddit, *Response, error) {
	path := "users/popular"
	l, resp, err := s.client.getListing(ctx, path, opts)
	if err != nil {
		return nil, resp, err
	}
	return l.Subreddits(), resp, nil
}

// New gets the most recently created user subreddits.
func (s *UserService) New(ctx context.Context, opts *ListUserOverviewOptions) ([]*Subreddit, *Response, error) {
	path := "users/new"
	l, resp, err := s.client.getListing(ctx, path, opts)
	if err != nil {
		return nil, resp, err
	}
	return l.Subreddits(), resp, nil
}

// Search for users.
// todo: maybe include the sort option? (relevance, activity)
func (s *UserService) Search(ctx context.Context, query string, opts *ListOptions) ([]*User, *Response, error) {
	path := fmt.Sprintf("users/search?q=%s", query)
	l, resp, err := s.client.getListing(ctx, path, opts)
	if err != nil {
		return nil, resp, err
	}
	return l.Users(), resp, nil
}
