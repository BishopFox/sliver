package httpclient

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/bishopfox/sliver/implant/sliver/cryptography"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
)

type staticHTTPDriver struct {
	response *http.Response
	err      error
}

func (driver *staticHTTPDriver) Do(*http.Request) (*http.Response, error) {
	return driver.response, driver.err
}

type trackingHTTPBody struct {
	reader     io.Reader
	readToEOF  bool
	closeCalls int
	closeErr   error
}

func (body *trackingHTTPBody) Read(buffer []byte) (int, error) {
	count, err := body.reader.Read(buffer)
	if errors.Is(err, io.EOF) {
		body.readToEOF = true
	}
	return count, err
}

func (body *trackingHTTPBody) Close() error {
	body.closeCalls++
	return body.closeErr
}

func newHTTPWriteTestClient(response *http.Response) *SliverHTTPClient {
	key := [32]byte{1, 2, 3, 4}
	return &SliverHTTPClient{
		Origin:     "http://127.0.0.1",
		driver:     &staticHTTPDriver{response: response},
		SessionCtx: cryptography.NewCipherContext(key),
		SessionID:  "test-session",
		Options:    &HTTPOptions{},
	}
}

func TestWriteEnvelopeDrainsAndClosesResponseBody(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantErr error
	}{
		{name: "accepted", status: http.StatusAccepted},
		{name: "rejected", status: http.StatusInternalServerError, wantErr: ErrStatusCodeUnexpected},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := &trackingHTTPBody{reader: bytes.NewBufferString("response-body")}
			client := newHTTPWriteTestClient(&http.Response{StatusCode: test.status, Body: body})
			err := client.WriteEnvelope(&sliverpb.Envelope{Type: sliverpb.MsgPing})
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("WriteEnvelope() error = %v, want %v", err, test.wantErr)
			}
			if !body.readToEOF {
				t.Fatal("WriteEnvelope did not drain the HTTP response body")
			}
			if body.closeCalls != 1 {
				t.Fatalf("HTTP response body closed %d times, want 1", body.closeCalls)
			}
		})
	}
}

func TestWriteEnvelopeClosesResponseBodyAfterDrainFailure(t *testing.T) {
	drainErr := errors.New("response read failed")
	body := &trackingHTTPBody{reader: errorHTTPReader{err: drainErr}}
	client := newHTTPWriteTestClient(&http.Response{StatusCode: http.StatusAccepted, Body: body})

	err := client.WriteEnvelope(&sliverpb.Envelope{Type: sliverpb.MsgPing})
	if !errors.Is(err, drainErr) {
		t.Fatalf("WriteEnvelope() error = %v, want %v", err, drainErr)
	}
	if body.closeCalls != 1 {
		t.Fatalf("HTTP response body closed %d times after drain failure, want 1", body.closeCalls)
	}
}

func TestWriteEnvelopeClosesResponseBodyWhenDriverReturnsResponseAndError(t *testing.T) {
	driverErr := errors.New("HTTP driver failed")
	body := &trackingHTTPBody{reader: bytes.NewBufferString("error-response")}
	client := newHTTPWriteTestClient(&http.Response{StatusCode: http.StatusBadGateway, Body: body})
	client.driver.(*staticHTTPDriver).err = driverErr

	err := client.WriteEnvelope(&sliverpb.Envelope{Type: sliverpb.MsgPing})
	if !errors.Is(err, driverErr) {
		t.Fatalf("WriteEnvelope() error = %v, want %v", err, driverErr)
	}
	if !body.readToEOF {
		t.Fatal("WriteEnvelope did not drain the HTTP error response body")
	}
	if body.closeCalls != 1 {
		t.Fatalf("HTTP error response body closed %d times, want 1", body.closeCalls)
	}
}

type errorHTTPReader struct {
	err error
}

func (reader errorHTTPReader) Read([]byte) (int, error) {
	return 0, reader.err
}
