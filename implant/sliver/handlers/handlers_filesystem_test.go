package handlers

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

func TestCdHandlerReportsChdirFailure(t *testing.T) {
	workingDirectory := t.TempDir()
	t.Chdir(workingDirectory)

	request, err := proto.Marshal(&sliverpb.CdReq{Path: filepath.Join(workingDirectory, "missing")})
	if err != nil {
		t.Fatal(err)
	}

	called := false
	cdHandler(request, func(data []byte, handlerErr error) {
		called = true
		if handlerErr != nil {
			t.Fatalf("cdHandler callback error: %v", handlerErr)
		}
		response := &sliverpb.Pwd{}
		if err := proto.Unmarshal(data, response); err != nil {
			t.Fatalf("decode cd response: %v", err)
		}
		if response.GetResponse().GetErr() == "" {
			t.Fatal("cdHandler returned success for a missing directory")
		}
		if response.Path != workingDirectory {
			t.Fatalf("working directory changed: got %q, want %q", response.Path, workingDirectory)
		}
	})
	if !called {
		t.Fatal("cdHandler did not invoke its response callback")
	}
}

func TestExtractFilesTruncatesOverwrittenFile(t *testing.T) {
	destination := t.TempDir()
	archive := func(content string) []byte {
		t.Helper()
		buffer := bytes.NewBuffer(nil)
		writer := tar.NewWriter(buffer)
		header := &tar.Header{Name: "bundle/item.txt", Mode: 0o600, Size: int64(len(content)), Typeflag: tar.TypeReg}
		if err := writer.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		return buffer.Bytes()
	}

	if written, skipped, err := extractFiles(archive("a deliberately long first payload\n"), destination, false); err != nil || written != 1 || skipped != 0 {
		t.Fatalf("initial extract: written=%d skipped=%d err=%v", written, skipped, err)
	}
	if written, skipped, err := extractFiles(archive("short\n"), destination, true); err != nil || written != 1 || skipped != 0 {
		t.Fatalf("overwrite extract: written=%d skipped=%d err=%v", written, skipped, err)
	}
	content, err := os.ReadFile(filepath.Join(destination, "bundle", "item.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "short\n" {
		t.Fatalf("overwritten content = %q, want exact shorter payload", content)
	}
}
