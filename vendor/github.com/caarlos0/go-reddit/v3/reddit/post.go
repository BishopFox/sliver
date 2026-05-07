package reddit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/go-querystring/query"
)

// PostService handles communication with the post
// related methods of the Reddit API.
//
// Reddit API docs: https://www.reddit.com/dev/api/#section_links_and_comments
type PostService struct {
	*postAndCommentService
	client *Client
}

type rootSubmittedPost struct {
	JSON struct {
		Data *Submitted `json:"data,omitempty"`
	} `json:"json"`
}

// Submitted is a newly submitted post on Reddit.
type Submitted struct {
	ID     string `json:"id,omitempty"`
	FullID string `json:"name,omitempty"`
	URL    string `json:"url,omitempty"`
}

// SubmitTextRequest are options used for text posts.
type SubmitTextRequest struct {
	Subreddit string `url:"sr,omitempty"`
	Title     string `url:"title,omitempty"`
	Text      string `url:"text,omitempty"`

	FlairID   string `url:"flair_id,omitempty"`
	FlairText string `url:"flair_text,omitempty"`

	SendReplies *bool `url:"sendreplies,omitempty"`
	NSFW        bool  `url:"nsfw,omitempty"`
	Spoiler     bool  `url:"spoiler,omitempty"`
}

// SubmitLinkRequest are options used for link posts.
type SubmitLinkRequest struct {
	Subreddit string `url:"sr,omitempty"`
	Title     string `url:"title,omitempty"`
	URL       string `url:"url,omitempty"`

	FlairID   string `url:"flair_id,omitempty"`
	FlairText string `url:"flair_text,omitempty"`

	SendReplies *bool `url:"sendreplies,omitempty"`
	Resubmit    bool  `url:"resubmit,omitempty"`
	NSFW        bool  `url:"nsfw,omitempty"`
	Spoiler     bool  `url:"spoiler,omitempty"`
}

// Get a post with its comments.
// id is the ID36 of the post, not its full id.
// Example: instead of t3_abc123, use abc123.
func (s *PostService) Get(ctx context.Context, id string) (*PostAndComments, *Response, error) {
	path := fmt.Sprintf("comments/%s", id)
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

// Duplicates returns the post with the id, and a list of its duplicates.
// id is the ID36 of the post, not its full id.
// Example: instead of t3_abc123, use abc123.
func (s *PostService) Duplicates(ctx context.Context, id string, opts *ListDuplicatePostOptions) (*Post, []*Post, *Response, error) {
	path := fmt.Sprintf("duplicates/%s", id)
	path, err := addOptions(path, opts)
	if err != nil {
		return nil, nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	var root [2]thing
	resp, err := s.client.Do(ctx, req, &root)
	if err != nil {
		return nil, nil, resp, err
	}

	listing1, _ := root[0].Listing()
	listing2, _ := root[1].Listing()

	post := listing1.Posts()[0]
	duplicates := listing2.Posts()

	resp.After = listing2.After()
	return post, duplicates, resp, nil
}

func (s *PostService) submit(ctx context.Context, v interface{}) (*Submitted, *Response, error) {
	path := "api/submit"

	form, err := query.Values(v)
	if err != nil {
		return nil, nil, err
	}
	form.Set("api_type", "json")

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, nil, err
	}

	root := new(rootSubmittedPost)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root.JSON.Data, resp, nil
}

// SubmitText submits a text post.
func (s *PostService) SubmitText(ctx context.Context, opts SubmitTextRequest) (*Submitted, *Response, error) {
	form := struct {
		SubmitTextRequest
		Kind string `url:"kind,omitempty"`
	}{opts, "self"}
	return s.submit(ctx, form)
}

// SubmitLink submits a link post.
func (s *PostService) SubmitLink(ctx context.Context, opts SubmitLinkRequest) (*Submitted, *Response, error) {
	form := struct {
		SubmitLinkRequest
		Kind string `url:"kind,omitempty"`
	}{opts, "link"}
	return s.submit(ctx, form)
}

// Edit a post.
func (s *PostService) Edit(ctx context.Context, id string, text string) (*Post, *Response, error) {
	path := "api/editusertext"

	form := url.Values{}
	form.Set("api_type", "json")
	form.Set("return_rtjson", "true")
	form.Set("thing_id", id)
	form.Set("text", text)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, nil, err
	}

	root := new(Post)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root, resp, nil
}

// Hide posts.
func (s *PostService) Hide(ctx context.Context, ids ...string) (*Response, error) {
	if len(ids) == 0 {
		return nil, errors.New("must provide at least 1 id")
	}

	path := "api/hide"

	form := url.Values{}
	form.Set("id", strings.Join(ids, ","))

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Unhide posts.
func (s *PostService) Unhide(ctx context.Context, ids ...string) (*Response, error) {
	if len(ids) == 0 {
		return nil, errors.New("must provide at least 1 id")
	}

	path := "api/unhide"

	form := url.Values{}
	form.Set("id", strings.Join(ids, ","))

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// MarkNSFW marks a post as NSFW.
func (s *PostService) MarkNSFW(ctx context.Context, id string) (*Response, error) {
	path := "api/marknsfw"

	form := url.Values{}
	form.Set("id", id)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// UnmarkNSFW unmarks a post as NSFW.
func (s *PostService) UnmarkNSFW(ctx context.Context, id string) (*Response, error) {
	path := "api/unmarknsfw"

	form := url.Values{}
	form.Set("id", id)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Spoiler marks a post as a spoiler.
func (s *PostService) Spoiler(ctx context.Context, id string) (*Response, error) {
	path := "api/spoiler"

	form := url.Values{}
	form.Set("id", id)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Unspoiler unmarks a post as a spoiler.
func (s *PostService) Unspoiler(ctx context.Context, id string) (*Response, error) {
	path := "api/unspoiler"

	form := url.Values{}
	form.Set("id", id)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Sticky a post in its subreddit.
// When bottom is true, the post will be set as the bottom sticky (the 2nd one).
// If no top sticky exists, the post will become the top sticky regardless.
// When attempting to sticky a post that's already stickied, it will return a 409 Conflict error.
func (s *PostService) Sticky(ctx context.Context, id string, bottom bool) (*Response, error) {
	path := "api/set_subreddit_sticky"

	form := url.Values{}
	form.Set("api_type", "json")
	form.Set("id", id)
	form.Set("state", "true")
	if !bottom {
		form.Set("num", "1")
	}

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Unsticky unstickies a post in its subreddit.
func (s *PostService) Unsticky(ctx context.Context, id string) (*Response, error) {
	path := "api/set_subreddit_sticky"

	form := url.Values{}
	form.Set("api_type", "json")
	form.Set("id", id)
	form.Set("state", "false")

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// PinToProfile pins one of your posts to your profile.
// TODO: very inconsistent behaviour, not sure I'm ready to include this parameter yet.
// The pos parameter should be a number between 1-4 (inclusive), indicating the position at which
// the post should appear on your profile.
// Note: The position will be bumped upward if there's space. E.g. if you only have 1 pinned post,
// and you try to pin another post to position 3, it will be pinned at 2.
// When attempting to pin a post that's already pinned, it will return a 409 Conflict error.
func (s *PostService) PinToProfile(ctx context.Context, id string) (*Response, error) {
	path := "api/set_subreddit_sticky"

	// if pos < 1 {
	// 	pos = 1
	// }
	// if pos > 4 {
	// 	pos = 4
	// }

	form := url.Values{}
	form.Set("api_type", "json")
	form.Set("id", id)
	form.Set("state", "true")
	form.Set("to_profile", "true")
	// form.Set("num", strconv.Itoa(pos))

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// UnpinFromProfile unpins one of your posts from your profile.
func (s *PostService) UnpinFromProfile(ctx context.Context, id string) (*Response, error) {
	path := "api/set_subreddit_sticky"

	form := url.Values{}
	form.Set("api_type", "json")
	form.Set("id", id)
	form.Set("state", "false")
	form.Set("to_profile", "true")

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// setSuggestedSort sets the suggested comment sort for the post.
// sort must be one of: confidence (i.e. best), top, new, controversial, old, random, qa, live
func (s *PostService) setSuggestedSort(ctx context.Context, id string, sort string) (*Response, error) {
	path := "api/set_suggested_sort"

	form := url.Values{}
	form.Set("api_type", "json")
	form.Set("id", id)
	form.Set("sort", sort)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// SetSuggestedSortBest sets the suggested comment sort for the post to best.
func (s *PostService) SetSuggestedSortBest(ctx context.Context, id string) (*Response, error) {
	return s.setSuggestedSort(ctx, id, "confidence")
}

// SetSuggestedSortTop sets the suggested comment sort for the post to top.
func (s *PostService) SetSuggestedSortTop(ctx context.Context, id string) (*Response, error) {
	return s.setSuggestedSort(ctx, id, "top")
}

// SetSuggestedSortNew sets the suggested comment sort for the post to new.
func (s *PostService) SetSuggestedSortNew(ctx context.Context, id string) (*Response, error) {
	return s.setSuggestedSort(ctx, id, "new")
}

// SetSuggestedSortControversial sets the suggested comment sort for the post to controversial.
func (s *PostService) SetSuggestedSortControversial(ctx context.Context, id string) (*Response, error) {
	return s.setSuggestedSort(ctx, id, "controversial")
}

// SetSuggestedSortOld sorts the comments on the posts randomly.
func (s *PostService) SetSuggestedSortOld(ctx context.Context, id string) (*Response, error) {
	return s.setSuggestedSort(ctx, id, "old")
}

// SetSuggestedSortRandom sets the suggested comment sort for the post to random.
func (s *PostService) SetSuggestedSortRandom(ctx context.Context, id string) (*Response, error) {
	return s.setSuggestedSort(ctx, id, "random")
}

// SetSuggestedSortAMA sets the suggested comment sort for the post to a Q&A styled fashion.
func (s *PostService) SetSuggestedSortAMA(ctx context.Context, id string) (*Response, error) {
	return s.setSuggestedSort(ctx, id, "qa")
}

// SetSuggestedSortLive sets the suggested comment sort for the post to stream new comments as they're posted.
// As of now, this is still in beta, so it's not a fully developed feature yet. It just sets the sort as "new" for now.
func (s *PostService) SetSuggestedSortLive(ctx context.Context, id string) (*Response, error) {
	return s.setSuggestedSort(ctx, id, "live")
}

// ClearSuggestedSort clears the suggested comment sort for the post.
func (s *PostService) ClearSuggestedSort(ctx context.Context, id string) (*Response, error) {
	return s.setSuggestedSort(ctx, id, "")
}

// EnableContestMode enables contest mode for the post.
// Comments will be sorted randomly and regular users cannot see comment scores.
func (s *PostService) EnableContestMode(ctx context.Context, id string) (*Response, error) {
	path := "api/set_contest_mode"

	form := url.Values{}
	form.Set("api_type", "json")
	form.Set("id", id)
	form.Set("state", "true")

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// DisableContestMode disables contest mode for the post.
func (s *PostService) DisableContestMode(ctx context.Context, id string) (*Response, error) {
	path := "api/set_contest_mode"

	form := url.Values{}
	form.Set("api_type", "json")
	form.Set("id", id)
	form.Set("state", "false")

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// LoadMoreComments retrieves more comments that were left out when initially fetching the post.
func (s *PostService) LoadMoreComments(ctx context.Context, pc *PostAndComments) (*Response, error) {
	if pc == nil {
		return nil, errors.New("*PostAndComments: cannot be nil")
	}

	if !pc.HasMore() {
		return nil, nil
	}

	postID := pc.Post.FullID
	commentIDs := pc.More.Children

	form := url.Values{}
	form.Set("api_type", "json")
	form.Set("link_id", postID)
	form.Set("children", strings.Join(commentIDs, ","))

	path := "api/morechildren"

	// This was originally a GET, but with POST you can send a bigger payload
	// since it's in the body and not the URI.
	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	root := new(struct {
		JSON struct {
			Data struct {
				Things things `json:"things"`
			} `json:"data"`
		} `json:"json"`
	})
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return resp, err
	}

	comments := root.JSON.Data.Things.Comments
	for _, c := range comments {
		pc.addCommentToTree(c)
	}

	noMore := true

	mores := root.JSON.Data.Things.Mores
	for _, m := range mores {
		if strings.HasPrefix(m.ParentID, kindPost+"_") {
			noMore = false
		}
		pc.addMoreToTree(m)
	}

	if noMore {
		pc.More = nil
	}

	return resp, nil
}

func (s *PostService) random(ctx context.Context, subreddits ...string) (*PostAndComments, *Response, error) {
	path := "random"
	if len(subreddits) > 0 {
		path = fmt.Sprintf("r/%s/random", strings.Join(subreddits, "+"))
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

// RandomFromSubreddits returns a random post and its comments from the subreddits.
// If no subreddits are provided, Reddit runs the query against your subscriptions.
func (s *PostService) RandomFromSubreddits(ctx context.Context, subreddits ...string) (*PostAndComments, *Response, error) {
	return s.random(ctx, subreddits...)
}

// Random returns a random post and its comments from all of Reddit.
func (s *PostService) Random(ctx context.Context) (*PostAndComments, *Response, error) {
	return s.random(ctx, "all")
}

// RandomFromSubscriptions returns a random post and its comments from your subscriptions.
func (s *PostService) RandomFromSubscriptions(ctx context.Context) (*PostAndComments, *Response, error) {
	return s.random(ctx)
}

// MarkVisited marks the post(s) as visited.
// This method requires a subscription to Reddit premium.
func (s *PostService) MarkVisited(ctx context.Context, ids ...string) (*Response, error) {
	if len(ids) == 0 {
		return nil, errors.New("must provide at least 1 id")
	}

	path := "api/store_visits"

	form := url.Values{}
	form.Set("links", strings.Join(ids, ","))

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}
