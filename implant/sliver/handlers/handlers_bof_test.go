//go:build linux || darwin || windows

package handlers

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/sliverarmory/reflektor/bof"
)

type fakeBOFObject struct {
	outputs      []bof.Output
	executeErr   error
	closeErr     error
	executeArgs  []byte
	executeCalls int
	closeCalls   int
}

func (object *fakeBOFObject) Execute(args []byte) ([]bof.Output, error) {
	object.executeCalls++
	object.executeArgs = append([]byte(nil), args...)
	return object.outputs, object.executeErr
}

func (object *fakeBOFObject) Close() error {
	object.closeCalls++
	return object.closeErr
}

func TestExecuteBOFUsesDefaultEntryPointLookup(t *testing.T) {
	for _, entryPoint := range []string{"", "go", "_go", "coffee", "_coffee"} {
		t.Run(entryPoint, func(t *testing.T) {
			object := &fakeBOFObject{}
			var gotOptions bof.LoadOptions
			_, err := executeBOFWithLoader([]byte("object"), entryPoint, nil, func(_ []byte, options bof.LoadOptions) (bofObject, error) {
				gotOptions = options
				return object, nil
			})
			if err != nil {
				t.Fatalf("executeBOF() error = %v", err)
			}
			if gotOptions.EntryPoint != "" {
				t.Fatalf("entry point option = %q, want default lookup", gotOptions.EntryPoint)
			}
			if object.closeCalls != 1 {
				t.Fatalf("Close() calls = %d, want 1", object.closeCalls)
			}
		})
	}
}

func TestExecuteBOFUsesExactCustomEntryPointAndConcatenatesOutput(t *testing.T) {
	wantArgs := []byte{4, 0, 0, 0, 1, 2, 3, 4}
	object := &fakeBOFObject{outputs: []bof.Output{
		{Type: bof.OutputDefault, Data: []byte("first")},
		{Type: bof.OutputError, Data: []byte("second")},
	}}
	var gotOptions bof.LoadOptions
	output, err := executeBOFWithLoader([]byte("object"), " CustomExport ", wantArgs, func(_ []byte, options bof.LoadOptions) (bofObject, error) {
		gotOptions = options
		return object, nil
	})
	if err != nil {
		t.Fatalf("executeBOF() error = %v", err)
	}
	if gotOptions.EntryPoint != " CustomExport " {
		t.Fatalf("entry point option = %q, want exact custom symbol", gotOptions.EntryPoint)
	}
	if !bytes.Equal(object.executeArgs, wantArgs) {
		t.Fatalf("Execute() args = %v, want %v", object.executeArgs, wantArgs)
	}
	if !bytes.Equal(output, []byte("firstsecond")) {
		t.Fatalf("output = %q, want ordered concatenation", output)
	}
	if object.executeCalls != 1 || object.closeCalls != 1 {
		t.Fatalf("Execute() calls = %d, Close() calls = %d, want 1 each", object.executeCalls, object.closeCalls)
	}
}

func TestExecuteBOFReportsExecuteAndCloseErrors(t *testing.T) {
	object := &fakeBOFObject{
		outputs: []bof.Output{
			{Data: []byte("partial")},
		},
		executeErr: errors.New("execute failed"),
		closeErr:   errors.New("close failed"),
	}
	output, err := executeBOFWithLoader([]byte("object"), "go", nil, func([]byte, bof.LoadOptions) (bofObject, error) {
		return object, nil
	})
	if !bytes.Equal(output, []byte("partial")) {
		t.Fatalf("output = %q, want partial output", output)
	}
	if err == nil || !strings.Contains(err.Error(), "execute failed") || !strings.Contains(err.Error(), "close failed") {
		t.Fatalf("error = %v, want execute and close failures", err)
	}
	if object.closeCalls != 1 {
		t.Fatalf("Close() calls = %d, want 1", object.closeCalls)
	}
}

func TestExecuteBOFReportsLoadErrorWithoutClosing(t *testing.T) {
	wantErr := errors.New("load failed")
	_, err := executeBOFWithLoader([]byte("object"), "go", nil, func([]byte, bof.LoadOptions) (bofObject, error) {
		return nil, wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want wrapped load error", err)
	}
}
