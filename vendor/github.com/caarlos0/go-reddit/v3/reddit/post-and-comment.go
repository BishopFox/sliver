package reddit

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// postAndCommentService handles communication with the post and comment
// related methods of the Reddit API.
// This service holds functionality common to both posts and comments.
//
// Reddit API docs: https://www.reddit.com/dev/api/#section_links_and_comments
type postAndCommentService struct {
	client *Client
}

type vote int

// Reddit interprets -1, 0, 1 as downvote, no vote, and upvote, respectively.
const (
	downvote vote = iota - 1
	novote
	upvote
)

// Delete a post or comment via its full ID.
func (s *postAndCommentService) Delete(ctx context.Context, id string) (*Response, error) {
	path := "api/del"

	form := url.Values{}
	form.Set("id", id)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Save a post or comment.
func (s *postAndCommentService) Save(ctx context.Context, id string) (*Response, error) {
	path := "api/save"

	form := url.Values{}
	form.Set("id", id)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Unsave a post or comment.
func (s *postAndCommentService) Unsave(ctx context.Context, id string) (*Response, error) {
	path := "api/unsave"

	form := url.Values{}
	form.Set("id", id)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// EnableReplies enables inbox replies for one of your posts or comments.
func (s *postAndCommentService) EnableReplies(ctx context.Context, id string) (*Response, error) {
	path := "api/sendreplies"

	form := url.Values{}
	form.Set("id", id)
	form.Set("state", "true")

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// DisableReplies dsables inbox replies for one of your posts or comments.
func (s *postAndCommentService) DisableReplies(ctx context.Context, id string) (*Response, error) {
	path := "api/sendreplies"

	form := url.Values{}
	form.Set("id", id)
	form.Set("state", "false")

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Lock a post or comment, preventing it from receiving new comments.
func (s *postAndCommentService) Lock(ctx context.Context, id string) (*Response, error) {
	path := "api/lock"

	form := url.Values{}
	form.Set("id", id)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Unlock a post or comment, allowing it to receive new comments.
func (s *postAndCommentService) Unlock(ctx context.Context, id string) (*Response, error) {
	path := "api/unlock"

	form := url.Values{}
	form.Set("id", id)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

func (s *postAndCommentService) vote(ctx context.Context, id string, vote vote) (*Response, error) {
	path := "api/vote"

	form := url.Values{}
	form.Set("id", id)
	form.Set("dir", strconv.Itoa(int(vote)))
	form.Set("rank", "10")

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Upvote a post or a comment.
func (s *postAndCommentService) Upvote(ctx context.Context, id string) (*Response, error) {
	return s.vote(ctx, id, upvote)
}

// Downvote a post or a comment.
func (s *postAndCommentService) Downvote(ctx context.Context, id string) (*Response, error) {
	return s.vote(ctx, id, downvote)
}

// RemoveVote removes your vote on a post or a comment.
func (s *postAndCommentService) RemoveVote(ctx context.Context, id string) (*Response, error) {
	return s.vote(ctx, id, novote)
}

// Report a post or comment.
// The reason must not be longer than 100 characters.
func (s *postAndCommentService) Report(ctx context.Context, id string, reason string) (*Response, error) {
	path := "api/report"

	form := url.Values{}
	form.Set("api_type", "json")
	form.Set("thing_id", id)
	form.Set("reason", reason)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}
