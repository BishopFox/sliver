package reddit

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/go-querystring/query"
)

// CollectionService handles communication with the collection
// related methods of the Reddit API.
//
// Reddit API docs: https://www.reddit.com/dev/api/#section_collections
type CollectionService struct {
	client *Client
}

// Collection is a mod curated group of posts within a subreddit.
type Collection struct {
	ID      string     `json:"collection_id,omitempty"`
	Created *Timestamp `json:"created_at_utc,omitempty"`
	Updated *Timestamp `json:"last_update_utc,omitempty"`

	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Permalink   string `json:"permalink,omitempty"`
	Layout      string `json:"display_layout,omitempty"`

	SubredditID string `json:"subreddit_id,omitempty"`
	Author      string `json:"author_name,omitempty"`
	AuthorID    string `json:"author_id,omitempty"`

	// Post at the top of the collection.
	// This does not appear when getting a list of collections.
	PrimaryPostID string   `json:"primary_link_id,omitempty"`
	PostIDs       []string `json:"link_ids,omitempty"`
}

// CollectionCreateRequest represents a request to create a collection.
type CollectionCreateRequest struct {
	Title       string `url:"title"`
	Description string `url:"description,omitempty"`
	SubredditID string `url:"sr_fullname"`
	// One of: TIMELINE, GALLERY.
	Layout string `url:"display_layout,omitempty"`
}

// Get gets a collection by its ID.
func (s *CollectionService) Get(ctx context.Context, id string) (*Collection, *Response, error) {
	path := "api/v1/collections/collection"

	params := struct {
		ID           string `url:"collection_id"`
		IncludePosts bool   `url:"include_links"`
	}{id, false}

	path, err := addOptions(path, params)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	collection := new(Collection)
	resp, err := s.client.Do(ctx, req, collection)
	if err != nil {
		return nil, resp, err
	}

	return collection, resp, nil
}

// FromSubreddit gets all collections in the subreddit.
func (s *CollectionService) FromSubreddit(ctx context.Context, id string) ([]*Collection, *Response, error) {
	path := "api/v1/collections/subreddit_collections"

	params := struct {
		SubredditID string `url:"sr_fullname"`
	}{id}

	path, err := addOptions(path, params)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	var collections []*Collection
	resp, err := s.client.Do(ctx, req, &collections)
	if err != nil {
		return nil, resp, err
	}

	return collections, resp, nil
}

// Create a collection.
func (s *CollectionService) Create(ctx context.Context, createRequest *CollectionCreateRequest) (*Collection, *Response, error) {
	if createRequest == nil {
		return nil, nil, errors.New("*CollectionCreateRequest: cannot be nil")
	}

	path := "api/v1/collections/create_collection"

	form, err := query.Values(createRequest)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, nil, err
	}

	collection := new(Collection)
	resp, err := s.client.Do(ctx, req, collection)
	if err != nil {
		return nil, resp, err
	}

	return collection, resp, nil
}

// Delete a collection via its id.
func (s *CollectionService) Delete(ctx context.Context, id string) (*Response, error) {
	path := "api/v1/collections/delete_collection"

	form := url.Values{}
	form.Set("collection_id", id)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// AddPost adds a post (via its full ID) to a collection (via its id).
func (s *CollectionService) AddPost(ctx context.Context, postID, collectionID string) (*Response, error) {
	path := "api/v1/collections/add_post_to_collection"

	form := url.Values{}
	form.Set("link_fullname", postID)
	form.Set("collection_id", collectionID)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// RemovePost removes a post (via its full ID) from a collection (via its id).
func (s *CollectionService) RemovePost(ctx context.Context, postID, collectionID string) (*Response, error) {
	path := "api/v1/collections/remove_post_in_collection"

	form := url.Values{}
	form.Set("link_fullname", postID)
	form.Set("collection_id", collectionID)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// ReorderPosts reorders posts in a collection.
func (s *CollectionService) ReorderPosts(ctx context.Context, collectionID string, postIDs ...string) (*Response, error) {
	path := "api/v1/collections/reorder_collection"

	form := url.Values{}
	form.Set("collection_id", collectionID)
	form.Set("link_ids", strings.Join(postIDs, ","))

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// UpdateTitle updates a collection's title.
func (s *CollectionService) UpdateTitle(ctx context.Context, id string, title string) (*Response, error) {
	path := "api/v1/collections/update_collection_title"

	form := url.Values{}
	form.Set("collection_id", id)
	form.Set("title", title)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// UpdateDescription updates a collection's description.
func (s *CollectionService) UpdateDescription(ctx context.Context, id string, description string) (*Response, error) {
	path := "api/v1/collections/update_collection_description"

	form := url.Values{}
	form.Set("collection_id", id)
	form.Set("description", description)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// UpdateLayoutTimeline updates a collection's layout to the timeline format.
func (s *CollectionService) UpdateLayoutTimeline(ctx context.Context, id string) (*Response, error) {
	path := "api/v1/collections/update_collection_display_layout"

	form := url.Values{}
	form.Set("collection_id", id)
	form.Set("display_layout", "TIMELINE")

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// UpdateLayoutGallery updates a collection's layout to the gallery format.
func (s *CollectionService) UpdateLayoutGallery(ctx context.Context, id string) (*Response, error) {
	path := "api/v1/collections/update_collection_display_layout"

	form := url.Values{}
	form.Set("collection_id", id)
	form.Set("display_layout", "GALLERY")

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Follow a collection.
func (s *CollectionService) Follow(ctx context.Context, id string) (*Response, error) {
	path := "api/v1/collections/follow_collection"

	form := url.Values{}
	form.Set("collection_id", id)
	form.Set("follow", "true")

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Unfollow a collection.
func (s *CollectionService) Unfollow(ctx context.Context, id string) (*Response, error) {
	path := "api/v1/collections/follow_collection"

	form := url.Values{}
	form.Set("collection_id", id)
	form.Set("follow", "false")

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}
