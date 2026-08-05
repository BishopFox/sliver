//go:build windows || darwin || linux

// Package wireguard implements the WireGuard transport for Sliver implants.
package wireguard

import (
	"encoding/binary"
	"errors"
	"net"
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

func readEnvelopeWithTimeout(t *testing.T, conn net.Conn) error {
	t.Helper()
	ch := make(chan error, 1)
	go func() {
		_, err := ReadEnvelope(conn)
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
	client, server := net.Pipe()
	defer func() { _ = client.Close() }()
	defer func() { _ = server.Close() }()

	go func() {
		_, _ = client.Write(rawSigStub())
		_, _ = client.Write(lengthPrefix(uint32(maxEnvelopeLength) + 1))
	}()

	err := readEnvelopeWithTimeout(t, server)
	if !errors.Is(err, errEnvelopeTooLarge) {
		t.Fatalf("expected errEnvelopeTooLarge, got %v", err)
	}
}

func TestReadEnvelopeRejectsZeroLength(t *testing.T) {
	client, server := net.Pipe()
	defer func() { _ = client.Close() }()
	defer func() { _ = server.Close() }()

	go func() {
		_, _ = client.Write(rawSigStub())
		_, _ = client.Write(lengthPrefix(0))
	}()

	err := readEnvelopeWithTimeout(t, server)
	if err == nil {
		t.Fatal("expected error for zero-length envelope")
	}
}
