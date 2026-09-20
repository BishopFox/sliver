//go:build (windows && (amd64 || arm64 || 386)) || (darwin && (arm64 || amd64)) || (linux && (arm64 || amd64 || 386))

package wasmnet

import (
	"bytes"
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tetratelabs/wazero"
)

func TestWasmNetworkHostModule(t *testing.T) {
	ctx := context.Background()
	runtime := wazero.NewRuntime(ctx)
	t.Cleanup(func() {
		mustNoError(t, runtime.Close(ctx))
	})
	network := newWasmNetwork(ctx)
	t.Cleanup(func() {
		mustNoError(t, network.Close())
	})

	module, err := network.Instantiate(ctx, runtime)
	mustNoError(t, err)
	assert.Equal(t, ModuleName, module.Name())
}

func TestWasmNetworkTCP(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	mustNoError(t, err)
	t.Cleanup(func() {
		_ = listener.Close()
	})
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer func() {
			_ = conn.Close()
		}()
		buffer := make([]byte, 32)
		n, readErr := conn.Read(buffer)
		if readErr == nil {
			_, _ = conn.Write(bytes.ToUpper(buffer[:n]))
		}
	}()

	network := newWasmNetwork(context.Background())
	t.Cleanup(func() {
		mustNoError(t, network.Close())
	})
	deadline := time.Now().Add(5 * time.Second).UnixNano()
	dialResult := runWasmNetworkOperation(t, network, func(ctx context.Context) wasmNetworkResult {
		return network.dial(ctx, "tcp4", "", listener.Addr().String(), deadline)
	})
	mustZero(t, dialResult.errno)
	assert.NotZero(t, dialResult.handle)

	writeResult := runWasmNetworkOperation(t, network, func(ctx context.Context) wasmNetworkResult {
		return network.write(ctx, dialResult.handle, []byte("hello"), deadline)
	})
	mustZero(t, writeResult.errno)
	assert.Equal(t, uint32(5), writeResult.n)

	readResult := runWasmNetworkOperation(t, network, func(ctx context.Context) wasmNetworkResult {
		return network.read(ctx, dialResult.handle, 32, deadline)
	})
	mustZero(t, readResult.errno)
	assert.Equal(t, []byte("HELLO"), readResult.data)
}

func TestWasmNetworkConnectedUDP(t *testing.T) {
	server, err := net.ListenPacket("udp4", "127.0.0.1:0")
	mustNoError(t, err)
	t.Cleanup(func() {
		_ = server.Close()
	})
	go func() {
		buffer := make([]byte, 32)
		n, addr, readErr := server.ReadFrom(buffer)
		if readErr == nil {
			_, _ = server.WriteTo(bytes.ToUpper(buffer[:n]), addr)
		}
	}()

	network := newWasmNetwork(context.Background())
	t.Cleanup(func() {
		mustNoError(t, network.Close())
	})
	deadline := time.Now().Add(5 * time.Second).UnixNano()
	dialResult := runWasmNetworkOperation(t, network, func(ctx context.Context) wasmNetworkResult {
		return network.dial(ctx, "udp4", "", server.LocalAddr().String(), deadline)
	})
	mustZero(t, dialResult.errno)

	writeResult := runWasmNetworkOperation(t, network, func(ctx context.Context) wasmNetworkResult {
		return network.write(ctx, dialResult.handle, []byte("datagram"), deadline)
	})
	mustZero(t, writeResult.errno)
	assert.Equal(t, uint32(8), writeResult.n)

	readResult := runWasmNetworkOperation(t, network, func(ctx context.Context) wasmNetworkResult {
		return network.read(ctx, dialResult.handle, 32, deadline)
	})
	mustZero(t, readResult.errno)
	assert.Equal(t, []byte("DATAGRAM"), readResult.data)
}

func TestWasmNetworkUnconnectedUDP(t *testing.T) {
	network := newWasmNetwork(context.Background())
	t.Cleanup(func() {
		mustNoError(t, network.Close())
	})
	handle, errno := network.listen("udp4", "127.0.0.1:0")
	mustZero(t, errno)
	socket, errno := network.socket(handle)
	mustZero(t, errno)

	client, err := net.ListenPacket("udp4", "127.0.0.1:0")
	mustNoError(t, err)
	t.Cleanup(func() {
		_ = client.Close()
	})
	_, err = client.WriteTo([]byte("ping"), socket.localAddr())
	mustNoError(t, err)

	deadline := time.Now().Add(5 * time.Second).UnixNano()
	recvResult := runWasmNetworkOperation(t, network, func(ctx context.Context) wasmNetworkResult {
		return network.recvFrom(ctx, handle, 32, deadline)
	})
	mustZero(t, recvResult.errno)
	assert.Equal(t, []byte("ping"), recvResult.data)
	assert.Equal(t, client.LocalAddr().String(), recvResult.addr)

	sendResult := runWasmNetworkOperation(t, network, func(ctx context.Context) wasmNetworkResult {
		return network.sendTo(ctx, handle, []byte("pong"), recvResult.addr, deadline)
	})
	mustZero(t, sendResult.errno)
	assert.Equal(t, uint32(4), sendResult.n)

	mustNoError(t, client.SetReadDeadline(time.Now().Add(5*time.Second)))
	buffer := make([]byte, 32)
	n, _, err := client.ReadFrom(buffer)
	mustNoError(t, err)
	assert.Equal(t, []byte("pong"), buffer[:n])
}

func TestWasmNetworkLookupAndCleanup(t *testing.T) {
	network := newWasmNetwork(context.Background())
	deadline := time.Now().Add(5 * time.Second).UnixNano()
	result := runWasmNetworkOperation(t, network, func(ctx context.Context) wasmNetworkResult {
		return network.lookup(ctx, "ip", "localhost", deadline)
	})
	mustZero(t, result.errno)
	assert.NotEmpty(t, result.addr)

	handle, errno := network.listen("tcp4", "127.0.0.1:0")
	mustZero(t, errno)
	assert.NotZero(t, handle)
	mustNoError(t, network.Close())
	_, errno = network.socket(handle)
	assert.Equal(t, uint32(wasiErrnoBadFile), errno)
	_, errno = network.listen("tcp4", "127.0.0.1:0")
	assert.Equal(t, uint32(wasiErrnoCanceled), errno)
}

func TestWasmNetworkOperationBudget(t *testing.T) {
	network := newWasmNetwork(context.Background())
	t.Cleanup(func() {
		mustNoError(t, network.Close())
	})

	operationCount := wasmNetworkMaxPending / wasmNetworkMaxBuffer
	operationIDs := make([]uint32, 0, operationCount)
	releaseWorkers := make(chan struct{})
	var workers sync.WaitGroup
	for range operationCount {
		workers.Add(1)
		opID, errno := network.startOperation(wasmNetworkMaxBuffer, func(ctx context.Context) wasmNetworkResult {
			defer workers.Done()
			<-ctx.Done()
			<-releaseWorkers
			return wasmNetworkResult{errno: wasiErrnoCanceled}
		})
		mustZero(t, errno)
		operationIDs = append(operationIDs, opID)
	}
	_, errno := network.startOperation(1, func(context.Context) wasmNetworkResult {
		return wasmNetworkResult{}
	})
	assert.Equal(t, uint32(wasiErrnoNoBuffer), errno)

	for _, opID := range operationIDs {
		mustZero(t, network.cancelOperation(opID))
	}
	_, errno = network.startOperation(1, func(context.Context) wasmNetworkResult {
		return wasmNetworkResult{}
	})
	assert.Equal(t, uint32(wasiErrnoNoBuffer), errno)

	network.mu.Lock()
	pending := network.pending
	operationLength := len(network.operations)
	network.mu.Unlock()
	assert.Equal(t, uint64(wasmNetworkMaxPending), pending)
	assert.Zero(t, operationLength)

	close(releaseWorkers)
	workers.Wait()
	deadline := time.Now().Add(5 * time.Second)
	for {
		network.mu.Lock()
		pending = network.pending
		network.mu.Unlock()
		if pending == 0 || time.Now().After(deadline) {
			break
		}
		time.Sleep(time.Millisecond)
	}
	assert.Zero(t, pending)

	opID, errno := network.startOperation(1, func(ctx context.Context) wasmNetworkResult {
		<-ctx.Done()
		return wasmNetworkResult{errno: wasiErrnoCanceled}
	})
	mustZero(t, errno)
	mustZero(t, network.cancelOperation(opID))
}

func TestWasmNetworkIDWrapSkipsLiveEntries(t *testing.T) {
	network := newWasmNetwork(context.Background())
	t.Cleanup(func() {
		mustNoError(t, network.Close())
	})

	firstHandle, errno := network.addHandle(&wasmNetworkSocket{})
	mustZero(t, errno)
	assert.Equal(t, uint32(1), firstHandle)
	network.mu.Lock()
	network.nextHandle = ^uint32(0)
	network.mu.Unlock()
	lastHandle, errno := network.addHandle(&wasmNetworkSocket{})
	mustZero(t, errno)
	assert.Equal(t, ^uint32(0), lastHandle)
	wrappedHandle, errno := network.addHandle(&wasmNetworkSocket{})
	mustZero(t, errno)
	assert.Equal(t, uint32(2), wrappedHandle)

	var workers sync.WaitGroup
	startBlockingOperation := func() uint32 {
		workers.Add(1)
		opID, startErrno := network.startOperation(0, func(ctx context.Context) wasmNetworkResult {
			defer workers.Done()
			<-ctx.Done()
			return wasmNetworkResult{errno: wasiErrnoCanceled}
		})
		mustZero(t, startErrno)
		return opID
	}
	firstOperation := startBlockingOperation()
	assert.Equal(t, uint32(1), firstOperation)
	network.mu.Lock()
	network.nextOp = ^uint32(0)
	network.mu.Unlock()
	lastOperation := startBlockingOperation()
	assert.Equal(t, ^uint32(0), lastOperation)
	wrappedOperation := startBlockingOperation()
	assert.Equal(t, uint32(2), wrappedOperation)
	for _, opID := range []uint32{firstOperation, lastOperation, wrappedOperation} {
		mustZero(t, network.cancelOperation(opID))
	}
	workers.Wait()
}

func TestWasmNetworkOperationLimitTracksCanceledWorkers(t *testing.T) {
	network := newWasmNetwork(context.Background())
	t.Cleanup(func() {
		mustNoError(t, network.Close())
	})

	releaseWorkers := make(chan struct{})
	operationIDs := make([]uint32, 0, wasmNetworkMaxOperations)
	var workers sync.WaitGroup
	for range wasmNetworkMaxOperations {
		workers.Add(1)
		opID, errno := network.startOperation(0, func(ctx context.Context) wasmNetworkResult {
			defer workers.Done()
			<-ctx.Done()
			<-releaseWorkers
			return wasmNetworkResult{errno: wasiErrnoCanceled}
		})
		mustZero(t, errno)
		operationIDs = append(operationIDs, opID)
	}
	for _, opID := range operationIDs {
		mustZero(t, network.cancelOperation(opID))
	}
	_, errno := network.startOperation(0, func(context.Context) wasmNetworkResult {
		return wasmNetworkResult{}
	})
	assert.Equal(t, uint32(wasiErrnoNoBuffer), errno)

	close(releaseWorkers)
	workers.Wait()
	deadline := time.Now().Add(5 * time.Second)
	for {
		network.mu.Lock()
		activeWorkers := network.workers
		network.mu.Unlock()
		if activeWorkers == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("active worker count did not drain: %d", activeWorkers)
		}
		time.Sleep(time.Millisecond)
	}
	opID, errno := network.startOperation(0, func(ctx context.Context) wasmNetworkResult {
		<-ctx.Done()
		return wasmNetworkResult{errno: wasiErrnoCanceled}
	})
	mustZero(t, errno)
	mustZero(t, network.cancelOperation(opID))
}

func TestWasmNetworkCancelCompletedHandle(t *testing.T) {
	network := newWasmNetwork(context.Background())
	t.Cleanup(func() {
		mustNoError(t, network.Close())
	})
	handle, errno := network.listen("tcp4", "127.0.0.1:0")
	mustZero(t, errno)

	opID, errno := network.startOperation(0, func(context.Context) wasmNetworkResult {
		return wasmNetworkResult{handle: handle}
	})
	mustZero(t, errno)
	network.mu.Lock()
	operation := network.operations[opID]
	network.mu.Unlock()
	select {
	case <-operation.done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for completed handle operation")
	}

	mustZero(t, network.cancelOperation(opID))
	_, errno = network.socket(handle)
	assert.Equal(t, uint32(wasiErrnoBadFile), errno)
}

func TestWasmNetworkInterruptStopWaits(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	entered := make(chan struct{})
	release := make(chan struct{})
	stop := wasmNetworkInterrupt(ctx, func() {
		close(entered)
		<-release
	})
	cancel()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("interrupt callback did not start")
	}

	returned := make(chan struct{})
	go func() {
		stop()
		close(returned)
	}()
	select {
	case <-returned:
		t.Fatal("interrupt stop returned before the callback exited")
	case <-time.After(25 * time.Millisecond):
	}
	close(release)
	select {
	case <-returned:
	case <-time.After(5 * time.Second):
		t.Fatal("interrupt stop did not return after the callback exited")
	}
}

func TestWasmNetworkNXDomainErrno(t *testing.T) {
	errno := wasmNetworkErrno(&net.DNSError{
		Err:        "no such host",
		Name:       "does-not-exist.invalid",
		IsNotFound: true,
	})
	assert.Equal(t, uint32(wasiErrnoNoEntry), errno)
}

func runWasmNetworkOperation(
	t *testing.T,
	network *Host,
	fn func(context.Context) wasmNetworkResult,
) wasmNetworkResult {
	t.Helper()
	opID, errno := network.startOperation(0, fn)
	mustZero(t, errno)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		result, done, pollErrno := network.pollOperation(opID)
		if pollErrno == wasiErrnoAgain {
			time.Sleep(time.Millisecond)
			continue
		}
		mustZero(t, pollErrno)
		assert.True(t, done)
		network.finishOperation(opID)
		return result
	}
	t.Fatal("timed out waiting for Wasm network operation")
	return wasmNetworkResult{}
}

func mustNoError(t *testing.T, err error, context ...string) {
	t.Helper()
	if len(context) > 0 {
		if !assert.NoError(t, err, context[0]) {
			t.FailNow()
		}
		return
	}
	if !assert.NoError(t, err) {
		t.FailNow()
	}
}

func mustZero(t *testing.T, value uint32, context ...string) {
	t.Helper()
	if len(context) > 0 {
		if !assert.Zero(t, value, context[0]) {
			t.FailNow()
		}
		return
	}
	if !assert.Zero(t, value) {
		t.FailNow()
	}
}
