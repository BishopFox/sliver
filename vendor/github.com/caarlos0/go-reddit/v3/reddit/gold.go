package reddit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// GoldService handles communication with the gold
// related methods of the Reddit API.
//
// Reddit API docs: https://www.reddit.com/dev/api/#section_gold
type GoldService struct {
	client *Client
}

// Gild the post or comment via its full ID.
// This requires you to own Reddit coins and will consume them.
func (s *GoldService) Gild(ctx context.Context, id string) (*Response, error) {
	path := fmt.Sprintf("api/v1/gold/gild/%s", id)
	req, err := s.client.NewRequest(http.MethodPost, path, nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(ctx, req, nil)
}

// Give the user between 1 and 36 (inclusive) months of gold.
// This requires you to own Reddit coins and will consume them.
func (s *GoldService) Give(ctx context.Context, username string, months int) (*Response, error) {
	if months < 1 || months > 36 {
		return nil, errors.New("months: must be between 1 and 36 (inclusive)")
	}

	path := fmt.Sprintf("api/v1/gold/give/%s", username)

	form := url.Values{}
	form.Set("months", strconv.Itoa(months))

	req, err := s.client.NewRequest(http.MethodPost, path, form)
	if err != nil {
		return nil, err
	}

	return s.client.Do(ctx, req, nil)
}
