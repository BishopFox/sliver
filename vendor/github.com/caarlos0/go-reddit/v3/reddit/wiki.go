package reddit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/go-querystring/query"
)

// WikiService handles communication with the wiki
// related methods of the Reddit API.
//
// Reddit API docs: https://www.reddit.com/dev/api/#section_wiki
type WikiService struct {
	client *Client
}

// WikiPage is a wiki page in a subreddit.
type WikiPage struct {
	Content   string `json:"content_md,omitempty"`
	Reason    string `json:"reason,omitempty"`
	MayRevise bool   `json:"may_revise"`

	RevisionID   string     `json:"revision_id,omitempty"`
	RevisionDate *Timestamp `json:"revision_date,omitempty"`
	RevisionBy   *User      `json:"revision_by,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (p *WikiPage) UnmarshalJSON(b []byte) error {
	root := new(struct {
		Content   string `json:"content_md,omitempty"`
		Reason    string `json:"reason,omitempty"`
		MayRevise bool   `json:"may_revise"`

		RevisionID   string     `json:"revision_id,omitempty"`
		RevisionDate *Timestamp `json:"revision_date,omitempty"`
		RevisionBy   thing      `json:"revision_by,omitempty"`
	})

	err := json.Unmarshal(b, root)
	if err != nil {
		return err
	}

	p.Content = root.Content
	p.Reason = root.Reason
	p.MayRevise = root.MayRevise

	p.RevisionID = root.RevisionID
	p.RevisionDate = root.RevisionDate

	if user, ok := root.RevisionBy.User(); ok {
		p.RevisionBy = user
	}

	return nil
}

// WikiPageEditRequest represents a request to edit a wiki page in a subreddit.
type WikiPageEditRequest struct {
	Subreddit string `url:"-"`
	Page      string `url:"page"`
	Content   string `url:"content"`
	// Optional, up to 256 characters long.
	Reason string `url:"reason,omitempty"`
}

// WikiPagePermissionLevel defines who can edit a specific wiki page in a subreddit.
type WikiPagePermissionLevel int

const (
	// PermissionSubredditWikiPermissions uses subreddit wiki permissions.
	PermissionSubredditWikiPermissions WikiPagePermissionLevel = iota
	// PermissionApprovedContributorsOnly is only for approved wiki contributors.
	PermissionApprovedContributorsOnly
	// PermissionModeratorsOnly is only for moderators.
	PermissionModeratorsOnly
)

// WikiPageSettings holds the settings for a specific wiki page.
type WikiPageSettings struct {
	PermissionLevel WikiPagePermissionLevel `json:"permlevel"`
	Listed          bool                    `json:"listed"`
	Editors         []*User                 `json:"editors"`
}

// WikiPageSettingsUpdateRequest represents a request to update the visibility and
// permissions of a wiki page.
type WikiPageSettingsUpdateRequest struct {
	// This HAS to be provided no matter what, or else we get a 500 response.
	PermissionLevel WikiPagePermissionLevel `url:"permlevel"`
	Listed          *bool                   `url:"listed,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (s *WikiPageSettings) UnmarshalJSON(b []byte) error {
	root := new(struct {
		PermissionLevel WikiPagePermissionLevel `json:"permlevel"`
		Listed          bool                    `json:"listed"`
		Things          []thing                 `json:"editors"`
	})

	err := json.Unmarshal(b, root)
	if err != nil {
		return err
	}

	s.PermissionLevel = root.PermissionLevel
	s.Listed = root.Listed

	for _, thing := range root.Things {
		if user, ok := thing.User(); ok {
			s.Editors = append(s.Editors, user)
		}
	}

	return nil
}

type wikiPageRevisionListing struct {
	Data struct {
		Revisions []*WikiPageRevision `json:"children"`
		After     string              `json:"after"`
	} `json:"data"`
}

func (l *wikiPageRevisionListing) After() string {
	return l.Data.After
}

// WikiPageRevision is a revision of a wiki page.
type WikiPageRevision struct {
	ID      string     `json:"id,omitempty"`
	Page    string     `json:"page,omitempty"`
	Created *Timestamp `json:"timestamp,omitempty"`
	Reason  string     `json:"reason,omitempty"`
	Hidden  bool       `json:"revision_hidden"`
	Author  *User      `json:"author,omitempty"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (r *WikiPageRevision) UnmarshalJSON(b []byte) error {
	root := new(struct {
		ID      string     `json:"id,omitempty"`
		Page    string     `json:"page,omitempty"`
		Created *Timestamp `json:"timestamp,omitempty"`
		Reason  string     `json:"reason,omitempty"`
		Hidden  bool       `json:"revision_hidden"`
		Author  thing      `json:"author,omitempty"`
	})

	err := json.Unmarshal(b, root)
	if err != nil {
		return err
	}

	r.ID = root.ID
	r.Page = root.Page
	r.Created = root.Created
	r.Reason = root.Reason
	r.Hidden = root.Hidden

	if user, ok := root.Author.User(); ok {
		r.Author = user
	}

	return nil
}

// Page gets a wiki page.
func (s *WikiService) Page(ctx context.Context, subreddit, page string) (*WikiPage, *Response, error) {
	return s.PageRevision(ctx, subreddit, page, "")
}

// PageRevision gets a wiki page at the version it was at the revisionID provided.
// If revisionID is an empty string, it will get the most recent version.
func (s *WikiService) PageRevision(ctx context.Context, subreddit, page, revisionID string) (*WikiPage, *Response, error) {
	path := fmt.Sprintf("r/%s/wiki/%s", subreddit, page)

	params := struct {
		RevisionID string `url:"v,omitempty"`
	}{revisionID}

	t, resp, err := s.client.getThing(ctx, path, params)
	if err != nil {
		return nil, resp, err
	}

	wikiPage, _ := t.WikiPage()
	return wikiPage, resp, nil
}

// Pages gets a list of wiki pages in the subreddit.
// Returns 403 Forbidden if the wiki is disabled.
func (s *WikiService) Pages(ctx context.Context, subreddit string) ([]string, *Response, error) {
	path := fmt.Sprintf("r/%s/wiki/pages", subreddit)
	t, resp, err := s.client.getThing(ctx, path, nil)
	if err != nil {
		return nil, resp, err
	}
	wikiPages, _ := t.WikiPages()
	return wikiPages, resp, nil
}

// Edit a wiki page.
func (s *WikiService) Edit(ctx context.Context, editRequest *WikiPageEditRequest) (*Response, error) {
	if editRequest == nil {
		return nil, errors.New("*WikiPageEditRequest: cannot be nil")
	}

	form, err := query.Values(editRequest)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("r/%s/api/wiki/edit", editRequest.Subreddit)
	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Revert a wiki page to a specific revision.
func (s *WikiService) Revert(ctx context.Context, subreddit, page, revisionID string) (*Response, error) {
	path := fmt.Sprintf("r/%s/api/wiki/revert", subreddit)

	form := url.Values{}
	form.Set("page", page)
	form.Set("revision", revisionID)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Settings gets the subreddit's wiki page's settings.
func (s *WikiService) Settings(ctx context.Context, subreddit, page string) (*WikiPageSettings, *Response, error) {
	path := fmt.Sprintf("r/%s/wiki/settings/%s", subreddit, page)
	t, resp, err := s.client.getThing(ctx, path, nil)
	if err != nil {
		return nil, resp, err
	}
	settings, _ := t.WikiPageSettings()
	return settings, resp, nil
}

// UpdateSettings updates the subreddit's wiki page's settings.
func (s *WikiService) UpdateSettings(ctx context.Context, subreddit, page string, updateRequest *WikiPageSettingsUpdateRequest) (*WikiPageSettings, *Response, error) {
	if updateRequest == nil {
		return nil, nil, errors.New("*WikiPageSettingsUpdateRequest: cannot be nil")
	}

	form, err := query.Values(updateRequest)
	if err != nil {
		return nil, nil, err
	}

	path := fmt.Sprintf("r/%s/wiki/settings/%s", subreddit, page)
	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, nil, err
	}

	root := new(thing)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	settings, _ := root.WikiPageSettings()
	return settings, resp, nil
}

// Discussions gets a list of discussions (posts) about the wiki page.
func (s *WikiService) Discussions(ctx context.Context, subreddit, page string, opts *ListOptions) ([]*Post, *Response, error) {
	path := fmt.Sprintf("r/%s/wiki/discussions/%s", subreddit, page)
	l, resp, err := s.client.getListing(ctx, path, opts)
	if err != nil {
		return nil, resp, err
	}
	return l.Posts(), resp, nil
}

// ToggleVisibility toggles the public visibility of a wiki page revision.
// The returned bool is whether the page was set to hidden or not.
func (s *WikiService) ToggleVisibility(ctx context.Context, subreddit, page, revisionID string) (bool, *Response, error) {
	path := fmt.Sprintf("r/%s/api/wiki/hide", subreddit)

	form := url.Values{}
	form.Set("page", page)
	form.Set("revision", revisionID)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return false, nil, err
	}

	root := new(struct {
		Status bool `json:"status"`
	})
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return false, resp, err
	}

	return root.Status, resp, nil
}

func (s *WikiService) revisions(ctx context.Context, subreddit, page string, opts *ListOptions) ([]*WikiPageRevision, *Response, error) {
	path := fmt.Sprintf("r/%s/wiki/revisions", subreddit)
	if page != "" {
		path += "/" + page
	}

	if opts != nil {
		const idPrefix = "WikiRevision_"
		if opts.After != "" && !strings.HasPrefix(opts.After, idPrefix) {
			opts.After = idPrefix + opts.After
		}
		if opts.Before != "" && !strings.HasPrefix(opts.Before, idPrefix) {
			opts.Before = idPrefix + opts.Before
		}
	}

	path, err := addOptions(path, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(wikiPageRevisionListing)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.Data.Revisions, resp, nil
}

// Revisions gets revisions of all pages in the wiki.
func (s *WikiService) Revisions(ctx context.Context, subreddit string, opts *ListOptions) ([]*WikiPageRevision, *Response, error) {
	return s.revisions(ctx, subreddit, "", opts)
}

// RevisionsPage gets revisions of the specific wiki page.
// If page is an empty string, it gets revisions of all pages in the wiki.
func (s *WikiService) RevisionsPage(ctx context.Context, subreddit, page string, opts *ListOptions) ([]*WikiPageRevision, *Response, error) {
	return s.revisions(ctx, subreddit, page, opts)
}

// Allow the user to edit the specified wiki page in the subreddit.
func (s *WikiService) Allow(ctx context.Context, subreddit, page, username string) (*Response, error) {
	path := fmt.Sprintf("r/%s/api/wiki/alloweditor/add", subreddit)

	form := url.Values{}
	form.Set("page", page)
	form.Set("username", username)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Deny the user the ability to edit the specified wiki page in the subreddit.
func (s *WikiService) Deny(ctx context.Context, subreddit, page, username string) (*Response, error) {
	path := fmt.Sprintf("r/%s/api/wiki/alloweditor/del", subreddit)

	form := url.Values{}
	form.Set("page", page)
	form.Set("username", username)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}
