//go:build darwin || linux

package handlers

import (
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

func TestUnixPermissionHandlersRegistered(t *testing.T) {
	handlers := GetSystemHandlers()
	for _, messageType := range []uint32{sliverpb.MsgChmodReq, sliverpb.MsgChownReq} {
		if _, ok := handlers[messageType]; !ok {
			t.Fatalf("message type %d is not registered", messageType)
		}
	}
}

func TestChmodHandlerRecursive(t *testing.T) {
	target := filepath.Join(t.TempDir(), "target")
	nested := filepath.Join(target, "nested")
	file := filepath.Join(nested, "fixture.txt")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}

	request, err := proto.Marshal(&sliverpb.ChmodReq{Path: target, FileMode: "0750", Recursive: true})
	if err != nil {
		t.Fatal(err)
	}

	called := false
	chmodHandler(request, func(data []byte, handlerErr error) {
		called = true
		if handlerErr != nil {
			t.Fatalf("chmodHandler callback error: %v", handlerErr)
		}
		response := &sliverpb.Chmod{}
		if err := proto.Unmarshal(data, response); err != nil {
			t.Fatalf("decode chmod response: %v", err)
		}
		if response.GetResponse().GetErr() != "" {
			t.Fatalf("chmodHandler response error: %s", response.GetResponse().GetErr())
		}
		if response.Path != target {
			t.Fatalf("chmodHandler path = %q, want %q", response.Path, target)
		}
	})
	if !called {
		t.Fatal("chmodHandler did not invoke its response callback")
	}

	for _, path := range []string{target, nested, file} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o750 {
			t.Errorf("%s mode = %o, want 750", path, got)
		}
	}
}

func TestChownHandlerRecursiveCurrentOwner(t *testing.T) {
	current, group, wantUID, wantGID := currentOwner(t)

	target := filepath.Join(t.TempDir(), "target")
	file := filepath.Join(target, "fixture.txt")
	if err := os.MkdirAll(target, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}

	request, err := proto.Marshal(&sliverpb.ChownReq{
		Path: target, Uid: current.Username, Gid: group.Name, Recursive: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	called := false
	chownHandler(request, func(data []byte, handlerErr error) {
		called = true
		if handlerErr != nil {
			t.Fatalf("chownHandler callback error: %v", handlerErr)
		}
		response := &sliverpb.Chown{}
		if err := proto.Unmarshal(data, response); err != nil {
			t.Fatalf("decode chown response: %v", err)
		}
		if response.GetResponse().GetErr() != "" {
			t.Fatalf("chownHandler response error: %s", response.GetResponse().GetErr())
		}
		if response.Path != target {
			t.Fatalf("chownHandler path = %q, want %q", response.Path, target)
		}
	})
	if !called {
		t.Fatal("chownHandler did not invoke its response callback")
	}

	assertOwner(t, []string{target, file}, wantUID, wantGID)
}

func currentOwner(t *testing.T) (*user.User, *user.Group, uint64, uint64) {
	t.Helper()
	current, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	group, err := user.LookupGroupId(current.Gid)
	if err != nil {
		t.Fatal(err)
	}
	wantUID, err := strconv.ParseUint(current.Uid, 10, 32)
	if err != nil {
		t.Fatal(err)
	}
	wantGID, err := strconv.ParseUint(group.Gid, 10, 32)
	if err != nil {
		t.Fatal(err)
	}
	return current, group, wantUID, wantGID
}

func assertOwner(t *testing.T, paths []string, wantUID uint64, wantGID uint64) {
	t.Helper()
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			t.Fatalf("%s stat type = %T, want *syscall.Stat_t", path, info.Sys())
		}
		if uint64(stat.Uid) != wantUID || uint64(stat.Gid) != wantGID {
			t.Errorf("%s owner = %d:%d, want %d:%d", path, stat.Uid, stat.Gid, wantUID, wantGID)
		}
	}
}
