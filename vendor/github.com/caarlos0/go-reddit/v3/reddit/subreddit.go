package reddit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-querystring/query"
)

// SubredditService handles communication with the subreddit
// related methods of the Reddit API.
//
// Reddit API docs: https://www.reddit.com/dev/api/#section_subreddits
type SubredditService struct {
	client *Client
}

type rootSubredditNames struct {
	Names []string `json:"names,omitempty"`
}

// Relationship holds information about a relationship (friend/blocked).
type Relationship struct {
	ID      string     `json:"rel_id,omitempty"`
	User    string     `json:"name,omitempty"`
	UserID  string     `json:"id,omitempty"`
	Created *Timestamp `json:"date,omitempty"`
}

// Moderator is a user who moderates a subreddit.
type Moderator struct {
	*Relationship
	Permissions []string `json:"mod_permissions"`
}

// Ban represents a banned relationship.
type Ban struct {
	*Relationship
	// nil means the ban is permanent
	DaysLeft *int   `json:"days_left"`
	Note     string `json:"note,omitempty"`
}

// SubredditRule is a rule in the subreddit.
type SubredditRule struct {
	// One of: comment, link (i.e. post), or all (i.e. both comment and link).
	Kind string `json:"kind,omitempty"`
	// Short description of the rule.
	Name string `json:"short_name,omitempty"`
	// The reason that will appear when a thing is reported in violation to this rule.
	ViolationReason string     `json:"violation_reason,omitempty"`
	Description     string     `json:"description,omitempty"`
	Priority        int        `json:"priority"`
	Created         *Timestamp `json:"created_utc,omitempty"`
}

// SubredditRuleCreateRequest represents a request to add a subreddit rule.
type SubredditRuleCreateRequest struct {
	// One of: comment, link (i.e. post) or all (i.e. both).
	Kind string `url:"kind"`
	// Short description of the rule. No longer than 100 characters.
	Name string `url:"short_name"`
	// The reason that will appear when a thing is reported in violation to this rule.
	// If this is empty, Reddit will set its value to Name by default.
	// No longer than 100 characters.
	ViolationReason string `url:"violation_reason,omitempty"`
	// Optional. No longer than 500 characters.
	Description string `url:"description,omitempty"`
}

func (r *SubredditRuleCreateRequest) validate() error {
	if r == nil {
		return errors.New("*SubredditRuleCreateRequest: cannot be nil")
	}

	switch r.Kind {
	case "comment", "link", "all":
		// intentionally left blank
	default:
		return errors.New("(*SubredditRuleCreateRequest).Kind: must be one of: comment, link, all")
	}

	if r.Name == "" || len(r.Name) > 100 {
		return errors.New("(*SubredditRuleCreateRequest).Name: must be between 1-100 characters")
	}

	if len(r.ViolationReason) > 100 {
		return errors.New("(*SubredditRuleCreateRequest).ViolationReason: cannot be longer than 100 characters")
	}

	if len(r.Description) > 500 {
		return errors.New("(*SubredditRuleCreateRequest).Description: cannot be longer than 500 characters")
	}

	return nil
}

// SubredditTrafficStats hold information about subreddit traffic.
type SubredditTrafficStats struct {
	// Traffic data is returned in the form of day, hour, and month.
	// Start is a timestamp indicating the start of the category, i.e.
	// start of the day for day, start of the hour for hour, and start of the month for month.
	Start       *Timestamp `json:"start"`
	UniqueViews int        `json:"unique_views"`
	TotalViews  int        `json:"total_views"`
	// This is only available for "day" traffic, not hour and month.
	// Therefore, it is always 0 by default for hour and month.
	Subscribers int `json:"subscribers"`
}

// SubredditImage is an image part of the image set of a subreddit.
type SubredditImage struct {
	Name string `json:"name"`
	Link string `json:"link"`
	URL  string `json:"url"`
}

// SubredditStyleSheet contains the subreddit's styling information.
type SubredditStyleSheet struct {
	SubredditID string            `json:"subreddit_id"`
	Images      []*SubredditImage `json:"images"`
	StyleSheet  string            `json:"stylesheet"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (s *SubredditTrafficStats) UnmarshalJSON(b []byte) error {
	var data [4]int
	err := json.Unmarshal(b, &data)
	if err != nil {
		return err
	}

	timestampByteValue, err := json.Marshal(data[0])
	if err != nil {
		return err
	}

	timestamp := new(Timestamp)
	err = timestamp.UnmarshalJSON(timestampByteValue)
	if err != nil {
		return err
	}

	s.Start = timestamp
	s.UniqueViews = data[1]
	s.TotalViews = data[2]
	s.Subscribers = data[3]

	return nil
}

// SubredditSettings are a subreddit's settings.
type SubredditSettings struct {
	// The id of the subreddit.
	ID string `url:"-" json:"subreddit_id,omitempty"`

	// One of: public, restricted, private, gold_restricted, archived, employees_only, gold_only, user.
	Type *string `url:"type,omitempty" json:"subreddit_type,omitempty"`

	// A valid IETF language tag (underscore separated).
	Language *string `url:"lang,omitempty" json:"language,omitempty"`

	// No longer than 100 characters.
	Title *string `url:"title,omitempty" json:"title,omitempty"`
	// Raw markdown text. No longer than 500 characters.
	Description *string `url:"public_description,omitempty" json:"public_description,omitempty"`
	// Raw markdown text. No longer than 10240 characters.
	Sidebar *string `url:"description,omitempty" json:"description,omitempty"`
	// Raw markdown text. No longer than 1024 characters.
	SubmissionText *string `url:"submit_text,omitempty" json:"submit_text,omitempty"`
	// Raw markdown text. No longer than 5000 characters.
	WelcomeMessage        *string `url:"welcome_message_text,omitempty" json:"welcome_message_text,omitempty"`
	WelcomeMessageEnabled *bool   `url:"welcome_message_enabled,omitempty" json:"welcome_message_enabled,omitempty"`

	AllowCrossposts      *bool `url:"allow_post_crossposts,omitempty" json:"allow_post_crossposts,omitempty"`
	AllowChatPosts       *bool `url:"allow_chat_post_creation,omitempty" json:"allow_chat_post_creation,omitempty"`
	AllowPollPosts       *bool `url:"allow_polls,omitempty" json:"allow_polls,omitempty"`
	AllowFreeFormReports *bool `url:"free_form_reports,omitempty" json:"free_form_reports,omitempty"`
	AllowOriginalContent *bool `url:"original_content_tag_enabled,omitempty" json:"original_content_tag_enabled,omitempty"`
	// Allow image uploads and links to image hosting sites.
	AllowImages                *bool `url:"allow_images,omitempty" json:"allow_images,omitempty"`
	AllowMultipleImagesPerPost *bool `url:"allow_galleries,omitempty" json:"allow_galleries,omitempty"`

	ExcludeSitewideBannedUsersContent *bool `url:"exclude_banned_modqueue,omitempty" json:"exclude_banned_modqueue,omitempty"`

	// An integer from 0 to 3.
	CrowdControlChalLevel *int `url:"crowd_control_chat_level,omitempty" json:"crowd_control_chat_level,omitempty"`

	// Mark all posts in this subreddit as Original Content (OC) on the desktop redesign.
	AllOriginalContent *bool `url:"all_original_content,omitempty" json:"all_original_content,omitempty"`

	// One of: none (recommended), confidence, top, new, controversial, old, random, qa, live.
	SuggestedCommentSort *string `url:"suggested_comment_sort,omitempty" json:"suggested_comment_sort,omitempty"`

	// No longer than 60 characters.
	SubmitLinkPostLabel *string `url:"submit_link_label,omitempty" json:"submit_link_label,omitempty"`
	// No longer than 60 characters.
	SubmitTextPostLabel *string `url:"submit_text_label,omitempty" json:"submit_text_label,omitempty"`

	// One of: any, link, self.
	PostType *string `url:"link_type,omitempty" json:"content_options,omitempty"`

	// One of: low (disable most filtering), high (standard), all (filter everything, requiring mod approval).
	SpamFilterStrengthLinkPosts *string `url:"spam_links,omitempty" json:"spam_links,omitempty"`
	// One of: low (disable most filtering), high (standard), all (filter everything, requiring mod approval).
	SpamFilterStrengthTextPosts *string `url:"spam_selfposts,omitempty" json:"spam_selfposts,omitempty"`
	// One of: low (disable most filtering), high (standard), all (filter everything, requiring mod approval).
	SpamFilterStrengthComments *string `url:"spam_comments,omitempty" json:"spam_comments,omitempty"`

	ShowContentThumbnails              *bool `url:"show_media,omitempty" json:"show_media,omitempty"`
	ExpandMediaPreviewsOnCommentsPages *bool `url:"show_media_preview,omitempty" json:"show_media_preview,omitempty"`

	CollapseDeletedComments *bool `url:"collapse_deleted_comments,omitempty" json:"collapse_deleted_comments,omitempty"`
	// An integer between 0 and 1440.
	MinutesToHideCommentScores *int `url:"comment_score_hide_mins,omitempty" json:"comment_score_hide_mins,omitempty"`

	// Enable marking posts as containing spoilers.
	SpoilersEnabled *bool `url:"spoilers_enabled,omitempty" json:"spoilers_enabled,omitempty"`

	// If there's an image header set, hovering the mouse over it will display this text.
	HeaderMouseoverText *string `url:"header-title,omitempty" json:"header_hover_text,omitempty"`

	// 6-digit rgb hex colour, e.g. #AABBCC.
	// Thematic colour for the subreddit on mobile.
	MobileColour *string `url:"key_color,omitempty" json:"key_color,omitempty"`

	// Can only be set to true if subreddit type is gold_only.
	HideAds *bool `url:"hide_ads,omitempty" json:"hide_ads,omitempty"`
	// Require viewers to be over 18 years old.
	NSFW *bool `url:"over_18,omitempty" json:"over_18,omitempty"`

	// Show up in high-traffic feeds: Allow your community to be in r/all, r/popular, and trending lists where it can be
	// seen by the general Reddit population.
	AllowDiscoveryInHighTrafficFeeds *bool `url:"allow_top,omitempty" json:"default_set,omitempty"`
	// Get recommended to individual redditors. Let Reddit recommend your community to people who have similar interests.
	AllowDiscoveryByIndividualUsers *bool `url:"allow_discovery,omitempty" json:"allow_discovery,omitempty"`

	// One of:
	// - disabled: wiki is disabled for everyone except mods.
	// - modonly: only mods, approved wiki contributors, or those on a page's edit list may edit.
	// - anyone: anyone who can submit to the subreddit may edit.
	WikiMode *string `url:"wikimode,omitempty" json:"wikimode,omitempty"`
	// Account age (in days) required to create and edit wiki pages.
	WikiMinimumAccountAge *int `url:"wiki_edit_age,omitempty" json:"wiki_edit_age,omitempty"`
	// Subreddit karma required to create and edit wiki pages.
	WikiMinimumKarma *int `url:"wiki_edit_karma,omitempty" json:"wiki_edit_karma,omitempty"`
}

// SubredditPostRequirements is a list of moderator-designed requirements to post to a subreddit.
type SubredditPostRequirements struct {
	// Sentence or two on how to successfully post to the subreddit.
	Guidelines              string `json:"guidelines_text"`
	GuidelinesDisplayPolicy string `json:"guidelines_display_policy"`

	TitleMinLength int `json:"title_text_min_length"`
	TitleMaxLength int `json:"title_text_max_length"`

	BodyMinLength int `json:"body_text_min_length"`
	BodyMaxLength int `json:"body_text_max_length"`

	// Do not allow any of these words in the title.
	TitleBlacklistedStrings []string `json:"title_blacklisted_strings"`
	// Do not allow any of these words in the body.
	BodyBlacklistedStrings []string `json:"body_blacklisted_strings"`

	// Require at least one of these words in the title.
	TitleRequiredStrings []string `json:"title_required_strings"`
	// Require at least one of these words in the body.
	BodyRequiredStrings []string `json:"body_required_strings"`

	// Block link posts with these domains.
	DomainBlacklist []string `json:"domain_blacklist"`
	// Only allow link posts with these domains.
	DomainWhitelist []string `json:"domain_whitelist"`

	// One of: required, notAllowed, none.
	// Meaning that a selftext post's body is required, not allowed, or optional, respectively.
	BodyRestrictionPolicy string `json:"body_restriction_policy"`
	LinkRestrictionPolicy string `json:"link_restriction_policy"`

	GalleryMinItems            int    `json:"gallery_min_items"`
	GalleryMaxItems            int    `json:"gallery_max_items"`
	GalleryCaptionsRequirement string `json:"gallery_captions_requirement"`
	GalleryURLsRequirement     string `json:"gallery_urls_requirement"`

	// Prevent users from posting a link that was already posted to your community within x days of the original.
	LinkRepostAge int  `json:"link_repost_age"`
	FlairRequired bool `json:"is_flair_required"`

	TitleRegexes []string `json:"title_regexes"`
	BodyRegexes  []string `json:"body_regexes"`
}

func (s *SubredditService) getPosts(ctx context.Context, sort string, subreddit string, opts interface{}) ([]*Post, *Response, error) {
	path := sort
	if subreddit != "" {
		path = fmt.Sprintf("r/%s/%s", subreddit, sort)
	}
	l, resp, err := s.client.getListing(ctx, path, opts)
	if err != nil {
		return nil, resp, err
	}
	return l.Posts(), resp, nil
}

// HotPosts returns the hottest posts from the specified subreddit.
// To search through multiple, separate the names with a plus (+), e.g. "golang+test".
// If none are defined, it returns the ones from your subscribed subreddits.
// To search through all, just specify "all".
// To search through all and filter out subreddits, provide "all-name1-name2".
// Note: when looking for hot posts in a subreddit, it will include the stickied
// posts (if any) PLUS posts from the limit parameter (25 by default).
func (s *SubredditService) HotPosts(ctx context.Context, subreddit string, opts *ListOptions) ([]*Post, *Response, error) {
	return s.getPosts(ctx, "hot", subreddit, opts)
}

// NewPosts returns the newest posts from the specified subreddit.
// To search through multiple, separate the names with a plus (+), e.g. "golang+test".
// If none are defined, it returns the ones from your subscribed subreddits.
// To search through all, just specify "all".
// To search through all and filter out subreddits, provide "all-name1-name2".
func (s *SubredditService) NewPosts(ctx context.Context, subreddit string, opts *ListOptions) ([]*Post, *Response, error) {
	return s.getPosts(ctx, "new", subreddit, opts)
}

// RisingPosts returns the rising posts from the specified subreddit.
// To search through multiple, separate the names with a plus (+), e.g. "golang+test".
// If none are defined, it returns the ones from your subscribed subreddits.
// To search through all, just specify "all".
// To search through all and filter out subreddits, provide "all-name1-name2".
func (s *SubredditService) RisingPosts(ctx context.Context, subreddit string, opts *ListOptions) ([]*Post, *Response, error) {
	return s.getPosts(ctx, "rising", subreddit, opts)
}

// ControversialPosts returns the most controversial posts from the specified subreddit.
// To search through multiple, separate the names with a plus (+), e.g. "golang+test".
// If none are defined, it returns the ones from your subscribed subreddits.
// To search through all, just specify "all".
// To search through all and filter out subreddits, provide "all-name1-name2".
func (s *SubredditService) ControversialPosts(ctx context.Context, subreddit string, opts *ListPostOptions) ([]*Post, *Response, error) {
	return s.getPosts(ctx, "controversial", subreddit, opts)
}

// TopPosts returns the top posts from the specified subreddit.
// To search through multiple, separate the names with a plus (+), e.g. "golang+test".
// If none are defined, it returns the ones from your subscribed subreddits.
// To search through all, just specify "all".
// To search through all and filter out subreddits, provide "all-name1-name2".
func (s *SubredditService) TopPosts(ctx context.Context, subreddit string, opts *ListPostOptions) ([]*Post, *Response, error) {
	return s.getPosts(ctx, "top", subreddit, opts)
}

// Get a subreddit by name.
func (s *SubredditService) Get(ctx context.Context, name string) (*Subreddit, *Response, error) {
	if name == "" {
		return nil, nil, errors.New("name: cannot be empty")
	}

	path := fmt.Sprintf("r/%s/about", name)
	t, resp, err := s.client.getThing(ctx, path, nil)
	if err != nil {
		return nil, resp, err
	}

	sr, _ := t.Subreddit()
	return sr, resp, nil
}

// Popular returns popular subreddits.
func (s *SubredditService) Popular(ctx context.Context, opts *ListSubredditOptions) ([]*Subreddit, *Response, error) {
	return s.getSubreddits(ctx, "subreddits/popular", opts)
}

// New returns new subreddits.
func (s *SubredditService) New(ctx context.Context, opts *ListSubredditOptions) ([]*Subreddit, *Response, error) {
	return s.getSubreddits(ctx, "subreddits/new", opts)
}

// Gold returns gold subreddits (i.e. only accessible to users with gold).
// It seems like it returns an empty list if you don't have gold.
func (s *SubredditService) Gold(ctx context.Context, opts *ListSubredditOptions) ([]*Subreddit, *Response, error) {
	return s.getSubreddits(ctx, "subreddits/gold", opts)
}

// Default returns default subreddits.
func (s *SubredditService) Default(ctx context.Context, opts *ListSubredditOptions) ([]*Subreddit, *Response, error) {
	return s.getSubreddits(ctx, "subreddits/default", opts)
}

// Subscribed returns the list of subreddits you are subscribed to.
func (s *SubredditService) Subscribed(ctx context.Context, opts *ListSubredditOptions) ([]*Subreddit, *Response, error) {
	return s.getSubreddits(ctx, "subreddits/mine/subscriber", opts)
}

// Approved returns the list of subreddits you are an approved user in.
func (s *SubredditService) Approved(ctx context.Context, opts *ListSubredditOptions) ([]*Subreddit, *Response, error) {
	return s.getSubreddits(ctx, "subreddits/mine/contributor", opts)
}

// Moderated returns the list of subreddits you are a moderator of.
func (s *SubredditService) Moderated(ctx context.Context, opts *ListSubredditOptions) ([]*Subreddit, *Response, error) {
	return s.getSubreddits(ctx, "subreddits/mine/moderator", opts)
}

// GetSticky1 returns the first stickied post on a subreddit (if it exists).
func (s *SubredditService) GetSticky1(ctx context.Context, subreddit string) (*PostAndComments, *Response, error) {
	return s.getSticky(ctx, subreddit, 1)
}

// GetSticky2 returns the second stickied post on a subreddit (if it exists).
func (s *SubredditService) GetSticky2(ctx context.Context, subreddit string) (*PostAndComments, *Response, error) {
	return s.getSticky(ctx, subreddit, 2)
}

func (s *SubredditService) handleSubscription(ctx context.Context, form url.Values) (*Response, error) {
	path := "api/subscribe"
	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

// Subscribe subscribes to subreddits based on their names.
func (s *SubredditService) Subscribe(ctx context.Context, subreddits ...string) (*Response, error) {
	form := url.Values{}
	form.Set("action", "sub")
	form.Set("sr_name", strings.Join(subreddits, ","))
	return s.handleSubscription(ctx, form)
}

// SubscribeByID subscribes to subreddits based on their id.
func (s *SubredditService) SubscribeByID(ctx context.Context, ids ...string) (*Response, error) {
	form := url.Values{}
	form.Set("action", "sub")
	form.Set("sr", strings.Join(ids, ","))
	return s.handleSubscription(ctx, form)
}

// Unsubscribe unsubscribes from subreddits based on their names.
func (s *SubredditService) Unsubscribe(ctx context.Context, subreddits ...string) (*Response, error) {
	form := url.Values{}
	form.Set("action", "unsub")
	form.Set("sr_name", strings.Join(subreddits, ","))
	return s.handleSubscription(ctx, form)
}

// UnsubscribeByID unsubscribes from subreddits based on their id.
func (s *SubredditService) UnsubscribeByID(ctx context.Context, ids ...string) (*Response, error) {
	form := url.Values{}
	form.Set("action", "unsub")
	form.Set("sr", strings.Join(ids, ","))
	return s.handleSubscription(ctx, form)
}

// Favorite the subreddit.
func (s *SubredditService) Favorite(ctx context.Context, subreddit string) (*Response, error) {
	path := "api/favorite"

	form := url.Values{}
	form.Set("sr_name", subreddit)
	form.Set("make_favorite", "true")
	form.Set("api_type", "json")

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Unfavorite the subreddit.
func (s *SubredditService) Unfavorite(ctx context.Context, subreddit string) (*Response, error) {
	path := "api/favorite"

	form := url.Values{}
	form.Set("sr_name", subreddit)
	form.Set("make_favorite", "false")
	form.Set("api_type", "json")

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Search for subreddits.
func (s *SubredditService) Search(ctx context.Context, query string, opts *ListSubredditOptions) ([]*Subreddit, *Response, error) {
	path := fmt.Sprintf("subreddits/search?q=%s", query)
	l, resp, err := s.client.getListing(ctx, path, opts)
	if err != nil {
		return nil, resp, err
	}
	return l.Subreddits(), resp, nil
}

// SearchNames searches for subreddits with names beginning with the query provided.
func (s *SubredditService) SearchNames(ctx context.Context, query string) ([]string, *Response, error) {
	path := fmt.Sprintf("api/search_reddit_names?query=%s", query)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(rootSubredditNames)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.Names, resp, nil
}

// SearchPosts searches for posts in the specified subreddit.
// To search through multiple, separate the names with a plus (+), e.g. "golang+test".
// If no subreddit is provided, the search is run against r/all.
func (s *SubredditService) SearchPosts(ctx context.Context, query string, subreddit string, opts *ListPostSearchOptions) ([]*Post, *Response, error) {
	if subreddit == "" {
		subreddit = "all"
	}

	path := fmt.Sprintf("r/%s/search", subreddit)
	path, err := addOptions(path, opts)
	if err != nil {
		return nil, nil, err
	}

	notAll := !strings.EqualFold(subreddit, "all")

	params := struct {
		Query              string `url:"q"`
		RestrictSubreddits bool   `url:"restrict_sr,omitempty"`
	}{query, notAll}

	t, resp, err := s.client.getThing(ctx, path, params)
	if err != nil {
		return nil, resp, err
	}

	l, _ := t.Listing()
	return l.Posts(), resp, nil
}

func (s *SubredditService) getSubreddits(ctx context.Context, path string, opts *ListSubredditOptions) ([]*Subreddit, *Response, error) {
	l, resp, err := s.client.getListing(ctx, path, opts)
	if err != nil {
		return nil, resp, err
	}
	return l.Subreddits(), resp, nil
}

// getSticky returns one of the 2 stickied posts of the subreddit (if they exist).
// Num should be equal to 1 or 2, depending on which one you want.
func (s *SubredditService) getSticky(ctx context.Context, subreddit string, num int) (*PostAndComments, *Response, error) {
	params := struct {
		Num int `url:"num"`
	}{num}

	path := fmt.Sprintf("r/%s/about/sticky", subreddit)
	path, err := addOptions(path, params)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(PostAndComments)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root, resp, nil
}

// todo: sr_detail's NSFW indicator is over_18 instead of over18
func (s *SubredditService) random(ctx context.Context, nsfw bool) (*Subreddit, *Response, error) {
	path := "r/random"
	if nsfw {
		path = "r/randnsfw"
	}

	params := struct {
		ExpandSubreddit bool `url:"sr_detail"`
		Limit           int  `url:"limit,omitempty"`
	}{true, 1}

	path, err := addOptions(path, params)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(struct {
		Data struct {
			Children [1]struct {
				Data struct {
					Subreddit *Subreddit `json:"sr_detail"`
				} `json:"data"`
			} `json:"children"`
		} `json:"data"`
	})
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	subreddit := root.Data.Children[0].Data.Subreddit
	return subreddit, resp, nil
}

// Random returns a random SFW subreddit.
func (s *SubredditService) Random(ctx context.Context) (*Subreddit, *Response, error) {
	return s.random(ctx, false)
}

// RandomNSFW returns a random NSFW subreddit.
func (s *SubredditService) RandomNSFW(ctx context.Context) (*Subreddit, *Response, error) {
	return s.random(ctx, true)
}

// SubmissionText gets the submission text for the subreddit.
// This text is set by the subreddit moderators and intended to be displayed on the submission form.
func (s *SubredditService) SubmissionText(ctx context.Context, name string) (string, *Response, error) {
	if name == "" {
		return "", nil, errors.New("name: cannot be empty")
	}

	path := fmt.Sprintf("r/%s/api/submit_text", name)
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return "", nil, err
	}

	root := new(struct {
		Text string `json:"submit_text"`
	})
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return "", resp, err
	}

	return root.Text, resp, err
}

// Banned gets banned users from the subreddit.
func (s *SubredditService) Banned(ctx context.Context, subreddit string, opts *ListOptions) ([]*Ban, *Response, error) {
	path := fmt.Sprintf("r/%s/about/banned", subreddit)

	path, err := addOptions(path, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(struct {
		Data struct {
			Bans  []*Ban `json:"children"`
			After string `json:"after"`
		} `json:"data"`
	})
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	resp.After = root.Data.After
	return root.Data.Bans, resp, nil
}

// Muted gets muted users from the subreddit.
func (s *SubredditService) Muted(ctx context.Context, subreddit string, opts *ListOptions) ([]*Relationship, *Response, error) {
	path := fmt.Sprintf("r/%s/about/muted", subreddit)

	path, err := addOptions(path, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(struct {
		Data struct {
			Relationships []*Relationship `json:"children"`
			After         string          `json:"after"`
		} `json:"data"`
	})
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	resp.After = root.Data.After
	return root.Data.Relationships, resp, nil
}

// WikiBanned gets banned users from the subreddit.
func (s *SubredditService) WikiBanned(ctx context.Context, subreddit string, opts *ListOptions) ([]*Ban, *Response, error) {
	path := fmt.Sprintf("r/%s/about/wikibanned", subreddit)

	path, err := addOptions(path, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(struct {
		Data struct {
			Bans  []*Ban `json:"children"`
			After string `json:"after"`
		} `json:"data"`
	})
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	resp.After = root.Data.After
	return root.Data.Bans, resp, nil
}

// Contributors gets contributors (also known as approved users) from the subreddit.
func (s *SubredditService) Contributors(ctx context.Context, subreddit string, opts *ListOptions) ([]*Relationship, *Response, error) {
	path := fmt.Sprintf("r/%s/about/contributors", subreddit)

	path, err := addOptions(path, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(struct {
		Data struct {
			Relationships []*Relationship `json:"children"`
			After         string          `json:"after"`
		} `json:"data"`
	})
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	resp.After = root.Data.After
	return root.Data.Relationships, resp, nil
}

// WikiContributors gets contributors of the wiki from the subreddit.
func (s *SubredditService) WikiContributors(ctx context.Context, subreddit string, opts *ListOptions) ([]*Relationship, *Response, error) {
	path := fmt.Sprintf("r/%s/about/wikicontributors", subreddit)

	path, err := addOptions(path, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(struct {
		Data struct {
			Relationships []*Relationship `json:"children"`
			After         string          `json:"after"`
		} `json:"data"`
	})
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	resp.After = root.Data.After
	return root.Data.Relationships, resp, nil
}

// Moderators gets the moderators of the subreddit.
func (s *SubredditService) Moderators(ctx context.Context, subreddit string) ([]*Moderator, *Response, error) {
	path := fmt.Sprintf("r/%s/about/moderators", subreddit)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(struct {
		Data struct {
			Moderators []*Moderator `json:"children"`
		} `json:"data"`
	})
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.Data.Moderators, resp, nil
}

// Rules gets the rules of the subreddit.
func (s *SubredditService) Rules(ctx context.Context, subreddit string) ([]*SubredditRule, *Response, error) {
	path := fmt.Sprintf("r/%s/about/rules", subreddit)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(struct {
		Rules []*SubredditRule `json:"rules"`
	})
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.Rules, resp, nil
}

// CreateRule adds a rule to the subreddit.
func (s *SubredditService) CreateRule(ctx context.Context, subreddit string, request *SubredditRuleCreateRequest) (*Response, error) {
	err := request.validate()
	if err != nil {
		return nil, err
	}

	form, err := query.Values(request)
	if err != nil {
		return nil, err
	}
	form.Set("api_type", "json")

	path := fmt.Sprintf("r/%s/api/add_subreddit_rule", subreddit)
	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Traffic gets the traffic data of the subreddit.
// It returns traffic data by day, hour, and month, respectively.
func (s *SubredditService) Traffic(ctx context.Context, subreddit string) ([]*SubredditTrafficStats, []*SubredditTrafficStats, []*SubredditTrafficStats, *Response, error) {
	path := fmt.Sprintf("r/%s/about/traffic", subreddit)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	root := new(struct {
		Day   []*SubredditTrafficStats `json:"day"`
		Hour  []*SubredditTrafficStats `json:"hour"`
		Month []*SubredditTrafficStats `json:"month"`
	})
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, nil, nil, resp, err
	}

	return root.Day, root.Hour, root.Month, resp, nil
}

// StyleSheet returns the subreddit's style sheet, as well as some information about images.
func (s *SubredditService) StyleSheet(ctx context.Context, subreddit string) (*SubredditStyleSheet, *Response, error) {
	path := fmt.Sprintf("r/%s/about/stylesheet", subreddit)
	t, resp, err := s.client.getThing(ctx, path, nil)
	if err != nil {
		return nil, resp, err
	}
	styleSheet, _ := t.StyleSheet()
	return styleSheet, resp, nil
}

// StyleSheetRaw returns the subreddit's style sheet with all comments and newlines stripped.
func (s *SubredditService) StyleSheetRaw(ctx context.Context, subreddit string) (string, *Response, error) {
	path := fmt.Sprintf("r/%s/stylesheet", subreddit)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return "", nil, err
	}

	buf := new(bytes.Buffer)
	resp, err := s.client.Do(ctx, req, buf)
	if err != nil {
		return "", resp, err
	}

	return buf.String(), resp, nil
}

// UpdateStyleSheet updates the style sheet of the subreddit.
// Providing a reason is optional.
func (s *SubredditService) UpdateStyleSheet(ctx context.Context, subreddit, styleSheet, reason string) (*Response, error) {
	path := fmt.Sprintf("r/%s/api/subreddit_stylesheet", subreddit)

	form := url.Values{}
	form.Set("api_type", "json")
	form.Set("op", "save")
	form.Set("stylesheet_contents", styleSheet)
	if reason != "" {
		form.Set("reason", reason)
	}

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// RemoveImage removes an image from the subreddit's custom image set.
// The call succeeds even if the named image does not exist.
func (s *SubredditService) RemoveImage(ctx context.Context, subreddit, imageName string) (*Response, error) {
	path := fmt.Sprintf("r/%s/api/delete_sr_img", subreddit)

	form := url.Values{}
	form.Set("api_type", "json")
	form.Set("img_name", imageName)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// RemoveHeader removes the subreddit's current header image.
// The call succeeds even if there's no header image.
func (s *SubredditService) RemoveHeader(ctx context.Context, subreddit string) (*Response, error) {
	path := fmt.Sprintf("r/%s/api/delete_sr_header", subreddit)

	form := url.Values{}
	form.Set("api_type", "json")

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// RemoveMobileHeader removes the subreddit's current mobile header.
// The call succeeds even if there's no mobile header.
func (s *SubredditService) RemoveMobileHeader(ctx context.Context, subreddit string) (*Response, error) {
	path := fmt.Sprintf("r/%s/api/delete_sr_banner", subreddit)

	form := url.Values{}
	form.Set("api_type", "json")

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// RemoveMobileIcon removes the subreddit's current mobile icon.
// The call succeeds even if there's no mobile icon.
func (s *SubredditService) RemoveMobileIcon(ctx context.Context, subreddit string) (*Response, error) {
	path := fmt.Sprintf("r/%s/api/delete_sr_icon", subreddit)

	form := url.Values{}
	form.Set("api_type", "json")

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

func (s *SubredditService) uploadImage(ctx context.Context, subreddit, imagePath, imageType, imageName string) (string, *Response, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return "", nil, err
	}
	defer file.Close()

	form := url.Values{}
	form.Set("upload_type", imageType)
	form.Set("name", imageName)
	form.Set("img_type", "png")

	ext := filepath.Ext(file.Name())
	if strings.EqualFold(ext, ".jpg") {
		form.Set("img_type", "jpg")
	}

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	for k := range form {
		writer.WriteField(k, form.Get(k))
	}

	part, err := writer.CreateFormFile("file", file.Name())
	if err != nil {
		return "", nil, err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return "", nil, err
	}

	err = writer.Close()
	if err != nil {
		return "", nil, err
	}

	path := fmt.Sprintf("r/%s/api/upload_sr_img", subreddit)
	u, err := s.client.BaseURL.Parse(path)
	if err != nil {
		return "", nil, err
	}

	req, err := http.NewRequest(http.MethodPost, u.String(), body)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set(headerContentType, writer.FormDataContentType())

	root := new(struct {
		Errors      []string `json:"errors"`
		ErrorValues []string `json:"errors_values"`
		ImageSource string   `json:"img_src"`
	})
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return "", resp, err
	}

	if len(root.ErrorValues) > 0 {
		err = fmt.Errorf("could not upload image: %s", strings.Join(root.ErrorValues, "; "))
		return "", resp, err
	}

	return root.ImageSource, resp, nil
}

// UploadImage uploads an image to the subreddit.
// If an image with the image name already exists, it it replaced.
// A successful call returns a link to the uploaded image.
func (s *SubredditService) UploadImage(ctx context.Context, subreddit, imagePath, imageName string) (string, *Response, error) {
	return s.uploadImage(ctx, subreddit, imagePath, "img", imageName)
}

// UploadHeader uploads an image to be user as the subreddit's header image.
// A successful call returns a link to the uploaded image.
func (s *SubredditService) UploadHeader(ctx context.Context, subreddit, imagePath, imageName string) (string, *Response, error) {
	return s.uploadImage(ctx, subreddit, imagePath, "header", imageName)
}

// UploadMobileHeader uploads an image to be user as the subreddit's mobile header image.
// A successful call returns a link to the uploaded image.
func (s *SubredditService) UploadMobileHeader(ctx context.Context, subreddit, imagePath, imageName string) (string, *Response, error) {
	return s.uploadImage(ctx, subreddit, imagePath, "banner", imageName)
}

// UploadMobileIcon uploads an image to be user as the subreddit's mobile icon.
// A successful call returns a link to the uploaded image.
func (s *SubredditService) UploadMobileIcon(ctx context.Context, subreddit, imagePath, imageName string) (string, *Response, error) {
	return s.uploadImage(ctx, subreddit, imagePath, "icon", imageName)
}

// Create a subreddit.
func (s *SubredditService) Create(ctx context.Context, name string, request *SubredditSettings) (*Response, error) {
	if request == nil {
		return nil, errors.New("*SubredditSettings: cannot be nil")
	}

	form, err := query.Values(request)
	if err != nil {
		return nil, err
	}
	form.Set("name", name)
	form.Set("api_type", "json")

	path := "api/site_admin"
	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Edit a subreddit.
// This endpoint expects all values of the request to be provided.
// To make this easier, it might be useful to get the subreddit's current settings via GetSettings(),
// and use that as a starting point. It even returns the subreddit's ID, which is needed for this.
func (s *SubredditService) Edit(ctx context.Context, subredditID string, request *SubredditSettings) (*Response, error) {
	if request == nil {
		return nil, errors.New("*SubredditSettings: cannot be nil")
	}

	form, err := query.Values(request)
	if err != nil {
		return nil, err
	}
	form.Set("sr", subredditID)
	form.Set("api_type", "json")

	path := "api/site_admin"
	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// GetSettings gets the settings of a subreddit.
func (s *SubredditService) GetSettings(ctx context.Context, subreddit string) (*SubredditSettings, *Response, error) {
	path := fmt.Sprintf("r/%s/about/edit", subreddit)
	t, resp, err := s.client.getThing(ctx, path, nil)
	if err != nil {
		return nil, resp, err
	}
	settings, _ := t.SubredditSettings()
	return settings, resp, nil
}

// PostRequirements returns the subreddit's moderator-designed requirements to post to it.
// Clients may use the values returned by this method to pre-validate submissions to the subreddit.
func (s *SubredditService) PostRequirements(ctx context.Context, subreddit string) (*SubredditPostRequirements, *Response, error) {
	path := fmt.Sprintf("api/v1/%s/post_requirements", subreddit)
	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	root := new(SubredditPostRequirements)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root, resp, nil
}
