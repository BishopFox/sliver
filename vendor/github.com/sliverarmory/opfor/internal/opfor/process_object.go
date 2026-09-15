package opfor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	osexec "os/exec"
	"strings"
	"sync"
)

// processObject is the portable counterpart of Sleep's ProcessObject. The
// associated I/O handle reads the child's stdout and writes its stdin; wait
// publishes the exit status independently of either stream's lifetime.
type processObject struct {
	cmd     *osexec.Cmd
	handle  *sleepIOHandle
	owner   *Script
	runtime *Runtime
	done    chan struct{}
	cancel  context.CancelFunc

	childOutput io.Closer
	output      *processOutputLimits

	mu        sync.RWMutex
	result    Value
	waitErr   error
	closed    bool
	finished  bool
	closeErr  error
	closeOnce sync.Once
}

type processSpec struct {
	command     []string
	environment []string
	directory   string
}

type processOpenError struct{ error }

type processOutputLimits struct {
	stdout *runtimeOutputWriter
	stderr *runtimeOutputWriter
}

func (limits *processOutputLimits) limitError() error {
	if limits == nil {
		return nil
	}
	return errors.Join(limits.stdout.LimitError(), limits.stderr.LimitError())
}

type processOutputReader struct {
	reader io.ReadCloser
	limits *processOutputLimits
}

func (reader *processOutputReader) Read(data []byte) (int, error) {
	if reader == nil || reader.reader == nil {
		return 0, io.ErrClosedPipe
	}
	read, err := reader.reader.Read(data)
	if err != nil {
		if limitErr := reader.limits.limitError(); limitErr != nil {
			return read, limitErr
		}
	}
	return read, err
}

func (reader *processOutputReader) Close() error {
	if reader == nil || reader.reader == nil {
		return nil
	}
	return reader.reader.Close()
}

func (reader *processOutputReader) sleepUnderlyingReader() io.Reader {
	if reader == nil {
		return nil
	}
	return reader.reader
}

func (state *ioBuiltinState) executeProcess(ctx context.Context, invocation Invocation) (Value, error) {
	spec, err := state.processArguments(invocation)
	if err != nil {
		var openError *processOpenError
		if errors.As(err, &openError) {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return Null(), ctxErr
			}
			return state.flagProcessOpenError(invocation, err)
		}
		return Null(), err
	}

	owner := state.runtime.script(invocation.Script)
	if invocation.Script != 0 && owner == nil {
		return Null(), ErrScriptUnloaded
	}
	process, err := state.startProcess(ctx, owner, spec)
	if err == nil {
		return ObjectValue(process.handle), nil
	}
	if errors.Is(err, ErrScriptUnloaded) || errors.Is(err, ErrRuntimeClosed) {
		return Null(), err
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return Null(), ctxErr
	}

	// ProcessObject.open catches launch failures, records a soft error, and
	// still returns an inert I/O object to the script.
	return state.flagProcessOpenError(invocation, err)
}

func (state *ioBuiltinState) flagProcessOpenError(invocation Invocation, err error) (Value, error) {
	handle := newIOHandle("process", nil, nil, false, false, false).withRuntimeOutputAccount(state.runtime.resources)
	_, flaggedErr := state.runtime.flagSourceError(invocation, err)
	return ObjectValue(handle), flaggedErr
}

func (state *ioBuiltinState) processArguments(invocation Invocation) (processSpec, error) {
	if len(invocation.Arguments) == 0 {
		return processSpec{}, &processOpenError{fmt.Errorf("&%s: missing command", builtinName(invocation.Name))}
	}

	var command []string
	if array, ok := invocation.Arg(0).Array(); ok {
		values := array.Values()
		command = make([]string, len(values))
		for index, value := range values {
			command[index] = value.String()
		}
	} else {
		command = splitProcessCommand(invocation.Arg(0).String())
	}
	if len(command) == 0 || command[0] == "" {
		return processSpec{}, &processOpenError{fmt.Errorf("&%s: command is empty", builtinName(invocation.Name))}
	}
	command[0] = strings.ReplaceAll(command[0], "/", string(os.PathSeparator))

	var environment []string
	if len(invocation.Arguments) > 1 && !invocation.Arg(1).IsNull() {
		hash, ok := invocation.Arg(1).Hash()
		if !ok {
			return processSpec{}, fmt.Errorf("&%s: expected environment hash--received: %s",
				builtinName(invocation.Name), invocation.Arg(1).Describe())
		}
		keys := hash.KeyValues()
		environment = make([]string, 0, len(keys))
		for _, key := range keys {
			value, _ := hash.GetValue(key)
			environment = append(environment, sleepCanonicalString(key)+"="+value.String())
		}
	}

	// OPFOR models Sleep's per-script working directory without mutating the
	// host Go process. Use it for an omitted/null start directory as well as for
	// resolving an explicit relative directory.
	directory := state.workingDirectory()
	if len(invocation.Arguments) > 2 && !invocation.Arg(2).IsNull() {
		directory = state.resolvePath(invocation.Arg(2).String())
	}
	return processSpec{command: command, environment: environment, directory: directory}, nil
}

// splitProcessCommand mirrors Java String.split("\\s"): every ASCII
// whitespace character is a delimiter, adjacent delimiters retain interior
// empty arguments, and trailing empty arguments are discarded.
func splitProcessCommand(command string) []string {
	if command == "" {
		return []string{""}
	}
	arguments := make([]string, 0, 4)
	start := 0
	for index := 0; index < len(command); index++ {
		if !isJavaWhitespace(command[index]) {
			continue
		}
		arguments = append(arguments, command[start:index])
		start = index + 1
	}
	arguments = append(arguments, command[start:])
	for len(arguments) > 0 && arguments[len(arguments)-1] == "" {
		arguments = arguments[:len(arguments)-1]
	}
	return arguments
}

func isJavaWhitespace(character byte) bool {
	switch character {
	case ' ', '\t', '\n', '\v', '\f', '\r':
		return true
	default:
		return false
	}
}

func (state *ioBuiltinState) startProcess(ctx context.Context, owner *Script, spec processSpec) (*processObject, error) {
	childInput, parentWriter, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("start command: %w", err)
	}
	parentReader, childOutput, err := os.Pipe()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("start command: %w", err), childInput.Close(), parentWriter.Close())
	}

	// A ProcessObject is asynchronous state owned by its Script (or by the
	// Runtime for a direct Runtime.Invoke). Detach the short-lived entry lease
	// while retaining the importer's cancellation/deadline; unload and Close
	// explicitly destroy and join their registered processes.
	processContext, releaseProcessContext := detachAsynchronousExecutionContextLease(ctx)
	processContext, cancelProcessContext := context.WithCancel(processContext)
	cancelProcessContext = cancelContextWithRelease(cancelProcessContext, releaseProcessContext)
	cmd := osexec.CommandContext(processContext, spec.command[0], spec.command[1:]...)
	cmd.Dir = spec.directory
	cmd.Env = spec.environment
	cmd.Stdin = childInput
	outputLimits := &processOutputLimits{
		stdout: newRuntimeOutputWriter(state.runtime.resources, childOutput),
		stderr: newRuntimeOutputWriter(state.runtime.resources, io.Discard),
	}
	cmd.Stdout = outputLimits.stdout
	// Java ProcessObject exposes only stdout/stdin. Its error stream is neither
	// copied to the script console nor promoted to a Sleep error.
	cmd.Stderr = outputLimits.stderr
	if err := cmd.Start(); err != nil {
		cancelProcessContext()
		return nil, errors.Join(fmt.Errorf("start command: %w", err), childInput.Close(), parentWriter.Close(), parentReader.Close(), childOutput.Close())
	}
	closeErr := childInput.Close()
	if closeErr != nil {
		_ = parentReader.Close()
		_ = parentWriter.Close()
		_ = childOutput.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		cancelProcessContext()
		return nil, fmt.Errorf("start command: close child pipe endpoints: %w", closeErr)
	}

	outputReader := &processOutputReader{reader: parentReader, limits: outputLimits}
	handle := newIOHandle("process", outputReader, parentWriter, true, true, false).withRuntimeOutputAccount(state.runtime.resources)
	process := &processObject{
		cmd: cmd, handle: handle, owner: owner, runtime: state.runtime,
		done: make(chan struct{}), cancel: cancelProcessContext, result: Null(), childOutput: childOutput, output: outputLimits,
	}
	handle.setProcess(process)
	go process.reap()

	if process.owner != nil {
		if !process.owner.registerProcess(process) {
			_ = process.close()
			_ = process.join(context.Background())
			return nil, ErrScriptUnloaded
		}
	} else if process.runtime != nil && !process.runtime.registerProcess(process) {
		_ = process.close()
		_ = process.join(context.Background())
		return nil, ErrRuntimeClosed
	}
	return process, nil
}

func (process *processObject) reap() {
	err := process.cmd.Wait()
	process.cancel()
	outputErr := process.output.limitError()
	if process.childOutput != nil {
		_ = process.childOutput.Close()
	}
	if outputErr != nil {
		err = outputErr
	}
	result := Int(0)
	var exitError *osexec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitError):
		result = integerValue(int64(exitError.ExitCode()))
		err = nil
	}

	process.mu.Lock()
	process.result = result
	process.waitErr = err
	process.finished = true
	// Lock ordering is process.mu then the owner Script/Runtime mutex. close
	// uses the same order, so the closed/finished handshake cannot miss the
	// unregister transition and done is not published before it completes.
	if process.closed {
		process.unregisterOwner()
	}
	close(process.done)
	process.mu.Unlock()
}

func (process *processObject) wait(ctx context.Context, runtime *Runtime, invocation Invocation) (Value, error) {
	select {
	case <-ctx.Done():
		return Null(), ctx.Err()
	case <-process.done:
	}
	process.mu.RLock()
	result, err := process.result, process.waitErr
	process.mu.RUnlock()
	if err == nil {
		return result, nil
	}
	if errors.Is(err, ErrResourceLimit) {
		return result, err
	}
	return runtime.flagSourceError(invocation, err)
}

func (process *processObject) close() error {
	if process == nil {
		return nil
	}
	process.closeOnce.Do(func() {
		process.mu.Lock()
		process.closed = true
		if process.finished {
			process.unregisterOwner()
		}
		process.mu.Unlock()
		if process.cmd != nil && process.cmd.Process != nil {
			// Process.destroy is best-effort; a naturally exited process is not a
			// close failure, and Sleep's IOObject.close swallows stream errors.
			// Destroy before waiting for the handle's write lock: a writer may be
			// blocked on a full child stdin pipe and only child exit can wake it.
			_ = process.cmd.Process.Kill()
		}
		// IOObject.close swallows stream close failures. Preserve that
		// best-effort lifecycle after the underlying process has been woken.
		_ = process.handle.close()
	})
	return process.closeErr
}

func (process *processObject) join(ctx context.Context) error {
	if process == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-process.done:
		return nil
	}
}

func (script *Script) registerProcess(process *processObject) bool {
	if script == nil || process == nil {
		return false
	}
	script.mu.Lock()
	defer script.mu.Unlock()
	if !script.active {
		return false
	}
	if script.processes == nil {
		script.processes = make(map[*processObject]struct{})
	}
	script.processes[process] = struct{}{}
	return true
}

func (script *Script) unregisterProcess(process *processObject) {
	if script == nil || process == nil {
		return
	}
	script.mu.Lock()
	delete(script.processes, process)
	script.mu.Unlock()
}

func (runtime *Runtime) registerProcess(process *processObject) bool {
	if runtime == nil || process == nil {
		return false
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if runtime.closing || runtime.closed {
		return false
	}
	if runtime.processes == nil {
		runtime.processes = make(map[*processObject]struct{})
	}
	runtime.processes[process] = struct{}{}
	return true
}

func (runtime *Runtime) unregisterProcess(process *processObject) {
	if runtime == nil || process == nil {
		return
	}
	runtime.mu.Lock()
	delete(runtime.processes, process)
	runtime.mu.Unlock()
}

func (process *processObject) unregisterOwner() {
	if process == nil {
		return
	}
	if process.owner != nil {
		process.owner.unregisterProcess(process)
		return
	}
	if process.runtime != nil {
		process.runtime.unregisterProcess(process)
	}
}
