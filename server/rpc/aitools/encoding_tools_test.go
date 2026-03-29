package aitools

import (
	"context"
	stdbase32 "encoding/base32"
	stdbase64 "encoding/base64"
	stdhex "encoding/hex"
	"encoding/json"
	"testing"
)

func TestValueEncodingToolsRoundTrip(t *testing.T) {
	exec := &executor{}
	ctx := context.Background()

	cases := []struct {
		encodeTool string
		decodeTool string
		expected   string
		variant    string
	}{
		{encodeTool: "base32_encode", decodeTool: "base32_decode", expected: stdbase32.StdEncoding.EncodeToString([]byte("hello world")), variant: "standard"},
		{encodeTool: "base64_encode", decodeTool: "base64_decode", expected: stdbase64.StdEncoding.EncodeToString([]byte("hello world")), variant: "standard"},
		{encodeTool: "hex_encode", decodeTool: "hex_decode", expected: stdhex.EncodeToString([]byte("hello world")), variant: ""},
	}

	for _, tc := range cases {
		t.Run(tc.encodeTool, func(t *testing.T) {
			encodedRaw, err := exec.CallTool(ctx, tc.encodeTool, `{"text":"hello world"}`)
			if err != nil {
				t.Fatalf("encode with %s: %v", tc.encodeTool, err)
			}

			var encoded encodingValueResult
			if err := json.Unmarshal([]byte(encodedRaw), &encoded); err != nil {
				t.Fatalf("decode encode result: %v", err)
			}
			if encoded.Value == "" {
				t.Fatalf("expected encoded value from %s", tc.encodeTool)
			}
			if encoded.Value != tc.expected {
				t.Fatalf("unexpected encoded value: got=%q want=%q", encoded.Value, tc.expected)
			}
			if encoded.Variant != tc.variant {
				t.Fatalf("unexpected variant: got=%q want=%q", encoded.Variant, tc.variant)
			}

			decodedRaw, err := exec.CallTool(ctx, tc.decodeTool, `{"value":"`+encoded.Value+`"}`)
			if err != nil {
				t.Fatalf("decode with %s: %v", tc.decodeTool, err)
			}

			var decoded encodingBytesResult
			if err := json.Unmarshal([]byte(decodedRaw), &decoded); err != nil {
				t.Fatalf("decode decode result: %v", err)
			}
			if decoded.Text != "hello world" {
				t.Fatalf("unexpected decoded text: %+v", decoded)
			}
			if decoded.Variant != tc.variant {
				t.Fatalf("unexpected decoded variant: got=%q want=%q", decoded.Variant, tc.variant)
			}
		})
	}
}

func TestGzipEncodingToolsRoundTrip(t *testing.T) {
	exec := &executor{}
	ctx := context.Background()

	encodedRaw, err := exec.CallTool(ctx, "gzip_encode", `{"text":"hello world"}`)
	if err != nil {
		t.Fatalf("gzip encode: %v", err)
	}

	var encoded encodingBytesResult
	if err := json.Unmarshal([]byte(encodedRaw), &encoded); err != nil {
		t.Fatalf("decode gzip encode result: %v", err)
	}
	if encoded.DataBase64 == "" {
		t.Fatalf("expected gzip output bytes, got %+v", encoded)
	}

	decodedRaw, err := exec.CallTool(ctx, "gzip_decode", `{"data_base64":"`+encoded.DataBase64+`"}`)
	if err != nil {
		t.Fatalf("gzip decode: %v", err)
	}

	var decoded encodingBytesResult
	if err := json.Unmarshal([]byte(decodedRaw), &decoded); err != nil {
		t.Fatalf("decode gzip decode result: %v", err)
	}
	if decoded.Text != "hello world" {
		t.Fatalf("unexpected gzip decode text: %+v", decoded)
	}
}

func TestValueDecodingToolsRespectMaxBytes(t *testing.T) {
	exec := &executor{}
	ctx := context.Background()

	encoded := stdbase64.StdEncoding.EncodeToString([]byte("abc"))
	if _, err := exec.CallTool(ctx, "gzip_decode", `{"data_base64":"`+encoded+`","max_bytes":2}`); err == nil {
		t.Fatal("expected gzip decode to reject invalid compressed data")
	}

	hexEncodedRaw, err := exec.CallTool(ctx, "hex_encode", `{"text":"abc"}`)
	if err != nil {
		t.Fatalf("hex encode: %v", err)
	}
	var hexEncoded encodingValueResult
	if err := json.Unmarshal([]byte(hexEncodedRaw), &hexEncoded); err != nil {
		t.Fatalf("decode hex encode result: %v", err)
	}

	if _, err := exec.CallTool(ctx, "hex_decode", `{"value":"`+hexEncoded.Value+`","max_bytes":2}`); err == nil {
		t.Fatal("expected hex decode max_bytes limit to be enforced")
	}
}
