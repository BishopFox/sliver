// Package mautrix implements the Matrix Client-Server API.
//
// Specification can be found at https://spec.matrix.org/v1.2/client-server-api/
package mautrix

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"go.mau.fi/util/ptr"
	"go.mau.fi/util/random"
	"go.mau.fi/util/retryafter"
	"golang.org/x/exp/maps"

	"maunium.net/go/mautrix/crypto/backup"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/pushrules"
)

type CryptoHelper interface {
	Encrypt(context.Context, id.RoomID, event.Type, any) (*event.EncryptedEventContent, error)
	Decrypt(context.Context, *event.Event) (*event.Event, error)
	WaitForSession(context.Context, id.RoomID, id.SenderKey, id.SessionID, time.Duration) bool
	RequestSession(context.Context, id.RoomID, id.SenderKey, id.SessionID, id.UserID, id.DeviceID)
	Init(context.Context) error
}

type VerificationHelper interface {
	// Init initializes the helper. This should be called before any other
	// methods.
	Init(context.Context) error

	// StartVerification starts an interactive verification flow with the given
	// user via a to-device event.
	StartVerification(ctx context.Context, to id.UserID) (id.VerificationTransactionID, error)
	// StartInRoomVerification starts an interactive verification flow with the
	// given user in the given room.
	StartInRoomVerification(ctx context.Context, roomID id.RoomID, to id.UserID) (id.VerificationTransactionID, error)

	// AcceptVerification accepts a verification request.
	AcceptVerification(ctx context.Context, txnID id.VerificationTransactionID) error
	// DismissVerification dismisses a verification request. This will not send
	// a cancellation to the other device. This method should only be called
	// *before* the request has been accepted and will error otherwise.
	DismissVerification(ctx context.Context, txnID id.VerificationTransactionID) error
	// CancelVerification cancels a verification request. This method should
	// only be called *after* the request has been accepted, although it will
	// not error if called beforehand.
	CancelVerification(ctx context.Context, txnID id.VerificationTransactionID, code event.VerificationCancelCode, reason string) error

	// HandleScannedQRData handles the data from a QR code scan.
	HandleScannedQRData(ctx context.Context, data []byte) error
	// ConfirmQRCodeScanned confirms that our QR code has been scanned.
	ConfirmQRCodeScanned(ctx context.Context, txnID id.VerificationTransactionID) error

	// StartSAS starts a SAS verification flow.
	StartSAS(ctx context.Context, txnID id.VerificationTransactionID) error
	// ConfirmSAS indicates that the user has confirmed that the SAS matches
	// SAS shown on the other user's device.
	ConfirmSAS(ctx context.Context, txnID id.VerificationTransactionID) error
}

// Client represents a Matrix client.
type Client struct {
	HomeserverURL  *url.URL     // The base homeserver URL
	UserID         id.UserID    // The user ID of the client. Used for forming HTTP paths which use the client's user ID.
	DeviceID       id.DeviceID  // The device ID of the client.
	AccessToken    string       // The access_token for the client.
	UserAgent      string       // The value for the User-Agent header
	Client         *http.Client // The underlying HTTP client which will be used to make HTTP requests.
	Syncer         Syncer       // The thing which can process /sync responses
	Store          SyncStore    // The thing which can store tokens/ids
	StateStore     StateStore
	Crypto         CryptoHelper
	Verification   VerificationHelper
	SpecVersions   *RespVersions
	ExternalClient *http.Client // The HTTP client used for external (not matrix) media HTTP requests.

	Log zerolog.Logger

	RequestHook  func(req *http.Request)
	ResponseHook func(req *http.Request, resp *http.Response, err error, duration time.Duration)

	UpdateRequestOnRetry func(req *http.Request, cause error) *http.Request

	SyncPresence event.Presence
	SyncTraceLog bool

	StreamSyncMinAge time.Duration

	// Number of times that mautrix will retry any HTTP request
	// if the request fails entirely or returns a HTTP gateway error (502-504)
	DefaultHTTPRetries int
	// Amount of time to wait between HTTP retries, defaults to 4 seconds
	DefaultHTTPBackoff time.Duration
	// Set to true to disable automatically sleeping on 429 errors.
	IgnoreRateLimit bool

	ResponseSizeLimit int64

	txnID int32

	// Should the ?user_id= query parameter be set in requests?
	// See https://spec.matrix.org/v1.6/application-service-api/#identity-assertion
	SetAppServiceUserID bool
	// Should the org.matrix.msc3202.device_id query parameter be set in requests?
	// See https://github.com/matrix-org/matrix-spec-proposals/pull/3202
	SetAppServiceDeviceID bool

	syncingID uint32 // Identifies the current Sync. Only one Sync can be active at any given time.
}

type ClientWellKnown struct {
	Homeserver     HomeserverInfo     `json:"m.homeserver"`
	IdentityServer IdentityServerInfo `json:"m.identity_server"`
}

type HomeserverInfo struct {
	BaseURL string `json:"base_url"`
}

type IdentityServerInfo struct {
	BaseURL string `json:"base_url"`
}

// DiscoverClientAPI resolves the client API URL from a Matrix server name.
// Use ParseUserID to extract the server name from a user ID.
// https://spec.matrix.org/v1.2/client-server-api/#server-discovery
func DiscoverClientAPI(ctx context.Context, serverName string) (*ClientWellKnown, error) {
	return DiscoverClientAPIWithClient(ctx, &http.Client{Timeout: 30 * time.Second}, serverName)
}

const WellKnownMaxSize = 64 * 1024

func DiscoverClientAPIWithClient(ctx context.Context, client *http.Client, serverName string) (*ClientWellKnown, error) {
	wellKnownURL := url.URL{
		Scheme: "https",
		Host:   serverName,
		Path:   "/.well-known/matrix/client",
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wellKnownURL.String(), nil)
	if err != nil {
		return nil, err
	}

	if runtime.GOOS != "js" {
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", DefaultUserAgent+" (.well-known fetcher)")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	} else if resp.ContentLength > WellKnownMaxSize {
		return nil, errors.New(".well-known response too large")
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, WellKnownMaxSize))
	if err != nil {
		return nil, err
	} else if len(data) >= WellKnownMaxSize {
		return nil, errors.New(".well-known response too large")
	}

	var wellKnown ClientWellKnown
	err = json.Unmarshal(data, &wellKnown)
	if err != nil {
		return nil, errors.New(".well-known response not JSON")
	}

	return &wellKnown, nil
}

// SetCredentials sets the user ID and access token on this client instance.
//
// Deprecated: use the StoreCredentials field in ReqLogin instead.
func (cli *Client) SetCredentials(userID id.UserID, accessToken string) {
	cli.AccessToken = accessToken
	cli.UserID = userID
}

// ClearCredentials removes the user ID and access token on this client instance.
func (cli *Client) ClearCredentials() {
	cli.AccessToken = ""
	cli.UserID = ""
	cli.DeviceID = ""
}

// Sync starts syncing with the provided Homeserver. If Sync() is called twice then the first sync will be stopped and the
// error will be nil.
//
// This function will block until a fatal /sync error occurs, so it should almost always be started as a new goroutine.
// Fatal sync errors can be caused by:
//   - The failure to create a filter.
//   - Client.Syncer.OnFailedSync returning an error in response to a failed sync.
//   - Client.Syncer.ProcessResponse returning an error.
//
// If you wish to continue retrying in spite of these fatal errors, call Sync() again.
func (cli *Client) Sync() error {
	return cli.SyncWithContext(context.Background())
}

func (cli *Client) SyncWithContext(ctx context.Context) error {
	// Mark the client as syncing.
	// We will keep syncing until the syncing state changes. Either because
	// Sync is called or StopSync is called.
	syncingID := cli.incrementSyncingID()
	nextBatch, err := cli.Store.LoadNextBatch(ctx, cli.UserID)
	if err != nil {
		return err
	}
	filterID, err := cli.Store.LoadFilterID(ctx, cli.UserID)
	if err != nil {
		return err
	}

	if filterID == "" {
		filterJSON := cli.Syncer.GetFilterJSON(cli.UserID)
		resFilter, err := cli.CreateFilter(ctx, filterJSON)
		if err != nil {
			return err
		}
		filterID = resFilter.FilterID
		if err := cli.Store.SaveFilterID(ctx, cli.UserID, filterID); err != nil {
			return err
		}
	}
	lastSuccessfulSync := time.Now().Add(-cli.StreamSyncMinAge - 1*time.Hour)
	// Always do first sync with 0 timeout
	isFailing := true
	for {
		streamResp := false
		if cli.StreamSyncMinAge > 0 && time.Since(lastSuccessfulSync) > cli.StreamSyncMinAge {
			cli.Log.Debug().Msg("Last sync is old, will stream next response")
			streamResp = true
		}
		timeout := 30000
		if isFailing || nextBatch == "" {
			timeout = 0
		}
		resSync, err := cli.FullSyncRequest(ctx, ReqSync{
			Timeout:        timeout,
			Since:          nextBatch,
			FilterID:       filterID,
			FullState:      false,
			SetPresence:    cli.SyncPresence,
			StreamResponse: streamResp,
		})
		if err != nil {
			isFailing = true
			if ctx.Err() != nil {
				return ctx.Err()
			}
			duration, err2 := cli.Syncer.OnFailedSync(resSync, err)
			if err2 != nil {
				return err2
			}
			if duration <= 0 {
				continue
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(duration):
				continue
			}
		}
		isFailing = false
		lastSuccessfulSync = time.Now()

		// Check that the syncing state hasn't changed
		// Either because we've stopped syncing or another sync has been started.
		// We discard the response from our sync.
		if cli.getSyncingID() != syncingID {
			return nil
		}

		// Save the token now *before* processing it. This means it's possible
		// to not process some events, but it means that we won't get constantly stuck processing
		// a malformed/buggy event which keeps making us panic.
		err = cli.Store.SaveNextBatch(ctx, cli.UserID, resSync.NextBatch)
		if err != nil {
			return err
		}
		if err = cli.Syncer.ProcessResponse(ctx, resSync, nextBatch); err != nil {
			return err
		}

		nextBatch = resSync.NextBatch
	}
}

func (cli *Client) incrementSyncingID() uint32 {
	return atomic.AddUint32(&cli.syncingID, 1)
}

func (cli *Client) getSyncingID() uint32 {
	return atomic.LoadUint32(&cli.syncingID)
}

// StopSync stops the ongoing sync started by Sync.
func (cli *Client) StopSync() {
	// Advance the syncing state so that any running Syncs will terminate.
	cli.incrementSyncingID()
}

type contextKey int

const (
	LogBodyContextKey contextKey = iota
	LogRequestIDContextKey
	MaxAttemptsContextKey
	SyncTokenContextKey
)

func (cli *Client) RequestStart(req *http.Request) {
	if cli != nil && cli.RequestHook != nil {
		cli.RequestHook(req)
	}
}

// WithMaxRetries updates the context to set the maximum number of retries for any HTTP requests made with the context.
//
// 0 means the request will only be attempted once and will not be retried.
// Negative values will remove the override and fallback to the defaults.
func WithMaxRetries(ctx context.Context, maxRetries int) context.Context {
	return context.WithValue(ctx, MaxAttemptsContextKey, maxRetries+1)
}

func (cli *Client) LogRequestDone(req *http.Request, resp *http.Response, err error, handlerErr error, contentLength int, duration time.Duration) {
	if cli == nil {
		return
	}
	var evt *zerolog.Event
	if errors.Is(err, context.Canceled) {
		evt = zerolog.Ctx(req.Context()).Warn()
	} else if err != nil {
		evt = zerolog.Ctx(req.Context()).Err(err)
	} else if handlerErr != nil {
		evt = zerolog.Ctx(req.Context()).Warn().
			AnErr("body_parse_err", handlerErr)
	} else if cli.SyncTraceLog && strings.HasSuffix(req.URL.Path, "/_matrix/client/v3/sync") {
		evt = zerolog.Ctx(req.Context()).Trace()
	} else {
		evt = zerolog.Ctx(req.Context()).Debug()
	}
	evt = evt.
		Str("method", req.Method).
		Str("url", req.URL.String()).
		Dur("duration", duration)
	if cli.ResponseHook != nil {
		cli.ResponseHook(req, resp, err, duration)
	}
	if resp != nil {
		mime := resp.Header.Get("Content-Type")
		length := resp.ContentLength
		if length == -1 && contentLength > 0 {
			length = int64(contentLength)
		}
		evt = evt.Int("status_code", resp.StatusCode).
			Int64("response_length", length).
			Str("response_mime", mime)
		if serverRequestID := resp.Header.Get("X-Beeper-Request-ID"); serverRequestID != "" {
			evt.Str("beeper_request_id", serverRequestID)
		}
	}
	if body := req.Context().Value(LogBodyContextKey); body != nil {
		evt.Interface("req_body", body)
	}
	if errors.Is(err, context.Canceled) {
		evt.Msg("Request canceled")
	} else if err != nil {
		evt.Msg("Request failed")
	} else if handlerErr != nil {
		evt.Msg("Request parsing failed")
	} else {
		evt.Msg("Request completed")
	}
}

func (cli *Client) MakeRequest(ctx context.Context, method string, httpURL string, reqBody any, resBody any) ([]byte, error) {
	return cli.MakeFullRequest(ctx, FullRequest{Method: method, URL: httpURL, RequestJSON: reqBody, ResponseJSON: resBody})
}

type ClientResponseHandler = func(req *http.Request, res *http.Response, responseJSON any, sizeLimit int64) ([]byte, error)

type FullRequest struct {
	Method            string
	URL               string
	Headers           http.Header
	RequestJSON       interface{}
	RequestBytes      []byte
	RequestBody       io.Reader
	RequestLength     int64
	ResponseJSON      interface{}
	MaxAttempts       int
	BackoffDuration   time.Duration
	SensitiveContent  bool
	Handler           ClientResponseHandler
	DontReadResponse  bool
	ResponseSizeLimit int64
	Logger            *zerolog.Logger
	Client            *http.Client
}

var requestID int32
var logSensitiveContent = os.Getenv("MAUTRIX_LOG_SENSITIVE_CONTENT") == "yes"

func (params *FullRequest) compileRequest(ctx context.Context) (*http.Request, error) {
	reqID := atomic.AddInt32(&requestID, 1)
	logger := zerolog.Ctx(ctx)
	if logger.GetLevel() == zerolog.Disabled || logger == zerolog.DefaultContextLogger {
		logger = params.Logger
	}
	ctx = logger.With().
		Int32("req_id", reqID).
		Logger().WithContext(ctx)

	var logBody any
	var reqBody io.Reader
	var reqLen int64
	if params.RequestJSON != nil {
		jsonStr, err := json.Marshal(params.RequestJSON)
		if err != nil {
			return nil, HTTPError{
				Message:      "failed to marshal JSON",
				WrappedError: err,
			}
		}
		if params.SensitiveContent && !logSensitiveContent {
			logBody = "<sensitive content omitted>"
		} else {
			logBody = params.RequestJSON
		}
		reqBody = bytes.NewReader(jsonStr)
		reqLen = int64(len(jsonStr))
	} else if params.RequestBytes != nil {
		logBody = fmt.Sprintf("<%d bytes>", len(params.RequestBytes))
		reqBody = bytes.NewReader(params.RequestBytes)
		reqLen = int64(len(params.RequestBytes))
	} else if params.RequestBody != nil {
		logBody = "<unknown stream of bytes>"
		reqLen = -1
		if params.RequestLength > 0 {
			logBody = fmt.Sprintf("<%d bytes>", params.RequestLength)
			reqLen = params.RequestLength
		} else if params.RequestLength == 0 {
			zerolog.Ctx(ctx).Warn().
				Msg("RequestBody passed without specifying request length")
		}
		reqBody = params.RequestBody
		if rsc, ok := params.RequestBody.(io.ReadSeekCloser); ok {
			// Prevent HTTP from closing the request body, it might be needed for retries
			reqBody = nopCloseSeeker{rsc}
		}
	} else if params.Method != http.MethodGet && params.Method != http.MethodHead {
		params.RequestJSON = struct{}{}
		logBody = params.RequestJSON
		reqBody = bytes.NewReader([]byte("{}"))
		reqLen = 2
	}
	ctx = context.WithValue(ctx, LogBodyContextKey, logBody)
	ctx = context.WithValue(ctx, LogRequestIDContextKey, int(reqID))
	req, err := http.NewRequestWithContext(ctx, params.Method, params.URL, reqBody)
	if err != nil {
		return nil, HTTPError{
			Message:      "failed to create request",
			WrappedError: err,
		}
	}
	if params.Headers != nil {
		req.Header = params.Headers
	}
	if params.RequestJSON != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.ContentLength = reqLen
	return req, nil
}

func (cli *Client) MakeFullRequest(ctx context.Context, params FullRequest) ([]byte, error) {
	data, _, err := cli.MakeFullRequestWithResp(ctx, params)
	return data, err
}

func (cli *Client) MakeFullRequestWithResp(ctx context.Context, params FullRequest) ([]byte, *http.Response, error) {
	if cli == nil {
		return nil, nil, ErrClientIsNil
	}
	if cli.HomeserverURL == nil || cli.HomeserverURL.Scheme == "" {
		return nil, nil, ErrClientHasNoHomeserver
	}
	if params.MaxAttempts == 0 {
		maxAttempts, ok := ctx.Value(MaxAttemptsContextKey).(int)
		if ok && maxAttempts > 0 {
			params.MaxAttempts = maxAttempts
		} else {
			params.MaxAttempts = 1 + cli.DefaultHTTPRetries
		}
	}
	if params.BackoffDuration == 0 {
		if cli.DefaultHTTPBackoff == 0 {
			params.BackoffDuration = 4 * time.Second
		} else {
			params.BackoffDuration = cli.DefaultHTTPBackoff
		}
	}
	if params.Logger == nil {
		params.Logger = &cli.Log
	}
	req, err := params.compileRequest(ctx)
	if err != nil {
		return nil, nil, err
	}
	if params.Handler == nil {
		if params.DontReadResponse {
			params.Handler = noopHandleResponse
		} else {
			params.Handler = handleNormalResponse
		}
	}
	if cli.UserAgent != "" {
		req.Header.Set("User-Agent", cli.UserAgent)
	}
	if len(cli.AccessToken) > 0 {
		req.Header.Set("Authorization", "Bearer "+cli.AccessToken)
	}
	if params.ResponseSizeLimit == 0 {
		params.ResponseSizeLimit = cli.ResponseSizeLimit
	}
	if params.ResponseSizeLimit == 0 {
		params.ResponseSizeLimit = DefaultResponseSizeLimit
	}
	if params.Client == nil {
		params.Client = cli.Client
	}
	return cli.executeCompiledRequest(
		req,
		params.MaxAttempts-1,
		params.BackoffDuration,
		params.ResponseJSON,
		params.Handler,
		params.DontReadResponse,
		params.ResponseSizeLimit,
		params.Client,
	)
}

func (cli *Client) cliOrContextLog(ctx context.Context) *zerolog.Logger {
	log := zerolog.Ctx(ctx)
	if log.GetLevel() == zerolog.Disabled || log == zerolog.DefaultContextLogger {
		return &cli.Log
	}
	return log
}

func (cli *Client) doRetry(
	req *http.Request,
	cause error,
	retries int,
	backoff time.Duration,
	responseJSON any,
	handler ClientResponseHandler,
	dontReadResponse bool,
	sizeLimit int64,
	client *http.Client,
) ([]byte, *http.Response, error) {
	log := zerolog.Ctx(req.Context())
	if req.Body != nil {
		var err error
		if req.GetBody != nil {
			req.Body, err = req.GetBody()
			if err != nil {
				log.Warn().Err(err).Msg("Failed to get new body to retry request")
				return nil, nil, cause
			}
		} else if bodySeeker, ok := req.Body.(io.ReadSeeker); ok {
			_, err = bodySeeker.Seek(0, io.SeekStart)
			if err != nil {
				log.Warn().Err(err).Msg("Failed to seek to beginning of request body")
				return nil, nil, cause
			}
		} else {
			log.Warn().Msg("Failed to get new body to retry request: GetBody is nil and Body is not an io.ReadSeeker")
			return nil, nil, cause
		}
	}
	log.Warn().Err(cause).
		Str("method", req.Method).
		Str("url", req.URL.String()).
		Int("retry_in_seconds", int(backoff.Seconds())).
		Msg("Request failed, retrying")
	select {
	case <-time.After(backoff):
	case <-req.Context().Done():
		return nil, nil, req.Context().Err()
	}
	if cli.UpdateRequestOnRetry != nil {
		req = cli.UpdateRequestOnRetry(req, cause)
	}
	return cli.executeCompiledRequest(req, retries-1, backoff*2, responseJSON, handler, dontReadResponse, sizeLimit, client)
}

func readResponseBody(req *http.Request, res *http.Response, limit int64) ([]byte, error) {
	if res.ContentLength > limit {
		return nil, HTTPError{
			Request:  req,
			Response: res,

			Message:      "not reading response",
			WrappedError: fmt.Errorf("%w (%.2f MiB)", ErrResponseTooLong, float64(res.ContentLength)/1024/1024),
		}
	}
	contents, err := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if err == nil && len(contents) > int(limit) {
		err = ErrBodyReadReachedLimit
	}
	if err != nil {
		return nil, HTTPError{
			Request:  req,
			Response: res,

			Message:      "failed to read response body",
			WrappedError: err,
		}
	}
	return contents, nil
}

func closeTemp(log *zerolog.Logger, file *os.File) {
	_ = file.Close()
	err := os.Remove(file.Name())
	if err != nil {
		log.Warn().Err(err).Str("file_name", file.Name()).Msg("Failed to remove response temp file")
	}
}

func streamResponse(req *http.Request, res *http.Response, responseJSON any, limit int64) ([]byte, error) {
	log := zerolog.Ctx(req.Context())
	file, err := os.CreateTemp("", "mautrix-response-")
	if err != nil {
		log.Warn().Err(err).Msg("Failed to create temporary file for streaming response")
		_, err = handleNormalResponse(req, res, responseJSON, limit)
		return nil, err
	}
	defer closeTemp(log, file)
	var n int64
	if n, err = io.Copy(file, io.LimitReader(res.Body, limit+1)); err != nil {
		return nil, fmt.Errorf("failed to copy response to file: %w", err)
	} else if n > limit {
		return nil, ErrBodyReadReachedLimit
	} else if _, err = file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek to beginning of response file: %w", err)
	} else if err = json.NewDecoder(file).Decode(responseJSON); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	} else {
		return nil, nil
	}
}

func noopHandleResponse(req *http.Request, res *http.Response, responseJSON any, limit int64) ([]byte, error) {
	return nil, nil
}

func handleNormalResponse(req *http.Request, res *http.Response, responseJSON any, limit int64) ([]byte, error) {
	if contents, err := readResponseBody(req, res, limit); err != nil {
		return nil, err
	} else if responseJSON == nil {
		return contents, nil
	} else if err = json.Unmarshal(contents, &responseJSON); err != nil {
		return nil, HTTPError{
			Request:  req,
			Response: res,

			Message:      "failed to unmarshal response body",
			ResponseBody: string(contents),
			WrappedError: err,
		}
	} else {
		return contents, nil
	}
}

const ErrorResponseSizeLimit = 512 * 1024

var DefaultResponseSizeLimit int64 = 512 * 1024 * 1024

func ParseErrorResponse(req *http.Request, res *http.Response) ([]byte, error) {
	defer res.Body.Close()
	contents, err := readResponseBody(req, res, ErrorResponseSizeLimit)
	if err != nil {
		return contents, err
	}

	respErr := &RespError{
		StatusCode: res.StatusCode,
	}
	if _ = json.Unmarshal(contents, respErr); respErr.ErrCode == "" {
		respErr = nil
	}

	return contents, HTTPError{
		Request:   req,
		Response:  res,
		RespError: respErr,
	}
}

func (cli *Client) executeCompiledRequest(
	req *http.Request,
	retries int,
	backoff time.Duration,
	responseJSON any,
	handler ClientResponseHandler,
	dontReadResponse bool,
	sizeLimit int64,
	client *http.Client,
) ([]byte, *http.Response, error) {
	cli.RequestStart(req)
	startTime := time.Now()
	res, err := client.Do(req)
	duration := time.Now().Sub(startTime)
	if res != nil && !dontReadResponse {
		defer res.Body.Close()
	}
	if err != nil {
		if retries > 0 && !errors.Is(err, context.Canceled) {
			return cli.doRetry(
				req, err, retries, backoff, responseJSON, handler, dontReadResponse, sizeLimit, client,
			)
		}
		err = HTTPError{
			Request:  req,
			Response: res,

			Message:      "request error",
			WrappedError: err,
		}
		cli.LogRequestDone(req, res, err, nil, 0, duration)
		return nil, res, err
	}

	if retries > 0 && retryafter.Should(res.StatusCode, !cli.IgnoreRateLimit) {
		backoff = retryafter.Parse(res.Header.Get("Retry-After"), backoff)
		return cli.doRetry(
			req, fmt.Errorf("HTTP %d", res.StatusCode), retries, backoff, responseJSON, handler, dontReadResponse, sizeLimit, client,
		)
	}

	var body []byte
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, err = ParseErrorResponse(req, res)
		cli.LogRequestDone(req, res, nil, nil, len(body), duration)
	} else {
		body, err = handler(req, res, responseJSON, sizeLimit)
		cli.LogRequestDone(req, res, nil, err, len(body), duration)
	}
	return body, res, err
}

// Whoami gets the user ID of the current user. See https://spec.matrix.org/v1.2/client-server-api/#get_matrixclientv3accountwhoami
func (cli *Client) Whoami(ctx context.Context) (resp *RespWhoami, err error) {
	urlPath := cli.BuildClientURL("v3", "account", "whoami")
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	return
}

// CreateFilter makes an HTTP request according to https://spec.matrix.org/v1.2/client-server-api/#post_matrixclientv3useruseridfilter
func (cli *Client) CreateFilter(ctx context.Context, filter *Filter) (resp *RespCreateFilter, err error) {
	urlPath := cli.BuildClientURL("v3", "user", cli.UserID, "filter")
	_, err = cli.MakeRequest(ctx, http.MethodPost, urlPath, filter, &resp)
	return
}

// SyncRequest makes an HTTP request according to https://spec.matrix.org/v1.2/client-server-api/#get_matrixclientv3sync
func (cli *Client) SyncRequest(ctx context.Context, timeout int, since, filterID string, fullState bool, setPresence event.Presence) (resp *RespSync, err error) {
	return cli.FullSyncRequest(ctx, ReqSync{
		Timeout:     timeout,
		Since:       since,
		FilterID:    filterID,
		FullState:   fullState,
		SetPresence: setPresence,
	})
}

type ReqSync struct {
	Timeout         int
	Since           string
	FilterID        string
	FullState       bool
	SetPresence     event.Presence
	StreamResponse  bool
	UseStateAfter   bool
	BeeperStreaming bool
	Client          *http.Client
}

func (req *ReqSync) BuildQuery() map[string]string {
	query := map[string]string{
		"timeout": strconv.Itoa(req.Timeout),
	}
	if req.Since != "" {
		query["since"] = req.Since
	}
	if req.FilterID != "" {
		query["filter"] = req.FilterID
	}
	if req.SetPresence != "" {
		query["set_presence"] = string(req.SetPresence)
	}
	if req.FullState {
		query["full_state"] = "true"
	}
	if req.UseStateAfter {
		query["use_state_after"] = "true"
	}
	if req.BeeperStreaming {
		query["com.beeper.streaming"] = "true"
	}
	return query
}

// FullSyncRequest makes an HTTP request according to https://spec.matrix.org/v1.2/client-server-api/#get_matrixclientv3sync
func (cli *Client) FullSyncRequest(ctx context.Context, req ReqSync) (resp *RespSync, err error) {
	urlPath := cli.BuildURLWithQuery(ClientURLPath{"v3", "sync"}, req.BuildQuery())
	fullReq := FullRequest{
		Method:       http.MethodGet,
		URL:          urlPath,
		ResponseJSON: &resp,
		Client:       req.Client,
		// We don't want automatic retries for SyncRequest, the Sync() wrapper handles those.
		MaxAttempts: 1,
	}
	if req.StreamResponse {
		fullReq.Handler = streamResponse
	}
	start := time.Now()
	_, err = cli.MakeFullRequest(ctx, fullReq)
	duration := time.Now().Sub(start)
	timeout := time.Duration(req.Timeout) * time.Millisecond
	buffer := 10 * time.Second
	if req.Since == "" {
		buffer = 1 * time.Minute
	}
	if err == nil && duration > timeout+buffer {
		cli.cliOrContextLog(ctx).Warn().
			Str("since", req.Since).
			Dur("duration", duration).
			Dur("timeout", timeout).
			Msg("Sync request took unusually long")
	}
	return
}

// RegisterAvailable checks if a username is valid and available for registration on the server.
//
// See https://spec.matrix.org/v1.4/client-server-api/#get_matrixclientv3registeravailable for more details
//
// This will always return an error if the username isn't available, so checking the actual response struct is generally
// not necessary. It is still returned for future-proofing. For a simple availability check, just check that the returned
// error is nil. `errors.Is` can be used to find the exact reason why a username isn't available:
//
//	_, err := cli.RegisterAvailable("cat")
//	if errors.Is(err, mautrix.MUserInUse) {
//		// Username is taken
//	} else if errors.Is(err, mautrix.MInvalidUsername) {
//		// Username is not valid
//	} else if errors.Is(err, mautrix.MExclusive) {
//		// Username is reserved for an appservice
//	} else if errors.Is(err, mautrix.MLimitExceeded) {
//		// Too many requests
//	} else if err != nil {
//		// Unknown error
//	} else {
//		// Username is available
//	}
func (cli *Client) RegisterAvailable(ctx context.Context, username string) (resp *RespRegisterAvailable, err error) {
	u := cli.BuildURLWithQuery(ClientURLPath{"v3", "register", "available"}, map[string]string{"username": username})
	_, err = cli.MakeRequest(ctx, http.MethodGet, u, nil, &resp)
	if err == nil && !resp.Available {
		err = fmt.Errorf(`request returned OK status without "available": true`)
	}
	return
}

func (cli *Client) register(ctx context.Context, url string, req *ReqRegister) (resp *RespRegister, uiaResp *RespUserInteractive, err error) {
	var bodyBytes []byte
	bodyBytes, err = cli.MakeFullRequest(ctx, FullRequest{
		Method:           http.MethodPost,
		URL:              url,
		RequestJSON:      req,
		SensitiveContent: len(req.Password) > 0,
	})
	if err != nil {
		httpErr, ok := err.(HTTPError)
		// if response has a 401 status, but doesn't have the errcode field, it's probably a UIA response.
		if ok && httpErr.IsStatus(http.StatusUnauthorized) && httpErr.RespError == nil {
			err = json.Unmarshal(bodyBytes, &uiaResp)
		}
	} else {
		// body should be RespRegister
		err = json.Unmarshal(bodyBytes, &resp)
	}
	return
}

// Register makes an HTTP request according to https://spec.matrix.org/v1.2/client-server-api/#post_matrixclientv3register
//
// Registers with kind=user. For kind=guest, see RegisterGuest.
func (cli *Client) Register(ctx context.Context, req *ReqRegister) (*RespRegister, *RespUserInteractive, error) {
	u := cli.BuildClientURL("v3", "register")
	return cli.register(ctx, u, req)
}

// RegisterGuest makes an HTTP request according to https://spec.matrix.org/v1.2/client-server-api/#post_matrixclientv3register
// with kind=guest.
//
// For kind=user, see Register.
func (cli *Client) RegisterGuest(ctx context.Context, req *ReqRegister) (*RespRegister, *RespUserInteractive, error) {
	query := map[string]string{
		"kind": "guest",
	}
	u := cli.BuildURLWithQuery(ClientURLPath{"v3", "register"}, query)
	return cli.register(ctx, u, req)
}

// RegisterDummy performs m.login.dummy registration according to https://spec.matrix.org/v1.2/client-server-api/#dummy-auth
//
// Only a username and password need to be provided on the ReqRegister struct. Most local/developer homeservers will allow registration
// this way. If the homeserver does not, an error is returned.
//
// This does not set credentials on the client instance. See SetCredentials() instead.
//
//	res, err := cli.RegisterDummy(&mautrix.ReqRegister{
//		Username: "alice",
//		Password: "wonderland",
//	})
//	if err != nil {
//		panic(err)
//	}
//	token := res.AccessToken
func (cli *Client) RegisterDummy(ctx context.Context, req *ReqRegister) (*RespRegister, error) {
	res, uia, err := cli.Register(ctx, req)
	if err != nil && uia == nil {
		return nil, err
	} else if uia == nil {
		return nil, errors.New("server did not return user-interactive auth flows")
	} else if !uia.HasSingleStageFlow(AuthTypeDummy) {
		return nil, errors.New("server does not support m.login.dummy")
	}
	req.Auth = BaseAuthData{Type: AuthTypeDummy, Session: uia.Session}
	res, _, err = cli.Register(ctx, req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// GetLoginFlows fetches the login flows that the homeserver supports using https://spec.matrix.org/v1.2/client-server-api/#get_matrixclientv3login
func (cli *Client) GetLoginFlows(ctx context.Context) (resp *RespLoginFlows, err error) {
	urlPath := cli.BuildClientURL("v3", "login")
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	return
}

// Login a user to the homeserver according to https://spec.matrix.org/v1.2/client-server-api/#post_matrixclientv3login
func (cli *Client) Login(ctx context.Context, req *ReqLogin) (resp *RespLogin, err error) {
	_, err = cli.MakeFullRequest(ctx, FullRequest{
		Method:           http.MethodPost,
		URL:              cli.BuildClientURL("v3", "login"),
		RequestJSON:      req,
		ResponseJSON:     &resp,
		SensitiveContent: len(req.Password) > 0 || len(req.Token) > 0,
	})
	if req.StoreCredentials && err == nil {
		cli.DeviceID = resp.DeviceID
		cli.AccessToken = resp.AccessToken
		cli.UserID = resp.UserID

		cli.Log.Debug().
			Str("user_id", cli.UserID.String()).
			Str("device_id", cli.DeviceID.String()).
			Msg("Stored credentials after login")
	}
	if req.StoreHomeserverURL && err == nil && resp.WellKnown != nil && len(resp.WellKnown.Homeserver.BaseURL) > 0 {
		var urlErr error
		cli.HomeserverURL, urlErr = url.Parse(resp.WellKnown.Homeserver.BaseURL)
		if urlErr != nil {
			cli.Log.Warn().
				Err(urlErr).
				Str("homeserver_url", resp.WellKnown.Homeserver.BaseURL).
				Msg("Failed to parse homeserver URL in login response")
		} else {
			cli.Log.Debug().
				Str("homeserver_url", cli.HomeserverURL.String()).
				Msg("Updated homeserver URL after login")
		}
	}
	return
}

// Create a device for an appservice user using MSC4190.
func (cli *Client) CreateDeviceMSC4190(ctx context.Context, deviceID id.DeviceID, initialDisplayName string) error {
	if len(deviceID) == 0 {
		deviceID = id.DeviceID(strings.ToUpper(random.String(10)))
	}
	_, err := cli.MakeRequest(ctx, http.MethodPut, cli.BuildClientURL("v3", "devices", deviceID), &ReqPutDevice{
		DisplayName: initialDisplayName,
	}, nil)
	if err != nil {
		return err
	}
	cli.DeviceID = deviceID
	cli.SetAppServiceDeviceID = true
	return nil
}

// Logout the current user. See https://spec.matrix.org/v1.2/client-server-api/#post_matrixclientv3logout
// This does not clear the credentials from the client instance. See ClearCredentials() instead.
func (cli *Client) Logout(ctx context.Context) (resp *RespLogout, err error) {
	urlPath := cli.BuildClientURL("v3", "logout")
	_, err = cli.MakeRequest(ctx, http.MethodPost, urlPath, nil, &resp)
	return
}

// LogoutAll logs out all the devices of the current user. See https://spec.matrix.org/v1.2/client-server-api/#post_matrixclientv3logoutall
// This does not clear the credentials from the client instance. See ClearCredentials() instead.
func (cli *Client) LogoutAll(ctx context.Context) (resp *RespLogout, err error) {
	urlPath := cli.BuildClientURL("v3", "logout", "all")
	_, err = cli.MakeRequest(ctx, http.MethodPost, urlPath, nil, &resp)
	return
}

// Versions returns the list of supported Matrix versions on this homeserver. See https://spec.matrix.org/v1.2/client-server-api/#get_matrixclientversions
func (cli *Client) Versions(ctx context.Context) (resp *RespVersions, err error) {
	urlPath := cli.BuildClientURL("versions")
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	if resp != nil {
		cli.SpecVersions = resp
	}
	return
}

// Capabilities returns capabilities on this homeserver. See https://spec.matrix.org/v1.3/client-server-api/#capabilities-negotiation
func (cli *Client) Capabilities(ctx context.Context) (resp *RespCapabilities, err error) {
	urlPath := cli.BuildClientURL("v3", "capabilities")
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	return
}

// JoinRoom joins the client to a room ID or alias. See https://spec.matrix.org/v1.13/client-server-api/#post_matrixclientv3joinroomidoralias
//
// The last parameter contains optional extra fields and can be left nil.
func (cli *Client) JoinRoom(ctx context.Context, roomIDorAlias string, req *ReqJoinRoom) (resp *RespJoinRoom, err error) {
	if req == nil {
		req = &ReqJoinRoom{}
	}
	urlPath := cli.BuildURLWithFullQuery(ClientURLPath{"v3", "join", roomIDorAlias}, func(q url.Values) {
		if len(req.Via) > 0 {
			q["via"] = req.Via
		}
	})
	_, err = cli.MakeRequest(ctx, http.MethodPost, urlPath, req, &resp)
	if err == nil && cli.StateStore != nil {
		err = cli.StateStore.SetMembership(ctx, resp.RoomID, cli.UserID, event.MembershipJoin)
		if err != nil {
			err = fmt.Errorf("failed to update state store: %w", err)
		}
	}
	return
}

// KnockRoom requests to join a room ID or alias. See https://spec.matrix.org/v1.13/client-server-api/#post_matrixclientv3knockroomidoralias
//
// The last parameter contains optional extra fields and can be left nil.
func (cli *Client) KnockRoom(ctx context.Context, roomIDorAlias string, req *ReqKnockRoom) (resp *RespKnockRoom, err error) {
	if req == nil {
		req = &ReqKnockRoom{}
	}
	urlPath := cli.BuildURLWithFullQuery(ClientURLPath{"v3", "knock", roomIDorAlias}, func(q url.Values) {
		if len(req.Via) > 0 {
			q["via"] = req.Via
		}
	})
	_, err = cli.MakeRequest(ctx, http.MethodPost, urlPath, req, &resp)
	if err == nil && cli.StateStore != nil {
		err = cli.StateStore.SetMembership(ctx, resp.RoomID, cli.UserID, event.MembershipKnock)
		if err != nil {
			err = fmt.Errorf("failed to update state store: %w", err)
		}
	}
	return
}

// JoinRoomByID joins the client to a room ID. See https://spec.matrix.org/v1.2/client-server-api/#post_matrixclientv3roomsroomidjoin
//
// Unlike JoinRoom, this method can only be used to join rooms that the server already knows about.
// It's mostly intended for bridges and other things where it's already certain that the server is in the room.
func (cli *Client) JoinRoomByID(ctx context.Context, roomID id.RoomID) (resp *RespJoinRoom, err error) {
	_, err = cli.MakeRequest(ctx, http.MethodPost, cli.BuildClientURL("v3", "rooms", roomID, "join"), nil, &resp)
	if err == nil && cli.StateStore != nil {
		err = cli.StateStore.SetMembership(ctx, resp.RoomID, cli.UserID, event.MembershipJoin)
		if err != nil {
			err = fmt.Errorf("failed to update state store: %w", err)
		}
	}
	return
}

func (cli *Client) GetProfile(ctx context.Context, mxid id.UserID) (resp *RespUserProfile, err error) {
	urlPath := cli.BuildClientURL("v3", "profile", mxid)
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	return
}

func (cli *Client) SearchUserDirectory(ctx context.Context, query string, limit int) (resp *RespSearchUserDirectory, err error) {
	urlPath := cli.BuildClientURL("v3", "user_directory", "search")
	_, err = cli.MakeRequest(ctx, http.MethodPost, urlPath, &ReqSearchUserDirectory{
		SearchTerm: query,
		Limit:      limit,
	}, &resp)
	return
}

func (cli *Client) GetMutualRooms(ctx context.Context, otherUserID id.UserID, extras ...ReqMutualRooms) (resp *RespMutualRooms, err error) {
	if cli.SpecVersions != nil && !cli.SpecVersions.Supports(FeatureMutualRooms) {
		err = fmt.Errorf("server does not support fetching mutual rooms")
		return
	}
	query := map[string]string{
		"user_id": otherUserID.String(),
	}
	if len(extras) > 0 {
		query["from"] = extras[0].From
	}
	urlPath := cli.BuildURLWithQuery(ClientURLPath{"unstable", "uk.half-shot.msc2666", "user", "mutual_rooms"}, query)
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	return
}

func (cli *Client) GetRoomSummary(ctx context.Context, roomIDOrAlias string, via ...string) (resp *RespRoomSummary, err error) {
	urlPath := ClientURLPath{"unstable", "im.nheko.summary", "summary", roomIDOrAlias}
	if cli.SpecVersions.ContainsGreaterOrEqual(SpecV115) {
		urlPath = ClientURLPath{"v1", "room_summary", roomIDOrAlias}
	}
	// TODO add version check after one is added to MSC3266
	fullURL := cli.BuildURLWithFullQuery(urlPath, func(q url.Values) {
		if len(via) > 0 {
			q["via"] = via
		}
	})
	_, err = cli.MakeRequest(ctx, http.MethodGet, fullURL, nil, &resp)
	return
}

// GetDisplayName returns the display name of the user with the specified MXID. See https://spec.matrix.org/v1.2/client-server-api/#get_matrixclientv3profileuseriddisplayname
func (cli *Client) GetDisplayName(ctx context.Context, mxid id.UserID) (resp *RespUserDisplayName, err error) {
	err = cli.GetProfileField(ctx, mxid, "displayname", &resp)
	return
}

// GetOwnDisplayName returns the user's display name. See https://spec.matrix.org/v1.2/client-server-api/#get_matrixclientv3profileuseriddisplayname
func (cli *Client) GetOwnDisplayName(ctx context.Context) (resp *RespUserDisplayName, err error) {
	return cli.GetDisplayName(ctx, cli.UserID)
}

// SetDisplayName sets the user's profile display name. See https://spec.matrix.org/v1.2/client-server-api/#put_matrixclientv3profileuseriddisplayname
func (cli *Client) SetDisplayName(ctx context.Context, displayName string) (err error) {
	return cli.SetProfileField(ctx, "displayname", displayName)
}

// SetProfileField sets an arbitrary profile field. See https://spec.matrix.org/v1.16/client-server-api/#put_matrixclientv3profileuseridkeyname
func (cli *Client) SetProfileField(ctx context.Context, key string, value any) (err error) {
	urlPath := cli.BuildClientURL("v3", "profile", cli.UserID, key)
	if key != "displayname" && key != "avatar_url" && !cli.SpecVersions.Supports(FeatureArbitraryProfileFields) && cli.SpecVersions.Supports(FeatureUnstableProfileFields) {
		urlPath = cli.BuildClientURL("unstable", "uk.tcpip.msc4133", "profile", cli.UserID, key)
	}
	_, err = cli.MakeRequest(ctx, http.MethodPut, urlPath, map[string]any{
		key: value,
	}, nil)
	return
}

// DeleteProfileField deletes an arbitrary profile field. See https://spec.matrix.org/v1.16/client-server-api/#put_matrixclientv3profileuseridkeyname
func (cli *Client) DeleteProfileField(ctx context.Context, key string) (err error) {
	urlPath := cli.BuildClientURL("v3", "profile", cli.UserID, key)
	if key != "displayname" && key != "avatar_url" && !cli.SpecVersions.Supports(FeatureArbitraryProfileFields) && cli.SpecVersions.Supports(FeatureUnstableProfileFields) {
		urlPath = cli.BuildClientURL("unstable", "uk.tcpip.msc4133", "profile", cli.UserID, key)
	}
	_, err = cli.MakeRequest(ctx, http.MethodDelete, urlPath, nil, nil)
	return
}

// GetProfileField gets an arbitrary profile field and parses the response into the given struct. See https://spec.matrix.org/unstable/client-server-api/#get_matrixclientv3profileuseridkeyname
func (cli *Client) GetProfileField(ctx context.Context, userID id.UserID, key string, into any) (err error) {
	urlPath := cli.BuildClientURL("v3", "profile", userID, key)
	if key != "displayname" && key != "avatar_url" && !cli.SpecVersions.Supports(FeatureArbitraryProfileFields) && cli.SpecVersions.Supports(FeatureUnstableProfileFields) {
		urlPath = cli.BuildClientURL("unstable", "uk.tcpip.msc4133", "profile", cli.UserID, key)
	}
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, into)
	return
}

// GetAvatarURL gets the avatar URL of the user with the specified MXID. See https://spec.matrix.org/v1.2/client-server-api/#get_matrixclientv3profileuseridavatar_url
func (cli *Client) GetAvatarURL(ctx context.Context, mxid id.UserID) (url id.ContentURI, err error) {
	s := struct {
		AvatarURL id.ContentURI `json:"avatar_url"`
	}{}
	err = cli.GetProfileField(ctx, mxid, "avatar_url", &s)
	url = s.AvatarURL
	return
}

// GetOwnAvatarURL gets the user's avatar URL. See https://spec.matrix.org/v1.2/client-server-api/#get_matrixclientv3profileuseridavatar_url
func (cli *Client) GetOwnAvatarURL(ctx context.Context) (url id.ContentURI, err error) {
	return cli.GetAvatarURL(ctx, cli.UserID)
}

// SetAvatarURL sets the user's avatar URL. See https://spec.matrix.org/v1.2/client-server-api/#put_matrixclientv3profileuseridavatar_url
func (cli *Client) SetAvatarURL(ctx context.Context, url id.ContentURI) (err error) {
	urlPath := cli.BuildClientURL("v3", "profile", cli.UserID, "avatar_url")
	s := struct {
		AvatarURL string `json:"avatar_url"`
	}{url.String()}
	_, err = cli.MakeRequest(ctx, http.MethodPut, urlPath, &s, nil)
	if err != nil {
		return err
	}

	return nil
}

// BeeperUpdateProfile sets custom fields in the user's profile.
func (cli *Client) BeeperUpdateProfile(ctx context.Context, data any) (err error) {
	urlPath := cli.BuildClientURL("v3", "profile", cli.UserID)
	_, err = cli.MakeRequest(ctx, http.MethodPatch, urlPath, data, nil)
	return
}

// GetAccountData gets the user's account data of this type. See https://spec.matrix.org/v1.2/client-server-api/#get_matrixclientv3useruseridaccount_datatype
func (cli *Client) GetAccountData(ctx context.Context, name string, output interface{}) (err error) {
	urlPath := cli.BuildClientURL("v3", "user", cli.UserID, "account_data", name)
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, output)
	return
}

// SetAccountData sets the user's account data of this type. See https://spec.matrix.org/v1.2/client-server-api/#put_matrixclientv3useruseridaccount_datatype
func (cli *Client) SetAccountData(ctx context.Context, name string, data interface{}) (err error) {
	urlPath := cli.BuildClientURL("v3", "user", cli.UserID, "account_data", name)
	_, err = cli.MakeRequest(ctx, http.MethodPut, urlPath, data, nil)
	if err != nil {
		return err
	}

	return nil
}

// GetRoomAccountData gets the user's account data of this type in a specific room. See https://spec.matrix.org/v1.2/client-server-api/#put_matrixclientv3useruseridaccount_datatype
func (cli *Client) GetRoomAccountData(ctx context.Context, roomID id.RoomID, name string, output interface{}) (err error) {
	urlPath := cli.BuildClientURL("v3", "user", cli.UserID, "rooms", roomID, "account_data", name)
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, output)
	return
}

// SetRoomAccountData sets the user's account data of this type in a specific room. See https://spec.matrix.org/v1.2/client-server-api/#put_matrixclientv3useruseridroomsroomidaccount_datatype
func (cli *Client) SetRoomAccountData(ctx context.Context, roomID id.RoomID, name string, data interface{}) (err error) {
	urlPath := cli.BuildClientURL("v3", "user", cli.UserID, "rooms", roomID, "account_data", name)
	_, err = cli.MakeRequest(ctx, http.MethodPut, urlPath, data, nil)
	if err != nil {
		return err
	}

	return nil
}

// SendMessageEvent sends a message event into a room. See https://spec.matrix.org/v1.2/client-server-api/#put_matrixclientv3roomsroomidsendeventtypetxnid
// contentJSON should be a pointer to something that can be encoded as JSON using json.Marshal.
func (cli *Client) SendMessageEvent(ctx context.Context, roomID id.RoomID, eventType event.Type, contentJSON interface{}, extra ...ReqSendEvent) (resp *RespSendEvent, err error) {
	var req ReqSendEvent
	if len(extra) > 0 {
		req = extra[0]
	}

	var txnID string
	if len(req.TransactionID) > 0 {
		txnID = req.TransactionID
	} else {
		txnID = cli.TxnID()
	}

	queryParams := map[string]string{}
	if req.Timestamp > 0 {
		queryParams["ts"] = strconv.FormatInt(req.Timestamp, 10)
	}
	if req.MeowEventID != "" {
		queryParams["fi.mau.event_id"] = req.MeowEventID.String()
	}
	if req.UnstableDelay > 0 {
		queryParams["org.matrix.msc4140.delay"] = strconv.FormatInt(req.UnstableDelay.Milliseconds(), 10)
	}

	if !req.DontEncrypt && cli != nil && cli.Crypto != nil && eventType != event.EventReaction && eventType != event.EventEncrypted {
		var isEncrypted bool
		isEncrypted, err = cli.StateStore.IsEncrypted(ctx, roomID)
		if err != nil {
			err = fmt.Errorf("failed to check if room is encrypted: %w", err)
			return
		}
		if isEncrypted {
			if contentJSON, err = cli.Crypto.Encrypt(ctx, roomID, eventType, contentJSON); err != nil {
				err = fmt.Errorf("failed to encrypt event: %w", err)
				return
			}
			eventType = event.EventEncrypted
		}
	}

	urlData := ClientURLPath{"v3", "rooms", roomID, "send", eventType.String(), txnID}
	urlPath := cli.BuildURLWithQuery(urlData, queryParams)
	_, err = cli.MakeRequest(ctx, http.MethodPut, urlPath, contentJSON, &resp)
	return
}

// SendStateEvent sends a state event into a room. See https://spec.matrix.org/v1.16/client-server-api/#put_matrixclientv3roomsroomidstateeventtypestatekey
// contentJSON should be a pointer to something that can be encoded as JSON using json.Marshal.
func (cli *Client) SendStateEvent(ctx context.Context, roomID id.RoomID, eventType event.Type, stateKey string, contentJSON any, extra ...ReqSendEvent) (resp *RespSendEvent, err error) {
	var req ReqSendEvent
	if len(extra) > 0 {
		req = extra[0]
	}

	queryParams := map[string]string{}
	if req.MeowEventID != "" {
		queryParams["fi.mau.event_id"] = req.MeowEventID.String()
	}
	if req.TransactionID != "" {
		queryParams["fi.mau.transaction_id"] = req.TransactionID
	}
	if req.UnstableDelay > 0 {
		queryParams["org.matrix.msc4140.delay"] = strconv.FormatInt(req.UnstableDelay.Milliseconds(), 10)
	}
	if req.Timestamp > 0 {
		queryParams["ts"] = strconv.FormatInt(req.Timestamp, 10)
	}

	urlData := ClientURLPath{"v3", "rooms", roomID, "state", eventType.String(), stateKey}
	urlPath := cli.BuildURLWithQuery(urlData, queryParams)
	_, err = cli.MakeRequest(ctx, http.MethodPut, urlPath, contentJSON, &resp)
	if err == nil && cli.StateStore != nil && req.UnstableDelay == 0 {
		cli.updateStoreWithOutgoingEvent(ctx, roomID, eventType, stateKey, contentJSON)
	}
	return
}

// SendMassagedStateEvent sends a state event into a room with a custom timestamp. See https://spec.matrix.org/v1.2/client-server-api/#put_matrixclientv3roomsroomidstateeventtypestatekey
// contentJSON should be a pointer to something that can be encoded as JSON using json.Marshal.
//
// Deprecated: SendStateEvent accepts a timestamp via ReqSendEvent and should be used instead.
func (cli *Client) SendMassagedStateEvent(ctx context.Context, roomID id.RoomID, eventType event.Type, stateKey string, contentJSON interface{}, ts int64) (resp *RespSendEvent, err error) {
	resp, err = cli.SendStateEvent(ctx, roomID, eventType, stateKey, contentJSON, ReqSendEvent{
		Timestamp: ts,
	})
	return
}

func (cli *Client) DelayedEvents(ctx context.Context, req *ReqDelayedEvents) (resp *RespDelayedEvents, err error) {
	query := map[string]string{}
	if req.DelayID != "" {
		query["delay_id"] = string(req.DelayID)
	}
	if req.Status != "" {
		query["status"] = string(req.Status)
	}
	if req.NextBatch != "" {
		query["next_batch"] = req.NextBatch
	}

	urlPath := cli.BuildURLWithQuery(ClientURLPath{"unstable", "org.matrix.msc4140", "delayed_events"}, query)
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, req, &resp)

	// Migration: merge old keys with new ones
	if resp != nil {
		resp.Scheduled = append(resp.Scheduled, resp.DelayedEvents...)
		resp.DelayedEvents = nil
		resp.Finalised = append(resp.Finalised, resp.FinalisedEvents...)
		resp.FinalisedEvents = nil
	}

	return
}

func (cli *Client) UpdateDelayedEvent(ctx context.Context, req *ReqUpdateDelayedEvent) (resp *RespUpdateDelayedEvent, err error) {
	urlPath := cli.BuildClientURL("unstable", "org.matrix.msc4140", "delayed_events", req.DelayID)
	_, err = cli.MakeRequest(ctx, http.MethodPost, urlPath, req, &resp)
	return
}

// SendText sends an m.room.message event into the given room with a msgtype of m.text
// See https://spec.matrix.org/v1.2/client-server-api/#mtext
func (cli *Client) SendText(ctx context.Context, roomID id.RoomID, text string) (*RespSendEvent, error) {
	return cli.SendMessageEvent(ctx, roomID, event.EventMessage, &event.MessageEventContent{
		MsgType: event.MsgText,
		Body:    text,
	})
}

// SendNotice sends an m.room.message event into the given room with a msgtype of m.notice
// See https://spec.matrix.org/v1.2/client-server-api/#mnotice
func (cli *Client) SendNotice(ctx context.Context, roomID id.RoomID, text string) (*RespSendEvent, error) {
	return cli.SendMessageEvent(ctx, roomID, event.EventMessage, &event.MessageEventContent{
		MsgType: event.MsgNotice,
		Body:    text,
	})
}

func (cli *Client) SendReaction(ctx context.Context, roomID id.RoomID, eventID id.EventID, reaction string) (*RespSendEvent, error) {
	return cli.SendMessageEvent(ctx, roomID, event.EventReaction, &event.ReactionEventContent{
		RelatesTo: event.RelatesTo{
			EventID: eventID,
			Type:    event.RelAnnotation,
			Key:     reaction,
		},
	})
}

// RedactEvent redacts the given event. See https://spec.matrix.org/v1.2/client-server-api/#put_matrixclientv3roomsroomidredacteventidtxnid
func (cli *Client) RedactEvent(ctx context.Context, roomID id.RoomID, eventID id.EventID, extra ...ReqRedact) (resp *RespSendEvent, err error) {
	req := ReqRedact{}
	if len(extra) > 0 {
		req = extra[0]
	}
	if req.Extra == nil {
		req.Extra = make(map[string]interface{})
	}
	if len(req.Reason) > 0 {
		req.Extra["reason"] = req.Reason
	}
	var txnID string
	if len(req.TxnID) > 0 {
		txnID = req.TxnID
	} else {
		txnID = cli.TxnID()
	}
	urlPath := cli.BuildClientURL("v3", "rooms", roomID, "redact", eventID, txnID)
	_, err = cli.MakeRequest(ctx, http.MethodPut, urlPath, req.Extra, &resp)
	return
}

func (cli *Client) UnstableRedactUserEvents(ctx context.Context, roomID id.RoomID, userID id.UserID, req *ReqRedactUser) (resp *RespRedactUserEvents, err error) {
	if req == nil {
		req = &ReqRedactUser{}
	}
	query := map[string]string{}
	if req.Limit > 0 {
		query["limit"] = strconv.Itoa(req.Limit)
	}
	urlPath := cli.BuildURLWithQuery(ClientURLPath{"unstable", "org.matrix.msc4194", "rooms", roomID, "redact", "user", userID}, query)
	_, err = cli.MakeRequest(ctx, http.MethodPost, urlPath, req, &resp)
	return
}

// CreateRoom creates a new Matrix room. See https://spec.matrix.org/v1.2/client-server-api/#post_matrixclientv3createroom
//
//	resp, err := cli.CreateRoom(&mautrix.ReqCreateRoom{
//		Preset: "public_chat",
//	})
//	fmt.Println("Room:", resp.RoomID)
func (cli *Client) CreateRoom(ctx context.Context, req *ReqCreateRoom) (resp *RespCreateRoom, err error) {
	urlPath := cli.BuildClientURL("v3", "createRoom")
	_, err = cli.MakeRequest(ctx, http.MethodPost, urlPath, req, &resp)
	if err == nil && cli.StateStore != nil {
		storeErr := cli.StateStore.SetMembership(ctx, resp.RoomID, cli.UserID, event.MembershipJoin)
		if storeErr != nil {
			cli.cliOrContextLog(ctx).Warn().Err(storeErr).
				Stringer("creator_user_id", cli.UserID).
				Msg("Failed to update creator membership in state store after creating room")
		}
		for _, evt := range req.InitialState {
			evt.RoomID = resp.RoomID
			if evt.StateKey == nil {
				evt.StateKey = ptr.Ptr("")
			}
			UpdateStateStore(ctx, cli.StateStore, evt)
		}
		inviteMembership := event.MembershipInvite
		if req.BeeperAutoJoinInvites {
			inviteMembership = event.MembershipJoin
		}
		for _, invitee := range req.Invite {
			storeErr = cli.StateStore.SetMembership(ctx, resp.RoomID, invitee, inviteMembership)
			if storeErr != nil {
				cli.cliOrContextLog(ctx).Warn().Err(storeErr).
					Stringer("invitee_user_id", invitee).
					Msg("Failed to update membership in state store after creating room")
			}
		}
	}
	return
}

// LeaveRoom leaves the given room. See https://spec.matrix.org/v1.2/client-server-api/#post_matrixclientv3roomsroomidleave
func (cli *Client) LeaveRoom(ctx context.Context, roomID id.RoomID, optionalReq ...*ReqLeave) (resp *RespLeaveRoom, err error) {
	req := &ReqLeave{}
	if len(optionalReq) == 1 {
		req = optionalReq[0]
	} else if len(optionalReq) > 1 {
		panic("invalid number of arguments to LeaveRoom")
	}
	u := cli.BuildClientURL("v3", "rooms", roomID, "leave")
	_, err = cli.MakeRequest(ctx, http.MethodPost, u, req, &resp)
	if err == nil && cli.StateStore != nil {
		err = cli.StateStore.SetMembership(ctx, roomID, cli.UserID, event.MembershipLeave)
		if err != nil {
			err = fmt.Errorf("failed to update membership in state store: %w", err)
		}
	}
	return
}

// ForgetRoom forgets a room entirely. See https://spec.matrix.org/v1.2/client-server-api/#post_matrixclientv3roomsroomidforget
func (cli *Client) ForgetRoom(ctx context.Context, roomID id.RoomID) (resp *RespForgetRoom, err error) {
	u := cli.BuildClientURL("v3", "rooms", roomID, "forget")
	_, err = cli.MakeRequest(ctx, http.MethodPost, u, struct{}{}, &resp)
	return
}

// InviteUser invites a user to a room. See https://spec.matrix.org/v1.2/client-server-api/#post_matrixclientv3roomsroomidinvite
func (cli *Client) InviteUser(ctx context.Context, roomID id.RoomID, req *ReqInviteUser) (resp *RespInviteUser, err error) {
	u := cli.BuildClientURL("v3", "rooms", roomID, "invite")
	_, err = cli.MakeRequest(ctx, http.MethodPost, u, req, &resp)
	if err == nil && cli.StateStore != nil {
		err = cli.StateStore.SetMembership(ctx, roomID, req.UserID, event.MembershipInvite)
		if err != nil {
			err = fmt.Errorf("failed to update membership in state store: %w", err)
		}
	}
	return
}

// InviteUserByThirdParty invites a third-party identifier to a room. See https://spec.matrix.org/v1.2/client-server-api/#post_matrixclientv3roomsroomidinvite-1
func (cli *Client) InviteUserByThirdParty(ctx context.Context, roomID id.RoomID, req *ReqInvite3PID) (resp *RespInviteUser, err error) {
	u := cli.BuildClientURL("v3", "rooms", roomID, "invite")
	_, err = cli.MakeRequest(ctx, http.MethodPost, u, req, &resp)
	return
}

// KickUser kicks a user from a room. See https://spec.matrix.org/v1.2/client-server-api/#post_matrixclientv3roomsroomidkick
func (cli *Client) KickUser(ctx context.Context, roomID id.RoomID, req *ReqKickUser) (resp *RespKickUser, err error) {
	u := cli.BuildClientURL("v3", "rooms", roomID, "kick")
	_, err = cli.MakeRequest(ctx, http.MethodPost, u, req, &resp)
	if err == nil && cli.StateStore != nil {
		err = cli.StateStore.SetMembership(ctx, roomID, req.UserID, event.MembershipLeave)
		if err != nil {
			err = fmt.Errorf("failed to update membership in state store: %w", err)
		}
	}
	return
}

// BanUser bans a user from a room. See https://spec.matrix.org/v1.2/client-server-api/#post_matrixclientv3roomsroomidban
func (cli *Client) BanUser(ctx context.Context, roomID id.RoomID, req *ReqBanUser) (resp *RespBanUser, err error) {
	u := cli.BuildClientURL("v3", "rooms", roomID, "ban")
	_, err = cli.MakeRequest(ctx, http.MethodPost, u, req, &resp)
	if err == nil && cli.StateStore != nil {
		err = cli.StateStore.SetMembership(ctx, roomID, req.UserID, event.MembershipBan)
		if err != nil {
			err = fmt.Errorf("failed to update membership in state store: %w", err)
		}
	}
	return
}

// UnbanUser unbans a user from a room. See https://spec.matrix.org/v1.2/client-server-api/#post_matrixclientv3roomsroomidunban
func (cli *Client) UnbanUser(ctx context.Context, roomID id.RoomID, req *ReqUnbanUser) (resp *RespUnbanUser, err error) {
	u := cli.BuildClientURL("v3", "rooms", roomID, "unban")
	_, err = cli.MakeRequest(ctx, http.MethodPost, u, req, &resp)
	if err == nil && cli.StateStore != nil {
		err = cli.StateStore.SetMembership(ctx, roomID, req.UserID, event.MembershipLeave)
		if err != nil {
			err = fmt.Errorf("failed to update membership in state store: %w", err)
		}
	}
	return
}

// UserTyping sets the typing status of the user. See https://spec.matrix.org/v1.2/client-server-api/#put_matrixclientv3roomsroomidtypinguserid
func (cli *Client) UserTyping(ctx context.Context, roomID id.RoomID, typing bool, timeout time.Duration) (resp *RespTyping, err error) {
	req := ReqTyping{Typing: typing, Timeout: timeout.Milliseconds()}
	u := cli.BuildClientURL("v3", "rooms", roomID, "typing", cli.UserID)
	_, err = cli.MakeRequest(ctx, http.MethodPut, u, req, &resp)
	return
}

// GetPresence gets the presence of the user with the specified MXID. See https://spec.matrix.org/v1.2/client-server-api/#get_matrixclientv3presenceuseridstatus
func (cli *Client) GetPresence(ctx context.Context, userID id.UserID) (resp *RespPresence, err error) {
	resp = new(RespPresence)
	u := cli.BuildClientURL("v3", "presence", userID, "status")
	_, err = cli.MakeRequest(ctx, http.MethodGet, u, nil, resp)
	return
}

// GetOwnPresence gets the user's presence. See https://spec.matrix.org/v1.2/client-server-api/#get_matrixclientv3presenceuseridstatus
func (cli *Client) GetOwnPresence(ctx context.Context) (resp *RespPresence, err error) {
	return cli.GetPresence(ctx, cli.UserID)
}

func (cli *Client) SetPresence(ctx context.Context, presence ReqPresence) (err error) {
	u := cli.BuildClientURL("v3", "presence", cli.UserID, "status")
	_, err = cli.MakeRequest(ctx, http.MethodPut, u, presence, nil)
	return
}

func (cli *Client) updateStoreWithOutgoingEvent(ctx context.Context, roomID id.RoomID, eventType event.Type, stateKey string, contentJSON interface{}) {
	if cli == nil || cli.StateStore == nil {
		return
	}
	fakeEvt := &event.Event{
		StateKey: &stateKey,
		Type:     eventType,
		RoomID:   roomID,
	}
	var err error
	fakeEvt.Content.VeryRaw, err = json.Marshal(contentJSON)
	if err != nil {
		cli.Log.Warn().Err(err).Msg("Failed to marshal state event content to update state store")
		return
	}
	err = json.Unmarshal(fakeEvt.Content.VeryRaw, &fakeEvt.Content.Raw)
	if err != nil {
		cli.Log.Warn().Err(err).Msg("Failed to unmarshal state event content to update state store")
		return
	}
	err = fakeEvt.Content.ParseRaw(fakeEvt.Type)
	if err != nil {
		switch fakeEvt.Type {
		case event.StateMember, event.StatePowerLevels, event.StateEncryption:
			cli.Log.Warn().Err(err).Msg("Failed to parse state event content to update state store")
		default:
			cli.Log.Debug().Err(err).Msg("Failed to parse state event content to update state store")
		}
		return
	}
	UpdateStateStore(ctx, cli.StateStore, fakeEvt)
}

// StateEvent gets the content of a single state event in a room.
// It will attempt to JSON unmarshal into the given "outContent" struct with the HTTP response body, or return an error.
// See https://spec.matrix.org/v1.2/client-server-api/#get_matrixclientv3roomsroomidstateeventtypestatekey
func (cli *Client) StateEvent(ctx context.Context, roomID id.RoomID, eventType event.Type, stateKey string, outContent interface{}) (err error) {
	u := cli.BuildClientURL("v3", "rooms", roomID, "state", eventType.String(), stateKey)
	_, err = cli.MakeRequest(ctx, http.MethodGet, u, nil, outContent)
	if err == nil && cli.StateStore != nil {
		cli.updateStoreWithOutgoingEvent(ctx, roomID, eventType, stateKey, outContent)
	}
	return
}

// FullStateEvent gets a single state event in a room. Unlike [StateEvent], this gets the entire event
// (including details like the sender and timestamp).
// This requires the server to support the ?format=event query parameter, which is currently missing from the spec.
// See https://github.com/matrix-org/matrix-spec/issues/1047 for more info
func (cli *Client) FullStateEvent(ctx context.Context, roomID id.RoomID, eventType event.Type, stateKey string) (evt *event.Event, err error) {
	u := cli.BuildURLWithQuery(ClientURLPath{"v3", "rooms", roomID, "state", eventType.String(), stateKey}, map[string]string{
		"format": "event",
	})
	_, err = cli.MakeRequest(ctx, http.MethodGet, u, nil, &evt)
	if evt != nil {
		evt.Type.Class = event.StateEventType
		_ = evt.Content.ParseRaw(evt.Type)
		if evt.RoomID == "" {
			evt.RoomID = roomID
		}
	}
	if err == nil && cli.StateStore != nil {
		UpdateStateStore(ctx, cli.StateStore, evt)
	}
	return
}

// parseRoomStateArray parses a JSON array as a stream and stores the events inside it in a room state map.
func parseRoomStateArray(req *http.Request, res *http.Response, responseJSON any, limit int64) ([]byte, error) {
	if res.ContentLength > limit {
		return nil, HTTPError{
			Request:  req,
			Response: res,

			Message:      "not reading response",
			WrappedError: fmt.Errorf("%w (%.2f MiB)", ErrResponseTooLong, float64(res.ContentLength)/1024/1024),
		}
	}
	response := make(RoomStateMap)
	responsePtr := responseJSON.(*map[event.Type]map[string]*event.Event)
	*responsePtr = response
	dec := json.NewDecoder(io.LimitReader(res.Body, limit))

	arrayStart, err := dec.Token()
	if err != nil {
		return nil, err
	} else if arrayStart != json.Delim('[') {
		return nil, fmt.Errorf("expected array start, got %+v", arrayStart)
	}

	for i := 1; dec.More(); i++ {
		var evt *event.Event
		err = dec.Decode(&evt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse state array item #%d: %v", i, err)
		}
		evt.Type.Class = event.StateEventType
		_ = evt.Content.ParseRaw(evt.Type)
		subMap, ok := response[evt.Type]
		if !ok {
			subMap = make(map[string]*event.Event)
			response[evt.Type] = subMap
		}
		subMap[*evt.StateKey] = evt
	}

	arrayEnd, err := dec.Token()
	if err != nil {
		return nil, err
	} else if arrayEnd != json.Delim(']') {
		return nil, fmt.Errorf("expected array end, got %+v", arrayStart)
	}
	return nil, nil
}

// State gets all state in a room.
// See https://spec.matrix.org/v1.2/client-server-api/#get_matrixclientv3roomsroomidstate
func (cli *Client) State(ctx context.Context, roomID id.RoomID) (stateMap RoomStateMap, err error) {
	_, err = cli.MakeFullRequest(ctx, FullRequest{
		Method:       http.MethodGet,
		URL:          cli.BuildClientURL("v3", "rooms", roomID, "state"),
		ResponseJSON: &stateMap,
		Handler:      parseRoomStateArray,
	})
	if stateMap != nil {
		pls, ok := stateMap[event.StatePowerLevels][""]
		if ok {
			pls.Content.AsPowerLevels().CreateEvent = stateMap[event.StateCreate][""]
		}
	}
	if err == nil && cli.StateStore != nil {
		for evtType, evts := range stateMap {
			if evtType == event.StateMember {
				continue
			}
			for _, evt := range evts {
				if evt.RoomID == "" {
					evt.RoomID = roomID
				}
				UpdateStateStore(ctx, cli.StateStore, evt)
			}
		}
		updateErr := cli.StateStore.ReplaceCachedMembers(ctx, roomID, maps.Values(stateMap[event.StateMember]))
		if updateErr != nil {
			cli.cliOrContextLog(ctx).Warn().Err(updateErr).
				Stringer("room_id", roomID).
				Msg("Failed to update members in state store after fetching members")
		}
	}
	return
}

// StateAsArray gets all the state in a room as an array. It does not update the state store.
// Use State to get the events as a map and also update the state store.
func (cli *Client) StateAsArray(ctx context.Context, roomID id.RoomID) (state []*event.Event, err error) {
	_, err = cli.MakeRequest(ctx, http.MethodGet, cli.BuildClientURL("v3", "rooms", roomID, "state"), nil, &state)
	if err == nil {
		for _, evt := range state {
			evt.Type.Class = event.StateEventType
		}
	}
	return
}

// GetMediaConfig fetches the configuration of the content repository, such as upload limitations.
func (cli *Client) GetMediaConfig(ctx context.Context) (resp *RespMediaConfig, err error) {
	_, err = cli.MakeRequest(ctx, http.MethodGet, cli.BuildClientURL("v1", "media", "config"), nil, &resp)
	return
}

func (cli *Client) RequestOpenIDToken(ctx context.Context) (resp *RespOpenIDToken, err error) {
	_, err = cli.MakeRequest(ctx, http.MethodPost, cli.BuildClientURL("v3", "user", cli.UserID, "openid", "request_token"), nil, &resp)
	return
}

// UploadLink uploads an HTTP URL and then returns an MXC URI.
func (cli *Client) UploadLink(ctx context.Context, link string) (*RespMediaUpload, error) {
	if cli == nil {
		return nil, ErrClientIsNil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return nil, err
	}

	res, err := cli.Client.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	return cli.Upload(ctx, res.Body, res.Header.Get("Content-Type"), res.ContentLength)
}

func (cli *Client) Download(ctx context.Context, mxcURL id.ContentURI) (*http.Response, error) {
	_, resp, err := cli.MakeFullRequestWithResp(ctx, FullRequest{
		Method:           http.MethodGet,
		URL:              cli.BuildClientURL("v1", "media", "download", mxcURL.Homeserver, mxcURL.FileID),
		DontReadResponse: true,
	})
	return resp, err
}

type DownloadThumbnailExtra struct {
	Method   string
	Animated bool
}

func (cli *Client) DownloadThumbnail(ctx context.Context, mxcURL id.ContentURI, height, width int, extras ...DownloadThumbnailExtra) (*http.Response, error) {
	if len(extras) > 1 {
		panic(fmt.Errorf("invalid number of arguments to DownloadThumbnail: %d", len(extras)))
	}
	var extra DownloadThumbnailExtra
	if len(extras) == 1 {
		extra = extras[0]
	}
	path := ClientURLPath{"v1", "media", "thumbnail", mxcURL.Homeserver, mxcURL.FileID}
	query := map[string]string{
		"height": strconv.Itoa(height),
		"width":  strconv.Itoa(width),
	}
	if extra.Method != "" {
		query["method"] = extra.Method
	}
	if extra.Animated {
		query["animated"] = "true"
	}
	_, resp, err := cli.MakeFullRequestWithResp(ctx, FullRequest{
		Method:           http.MethodGet,
		URL:              cli.BuildURLWithQuery(path, query),
		DontReadResponse: true,
	})
	return resp, err
}

func (cli *Client) DownloadBytes(ctx context.Context, mxcURL id.ContentURI) ([]byte, error) {
	resp, err := cli.Download(ctx, mxcURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

type ReqCreateMXC struct {
	BeeperUniqueID string
	BeeperRoomID   id.RoomID
}

// CreateMXC creates a blank Matrix content URI to allow uploading the content asynchronously later.
//
// See https://spec.matrix.org/v1.7/client-server-api/#post_matrixmediav1create
func (cli *Client) CreateMXC(ctx context.Context, extra ...ReqCreateMXC) (*RespCreateMXC, error) {
	var m RespCreateMXC
	query := map[string]string{}
	if len(extra) > 0 {
		if extra[0].BeeperUniqueID != "" {
			query["com.beeper.unique_id"] = extra[0].BeeperUniqueID
		}
		if extra[0].BeeperRoomID != "" {
			query["com.beeper.room_id"] = string(extra[0].BeeperRoomID)
		}
	}
	createURL := cli.BuildURLWithQuery(MediaURLPath{"v1", "create"}, query)
	_, err := cli.MakeRequest(ctx, http.MethodPost, createURL, nil, &m)
	return &m, err
}

// UploadAsync creates a blank content URI with CreateMXC, starts uploading the data in the background
// and returns the created MXC immediately.
//
// See https://spec.matrix.org/v1.7/client-server-api/#post_matrixmediav1create
// and https://spec.matrix.org/v1.7/client-server-api/#put_matrixmediav3uploadservernamemediaid
func (cli *Client) UploadAsync(ctx context.Context, req ReqUploadMedia) (*RespCreateMXC, error) {
	resp, err := cli.CreateMXC(ctx)
	if err != nil {
		req.DoneCallback()
		return nil, err
	}
	req.MXC = resp.ContentURI
	req.UnstableUploadURL = resp.UnstableUploadURL
	go func() {
		_, err = cli.UploadMedia(ctx, req)
		if err != nil {
			cli.Log.Error().Stringer("mxc", req.MXC).Err(err).Msg("Async upload of media failed")
		}
	}()
	return resp, nil
}

func (cli *Client) UploadBytes(ctx context.Context, data []byte, contentType string) (*RespMediaUpload, error) {
	return cli.UploadBytesWithName(ctx, data, contentType, "")
}

func (cli *Client) UploadBytesWithName(ctx context.Context, data []byte, contentType, fileName string) (*RespMediaUpload, error) {
	return cli.UploadMedia(ctx, ReqUploadMedia{
		ContentBytes: data,
		ContentType:  contentType,
		FileName:     fileName,
	})
}

// Upload uploads the given data to the content repository and returns an MXC URI.
//
// Deprecated: UploadMedia should be used instead.
func (cli *Client) Upload(ctx context.Context, content io.Reader, contentType string, contentLength int64) (*RespMediaUpload, error) {
	return cli.UploadMedia(ctx, ReqUploadMedia{
		Content:       content,
		ContentLength: contentLength,
		ContentType:   contentType,
	})
}

type ReqUploadMedia struct {
	ContentBytes  []byte
	Content       io.Reader
	ContentLength int64
	ContentType   string
	FileName      string

	DoneCallback func()

	// MXC specifies an existing MXC URI which doesn't have content yet to upload into.
	// See https://spec.matrix.org/unstable/client-server-api/#put_matrixmediav3uploadservernamemediaid
	MXC id.ContentURI

	// UnstableUploadURL specifies the URL to upload the content to. MXC must also be set.
	// see https://github.com/matrix-org/matrix-spec-proposals/pull/3870 for more info
	UnstableUploadURL string
}

func (cli *Client) tryUploadMediaToURL(ctx context.Context, url, contentType string, content io.Reader, contentLength int64) (*http.Response, error) {
	cli.Log.Debug().Str("url", url).Msg("Uploading media to external URL")
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, content)
	if err != nil {
		return nil, err
	}
	req.ContentLength = contentLength
	req.Header.Set("Content-Type", contentType)
	if cli.UserAgent != "" {
		req.Header.Set("User-Agent", cli.UserAgent+" (external media uploader)")
	}

	if cli.ExternalClient != nil {
		return cli.ExternalClient.Do(req)
	} else {
		return http.DefaultClient.Do(req)
	}
}

func (cli *Client) uploadMediaToURL(ctx context.Context, data ReqUploadMedia) (*RespMediaUpload, error) {
	retries := cli.DefaultHTTPRetries
	reader := data.Content
	if data.ContentBytes != nil {
		data.ContentLength = int64(len(data.ContentBytes))
		reader = bytes.NewReader(data.ContentBytes)
	} else if rsc, ok := reader.(io.ReadSeekCloser); ok {
		// Prevent HTTP from closing the request body, it might be needed for retries
		reader = nopCloseSeeker{rsc}
	}
	readerSeeker, canSeek := reader.(io.ReadSeeker)
	if !canSeek {
		retries = 0
	}
	for {
		resp, err := cli.tryUploadMediaToURL(ctx, data.UnstableUploadURL, data.ContentType, reader, data.ContentLength)
		if err == nil {
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				// Everything is fine
				break
			}
			err = fmt.Errorf("HTTP %d", resp.StatusCode)
		} else if errors.Is(err, context.Canceled) {
			cli.Log.Warn().Str("url", data.UnstableUploadURL).Msg("External media upload canceled")
			return nil, err
		}
		if retries <= 0 {
			cli.Log.Warn().Str("url", data.UnstableUploadURL).Err(err).
				Msg("Error uploading media to external URL, not retrying")
			return nil, err
		}
		cli.Log.Warn().Str("url", data.UnstableUploadURL).Err(err).
			Msg("Error uploading media to external URL, retrying")
		retries--
		_, err = readerSeeker.Seek(0, io.SeekStart)
		if err != nil {
			return nil, fmt.Errorf("failed to seek back to start of reader: %w", err)
		}
	}

	query := map[string]string{}
	if len(data.FileName) > 0 {
		query["filename"] = data.FileName
	}

	notifyURL := cli.BuildURLWithQuery(MediaURLPath{"unstable", "com.beeper.msc3870", "upload", data.MXC.Homeserver, data.MXC.FileID, "complete"}, query)

	var m *RespMediaUpload
	_, err := cli.MakeRequest(ctx, http.MethodPost, notifyURL, nil, &m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

type nopCloseSeeker struct {
	io.ReadSeeker
}

func (nopCloseSeeker) Close() error {
	return nil
}

// UploadMedia uploads the given data to the content repository and returns an MXC URI.
// See https://spec.matrix.org/v1.7/client-server-api/#post_matrixmediav3upload
func (cli *Client) UploadMedia(ctx context.Context, data ReqUploadMedia) (*RespMediaUpload, error) {
	if data.DoneCallback != nil {
		defer data.DoneCallback()
	}
	if cli == nil {
		return nil, ErrClientIsNil
	}
	if data.UnstableUploadURL != "" {
		if data.MXC.IsEmpty() {
			return nil, errors.New("MXC must also be set when uploading to external URL")
		}
		return cli.uploadMediaToURL(ctx, data)
	}
	u, _ := url.Parse(cli.BuildURL(MediaURLPath{"v3", "upload"}))
	method := http.MethodPost
	if !data.MXC.IsEmpty() {
		u, _ = url.Parse(cli.BuildURL(MediaURLPath{"v3", "upload", data.MXC.Homeserver, data.MXC.FileID}))
		method = http.MethodPut
	}
	if len(data.FileName) > 0 {
		q := u.Query()
		q.Set("filename", data.FileName)
		u.RawQuery = q.Encode()
	}

	var headers http.Header
	if len(data.ContentType) > 0 {
		headers = http.Header{"Content-Type": []string{data.ContentType}}
	}

	var m RespMediaUpload
	_, err := cli.MakeFullRequest(ctx, FullRequest{
		Method:        method,
		URL:           u.String(),
		Headers:       headers,
		RequestBytes:  data.ContentBytes,
		RequestBody:   data.Content,
		RequestLength: data.ContentLength,
		ResponseJSON:  &m,
	})
	return &m, err
}

// GetURLPreview asks the homeserver to fetch a preview for a given URL.
//
// See https://spec.matrix.org/v1.2/client-server-api/#get_matrixmediav3preview_url
func (cli *Client) GetURLPreview(ctx context.Context, url string) (*RespPreviewURL, error) {
	reqURL := cli.BuildURLWithQuery(ClientURLPath{"v1", "media", "preview_url"}, map[string]string{
		"url": url,
	})
	var output RespPreviewURL
	_, err := cli.MakeRequest(ctx, http.MethodGet, reqURL, nil, &output)
	return &output, err
}

// JoinedMembers returns a map of joined room members. See https://spec.matrix.org/v1.2/client-server-api/#get_matrixclientv3roomsroomidjoined_members
//
// In general, usage of this API is discouraged in favour of /sync, as calling this API can race with incoming membership changes.
// This API is primarily designed for application services which may want to efficiently look up joined members in a room.
func (cli *Client) JoinedMembers(ctx context.Context, roomID id.RoomID) (resp *RespJoinedMembers, err error) {
	u := cli.BuildClientURL("v3", "rooms", roomID, "joined_members")
	_, err = cli.MakeRequest(ctx, http.MethodGet, u, nil, &resp)
	if err == nil && cli.StateStore != nil {
		fakeEvents := make([]*event.Event, len(resp.Joined))
		i := 0
		for userID, member := range resp.Joined {
			fakeEvents[i] = &event.Event{
				StateKey: ptr.Ptr(userID.String()),
				Type:     event.StateMember,
				RoomID:   roomID,
				Content: event.Content{Parsed: &event.MemberEventContent{
					Membership:  event.MembershipJoin,
					AvatarURL:   id.ContentURIString(member.AvatarURL),
					Displayname: member.DisplayName,
				}},
			}
			i++
		}
		updateErr := cli.StateStore.ReplaceCachedMembers(ctx, roomID, fakeEvents, event.MembershipJoin)
		if updateErr != nil {
			cli.cliOrContextLog(ctx).Warn().Err(updateErr).
				Stringer("room_id", roomID).
				Msg("Failed to update members in state store after fetching joined members")
		}
	}
	return
}

func (cli *Client) Members(ctx context.Context, roomID id.RoomID, req ...ReqMembers) (resp *RespMembers, err error) {
	var extra ReqMembers
	if len(req) > 0 {
		extra = req[0]
	}
	query := map[string]string{}
	if len(extra.At) > 0 {
		query["at"] = extra.At
	}
	if len(extra.Membership) > 0 {
		query["membership"] = string(extra.Membership)
	}
	if len(extra.NotMembership) > 0 {
		query["not_membership"] = string(extra.NotMembership)
	}
	u := cli.BuildURLWithQuery(ClientURLPath{"v3", "rooms", roomID, "members"}, query)
	_, err = cli.MakeRequest(ctx, http.MethodGet, u, nil, &resp)
	if err == nil {
		for _, evt := range resp.Chunk {
			_ = evt.Content.ParseRaw(evt.Type)
		}
	}
	if err == nil && cli.StateStore != nil {
		var onlyMemberships []event.Membership
		if extra.Membership != "" {
			onlyMemberships = []event.Membership{extra.Membership}
		} else if extra.NotMembership != "" {
			onlyMemberships = []event.Membership{event.MembershipJoin, event.MembershipLeave, event.MembershipInvite, event.MembershipBan, event.MembershipKnock}
			onlyMemberships = slices.DeleteFunc(onlyMemberships, func(m event.Membership) bool {
				return m == extra.NotMembership
			})
		}
		updateErr := cli.StateStore.ReplaceCachedMembers(ctx, roomID, resp.Chunk, onlyMemberships...)
		if updateErr != nil {
			cli.cliOrContextLog(ctx).Warn().Err(updateErr).
				Stringer("room_id", roomID).
				Msg("Failed to update members in state store after fetching members")
		}
	}
	return
}

// JoinedRooms returns a list of rooms which the client is joined to. See https://spec.matrix.org/v1.2/client-server-api/#get_matrixclientv3joined_rooms
//
// In general, usage of this API is discouraged in favour of /sync, as calling this API can race with incoming membership changes.
// This API is primarily designed for application services which may want to efficiently look up joined rooms.
func (cli *Client) JoinedRooms(ctx context.Context) (resp *RespJoinedRooms, err error) {
	u := cli.BuildClientURL("v3", "joined_rooms")
	_, err = cli.MakeRequest(ctx, http.MethodGet, u, nil, &resp)
	return
}

func (cli *Client) PublicRooms(ctx context.Context, req *ReqPublicRooms) (resp *RespPublicRooms, err error) {
	urlPath := cli.BuildURLWithQuery(ClientURLPath{"v3", "publicRooms"}, req.Query())
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	return
}

// Hierarchy returns a list of rooms that are in the room's hierarchy. See https://spec.matrix.org/v1.4/client-server-api/#get_matrixclientv1roomsroomidhierarchy
//
// The hierarchy API is provided to walk the space tree and discover the rooms with their aesthetic details. works in a depth-first manner:
// when it encounters another space as a child it recurses into that space before returning non-space children.
//
// The second function parameter specifies query parameters to limit the response. No query parameters will be added if it's nil.
func (cli *Client) Hierarchy(ctx context.Context, roomID id.RoomID, req *ReqHierarchy) (resp *RespHierarchy, err error) {
	urlPath := cli.BuildURLWithQuery(ClientURLPath{"v1", "rooms", roomID, "hierarchy"}, req.Query())
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	return
}

// Messages returns a list of message and state events for a room. It uses
// pagination query parameters to paginate history in the room.
// See https://spec.matrix.org/v1.12/client-server-api/#get_matrixclientv3roomsroomidmessages
func (cli *Client) Messages(ctx context.Context, roomID id.RoomID, from, to string, dir Direction, filter *FilterPart, limit int) (resp *RespMessages, err error) {
	query := map[string]string{
		"dir": string(dir),
	}
	if filter != nil {
		filterJSON, err := json.Marshal(filter)
		if err != nil {
			return nil, err
		}
		query["filter"] = string(filterJSON)
	}
	if from != "" {
		query["from"] = from
	}
	if to != "" {
		query["to"] = to
	}
	if limit != 0 {
		query["limit"] = strconv.Itoa(limit)
	}

	urlPath := cli.BuildURLWithQuery(ClientURLPath{"v3", "rooms", roomID, "messages"}, query)
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	return
}

// TimestampToEvent finds the ID of the event closest to the given timestamp.
//
// See https://spec.matrix.org/v1.6/client-server-api/#get_matrixclientv1roomsroomidtimestamp_to_event
func (cli *Client) TimestampToEvent(ctx context.Context, roomID id.RoomID, timestamp time.Time, dir Direction) (resp *RespTimestampToEvent, err error) {
	query := map[string]string{
		"ts":  strconv.FormatInt(timestamp.UnixMilli(), 10),
		"dir": string(dir),
	}
	urlPath := cli.BuildURLWithQuery(ClientURLPath{"v1", "rooms", roomID, "timestamp_to_event"}, query)
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	return
}

// Context returns a number of events that happened just before and after the
// specified event. It use pagination query parameters to paginate history in
// the room.
// See https://spec.matrix.org/v1.2/client-server-api/#get_matrixclientv3roomsroomidcontexteventid
func (cli *Client) Context(ctx context.Context, roomID id.RoomID, eventID id.EventID, filter *FilterPart, limit int) (resp *RespContext, err error) {
	query := map[string]string{}
	if filter != nil {
		filterJSON, err := json.Marshal(filter)
		if err != nil {
			return nil, err
		}
		query["filter"] = string(filterJSON)
	}
	if limit != 0 {
		query["limit"] = strconv.Itoa(limit)
	}

	urlPath := cli.BuildURLWithQuery(ClientURLPath{"v3", "rooms", roomID, "context", eventID}, query)
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	return
}

func (cli *Client) GetEvent(ctx context.Context, roomID id.RoomID, eventID id.EventID) (resp *event.Event, err error) {
	urlPath := cli.BuildClientURL("v3", "rooms", roomID, "event", eventID)
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	return
}

func (cli *Client) GetUnredactedEventContent(ctx context.Context, roomID id.RoomID, eventID id.EventID) (resp *event.Event, err error) {
	urlPath := cli.BuildURLWithQuery(ClientURLPath{"v3", "rooms", roomID, "event", eventID}, map[string]string{
		"fi.mau.msc2815.include_unredacted_content": "true",
	})
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	return
}

func (cli *Client) GetRelations(ctx context.Context, roomID id.RoomID, eventID id.EventID, req *ReqGetRelations) (resp *RespGetRelations, err error) {
	urlPath := cli.BuildURLWithQuery(append(ClientURLPath{"v1", "rooms", roomID, "relations", eventID}, req.PathSuffix()...), req.Query())
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	return
}

func (cli *Client) MarkRead(ctx context.Context, roomID id.RoomID, eventID id.EventID) (err error) {
	return cli.SendReceipt(ctx, roomID, eventID, event.ReceiptTypeRead, nil)
}

// MarkReadWithContent sends a read receipt including custom data.
//
// Deprecated: Use SendReceipt instead.
func (cli *Client) MarkReadWithContent(ctx context.Context, roomID id.RoomID, eventID id.EventID, content interface{}) (err error) {
	return cli.SendReceipt(ctx, roomID, eventID, event.ReceiptTypeRead, content)
}

// SendReceipt sends a receipt, usually specifically a read receipt.
//
// Passing nil as the content is safe, the library will automatically replace it with an empty JSON object.
// To mark a message in a specific thread as read, use pass a ReqSendReceipt as the content.
func (cli *Client) SendReceipt(ctx context.Context, roomID id.RoomID, eventID id.EventID, receiptType event.ReceiptType, content interface{}) (err error) {
	urlPath := cli.BuildClientURL("v3", "rooms", roomID, "receipt", receiptType, eventID)
	_, err = cli.MakeRequest(ctx, http.MethodPost, urlPath, content, nil)
	return
}

func (cli *Client) SetReadMarkers(ctx context.Context, roomID id.RoomID, content interface{}) (err error) {
	urlPath := cli.BuildClientURL("v3", "rooms", roomID, "read_markers")
	_, err = cli.MakeRequest(ctx, http.MethodPost, urlPath, content, nil)
	return
}

func (cli *Client) SetBeeperInboxState(ctx context.Context, roomID id.RoomID, content *ReqSetBeeperInboxState) (err error) {
	urlPath := cli.BuildClientURL("unstable", "com.beeper.inbox", "user", cli.UserID, "rooms", roomID, "inbox_state")
	_, err = cli.MakeRequest(ctx, http.MethodPut, urlPath, content, nil)
	return
}

func (cli *Client) AddTag(ctx context.Context, roomID id.RoomID, tag event.RoomTag, order float64) error {
	return cli.AddTagWithCustomData(ctx, roomID, tag, &event.TagMetadata{
		Order: json.Number(strconv.FormatFloat(order, 'e', -1, 64)),
	})
}

func (cli *Client) AddTagWithCustomData(ctx context.Context, roomID id.RoomID, tag event.RoomTag, data any) (err error) {
	urlPath := cli.BuildClientURL("v3", "user", cli.UserID, "rooms", roomID, "tags", tag)
	_, err = cli.MakeRequest(ctx, http.MethodPut, urlPath, data, nil)
	return
}

func (cli *Client) GetTags(ctx context.Context, roomID id.RoomID) (tags event.TagEventContent, err error) {
	err = cli.GetTagsWithCustomData(ctx, roomID, &tags)
	return
}

func (cli *Client) GetTagsWithCustomData(ctx context.Context, roomID id.RoomID, resp any) (err error) {
	urlPath := cli.BuildClientURL("v3", "user", cli.UserID, "rooms", roomID, "tags")
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	return
}

func (cli *Client) RemoveTag(ctx context.Context, roomID id.RoomID, tag event.RoomTag) (err error) {
	urlPath := cli.BuildClientURL("v3", "user", cli.UserID, "rooms", roomID, "tags", tag)
	_, err = cli.MakeRequest(ctx, http.MethodDelete, urlPath, nil, nil)
	return
}

// Deprecated: Synapse may not handle setting m.tag directly properly, so you should use the Add/RemoveTag methods instead.
func (cli *Client) SetTags(ctx context.Context, roomID id.RoomID, tags event.Tags) (err error) {
	return cli.SetRoomAccountData(ctx, roomID, "m.tag", map[string]event.Tags{
		"tags": tags,
	})
}

// TurnServer returns turn server details and credentials for the client to use when initiating calls.
// See https://spec.matrix.org/v1.2/client-server-api/#get_matrixclientv3voipturnserver
func (cli *Client) TurnServer(ctx context.Context) (resp *RespTurnServer, err error) {
	urlPath := cli.BuildClientURL("v3", "voip", "turnServer")
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	return
}

func (cli *Client) CreateAlias(ctx context.Context, alias id.RoomAlias, roomID id.RoomID) (resp *RespAliasCreate, err error) {
	urlPath := cli.BuildClientURL("v3", "directory", "room", alias)
	_, err = cli.MakeRequest(ctx, http.MethodPut, urlPath, &ReqAliasCreate{RoomID: roomID}, &resp)
	return
}

func (cli *Client) ResolveAlias(ctx context.Context, alias id.RoomAlias) (resp *RespAliasResolve, err error) {
	urlPath := cli.BuildClientURL("v3", "directory", "room", alias)
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	return
}

func (cli *Client) DeleteAlias(ctx context.Context, alias id.RoomAlias) (resp *RespAliasDelete, err error) {
	urlPath := cli.BuildClientURL("v3", "directory", "room", alias)
	_, err = cli.MakeRequest(ctx, http.MethodDelete, urlPath, nil, &resp)
	return
}

func (cli *Client) GetAliases(ctx context.Context, roomID id.RoomID) (resp *RespAliasList, err error) {
	urlPath := cli.BuildClientURL("v3", "rooms", roomID, "aliases")
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	return
}

func (cli *Client) UploadKeys(ctx context.Context, req *ReqUploadKeys) (resp *RespUploadKeys, err error) {
	urlPath := cli.BuildClientURL("v3", "keys", "upload")
	_, err = cli.MakeRequest(ctx, http.MethodPost, urlPath, req, &resp)
	return
}

func (cli *Client) QueryKeys(ctx context.Context, req *ReqQueryKeys) (resp *RespQueryKeys, err error) {
	urlPath := cli.BuildClientURL("v3", "keys", "query")
	_, err = cli.MakeRequest(ctx, http.MethodPost, urlPath, req, &resp)
	return
}

func (cli *Client) ClaimKeys(ctx context.Context, req *ReqClaimKeys) (resp *RespClaimKeys, err error) {
	urlPath := cli.BuildClientURL("v3", "keys", "claim")
	_, err = cli.MakeRequest(ctx, http.MethodPost, urlPath, req, &resp)
	return
}

func (cli *Client) GetKeyChanges(ctx context.Context, from, to string) (resp *RespKeyChanges, err error) {
	urlPath := cli.BuildURLWithQuery(ClientURLPath{"v3", "keys", "changes"}, map[string]string{
		"from": from,
		"to":   to,
	})
	_, err = cli.MakeRequest(ctx, http.MethodPost, urlPath, nil, &resp)
	return
}

// GetKeyBackup retrieves the keys from the backup.
//
// See: https://spec.matrix.org/v1.9/client-server-api/#get_matrixclientv3room_keyskeys
func (cli *Client) GetKeyBackup(ctx context.Context, version id.KeyBackupVersion) (resp *RespRoomKeys[backup.EncryptedSessionData[backup.MegolmSessionData]], err error) {
	urlPath := cli.BuildURLWithQuery(ClientURLPath{"v3", "room_keys", "keys"}, map[string]string{
		"version": string(version),
	})
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	return
}

// PutKeysInBackup stores several keys in the backup.
//
// See: https://spec.matrix.org/v1.9/client-server-api/#put_matrixclientv3room_keyskeys
func (cli *Client) PutKeysInBackup(ctx context.Context, version id.KeyBackupVersion, req *ReqKeyBackup) (resp *RespRoomKeysUpdate, err error) {
	urlPath := cli.BuildURLWithQuery(ClientURLPath{"v3", "room_keys", "keys"}, map[string]string{
		"version": string(version),
	})
	_, err = cli.MakeRequest(ctx, http.MethodPut, urlPath, req, &resp)
	return
}

// DeleteKeyBackup deletes all keys from the backup.
//
// See: https://spec.matrix.org/v1.9/client-server-api/#delete_matrixclientv3room_keyskeys
func (cli *Client) DeleteKeyBackup(ctx context.Context, version id.KeyBackupVersion) (resp *RespRoomKeysUpdate, err error) {
	urlPath := cli.BuildURLWithQuery(ClientURLPath{"v3", "room_keys", "keys"}, map[string]string{
		"version": string(version),
	})
	_, err = cli.MakeRequest(ctx, http.MethodDelete, urlPath, nil, &resp)
	return
}

// GetKeyBackupForRoom retrieves the keys from the backup for the given room.
//
// See: https://spec.matrix.org/v1.9/client-server-api/#get_matrixclientv3room_keyskeysroomid
func (cli *Client) GetKeyBackupForRoom(
	ctx context.Context, version id.KeyBackupVersion, roomID id.RoomID,
) (resp *RespRoomKeyBackup[backup.EncryptedSessionData[backup.MegolmSessionData]], err error) {
	urlPath := cli.BuildURLWithQuery(ClientURLPath{"v3", "room_keys", "keys", roomID.String()}, map[string]string{
		"version": string(version),
	})
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	return
}

// PutKeysInBackupForRoom stores several keys in the backup for the given room.
//
// See: https://spec.matrix.org/v1.9/client-server-api/#put_matrixclientv3room_keyskeysroomid
func (cli *Client) PutKeysInBackupForRoom(ctx context.Context, version id.KeyBackupVersion, roomID id.RoomID, req *ReqRoomKeyBackup) (resp *RespRoomKeysUpdate, err error) {
	urlPath := cli.BuildURLWithQuery(ClientURLPath{"v3", "room_keys", "keys", roomID.String()}, map[string]string{
		"version": string(version),
	})
	_, err = cli.MakeRequest(ctx, http.MethodPut, urlPath, req, &resp)
	return
}

// DeleteKeysFromBackupForRoom deletes all the keys in the backup for the given
// room.
//
// See: https://spec.matrix.org/v1.9/client-server-api/#delete_matrixclientv3room_keyskeysroomid
func (cli *Client) DeleteKeysFromBackupForRoom(ctx context.Context, version id.KeyBackupVersion, roomID id.RoomID) (resp *RespRoomKeysUpdate, err error) {
	urlPath := cli.BuildURLWithQuery(ClientURLPath{"v3", "room_keys", "keys", roomID.String()}, map[string]string{
		"version": string(version),
	})
	_, err = cli.MakeRequest(ctx, http.MethodDelete, urlPath, nil, &resp)
	return
}

// GetKeyBackupForRoomAndSession retrieves a key from the backup.
//
// See: https://spec.matrix.org/v1.9/client-server-api/#get_matrixclientv3room_keyskeysroomidsessionid
func (cli *Client) GetKeyBackupForRoomAndSession(
	ctx context.Context, version id.KeyBackupVersion, roomID id.RoomID, sessionID id.SessionID,
) (resp *RespKeyBackupData[backup.EncryptedSessionData[backup.MegolmSessionData]], err error) {
	urlPath := cli.BuildURLWithQuery(ClientURLPath{"v3", "room_keys", "keys", roomID.String(), sessionID.String()}, map[string]string{
		"version": string(version),
	})
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	return
}

// PutKeysInBackupForRoomAndSession stores a key in the backup.
//
// See: https://spec.matrix.org/v1.9/client-server-api/#put_matrixclientv3room_keyskeysroomidsessionid
func (cli *Client) PutKeysInBackupForRoomAndSession(ctx context.Context, version id.KeyBackupVersion, roomID id.RoomID, sessionID id.SessionID, req *ReqKeyBackupData) (resp *RespRoomKeysUpdate, err error) {
	urlPath := cli.BuildURLWithQuery(ClientURLPath{"v3", "room_keys", "keys", roomID.String(), sessionID.String()}, map[string]string{
		"version": string(version),
	})
	_, err = cli.MakeRequest(ctx, http.MethodPut, urlPath, req, &resp)
	return
}

// DeleteKeysInBackupForRoomAndSession deletes a key from the backup.
//
// See: https://spec.matrix.org/v1.9/client-server-api/#delete_matrixclientv3room_keyskeysroomidsessionid
func (cli *Client) DeleteKeysInBackupForRoomAndSession(ctx context.Context, version id.KeyBackupVersion, roomID id.RoomID, sessionID id.SessionID) (resp *RespRoomKeysUpdate, err error) {
	urlPath := cli.BuildURLWithQuery(ClientURLPath{"v3", "room_keys", "keys", roomID.String(), sessionID.String()}, map[string]string{
		"version": string(version),
	})
	_, err = cli.MakeRequest(ctx, http.MethodDelete, urlPath, nil, &resp)
	return
}

// GetKeyBackupLatestVersion returns information about the latest backup version.
//
// See: https://spec.matrix.org/v1.9/client-server-api/#get_matrixclientv3room_keysversion
func (cli *Client) GetKeyBackupLatestVersion(ctx context.Context) (resp *RespRoomKeysVersion[backup.MegolmAuthData], err error) {
	urlPath := cli.BuildClientURL("v3", "room_keys", "version")
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	return
}

// CreateKeyBackupVersion creates a new key backup.
//
// See: https://spec.matrix.org/v1.9/client-server-api/#post_matrixclientv3room_keysversion
func (cli *Client) CreateKeyBackupVersion(ctx context.Context, req *ReqRoomKeysVersionCreate[backup.MegolmAuthData]) (resp *RespRoomKeysVersionCreate, err error) {
	urlPath := cli.BuildClientURL("v3", "room_keys", "version")
	_, err = cli.MakeRequest(ctx, http.MethodPost, urlPath, req, &resp)
	return
}

// GetKeyBackupVersion returns information about an existing key backup.
//
// See: https://spec.matrix.org/v1.9/client-server-api/#get_matrixclientv3room_keysversionversion
func (cli *Client) GetKeyBackupVersion(ctx context.Context, version id.KeyBackupVersion) (resp *RespRoomKeysVersion[backup.MegolmAuthData], err error) {
	urlPath := cli.BuildClientURL("v3", "room_keys", "version", version)
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	return
}

// UpdateKeyBackupVersion updates information about an existing key backup. Only
// the auth_data can be modified.
//
// See: https://spec.matrix.org/v1.9/client-server-api/#put_matrixclientv3room_keysversionversion
func (cli *Client) UpdateKeyBackupVersion(ctx context.Context, version id.KeyBackupVersion, req *ReqRoomKeysVersionUpdate[backup.MegolmAuthData]) error {
	urlPath := cli.BuildClientURL("v3", "room_keys", "version", version)
	_, err := cli.MakeRequest(ctx, http.MethodPut, urlPath, nil, nil)
	return err
}

// DeleteKeyBackupVersion deletes an existing key backup. Both the information
// about the backup, as well as all key data related to the backup will be
// deleted.
//
// See: https://spec.matrix.org/v1.1/client-server-api/#delete_matrixclientv3room_keysversionversion
func (cli *Client) DeleteKeyBackupVersion(ctx context.Context, version id.KeyBackupVersion) error {
	urlPath := cli.BuildClientURL("v3", "room_keys", "version", version)
	_, err := cli.MakeRequest(ctx, http.MethodDelete, urlPath, nil, nil)
	return err
}

func (cli *Client) SendToDevice(ctx context.Context, eventType event.Type, req *ReqSendToDevice) (resp *RespSendToDevice, err error) {
	urlPath := cli.BuildClientURL("v3", "sendToDevice", eventType.String(), cli.TxnID())
	_, err = cli.MakeRequest(ctx, http.MethodPut, urlPath, req, &resp)
	return
}

func (cli *Client) GetDevicesInfo(ctx context.Context) (resp *RespDevicesInfo, err error) {
	urlPath := cli.BuildClientURL("v3", "devices")
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	return
}

func (cli *Client) GetDeviceInfo(ctx context.Context, deviceID id.DeviceID) (resp *RespDeviceInfo, err error) {
	urlPath := cli.BuildClientURL("v3", "devices", deviceID)
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	return
}

func (cli *Client) SetDeviceInfo(ctx context.Context, deviceID id.DeviceID, req *ReqDeviceInfo) error {
	urlPath := cli.BuildClientURL("v3", "devices", deviceID)
	_, err := cli.MakeRequest(ctx, http.MethodPut, urlPath, req, nil)
	return err
}

func (cli *Client) DeleteDevice(ctx context.Context, deviceID id.DeviceID, req *ReqDeleteDevice) error {
	urlPath := cli.BuildClientURL("v3", "devices", deviceID)
	_, err := cli.MakeRequest(ctx, http.MethodDelete, urlPath, req, nil)
	return err
}

func (cli *Client) DeleteDevices(ctx context.Context, req *ReqDeleteDevices) error {
	urlPath := cli.BuildClientURL("v3", "delete_devices")
	_, err := cli.MakeRequest(ctx, http.MethodPost, urlPath, req, nil)
	return err
}

type UIACallback = func(*RespUserInteractive) interface{}

// UploadCrossSigningKeys uploads the given cross-signing keys to the server.
// Because the endpoint requires user-interactive authentication a callback must be provided that,
// given the UI auth parameters, produces the required result (or nil to end the flow).
func (cli *Client) UploadCrossSigningKeys(ctx context.Context, keys *UploadCrossSigningKeysReq, uiaCallback UIACallback) error {
	content, err := cli.MakeFullRequest(ctx, FullRequest{
		Method:           http.MethodPost,
		URL:              cli.BuildClientURL("v3", "keys", "device_signing", "upload"),
		RequestJSON:      keys,
		SensitiveContent: keys.Auth != nil,
	})
	if respErr, ok := err.(HTTPError); ok && respErr.IsStatus(http.StatusUnauthorized) && uiaCallback != nil {
		// try again with UI auth
		var uiAuthResp RespUserInteractive
		if err := json.Unmarshal(content, &uiAuthResp); err != nil {
			return fmt.Errorf("failed to decode UIA response: %w", err)
		}
		auth := uiaCallback(&uiAuthResp)
		if auth != nil {
			keys.Auth = auth
			return cli.UploadCrossSigningKeys(ctx, keys, nil)
		}
	}
	return err
}

func (cli *Client) UploadSignatures(ctx context.Context, req *ReqUploadSignatures) (resp *RespUploadSignatures, err error) {
	urlPath := cli.BuildClientURL("v3", "keys", "signatures", "upload")
	_, err = cli.MakeRequest(ctx, http.MethodPost, urlPath, req, &resp)
	return
}

// GetPushRules returns the push notification rules for the global scope.
func (cli *Client) GetPushRules(ctx context.Context) (*pushrules.PushRuleset, error) {
	return cli.GetScopedPushRules(ctx, "global")
}

// GetScopedPushRules returns the push notification rules for the given scope.
func (cli *Client) GetScopedPushRules(ctx context.Context, scope string) (resp *pushrules.PushRuleset, err error) {
	u, _ := url.Parse(cli.BuildClientURL("v3", "pushrules", scope))
	// client.BuildURL returns the URL without a trailing slash, but the pushrules endpoint requires the slash.
	u.Path += "/"
	_, err = cli.MakeRequest(ctx, http.MethodGet, u.String(), nil, &resp)
	return
}

func (cli *Client) GetPushRule(ctx context.Context, scope string, kind pushrules.PushRuleType, ruleID string) (resp *pushrules.PushRule, err error) {
	urlPath := cli.BuildClientURL("v3", "pushrules", scope, kind, ruleID)
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	if resp != nil {
		resp.Type = kind
	}
	return
}

func (cli *Client) DeletePushRule(ctx context.Context, scope string, kind pushrules.PushRuleType, ruleID string) error {
	urlPath := cli.BuildClientURL("v3", "pushrules", scope, kind, ruleID)
	_, err := cli.MakeRequest(ctx, http.MethodDelete, urlPath, nil, nil)
	return err
}

func (cli *Client) PutPushRule(ctx context.Context, scope string, kind pushrules.PushRuleType, ruleID string, req *ReqPutPushRule) error {
	query := make(map[string]string)
	if len(req.After) > 0 {
		query["after"] = req.After
	}
	if len(req.Before) > 0 {
		query["before"] = req.Before
	}
	urlPath := cli.BuildURLWithQuery(ClientURLPath{"v3", "pushrules", scope, kind, ruleID}, query)
	_, err := cli.MakeRequest(ctx, http.MethodPut, urlPath, req, nil)
	return err
}

func (cli *Client) ReportEvent(ctx context.Context, roomID id.RoomID, eventID id.EventID, reason string) error {
	urlPath := cli.BuildClientURL("v3", "rooms", roomID, "report", eventID)
	_, err := cli.MakeRequest(ctx, http.MethodPost, urlPath, &ReqReport{Reason: reason, Score: -100}, nil)
	return err
}

func (cli *Client) ReportRoom(ctx context.Context, roomID id.RoomID, reason string) error {
	urlPath := cli.BuildClientURL("v3", "rooms", roomID, "report")
	_, err := cli.MakeRequest(ctx, http.MethodPost, urlPath, &ReqReport{Reason: reason, Score: -100}, nil)
	return err
}

// AdminWhoIs fetches session information belonging to a specific user. Typically requires being a server admin.
//
// https://spec.matrix.org/v1.15/client-server-api/#get_matrixclientv3adminwhoisuserid
func (cli *Client) AdminWhoIs(ctx context.Context, userID id.UserID) (resp RespWhoIs, err error) {
	urlPath := cli.BuildClientURL("v3", "admin", "whois", userID)
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, &resp)
	return
}

// UnstableGetSuspendedStatus uses MSC4323 to check if a user is suspended.
func (cli *Client) UnstableGetSuspendedStatus(ctx context.Context, userID id.UserID) (res *RespSuspended, err error) {
	urlPath := cli.BuildClientURL("unstable", "uk.timedout.msc4323", "admin", "suspend", userID)
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, res)
	return
}

// UnstableGetLockStatus uses MSC4323 to check if a user is locked.
func (cli *Client) UnstableGetLockStatus(ctx context.Context, userID id.UserID) (res *RespLocked, err error) {
	urlPath := cli.BuildClientURL("unstable", "uk.timedout.msc4323", "admin", "lock", userID)
	_, err = cli.MakeRequest(ctx, http.MethodGet, urlPath, nil, res)
	return
}

// UnstableSetSuspendedStatus uses MSC4323 to set whether a user account is suspended.
func (cli *Client) UnstableSetSuspendedStatus(ctx context.Context, userID id.UserID, suspended bool) (res *RespSuspended, err error) {
	urlPath := cli.BuildClientURL("unstable", "uk.timedout.msc4323", "admin", "suspend", userID)
	_, err = cli.MakeRequest(ctx, http.MethodPut, urlPath, &ReqSuspend{Suspended: suspended}, res)
	return
}

// UnstableSetLockStatus uses MSC4323 to set whether a user account is locked.
func (cli *Client) UnstableSetLockStatus(ctx context.Context, userID id.UserID, locked bool) (res *RespLocked, err error) {
	urlPath := cli.BuildClientURL("unstable", "uk.timedout.msc4323", "admin", "lock", userID)
	_, err = cli.MakeRequest(ctx, http.MethodPut, urlPath, &ReqLocked{Locked: locked}, res)
	return
}

func (cli *Client) AppservicePing(ctx context.Context, id, txnID string) (resp *RespAppservicePing, err error) {
	_, err = cli.MakeFullRequest(ctx, FullRequest{
		Method:       http.MethodPost,
		URL:          cli.BuildClientURL("v1", "appservice", id, "ping"),
		RequestJSON:  &ReqAppservicePing{TxnID: txnID},
		ResponseJSON: &resp,
		// This endpoint intentionally returns 50x, so don't retry
		MaxAttempts: 1,
	})
	return
}

func (cli *Client) BeeperBatchSend(ctx context.Context, roomID id.RoomID, req *ReqBeeperBatchSend) (resp *RespBeeperBatchSend, err error) {
	u := cli.BuildClientURL("unstable", "com.beeper.backfill", "rooms", roomID, "batch_send")
	_, err = cli.MakeRequest(ctx, http.MethodPost, u, req, &resp)
	return
}

func (cli *Client) BeeperMergeRooms(ctx context.Context, req *ReqBeeperMergeRoom) (resp *RespBeeperMergeRoom, err error) {
	urlPath := cli.BuildClientURL("unstable", "com.beeper.chatmerging", "merge")
	_, err = cli.MakeRequest(ctx, http.MethodPost, urlPath, req, &resp)
	return
}

func (cli *Client) BeeperSplitRoom(ctx context.Context, req *ReqBeeperSplitRoom) (resp *RespBeeperSplitRoom, err error) {
	urlPath := cli.BuildClientURL("unstable", "com.beeper.chatmerging", "rooms", req.RoomID, "split")
	_, err = cli.MakeRequest(ctx, http.MethodPost, urlPath, req, &resp)
	return
}
func (cli *Client) BeeperDeleteRoom(ctx context.Context, roomID id.RoomID) (err error) {
	urlPath := cli.BuildClientURL("unstable", "com.beeper.yeet", "rooms", roomID, "delete")
	_, err = cli.MakeRequest(ctx, http.MethodPost, urlPath, nil, nil)
	return
}

// TxnID returns the next transaction ID.
func (cli *Client) TxnID() string {
	if cli == nil {
		return "client is nil"
	}
	txnID := atomic.AddInt32(&cli.txnID, 1)
	return fmt.Sprintf("mautrix-go_%d_%d", time.Now().UnixNano(), txnID)
}

// NewClient creates a new Matrix Client ready for syncing
func NewClient(homeserverURL string, userID id.UserID, accessToken string) (*Client, error) {
	hsURL, err := ParseAndNormalizeBaseURL(homeserverURL)
	if err != nil {
		return nil, err
	}
	return &Client{
		AccessToken:   accessToken,
		UserAgent:     DefaultUserAgent,
		HomeserverURL: hsURL,
		UserID:        userID,
		Client:        &http.Client{Timeout: 180 * time.Second},
		Syncer:        NewDefaultSyncer(),
		Log:           zerolog.Nop(),
		// By default, use an in-memory store which will never save filter ids / next batch tokens to disk.
		// The client will work with this storer: it just won't remember across restarts.
		// In practice, a database backend should be used.
		Store: NewMemorySyncStore(),
	}, nil
}
