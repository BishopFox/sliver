package assets

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDownloadFileRetriesTransientHTTPStatus(t *testing.T) {
	content := []byte("complete asset")
	attempts := 0
	r := testDownloadRunner(downloadRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		switch attempts {
		case 1:
			return downloadResponse(req, http.StatusInternalServerError, nil, 0), nil
		case 2:
			return downloadResponse(req, http.StatusTooManyRequests, nil, 0), nil
		default:
			return downloadResponse(req, http.StatusOK, content, int64(len(content))), nil
		}
	}))

	dir := t.TempDir()
	dest := filepath.Join(dir, "asset.bin")
	if err := r.downloadFile("https://example.test/asset.bin", dest); err != nil {
		t.Fatalf("downloadFile() error = %v", err)
	}
	if attempts != defaultDownloadAttempts {
		t.Fatalf("download attempts = %d, want %d", attempts, defaultDownloadAttempts)
	}
	assertFileContents(t, dest, content)
	assertNoDownloadTemps(t, dir)
}

func TestDownloadFileRetriesShortBodyWithoutAppendingPartialData(t *testing.T) {
	partial := []byte("partial")
	content := []byte("complete asset")
	attempts := 0
	r := testDownloadRunner(downloadRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		if attempts == 1 {
			return downloadResponse(req, http.StatusOK, partial, int64(len(partial)+8)), nil
		}
		return downloadResponse(req, http.StatusOK, content, int64(len(content))), nil
	}))

	dir := t.TempDir()
	dest := filepath.Join(dir, "asset.bin")
	if err := r.downloadFile("https://example.test/asset.bin", dest); err != nil {
		t.Fatalf("downloadFile() error = %v", err)
	}
	if attempts != 2 {
		t.Fatalf("download attempts = %d, want 2", attempts)
	}
	assertFileContents(t, dest, content)
	assertNoDownloadTemps(t, dir)
}

func TestDownloadFileRetriesMidstreamTimeoutWithoutAppendingPartialData(t *testing.T) {
	partial := []byte("partial")
	content := []byte("complete asset")
	attempts := 0
	r := testDownloadRunner(downloadRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		if attempts == 1 {
			response := downloadResponse(req, http.StatusOK, nil, int64(len(partial)+8))
			response.Body = &errorAfterReadCloser{
				reader: bytes.NewReader(partial),
				err:    temporaryNetworkError{},
			}
			return response, nil
		}
		return downloadResponse(req, http.StatusOK, content, int64(len(content))), nil
	}))

	dir := t.TempDir()
	dest := filepath.Join(dir, "asset.bin")
	if err := r.downloadFile("https://example.test/asset.bin", dest); err != nil {
		t.Fatalf("downloadFile() error = %v", err)
	}
	if attempts != 2 {
		t.Fatalf("download attempts = %d, want 2", attempts)
	}
	assertFileContents(t, dest, content)
	assertNoDownloadTemps(t, dir)
}

func TestDownloadFileDoesNotRetryPermanentHTTPStatus(t *testing.T) {
	attempts := 0
	r := testDownloadRunner(downloadRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		return downloadResponse(req, http.StatusNotFound, nil, 0), nil
	}))

	dir := t.TempDir()
	dest := filepath.Join(dir, "asset.bin")
	sentinel := []byte("existing asset")
	if err := os.WriteFile(dest, sentinel, 0o600); err != nil {
		t.Fatalf("write existing destination: %v", err)
	}

	if err := r.downloadFile("https://example.test/missing.bin", dest); err == nil {
		t.Fatal("downloadFile() succeeded for HTTP 404")
	}
	if attempts != 1 {
		t.Fatalf("download attempts = %d, want 1", attempts)
	}
	assertFileContents(t, dest, sentinel)
	assertNoDownloadTemps(t, dir)
}

func TestDownloadFileExhaustsTransportRetriesSafely(t *testing.T) {
	attempts := 0
	r := testDownloadRunner(downloadRoundTripFunc(func(*http.Request) (*http.Response, error) {
		attempts++
		return nil, temporaryNetworkError{}
	}))

	dir := t.TempDir()
	dest := filepath.Join(dir, "asset.bin")
	sentinel := []byte("existing asset")
	if err := os.WriteFile(dest, sentinel, 0o600); err != nil {
		t.Fatalf("write existing destination: %v", err)
	}

	if err := r.downloadFile("https://example.test/asset.bin", dest); err == nil {
		t.Fatal("downloadFile() succeeded after exhausted transport errors")
	}
	if attempts != defaultDownloadAttempts {
		t.Fatalf("download attempts = %d, want %d", attempts, defaultDownloadAttempts)
	}
	assertFileContents(t, dest, sentinel)
	assertNoDownloadTemps(t, dir)
}

func TestDownloadFileDoesNotRetryPermanentTransportError(t *testing.T) {
	attempts := 0
	r := testDownloadRunner(downloadRoundTripFunc(func(*http.Request) (*http.Response, error) {
		attempts++
		return nil, errors.New("permanent certificate failure")
	}))

	dir := t.TempDir()
	dest := filepath.Join(dir, "asset.bin")
	if err := r.downloadFile("https://example.test/asset.bin", dest); err == nil {
		t.Fatal("downloadFile() succeeded after a permanent transport error")
	}
	if attempts != 1 {
		t.Fatalf("download attempts = %d, want 1", attempts)
	}
	assertNoDownloadTemps(t, dir)
}

func TestWaitForDownloadRetryHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := waitForDownloadRetry(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("waitForDownloadRetry() error = %v, want context.Canceled", err)
	}
}

func TestDownloadToTempHonorsRequestCancellation(t *testing.T) {
	started := make(chan struct{})
	attempts := 0
	r := testDownloadRunner(downloadRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		close(started)
		<-req.Context().Done()
		return nil, req.Context().Err()
	}))

	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := r.downloadToTempWithContext(ctx, "https://example.test/asset.bin", dir)
		result <- err
	}()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("download request did not start")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("downloadToTempWithContext() error = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("downloadToTempWithContext() did not stop after cancellation")
	}
	if attempts != 1 {
		t.Fatalf("download attempts = %d, want 1", attempts)
	}
	assertNoDownloadTemps(t, dir)
}

type downloadRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn downloadRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

type temporaryNetworkError struct{}

func (temporaryNetworkError) Error() string   { return "temporary network timeout" }
func (temporaryNetworkError) Timeout() bool   { return true }
func (temporaryNetworkError) Temporary() bool { return true }

type errorAfterReadCloser struct {
	reader *bytes.Reader
	err    error
}

func (r *errorAfterReadCloser) Read(data []byte) (int, error) {
	if r.reader.Len() > 0 {
		return r.reader.Read(data)
	}
	return 0, r.err
}

func (*errorAfterReadCloser) Close() error { return nil }

func testDownloadRunner(transport http.RoundTripper) *runner {
	return &runner{
		logger:             newLogger(false, true, true),
		httpClient:         &http.Client{Transport: transport},
		downloadAttempts:   defaultDownloadAttempts,
		downloadRetryDelay: 0,
	}
}

func downloadResponse(req *http.Request, statusCode int, body []byte, contentLength int64) *http.Response {
	return &http.Response{
		StatusCode:    statusCode,
		Status:        fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		Header:        make(http.Header),
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: contentLength,
		Request:       req,
	}
}

func assertFileContents(t *testing.T, path string, expected []byte) {
	t.Helper()
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !bytes.Equal(actual, expected) {
		t.Fatalf("%s = %q, want %q", path, actual, expected)
	}
}

func assertNoDownloadTemps(t *testing.T, dir string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "download-*"))
	if err != nil {
		t.Fatalf("glob download temp files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("leaked download temp files: %v", matches)
	}
}
