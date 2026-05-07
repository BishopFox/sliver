package c2

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strings"
	"testing"

	implantWG "github.com/bishopfox/sliver/implant/sliver/transports/wireguard"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/bishopfox/sliver/util/minisign"
)

func TestWGSocketReadEnvelopeAcceptsValidSignature(t *testing.T) {
	clearMTLSImplantSigKeyCache(t)

	want := &sliverpb.Envelope{
		Type: sliverpb.MsgPing,
		Data: []byte("hello"),
	}
	frame, keyID, pub := newImplantSignedEnvelopeFrame(t, "peer-private-key-test", want)
	mtlsImplantSigKeyCache.Store(keyID, pub)

	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()

	go func() {
		_, _ = cli.Write(frame)
		_ = cli.Close()
	}()

	got, err := socketWGReadEnvelope(srv)
	if err != nil {
		t.Fatalf("expected valid signature, got err=%v", err)
	}
	if got.Type != want.Type || !bytes.Equal(got.Data, want.Data) {
		t.Fatalf("unexpected envelope: got=%+v want=%+v", got, want)
	}
}

func TestWGSocketReadEnvelopeRejectsUnsupportedAlgorithm(t *testing.T) {
	clearMTLSImplantSigKeyCache(t)

	frame, keyID, pub := newImplantSignedEnvelopeFrame(t, "peer-private-key-test", &sliverpb.Envelope{
		Type: sliverpb.MsgPing,
		Data: []byte("hello"),
	})
	mtlsImplantSigKeyCache.Store(keyID, pub)

	binary.LittleEndian.PutUint16(frame[:2], minisign.HashEdDSA)

	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()

	errCh := make(chan error, 1)
	go func() {
		_, err := socketWGReadEnvelope(srv)
		errCh <- err
	}()
	go func() {
		_, _ = cli.Write(frame)
		_ = cli.Close()
	}()

	err := <-errCh
	if err == nil || !strings.Contains(err.Error(), "unsupported signature algorithm") {
		t.Fatalf("expected unsupported signature algorithm error, got=%v", err)
	}
}

func TestWGSocketReadEnvelopeRejectsInvalidSignature(t *testing.T) {
	clearMTLSImplantSigKeyCache(t)

	frame, keyID, pub := newImplantSignedEnvelopeFrame(t, "peer-private-key-test", &sliverpb.Envelope{
		Type: sliverpb.MsgPing,
		Data: []byte("hello"),
	})
	mtlsImplantSigKeyCache.Store(keyID, pub)

	frame[10] ^= 0xff

	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()

	errCh := make(chan error, 1)
	go func() {
		_, err := socketWGReadEnvelope(srv)
		errCh <- err
	}()
	go func() {
		_, _ = cli.Write(frame)
		_ = cli.Close()
	}()

	err := <-errCh
	if err == nil || !strings.Contains(err.Error(), "invalid signature") {
		t.Fatalf("expected invalid signature error, got=%v", err)
	}
}

func TestWGSocketReadEnvelopeRejectsTamperedMessage(t *testing.T) {
	clearMTLSImplantSigKeyCache(t)

	frame, keyID, pub := newImplantSignedEnvelopeFrame(t, "peer-private-key-test", &sliverpb.Envelope{
		Type: sliverpb.MsgPing,
		Data: []byte("hello"),
	})
	mtlsImplantSigKeyCache.Store(keyID, pub)

	dataOffset := minisign.RawSigSize + 4
	frame[dataOffset] ^= 0x01

	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()

	errCh := make(chan error, 1)
	go func() {
		_, err := socketWGReadEnvelope(srv)
		errCh <- err
	}()
	go func() {
		_, _ = cli.Write(frame)
		_ = cli.Close()
	}()

	err := <-errCh
	if err == nil || !strings.Contains(err.Error(), "invalid signature") {
		t.Fatalf("expected invalid signature error, got=%v", err)
	}
}

func TestWGSocketReadEnvelopeRejectsTruncatedSignature(t *testing.T) {
	clearMTLSImplantSigKeyCache(t)

	frame, _, _ := newImplantSignedEnvelopeFrame(t, "peer-private-key-test", &sliverpb.Envelope{
		Type: sliverpb.MsgPing,
		Data: []byte("hello"),
	})

	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()

	errCh := make(chan error, 1)
	go func() {
		_, err := socketWGReadEnvelope(srv)
		errCh <- err
	}()
	go func() {
		_, _ = cli.Write(frame[:minisign.RawSigSize-1])
		_ = cli.Close()
	}()

	err := <-errCh
	if err == nil {
		t.Fatalf("expected error for truncated signature, got nil")
	}
	if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("expected EOF for truncated signature, got=%v", err)
	}
}

func TestWireGuardImplantReadEnvelopeAcceptsValidServerSignature(t *testing.T) {
	want := &sliverpb.Envelope{
		Type: sliverpb.MsgPing,
		Data: []byte("hello"),
	}
	frame := newServerSignedEnvelopeFrame(t, want)

	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()

	go func() {
		_, _ = cli.Write(frame)
		_ = cli.Close()
	}()

	got, err := implantWG.ReadEnvelope(srv)
	if err != nil {
		t.Fatalf("expected valid signature, got err=%v", err)
	}
	if got.Type != want.Type || !bytes.Equal(got.Data, want.Data) {
		t.Fatalf("unexpected envelope: got=%+v want=%+v", got, want)
	}
}

func TestWireGuardImplantReadEnvelopeRejectsInvalidServerSignature(t *testing.T) {
	frame := newServerSignedEnvelopeFrame(t, &sliverpb.Envelope{
		Type: sliverpb.MsgPing,
		Data: []byte("hello"),
	})
	frame[10] ^= 0xff

	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()

	go func() {
		_, _ = cli.Write(frame)
		_ = cli.Close()
	}()

	_, err := implantWG.ReadEnvelope(srv)
	if err == nil || !strings.Contains(err.Error(), "invalid signature") {
		t.Fatalf("expected invalid signature error, got=%v", err)
	}
}

func TestWireGuardImplantReadEnvelopeRejectsWrongServerKeyID(t *testing.T) {
	frame := newServerSignedEnvelopeFrame(t, &sliverpb.Envelope{
		Type: sliverpb.MsgPing,
		Data: []byte("hello"),
	})
	frame[2] ^= 0xff

	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()

	go func() {
		_, _ = cli.Write(frame)
		_ = cli.Close()
	}()

	_, err := implantWG.ReadEnvelope(srv)
	if err == nil || !strings.Contains(err.Error(), "invalid signature") {
		t.Fatalf("expected invalid signature error, got=%v", err)
	}
}

func TestWireGuardImplantReadEnvelopeRejectsAlgorithmDowngradeTrick(t *testing.T) {
	frame := newServerSignedEnvelopeFrame(t, &sliverpb.Envelope{
		Type: sliverpb.MsgPing,
		Data: []byte("hello"),
	})
	binary.LittleEndian.PutUint16(frame[:2], minisign.HashEdDSA)

	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()

	go func() {
		_, _ = cli.Write(frame)
		_ = cli.Close()
	}()

	_, err := implantWG.ReadEnvelope(srv)
	if err == nil || !strings.Contains(err.Error(), "invalid signature") {
		t.Fatalf("expected invalid signature error, got=%v", err)
	}
}

func TestWireGuardImplantReadEnvelopeRejectsTamperedServerMessage(t *testing.T) {
	frame := newServerSignedEnvelopeFrame(t, &sliverpb.Envelope{
		Type: sliverpb.MsgPing,
		Data: []byte("hello"),
	})
	dataOffset := minisign.RawSigSize + 4
	frame[dataOffset] ^= 0x01

	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()

	go func() {
		_, _ = cli.Write(frame)
		_ = cli.Close()
	}()

	_, err := implantWG.ReadEnvelope(srv)
	if err == nil || !strings.Contains(err.Error(), "invalid signature") {
		t.Fatalf("expected invalid signature error, got=%v", err)
	}
}

func TestWireGuardImplantReadEnvelopeRejectsTruncatedServerSignature(t *testing.T) {
	frame := newServerSignedEnvelopeFrame(t, &sliverpb.Envelope{
		Type: sliverpb.MsgPing,
		Data: []byte("hello"),
	})

	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()

	go func() {
		_, _ = cli.Write(frame[:minisign.RawSigSize-1])
		_ = cli.Close()
	}()

	_, err := implantWG.ReadEnvelope(srv)
	if err == nil {
		t.Fatalf("expected error for truncated signature, got nil")
	}
	if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("expected EOF for truncated signature, got=%v", err)
	}
}
