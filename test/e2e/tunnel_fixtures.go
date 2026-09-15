package e2e

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

const (
	deterministicHTTPPath                 = "/sliver-e2e/tunnel"
	tunnelFullDuplexPayloadBytes          = 8 * 1024 * 1024
	tunnelFullDuplexMinimumBytesPerSecond = 384 * 1024
)

var tunnelFullDuplexChunks = []int{1, 4095, 4108, 32768, 65537}

func deterministicTunnelPayload(label string, size int) []byte {
	if size <= 0 {
		return nil
	}
	payload := make([]byte, size)
	seed := []byte(label)
	var counter uint64
	for offset := 0; offset < len(payload); {
		hasher := sha256.New()
		_, _ = hasher.Write(seed)
		var encodedCounter [8]byte
		binary.LittleEndian.PutUint64(encodedCounter[:], counter)
		_, _ = hasher.Write(encodedCounter[:])
		block := hasher.Sum(nil)
		offset += copy(payload[offset:], block)
		counter++
	}
	return payload
}

func tunnelEchoRoundTrip(ctx context.Context, address string, payload []byte, chunks []int) error {
	connection, err := (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, "tcp4", address)
	if err != nil {
		return fmt.Errorf("dial tunnel fixture %s: %w", address, err)
	}
	defer func() { _ = connection.Close() }()
	return tunnelEchoRoundTripOnConn(ctx, connection, payload, chunks)
}

func tunnelEchoRoundTripOnConn(ctx context.Context, connection net.Conn, payload []byte, chunks []int) error {
	if len(payload) == 0 {
		return errors.New("tunnel echo payload must not be empty")
	}
	if len(chunks) == 0 {
		chunks = []int{len(payload)}
	}
	if deadline, ok := ctx.Deadline(); ok {
		if err := connection.SetDeadline(deadline); err != nil {
			return fmt.Errorf("set tunnel fixture deadline: %w", err)
		}
	}

	for offset, chunkIndex := 0, 0; offset < len(payload); chunkIndex++ {
		chunkSize := chunks[chunkIndex%len(chunks)]
		if chunkSize <= 0 {
			return fmt.Errorf("invalid tunnel echo chunk size %d", chunkSize)
		}
		end := offset + chunkSize
		if end > len(payload) {
			end = len(payload)
		}
		for offset < end {
			written, err := connection.Write(payload[offset:end])
			if err != nil {
				return fmt.Errorf("write tunnel fixture bytes %d-%d: %w", offset, end, err)
			}
			if written == 0 {
				return io.ErrNoProgress
			}
			offset += written
		}
	}

	echoed := make([]byte, len(payload))
	if _, err := io.ReadFull(connection, echoed); err != nil {
		return fmt.Errorf("read %d-byte tunnel fixture echo: %w", len(payload), err)
	}
	if !bytes.Equal(echoed, payload) {
		return fmt.Errorf("tunnel fixture echo mismatch: got %d bytes, want exact %d-byte payload", len(echoed), len(payload))
	}
	return nil
}

// tunnelFullDuplexEchoOnConn writes and reads an independent, deterministic
// stream concurrently. Unlike a write-then-read echo check, this fills both
// tunnel directions at once and applies enough backpressure to catch pacing,
// shared-loop stalls, truncation, and ordering regressions under sustained load.
func tunnelFullDuplexEchoOnConn(ctx context.Context, connection net.Conn, label string, payloadBytes int, minimumBytesPerSecond int64) (time.Duration, error) {
	if payloadBytes <= 0 {
		return 0, errors.New("full-duplex payload size must be positive")
	}
	if minimumBytesPerSecond <= 0 {
		return 0, errors.New("full-duplex minimum throughput must be positive")
	}
	deadline := time.Now().Add(2 * time.Minute)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err := connection.SetDeadline(deadline); err != nil {
		return 0, fmt.Errorf("set full-duplex deadline: %w", err)
	}
	payload := deterministicTunnelPayload(label, payloadBytes)
	echoed := make([]byte, len(payload))
	writeDone := make(chan error, 1)
	started := time.Now()
	go func() {
		writeDone <- writeTunnelChunks(connection, payload, tunnelFullDuplexChunks)
	}()
	readBytes, readErr := io.ReadFull(connection, echoed)
	writeErr := <-writeDone
	elapsed := time.Since(started)
	if err := errors.Join(
		wrapIfError("write full-duplex stream", writeErr),
		wrapIfError(fmt.Sprintf("read full-duplex echo (%d/%d bytes)", readBytes, len(echoed)), readErr),
	); err != nil {
		return elapsed, err
	}
	if !bytes.Equal(echoed, payload) {
		wantDigest := sha256.Sum256(payload)
		gotDigest := sha256.Sum256(echoed)
		return elapsed, fmt.Errorf("full-duplex echo mismatch: got sha256=%x, want sha256=%x", gotDigest, wantDigest)
	}
	bytesPerSecond := int64(float64(payloadBytes) / elapsed.Seconds())
	if bytesPerSecond < minimumBytesPerSecond {
		return elapsed, fmt.Errorf(
			"full-duplex one-way throughput %d bytes/s is below %d bytes/s (%d bytes in %s)",
			bytesPerSecond,
			minimumBytesPerSecond,
			payloadBytes,
			elapsed.Round(time.Millisecond),
		)
	}
	return elapsed, nil
}

func writeTunnelChunks(writer io.Writer, payload []byte, chunks []int) error {
	if len(chunks) == 0 {
		return errors.New("tunnel write chunks must not be empty")
	}
	for offset, chunkIndex := 0, 0; offset < len(payload); chunkIndex++ {
		chunkSize := chunks[chunkIndex%len(chunks)]
		if chunkSize <= 0 {
			return fmt.Errorf("invalid tunnel write chunk size %d", chunkSize)
		}
		end := offset + chunkSize
		if end > len(payload) {
			end = len(payload)
		}
		for offset < end {
			written, err := writer.Write(payload[offset:end])
			if err != nil {
				return err
			}
			if written == 0 {
				return io.ErrNoProgress
			}
			offset += written
		}
	}
	return nil
}

type deterministicHTTPServer struct {
	listener net.Listener
	server   *http.Server
	done     chan error
	marker   string
	body     []byte
}

func startDeterministicHTTPServer(marker string) (*deterministicHTTPServer, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	bodySeed := []byte("sliver-e2e-http/" + marker + "/0123456789abcdef\n")
	body := bytes.Repeat(bodySeed, ((64*1024+17)/len(bodySeed))+1)
	body = body[:64*1024+17]
	body[len(body)-1] = 'Z'

	fixture := &deterministicHTTPServer{
		listener: listener,
		done:     make(chan error, 1),
		marker:   marker,
		body:     body,
	}
	mux := http.NewServeMux()
	mux.HandleFunc(deterministicHTTPPath, func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			response.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		response.Header().Set("Content-Type", "application/octet-stream")
		response.Header().Set("X-Sliver-E2E-Marker", marker)
		response.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write(body)
	})
	fixture.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		serveErr := fixture.server.Serve(listener)
		if errors.Is(serveErr, http.ErrServerClosed) {
			serveErr = nil
		}
		fixture.done <- serveErr
	}()
	return fixture, nil
}

func (fixture *deterministicHTTPServer) address() string {
	return fixture.listener.Addr().String()
}

func (fixture *deterministicHTTPServer) url(address string) string {
	return "http://" + address + deterministicHTTPPath
}

func (fixture *deterministicHTTPServer) close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	shutdownErr := fixture.server.Shutdown(ctx)
	serveErr := <-fixture.done
	return errors.Join(shutdownErr, serveErr)
}

func requestDeterministicHTTP(ctx context.Context, address string, fixture *deterministicHTTPServer) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fixture.url(address), nil)
	if err != nil {
		return err
	}
	transport := &http.Transport{
		Proxy:             nil,
		DisableKeepAlives: true,
	}
	defer transport.CloseIdleConnections()
	response, err := (&http.Client{Transport: transport}).Do(request)
	if err != nil {
		return fmt.Errorf("GET deterministic HTTP fixture through %s: %w", address, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("deterministic HTTP status = %s, want %s", response.Status, http.StatusText(http.StatusOK))
	}
	if got := response.Header.Get("X-Sliver-E2E-Marker"); got != fixture.marker {
		return fmt.Errorf("deterministic HTTP marker = %q, want %q", got, fixture.marker)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read deterministic HTTP body: %w", err)
	}
	if !bytes.Equal(body, fixture.body) {
		return fmt.Errorf("deterministic HTTP body mismatch: got %d bytes, want exact %d-byte body", len(body), len(fixture.body))
	}
	return nil
}
