//go:build (windows && (arm64 || amd64 || 386)) || (darwin && (arm64 || amd64)) || (linux && (arm64 || amd64 || 386))

package extension

import (
	"bytes"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWasmMemFSGuestIntegration(t *testing.T) {
	wasmPath := buildWasmTestModule(
		t,
		"SLIVER_WASM_MEMFS_RW_WASM",
		"./implant/sliver/extension/testdata/memfs-rw",
	)
	wasm, err := os.ReadFile(wasmPath)
	mustNoError(t, err)

	t.Run("read-write state persists across executions", func(t *testing.T) {
		extension, err := NewWasmExtension(t.Name(), wasm, nil)
		mustNoError(t, err)

		stdout, stderr, exitCodes, executeErrors := executeWasmMemFSGuest(
			t,
			extension,
			[]string{"write"},
			[]string{"read"},
		)
		mustNoError(t, executeErrors[0], stderr)
		mustNoError(t, executeErrors[1], stderr)
		mustZero(t, exitCodes[0], stderr)
		mustZero(t, exitCodes[1], stderr)
		assert.Contains(t, stdout, "memfs-rw-write-ok")
		assert.Contains(t, stdout, "memfs-rw-state-ok")
	})

	t.Run("read-only rejects guest mutations", func(t *testing.T) {
		extension, err := NewWasmExtensionWithOptions(
			t.Name(),
			wasm,
			map[string][]byte{"seed.txt": []byte("seed-data")},
			WithReadOnlyMemFS(),
		)
		mustNoError(t, err)

		stdout, stderr, exitCodes, executeErrors := executeWasmMemFSGuest(
			t,
			extension,
			[]string{"readonly"},
		)
		mustNoError(t, executeErrors[0], stderr)
		mustZero(t, exitCodes[0], stderr)
		assert.Contains(t, stdout, "memfs-readonly-ok")
	})
}

func executeWasmMemFSGuest(
	t *testing.T,
	extension *WasmExtension,
	executions ...[]string,
) (stdout string, stderr string, exitCodes []uint32, executeErrors []error) {
	t.Helper()
	t.Cleanup(func() {
		_ = extension.Close()
	})

	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	var readers sync.WaitGroup
	readers.Add(2)
	go func() {
		defer readers.Done()
		_, _ = io.Copy(&stdoutBuffer, extension.Stdout.Reader)
	}()
	go func() {
		defer readers.Done()
		_, _ = io.Copy(&stderrBuffer, extension.Stderr.Reader)
	}()

	exitCodes = make([]uint32, 0, len(executions))
	executeErrors = make([]error, 0, len(executions))
	for _, args := range executions {
		exitCode, err := extension.Execute(args)
		exitCodes = append(exitCodes, exitCode)
		executeErrors = append(executeErrors, err)
	}

	_ = extension.Stdout.Writer.Close()
	_ = extension.Stderr.Writer.Close()
	readers.Wait()
	return stdoutBuffer.String(), stderrBuffer.String(), exitCodes, executeErrors
}
