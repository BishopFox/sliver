package tunnel_handlers

import (
	"bytes"
	"compress/gzip"
	"testing"
)

func TestDecodeWasmGzip(t *testing.T) {
	// An empty WebAssembly v1 module has the magic and version headers.
	wasm := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(wasm); err != nil {
		t.Fatalf("compress WASM: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("finish gzip stream: %v", err)
	}

	got, err := decodeWasmGzip(compressed.Bytes())
	if err != nil {
		t.Fatalf("decode valid WASM payload: %v", err)
	}
	if !bytes.Equal(got, wasm) {
		t.Fatalf("decoded payload = %x, want %x", got, wasm)
	}

	if _, err := decodeWasmGzip([]byte("not a gzip stream")); err == nil {
		t.Fatal("expected malformed gzip to fail")
	}
}
