package handlers

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"golang.org/x/sys/unix"
	"google.golang.org/protobuf/proto"
)

func TestMemfilesAddAndRemoveLifecycle(t *testing.T) {
	var addResponse *sliverpb.MemfilesAdd
	memfilesAddHandler(nil, func(data []byte, handlerErr error) {
		if handlerErr != nil {
			t.Fatalf("memfilesAddHandler callback error: %v", handlerErr)
		}
		addResponse = &sliverpb.MemfilesAdd{}
		if err := proto.Unmarshal(data, addResponse); err != nil {
			t.Fatalf("decode add response: %v", err)
		}
	})
	if addResponse == nil {
		t.Fatal("memfilesAddHandler did not invoke its response callback")
	}
	if errText := addResponse.GetResponse().GetErr(); errText != "" {
		t.Fatalf("memfilesAddHandler returned error: %s", errText)
	}
	if addResponse.Fd < 3 {
		t.Fatalf("memfilesAddHandler returned invalid fd %d", addResponse.Fd)
	}
	fd := int(addResponse.Fd)
	defer unix.Close(fd)

	fdPath := fmt.Sprintf("/proc/%d/fd/%d", os.Getpid(), fd)
	link, err := os.Readlink(fdPath)
	if err != nil {
		t.Fatalf("read memfd link: %v", err)
	}
	if !strings.Contains(link, "/memfd:") {
		t.Fatalf("fd link %q does not identify a memfd", link)
	}

	request, err := proto.Marshal(&sliverpb.MemfilesRmReq{Fd: addResponse.Fd})
	if err != nil {
		t.Fatal(err)
	}
	var removeResponse *sliverpb.MemfilesRm
	memfilesRmHandler(request, func(data []byte, handlerErr error) {
		if handlerErr != nil {
			t.Fatalf("memfilesRmHandler callback error: %v", handlerErr)
		}
		removeResponse = &sliverpb.MemfilesRm{}
		if err := proto.Unmarshal(data, removeResponse); err != nil {
			t.Fatalf("decode remove response: %v", err)
		}
	})
	if removeResponse == nil {
		t.Fatal("memfilesRmHandler did not invoke its response callback")
	}
	if errText := removeResponse.GetResponse().GetErr(); errText != "" {
		t.Fatalf("memfilesRmHandler returned error: %s", errText)
	}
	if _, err := os.Readlink(fdPath); !os.IsNotExist(err) {
		t.Fatalf("memfd still accessible after removal: %v", err)
	}
}
