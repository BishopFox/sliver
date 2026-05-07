// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package genai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"log"
	"math"
	"net/http"
	"net/textproto"
	"net/url"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"
)

const maxChunkSize = 8 * 1024 * 1024 // 8 MB chunk size
const maxRetryCount = 3
const initialRetryDelay = time.Second
const delayMultiplier = 2

type apiClient struct {
	clientConfig *ClientConfig
}

// sendStreamRequest issues an server streaming API request and returns a map of the response contents.
func sendStreamRequest[T responseStream[R], R any](ctx context.Context, ac *apiClient, path string, method string, body map[string]any, httpOptions *HTTPOptions, output *responseStream[R]) error {
	req, httpOptions, err := buildRequest(ctx, ac, path, body, method, httpOptions)
	if err != nil {
		return err
	}

	// Handle context timeout.
	// The request's context deadline is set using [HTTPOptions.Timeout].
	// [ClientConfig.HTTPClient.Timeout] does not affect the context deadline for the request.
	// [ClientConfig.HTTPClient.Timeout] is used along with `x-server-timeout` header in order to
	// get the end-to-end timeout value for logging.
	requestContext := ctx
	timeout := httpOptions.Timeout
	var cancel context.CancelFunc
	if timeout != nil && *timeout > 0*time.Second && isTimeoutBeforeDeadline(ctx, *timeout) {
		requestContext, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}
	req = req.WithContext(requestContext)

	resp, err := doRequest(ac, req)
	if err != nil {
		return err
	}

	// resp.Body will be closed by the iterator
	return deserializeStreamResponse(resp, output)
}

// sendRequest issues an API request and returns a map of the response contents.
func sendRequest(ctx context.Context, ac *apiClient, path string, method string, body map[string]any, httpOptions *HTTPOptions) (map[string]any, error) {

	req, httpOptions, err := buildRequest(ctx, ac, path, body, method, httpOptions)
	if err != nil {
		return nil, err
	}

	requestContext := ctx
	timeout := httpOptions.Timeout
	var cancel context.CancelFunc
	if timeout != nil && *timeout > 0*time.Second && isTimeoutBeforeDeadline(ctx, *timeout) {
		requestContext, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}
	req = req.WithContext(requestContext)

	resp, err := doRequest(ac, req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	return deserializeUnaryResponse(resp)
}

func downloadFile(ctx context.Context, ac *apiClient, path string, httpOptions *HTTPOptions) ([]byte, error) {
	// The client and request timeout are not used for downloadFile.
	// TODO(b/427540996): implement timeout.
	req, _, err := buildRequest(ctx, ac, path, nil, http.MethodGet, httpOptions)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	resp, err := doRequest(ac, req)
	if err != nil {
		return nil, err
	}

	return io.ReadAll(resp.Body)
}

func mapToStruct[R any](input map[string]any, output *R) error {
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(input)
	if err != nil {
		return fmt.Errorf("mapToStruct: error encoding input %#v: %w", input, err)
	}
	err = json.Unmarshal(b.Bytes(), output)
	if err != nil {
		return fmt.Errorf("mapToStruct: error unmarshalling input %#v: %w", input, err)
	}
	return nil
}

func (ac *apiClient) createAPIURL(suffix, method string, httpOptions *HTTPOptions) (*url.URL, error) {
	path, query, _ := strings.Cut(suffix, "?")

	u, err := url.Parse(httpOptions.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("createAPIURL: error parsing base URL: %w", err)
	}

	var finalURL *url.URL
	if ac.clientConfig.Backend == BackendVertexAI {
		queryVertexBaseModel := method == http.MethodGet && strings.HasPrefix(path, "publishers/google/models")
		if ac.clientConfig.APIKey == "" && (!strings.HasPrefix(path, "projects/") && !queryVertexBaseModel) {
			path = fmt.Sprintf("projects/%s/locations/%s/%s", ac.clientConfig.Project, ac.clientConfig.Location, path)
		}
		finalURL = u.JoinPath(httpOptions.APIVersion, path)
	} else {
		if !strings.Contains(path, fmt.Sprintf("/%s/", httpOptions.APIVersion)) {
			path = fmt.Sprintf("%s/%s", httpOptions.APIVersion, path)
		}
		finalURL = u.JoinPath(path)
	}

	finalURL.RawQuery = query
	return finalURL, nil
}

// patchHTTPOptions merges two HttpOptions objects, creating a new one.
// Fields from patchOptions will overwrite fields from options.
func patchHTTPOptions(options, patchOptions HTTPOptions) (*HTTPOptions, error) {
	// Start with a shallow copy of the base options.
	copyOption := HTTPOptions{Headers: http.Header{}}
	err := deepCopy(options, &copyOption)
	if err != nil {
		return nil, err
	}

	// Deep copy the Headers map to avoid modifying the original options' map.
	// The Python code effectively does this by creating a new dictionary.
	mergedHeaders := http.Header{}
	for k, v := range options.Headers {
		mergedHeaders[textproto.CanonicalMIMEHeaderKey(k)] = v
	}
	for k, v := range patchOptions.Headers {
		mergedHeaders[textproto.CanonicalMIMEHeaderKey(k)] = v
	}
	copyOption.Headers = mergedHeaders

	// BaseURL and APIVersion is value type because explicitly setting
	// request HTTPOption to empty string won't override the client HTTPOption.
	if patchOptions.BaseURL != "" {
		copyOption.BaseURL = patchOptions.BaseURL
	}
	if patchOptions.APIVersion != "" {
		copyOption.APIVersion = patchOptions.APIVersion
	}
	if patchOptions.ExtrasRequestProvider != nil {
		copyOption.ExtrasRequestProvider = patchOptions.ExtrasRequestProvider
	}
	if patchOptions.ExtraBody != nil {
		copyOption.ExtraBody = patchOptions.ExtraBody
	}
	// Request timeout config overrides client timeout config.
	// So we need a pointer type so that we know the request timeout
	// is explicitly set or not.
	// Especially when request timeout is explicitly set to Ptr[int32](0),
	// then it means no timeout regardless client timeout is non-zero.
	if patchOptions.Timeout != nil {
		copyOption.Timeout = patchOptions.Timeout
	}
	appendSDKHeaders(copyOption.Headers)

	return &copyOption, nil
}

// appendLibraryVersionHeaders appends telemetry headers to the headers map.
// It modifies the map in place.
func appendSDKHeaders(headers http.Header) {
	if headers == nil {
		return
	}

	libraryLabel := fmt.Sprintf("google-genai-sdk/%s", version)
	languageLabel := fmt.Sprintf("gl-go/%s", runtime.Version())
	versionHeaderValue := fmt.Sprintf("%s %s", libraryLabel, languageLabel)

	if !slices.Contains(headers.Values("user-agent"), versionHeaderValue) {
		headers.Add("user-agent", versionHeaderValue)
	}

	if !slices.Contains(headers.Values("x-goog-api-client"), versionHeaderValue) {
		headers.Add("x-goog-api-client", versionHeaderValue)
	}
}

func buildRequest(ctx context.Context, ac *apiClient, path string, body map[string]any, method string, httpOptions *HTTPOptions) (*http.Request, *HTTPOptions, error) {
	patchedHTTPOptions, err := patchHTTPOptions(ac.clientConfig.HTTPOptions, *httpOptions)
	if err != nil {
		return nil, nil, err
	}
	url, err := ac.createAPIURL(path, method, patchedHTTPOptions)
	if err != nil {
		return nil, nil, err
	}

	if patchedHTTPOptions.ExtraBody != nil {
		recursiveMapMerge(body, patchedHTTPOptions.ExtraBody)
	}

	if patchedHTTPOptions.ExtrasRequestProvider != nil {
		body = httpOptions.ExtrasRequestProvider(body)
	}

	b := new(bytes.Buffer)
	if len(body) > 0 {
		if err := json.NewEncoder(b).Encode(body); err != nil {
			return nil, nil, fmt.Errorf("buildRequest: error encoding body %#v: %w", body, err)
		}
	}

	// Create a new HTTP request
	req, err := http.NewRequest(method, url.String(), b)
	if err != nil {
		return nil, nil, err
	}
	// Set headers
	req.Header = patchedHTTPOptions.Headers
	timeoutSeconds := inferTimeout(ctx, ac, patchedHTTPOptions.Timeout).Seconds()
	if timeoutSeconds > 0 {
		req.Header.Set("x-server-timeout", strconv.FormatInt(int64(math.Ceil(timeoutSeconds)), 10))
	}

	req.Header.Set("Content-Type", "application/json")
	if ac.clientConfig.APIKey != "" {
		req.Header.Set("x-goog-api-key", ac.clientConfig.APIKey)
	}

	return req, patchedHTTPOptions, nil
}

// recursiveMapMerge recursively merges key-value pairs from a source map (`src`)
// into a destination map (`dest`), modifying `dest` in-place.
//
// If a key exists in both maps and both corresponding values are maps
// of type `map[string]any`, it merges them recursively. Otherwise, the value
// from `src` overwrites the value in `dest`.
//
// The function logs a warning if a key's value in `dest` is overwritten by a
// value of a different type from `src`.
func recursiveMapMerge(dest, src map[string]any) {
	if dest == nil || src == nil {
		return
	}
	for key, value := range src {
		targetVal, keyExists := dest[key]
		destMap, isDestMap := targetVal.(map[string]any)
		srcMap, isSrcMap := value.(map[string]any)

		if keyExists && isDestMap && isSrcMap {
			recursiveMapMerge(destMap, srcMap)
		} else if keyExists && targetVal != nil && value != nil &&
			reflect.TypeOf(targetVal) != reflect.TypeOf(value) {
			log.Printf("Warning: Type mismatch for key '%s'. Existing type: %T, new type: %T. Overwriting.", key, targetVal, value)
			dest[key] = value
		} else {
			dest[key] = value
		}
	}
}

// TODO(b/428730853): HTTP Client timeout should be considered.
func isTimeoutBeforeDeadline(ctx context.Context, timeout time.Duration) bool {
	deadline, ok := ctx.Deadline()
	if !ok {
		return true
	}
	return timeout < time.Until(deadline)
}

func inferTimeout(ctx context.Context, ac *apiClient, requestTimeout *time.Duration) time.Duration {
	// ac.clientConfig.HTTPClient is not nil because it's initialized in the NewClient function.
	httpClientTimeout := ac.clientConfig.HTTPClient.Timeout
	contextTimeout := 0 * time.Second
	effectiveTimeout := 0 * time.Second

	if deadline, ok := ctx.Deadline(); ok {
		contextTimeout = time.Until(deadline)
	}

	// If context timeout or httpClient timeout is less than request timeout,
	// Then the smaller one takes precedence.
	if requestTimeout != nil {
		effectiveTimeout = *requestTimeout
	}
	if httpClientTimeout != 0 {
		if effectiveTimeout == 0 {
			effectiveTimeout = httpClientTimeout
		} else {
			effectiveTimeout = min(effectiveTimeout, httpClientTimeout)
		}
	}
	if contextTimeout != 0 {
		if effectiveTimeout == 0 {
			effectiveTimeout = contextTimeout
		} else {
			effectiveTimeout = min(effectiveTimeout, contextTimeout)
		}
	}
	return effectiveTimeout
}

func doRequest(ac *apiClient, req *http.Request) (*http.Response, error) {
	// Create a new HTTP client and send the request
	client := ac.clientConfig.HTTPClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("doRequest: error sending request: %w", err)
	}
	return resp, nil
}

func deserializeUnaryResponse(resp *http.Response) (map[string]any, error) {
	if !httpStatusOk(resp) {
		return nil, newAPIError(resp)
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	output := make(map[string]any)
	if len(respBody) > 0 {
		err = json.Unmarshal(respBody, &output)
		if err != nil {
			return nil, fmt.Errorf("deserializeUnaryResponse: error unmarshalling response: %w\n%s", err, respBody)
		}
	}

	httpResponse := map[string]any{
		"headers": resp.Header,
	}
	output["sdkHttpResponse"] = httpResponse
	return output, nil
}

type responseStream[R any] struct {
	r  *bufio.Scanner
	rc io.ReadCloser
	h  http.Header
}

func iterateResponseStream[R any](rs *responseStream[R], responseConverter func(responseMap map[string]any) (*R, error)) iter.Seq2[*R, error] {
	return func(yield func(*R, error) bool) {
		defer func() {
			// Close the response body range over function is done.
			if err := rs.rc.Close(); err != nil {
				log.Printf("Error closing response body: %v", err)
			}
		}()
		for rs.r.Scan() {
			line := rs.r.Bytes()
			if len(line) == 0 {
				continue
			}
			prefix, data, _ := bytes.Cut(line, []byte(":"))
			switch string(prefix) {
			case "data":
				// Step 1: Unmarshal the JSON into a map[string]any so that we can call fromConverter
				// in Step 2.
				respRaw := make(map[string]any)
				if err := json.Unmarshal(data, &respRaw); err != nil {
					err = fmt.Errorf("iterateResponseStream: error unmarshalling data %s:%s. error: %w", string(prefix), string(data), err)
					if !yield(nil, err) {
						return
					}
				}
				// Step 2: The toStruct function calls fromConverter(handle Vertex and MLDev schema
				// difference and get a unified response). Then toStruct function converts the unified
				// response from map[string]any to struct type.
				// var resp = new(R)
				resp, err := responseConverter(respRaw)
				if err != nil {
					if !yield(nil, err) {
						return
					}
				}

				// Step 3: Add the sdkHttpResponse to the response.
				v := reflect.ValueOf(resp).Elem()
				if v.Kind() == reflect.Struct {
					field := v.FieldByName("SDKHTTPResponse")
					if field.IsValid() && field.CanSet() {
						if field.IsNil() {
							field.Set(reflect.ValueOf(&HTTPResponse{}))
						}
						field.Interface().(*HTTPResponse).Headers = rs.h
					}
				}

				// Step 4: yield the response.
				if !yield(resp, nil) {
					return
				}
			default:
				var err error
				if len(line) > 0 {
					var respWithError = new(responseWithError)
					// Stream chunk that doesn't matches error format.
					if marshalErr := json.Unmarshal(line, respWithError); marshalErr != nil {
						err = fmt.Errorf("iterateResponseStream: invalid stream chunk: %s:%s", string(prefix), string(data))
					}
					// Stream chunk that matches error format.
					if respWithError.ErrorInfo != nil {
						err = *respWithError.ErrorInfo
					}
				}
				if err == nil {
					err = fmt.Errorf("iterateResponseStream: invalid stream chunk: %s:%s", string(prefix), string(data))
				}
				if !yield(nil, err) {
					return
				}
			}
		}
		if rs.r.Err() != nil {
			if rs.r.Err() == bufio.ErrTooLong {
				log.Printf("The response is too large to process in streaming mode. Please use a non-streaming method.")
			}
			log.Printf("Error %v", rs.r.Err())
		}
	}
}

// APIError contains an error response from the server.
type APIError struct {
	// Code is the HTTP response status code.
	Code int `json:"code,omitempty"`
	// Message is the server response message.
	Message string `json:"message,omitempty"`
	// Status is the server response status.
	Status string `json:"status,omitempty"`
	// Details field provides more context to an error.
	Details []map[string]any `json:"details,omitempty"`
}

type responseWithError struct {
	ErrorInfo *APIError `json:"error,omitempty"`
}

func newAPIError(resp *http.Response) error {
	var respWithError = new(responseWithError)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("newAPIError: error reading response body: %w. Response: %v", err, string(body))
	}

	if len(body) > 0 {
		if err := json.Unmarshal(body, respWithError); err != nil {
			// Handle plain text error message. File upload backend doesn't return json error message.
			return APIError{Code: resp.StatusCode, Status: resp.Status, Message: string(body)}
		}

		// Check if we successfully parsed an error response
		if respWithError.ErrorInfo != nil {
			return *respWithError.ErrorInfo
		}

		// Valid JSON but no error field - treat as generic error with body content
		return APIError{Code: resp.StatusCode, Status: resp.Status, Message: string(body)}
	}
	return APIError{Code: resp.StatusCode, Status: resp.Status}
}

// Error returns a string representation of the APIError.
func (e APIError) Error() string {
	return fmt.Sprintf(
		"Error %d, Message: %s, Status: %s, Details: %v",
		e.Code, e.Message, e.Status, e.Details,
	)
}

func httpStatusOk(resp *http.Response) bool {
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func deserializeStreamResponse[T responseStream[R], R any](resp *http.Response, output *responseStream[R]) error {
	if !httpStatusOk(resp) {
		defer resp.Body.Close()
		return newAPIError(resp)
	}
	output.r = bufio.NewScanner(resp.Body)
	// Scanner default buffer max size is 64*1024 (64KB).
	// We provide 1KB byte buffer to the scanner and set max to 256MB.
	// When data exceed 1KB, then scanner will allocate new memory up to 256MB.
	// When data exceed 256MB, scanner will stop and returns err: bufio.ErrTooLong.
	output.r.Buffer(make([]byte, 1024), 268435456)

	output.r.Split(scan)
	output.rc = resp.Body
	output.h = resp.Header
	return nil
}

// dropCR drops a terminal \r from the data.
func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

func scan(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	// Look for two consecutive newlines in the data
	if i := bytes.Index(data, []byte("\n\n")); i >= 0 {
		// We have a full two-newline-terminated token.
		return i + 2, dropCR(data[0:i]), nil
	}

	// Handle the case of Windows-style newlines (\r\n\r\n)
	if i := bytes.Index(data, []byte("\r\n\r\n")); i >= 0 {
		// We have a full Windows-style two-newline-terminated token.
		return i + 4, dropCR(data[0:i]), nil
	}

	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), dropCR(data), nil
	}
	// Request more data.
	return 0, nil, nil
}

func (ac *apiClient) upload(ctx context.Context, r io.Reader, uploadURL string, httpOptions *HTTPOptions) (map[string]any, error) {
	var offset int64 = 0
	var resp *http.Response
	var respBody map[string]any
	var uploadCommand = "upload"

	buffer := make([]byte, maxChunkSize)
	for {
		bytesRead, err := io.ReadFull(r, buffer)
		// Check both EOF and UnexpectedEOF errors.
		// ErrUnexpectedEOF: Reading a file file_size%maxChunkSize<len(buffer).
		// EOF: Reading a file file_size%maxChunkSize==0. The underlying reader return 0 bytes buffer and EOF at next call.
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			uploadCommand += ", finalize"
		} else if err != nil {
			return nil, fmt.Errorf("Failed to read bytes from file at offset %d: %w. Bytes actually read: %d", offset, err, bytesRead)
		}
		for attempt := 0; attempt < maxRetryCount; attempt++ {
			patchedHTTPOptions, err := patchHTTPOptions(ac.clientConfig.HTTPOptions, *httpOptions)
			if err != nil {
				return nil, err
			}

			// TODO(b/427540996): Support timeout.
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bytes.NewReader(buffer[:bytesRead]))
			if err != nil {
				return nil, fmt.Errorf("Failed to create upload request for chunk at offset %d: %w", offset, err)
			}

			req.Header = patchedHTTPOptions.Headers
			req.Header.Set("Content-Type", "application/json")
			if ac.clientConfig.APIKey != "" {
				req.Header.Set("x-goog-api-key", ac.clientConfig.APIKey)
			}
			// TODO(b/427540996): Add timeout logging.

			req.Header.Set("X-Goog-Upload-Command", uploadCommand)
			req.Header.Set("X-Goog-Upload-Offset", strconv.FormatInt(offset, 10))
			req.Header.Set("Content-Length", strconv.FormatInt(int64(bytesRead), 10))
			resp, err = doRequest(ac, req)
			if err != nil {
				return nil, fmt.Errorf("upload request failed for chunk at offset %d: %w", offset, err)
			}
			if resp.Header.Get("X-Goog-Upload-Status") != "" {
				break
			}
			resp.Body.Close()

			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("upload aborted while waiting to retry (attempt %d, offset %d): %w", attempt+1, offset, ctx.Err())
			case <-time.After(initialRetryDelay * time.Duration(delayMultiplier^attempt)):
				// Sleep completed, continue to the next attempt.
			}
		}
		defer resp.Body.Close()

		respBody, err = deserializeUnaryResponse(resp)
		if err != nil {
			return nil, fmt.Errorf("response body is invalid for chunk at offset %d: %w", offset, err)
		}

		offset += int64(bytesRead)

		uploadStatus := resp.Header.Get("X-Goog-Upload-Status")

		if uploadStatus != "final" && strings.Contains(uploadCommand, "finalize") {
			return nil, fmt.Errorf("send finalize command but doesn't receive final status. Offset %d, Bytes read: %d, Upload status: %s", offset, bytesRead, uploadStatus)
		}
		if uploadStatus != "active" {
			// Upload is complete ('final') or interrupted ('cancelled', etc.)
			break
		}
	}

	if resp == nil {
		return nil, fmt.Errorf("Upload request failed. No response received")
	}

	finalUploadStatus := resp.Header.Get("X-Goog-Upload-Status")
	if finalUploadStatus != "final" {
		return nil, fmt.Errorf("Failed to upload file: Upload status is not finalized")
	}

	return respBody, nil
}

func (ac *apiClient) uploadFile(ctx context.Context, r io.Reader, uploadURL string, httpOptions *HTTPOptions) (*File, error) {
	respBody, err := ac.upload(ctx, r, uploadURL, httpOptions)
	if err != nil {
		return nil, err // Propagate any errors from the upload process
	}
	if respBody == nil {
		return nil, fmt.Errorf("upload completed but response body was empty")
	}

	var response = new(File)
	err = mapToStruct(respBody["file"].(map[string]any), &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (ac *apiClient) uploadToFileSearchStore(ctx context.Context, r io.Reader, uploadURL string, httpOptions *HTTPOptions) (*UploadToFileSearchStoreOperation, error) {
	respBody, err := ac.upload(ctx, r, uploadURL, httpOptions)
	if err != nil {
		return nil, err // Propagate any errors from the upload process
	}
	if respBody == nil {
		return nil, fmt.Errorf("upload completed but response body was empty")
	}

	var response = new(UploadToFileSearchStoreOperation)
	err = mapToStruct(respBody, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}
