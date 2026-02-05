package c2

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strings"
	"testing"

	implantMTLS "github.com/bishopfox/sliver/implant/sliver/transports/mtls"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	serverCrypto "github.com/bishopfox/sliver/server/cryptography"
	"github.com/bishopfox/sliver/util/minisign"
	"golang.org/x/crypto/blake2b"
	"google.golang.org/protobuf/proto"
)

func clearMTLSImplantSigKeyCache(t *testing.T) {
	t.Helper()
	mtlsImplantSigKeyCache.Range(func(key, _ any) bool {
		mtlsImplantSigKeyCache.Delete(key)
		return true
	})
}

func newImplantSignedEnvelopeFrame(t *testing.T, peerPrivateKey string, envelope *sliverpb.Envelope) ([]byte, uint64, ed25519.PublicKey) {
	t.Helper()

	data, err := proto.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	seed := sha256.Sum256([]byte(mtlsEnvelopeSigningSeedPrefix + peerPrivateKey))
	priv := ed25519.NewKeyFromSeed(seed[:])
	pub := priv.Public().(ed25519.PublicKey)

	digest := blake2b.Sum256(pub)
	keyID := binary.LittleEndian.Uint64(digest[:8])

	rawSigBuf := make([]byte, minisign.RawSigSize)
	binary.LittleEndian.PutUint16(rawSigBuf[:2], minisign.EdDSA)
	binary.LittleEndian.PutUint64(rawSigBuf[2:10], keyID)
	copy(rawSigBuf[10:], ed25519.Sign(priv, data))

	frame := make([]byte, 0, minisign.RawSigSize+4+len(data))
	frame = append(frame, rawSigBuf...)

	dataLengthBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(dataLengthBuf, uint32(len(data)))
	frame = append(frame, dataLengthBuf...)
	frame = append(frame, data...)

	return frame, keyID, pub
}

func newServerSignedEnvelopeFrame(t *testing.T, envelope *sliverpb.Envelope) []byte {
	t.Helper()

	data, err := proto.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	rawSig := minisign.SignRawBuf(*serverCrypto.MinisignServerPrivateKey(), data)

	frame := make([]byte, 0, minisign.RawSigSize+4+len(data))
	frame = append(frame, rawSig[:]...)

	dataLengthBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(dataLengthBuf, uint32(len(data)))
	frame = append(frame, dataLengthBuf...)
	frame = append(frame, data...)

	return frame
}

func TestMTLSSocketReadEnvelopeRejectsUnsupportedAlgorithm(t *testing.T) {
	clearMTLSImplantSigKeyCache(t)

	frame, keyID, pub := newImplantSignedEnvelopeFrame(t, "peer-private-key-test", &sliverpb.Envelope{
		Type: sliverpb.MsgPing,
		Data: []byte("hello"),
	})
	mtlsImplantSigKeyCache.Store(keyID, pub)

	// Flip to HashEdDSA (or any non-EdDSA value), should be rejected.
	binary.LittleEndian.PutUint16(frame[:2], minisign.HashEdDSA)

	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()

	errCh := make(chan error, 1)
	go func() {
		_, err := socketReadEnvelope(srv)
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

func TestMTLSSocketReadEnvelopeAcceptsValidSignature(t *testing.T) {
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

	got, err := socketReadEnvelope(srv)
	if err != nil {
		t.Fatalf("expected valid signature, got err=%v", err)
	}
	if got.Type != want.Type || !bytes.Equal(got.Data, want.Data) {
		t.Fatalf("unexpected envelope: got=%+v want=%+v", got, want)
	}
}

func TestMTLSSocketReadEnvelopeRejectsInvalidSignature(t *testing.T) {
	clearMTLSImplantSigKeyCache(t)

	frame, keyID, pub := newImplantSignedEnvelopeFrame(t, "peer-private-key-test", &sliverpb.Envelope{
		Type: sliverpb.MsgPing,
		Data: []byte("hello"),
	})
	mtlsImplantSigKeyCache.Store(keyID, pub)

	// Corrupt a signature byte.
	frame[10] ^= 0xff

	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()

	errCh := make(chan error, 1)
	go func() {
		_, err := socketReadEnvelope(srv)
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

func TestMTLSSocketReadEnvelopeRejectsTamperedMessage(t *testing.T) {
	clearMTLSImplantSigKeyCache(t)

	frame, keyID, pub := newImplantSignedEnvelopeFrame(t, "peer-private-key-test", &sliverpb.Envelope{
		Type: sliverpb.MsgPing,
		Data: []byte("hello"),
	})
	mtlsImplantSigKeyCache.Store(keyID, pub)

	// Flip a byte in the protobuf payload after signing.
	dataOffset := minisign.RawSigSize + 4
	frame[dataOffset] ^= 0x01

	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()

	errCh := make(chan error, 1)
	go func() {
		_, err := socketReadEnvelope(srv)
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

func TestMTLSSocketReadEnvelopeRejectsWrongKeyID(t *testing.T) {
	clearMTLSImplantSigKeyCache(t)

	frame, keyID, _ := newImplantSignedEnvelopeFrame(t, "peer-private-key-test", &sliverpb.Envelope{
		Type: sliverpb.MsgPing,
		Data: []byte("hello"),
	})

	// Swap the key ID to a different cached public key (signature won't match).
	wrongPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ed25519 key: %v", err)
	}
	wrongKeyID := keyID + 1
	mtlsImplantSigKeyCache.Store(wrongKeyID, wrongPub)
	binary.LittleEndian.PutUint64(frame[2:10], wrongKeyID)

	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()

	errCh := make(chan error, 1)
	go func() {
		_, err := socketReadEnvelope(srv)
		errCh <- err
	}()
	go func() {
		_, _ = cli.Write(frame)
		_ = cli.Close()
	}()

	err = <-errCh
	if err == nil || !strings.Contains(err.Error(), "invalid signature") {
		t.Fatalf("expected invalid signature error, got=%v", err)
	}
}

func TestMTLSSocketReadEnvelopeRejectsTruncatedSignature(t *testing.T) {
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
		_, err := socketReadEnvelope(srv)
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

func TestMTLSImplantReadEnvelopeRejectsInvalidServerSignature(t *testing.T) {
	frame := newServerSignedEnvelopeFrame(t, &sliverpb.Envelope{
		Type: sliverpb.MsgPing,
		Data: []byte("hello"),
	})

	// Corrupt a signature byte.
	frame[10] ^= 0xff

	_, err := implantMTLS.ReadEnvelope(bytes.NewReader(frame))
	if err == nil || !strings.Contains(err.Error(), "invalid signature") {
		t.Fatalf("expected invalid signature error, got=%v", err)
	}
}

func TestMTLSImplantReadEnvelopeAcceptsValidServerSignature(t *testing.T) {
	want := &sliverpb.Envelope{
		Type: sliverpb.MsgPing,
		Data: []byte("hello"),
	}
	frame := newServerSignedEnvelopeFrame(t, want)

	got, err := implantMTLS.ReadEnvelope(bytes.NewReader(frame))
	if err != nil {
		t.Fatalf("expected valid signature, got err=%v", err)
	}
	if got.Type != want.Type || !bytes.Equal(got.Data, want.Data) {
		t.Fatalf("unexpected envelope: got=%+v want=%+v", got, want)
	}
}

func TestMTLSImplantReadEnvelopeRejectsWrongServerKeyID(t *testing.T) {
	frame := newServerSignedEnvelopeFrame(t, &sliverpb.Envelope{
		Type: sliverpb.MsgPing,
		Data: []byte("hello"),
	})

	// Flip key ID bytes.
	frame[2] ^= 0xff

	_, err := implantMTLS.ReadEnvelope(bytes.NewReader(frame))
	if err == nil || !strings.Contains(err.Error(), "invalid signature") {
		t.Fatalf("expected invalid signature error, got=%v", err)
	}
}

func TestMTLSImplantReadEnvelopeRejectsAlgorithmDowngradeTrick(t *testing.T) {
	frame := newServerSignedEnvelopeFrame(t, &sliverpb.Envelope{
		Type: sliverpb.MsgPing,
		Data: []byte("hello"),
	})

	// Change algorithm to HashEdDSA without changing the signature; verification must fail.
	binary.LittleEndian.PutUint16(frame[:2], minisign.HashEdDSA)

	_, err := implantMTLS.ReadEnvelope(bytes.NewReader(frame))
	if err == nil || !strings.Contains(err.Error(), "invalid signature") {
		t.Fatalf("expected invalid signature error, got=%v", err)
	}
}

func TestMTLSImplantReadEnvelopeRejectsTamperedServerMessage(t *testing.T) {
	frame := newServerSignedEnvelopeFrame(t, &sliverpb.Envelope{
		Type: sliverpb.MsgPing,
		Data: []byte("hello"),
	})

	// Flip a byte in the protobuf payload after signing.
	dataOffset := minisign.RawSigSize + 4
	frame[dataOffset] ^= 0x01

	_, err := implantMTLS.ReadEnvelope(bytes.NewReader(frame))
	if err == nil || !strings.Contains(err.Error(), "invalid signature") {
		t.Fatalf("expected invalid signature error, got=%v", err)
	}
}

func TestMTLSImplantReadEnvelopeRejectsTruncatedServerSignature(t *testing.T) {
	frame := newServerSignedEnvelopeFrame(t, &sliverpb.Envelope{
		Type: sliverpb.MsgPing,
		Data: []byte("hello"),
	})

	_, err := implantMTLS.ReadEnvelope(bytes.NewReader(frame[:minisign.RawSigSize-1]))
	if err == nil {
		t.Fatalf("expected error for truncated signature, got nil")
	}
	if !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("expected EOF for truncated signature, got=%v", err)
	}
}
