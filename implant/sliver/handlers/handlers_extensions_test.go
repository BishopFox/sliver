//go:build linux || darwin || windows

package handlers

import (
	"bytes"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bishopfox/sliver/implant/sliver/extension"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"google.golang.org/protobuf/proto"
)

type failingExtension struct {
	id      string
	loadErr error
}

type callbackLifetimeExtension struct {
	id               string
	output           []byte
	nativeOutput     []byte
	callActive       atomic.Bool
	callbackReturned chan struct{}
	allowCallReturn  chan struct{}
	allowReturnOnce  sync.Once
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

func (e *callbackLifetimeExtension) Load() error {
	return nil
}

func (e *callbackLifetimeExtension) Call(_ string, _ []byte, callback func([]byte)) error {
	e.callActive.Store(true)
	defer e.callActive.Store(false)

	e.nativeOutput = append([]byte(nil), e.output...)
	callback(e.nativeOutput)
	for index := range e.nativeOutput {
		e.nativeOutput[index] = 0xa5
	}
	close(e.callbackReturned)
	<-e.allowCallReturn
	return nil
}

func (e *callbackLifetimeExtension) GetID() string {
	return e.id
}

func (e *callbackLifetimeExtension) GetArch() string {
	return "test"
}

func (e *callbackLifetimeExtension) allowReturn() {
	e.allowReturnOnce.Do(func() {
		close(e.allowCallReturn)
	})
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

func TestCallExtensionHandlerDefersResponseAndCopiesCallbackOutput(t *testing.T) {
	const extensionID = "handler-test-callback-lifetime"
	wantOutput := []byte("native extension output")
	ext := &callbackLifetimeExtension{
		id:               extensionID,
		output:           wantOutput,
		callbackReturned: make(chan struct{}),
		allowCallReturn:  make(chan struct{}),
	}
	extension.Add(ext)
	defer ext.allowReturn()

	request, err := proto.Marshal(&sliverpb.CallExtensionReq{Name: extensionID, Export: "Run"})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	type handlerResponse struct {
		data       []byte
		err        error
		callActive bool
	}
	responses := make(chan handlerResponse, 2)
	handlerDone := make(chan struct{})
	go func() {
		defer close(handlerDone)
		callExtensionHandler(request, func(data []byte, err error) {
			responses <- handlerResponse{
				data:       append([]byte(nil), data...),
				err:        err,
				callActive: ext.callActive.Load(),
			}
		})
	}()

	select {
	case <-ext.callbackReturned:
	case <-time.After(5 * time.Second):
		t.Fatal("extension callback did not return")
	}
	select {
	case <-responses:
		t.Fatal("RPC response was delivered before Extension.Call returned")
	default:
	}

	ext.allowReturn()
	var response handlerResponse
	select {
	case response = <-responses:
	case <-time.After(5 * time.Second):
		t.Fatal("callExtensionHandler did not deliver a response")
	}
	if response.callActive {
		t.Fatal("RPC response was delivered while Extension.Call was active")
	}
	if response.err != nil {
		t.Fatalf("response callback returned error: %v", response.err)
	}

	callResponse := &sliverpb.CallExtension{}
	if err := proto.Unmarshal(response.data, callResponse); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !bytes.Equal(callResponse.Output, wantOutput) {
		t.Fatalf("response output = %q, want %q", callResponse.Output, wantOutput)
	}
	if bytes.Equal(ext.nativeOutput, wantOutput) {
		t.Fatal("fake native output was not poisoned after the callback")
	}

	select {
	case <-handlerDone:
	case <-time.After(5 * time.Second):
		t.Fatal("callExtensionHandler did not return")
	}
	select {
	case <-responses:
		t.Fatal("callExtensionHandler delivered more than one response")
	default:
	}
}

func TestCallExtensionRoutesBOFDataToBuiltInExecutor(t *testing.T) {
	wantData := []byte("bof object")
	wantArgs := []byte{4, 0, 0, 0, 1, 2, 3, 4}
	wantOutput := []byte("partial BOF output")
	wantErr := errors.New("BOF execution failed")
	request, err := proto.Marshal(&sliverpb.CallExtensionReq{
		Name:    "must-not-use-native-registry",
		Export:  "CustomExport",
		Args:    wantArgs,
		BOFData: wantData,
		IsBOF:   true,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	var (
		responseData []byte
		responseErr  error
		responses    int
	)
	callExtension(request, func(data []byte, err error) {
		responses++
		responseData = data
		responseErr = err
	}, func(data []byte, entryPoint string, args []byte) ([]byte, error) {
		if !bytes.Equal(data, wantData) {
			t.Fatalf("BOF data = %q, want %q", data, wantData)
		}
		if entryPoint != "CustomExport" {
			t.Fatalf("entry point = %q, want CustomExport", entryPoint)
		}
		if !bytes.Equal(args, wantArgs) {
			t.Fatalf("args = %v, want %v", args, wantArgs)
		}
		return wantOutput, wantErr
	})

	if responseErr != nil {
		t.Fatalf("response callback returned error: %v", responseErr)
	}
	if responses != 1 {
		t.Fatalf("got %d responses, want 1", responses)
	}
	response := &sliverpb.CallExtension{}
	if err := proto.Unmarshal(responseData, response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !bytes.Equal(response.Output, wantOutput) {
		t.Fatalf("output = %q, want %q", response.Output, wantOutput)
	}
	if response.GetResponse().GetErr() != wantErr.Error() {
		t.Fatalf("response error = %q, want %q", response.GetResponse().GetErr(), wantErr)
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
