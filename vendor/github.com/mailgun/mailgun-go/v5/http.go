package mailgun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/mailgun/errors"
)

var invalidURL = regexp.MustCompile(`/v\d+.*`)

type httpRequest struct {
	URL               string
	Parameters        map[string][]string
	Headers           map[string]string
	BasicAuthUser     string
	BasicAuthPassword string
	// TODO(vtopc): get rid of this, should be (*Client).Do(*httpRequest):
	Client *http.Client
}

type httpResponse struct {
	Code   int
	Data   []byte
	Header http.Header
}

type payload interface {
	getPayloadBuffer() (*bytes.Buffer, error)
	getContentType() (string, error)
	getValues() []keyValuePair
}

type keyValuePair struct {
	key   string
	value string
}

type keyNameRC struct {
	key   string
	name  string
	value io.ReadCloser
}

type keyNameBuff struct {
	key   string
	name  string
	value []byte
}

type FormDataPayload struct {
	contentType string
	Values      []keyValuePair
	Files       []keyValuePair
	ReadClosers []keyNameRC
	Buffers     []keyNameBuff
}

type urlEncodedPayload struct {
	Values []keyValuePair
}

type jsonEncodedPayload struct {
	payload any
}

func newHTTPRequest(uri string) *httpRequest {
	return &httpRequest{URL: uri, Client: http.DefaultClient}
}

func (r *httpRequest) addParameter(name, value string) {
	if r.Parameters == nil {
		r.Parameters = make(map[string][]string)
	}
	r.Parameters[name] = append(r.Parameters[name], value)
}

// TODO(vtopc): get rid of this, should be (*Client).Do(*httpRequest)
func (r *httpRequest) setClient(c *http.Client) {
	r.Client = c
}

func (r *httpRequest) setBasicAuth(user, password string) {
	r.BasicAuthUser = user
	r.BasicAuthPassword = password
}

func newJSONEncodedPayload(payload any) *jsonEncodedPayload {
	return &jsonEncodedPayload{payload: payload}
}

func (j *jsonEncodedPayload) getPayloadBuffer() (*bytes.Buffer, error) {
	b, err := json.Marshal(j.payload)
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(b), nil
}

func (*jsonEncodedPayload) getContentType() (string, error) {
	return "application/json", nil
}

func (*jsonEncodedPayload) getValues() []keyValuePair {
	return nil
}

func newUrlEncodedPayload() *urlEncodedPayload {
	return &urlEncodedPayload{}
}

func (f *urlEncodedPayload) addValue(key, value string) {
	f.Values = append(f.Values, keyValuePair{key: key, value: value})
}

func (f *urlEncodedPayload) getPayloadBuffer() (*bytes.Buffer, error) {
	data := url.Values{}
	for _, keyVal := range f.Values {
		data.Add(keyVal.key, keyVal.value)
	}

	return bytes.NewBufferString(data.Encode()), nil
}

func (*urlEncodedPayload) getContentType() (string, error) {
	return "application/x-www-form-urlencoded", nil
}

func (f *urlEncodedPayload) getValues() []keyValuePair {
	return f.Values
}

func (r *httpResponse) parseFromJSON(v any) error {
	return json.Unmarshal(r.Data, v)
}

func NewFormDataPayload() *FormDataPayload {
	return &FormDataPayload{}
}

func (f *FormDataPayload) getValues() []keyValuePair {
	return f.Values
}

func (f *FormDataPayload) addValue(key, value string) {
	f.Values = append(f.Values, keyValuePair{key: key, value: value})
}

func (f *FormDataPayload) addFile(key, file string) {
	f.Files = append(f.Files, keyValuePair{key: key, value: file})
}

func (f *FormDataPayload) addBuffer(key, file string, buff []byte) {
	f.Buffers = append(f.Buffers, keyNameBuff{key: key, name: file, value: buff})
}

func (f *FormDataPayload) addReadCloser(key, name string, rc io.ReadCloser) {
	f.ReadClosers = append(f.ReadClosers, keyNameRC{key: key, name: name, value: rc})
}

func (f *FormDataPayload) getPayloadBuffer() (*bytes.Buffer, error) {
	data := &bytes.Buffer{}
	writer := multipart.NewWriter(data)
	defer writer.Close()

	for _, keyVal := range f.Values {
		tmp, err := writer.CreateFormField(keyVal.key)
		if err != nil {
			return nil, err
		}

		_, err = tmp.Write([]byte(keyVal.value))
		if err != nil {
			return nil, err
		}
	}

	for _, file := range f.Files {
		err := func() error {
			tmp, err := writer.CreateFormFile(file.key, path.Base(file.value))
			if err != nil {
				return err
			}

			fp, err := os.Open(file.value)
			if err != nil {
				return err
			}

			defer fp.Close()

			_, err = io.Copy(tmp, fp)
			if err != nil {
				return err
			}

			return nil
		}()
		if err != nil {
			return nil, err
		}
	}

	for _, file := range f.ReadClosers {
		err := func() error {
			tmp, err := writer.CreateFormFile(file.key, file.name)
			if err != nil {
				return err
			}

			defer file.value.Close()

			_, err = io.Copy(tmp, file.value)
			if err != nil {
				return err
			}

			return nil
		}()
		if err != nil {
			return nil, err
		}
	}

	for _, buff := range f.Buffers {
		tmp, err := writer.CreateFormFile(buff.key, buff.name)
		if err != nil {
			return nil, err
		}

		r := bytes.NewReader(buff.value)
		_, err = io.Copy(tmp, r)
		if err != nil {
			return nil, err
		}
	}

	// TODO(vtopc): getPayloadBuffer is not just a getter, it also sets the content type
	f.contentType = writer.FormDataContentType()

	return data, nil
}

func (f *FormDataPayload) getContentType() (string, error) {
	if f.contentType == "" {
		_, err := f.getPayloadBuffer()
		if err != nil {
			return "", err
		}
	}

	return f.contentType, nil
}

func (r *httpRequest) addHeader(name, value string) {
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	r.Headers[name] = value
}

func (r *httpRequest) makeGetRequest(ctx context.Context) (*httpResponse, error) {
	return r.do(ctx, http.MethodGet, nil)
}

func (r *httpRequest) makePostRequest(ctx context.Context, payload payload) (*httpResponse, error) {
	return r.do(ctx, http.MethodPost, payload)
}

func (r *httpRequest) makePutRequest(ctx context.Context, payload payload) (*httpResponse, error) {
	return r.do(ctx, http.MethodPut, payload)
}

func (r *httpRequest) makeDeleteRequest(ctx context.Context) (*httpResponse, error) {
	return r.do(ctx, http.MethodDelete, nil)
}

func (r *httpRequest) NewRequest(ctx context.Context, method string, payload payload) (*http.Request, error) {
	uri, err := r.generateUrlWithParameters()
	if err != nil {
		return nil, err
	}

	var body io.Reader
	if payload != nil {
		if body, err = payload.getPayloadBuffer(); err != nil {
			return nil, err
		}
	} else {
		body = nil
	}

	req, err := http.NewRequestWithContext(ctx, method, uri, body)
	if err != nil {
		return nil, err
	}

	if payload != nil {
		contentType, err := payload.getContentType()
		if err != nil {
			return nil, err
		}

		req.Header.Add("Content-Type", contentType)
	}

	if r.BasicAuthUser != "" && r.BasicAuthPassword != "" {
		req.SetBasicAuth(r.BasicAuthUser, r.BasicAuthPassword)
	}

	for header, value := range r.Headers {
		// Special case, override the Host header
		if header == "Host" {
			req.Host = value
			continue
		}
		req.Header.Add(header, value)
	}

	return req, nil
}

func (r *httpRequest) do(ctx context.Context, method string, payload payload) (*httpResponse, error) {
	req, err := r.NewRequest(ctx, method, payload)
	if err != nil {
		return nil, err
	}

	if Debug {
		fmt.Println(curlString(req, payload))
	}

	resp, err := r.Client.Do(req)
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) && urlErr != nil && errors.Is(urlErr.Err, io.EOF) {
			return nil, errors.Wrap(err, "remote server prematurely closed connection")
		}

		return nil, errors.Wrap(err, "while making http request")
	}

	defer resp.Body.Close()

	response := httpResponse{
		Code:   resp.StatusCode,
		Header: resp.Header,
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "while reading response body")
	}

	response.Data = responseBody

	return &response, nil
}

func (r *httpRequest) generateUrlWithParameters() (string, error) {
	uri, err := url.Parse(r.URL)
	if err != nil {
		return "", err
	}

	q := uri.Query()
	if len(r.Parameters) > 0 {
		for name, values := range r.Parameters {
			for _, value := range values {
				q.Add(name, value)
			}
		}
	}
	uri.RawQuery = q.Encode()

	return uri.String(), nil
}

func curlString(req *http.Request, p payload) string {
	parts := []string{"curl", "-i", "-X", req.Method, req.URL.String()}
	for key, value := range req.Header {
		if key == "Authorization" {
			parts = append(parts, fmt.Sprintf("-H \"%s: %s\"", key, "<redacted>"))
		} else {
			parts = append(parts, fmt.Sprintf("-H \"%s: %s\"", key, value[0]))
		}
	}

	// Special case for Host
	if req.Host != "" {
		parts = append(parts, fmt.Sprintf("-H \"Host: %s\"", req.Host))
	}

	if p != nil {
		contentType, _ := p.getContentType()
		if contentType == "application/json" {
			b, err := p.getPayloadBuffer()
			if err != nil {
				return "Unable to get payload buffer: " + err.Error()
			}
			parts = append(parts, fmt.Sprintf("--data '%s'", b.String()))
		} else {
			for _, param := range p.getValues() {
				parts = append(parts, fmt.Sprintf(" -F %s='%s'", param.key, param.value))
			}
		}
	}

	return strings.Join(parts, " ")
}
