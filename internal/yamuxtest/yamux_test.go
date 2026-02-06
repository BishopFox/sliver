package yamuxtest

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"

	"github.com/hashicorp/yamux"
)

func TestYamuxConcurrentStreams(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	serverSession, err := yamux.Server(serverConn, nil)
	if err != nil {
		t.Fatalf("server session: %v", err)
	}
	defer serverSession.Close()

	clientSession, err := yamux.Client(clientConn, nil)
	if err != nil {
		t.Fatalf("client session: %v", err)
	}
	defer clientSession.Close()

	const streams = 16
	payloads := make([][]byte, streams)
	for i := 0; i < streams; i++ {
		payloads[i] = []byte(fmt.Sprintf("stream=%d payload=%s", i, bytes.Repeat([]byte{byte('a' + i)}, 4096)))
	}

	var openWg sync.WaitGroup
	openWg.Add(streams)
	for i := 0; i < streams; i++ {
		i := i
		go func() {
			defer openWg.Done()
			stream, err := clientSession.Open()
			if err != nil {
				t.Errorf("open stream %d: %v", i, err)
				return
			}
			_, err = stream.Write(payloads[i])
			_ = stream.Close()
			if err != nil {
				t.Errorf("write stream %d: %v", i, err)
				return
			}
		}()
	}

	openWg.Wait()

	wantSet := map[string]struct{}{}
	for _, p := range payloads {
		wantSet[string(p)] = struct{}{}
	}

	gotSet := map[string]struct{}{}
	for i := 0; i < streams; i++ {
		stream, err := serverSession.Accept()
		if err != nil {
			t.Fatalf("accept stream %d: %v", i, err)
		}
		b, err := io.ReadAll(stream)
		_ = stream.Close()
		if err != nil {
			t.Fatalf("read stream %d: %v", i, err)
		}
		gotSet[string(b)] = struct{}{}
	}

	if len(gotSet) != len(wantSet) {
		t.Fatalf("unexpected payload count: got=%d want=%d", len(gotSet), len(wantSet))
	}
	for want := range wantSet {
		if _, ok := gotSet[want]; !ok {
			t.Fatalf("missing payload: %q", want)
		}
	}
}
