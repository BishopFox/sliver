package handlers

import (
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
