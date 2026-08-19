//go:build linux || darwin || windows

package handlers

import (
	"errors"
	"slices"
	"testing"

	"github.com/bishopfox/sliver/implant/sliver/extension"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

type failingExtension struct {
	id      string
	loadErr error
}

func (f *failingExtension) Load() error {
	return f.loadErr
}

func (f *failingExtension) Call(string, []byte, func([]byte)) error {
	return nil
}

func (f *failingExtension) GetID() string {
	return f.id
}

func (f *failingExtension) GetArch() string {
	return "test"
}

func TestRegisterExtensionDoesNotAddFailedLoad(t *testing.T) {
	const extensionID = "handler-test-failed-load"
	wantErr := errors.New("load failed")
	request, err := proto.Marshal(&sliverpb.RegisterExtensionReq{Name: extensionID})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	var (
		responseData []byte
		responseErr  error
		responses    int
	)
	registerExtension(request, func(data []byte, err error) {
		responses++
		responseData = data
		responseErr = err
	}, func(_ []byte, id string, _, _ string) extension.Extension {
		return &failingExtension{id: id, loadErr: wantErr}
	})

	if responseErr != nil {
		t.Fatalf("response callback returned error: %v", responseErr)
	}
	if responses != 1 {
		t.Fatalf("got %d responses, want 1", responses)
	}
	response := &sliverpb.RegisterExtension{}
	if err := proto.Unmarshal(responseData, response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.GetResponse().GetErr() != wantErr.Error() {
		t.Fatalf("got response error %q, want %q", response.GetResponse().GetErr(), wantErr)
	}
	if slices.Contains(extension.List(), extensionID) {
		t.Fatalf("extension %q was registered after its load failed", extensionID)
	}
}

func TestSystemHandlersIncludeExtensions(t *testing.T) {
	handlers := GetSystemHandlers()
	for _, messageType := range []uint32{
		sliverpb.MsgRegisterExtensionReq,
		sliverpb.MsgCallExtensionReq,
		sliverpb.MsgListExtensionsReq,
	} {
		if handlers[messageType] == nil {
			t.Errorf("handler for message type %d is not registered", messageType)
		}
	}
}
