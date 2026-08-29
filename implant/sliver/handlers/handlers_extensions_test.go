//go:build linux || darwin || windows

package handlers

import (
	"bytes"
	"errors"
	"math"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bishopfox/sliver/implant/sliver/extension"
	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/sliverpb"
	"github.com/sliverarmory/reflektor/bof"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
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

func TestCallExtensionBOFOutputsProtobufRoundTrip(t *testing.T) {
	requestDescriptor := (&sliverpb.CallExtensionReq{}).ProtoReflect().Descriptor()
	wantOutputField := requestDescriptor.Fields().ByName("WantBOFOutputs")
	if wantOutputField == nil || wantOutputField.Number() != 7 {
		t.Fatalf("WantBOFOutputs field = %v, want field number 7", wantOutputField)
	}
	wantRequest := &sliverpb.CallExtensionReq{IsBOF: true, WantBOFOutputs: true}
	encodedRequest, err := proto.Marshal(wantRequest)
	if err != nil {
		t.Fatalf("marshal CallExtensionReq: %v", err)
	}
	gotRequest := &sliverpb.CallExtensionReq{}
	if err := proto.Unmarshal(encodedRequest, gotRequest); err != nil {
		t.Fatalf("unmarshal CallExtensionReq: %v", err)
	}
	if !gotRequest.GetIsBOF() || !gotRequest.GetWantBOFOutputs() {
		t.Fatalf("request round trip = {IsBOF:%t WantBOFOutputs:%t}, want both true", gotRequest.GetIsBOF(), gotRequest.GetWantBOFOutputs())
	}

	descriptor := (&sliverpb.CallExtension{}).ProtoReflect().Descriptor()
	for _, field := range []struct {
		name   protoreflect.Name
		number protoreflect.FieldNumber
	}{
		{name: "Output", number: 1},
		{name: "ServerStore", number: 2},
		{name: "BOFOutputs", number: 3},
		{name: "Response", number: 9},
	} {
		got := descriptor.Fields().ByName(field.name)
		if got == nil || got.Number() != field.number {
			t.Fatalf("field %q number = %v, want %d", field.name, got, field.number)
		}
	}
	if got := descriptor.Fields().ByName("BOFOutputs").Cardinality(); got != protoreflect.Repeated {
		t.Fatalf("BOFOutputs cardinality = %v, want repeated", got)
	}

	wantResponse := &sliverpb.CallExtension{
		Output: []byte{'a', 0x00, 0xff, 'b'},
		BOFOutputs: []*sliverpb.BOFOutput{
			{Type: 0, Data: []byte("default")},
			{Type: 0x0d, Data: []byte("error")},
			{Type: math.MinInt32, Data: []byte{0x00, 0xff}},
		},
	}
	encodedResponse, err := proto.Marshal(wantResponse)
	if err != nil {
		t.Fatalf("marshal CallExtension: %v", err)
	}
	gotResponse := &sliverpb.CallExtension{}
	if err := proto.Unmarshal(encodedResponse, gotResponse); err != nil {
		t.Fatalf("unmarshal CallExtension: %v", err)
	}
	if !bytes.Equal(gotResponse.GetOutput(), wantResponse.GetOutput()) {
		t.Fatalf("legacy Output = %v, want %v", gotResponse.GetOutput(), wantResponse.GetOutput())
	}
	if len(gotResponse.GetBOFOutputs()) != len(wantResponse.GetBOFOutputs()) {
		t.Fatalf("BOFOutputs records = %d, want %d", len(gotResponse.GetBOFOutputs()), len(wantResponse.GetBOFOutputs()))
	}
	for index, wantRecord := range wantResponse.GetBOFOutputs() {
		gotRecord := gotResponse.GetBOFOutputs()[index]
		if gotRecord.GetType() != wantRecord.GetType() || !bytes.Equal(gotRecord.GetData(), wantRecord.GetData()) {
			t.Fatalf("BOFOutputs[%d] = {Type:%d Data:%v}, want {Type:%d Data:%v}", index, gotRecord.GetType(), gotRecord.GetData(), wantRecord.GetType(), wantRecord.GetData())
		}
	}
}

func TestCallExtensionLegacyWireEncodingIsUnchanged(t *testing.T) {
	message := &sliverpb.CallExtension{
		Output:      []byte{'a', 0x00, 0xff, 'b'},
		ServerStore: true,
		Response:    &commonpb.Response{},
	}
	got, err := (proto.MarshalOptions{Deterministic: true}).Marshal(message)
	if err != nil {
		t.Fatalf("marshal legacy CallExtension fields: %v", err)
	}
	want := []byte{0x0a, 0x04, 'a', 0x00, 0xff, 'b', 0x10, 0x01, 0x4a, 0x00}
	if !bytes.Equal(got, want) {
		t.Fatalf("legacy wire bytes = %x, want pre-change encoding %x", got, want)
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
	if len(callResponse.GetBOFOutputs()) != 0 {
		t.Fatalf("native extension response has %d BOF output records, want none", len(callResponse.GetBOFOutputs()))
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

func TestCallExtensionPreservesPartialBOFOutputOnError(t *testing.T) {
	wantData := []byte("bof object")
	wantArgs := []byte{4, 0, 0, 0, 1, 2, 3, 4}
	wantOutputs := []bof.Output{
		{Type: bof.OutputDefault, Data: []byte("partial ")},
		{Type: bof.OutputError, Data: []byte("BOF output")},
		{Type: -2147483648, Data: []byte{0x00, 0xff}},
	}
	wantLegacyOutput := []byte{'p', 'a', 'r', 't', 'i', 'a', 'l', ' ', 'B', 'O', 'F', ' ', 'o', 'u', 't', 'p', 'u', 't', 0x00, 0xff}
	wantErr := errors.New("BOF execution failed")
	for _, test := range []struct {
		name      string
		wantTyped bool
	}{
		{name: "legacy", wantTyped: false},
		{name: "typed", wantTyped: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			request, err := proto.Marshal(&sliverpb.CallExtensionReq{
				Name:           "must-not-use-native-registry",
				Export:         "CustomExport",
				Args:           wantArgs,
				BOFData:        wantData,
				IsBOF:          true,
				WantBOFOutputs: test.wantTyped,
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
			}, func(data []byte, entryPoint string, args []byte) ([]bof.Output, error) {
				if !bytes.Equal(data, wantData) {
					t.Fatalf("BOF data = %q, want %q", data, wantData)
				}
				if entryPoint != "CustomExport" {
					t.Fatalf("entry point = %q, want CustomExport", entryPoint)
				}
				if !bytes.Equal(args, wantArgs) {
					t.Fatalf("args = %v, want %v", args, wantArgs)
				}
				return wantOutputs, wantErr
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
			if response.GetResponse().GetErr() != wantErr.Error() {
				t.Fatalf("response error = %q, want %q", response.GetResponse().GetErr(), wantErr)
			}
			if !test.wantTyped {
				if !bytes.Equal(response.GetOutput(), wantLegacyOutput) {
					t.Fatalf("legacy output = %v, want %v", response.GetOutput(), wantLegacyOutput)
				}
				if len(response.GetBOFOutputs()) != 0 {
					t.Fatalf("typed output records = %d, want none", len(response.GetBOFOutputs()))
				}
				return
			}
			if len(response.GetOutput()) != 0 {
				t.Fatalf("legacy output = %v, want none", response.GetOutput())
			}
			if len(response.GetBOFOutputs()) != len(wantOutputs) {
				t.Fatalf("typed output records = %d, want %d", len(response.GetBOFOutputs()), len(wantOutputs))
			}
			for index, want := range wantOutputs {
				got := response.GetBOFOutputs()[index]
				if got.GetType() != int32(want.Type) || !bytes.Equal(got.GetData(), want.Data) {
					t.Fatalf("typed output[%d] = {Type:%d Data:%v}, want {Type:%d Data:%v}", index, got.GetType(), got.GetData(), want.Type, want.Data)
				}
			}
		})
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
