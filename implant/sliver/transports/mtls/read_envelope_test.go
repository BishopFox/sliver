package mtls

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
	"time"

	"github.com/bishopfox/sliver/implant/sliver/cryptography"
)

func rawSigStub() []byte {
	return make([]byte, cryptography.RawSigSize)
}

func lengthPrefix(n uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, n)
	return b
}

func readEnvelopeWithTimeout(t *testing.T, buf *bytes.Reader) error {
	t.Helper()
	ch := make(chan error, 1)
	go func() {
		_, err := ReadEnvelope(buf)
		ch <- err
	}()
	select {
	case err := <-ch:
		return err
	case <-time.After(3 * time.Second):
		t.Fatal("ReadEnvelope did not return in time (possible unbounded allocation)")
		return nil
	}
}

func TestReadEnvelopeRejectsOversizedFrame(t *testing.T) {
	var frame []byte
	frame = append(frame, rawSigStub()...)
	frame = append(frame, lengthPrefix(uint32(maxEnvelopeLength)+1)...)
	buf := bytes.NewReader(frame)

	err := readEnvelopeWithTimeout(t, buf)
	if !errors.Is(err, errEnvelopeTooLarge) {
		t.Fatalf("expected errEnvelopeTooLarge, got %v", err)
	}
}

func TestReadEnvelopeRejectsZeroLength(t *testing.T) {
	var frame []byte
	frame = append(frame, rawSigStub()...)
	frame = append(frame, lengthPrefix(0)...)
	buf := bytes.NewReader(frame)

	err := readEnvelopeWithTimeout(t, buf)
	if err == nil {
		t.Fatal("expected error for zero-length envelope")
	}
}
