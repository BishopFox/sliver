package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Apps Manifest Create Response Errors ("/apps.manifest.create")
type AppsManifestCreateResponseError struct {
	Code             string `json:"code,omitempty"`
	Message          string `json:"message"`
	Pointer          string `json:"pointer"`
	RelatedComponent string `json:"related_component,omitempty"`
}

// Conversations Invite Response Errors ("/conversations.invite")
type ConversationsInviteResponseError struct {
	Error string `json:"error"`
	Ok    bool   `json:"ok"`
	User  string `json:"user"`
}

func (t ConversationsInviteResponseError) Err() error {
	if !t.Ok {
		return fmt.Errorf("conversations invite error (user: %s): %s", t.User, t.Error)
	}
	return nil
}

// SlackResponseErrors represents a union type for different error structures
type SlackResponseErrors struct {
	AppsManifestCreateResponseError  *AppsManifestCreateResponseError  `json:"-"`
	ConversationsInviteResponseError *ConversationsInviteResponseError `json:"-"`
	Message                          *string                           `json:"-"`
}

// MarshalJSON implements custom marshaling for SlackResponseErrors
func (e SlackResponseErrors) MarshalJSON() ([]byte, error) {
	if e.AppsManifestCreateResponseError != nil {
		return json.Marshal(e.AppsManifestCreateResponseError)
	}
	if e.ConversationsInviteResponseError != nil {
		return json.Marshal(e.ConversationsInviteResponseError)
	}
	if e.Message != nil {
		return json.Marshal(*e.Message)
	}
	return json.Marshal(nil)
}

// UnmarshalJSON implements custom unmarshaling for SlackResponseErrors
func (e *SlackResponseErrors) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		return nil
	}

	// Try to determine the error type by checking for unique fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		// If we can't unmarshal as object, try as string (fallback case)
		//
		// For more details on this specific problem look up issue
		// https://github.com/slack-go/slack/issues/1446.
		var stringError string
		if stringErr := json.Unmarshal(data, &stringError); stringErr == nil {
			e.Message = &stringError
			return nil
		}
		return err
	}

	if _, hasPointer := raw["pointer"]; hasPointer {
		if _, hasMessage := raw["message"]; hasMessage {
			var amc AppsManifestCreateResponseError
			if err := json.Unmarshal(data, &amc); err != nil {
				return err
			}
			e.AppsManifestCreateResponseError = &amc
			return nil
		}
	}

	if _, hasUser := raw["user"]; hasUser {
		if _, hasError := raw["error"]; hasError {
			if _, hasOk := raw["ok"]; hasOk {
				var ci ConversationsInviteResponseError
				if err := json.Unmarshal(data, &ci); err != nil {
					return err
				}
				e.ConversationsInviteResponseError = &ci
				return nil
			}
		}
	}

	return fmt.Errorf("unknown error structure: %s", string(data))
}

// SlackResponse handles parsing out errors from the web api.
type SlackResponse struct {
	Ok               bool                  `json:"ok"`
	Error            string                `json:"error"`
	Errors           []SlackResponseErrors `json:"errors,omitempty"`
	ResponseMetadata ResponseMetadata      `json:"response_metadata"`
}

func (t SlackResponse) Err() error {
	if t.Ok {
		return nil
	}

	// handle pure text based responses like chat.post
	// which while they have a slack response in their data structure
	// it doesn't actually get set during parsing.
	if strings.TrimSpace(t.Error) == "" {
		return nil
	}

	return SlackErrorResponse{Err: t.Error, Errors: t.Errors, ResponseMetadata: t.ResponseMetadata}
}

// SlackErrorResponse brings along the metadata of errors returned by the Slack API.
type SlackErrorResponse struct {
	Err              string
	Errors           []SlackResponseErrors
	ResponseMetadata ResponseMetadata
}

func (r SlackErrorResponse) Error() string { return r.Err }

// RateLimitedError represents the rate limit response from slack
type RateLimitedError struct {
	RetryAfter time.Duration
}

func (e *RateLimitedError) Error() string {
	return fmt.Sprintf("slack rate limit exceeded, retry after %s", e.RetryAfter)
}

func (e *RateLimitedError) Retryable() bool {
	return true
}

func fileUploadReq(ctx context.Context, path string, r io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, r)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func downloadFile(ctx context.Context, client httpClient, token string, downloadURL string, writer io.Writer, d Debug) error {
	if downloadURL == "" {
		return fmt.Errorf("received empty download URL")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, &bytes.Buffer{})
	if err != nil {
		return err
	}

	var bearer = "Bearer " + token
	req.Header.Add("Authorization", bearer)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	err = checkStatusCode(resp, d)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, resp.Body)

	return err
}

func formReq(ctx context.Context, endpoint string, values url.Values) (req *http.Request, err error) {
	if req, err = http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode())); err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
}

func jsonReq(ctx context.Context, endpoint string, body interface{}) (req *http.Request, err error) {
	buffer := bytes.NewBuffer([]byte{})
	if err = json.NewEncoder(buffer).Encode(body); err != nil {
		return nil, err
	}

	if req, err = http.NewRequestWithContext(ctx, http.MethodPost, endpoint, buffer); err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	return req, nil
}

func postLocalWithMultipartResponse(ctx context.Context, client httpClient, method, fpath, fieldname, token string, values url.Values, intf interface{}, d Debug) error {
	fullpath, err := filepath.Abs(fpath)
	if err != nil {
		return err
	}
	file, err := os.Open(fullpath)
	if err != nil {
		return err
	}
	defer file.Close()

	return postWithMultipartResponse(ctx, client, method, filepath.Base(fpath), fieldname, token, values, file, intf, d)
}

func postWithMultipartResponse(ctx context.Context, client httpClient, path, name, fieldname, token string, values url.Values, r io.Reader, intf interface{}, d Debug) error {
	pipeReader, pipeWriter := io.Pipe()
	wr := multipart.NewWriter(pipeWriter)

	errc := make(chan error)
	go func() {
		defer pipeWriter.Close()
		defer wr.Close()
		err := createFormFields(wr, values)
		if err != nil {
			errc <- err
			return
		}
		ioWriter, err := wr.CreateFormFile(fieldname, name)
		if err != nil {
			errc <- err
			return
		}
		_, err = io.Copy(ioWriter, r)
		if err != nil {
			errc <- err
			return
		}
		if err = wr.Close(); err != nil {
			errc <- err
			return
		}
	}()

	req, err := fileUploadReq(ctx, path, pipeReader)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", wr.FormDataContentType())
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := client.Do(req)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = checkStatusCode(resp, d)
	if err != nil {
		return err
	}

	select {
	case err = <-errc:
		return err
	default:
		return newJSONParser(intf)(resp)
	}
}

func createFormFields(mw *multipart.Writer, values url.Values) error {
	for key, value := range values {
		writer, err := mw.CreateFormField(key)
		if err != nil {
			return err
		}
		_, err = writer.Write([]byte(value[0]))
		if err != nil {
			return err
		}
	}
	return nil
}

func doPost(client httpClient, req *http.Request, parser responseParser, d Debug) error {
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = checkStatusCode(resp, d)
	if err != nil {
		return err
	}

	return parser(resp)
}

// post JSON.
func postJSON(ctx context.Context, client httpClient, endpoint, token string, json []byte, intf interface{}, d Debug) error {
	reqBody := bytes.NewBuffer(json)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	return doPost(client, req, newJSONParser(intf), d)
}

// post a url encoded form.
func postForm(ctx context.Context, client httpClient, endpoint string, values url.Values, intf interface{}, d Debug) error {
	reqBody := strings.NewReader(values.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return doPost(client, req, newJSONParser(intf), d)
}

func getResource(ctx context.Context, client httpClient, endpoint, token string, values url.Values, intf interface{}, d Debug) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	req.URL.RawQuery = values.Encode()

	return doPost(client, req, newJSONParser(intf), d)
}

func parseAdminResponse(ctx context.Context, client httpClient, method string, teamName string, values url.Values, intf interface{}, d Debug) error {
	endpoint := fmt.Sprintf(WEBAPIURLFormat, teamName, method, time.Now().Unix())
	return postForm(ctx, client, endpoint, values, intf, d)
}

func logResponse(resp *http.Response, d Debug) error {
	if d.Debug() {
		text, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return err
		}
		d.Debugln(string(text))
	}

	return nil
}

func okJSONHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json")
	response, _ := json.Marshal(SlackResponse{
		Ok: true,
	})
	rw.Write(response)
}

func checkStatusCode(resp *http.Response, d Debug) error {
	if resp.StatusCode == http.StatusTooManyRequests && resp.Header.Get("Retry-After") != "" {
		retry, err := strconv.ParseInt(resp.Header.Get("Retry-After"), 10, 64)
		if err != nil {
			return err
		}
		return &RateLimitedError{time.Duration(retry) * time.Second}
	}

	// Slack seems to send an HTML body along with 5xx error codes. Don't parse it.
	if resp.StatusCode != http.StatusOK {
		logResponse(resp, d)
		return StatusCodeError{Code: resp.StatusCode, Status: resp.Status}
	}

	return nil
}

type responseParser func(*http.Response) error

func newJSONParser(dst interface{}) responseParser {
	return func(resp *http.Response) error {
		if dst == nil {
			return nil
		}
		return json.NewDecoder(resp.Body).Decode(dst)
	}
}

func newTextParser(dst interface{}) responseParser {
	return func(resp *http.Response) error {
		if dst == nil {
			return nil
		}

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		if !bytes.Equal(b, []byte("ok")) {
			return errors.New(string(b))
		}

		return nil
	}
}

func newContentTypeParser(dst interface{}) responseParser {
	return func(req *http.Response) (err error) {
		var (
			ctype string
		)

		if ctype, _, err = mime.ParseMediaType(req.Header.Get("Content-Type")); err != nil {
			return err
		}

		switch ctype {
		case "application/json":
			return newJSONParser(dst)(req)
		default:
			return newTextParser(dst)(req)
		}
	}
}
