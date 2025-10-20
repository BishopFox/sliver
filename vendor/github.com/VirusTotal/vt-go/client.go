// Copyright © 2017 The vt-go authors. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package vt

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Client for interacting with VirusTotal API.
type Client struct {
	// APIKey is the VirusTotal API key that identifies the user making the
	// requests.
	APIKey string
	// Agent is a string included in the User-Agent header of every request
	// sent to VirusTotal's servers. Users of this client are encouraged to
	// use some string that uniquely identify the program making the requests.
	Agent      string
	httpClient *http.Client
}

type requestOptions struct {
	headers map[string]string
}

// RequestOption represents an option passed to some functions in this package.
type RequestOption func(*requestOptions)

// WithHeader specifies a header to be included in the request.
func WithHeader(header, value string) RequestOption {
	return func(opts *requestOptions) {
		if opts.headers == nil {
			opts.headers = make(map[string]string)
		}
		opts.headers[header] = value
	}
}

func opts(opts ...RequestOption) *requestOptions {
	o := &requestOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

type ClientOption func(*Client)

func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// NewClient creates a new client for interacting with the VirusTotal API using
// the provided API key.
func NewClient(APIKey string, opts ...ClientOption) *Client {
	c := &Client{APIKey: APIKey, httpClient: &http.Client{}}

	for _, o := range opts {
		o(c)
	}

	return c
}

// sendRequest sends a HTTP request to the VirusTotal REST API.
func (cli *Client) sendRequest(method string, url *url.URL, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, url.String(), body)
	if err != nil {
		return nil, err
	}
	agent := cli.Agent
	if agent == "" {
		agent = "unknown"
	}
	// AppEngine server decides whether or not it should serve gzipped content
	// based on Accept-Encoding and User-Agent. Non-standard UAs are not served
	// with gzipped content unless it contains the string "gzip" somewhere.
	// See: https://cloud.google.com/appengine/kb/#compression
	req.Header.Set("User-Agent", fmt.Sprintf("%s; vtgo %s; gzip", agent, version))
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("X-Apikey", cli.APIKey)

	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	return (cli.httpClient).Do(req)
}

// parseResponse parses a HTTP response received from the VirusTotal REST API.
// If a valid JSON response was received from the server this function returns
// a pointer to a Response structure. An error is returned either if the response
// was not a valid JSON or if it was a valid JSON but contained an API error.
// Notice that this means that both return values can be non-nil.
func (cli *Client) parseResponse(resp *http.Response) (*Response, error) {

	apiresp := &Response{}

	if resp.ContentLength == 0 {
		return apiresp, nil
	}

	// If the response has some content its format should be JSON
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		return nil, fmt.Errorf("Expecting JSON response from %s %s",
			resp.Request.Method, resp.Request.URL.String())
	}

	// Prepare gzip reader for uncompressing gzipped JSON response
	ungzipper, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, err
	}
	defer ungzipper.Close()

	if err := json.NewDecoder(ungzipper).Decode(apiresp); err != nil {
		return nil, err
	}

	// Check if the response was an error
	if apiresp.Error.Code != "" {
		return apiresp, apiresp.Error
	}

	return apiresp, nil
}

// Get sends a GET request to the specified API endpoint. This is a low level
// primitive that returns a Response struct, where the response's data is in
// raw form. See GetObject and GetData for higher level primitives.
func (cli *Client) Get(url *url.URL, options ...RequestOption) (*Response, error) {
	o := opts(options...)
	httpResp, err := cli.sendRequest("GET", url, nil, o.headers)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()
	return cli.parseResponse(httpResp)
}

// Post sends a POST request to the specified API endpoint.
func (cli *Client) Post(url *url.URL, req *Request, options ...RequestOption) (*Response, error) {
	var b []byte
	var err error
	if req != nil {
		b, err = json.Marshal(req)
		if err != nil {
			return nil, err
		}
	}
	o := opts(options...)
	httpResp, err := cli.sendRequest("POST", url, bytes.NewReader(b), o.headers)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()
	return cli.parseResponse(httpResp)
}

// Patch sends a PATCH request to the specified API endpoint.
func (cli *Client) Patch(url *url.URL, req *Request, options ...RequestOption) (*Response, error) {
	var b []byte
	var err error
	if req != nil {
		b, err = json.Marshal(req)
		if err != nil {
			return nil, err
		}
	}
	o := opts(options...)
	httpResp, err := cli.sendRequest("PATCH", url, bytes.NewReader(b), o.headers)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()
	return cli.parseResponse(httpResp)
}

// Delete sends a DELETE request to the specified API endpoint.
func (cli *Client) Delete(url *url.URL, options ...RequestOption) (*Response, error) {
	o := opts(options...)
	httpResp, err := cli.sendRequest("DELETE", url, nil, o.headers)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()
	return cli.parseResponse(httpResp)
}

// GetData sends a GET request to the specified API endpoint and unmarshals the
// JSON-encoded data received in the API response. The unmarshalled data is put
// into the specified target. The target must be of an appropriate type capable
// of receiving the data returned by the the endpoint. If the data returned by
// the endpoint is an object you can use GetObject instead.
func (cli *Client) GetData(url *url.URL, target interface{}, options ...RequestOption) (*Response, error) {
	resp, err := cli.Get(url, options...)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(resp.Data))
	decoder.UseNumber()
	return resp, decoder.Decode(target)
}

// PostData sends a POST request to the specified API endpoint. The data argument
// is JSON-encoded and wrapped as {'data': <JSON-encoded data> }.
func (cli *Client) PostData(url *url.URL, data interface{}, options ...RequestOption) (*Response, error) {
	req := &Request{}
	req.Data = data
	return cli.Post(url, req, options...)
}

// PostObject adds an Object to a collection. The specified URL must point to
// a collection, not an object, but not all collections accept this operation.
// For more information about collection and objects in the VirusTotal API see:
//
// https://developers.virustotal.com/v3.0/reference#objects
// https://developers.virustotal.com/v3.0/reference#collections
//
// This function updates the object with data returned from the server, so
// the object's attributes can differ from those it had before the call.
//
// Example:
//	obj := vt.NewObject("hunting_ruleset")
//	obj.SetString("name", "test")
//	obj.SetString("rules", "rule test {condition: false}")
//
//	client.PostObject(vt.URL("intelligence/hunting_rulesets"), obj)
//
func (cli *Client) PostObject(url *url.URL, obj *Object, options ...RequestOption) error {
	req := &Request{}
	req.Data = modifiedObject(*obj)
	resp, err := cli.Post(url, req, options...)
	if err != nil {
		return err
	}
	return json.Unmarshal(resp.Data, obj)
}

// GetObject returns an Object from a URL. The specified URL must reference
// an object, not a collection. This means that GetObject can be used with URLs
// like /files/{file_id} and /urls/{url_id}, which return an individual object
// but not with /comments, which returns a collection of objects.
func (cli *Client) GetObject(url *url.URL, options ...RequestOption) (*Object, error) {
	obj := &Object{}
	if _, err := cli.GetData(url, obj, options...); err != nil {
		return nil, err
	}
	return obj, nil
}

// PatchObject modifies an existing object.
func (cli *Client) PatchObject(url *url.URL, obj *Object, options ...RequestOption) error {
	req := &Request{}
	req.Data = modifiedObject(*obj)
	resp, err := cli.Patch(url, req, options...)
	if err != nil {
		return err
	}
	return json.Unmarshal(resp.Data, obj)
}

// DownloadFile downloads a file given its hash (SHA-256, SHA-1 or MD5). The
// file is written into the provided io.Writer.
func (cli *Client) DownloadFile(hash string, w io.Writer) (int64, error) {
	u := URL("files/%s/download", hash)
	resp, err := cli.sendRequest("GET", u, nil, nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return io.Copy(w, resp.Body)
	}

	// See if there is an error in the response.
	if _, err := cli.parseResponse(resp); err != nil {
		return 0, err
	}

	// Last resort return a generic error.
	return 0, fmt.Errorf("Unknown error downloading %q, HTTP response code: %d", hash, resp.StatusCode)
}

// Iterator returns an iterator for a collection. If the endpoint passed to the
// iterator returns a single object instead of a collection, it behaves as if
// iterating over a collection with a single object. Iterators are usually
// used like this:
//
//  cli := vt.Client(<api key>)
//  it, err := cli.Iterator(vt.URL(<collection path>))
//  if err != nil {
//	  ...handle error
//  }
//  defer it.Close()
//  for it.Next() {
//    obj := it.Get()
//    ...do something with obj
//  }
//  if err := it.Error(); err != nil {
//    ...handle error
//  }
//
func (cli *Client) Iterator(url *url.URL, options ...IteratorOption) (*Iterator, error) {
	return newIterator(cli, url, options...)
}

// Search for files using VirusTotal Intelligence query language.
// Example:
//   it, err := client.Search("p:10+ size:30MB+")
//
func (cli *Client) Search(query string, options ...IteratorOption) (*Iterator, error) {
	u := URL("intelligence/search")
	q := u.Query()
	q.Add("query", query)
	u.RawQuery = q.Encode()
	return newIterator(cli, u, options...)
}

// Metadata describes the structure returned by /api/v3/metadata with metadata
// about VirusTotal, including the relationships supported by each object type.
type Metadata struct {
	// Dictionary where keys are the names of the Antivirus engines currently
	// supported by VirusTotal.
	Engines map[string]interface{} `json:"engines" yaml:"engines"`
	// Dictionary containing the relationships supported by each object type in
	// the VirusTotal API. Keys in this dictionary are object types, and values
	// are a list of RelationshipMeta structures with information about the
	// relationship.
	Relationships map[string][]RelationshipMeta `json:"relationships" yaml:"relationships"`
	Privileges    []string                      `json:"privileges" yaml:"privileges"`
}

// RelationshipMeta is the structure returned by each relationship from the
// /api/v3/metadata endpoint.
type RelationshipMeta struct {
	// Name of the relationship.
	Name string `json:"name" yaml:"name"`
	// A verbose description for the relationship.
	Description string `json:"description" yaml:"description"`
}

// GetMetadata retrieves VirusTotal metadata by calling the /api/v3/metadata
// endpoint.
func (cli *Client) GetMetadata() (*Metadata, error) {
	metadata := &Metadata{}
	if _, err := cli.GetData(URL("metadata"), metadata); err != nil {
		return nil, err
	}
	return metadata, nil
}

// NewFileScanner returns a new FileScanner.
func (cli *Client) NewFileScanner() *FileScanner {
	return &FileScanner{cli: cli}
}

// NewURLScanner returns a new URLScanner.
func (cli *Client) NewURLScanner() *URLScanner {
	return &URLScanner{cli: cli}
}

// NewMonitorUploader returns a new MonitorUploader.
func (cli *Client) NewMonitorUploader() *MonitorUploader {
	return &MonitorUploader{cli: cli}
}
