package c2

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"net"
	"runtime"
	"testing"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/minisign"
)

type readSizeRecorder struct {
	reader  io.Reader
	maxRead int
}

func (r *readSizeRecorder) Read(p []byte) (int, error) {
	if len(p) > r.maxRead {
		r.maxRead = len(p)
	}
	return r.reader.Read(p)
}

func envelopeFrameWithDeclaredLength(dataLength uint32, payload []byte) []byte {
	frame := make([]byte, minisign.RawSigSize+4+len(payload))
	binary.LittleEndian.PutUint16(frame[:2], minisign.EdDSA)
	binary.LittleEndian.PutUint32(frame[minisign.RawSigSize:minisign.RawSigSize+4], dataLength)
	copy(frame[minisign.RawSigSize+4:], payload)
	return frame
}

func measureEnvelopeReadAlloc(t *testing.T, readFunc func(net.Conn) (*sliverpb.Envelope, error), frame []byte) (uint64, error) {
	t.Helper()

	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()

	errCh := make(chan error, 1)
	go func() {
		_, err := readFunc(srv)
		errCh <- err
	}()

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	go func() {
		_, _ = cli.Write(frame)
		_ = cli.Close()
	}()

	err := <-errCh

	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	if after.TotalAlloc < before.TotalAlloc {
		return 0, err
	}
	return after.TotalAlloc - before.TotalAlloc, err
}

func assertLargeTruncatedFrameHasBoundedAlloc(t *testing.T, readFunc func(net.Conn) (*sliverpb.Envelope, error)) {
	t.Helper()

	declaredLength := uint32(socketEnvelopeDiskSpoolThreshold + (8 * 1024 * 1024))
	frame := envelopeFrameWithDeclaredLength(declaredLength, []byte{0x01})

	minAllocDelta := uint64(math.MaxUint64)
	for i := 0; i < 3; i++ {
		allocDelta, err := measureEnvelopeReadAlloc(t, readFunc, frame)
		if err == nil {
			t.Fatalf("expected read failure for truncated frame")
		}
		if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("expected EOF-like error for truncated frame, got %v", err)
		}
		if allocDelta < minAllocDelta {
			minAllocDelta = allocDelta
		}
	}

	const maxAllowedAlloc = 24 * 1024 * 1024
	if minAllocDelta > maxAllowedAlloc {
		t.Fatalf("expected bounded allocation <= %d bytes, got %d", maxAllowedAlloc, minAllocDelta)
	}
}

func TestReadSocketEnvelopeDataKeepsSmallPayloadsInMemory(t *testing.T) {
	payload := bytes.Repeat([]byte{0x7a}, 4096)
	recorder := &readSizeRecorder{reader: bytes.NewReader(payload)}

	got, err := readSocketEnvelopeData(recorder, len(payload), 8192)
	if err != nil {
		t.Fatalf("readSocketEnvelopeData returned err: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("unexpected payload contents")
	}
	if recorder.maxRead != len(payload) {
		t.Fatalf("expected in-memory read size %d, got %d", len(payload), recorder.maxRead)
	}
}

func TestReadSocketEnvelopeDataSpoolsLargePayloadsInChunks(t *testing.T) {
	payload := bytes.Repeat([]byte{0x42}, (2*socketEnvelopeReadChunkSize)+123)
	recorder := &readSizeRecorder{reader: bytes.NewReader(payload)}

	got, err := readSocketEnvelopeData(recorder, len(payload), socketEnvelopeReadChunkSize/2)
	if err != nil {
		t.Fatalf("readSocketEnvelopeData returned err: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("unexpected payload contents")
	}
	if recorder.maxRead > socketEnvelopeReadChunkSize {
		t.Fatalf("expected max read chunk <= %d, got %d", socketEnvelopeReadChunkSize, recorder.maxRead)
	}
}

func TestMTLSSocketReadEnvelopeLargeTruncatedMessageBoundsAllocation(t *testing.T) {
	assertLargeTruncatedFrameHasBoundedAlloc(t, socketReadEnvelope)
}

func TestWGSocketReadEnvelopeLargeTruncatedMessageBoundsAllocation(t *testing.T) {
	assertLargeTruncatedFrameHasBoundedAlloc(t, socketWGReadEnvelope)
}
