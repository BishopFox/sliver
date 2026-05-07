//Package rest provides a RocketChat rest client.
package rest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

var (
	ResponseErr = fmt.Errorf("got false response")
)

type Response interface {
	OK() error
}

type Client struct {
	Protocol string
	Host     string
	Path     string
	Port     string
	Version  string

	// Use this switch to see all network communication.
	Debug bool

	auth *authInfo
}

type Status struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`

	Status  string `json:"status"`
	Message string `json:"message"`
}

type authInfo struct {
	token string
	id    string
}

func (s Status) OK() error {
	if s.Success {
		return nil
	}

	if len(s.Error) > 0 {
		return fmt.Errorf(s.Error)
	}

	if s.Status == "success" {
		return nil
	}

	if len(s.Message) > 0 {
		return fmt.Errorf("status: %s, message: %s", s.Status, s.Message)
	}
	return ResponseErr
}

// StatusResponse The base for the most of the json responses
type StatusResponse struct {
	Status
	Channel string `json:"channel"`
}

func NewClient(serverUrl *url.URL, debug bool) *Client {
	protocol := "http"
	port := "80"

	if serverUrl.Scheme == "https" {
		protocol = "https"
		port = "443"
	}

	if len(serverUrl.Port()) > 0 {
		port = serverUrl.Port()
	}

	return &Client{Host: serverUrl.Hostname(), Path: serverUrl.Path, Port: port, Protocol: protocol, Version: "v1", Debug: debug}
}

func (c *Client) getUrl() string {
	if len(c.Version) == 0 {
		c.Version = "v1"
	}
	return fmt.Sprintf("%v://%v:%v%s/api/%s", c.Protocol, c.Host, c.Port, c.Path, c.Version)
}

// Get call Get
func (c *Client) Get(api string, params url.Values, response Response) error {
	return c.doRequest(http.MethodGet, api, params, nil, response)
}

// Post call as JSON
func (c *Client) Post(api string, body io.Reader, response Response) error {
	return c.doRequest(http.MethodPost, api, nil, body, response)
}

// PostForm call as Form Data
func (c *Client) PostForm(api string, params url.Values, response Response) error {
	return c.doRequest(http.MethodPost, api, params, nil, response)
}

func (c *Client) doRequest(method, api string, params url.Values, body io.Reader, response Response) error {
	contentType := "application/x-www-form-urlencoded"
	if method == http.MethodPost {
		if body != nil {
			contentType = "application/json"
		} else if len(params) > 0 {
			body = bytes.NewBufferString(params.Encode())
		}
	}

	request, err := http.NewRequest(method, c.getUrl()+"/"+api, body)
	if err != nil {
		return err
	}

	if method == http.MethodGet {
		if len(params) > 0 {
			request.URL.RawQuery = params.Encode()
		}
	} else {
		request.Header.Set("Content-Type", contentType)
	}

	if c.auth != nil {
		request.Header.Set("X-Auth-Token", c.auth.token)
		request.Header.Set("X-User-Id", c.auth.id)
	}

	if c.Debug {
		log.Println(request)
	}

	resp, err := http.DefaultClient.Do(request)

	if err != nil {
		return err
	}

	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)

	if c.Debug {
		log.Println(string(bodyBytes))
	}

	var parse bool
	if err == nil {
		if e := json.Unmarshal(bodyBytes, response); e == nil {
			parse = true
		}
	}
	if resp.StatusCode != http.StatusOK {
		if parse {
			return response.OK()
		}
		return errors.New("Request error: " + resp.Status)
	}

	if err != nil {
		return err
	}

	return response.OK()
}
