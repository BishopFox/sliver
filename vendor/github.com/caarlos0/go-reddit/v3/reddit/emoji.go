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
	"strconv"
	"strings"

	"github.com/google/go-querystring/query"
	"golang.org/x/net/context/ctxhttp"
)

// EmojiService handles communication with the emoji
// related methods of the Reddit API.
//
// Reddit API docs: https://www.reddit.com/dev/api/#section_emoji
type EmojiService struct {
	client *Client
}

// Emoji is a graphic element you can include in a post flair or user flair.
type Emoji struct {
	Name             string `json:"name,omitempty"`
	URL              string `json:"url,omitempty"`
	UserFlairAllowed bool   `json:"user_flair_allowed,omitempty"`
	PostFlairAllowed bool   `json:"post_flair_allowed,omitempty"`
	ModFlairOnly     bool   `json:"mod_flair_only,omitempty"`
	// ID of the user who created this emoji.
	CreatedBy string `json:"created_by,omitempty"`
}

// EmojiCreateOrUpdateRequest represents a request to create/update an emoji.
type EmojiCreateOrUpdateRequest struct {
	Name             string `url:"name"`
	UserFlairAllowed *bool  `url:"user_flair_allowed,omitempty"`
	PostFlairAllowed *bool  `url:"post_flair_allowed,omitempty"`
	ModFlairOnly     *bool  `url:"mod_flair_only,omitempty"`
}

func (r *EmojiCreateOrUpdateRequest) validate() error {
	if r == nil {
		return errors.New("*EmojiCreateOrUpdateRequest: cannot be nil")
	}
	if r.Name == "" {
		return errors.New("(*EmojiCreateOrUpdateRequest).Name: cannot be empty")
	}
	return nil
}

type emojis []*Emoji

func (e *emojis) UnmarshalJSON(data []byte) (err error) {
	emojiMap := make(map[string]json.RawMessage)
	err = json.Unmarshal(data, &emojiMap)
	if err != nil {
		return
	}

	*e = make(emojis, 0, len(emojiMap))
	for emojiName, emojiValue := range emojiMap {
		emoji := new(Emoji)
		err = json.Unmarshal(emojiValue, emoji)
		if err != nil {
			return
		}
		emoji.Name = emojiName
		*e = append(*e, emoji)
	}

	return
}

// Get the default set of Reddit emojis and those of the subreddit, respectively.
func (s *EmojiService) Get(ctx context.Context, subreddit string) ([]*Emoji, []*Emoji, *Response, error) {
	path := fmt.Sprintf("api/v1/%s/emojis/all", subreddit)

	req, err := s.client.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	root := make(map[string]emojis)
	resp, err := s.client.Do(ctx, req, &root)
	if err != nil {
		return nil, nil, resp, err
	}

	defaultEmojis := root["snoomojis"]
	var subredditEmojis []*Emoji

	for k, v := range root {
		if strings.HasPrefix(k, kindSubreddit) {
			subredditEmojis = v
			break
		}
	}

	return defaultEmojis, subredditEmojis, resp, nil
}

// Delete the emoji from the subreddit.
func (s *EmojiService) Delete(ctx context.Context, subreddit string, emoji string) (*Response, error) {
	path := fmt.Sprintf("api/v1/%s/emoji/%s", subreddit, emoji)
	req, err := s.client.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

// SetSize sets the custom emoji size in the subreddit.
// Both height and width must be between 1 and 40 (inclusive).
func (s *EmojiService) SetSize(ctx context.Context, subreddit string, height, width int) (*Response, error) {
	path := fmt.Sprintf("api/v1/%s/emoji_custom_size", subreddit)

	form := url.Values{}
	form.Set("height", strconv.Itoa(height))
	form.Set("width", strconv.Itoa(width))

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// DisableCustomSize disables the custom emoji size in the subreddit.
func (s *EmojiService) DisableCustomSize(ctx context.Context, subreddit string) (*Response, error) {
	path := fmt.Sprintf("api/v1/%s/emoji_custom_size", subreddit)
	req, err := s.client.NewRequest(http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

func (s *EmojiService) lease(ctx context.Context, subreddit, imagePath string) (string, map[string]string, *Response, error) {
	path := fmt.Sprintf("api/v1/%s/emoji_asset_upload_s3.json", subreddit)

	form := url.Values{}
	form.Set("filepath", imagePath)
	form.Set("mimetype", "image/jpeg")
	if strings.HasSuffix(strings.ToLower(imagePath), ".png") {
		form.Set("mimetype", "image/png")
	}

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return "", nil, nil, err
	}

	var response struct {
		S3UploadLease struct {
			Action string `json:"action"`
			Fields []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"fields"`
		} `json:"s3UploadLease"`
	}

	resp, err := s.client.Do(ctx, req, &response)
	if err != nil {
		return "", nil, resp, err
	}

	uploadURL := fmt.Sprintf("http:%s", response.S3UploadLease.Action)

	fields := make(map[string]string)
	for _, field := range response.S3UploadLease.Fields {
		fields[field.Name] = field.Value
	}

	return uploadURL, fields, resp, nil
}

func (s *EmojiService) upload(ctx context.Context, subreddit string, createRequest *EmojiCreateOrUpdateRequest, awsKey string) (*Response, error) {
	path := fmt.Sprintf("api/v1/%s/emoji.json", subreddit)

	form, err := query.Values(createRequest)
	if err != nil {
		return nil, err
	}
	form.Set("s3_key", awsKey)

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}

// Upload an emoji to the subreddit.
func (s *EmojiService) Upload(ctx context.Context, subreddit string, createRequest *EmojiCreateOrUpdateRequest, imagePath string) (*Response, error) {
	err := createRequest.validate()
	if err != nil {
		return nil, err
	}

	uploadURL, fields, resp, err := s.lease(ctx, subreddit, imagePath)
	if err != nil {
		return resp, err
	}

	file, err := os.Open(imagePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	// AWS ignores all fields in the request that come after the file field, so we need to set these before
	// https://stackoverflow.com/questions/15234496/upload-directly-to-amazon-s3-using-ajax-returning-error-bucket-post-must-contai/15235866#15235866
	for k, v := range fields {
		writer.WriteField(k, v)
	}

	part, err := writer.CreateFormFile("file", file.Name())
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	httpResponse, err := ctxhttp.Post(ctx, nil, uploadURL, writer.FormDataContentType(), body)
	if err != nil {
		return nil, err
	}

	err = CheckResponse(httpResponse)
	if err != nil {
		return newResponse(httpResponse), err
	}

	return s.upload(ctx, subreddit, createRequest, fields["key"])
}

// Update updates an emoji on the subreddit.
func (s *EmojiService) Update(ctx context.Context, subreddit string, updateRequest *EmojiCreateOrUpdateRequest) (*Response, error) {
	err := updateRequest.validate()
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("api/v1/%s/emoji_permissions", subreddit)

	form, err := query.Values(updateRequest)
	if err != nil {
		return nil, err
	}

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}
