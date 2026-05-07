package fcm

import (
	"fmt"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

// Option configurates Client with defined option.
type Option func(*Client) error

// WithHTTPClient returns Option to configure HTTP Client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) error {
		c.httpClient = httpClient
		return nil
	}
}

// WithHTTPProxy returns Option to configure HTTP Client proxy.
func WithHTTPProxy(proxyURL string) Option {
	return func(c *Client) error {
		proxy, err := url.Parse(proxyURL)
		if err != nil {
			return err
		}
		httpClient := &http.Client{
			Transport: &http.Transport{Proxy: http.ProxyURL(proxy)},
		}
		c.httpClient = httpClient
		return nil
	}
}

// WithCredentialsFile returns a ClientOption that authenticates
// API calls with the given service account or refresh token JSON
// credentials file.
func WithCredentialsFile(filename string) Option {
	return func(c *Client) error {
		data, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("cannot read credentials file: %v", err)
		}
		c.credentialsJSON = data
		c.options = append(c.options, option.WithCredentialsJSON(data))
		return nil
	}
}

// WithCredentialsJSON returns a ClientOption that authenticates
// API calls with the given service account or refresh token JSON
// credentials.
func WithCredentialsJSON(data []byte) Option {
	return func(c *Client) error {
		c.credentialsJSON = data
		c.options = append(c.options, option.WithCredentialsJSON(data))
		return nil
	}
}

// WithEndpoint returns Option to configure endpoint.
func WithEndpoint(endpoint string) Option {
	return func(c *Client) error {
		c.options = append(c.options, option.WithEndpoint(endpoint))
		return nil
	}
}

// WithServiceAccount returns Option to configure service account.
func WithServiceAccount(serviceAccount string) Option {
	return func(c *Client) error {
		c.serviceAcount = serviceAccount
		return nil
	}
}

// WithProjectID returns Option to configure project ID.
func WithProjectID(projectID string) Option {
	return func(c *Client) error {
		c.projectID = projectID
		return nil
	}
}

// WithTokenSource returns a ClientOption that specifies an OAuth2 token
// source to be used as the basis for authentication.
func WithTokenSource(s oauth2.TokenSource) Option {
	return func(c *Client) error {
		c.options = append(c.options, option.WithTokenSource(s))
		return nil
	}
}

// WithDebug returns Option to configure debug mode.
func WithDebug(debug bool) Option {
	return func(c *Client) error {
		c.debug = debug
		return nil
	}
}

// WithCustomClientOption is an option function that allows you to provide custom client options.
// It appends the provided custom options to the client's options list.
// The custom options are applied when sending requests to the FCM server.
// If no custom options are provided, this function does nothing.
//
// Parameters:
//   - opts: The custom client options to be appended to the client's options list.
//
// Returns:
//   - An error if there was an issue appending the custom options to the client's options list, or nil otherwise.
func WithCustomClientOption(opts ...option.ClientOption) Option {
	return func(c *Client) error {
		if len(opts) == 0 {
			return nil
		}
		c.options = append(c.options, opts...)
		return nil
	}
}
