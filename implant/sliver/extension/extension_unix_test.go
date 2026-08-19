//go:build (darwin && (amd64 || arm64)) || (linux && (386 || amd64 || arm64))

package extension

import (
	"bytes"
	"testing"
	"unsafe"
)

func TestUnixExtensionCallbackCopiesOutput(t *testing.T) {
	extension := &UnixExtension{}
	var got []byte
	call := &extensionCall{onFinish: func(output []byte) {
		got = output
	}}
	extension.active.Store(call)
	defer extension.active.Store(nil)

	source := []byte("extension output")
	result := extension.extensionCallback(uintptr(unsafe.Pointer(&source[0])), int32(len(source)))
	if result != Success {
		t.Fatalf("extension callback returned %d, want %d", result, Success)
	}
	source[0] = 'X'
	if !bytes.Equal(got, []byte("extension output")) {
		t.Fatalf("extension callback output = %q, want an independent copy", got)
	}
}

func TestUnixExtensionCallbackRejectsInvalidCalls(t *testing.T) {
	extension := &UnixExtension{}
	if result := extension.extensionCallback(0, 1); result != Failure {
		t.Fatalf("callback without an active call returned %d, want %d", result, Failure)
	}

	extension.active.Store(&extensionCall{onFinish: func([]byte) {}})
	defer extension.active.Store(nil)
	if result := extension.extensionCallback(0, 1); result != Failure {
		t.Fatalf("callback with a nil data pointer returned %d, want %d", result, Failure)
	}
	if result := extension.extensionCallback(1, -1); result != Failure {
		t.Fatalf("callback with a negative length returned %d, want %d", result, Failure)
	}
}

func TestUnixExtensionCallbackContainsPanics(t *testing.T) {
	extension := &UnixExtension{}
	extension.active.Store(&extensionCall{onFinish: func([]byte) {
		panic("callback failed")
	}})
	defer extension.active.Store(nil)

	source := []byte("output")
	if result := extension.extensionCallback(uintptr(unsafe.Pointer(&source[0])), int32(len(source))); result != Failure {
		t.Fatalf("panicking callback returned %d, want %d", result, Failure)
	}
}
