package reddit

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

// CommentService handles communication with the comment
// related methods of the Reddit API.
//
// Reddit API docs: https://www.reddit.com/dev/api/#section_links_and_comments
type CommentService struct {
	*postAndCommentService
	client *Client
}

// Submit a comment as a reply to a post, comment, or message.
// parentID is the full ID of the thing being replied to.
func (s *CommentService) Submit(ctx context.Context, parentID string, text string) (*Comment, *Response, error) {
	path := "api/comment"

	form := url.Values{}
	form.Set("api_type", "json")
	form.Set("return_rtjson", "true")
	form.Set("parent", parentID)
	form.Set("text", text)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, nil, err
	}

	root := new(Comment)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root, resp, nil
}

// Edit a comment.
func (s *CommentService) Edit(ctx context.Context, id string, text string) (*Comment, *Response, error) {
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

	root := new(Comment)
	resp, err := s.client.Do(ctx, req, root)
	if err != nil {
		return nil, resp, err
	}

	return root, resp, nil
}

// LoadMoreReplies retrieves more replies that were left out when initially fetching the comment.
func (s *CommentService) LoadMoreReplies(ctx context.Context, comment *Comment) (*Response, error) {
	if comment == nil {
		return nil, errors.New("*Comment: cannot be nil")
	}

	if !comment.HasMore() {
		return nil, nil
	}

	postID := comment.PostID
	commentIDs := comment.Replies.More.Children

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
	mores := root.JSON.Data.Things.Mores

	for _, c := range comments {
		comment.addCommentToReplies(c)
	}

	if len(mores) > 0 {
		comment.Replies.More = mores[0]
	} else {
		comment.Replies.More = nil
	}

	return resp, nil
}
