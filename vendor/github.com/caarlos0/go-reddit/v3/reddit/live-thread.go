package reddit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/google/go-querystring/query"
)

// LiveThreadService handles communication with the live thread
// related methods of the Reddit API.
//
// Reddit API docs: https://www.reddit.com/dev/api/#section_live
type LiveThreadService struct {
	client *Client
}

// LiveThread is a thread on Reddit that provides real-time updates.
type LiveThread struct {
	ID      string     `json:"id,omitempty"`
	FullID  string     `json:"name,omitempty"`
	Created *Timestamp `json:"created_utc,omitempty"`

	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Resources   string `json:"resources,omitempty"`

	State             string `json:"state,omitempty"`
	ViewerCount       int    `json:"viewer_count"`
	ViewerCountFuzzed bool   `json:"viewer_count_fuzzed"`

	// Empty when a live thread has ended.
	WebSocketURL string `json:"websocket_url,omitempty"`

	Announcement bool `json:"is_announcement"`
	NSFW         bool `json:"nsfw"`
}

// LiveThreadUpdate is an update in a live thread.
type LiveThreadUpdate struct {
	ID      string     `json:"id,omitempty"`
	FullID  string     `json:"name,omitempty"`
	Author  string     `json:"author,omitempty"`
	Created *Timestamp `json:"created_utc,omitempty"`

	Body         string   `json:"body,omitempty"`
	EmbeddedURLs []string `json:"embeds,omitempty"`
	Stricken     bool     `json:"stricken"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (u *LiveThreadUpdate) UnmarshalJSON(b []byte) error {
	root := new(struct {
		ID      string     `json:"id"`
		FullID  string     `json:"name"`
		Author  string     `json:"author"`
		Created *Timestamp `json:"created_utc"`

		Body         string `json:"body"`
		EmbeddedURLs []struct {
			URL string `json:"url"`
		} `json:"embeds"`
		Stricken bool `json:"stricken"`
	})

	err := json.Unmarshal(b, root)
	if err != nil {
		return err
	}

	u.ID = root.ID
	u.FullID = root.FullID
	u.Author = root.Author
	u.Created = root.Created

	u.Body = root.Body
	u.Stricken = root.Stricken

	for _, eu := range root.EmbeddedURLs {
		u.EmbeddedURLs = append(u.EmbeddedURLs, eu.URL)
	}

	return nil
}

// LiveThreadCreateOrUpdateRequest represents a request to create/update a live thread.
type LiveThreadCreateOrUpdateRequest struct {
	// No longer than 120 characters.
	Title       string `url:"title,omitempty"`
	Description string `url:"description,omitempty"`
	Resources   string `url:"resources,omitempty"`
	NSFW        *bool  `url:"nsfw,omitempty"`
}

// LiveThreadContributor is a user that can contribute to a live thread.
type LiveThreadContributor struct {
	ID          string   `json:"id,omitempty"`
	Name        string   `json:"name,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

// LiveThreadContributors is a list of users that can contribute to a live thread.
type LiveThreadContributors struct {
	Current []*LiveThreadContributor `json:"current_contributors"`
	// This is only filled if you are a contributor in the live thread with the "manage" permission.
	Invited []*LiveThreadContributor `json:"invited_contributors,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (c *LiveThreadContributors) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return errors.New("no bytes to unmarshal")
	}

	// neat trick taken from:
	// https://www.calhoun.io/how-to-parse-json-that-varies-between-an-array-or-a-single-item-with-go
	switch b[0] {
	case '{':
		return c.unmarshalSingle(b)
	case '[':
		return c.unmarshalMany(b)
	}

	// This shouldn't really happen as the standard library seems to strip
	// whitespace from the bytes being passed in, but just in case let's guess at
	// multiple tags and fall back to a single one if that doesn't work.
	err := c.unmarshalSingle(b)
	if err != nil {
		return c.unmarshalMany(b)
	}

	return nil
}

func (c *LiveThreadContributors) unmarshalSingle(b []byte) error {
	root := new(struct {
		Data struct {
			Children []*LiveThreadContributor `json:"children"`
		} `json:"data"`
	})

	err := json.Unmarshal(b, &root)
	if err != nil {
		return err
	}

	c.Current = root.Data.Children
	return nil
}

func (c *LiveThreadContributors) unmarshalMany(b []byte) error {
	var root [2]struct {
		Data struct {
			Children []*LiveThreadContributor `json:"children"`
		} `json:"data"`
	}

	err := json.Unmarshal(b, &root)
	if err != nil {
		return err
	}

	c.Current = root[0].Data.Children
	c.Invited = root[1].Data.Children
	return nil
}

// LiveThreadPermissions are the different permissions contributors have or don't have for a live thread.
// Read about them here: https://mods.reddithelp.com/hc/en-us/articles/360009381491-User-Management-moderators-and-permissions
type LiveThreadPermissions struct {
	All         bool `permission:"all"`
	Close       bool `permission:"close"`
	Discussions bool `permission:"discussions"`
	Edit        bool `permission:"edit"`
	Manage      bool `permission:"manage"`
	Settings    bool `permission:"settings"`
	// Posting updates to the thread.
	Update bool `permission:"update"`
}

func (p *LiveThreadPermissions) String() (s string) {
	if p == nil {
		return "+all"
	}

	t := reflect.TypeOf(*p)
	v := reflect.ValueOf(*p)

	for i := 0; i < t.NumField(); i++ {
		if v.Field(i).Kind() != reflect.Bool {
			continue
		}

		permission := t.Field(i).Tag.Get("permission")
		permitted := v.Field(i).Bool()

		if permitted {
			s += "+"
		} else {
			s += "-"
		}

		s += permission

		if i != t.NumField()-1 {
			s += ","
		}
	}

	return
}

// Now gets information about the currently featured live thread.
// This returns an empty 204 response if no thread is currently featured.
func (s *LiveThreadService) Now(ctx context.Context) (*LiveThread, *Response, error) {
	path := "api/live/happening_now"
	t, resp, err := s.client.getThing(ctx, path, nil)
	if err != nil {
		if err == io.EOF && resp != nil && resp.StatusCode == http.StatusNoContent {
			return nil, resp, nil
		}
		return nil, resp, err
	}
	liveThread, _ := t.LiveThread()
	return liveThread, resp, nil
}

// Get information about a live thread.
func (s *LiveThreadService) Get(ctx context.Context, id string) (*LiveThread, *Response, error) {
	path := fmt.Sprintf("live/%s/about", id)
	t, resp, err := s.client.getThing(ctx, path, nil)
	if err != nil {
		return nil, resp, err
	}
	liveThread, _ := t.LiveThread()
	return liveThread, resp, nil
}

// GetMultiple gets information about multiple live threads.
func (s *LiveThreadService) GetMultiple(ctx context.Context, ids ...string) ([]*LiveThread, *Response, error) {
	if len(ids) == 0 {
		return nil, nil, errors.New("must provide at least 1 id")
	}
	path := fmt.Sprintf("api/live/by_id/%s", strings.Join(ids, ","))
	l, resp, err := s.client.getListing(ctx, path, nil)
	if err != nil {
		return nil, resp, err
	}
	return l.LiveThreads(), resp, nil
}

// Update the live thread by posting an update to it.
// Requires the "update" permission.
func (s *LiveThreadService) Update(ctx context.Context, id, text string) (*Response, error) {
	form := url.Values{}
	form.Set("api_type", "json")
	form.Set("body", text)

	path := fmt.Sprintf("api/live/%s/update", id)
	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Updates gets a list of updates posted in the live thread.
func (s *LiveThreadService) Updates(ctx context.Context, id string, opts *ListOptions) ([]*LiveThreadUpdate, *Response, error) {
	path := fmt.Sprintf("live/%s", id)
	l, resp, err := s.client.getListing(ctx, path, opts)
	if err != nil {
		return nil, resp, err
	}
	return l.LiveThreadUpdates(), resp, nil
}

// UpdateByID gets a specific update in the live thread by its id.
// The ID of the update is the "short" one, i.e. the one that doesn't start with "LiveUpdate_".
func (s *LiveThreadService) UpdateByID(ctx context.Context, threadID, updateID string) (*LiveThreadUpdate, *Response, error) {
	path := fmt.Sprintf("live/%s/updates/%s", threadID, updateID)

	// this endpoint returns a listing
	l, resp, err := s.client.getListing(ctx, path, nil)
	if err != nil {
		return nil, resp, err
	}

	var update *LiveThreadUpdate
	updates := l.LiveThreadUpdates()
	if len(updates) > 0 {
		update = updates[0]
	}

	return update, resp, nil
}

// Discussions gets a list of discussions (posts) about the live thread.
func (s *LiveThreadService) Discussions(ctx context.Context, id string, opts *ListOptions) ([]*Post, *Response, error) {
	path := fmt.Sprintf("live/%s/discussions", id)
	l, resp, err := s.client.getListing(ctx, path, opts)
	if err != nil {
		return nil, resp, err
	}
	return l.Posts(), resp, nil
}

// Strike (mark incorrect and cross out) the content of an update.
// You must either be the author of the update or have the "edit" permission.
func (s *LiveThreadService) Strike(ctx context.Context, threadID, updateID string) (*Response, error) {
	form := url.Values{}
	form.Set("api_type", "json")
	form.Set("id", updateID)

	path := fmt.Sprintf("api/live/%s/strike_update", threadID)
	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Delete an update from the live thread.
// You must either be the author of the update or have the "edit" permission.
func (s *LiveThreadService) Delete(ctx context.Context, threadID, updateID string) (*Response, error) {
	form := url.Values{}
	form.Set("api_type", "json")
	form.Set("id", updateID)

	path := fmt.Sprintf("api/live/%s/delete_update", threadID)
	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Create a live thread and get its id.
func (s *LiveThreadService) Create(ctx context.Context, request *LiveThreadCreateOrUpdateRequest) (string, *Response, error) {
	if request == nil {
		return "", nil, errors.New("*LiveThreadCreateOrUpdateRequest: cannot be nil")
	}

	form, err := query.Values(request)
	if err != nil {
		return "", nil, err
	}
	form.Set("api_type", "json")

	path := "api/live/create"
	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return "", nil, err
	}

	root := new(struct {
		JSON struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		} `json:"json"`
	})
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return "", resp, err
	}

	return root.JSON.Data.ID, resp, nil
}

// Close the thread permanently, disallowing future updates.
func (s *LiveThreadService) Close(ctx context.Context, id string) (*Response, error) {
	form := url.Values{}
	form.Set("api_type", "json")

	path := fmt.Sprintf("api/live/%s/close_thread", id)
	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Configure the thread.
// Requires the "settings" permission.
func (s *LiveThreadService) Configure(ctx context.Context, id string, request *LiveThreadCreateOrUpdateRequest) (*Response, error) {
	if request == nil {
		return nil, errors.New("*LiveThreadCreateOrUpdateRequest: cannot be nil")
	}

	form, err := query.Values(request)
	if err != nil {
		return nil, err
	}
	form.Set("api_type", "json")

	path := fmt.Sprintf("api/live/%s/edit", id)
	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(ctx, req, nil)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// Contributors gets a list of users that are contributors to the live thread.
// If you are a contributor and you have the "manage" permission (to manage contributors), you
// also get a list of invited contributors that haven't yet accepted/refused their invitation.
func (s *LiveThreadService) Contributors(ctx context.Context, id string) (*LiveThreadContributors, *Response, error) {
	path := fmt.Sprintf("live/%s/contributors", id)
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(LiveThreadContributors)
	resp, err := s.client.Do(ctx, req, &root)
	if err != nil {
		return nil, resp, err
	}

	return root, resp, nil
}

// Accept a pending invite to contribute to the live thread.
func (s *LiveThreadService) Accept(ctx context.Context, id string) (*Response, error) {
	form := url.Values{}
	form.Set("api_type", "json")

	path := fmt.Sprintf("api/live/%s/accept_contributor_invite", id)
	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Leave the live thread by abdicating your status as contributor.
func (s *LiveThreadService) Leave(ctx context.Context, id string) (*Response, error) {
	form := url.Values{}
	form.Set("api_type", "json")

	path := fmt.Sprintf("api/live/%s/leave_contributor", id)
	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Invite another user to contribute to the live thread.
// If permissions is nil, all permissions will be granted.
// Requires the "manage" permission.
func (s *LiveThreadService) Invite(ctx context.Context, id, username string, permissions *LiveThreadPermissions) (*Response, error) {
	form := url.Values{}
	form.Set("api_type", "json")
	form.Set("name", username)
	form.Set("type", "liveupdate_contributor_invite")
	form.Set("permissions", permissions.String())

	path := fmt.Sprintf("api/live/%s/invite_contributor", id)
	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Uninvite a user that's been invited to contribute to a live thread via their full ID.
// Requires the "manage" permission.
func (s *LiveThreadService) Uninvite(ctx context.Context, threadID, userID string) (*Response, error) {
	form := url.Values{}
	form.Set("api_type", "json")
	form.Set("id", userID)

	path := fmt.Sprintf("api/live/%s/rm_contributor_invite", threadID)
	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// SetPermissions sets the permissions for the contributor in the live thread.
// If permissions is nil, all permissions will be granted.
// Requires the "manage" permission.
func (s *LiveThreadService) SetPermissions(ctx context.Context, id, username string, permissions *LiveThreadPermissions) (*Response, error) {
	form := url.Values{}
	form.Set("api_type", "json")
	form.Set("name", username)
	form.Set("type", "liveupdate_contributor")
	form.Set("permissions", permissions.String())

	path := fmt.Sprintf("api/live/%s/set_contributor_permissions", id)
	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// SetPermissionsForInvite sets the permissions for a contributor who's yet to accept/refuse their invite.
// If permissions is nil, all permissions will be granted.
// Requires the "manage" permission.
func (s *LiveThreadService) SetPermissionsForInvite(ctx context.Context, id, username string, permissions *LiveThreadPermissions) (*Response, error) {
	form := url.Values{}
	form.Set("api_type", "json")
	form.Set("name", username)
	form.Set("type", "liveupdate_contributor_invite")
	form.Set("permissions", permissions.String())

	path := fmt.Sprintf("api/live/%s/set_contributor_permissions", id)
	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Revoke a user's contributorship via their full ID.
// Requires the "manage" permission.
func (s *LiveThreadService) Revoke(ctx context.Context, threadID, userID string) (*Response, error) {
	form := url.Values{}
	form.Set("api_type", "json")
	form.Set("id", userID)

	path := fmt.Sprintf("api/live/%s/rm_contributor", threadID)
	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// HideDiscussion hides a linked post from the live thread's discussion sidebar.
// The postID should be the base36 ID of the post, i.e. not its full id.
func (s *LiveThreadService) HideDiscussion(ctx context.Context, threadID, postID string) (*Response, error) {
	form := url.Values{}
	form.Set("api_type", "json")
	form.Set("link", postID)

	path := fmt.Sprintf("api/live/%s/hide_discussion", threadID)
	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// UnhideDiscussion unhides a linked post from the live thread's discussion sidebar.
// The postID should be the base36 ID of the post, i.e. not its full id.
func (s *LiveThreadService) UnhideDiscussion(ctx context.Context, threadID, postID string) (*Response, error) {
	form := url.Values{}
	form.Set("api_type", "json")
	form.Set("link", postID)

	path := fmt.Sprintf("api/live/%s/unhide_discussion", threadID)
	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Report the live thread.
// The reason should be one of:
// spam, vote-manipulation, personal-information, sexualizing-minors, site-breaking
func (s *LiveThreadService) Report(ctx context.Context, id, reason string) (*Response, error) {
	switch reason {
	case "spam", "vote-manipulation", "personal-information", "sexualizing-minors", "site-breaking":
	default:
		return nil, errors.New("invalid reason for reporting live thread: " + reason)
	}

	form := url.Values{}
	form.Set("api_type", "json")
	form.Set("type", reason)

	path := fmt.Sprintf("api/live/%s/report", id)
	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}
