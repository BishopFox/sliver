package c2

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/server/core"
	serverCrypto "github.com/bishopfox/sliver/server/cryptography"
	serverHandlers "github.com/bishopfox/sliver/server/handlers"
	"google.golang.org/protobuf/proto"
)

type orderingResponseWriter struct {
	header http.Header
	status chan int
	once   sync.Once
}

type orderingIdentityEncoder struct{}

func (orderingIdentityEncoder) Encode(data []byte) ([]byte, error) { return data, nil }
func (orderingIdentityEncoder) Decode(data []byte) ([]byte, error) { return data, nil }

func newOrderingResponseWriter() *orderingResponseWriter {
	return &orderingResponseWriter{
		header: http.Header{},
		status: make(chan int, 1),
	}
}

func (writer *orderingResponseWriter) Header() http.Header {
	return writer.header
}

func (writer *orderingResponseWriter) WriteHeader(status int) {
	writer.once.Do(func() { writer.status <- status })
}

func (writer *orderingResponseWriter) Write(data []byte) (int, error) {
	writer.WriteHeader(http.StatusOK)
	return len(data), nil
}

func TestHTTPSessionAcknowledgesPostAfterSynchronousDispatch(t *testing.T) {
	key := [32]byte{9, 8, 7, 6}
	httpSession := &HTTPSession{
		ID:          "ordering-session",
		ImplantConn: core.NewImplantConnection("http(s)", "test"),
		CipherCtx:   serverCrypto.NewCipherContext(key),
	}
	t.Cleanup(httpSession.ImplantConn.Close)
	server := &SliverHTTPC2{
		HTTPSessions: &HTTPSessions{active: map[string]*HTTPSession{}, mutex: &sync.RWMutex{}},
	}

	envelopeData, err := proto.Marshal(&sliverpb.Envelope{Type: sliverpb.MsgPing})
	if err != nil {
		t.Fatalf("marshal HTTP test envelope: %v", err)
	}
	ciphertext, err := serverCrypto.Encrypt(key, envelopeData)
	if err != nil {
		t.Fatalf("encrypt HTTP test envelope: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/session", bytes.NewReader(ciphertext))
	response := newOrderingResponseWriter()

	dispatchStarted := make(chan struct{})
	dispatchRelease := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(dispatchRelease) }) }
	t.Cleanup(release)
	handlers := map[uint32]serverHandlers.ServerHandler{
		sliverpb.MsgPing: func(*core.ImplantConnection, []byte) *sliverpb.Envelope {
			close(dispatchStarted)
			<-dispatchRelease
			return nil
		},
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		server.sessionHandlerWithHandlers(
			response,
			request,
			httpSession,
			&clientpb.HTTPC2Config{},
			orderingIdentityEncoder{},
			handlers,
		)
	}()

	select {
	case <-dispatchStarted:
	case <-time.After(time.Second):
		t.Fatal("HTTP session handler did not begin envelope dispatch")
	}
	select {
	case status := <-response.status:
		t.Fatalf("HTTP session acknowledged status %d before dispatch completed", status)
	case <-time.After(50 * time.Millisecond):
	}

	release()
	select {
	case status := <-response.status:
		if status != http.StatusAccepted {
			t.Fatalf("HTTP session acknowledgement = %d, want %d", status, http.StatusAccepted)
		}
	case <-time.After(time.Second):
		t.Fatal("HTTP session did not acknowledge completed dispatch")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("HTTP session handler did not return after dispatch")
	}
}
