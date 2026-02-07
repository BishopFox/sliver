package mtls

import (
	"bytes"
	"crypto/tls"
	"testing"

	pb "github.com/bishopfox/sliver/protobuf/sliverpb"
)

func TestWriteEnvelope_NilEnvelope(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	if err := WriteEnvelope(&buf, nil); err == nil {
		t.Fatalf("expected error for nil envelope")
	}
}

func TestWriteEnvelope_NilWriter(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("WriteEnvelope panicked: %v", r)
		}
	}()

	if err := WriteEnvelope(nil, &pb.Envelope{}); err == nil {
		t.Fatalf("expected error for nil writer")
	}
}

func TestWriteEnvelope_NilTLSConnWriter(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("WriteEnvelope panicked: %v", r)
		}
	}()

	var conn *tls.Conn
	if err := WriteEnvelope(conn, &pb.Envelope{}); err == nil {
		t.Fatalf("expected error for nil *tls.Conn writer")
	}
}
