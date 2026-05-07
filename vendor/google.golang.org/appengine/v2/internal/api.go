// Copyright 2011 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package internal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"

	remotepb "google.golang.org/appengine/v2/internal/remote_api"
)

const (
	apiPath = "/rpc_http"
)

var (
	// Incoming headers.
	ticketHeader       = http.CanonicalHeaderKey("X-AppEngine-API-Ticket")
	dapperHeader       = http.CanonicalHeaderKey("X-Google-DapperTraceInfo")
	traceHeader        = http.CanonicalHeaderKey("X-Cloud-Trace-Context")
	curNamespaceHeader = http.CanonicalHeaderKey("X-AppEngine-Current-Namespace")
	userIPHeader       = http.CanonicalHeaderKey("X-AppEngine-User-IP")
	remoteAddrHeader   = http.CanonicalHeaderKey("X-AppEngine-Remote-Addr")
	devRequestIdHeader = http.CanonicalHeaderKey("X-Appengine-Dev-Request-Id")

	// Outgoing headers.
	apiEndpointHeader      = http.CanonicalHeaderKey("X-Google-RPC-Service-Endpoint")
	apiEndpointHeaderValue = []string{"app-engine-apis"}
	apiMethodHeader        = http.CanonicalHeaderKey("X-Google-RPC-Service-Method")
	apiMethodHeaderValue   = []string{"/VMRemoteAPI.CallRemoteAPI"}
	apiDeadlineHeader      = http.CanonicalHeaderKey("X-Google-RPC-Service-Deadline")
	apiContentType         = http.CanonicalHeaderKey("Content-Type")
	apiContentTypeValue    = []string{"application/octet-stream"}
	logFlushHeader         = http.CanonicalHeaderKey("X-AppEngine-Log-Flush-Count")

	apiHTTPClient = &http.Client{
		Transport: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			Dial:                limitDial,
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 10000,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	logStream io.Writer        = os.Stderr // For test hooks.
	timeNow   func() time.Time = time.Now  // For test hooks.
)

func apiURL(ctx context.Context) *url.URL {
	host, port := "appengine.googleapis.internal", "10001"
	if h := os.Getenv("API_HOST"); h != "" {
		host = h
	}
	if hostOverride := ctx.Value(apiHostOverrideKey); hostOverride != nil {
		host = hostOverride.(string)
	}
	if p := os.Getenv("API_PORT"); p != "" {
		port = p
	}
	if portOverride := ctx.Value(apiPortOverrideKey); portOverride != nil {
		port = portOverride.(string)
	}
	return &url.URL{
		Scheme: "http",
		Host:   host + ":" + port,
		Path:   apiPath,
	}
}

// Middleware wraps an http handler so that it can make GAE API calls
func Middleware(next http.Handler) http.Handler {
	return handleHTTPMiddleware(executeRequestSafelyMiddleware(next))
}

func handleHTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := &aeContext{
			req:       r,
			outHeader: w.Header(),
		}
		r = r.WithContext(withContext(r.Context(), c))
		c.req = r

		// Patch up RemoteAddr so it looks reasonable.
		if addr := r.Header.Get(userIPHeader); addr != "" {
			r.RemoteAddr = addr
		} else if addr = r.Header.Get(remoteAddrHeader); addr != "" {
			r.RemoteAddr = addr
		} else {
			// Should not normally reach here, but pick a sensible default anyway.
			r.RemoteAddr = "127.0.0.1"
		}
		// The address in the headers will most likely be of these forms:
		//	123.123.123.123
		//	2001:db8::1
		// net/http.Request.RemoteAddr is specified to be in "IP:port" form.
		if _, _, err := net.SplitHostPort(r.RemoteAddr); err != nil {
			// Assume the remote address is only a host; add a default port.
			r.RemoteAddr = net.JoinHostPort(r.RemoteAddr, "80")
		}

		next.ServeHTTP(c, r)
		c.outHeader = nil // make sure header changes aren't respected any more

		// Avoid nil Write call if c.Write is never called.
		if c.outCode != 0 {
			w.WriteHeader(c.outCode)
		}
		if c.outBody != nil {
			w.Write(c.outBody)
		}
	})
}

func executeRequestSafelyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if x := recover(); x != nil {
				c := w.(*aeContext)
				logf(c, 4, "%s", renderPanic(x)) // 4 == critical
				c.outCode = 500
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func renderPanic(x interface{}) string {
	buf := make([]byte, 16<<10) // 16 KB should be plenty
	buf = buf[:runtime.Stack(buf, false)]

	// Remove the first few stack frames:
	//   this func
	//   the recover closure in the caller
	// That will root the stack trace at the site of the panic.
	const (
		skipStart  = "internal.renderPanic"
		skipFrames = 2
	)
	start := bytes.Index(buf, []byte(skipStart))
	p := start
	for i := 0; i < skipFrames*2 && p+1 < len(buf); i++ {
		p = bytes.IndexByte(buf[p+1:], '\n') + p + 1
		if p < 0 {
			break
		}
	}
	if p >= 0 {
		// buf[start:p+1] is the block to remove.
		// Copy buf[p+1:] over buf[start:] and shrink buf.
		copy(buf[start:], buf[p+1:])
		buf = buf[:len(buf)-(p+1-start)]
	}

	// Add panic heading.
	head := fmt.Sprintf("panic: %v\n\n", x)
	if len(head) > len(buf) {
		// Extremely unlikely to happen.
		return head
	}
	copy(buf[len(head):], buf)
	copy(buf, head)

	return string(buf)
}

// aeContext represents the context of an in-flight HTTP request.
// It implements the appengine.Context and http.ResponseWriter interfaces.
type aeContext struct {
	req *http.Request

	outCode   int
	outHeader http.Header
	outBody   []byte
}

var contextKey = "holds a *context"

// jointContext joins two contexts in a superficial way.
// It takes values and timeouts from a base context, and only values from another context.
type jointContext struct {
	base       context.Context
	valuesOnly context.Context
}

func (c jointContext) Deadline() (time.Time, bool) {
	return c.base.Deadline()
}

func (c jointContext) Done() <-chan struct{} {
	return c.base.Done()
}

func (c jointContext) Err() error {
	return c.base.Err()
}

func (c jointContext) Value(key interface{}) interface{} {
	if val := c.base.Value(key); val != nil {
		return val
	}
	return c.valuesOnly.Value(key)
}

// fromContext returns the App Engine context or nil if ctx is not
// derived from an App Engine context.
func fromContext(ctx context.Context) *aeContext {
	c, _ := ctx.Value(&contextKey).(*aeContext)
	return c
}

func withContext(parent context.Context, c *aeContext) context.Context {
	ctx := context.WithValue(parent, &contextKey, c)
	if ns := c.req.Header.Get(curNamespaceHeader); ns != "" {
		ctx = withNamespace(ctx, ns)
	}
	return ctx
}

func toContext(c *aeContext) context.Context {
	return withContext(context.Background(), c)
}

func IncomingHeaders(ctx context.Context) http.Header {
	if c := fromContext(ctx); c != nil {
		return c.req.Header
	}
	return nil
}

func ReqContext(req *http.Request) context.Context {
	return req.Context()
}

func WithContext(parent context.Context, req *http.Request) context.Context {
	return jointContext{
		base:       parent,
		valuesOnly: req.Context(),
	}
}

// RegisterTestRequest registers the HTTP request req for testing, such that
// any API calls are sent to the provided URL.
// It should only be used by test code or test helpers like aetest.
func RegisterTestRequest(req *http.Request, apiURL *url.URL, appID string) *http.Request {
	ctx := req.Context()
	ctx = withAPIHostOverride(ctx, apiURL.Hostname())
	ctx = withAPIPortOverride(ctx, apiURL.Port())
	ctx = WithAppIDOverride(ctx, appID)

	// use the unregistered request as a placeholder so that withContext can read the headers
	c := &aeContext{req: req}
	c.req = req.WithContext(withContext(ctx, c))
	return c.req
}

var errTimeout = &CallError{
	Detail:  "Deadline exceeded",
	Code:    int32(remotepb.RpcError_CANCELLED),
	Timeout: true,
}

func (c *aeContext) Header() http.Header { return c.outHeader }

// Copied from $GOROOT/src/pkg/net/http/transfer.go. Some response status
// codes do not permit a response body (nor response entity headers such as
// Content-Length, Content-Type, etc).
func bodyAllowedForStatus(status int) bool {
	switch {
	case status >= 100 && status <= 199:
		return false
	case status == 204:
		return false
	case status == 304:
		return false
	}
	return true
}

func (c *aeContext) Write(b []byte) (int, error) {
	if c.outCode == 0 {
		c.WriteHeader(http.StatusOK)
	}
	if len(b) > 0 && !bodyAllowedForStatus(c.outCode) {
		return 0, http.ErrBodyNotAllowed
	}
	c.outBody = append(c.outBody, b...)
	return len(b), nil
}

func (c *aeContext) WriteHeader(code int) {
	if c.outCode != 0 {
		logf(c, 3, "WriteHeader called multiple times on request.") // error level
		return
	}
	c.outCode = code
}

func post(ctx context.Context, body []byte, timeout time.Duration) (b []byte, err error) {
	apiURL := apiURL(ctx)
	hreq := &http.Request{
		Method: "POST",
		URL:    apiURL,
		Header: http.Header{
			apiEndpointHeader: apiEndpointHeaderValue,
			apiMethodHeader:   apiMethodHeaderValue,
			apiContentType:    apiContentTypeValue,
			apiDeadlineHeader: []string{strconv.FormatFloat(timeout.Seconds(), 'f', -1, 64)},
		},
		Body:          ioutil.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
		Host:          apiURL.Host,
	}
	c := fromContext(ctx)
	if c != nil {
		if info := c.req.Header.Get(dapperHeader); info != "" {
			hreq.Header.Set(dapperHeader, info)
		}
		if info := c.req.Header.Get(traceHeader); info != "" {
			hreq.Header.Set(traceHeader, info)
		}
	}

	tr := apiHTTPClient.Transport.(*http.Transport)

	var timedOut int32 // atomic; set to 1 if timed out
	t := time.AfterFunc(timeout, func() {
		atomic.StoreInt32(&timedOut, 1)
		tr.CancelRequest(hreq)
	})
	defer t.Stop()
	defer func() {
		// Check if timeout was exceeded.
		if atomic.LoadInt32(&timedOut) != 0 {
			err = errTimeout
		}
	}()

	hresp, err := apiHTTPClient.Do(hreq)
	if err != nil {
		return nil, &CallError{
			Detail: fmt.Sprintf("service bridge HTTP failed: %v", err),
			Code:   int32(remotepb.RpcError_UNKNOWN),
		}
	}
	defer hresp.Body.Close()
	hrespBody, err := ioutil.ReadAll(hresp.Body)
	if hresp.StatusCode != 200 {
		return nil, &CallError{
			Detail: fmt.Sprintf("service bridge returned HTTP %d (%q)", hresp.StatusCode, hrespBody),
			Code:   int32(remotepb.RpcError_UNKNOWN),
		}
	}
	if err != nil {
		return nil, &CallError{
			Detail: fmt.Sprintf("service bridge response bad: %v", err),
			Code:   int32(remotepb.RpcError_UNKNOWN),
		}
	}
	return hrespBody, nil
}

func Call(ctx context.Context, service, method string, in, out proto.Message) error {
	if ns := NamespaceFromContext(ctx); ns != "" {
		if fn, ok := NamespaceMods[service]; ok {
			fn(in, ns)
		}
	}

	if f, ctx, ok := callOverrideFromContext(ctx); ok {
		return f(ctx, service, method, in, out)
	}

	// Handle already-done contexts quickly.
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c := fromContext(ctx)

	// Apply transaction modifications if we're in a transaction.
	if t := transactionFromContext(ctx); t != nil {
		if t.finished {
			return errors.New("transaction context has expired")
		}
		applyTransaction(in, &t.transaction)
	}

	// Default RPC timeout is 60s.
	timeout := 60 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		timeout = deadline.Sub(time.Now())
	}

	data, err := proto.Marshal(in)
	if err != nil {
		return err
	}

	ticket := ""
	if c != nil {
		ticket = c.req.Header.Get(ticketHeader)
		if dri := c.req.Header.Get(devRequestIdHeader); IsDevAppServer() && dri != "" {
			ticket = dri
		}
	}
	req := &remotepb.Request{
		ServiceName: &service,
		Method:      &method,
		Request:     data,
		RequestId:   &ticket,
	}
	hreqBody, err := proto.Marshal(req)
	if err != nil {
		return err
	}

	hrespBody, err := post(ctx, hreqBody, timeout)
	if err != nil {
		return err
	}

	res := &remotepb.Response{}
	if err := proto.Unmarshal(hrespBody, res); err != nil {
		return err
	}
	if res.RpcError != nil {
		ce := &CallError{
			Detail: res.RpcError.GetDetail(),
			Code:   *res.RpcError.Code,
		}
		switch remotepb.RpcError_ErrorCode(ce.Code) {
		case remotepb.RpcError_CANCELLED, remotepb.RpcError_DEADLINE_EXCEEDED:
			ce.Timeout = true
		}
		return ce
	}
	if res.ApplicationError != nil {
		return &APIError{
			Service: *req.ServiceName,
			Detail:  res.ApplicationError.GetDetail(),
			Code:    *res.ApplicationError.Code,
		}
	}
	if res.Exception != nil || res.JavaException != nil {
		// This shouldn't happen, but let's be defensive.
		return &CallError{
			Detail: "service bridge returned exception",
			Code:   int32(remotepb.RpcError_UNKNOWN),
		}
	}
	return proto.Unmarshal(res.Response, out)
}

func (c *aeContext) Request() *http.Request {
	return c.req
}

func ContextForTesting(req *http.Request) context.Context {
	return toContext(&aeContext{req: req})
}
