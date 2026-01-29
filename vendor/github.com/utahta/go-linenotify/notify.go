package linenotify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type (
	// NotifyResponse represents response that LINE Notify API
	NotifyResponse struct {
		Status    int    `json:"status"`
		Message   string `json:"message"`
		RateLimit RateLimit
	}
)

var (
	ErrNotifyInvalidAccessToken = errors.New("invalid access token")
)

// Notify provides convenient Notify* interface
func (c *Client) Notify(ctx context.Context, token, message, imageThumbnail, imageFullsize string, image io.Reader) (*NotifyResponse, error) {
	if image != nil {
		return c.NotifyWithImage(ctx, token, message, image)
	}
	return c.NotifyWithImageURL(ctx, token, message, imageThumbnail, imageFullsize)
}

// NotifyMessage notify text message
func (c *Client) NotifyMessage(ctx context.Context, token, message string) (*NotifyResponse, error) {
	return c.NotifyWithImageURL(ctx, token, message, "", "")
}

// NotifyWithImage notify text message and image by binary
func (c *Client) NotifyWithImage(ctx context.Context, token, message string, image io.Reader) (*NotifyResponse, error) {
	body, contentType, err := c.requestBodyWithImage(message, image)
	if err != nil {
		return nil, err
	}
	return c.notify(ctx, token, body, contentType)
}

// NotifyWithImageURL notify text message and image by url
func (c *Client) NotifyWithImageURL(ctx context.Context, token, message, imageThumbnail, imageFullsize string) (*NotifyResponse, error) {
	body, contentType, err := c.requestBody(message, imageThumbnail, imageFullsize)
	if err != nil {
		return nil, err
	}
	return c.notify(ctx, token, body, contentType)
}

func (c *Client) notify(ctx context.Context, token string, body io.Reader, contentType string) (*NotifyResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", "https://notify-api.line.me/api/notify", body)
	if err != nil {
		return nil, fmt.Errorf("failed to new request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to notify: %w", err)
	}
	defer resp.Body.Close()

	nResp := &NotifyResponse{}
	err = json.NewDecoder(resp.Body).Decode(nResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode notification response: %w", err)
	}
	nResp.RateLimit.Parse(resp.Header)

	if resp.StatusCode == http.StatusUnauthorized {
		return nResp, ErrNotifyInvalidAccessToken
	}

	if resp.StatusCode != http.StatusOK {
		return nResp, errors.New(nResp.Message)
	}
	return nResp, nil
}

func (c *Client) requestBody(message, imageThumbnail, imageFullsize string) (io.Reader, string, error) {
	v := url.Values{}
	v.Add("message", message)
	if imageThumbnail != "" {
		v.Add("imageThumbnail", imageThumbnail)
	}
	if imageFullsize != "" {
		v.Add("imageFullsize", imageFullsize)
	}
	return strings.NewReader(v.Encode()), "application/x-www-form-urlencoded", nil
}

func (c *Client) requestBodyWithImage(message string, image io.Reader) (io.Reader, string, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	if err := w.WriteField("message", message); err != nil {
		return nil, "", err
	}

	randomID, err := uuid.NewRandom()
	if err != nil {
		return nil, "", err
	}

	fw, err := w.CreateFormFile("imageFile", randomID.String())
	if err != nil {
		return nil, "", err
	}

	if _, err := io.Copy(fw, image); err != nil {
		return nil, "", err
	}

	if err := w.Close(); err != nil {
		return nil, "", err
	}

	return &b, w.FormDataContentType(), nil
}
