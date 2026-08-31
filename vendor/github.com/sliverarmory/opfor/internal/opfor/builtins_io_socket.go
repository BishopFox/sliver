package opfor

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	sleepSocketDefaultHost    = "127.0.0.1"
	sleepSocketDefaultPort    = int32(1)
	sleepSocketDefaultTimeout = int32(60_000)
	sleepSocketDefaultLinger  = int32(5)
)

type sleepSocketOperation uint8

const (
	sleepSocketConnect sleepSocketOperation = iota
	sleepSocketListen
)

// sleepSocketState is Runtime-owned. Sleep 2.1 keeps its listener table in a
// process-global static map and never clears it when a script unloads. OPFOR
// deliberately scopes the same requested-port cache to one Runtime so Close
// can cancel blocked work and release operating-system resources safely.
type sleepSocketState struct {
	runtime *Runtime

	mu        sync.Mutex
	closed    bool
	listeners map[int32]*sleepSocketListener
	tasks     map[*sleepIOHandle]*sleepSocketTask
}

type sleepSocketListener struct {
	requestedPort int32
	listener      *net.TCPListener
	acceptor      sleepSocketAcceptor
}

// sleepSocketAcceptor isolates the platform-specific mechanism used to let
// several cached ServerSocket accepts wait at once. Unix duplicates the
// listener descriptor, while Windows uses a broker because TCPListener.File is
// not implemented there.
type sleepSocketAcceptor interface {
	accept(context.Context, int32) (net.Conn, error)
	close() error
}

type sleepSocketTask struct {
	state      *sleepSocketState
	owner      *Script
	handle     *sleepIOHandle
	invocation Invocation
	operation  sleepSocketOperation
	callback   Callable
	peer       Argument
	timeout    int32
	linger     int32
	host       string
	port       int32
	laddr      string
	laddrSet   bool
	lport      int32
	backlog    int32

	ctx            context.Context
	cancel         context.CancelCauseFunc
	releaseContext func()
	done           chan struct{}
	once           sync.Once

	mu              sync.Mutex
	conn            net.Conn
	connCloser      *sleepOnceCloser
	attached        bool
	finished        bool
	closing         bool
	invocationError error
}

type sleepSocketOptions struct {
	laddr    string
	laddrSet bool
	lport    int32
	linger   int32
	backlog  int32
}

type sleepSocketCall struct {
	positional []Argument
	options    sleepSocketOptions
}

func newSleepSocketState(runtime *Runtime) *sleepSocketState {
	return &sleepSocketState{
		runtime:   runtime,
		listeners: make(map[int32]*sleepSocketListener),
		tasks:     make(map[*sleepIOHandle]*sleepSocketTask),
	}
}

func parseSleepSocketCall(invocation Invocation) sleepSocketCall {
	call := sleepSocketCall{options: sleepSocketOptions{linger: sleepSocketDefaultLinger}}
	positional, named := extractSleepNamedArguments(invocation.Arguments)
	call.positional = positional
	for name, argument := range named {
		value := argument.Resolve()
		switch name {
		case "laddr":
			call.options.laddr = value.String()
			call.options.laddrSet = true
		case "lport":
			call.options.lport = sleepSocketInt32(value)
		case "linger":
			call.options.linger = sleepSocketInt32(value)
		case "backlog":
			call.options.backlog = sleepSocketInt32(value)
		}
	}
	return call
}

func (state *ioBuiltinState) connect(ctx context.Context, invocation Invocation) (Value, error) {
	call := parseSleepSocketCall(invocation)
	host := sleepSocketDefaultHost
	port := sleepSocketDefaultPort
	timeout := sleepSocketDefaultTimeout
	if len(call.positional) > 0 {
		host = call.positional[0].Resolve().String()
	}
	if len(call.positional) > 1 {
		port = sleepSocketInt32(call.positional[1].Resolve())
	}
	if len(call.positional) > 2 {
		timeout = sleepSocketInt32(call.positional[2].Resolve())
	}
	callback, err := sleepSocketCallback(invocation, call.positional)
	if err != nil {
		return Null(), sleepIOBridgeWarning(ctx, invocation, err)
	}

	task, err := state.newSleepSocketTask(ctx, invocation, sleepSocketConnect, callback)
	if err != nil {
		return Null(), err
	}
	task.host = host
	task.port = port
	task.timeout = timeout
	task.laddr = call.options.laddr
	task.laddrSet = call.options.laddrSet
	task.lport = call.options.lport
	task.linger = call.options.linger
	task.backlog = call.options.backlog

	if callback != nil {
		go task.run()
		return ObjectValue(task.handle), nil
	}
	task.run()
	return ObjectValue(task.handle), task.flowError()
}

func (state *ioBuiltinState) listen(ctx context.Context, invocation Invocation) (Value, error) {
	call := parseSleepSocketCall(invocation)
	port := int32(-1)
	timeout := sleepSocketDefaultTimeout
	peer := Argument{Value: Null()}
	if len(call.positional) > 0 {
		port = sleepSocketInt32(call.positional[0].Resolve())
	}
	if len(call.positional) > 1 {
		timeout = sleepSocketInt32(call.positional[1].Resolve())
	}
	if len(call.positional) > 2 {
		peer = call.positional[2]
	}
	callback, err := sleepSocketCallback(invocation, call.positional)
	if err != nil {
		return Null(), sleepIOBridgeWarning(ctx, invocation, err)
	}

	task, err := state.newSleepSocketTask(ctx, invocation, sleepSocketListen, callback)
	if err != nil {
		return Null(), err
	}
	task.port = port
	task.timeout = timeout
	task.peer = peer
	task.laddr = call.options.laddr
	task.laddrSet = call.options.laddrSet
	task.lport = call.options.lport
	task.linger = call.options.linger
	task.backlog = call.options.backlog

	if callback != nil {
		go task.run()
		return ObjectValue(task.handle), nil
	}
	task.run()
	return ObjectValue(task.handle), task.flowError()
}

func sleepSocketCallback(invocation Invocation, positional []Argument) (Callable, error) {
	if len(positional) < 4 {
		return nil, nil
	}
	value := positional[3].Resolve()
	// BasicIO's asynchronous socket callback belongs to the retained raw Sleep
	// execution environment, not to an importer integration registration. Keep
	// it valid across ScriptLoader registry unload while terminal Script unload
	// still revokes it.
	callback, err := retainScriptLifetimeCallback(invocation, value)
	if err == nil {
		return callback, nil
	}
	if errors.Is(err, ErrInvalidCallable) {
		name := value.String()
		// ScriptEnvironment.getFunction performs an exact lookup. Sleep stores
		// named functions with their leading ampersand, so "&name" resolves but
		// the otherwise-identical bare string "name" does not.
		if strings.HasPrefix(name, "&") {
			if owner := invocation.Runtime.script(invocation.Script); owner != nil && owner.Active() {
				if closure, ok := owner.resolveFunction(name).(*scriptClosure); ok && closure != nil {
					return &scriptLifetimeCallback{owner: owner, callable: closure}, nil
				}
			}
		}
		return nil, fmt.Errorf("&%s: expected &closure--received: %s", builtinName(invocation.Name), value.Describe())
	}
	return nil, err
}

func (state *ioBuiltinState) newSleepSocketTask(
	ctx context.Context,
	invocation Invocation,
	operation sleepSocketOperation,
	callback Callable,
) (*sleepSocketTask, error) {
	if state == nil || state.runtime == nil || state.runtime.socketState == nil {
		return nil, errors.New("opfor: socket runtime is unavailable")
	}
	owner := state.runtime.script(invocation.Script)
	if invocation.Script != 0 && (owner == nil || !owner.Active()) {
		return nil, ErrScriptUnloaded
	}
	taskContext, releaseTaskContext, cancel := newAsynchronousExecutionTaskContext(ctx)
	handle := newIOHandle("socket", nil, nil, true, true, false).withRuntimeOutputAccount(state.runtime.resources)
	task := &sleepSocketTask{
		state:          state.runtime.socketState,
		owner:          owner,
		handle:         handle,
		invocation:     invocation,
		operation:      operation,
		callback:       callback,
		ctx:            taskContext,
		cancel:         cancel,
		releaseContext: releaseTaskContext,
		done:           make(chan struct{}),
	}
	if callback != nil {
		handle.setWorker(task)
	}
	if err := task.state.register(task); err != nil {
		cancel(context.Canceled)
		task.releaseContext()
		return nil, err
	}
	if owner != nil {
		owner.mu.Lock()
		if !owner.active {
			owner.mu.Unlock()
			task.state.unregister(task)
			cancel(context.Canceled)
			task.releaseContext()
			return nil, ErrScriptUnloaded
		}
		if owner.socketTasks == nil {
			owner.socketTasks = make(map[*sleepSocketTask]struct{})
		}
		owner.socketTasks[task] = struct{}{}
		owner.mu.Unlock()
	}
	return task, nil
}

func (state *sleepSocketState) register(task *sleepSocketTask) error {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.closed {
		return ErrScriptUnloaded
	}
	state.tasks[task.handle] = task
	return nil
}

func (state *sleepSocketState) unregister(task *sleepSocketTask) {
	if state == nil || task == nil {
		return
	}
	state.mu.Lock()
	if state.tasks[task.handle] == task {
		delete(state.tasks, task.handle)
	}
	state.mu.Unlock()
	if task.owner != nil {
		task.owner.mu.Lock()
		delete(task.owner.socketTasks, task)
		task.owner.mu.Unlock()
	}
}

func (state *sleepSocketState) lookup(handle *sleepIOHandle) *sleepSocketTask {
	if state == nil || handle == nil {
		return nil
	}
	state.mu.Lock()
	task := state.tasks[handle]
	state.mu.Unlock()
	return task
}

func (task *sleepSocketTask) run() {
	var networkErr error
	switch task.operation {
	case sleepSocketConnect:
		networkErr = task.runConnect()
	case sleepSocketListen:
		networkErr = task.runListen()
	}

	if networkErr != nil && !isSleepSocketCancellation(task.ctx, networkErr) {
		_, task.invocationError = task.state.runtime.flagSourceError(task.invocation, networkErr)
	}
	if task.callback != nil {
		callbackContext := withCurrentFiber(task.ctx, nil)
		callbackContext = withExecutionMeter(callbackContext, task.state.runtime)
		switch callback := task.callback.(type) {
		case *invocationCallback:
			_, _ = callback.invokeNamed(callbackContext, "&callback", ObjectValue(task.handle))
		case *scriptLifetimeCallback:
			_, _ = callback.invokeNamed(callbackContext, "&callback", ObjectValue(task.handle))
		default:
			_, _ = task.callback.Invoke(callbackContext, ObjectValue(task.handle))
		}
	}
	task.complete()
}

// invocationError is published before done closes and is used only by the
// synchronous bridge call. wait deliberately returns $null for socket workers,
// including callbacks that failed, matching SocketObject's Thread contract.
func (task *sleepSocketTask) flowError() error {
	if task == nil {
		return nil
	}
	<-task.done
	return task.invocationError
}

func (task *sleepSocketTask) complete() {
	task.once.Do(func() {
		if task.releaseContext != nil {
			task.releaseContext()
		}
		task.mu.Lock()
		task.finished = true
		closing := task.closing
		task.mu.Unlock()
		if closing {
			task.state.unregister(task)
		}
		close(task.done)
	})
}

func (task *sleepSocketTask) runConnect() error {
	host := task.host
	if host == "" {
		host = sleepSocketDefaultHost
	}
	dialer := net.Dialer{}
	if task.timeout > 0 {
		dialer.Timeout = time.Duration(task.timeout) * time.Millisecond
	}
	if task.laddrSet {
		if task.lport < 0 || task.lport > 65_535 {
			return fmt.Errorf("java.lang.IllegalArgumentException: port out of range:%d", task.lport)
		}
		localHost := task.laddr
		if localHost == "" {
			localHost = sleepSocketDefaultHost
		}
		localIP, err := sleepSocketResolveIP(task.ctx, localHost)
		if err != nil {
			if task.ctx.Err() != nil {
				return task.ctx.Err()
			}
			return errors.New("java.net.SocketException: Unresolved address")
		}
		dialer.LocalAddr = &net.TCPAddr{IP: localIP.IP, Port: int(task.lport), Zone: localIP.Zone}
	}
	if task.port < 0 || task.port > 65_535 {
		return fmt.Errorf("java.lang.IllegalArgumentException: port out of range:%d", task.port)
	}
	if task.timeout < 0 {
		return errors.New("java.lang.IllegalArgumentException: connect: timeout can't be negative")
	}

	address := net.JoinHostPort(sleepSocketUnwrapIPLiteral(host), strconv.Itoa(int(task.port)))
	conn, err := dialer.DialContext(task.ctx, "tcp", address)
	if err != nil {
		return sleepJavaConnectError(task.host, err)
	}
	if !task.adopt(conn) {
		return task.ctx.Err()
	}
	if task.linger < 0 {
		return errors.New("java.lang.IllegalArgumentException: invalid value for SO_LINGER")
	}
	if err := setSleepSocketLinger(conn, task.linger); err != nil {
		return sleepJavaSocketError(err)
	}
	if !task.attach(conn) {
		return task.ctx.Err()
	}
	return nil
}

func (task *sleepSocketTask) runListen() error {
	listener, err := task.state.listener(task.ctx, task.port, task.laddr, task.laddrSet, task.backlog)
	if err != nil {
		return err
	}
	if task.timeout < 0 {
		return errors.New("java.lang.IllegalArgumentException: timeout < 0")
	}
	conn, err := listener.accept(task.ctx, task.timeout)
	if err != nil {
		if task.ctx.Err() != nil {
			return task.ctx.Err()
		}
		return sleepJavaAcceptError(err)
	}
	if !task.adopt(conn) {
		return task.ctx.Err()
	}
	if task.linger < 0 {
		return errors.New("java.lang.IllegalArgumentException: invalid value for SO_LINGER")
	}
	if err := setSleepSocketLinger(conn, task.linger); err != nil {
		return sleepJavaSocketError(err)
	}
	if remote, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
		task.peer.Set(String(sleepJavaHostAddress(remote.IP, remote.Zone)))
	} else if conn.RemoteAddr() != nil {
		task.peer.Set(String(conn.RemoteAddr().String()))
	}
	if !task.attach(conn) {
		return task.ctx.Err()
	}
	return nil
}

func (task *sleepSocketTask) adopt(conn net.Conn) bool {
	task.mu.Lock()
	if task.closing {
		task.mu.Unlock()
		_ = conn.Close()
		return false
	}
	task.conn = conn
	task.connCloser = &sleepOnceCloser{target: &sleepSocketConnCloser{task: task, conn: conn}}
	task.mu.Unlock()
	return true
}

func (task *sleepSocketTask) attach(conn net.Conn) bool {
	task.mu.Lock()
	closer := task.connCloser
	if task.closing || task.conn != conn || closer == nil {
		task.mu.Unlock()
		if closer != nil {
			_ = closer.Close()
		} else {
			_ = conn.Close()
		}
		return false
	}
	task.handle.readMu.Lock()
	task.handle.writeMu.Lock()
	task.handle.mu.Lock()
	task.handle.reader = bufio.NewReaderSize(conn, sleepIOReadBufferSize)
	task.handle.readSource = conn
	task.handle.writer = conn
	// Adoption transfers the connection to this task-owned coordinator before
	// linger/address setup. Reuse that exact closer after attachment so closef,
	// cancellation, and handle teardown cannot bypass or nest ownership.
	task.handle.readCloser = closer
	task.handle.writeClose = closer
	task.handle.ownRead = true
	task.handle.ownWrite = true
	task.handle.textDecoder.reset(sleepCharsetUTF8)
	task.handle.textEncoder.reset(sleepCharsetUTF8)
	task.attached = true
	task.handle.mu.Unlock()
	task.handle.writeMu.Unlock()
	task.handle.readMu.Unlock()
	task.mu.Unlock()
	return true
}

type sleepSocketConnCloser struct {
	task *sleepSocketTask
	conn net.Conn
}

func (closer *sleepSocketConnCloser) Close() error {
	if closer == nil {
		return nil
	}
	if closer.conn != nil {
		_ = closer.conn.Close()
	}
	if closer.task != nil {
		closer.task.connectionClosed()
	}
	return nil
}

func (task *sleepSocketTask) connectionClosed() {
	if task == nil {
		return
	}
	task.mu.Lock()
	task.conn = nil
	task.attached = false
	task.closing = true
	finished := task.finished
	task.mu.Unlock()
	if finished {
		task.state.unregister(task)
	}
}

// closeHandle mirrors closef(SocketObject): closing an already-connected
// handle closes the accepted/dialed socket, while closing a callback-mode
// placeholder does not cancel its pending connect/accept operation. Once a
// failed placeholder has completed, closef can release its bookkeeping without
// changing any observable I/O or wait behavior.
func (task *sleepSocketTask) closeHandle() {
	if task == nil {
		return
	}
	task.mu.Lock()
	conn := task.conn
	closer := task.connCloser
	attached := task.attached
	finished := task.finished
	if conn == nil && finished {
		task.closing = true
	}
	task.mu.Unlock()
	if conn == nil || closer == nil {
		if finished {
			task.state.unregister(task)
		}
		return
	}
	if attached {
		// Close the transport before waiting for the handle's write lock. A
		// blocked socket write owns writeMu and is released by conn.Close.
		_ = closer.Close()
		_ = task.handle.close()
		return
	}
	_ = closer.Close()
}

func (task *sleepSocketTask) cancelAndClose() {
	if task == nil {
		return
	}
	task.mu.Lock()
	task.closing = true
	conn := task.conn
	closer := task.connCloser
	attached := task.attached
	finished := task.finished
	task.mu.Unlock()
	task.cancel(context.Canceled)
	if task.releaseContext != nil {
		task.releaseContext()
	}
	if conn != nil && closer != nil {
		if attached {
			// Closing the transport first breaks the blocked-write -> writeMu ->
			// handle.close cycle during terminal Script/Runtime cleanup.
			_ = closer.Close()
			_ = task.handle.close()
		} else {
			_ = closer.Close()
		}
	}
	if finished {
		task.state.unregister(task)
	}
}

func (task *sleepSocketTask) join(ctx context.Context) error {
	if task == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-task.done:
		return nil
	}
}

func (task *sleepSocketTask) wait(ctx context.Context, runtime *Runtime, invocation Invocation) (Value, error) {
	if task == nil || task.callback == nil {
		return Null(), nil
	}
	timeout := int64(0)
	if len(invocation.Arguments) > 1 {
		timeout = sleepSocketInt64(invocation.Arg(1))
	}
	// Thread.join is only called while the worker is alive. A completed socket
	// worker therefore ignores even a negative timeout on repeated wait calls.
	select {
	case <-task.done:
		return Null(), nil
	default:
	}
	if timeout < 0 {
		return runtime.flagSourceError(invocation, errors.New("java.lang.IllegalArgumentException: timeout value is negative"))
	}
	if timeout == 0 {
		select {
		case <-ctx.Done():
			return Null(), ctx.Err()
		case <-task.done:
			return Null(), nil
		}
	}
	if timeout > int64((time.Duration(1<<63-1))/time.Millisecond) {
		// Thread.join accepts every nonnegative long. Values beyond Go's
		// time.Duration range are indistinguishable from an unbounded wait for
		// this process, while context cancellation remains available to hosts.
		select {
		case <-ctx.Done():
			return Null(), ctx.Err()
		case <-task.done:
			return Null(), nil
		}
	}
	timer := time.NewTimer(time.Duration(timeout) * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return Null(), ctx.Err()
	case <-task.done:
		return Null(), nil
	case <-timer.C:
		select {
		case <-task.done:
			return Null(), nil
		default:
		}
		return runtime.flagSourceError(invocation, errors.New("java.io.IOException: wait on object timed out"))
	}
}

func (state *sleepSocketState) listener(
	ctx context.Context,
	port int32,
	laddr string,
	laddrSet bool,
	backlog int32,
) (*sleepSocketListener, error) {
	state.mu.Lock()
	if state.closed {
		state.mu.Unlock()
		return nil, context.Canceled
	}
	if cached := state.listeners[port]; cached != nil {
		state.mu.Unlock()
		return cached, nil
	}
	state.mu.Unlock()

	var address *net.TCPAddr
	if laddrSet {
		host := laddr
		if host == "" {
			host = sleepSocketDefaultHost
		}
		resolved, err := sleepSocketResolveIP(ctx, host)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, sleepJavaListenUnknownHost(laddr)
		}
		address = &net.TCPAddr{IP: resolved.IP, Port: int(port), Zone: resolved.Zone}
	} else {
		address = &net.TCPAddr{Port: int(port)}
	}
	if port < 0 || port > 65_535 {
		return nil, fmt.Errorf("java.lang.IllegalArgumentException: Port value out of range: %d", port)
	}
	listener, err := net.ListenTCP("tcp", address)
	if err != nil {
		return nil, sleepJavaListenError(err)
	}
	// java.net.ServerSocket treats backlog <= 0 as its implementation default.
	// The portable net package does not expose listen(2)'s backlog, so positive
	// values are retained as accepted syntax but use Go's platform default too.
	_ = backlog
	entry := &sleepSocketListener{
		requestedPort: port,
		listener:      listener,
		acceptor:      newSleepSocketAcceptor(listener),
	}
	state.mu.Lock()
	if state.closed {
		state.mu.Unlock()
		_ = entry.close()
		return nil, context.Canceled
	}
	if cached := state.listeners[port]; cached != nil {
		state.mu.Unlock()
		_ = entry.close()
		return cached, nil
	}
	state.listeners[port] = entry
	state.mu.Unlock()
	return entry, nil
}

func (listener *sleepSocketListener) accept(ctx context.Context, timeout int32) (net.Conn, error) {
	if listener == nil || listener.acceptor == nil {
		return nil, net.ErrClosed
	}
	return listener.acceptor.accept(ctx, timeout)
}

func (listener *sleepSocketListener) close() error {
	if listener == nil || listener.acceptor == nil {
		return nil
	}
	return listener.acceptor.close()
}

func (state *sleepSocketState) release(port int32) {
	if state == nil {
		return
	}
	state.mu.Lock()
	listener := state.listeners[port]
	delete(state.listeners, port)
	state.mu.Unlock()
	if listener != nil {
		_ = listener.close()
	}
}

func (state *sleepSocketState) shutdown() []*sleepSocketTask {
	if state == nil {
		return nil
	}
	state.mu.Lock()
	state.closed = true
	listeners := make([]*sleepSocketListener, 0, len(state.listeners))
	for _, listener := range state.listeners {
		listeners = append(listeners, listener)
	}
	state.listeners = make(map[int32]*sleepSocketListener)
	tasks := make([]*sleepSocketTask, 0, len(state.tasks))
	for _, task := range state.tasks {
		tasks = append(tasks, task)
	}
	state.mu.Unlock()
	for _, task := range tasks {
		task.cancelAndClose()
	}
	for _, listener := range listeners {
		_ = listener.close()
	}
	return tasks
}

func setSleepSocketLinger(conn net.Conn, linger int32) error {
	tcp, ok := conn.(*net.TCPConn)
	if !ok {
		return nil
	}
	if linger > 65_535 {
		linger = 65_535
	}
	return tcp.SetLinger(int(linger))
}

func sleepJavaHostAddress(ip net.IP, zone string) string {
	if ipv4 := ip.To4(); ipv4 != nil {
		return ipv4.String()
	}
	ipv6 := ip.To16()
	if ipv6 == nil {
		return ip.String()
	}
	parts := make([]string, 8)
	for index := range parts {
		parts[index] = strconv.FormatUint(uint64(binary.BigEndian.Uint16(ipv6[index*2:])), 16)
	}
	address := strings.Join(parts, ":")
	if zone != "" {
		address += "%" + zone
	}
	return address
}

func sleepSocketResolveIP(ctx context.Context, host string) (*net.IPAddr, error) {
	literal := sleepSocketUnwrapIPLiteral(host)
	zone := ""
	if separator := strings.LastIndexByte(literal, '%'); separator > 0 {
		zone = literal[separator+1:]
		literal = literal[:separator]
	}
	if ip := net.ParseIP(literal); ip != nil {
		return &net.IPAddr{IP: ip, Zone: zone}, nil
	}
	if strings.HasPrefix(host, "[") {
		return nil, &net.AddrError{Err: "invalid IPv6 address literal", Addr: host}
	}
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	if len(addresses) == 0 {
		return nil, &net.DNSError{Name: host, Err: "no such host"}
	}
	return &addresses[0], nil
}

func sleepSocketUnwrapIPLiteral(host string) string {
	if len(host) >= 2 && host[0] == '[' && host[len(host)-1] == ']' {
		inside := host[1 : len(host)-1]
		if strings.Contains(inside, ":") {
			return inside
		}
	}
	return host
}

// StringValue delegates to Integer.parseInt/Long.parseLong, while Java object
// scalars use Integer.decode/Long.decode. Keep that distinction here instead
// of Value.Int32/Int64's intentionally friendlier general-purpose coercion.
func sleepSocketInt32(value Value) int32 {
	switch value.kind {
	case KindInt:
		return value.data.(int32)
	case KindLong:
		return int32(value.data.(int64))
	case KindDouble:
		number := value.data.(float64)
		switch {
		case math.IsNaN(number):
			return 0
		case number >= float64(math.MaxInt32):
			return math.MaxInt32
		case number <= float64(math.MinInt32):
			return math.MinInt32
		default:
			return int32(number)
		}
	case KindString:
		parsed, err := strconv.ParseInt(value.data.(string), 10, 32)
		if err == nil {
			return int32(parsed)
		}
		return 0
	case KindObject, KindFunction:
		parsed, ok := sleepSocketDecodeInt(value.String(), 32)
		if ok {
			return int32(parsed)
		}
	}
	return 0
}

func sleepSocketInt64(value Value) int64 {
	switch value.kind {
	case KindInt:
		return int64(value.data.(int32))
	case KindLong:
		return value.data.(int64)
	case KindDouble:
		number := value.data.(float64)
		switch {
		case math.IsNaN(number):
			return 0
		case number >= float64(math.MaxInt64):
			return math.MaxInt64
		case number <= float64(math.MinInt64):
			return math.MinInt64
		default:
			return int64(number)
		}
	case KindString:
		parsed, err := strconv.ParseInt(value.data.(string), 10, 64)
		if err == nil {
			return parsed
		}
		return 0
	case KindObject, KindFunction:
		parsed, ok := sleepSocketDecodeInt(value.String(), 64)
		if ok {
			return parsed
		}
	}
	return 0
}

func sleepSocketDecodeInt(text string, bits int) (int64, bool) {
	switch text {
	case "":
		return 0, true
	case "true":
		return 1, true
	case "false":
		return 0, true
	}
	sign := ""
	unsigned := text
	if strings.HasPrefix(unsigned, "+") || strings.HasPrefix(unsigned, "-") {
		sign, unsigned = unsigned[:1], unsigned[1:]
	}
	if strings.HasPrefix(unsigned, "#") {
		unsigned = unsigned[1:]
		if strings.HasPrefix(unsigned, "+") || strings.HasPrefix(unsigned, "-") {
			return 0, false
		}
		parsed, err := strconv.ParseInt(sign+unsigned, 16, bits)
		return parsed, err == nil
	}
	base := 10
	switch {
	case strings.HasPrefix(unsigned, "0x") || strings.HasPrefix(unsigned, "0X"):
		base = 16
		unsigned = unsigned[2:]
	case len(unsigned) > 1 && unsigned[0] == '0':
		base = 8
		unsigned = unsigned[1:]
	}
	if unsigned == "" || strings.HasPrefix(unsigned, "+") || strings.HasPrefix(unsigned, "-") {
		return 0, false
	}
	parsed, err := strconv.ParseInt(sign+unsigned, base, bits)
	return parsed, err == nil
}

func isSleepSocketCancellation(ctx context.Context, err error) bool {
	return ctx != nil && ctx.Err() != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
}

func sleepJavaConnectError(host string, err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var dnsError *net.DNSError
	if errors.As(err, &dnsError) {
		return fmt.Errorf("java.net.UnknownHostException: %s", host)
	}
	var addressError *net.AddrError
	if errors.As(err, &addressError) {
		return fmt.Errorf("java.net.UnknownHostException: %s", host)
	}
	if errors.Is(err, syscall.EADDRINUSE) || errors.Is(err, syscall.EADDRNOTAVAIL) {
		return fmt.Errorf("java.net.BindException: %s", sleepSocketErrorDetail(err))
	}
	// Windows reports Winsock's WSAECONNREFUSED (10061), while
	// syscall.ECONNREFUSED there is a distinct synthetic value.
	if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.Errno(10_061)) {
		return errors.New("java.net.ConnectException: Connection refused")
	}
	if errors.Is(err, syscall.ENETUNREACH) || errors.Is(err, syscall.EHOSTUNREACH) {
		return errors.New("java.net.NoRouteToHostException: No route to host")
	}
	if netError, ok := err.(net.Error); ok && netError.Timeout() {
		return errors.New("java.net.SocketTimeoutException: connect timed out")
	}
	return sleepJavaSocketError(err)
}

func sleepJavaListenError(err error) error {
	if errors.Is(err, syscall.EADDRINUSE) {
		return errors.New("java.net.BindException: Address already in use")
	}
	return sleepJavaSocketError(err)
}

func sleepJavaListenUnknownHost(host string) error {
	if strings.HasPrefix(host, "[") && !strings.HasSuffix(host, "]") {
		return fmt.Errorf("java.net.UnknownHostException: %s: invalid IPv6 address literal", host)
	}
	return fmt.Errorf("java.net.UnknownHostException: %s", host)
}

func sleepJavaAcceptError(err error) error {
	if errors.Is(err, net.ErrClosed) {
		return errors.New("java.net.SocketException: Socket closed")
	}
	if netError, ok := err.(net.Error); ok && netError.Timeout() {
		return errors.New("java.net.SocketTimeoutException: Accept timed out")
	}
	return sleepJavaSocketError(err)
}

func sleepJavaSocketError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, net.ErrClosed) {
		return errors.New("java.net.SocketException: Socket closed")
	}
	var operationError *net.OpError
	if errors.As(err, &operationError) && operationError.Err != nil {
		err = operationError.Err
	}
	return fmt.Errorf("java.net.SocketException: %s", err)
}

func sleepSocketErrorDetail(err error) error {
	var operationError *net.OpError
	if errors.As(err, &operationError) && operationError.Err != nil {
		return operationError.Err
	}
	return err
}
