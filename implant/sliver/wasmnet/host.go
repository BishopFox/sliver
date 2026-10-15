//go:build (windows && (amd64 || arm64 || 386)) || (darwin && (arm64 || amd64)) || (linux && (arm64 || amd64 || 386))

package wasmnet

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTIBILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

const (
	wasmNetworkMaxHandles    = 128
	wasmNetworkMaxOperations = 256
	wasmNetworkMaxBuffer     = 8 << 20
	wasmNetworkMaxAddress    = 4 << 10
	wasmNetworkMaxPending    = 64 << 20
)

// These are WASI Preview 1 errno values, not the host operating system's
// syscall numbers.
const (
	wasiErrnoSuccess              uint32 = 0
	wasiErrnoAccess               uint32 = 2
	wasiErrnoAddrInUse            uint32 = 3
	wasiErrnoAddrNotAvailable     uint32 = 4
	wasiErrnoAddressFamily        uint32 = 5
	wasiErrnoAgain                uint32 = 6
	wasiErrnoBadFile              uint32 = 8
	wasiErrnoCanceled             uint32 = 11
	wasiErrnoConnAborted          uint32 = 13
	wasiErrnoConnRefused          uint32 = 14
	wasiErrnoConnReset            uint32 = 15
	wasiErrnoFault                uint32 = 21
	wasiErrnoHostUnreachable      uint32 = 23
	wasiErrnoInterrupted          uint32 = 27
	wasiErrnoInvalid              uint32 = 28
	wasiErrnoIO                   uint32 = 29
	wasiErrnoIsConnected          uint32 = 30
	wasiErrnoMessageSize          uint32 = 35
	wasiErrnoNetworkDown          uint32 = 38
	wasiErrnoNetworkReset         uint32 = 39
	wasiErrnoNetworkUnreachable   uint32 = 40
	wasiErrnoNoBuffer             uint32 = 42
	wasiErrnoNoEntry              uint32 = 44
	wasiErrnoNoProtocolOption     uint32 = 50
	wasiErrnoNotImplemented       uint32 = 52
	wasiErrnoNotConnected         uint32 = 53
	wasiErrnoNotSocket            uint32 = 57
	wasiErrnoNotSupported         uint32 = 58
	wasiErrnoPipe                 uint32 = 64
	wasiErrnoProtocolNotSupported uint32 = 66
	wasiErrnoRange                uint32 = 68
	wasiErrnoTimedOut             uint32 = 73
)

type wasmNetworkSocket struct {
	network  string
	conn     net.Conn
	listener net.Listener
	packet   net.PacketConn
}

func (s *wasmNetworkSocket) close() error {
	switch {
	case s.listener != nil:
		return s.listener.Close()
	case s.conn != nil:
		return s.conn.Close()
	case s.packet != nil:
		return s.packet.Close()
	default:
		return nil
	}
}

func (s *wasmNetworkSocket) localAddr() net.Addr {
	switch {
	case s.conn != nil:
		return s.conn.LocalAddr()
	case s.listener != nil:
		return s.listener.Addr()
	case s.packet != nil:
		return s.packet.LocalAddr()
	default:
		return nil
	}
}

type wasmNetworkResult struct {
	data   []byte
	addr   string
	n      uint32
	handle uint32
	errno  uint32
}

type wasmNetworkOperation struct {
	ctx         context.Context
	cancel      context.CancelFunc
	done        chan struct{}
	reservation uint64
	release     sync.Once
	workerDone  sync.Once
	result      wasmNetworkResult
}

// Host implements the Sliver WASI networking host module.
type Host struct {
	ctx    context.Context
	cancel context.CancelFunc

	mu         sync.Mutex
	closed     bool
	nextHandle uint32
	nextOp     uint32
	handles    map[uint32]*wasmNetworkSocket
	operations map[uint32]*wasmNetworkOperation
	pending    uint64
	workers    uint32
}

func newWasmNetwork(parent context.Context) *Host {
	ctx, cancel := context.WithCancel(parent)
	return &Host{
		ctx:        ctx,
		cancel:     cancel,
		nextHandle: 1,
		nextOp:     1,
		handles:    map[uint32]*wasmNetworkSocket{},
		operations: map[uint32]*wasmNetworkOperation{},
	}
}

// New returns a networking Host derived from parent.
func New(parent context.Context) *Host {
	return newWasmNetwork(parent)
}

// Instantiate registers and instantiates the networking host module in runtime.
func (w *Host) Instantiate(ctx context.Context, runtime wazero.Runtime) (api.Module, error) {
	builder := runtime.NewHostModuleBuilder(ModuleName)
	builder.NewFunctionBuilder().WithFunc(w.hostDialStart).Export("dial_start")
	builder.NewFunctionBuilder().WithFunc(w.hostListen).Export("listen")
	builder.NewFunctionBuilder().WithFunc(w.hostAcceptStart).Export("accept_start")
	builder.NewFunctionBuilder().WithFunc(w.hostReadStart).Export("read_start")
	builder.NewFunctionBuilder().WithFunc(w.hostWriteStart).Export("write_start")
	builder.NewFunctionBuilder().WithFunc(w.hostRecvFromStart).Export("recv_from_start")
	builder.NewFunctionBuilder().WithFunc(w.hostSendToStart).Export("send_to_start")
	builder.NewFunctionBuilder().WithFunc(w.hostOperationPoll).Export("op_poll")
	builder.NewFunctionBuilder().WithFunc(w.hostOperationCancel).Export("op_cancel")
	builder.NewFunctionBuilder().WithFunc(w.hostShutdown).Export("shutdown")
	builder.NewFunctionBuilder().WithFunc(w.hostClose).Export("close")
	builder.NewFunctionBuilder().WithFunc(w.hostGetAddr).Export("get_addr")
	builder.NewFunctionBuilder().WithFunc(w.hostSetDeadline).Export("set_deadline")
	builder.NewFunctionBuilder().WithFunc(w.hostLookupStart).Export("lookup_start")
	return builder.Instantiate(ctx)
}

// Close cancels in-flight operations and closes every socket owned by the host.
func (w *Host) Close() error {
	w.cancel()

	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	handles := make([]*wasmNetworkSocket, 0, len(w.handles))
	for _, socket := range w.handles {
		handles = append(handles, socket)
	}
	operations := make([]*wasmNetworkOperation, 0, len(w.operations))
	for _, operation := range w.operations {
		operations = append(operations, operation)
	}
	w.handles = map[uint32]*wasmNetworkSocket{}
	w.operations = map[uint32]*wasmNetworkOperation{}
	w.mu.Unlock()

	for _, operation := range operations {
		operation.cancel()
		w.releaseOperationReservation(operation)
	}
	var closeErr error
	for _, socket := range handles {
		if err := socket.close(); err != nil && !errors.Is(err, net.ErrClosed) {
			closeErr = errors.Join(closeErr, err)
		}
	}
	return closeErr
}

func (w *Host) addHandle(socket *wasmNetworkSocket) (uint32, uint32) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return 0, wasiErrnoCanceled
	}
	if len(w.handles) >= wasmNetworkMaxHandles {
		return 0, wasiErrnoNoBuffer
	}
	handle := w.nextHandleIDLocked()
	w.handles[handle] = socket
	return handle, wasiErrnoSuccess
}

func (w *Host) nextHandleIDLocked() uint32 {
	for {
		handle := w.nextHandle
		w.nextHandle++
		if w.nextHandle == 0 {
			w.nextHandle = 1
		}
		if _, exists := w.handles[handle]; handle != 0 && !exists {
			return handle
		}
	}
}

func (w *Host) socket(handle uint32) (*wasmNetworkSocket, uint32) {
	w.mu.Lock()
	defer w.mu.Unlock()
	socket, ok := w.handles[handle]
	if !ok {
		return nil, wasiErrnoBadFile
	}
	return socket, wasiErrnoSuccess
}

func (w *Host) removeHandle(handle uint32) (*wasmNetworkSocket, uint32) {
	w.mu.Lock()
	defer w.mu.Unlock()
	socket, ok := w.handles[handle]
	if !ok {
		return nil, wasiErrnoBadFile
	}
	delete(w.handles, handle)
	return socket, wasiErrnoSuccess
}

func (w *Host) reserveOperation(reservation uint64) (uint32, *wasmNetworkOperation, uint32) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return 0, nil, wasiErrnoCanceled
	}
	if len(w.operations) >= wasmNetworkMaxOperations ||
		w.workers >= wasmNetworkMaxOperations ||
		reservation > wasmNetworkMaxPending ||
		w.pending > wasmNetworkMaxPending-reservation {
		return 0, nil, wasiErrnoNoBuffer
	}
	ctx, cancel := context.WithCancel(w.ctx)
	opID := w.nextOperationIDLocked()
	operation := &wasmNetworkOperation{
		ctx:         ctx,
		cancel:      cancel,
		done:        make(chan struct{}),
		reservation: reservation,
	}
	w.operations[opID] = operation
	w.pending += reservation
	w.workers++
	return opID, operation, wasiErrnoSuccess
}

func (w *Host) nextOperationIDLocked() uint32 {
	for {
		opID := w.nextOp
		w.nextOp++
		if w.nextOp == 0 {
			w.nextOp = 1
		}
		if _, exists := w.operations[opID]; opID != 0 && !exists {
			return opID
		}
	}
}

func (w *Host) launchOperation(
	opID uint32,
	operation *wasmNetworkOperation,
	fn func(context.Context) wasmNetworkResult,
) {
	go func() {
		defer w.releaseOperationWorker(operation)
		result := fn(operation.ctx)
		var orphanHandle uint32
		retained := false
		w.mu.Lock()
		if current, ok := w.operations[opID]; ok && current == operation {
			operation.result = result
			close(operation.done)
			retained = true
		} else {
			orphanHandle = result.handle
		}
		w.mu.Unlock()
		if orphanHandle != 0 {
			_ = w.closeHandle(orphanHandle)
		}
		if !retained {
			w.releaseOperationReservation(operation)
		}
	}()
}

func (w *Host) startOperation(reservation uint64, fn func(context.Context) wasmNetworkResult) (uint32, uint32) {
	opID, operation, errno := w.reserveOperation(reservation)
	if errno != 0 {
		return 0, errno
	}
	w.launchOperation(opID, operation, fn)
	return opID, wasiErrnoSuccess
}

func (w *Host) pollOperation(opID uint32) (wasmNetworkResult, bool, uint32) {
	w.mu.Lock()
	defer w.mu.Unlock()
	operation, ok := w.operations[opID]
	if !ok {
		return wasmNetworkResult{}, false, wasiErrnoBadFile
	}
	select {
	case <-operation.done:
		return operation.result, true, wasiErrnoSuccess
	default:
		return wasmNetworkResult{}, false, wasiErrnoAgain
	}
}

func (w *Host) finishOperation(opID uint32) {
	w.mu.Lock()
	operation, ok := w.operations[opID]
	if ok {
		delete(w.operations, opID)
	}
	w.mu.Unlock()
	if ok {
		operation.cancel()
		w.releaseOperationReservation(operation)
	}
}

func (w *Host) cancelOperation(opID uint32) uint32 {
	w.mu.Lock()
	operation, ok := w.operations[opID]
	var resultHandle uint32
	completed := false
	if ok {
		delete(w.operations, opID)
		select {
		case <-operation.done:
			completed = true
			resultHandle = operation.result.handle
		default:
		}
	}
	w.mu.Unlock()
	if !ok {
		return wasiErrnoBadFile
	}
	operation.cancel()
	if resultHandle != 0 {
		_ = w.closeHandle(resultHandle)
	}
	if completed {
		w.releaseOperationReservation(operation)
	}
	return wasiErrnoSuccess
}

func (w *Host) discardReservedOperation(opID uint32, operation *wasmNetworkOperation) {
	w.mu.Lock()
	if current, ok := w.operations[opID]; ok && current == operation {
		delete(w.operations, opID)
	}
	w.mu.Unlock()
	operation.cancel()
	w.releaseOperationReservation(operation)
	w.releaseOperationWorker(operation)
}

func (w *Host) releaseOperationReservation(operation *wasmNetworkOperation) {
	operation.release.Do(func() {
		w.mu.Lock()
		w.pending -= operation.reservation
		w.mu.Unlock()
	})
}

func (w *Host) releaseOperationWorker(operation *wasmNetworkOperation) {
	operation.workerDone.Do(func() {
		w.mu.Lock()
		w.workers--
		w.mu.Unlock()
	})
}

func (w *Host) dial(ctx context.Context, network, local, remote string, deadline int64) wasmNetworkResult {
	if errno := validateWasmNetwork(network, false); errno != 0 {
		return wasmNetworkResult{errno: errno}
	}
	if remote == "" || len(remote) > wasmNetworkMaxAddress || len(local) > wasmNetworkMaxAddress {
		return wasmNetworkResult{errno: wasiErrnoInvalid}
	}
	ctx, cancel := wasmNetworkDeadlineContext(ctx, deadline)
	defer cancel()

	dialer := net.Dialer{}
	if local != "" {
		addr, err := resolveWasmNetworkAddr(network, local)
		if err != nil {
			return wasmNetworkResult{errno: wasmNetworkErrno(err)}
		}
		dialer.LocalAddr = addr
	}
	conn, err := dialer.DialContext(ctx, network, remote)
	if err != nil {
		return wasmNetworkResult{errno: wasmNetworkErrno(err)}
	}
	socket := &wasmNetworkSocket{network: network, conn: conn}
	if packet, ok := conn.(net.PacketConn); ok {
		socket.packet = packet
	}
	handle, errno := w.addHandle(socket)
	if errno != 0 {
		_ = conn.Close()
		return wasmNetworkResult{errno: errno}
	}
	return wasmNetworkResult{handle: handle}
}

func (w *Host) listen(network, local string) (uint32, uint32) {
	if errno := validateWasmNetwork(network, true); errno != 0 {
		return 0, errno
	}
	if len(local) > wasmNetworkMaxAddress {
		return 0, wasiErrnoInvalid
	}

	var socket *wasmNetworkSocket
	if strings.HasPrefix(network, "tcp") {
		listener, err := net.Listen(network, local)
		if err != nil {
			return 0, wasmNetworkErrno(err)
		}
		socket = &wasmNetworkSocket{network: network, listener: listener}
	} else {
		packet, err := net.ListenPacket(network, local)
		if err != nil {
			return 0, wasmNetworkErrno(err)
		}
		socket = &wasmNetworkSocket{network: network, packet: packet}
	}
	handle, errno := w.addHandle(socket)
	if errno != 0 {
		_ = socket.close()
	}
	return handle, errno
}

func (w *Host) accept(ctx context.Context, handle uint32, _ int64) wasmNetworkResult {
	socket, errno := w.socket(handle)
	if errno != 0 {
		return wasmNetworkResult{errno: errno}
	}
	if socket.listener == nil {
		return wasmNetworkResult{errno: wasiErrnoNotSocket}
	}
	stopInterrupt := wasmNetworkInterrupt(ctx, func() {
		_ = setListenerDeadline(socket.listener, time.Now().UnixNano())
	})
	defer stopInterrupt()
	conn, err := socket.listener.Accept()
	if err != nil {
		if ctx.Err() != nil {
			return wasmNetworkResult{errno: wasmNetworkErrno(ctx.Err())}
		}
		return wasmNetworkResult{errno: wasmNetworkErrno(err)}
	}
	accepted := &wasmNetworkSocket{network: socket.network, conn: conn}
	if packet, ok := conn.(net.PacketConn); ok {
		accepted.packet = packet
	}
	acceptedHandle, errno := w.addHandle(accepted)
	if errno != 0 {
		_ = conn.Close()
		return wasmNetworkResult{errno: errno}
	}
	return wasmNetworkResult{handle: acceptedHandle}
}

func (w *Host) read(ctx context.Context, handle, maxLen uint32, _ int64) wasmNetworkResult {
	if maxLen > wasmNetworkMaxBuffer {
		return wasmNetworkResult{errno: wasiErrnoMessageSize}
	}
	socket, errno := w.socket(handle)
	if errno != 0 {
		return wasmNetworkResult{errno: errno}
	}
	if socket.conn == nil {
		return wasmNetworkResult{errno: wasiErrnoNotConnected}
	}
	stopInterrupt := wasmNetworkInterrupt(ctx, func() {
		_ = socket.conn.SetReadDeadline(time.Now())
	})
	defer stopInterrupt()
	data := make([]byte, maxLen)
	n, err := socket.conn.Read(data)
	if ctx.Err() != nil && n == 0 {
		err = ctx.Err()
	}
	if errors.Is(err, io.EOF) {
		err = nil
	}
	return wasmNetworkResult{data: data[:n], n: uint32(n), errno: wasmNetworkErrno(err)}
}

func (w *Host) write(ctx context.Context, handle uint32, data []byte, _ int64) wasmNetworkResult {
	socket, errno := w.socket(handle)
	if errno != 0 {
		return wasmNetworkResult{errno: errno}
	}
	if socket.conn == nil {
		return wasmNetworkResult{errno: wasiErrnoNotConnected}
	}
	stopInterrupt := wasmNetworkInterrupt(ctx, func() {
		_ = socket.conn.SetWriteDeadline(time.Now())
	})
	defer stopInterrupt()
	n, err := socket.conn.Write(data)
	if ctx.Err() != nil && n == 0 {
		err = ctx.Err()
	}
	return wasmNetworkResult{n: uint32(n), errno: wasmNetworkErrno(err)}
}

func (w *Host) recvFrom(ctx context.Context, handle, maxLen uint32, _ int64) wasmNetworkResult {
	if maxLen > wasmNetworkMaxBuffer {
		return wasmNetworkResult{errno: wasiErrnoMessageSize}
	}
	socket, errno := w.socket(handle)
	if errno != 0 {
		return wasmNetworkResult{errno: errno}
	}
	if socket.packet == nil {
		return wasmNetworkResult{errno: wasiErrnoNotSocket}
	}
	stopInterrupt := wasmNetworkInterrupt(ctx, func() {
		_ = socket.packet.SetReadDeadline(time.Now())
	})
	defer stopInterrupt()
	data := make([]byte, maxLen)
	n, addr, err := socket.packet.ReadFrom(data)
	if ctx.Err() != nil && n == 0 {
		err = ctx.Err()
	}
	addrString := ""
	if addr != nil {
		addrString = addr.String()
	}
	return wasmNetworkResult{data: data[:n], addr: addrString, n: uint32(n), errno: wasmNetworkErrno(err)}
}

func (w *Host) sendTo(ctx context.Context, handle uint32, data []byte, remote string, _ int64) wasmNetworkResult {
	if len(remote) > wasmNetworkMaxAddress {
		return wasmNetworkResult{errno: wasiErrnoInvalid}
	}
	socket, errno := w.socket(handle)
	if errno != 0 {
		return wasmNetworkResult{errno: errno}
	}
	if socket.packet == nil {
		return wasmNetworkResult{errno: wasiErrnoNotSocket}
	}
	stopInterrupt := wasmNetworkInterrupt(ctx, func() {
		_ = socket.packet.SetWriteDeadline(time.Now())
	})
	defer stopInterrupt()
	addr, err := resolveWasmNetworkAddr(socket.network, remote)
	if err != nil {
		return wasmNetworkResult{errno: wasmNetworkErrno(err)}
	}
	n, err := socket.packet.WriteTo(data, addr)
	if ctx.Err() != nil && n == 0 {
		err = ctx.Err()
	}
	return wasmNetworkResult{n: uint32(n), errno: wasmNetworkErrno(err)}
}

func (w *Host) lookup(ctx context.Context, network, name string, deadline int64) wasmNetworkResult {
	switch network {
	case "ip", "ip4", "ip6":
	default:
		return wasmNetworkResult{errno: wasiErrnoAddressFamily}
	}
	if name == "" || len(name) > wasmNetworkMaxAddress {
		return wasmNetworkResult{errno: wasiErrnoInvalid}
	}
	ctx, cancel := wasmNetworkDeadlineContext(ctx, deadline)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupNetIP(ctx, network, name)
	if err != nil {
		return wasmNetworkResult{errno: wasmNetworkErrno(err)}
	}
	if len(addrs) == 0 {
		return wasmNetworkResult{errno: wasiErrnoNoEntry}
	}
	values := make([]string, 0, len(addrs))
	resultLength := 0
	for _, addr := range addrs {
		value := addr.String()
		if len(values) > 0 {
			resultLength++
		}
		resultLength += len(value)
		if resultLength > wasmNetworkMaxAddress {
			return wasmNetworkResult{errno: wasiErrnoMessageSize}
		}
		values = append(values, value)
	}
	return wasmNetworkResult{addr: strings.Join(values, "\x00")}
}

func (w *Host) hostDialStart(
	_ context.Context,
	module api.Module,
	networkPtr, networkLen, localPtr, localLen, remotePtr, remoteLen uint32,
	deadline int64,
	opOut uint32,
) uint32 {
	network, errno := wasmNetworkReadString(module, networkPtr, networkLen)
	if errno != 0 {
		return errno
	}
	local, errno := wasmNetworkReadString(module, localPtr, localLen)
	if errno != 0 {
		return errno
	}
	remote, errno := wasmNetworkReadString(module, remotePtr, remoteLen)
	if errno != 0 {
		return errno
	}
	op, errno := w.startOperation(0, func(ctx context.Context) wasmNetworkResult {
		return w.dial(ctx, network, local, remote, deadline)
	})
	if errno != 0 {
		return errno
	}
	if !module.Memory().WriteUint32Le(opOut, op) {
		_ = w.cancelOperation(op)
		return wasiErrnoFault
	}
	return wasiErrnoSuccess
}

func (w *Host) hostListen(
	_ context.Context,
	module api.Module,
	networkPtr, networkLen, localPtr, localLen, handleOut uint32,
) uint32 {
	network, errno := wasmNetworkReadString(module, networkPtr, networkLen)
	if errno != 0 {
		return errno
	}
	local, errno := wasmNetworkReadString(module, localPtr, localLen)
	if errno != 0 {
		return errno
	}
	handle, errno := w.listen(network, local)
	if errno != 0 {
		return errno
	}
	if !module.Memory().WriteUint32Le(handleOut, handle) {
		_ = w.closeHandle(handle)
		return wasiErrnoFault
	}
	return wasiErrnoSuccess
}

func (w *Host) hostAcceptStart(
	_ context.Context,
	module api.Module,
	handle uint32,
	deadline int64,
	opOut uint32,
) uint32 {
	if _, errno := w.socket(handle); errno != 0 {
		return errno
	}
	op, errno := w.startOperation(0, func(ctx context.Context) wasmNetworkResult {
		return w.accept(ctx, handle, deadline)
	})
	if errno != 0 {
		return errno
	}
	if !module.Memory().WriteUint32Le(opOut, op) {
		_ = w.cancelOperation(op)
		return wasiErrnoFault
	}
	return wasiErrnoSuccess
}

func (w *Host) hostReadStart(
	_ context.Context,
	module api.Module,
	handle, maxLen uint32,
	deadline int64,
	opOut uint32,
) uint32 {
	if maxLen > wasmNetworkMaxBuffer {
		return wasiErrnoMessageSize
	}
	if _, errno := w.socket(handle); errno != 0 {
		return errno
	}
	op, errno := w.startOperation(uint64(maxLen), func(ctx context.Context) wasmNetworkResult {
		return w.read(ctx, handle, maxLen, deadline)
	})
	if errno != 0 {
		return errno
	}
	if !module.Memory().WriteUint32Le(opOut, op) {
		_ = w.cancelOperation(op)
		return wasiErrnoFault
	}
	return wasiErrnoSuccess
}

func (w *Host) hostWriteStart(
	_ context.Context,
	module api.Module,
	handle, dataPtr, dataLen uint32,
	deadline int64,
	opOut uint32,
) uint32 {
	if dataLen > wasmNetworkMaxBuffer {
		return wasiErrnoMessageSize
	}
	if _, errno := w.socket(handle); errno != 0 {
		return errno
	}
	op, operation, errno := w.reserveOperation(uint64(dataLen))
	if errno != 0 {
		return errno
	}
	data, errno := wasmNetworkReadBytes(module, dataPtr, dataLen)
	if errno != 0 {
		w.discardReservedOperation(op, operation)
		return errno
	}
	w.launchOperation(op, operation, func(ctx context.Context) wasmNetworkResult {
		return w.write(ctx, handle, data, deadline)
	})
	if !module.Memory().WriteUint32Le(opOut, op) {
		_ = w.cancelOperation(op)
		return wasiErrnoFault
	}
	return wasiErrnoSuccess
}

func (w *Host) hostRecvFromStart(
	_ context.Context,
	module api.Module,
	handle, maxLen uint32,
	deadline int64,
	opOut uint32,
) uint32 {
	if maxLen > wasmNetworkMaxBuffer {
		return wasiErrnoMessageSize
	}
	if _, errno := w.socket(handle); errno != 0 {
		return errno
	}
	op, errno := w.startOperation(uint64(maxLen), func(ctx context.Context) wasmNetworkResult {
		return w.recvFrom(ctx, handle, maxLen, deadline)
	})
	if errno != 0 {
		return errno
	}
	if !module.Memory().WriteUint32Le(opOut, op) {
		_ = w.cancelOperation(op)
		return wasiErrnoFault
	}
	return wasiErrnoSuccess
}

func (w *Host) hostSendToStart(
	_ context.Context,
	module api.Module,
	handle, dataPtr, dataLen, remotePtr, remoteLen uint32,
	deadline int64,
	opOut uint32,
) uint32 {
	if dataLen > wasmNetworkMaxBuffer {
		return wasiErrnoMessageSize
	}
	remote, errno := wasmNetworkReadString(module, remotePtr, remoteLen)
	if errno != 0 {
		return errno
	}
	if _, errno = w.socket(handle); errno != 0 {
		return errno
	}
	op, operation, errno := w.reserveOperation(uint64(dataLen))
	if errno != 0 {
		return errno
	}
	data, errno := wasmNetworkReadBytes(module, dataPtr, dataLen)
	if errno != 0 {
		w.discardReservedOperation(op, operation)
		return errno
	}
	w.launchOperation(op, operation, func(ctx context.Context) wasmNetworkResult {
		return w.sendTo(ctx, handle, data, remote, deadline)
	})
	if !module.Memory().WriteUint32Le(opOut, op) {
		_ = w.cancelOperation(op)
		return wasiErrnoFault
	}
	return wasiErrnoSuccess
}

func (w *Host) hostLookupStart(
	_ context.Context,
	module api.Module,
	networkPtr, networkLen, namePtr, nameLen uint32,
	deadline int64,
	opOut uint32,
) uint32 {
	network, errno := wasmNetworkReadString(module, networkPtr, networkLen)
	if errno != 0 {
		return errno
	}
	name, errno := wasmNetworkReadString(module, namePtr, nameLen)
	if errno != 0 {
		return errno
	}
	op, errno := w.startOperation(wasmNetworkMaxAddress, func(ctx context.Context) wasmNetworkResult {
		return w.lookup(ctx, network, name, deadline)
	})
	if errno != 0 {
		return errno
	}
	if !module.Memory().WriteUint32Le(opOut, op) {
		_ = w.cancelOperation(op)
		return wasiErrnoFault
	}
	return wasiErrnoSuccess
}

//nolint:gocyclo // Polling is a protocol state transition with defensive cleanup at each fault.
func (w *Host) hostOperationPoll(
	_ context.Context,
	module api.Module,
	opID, dataPtr, dataCap, dataLenOut, addrPtr, addrCap, addrLenOut, handleOut uint32,
) uint32 {
	result, done, errno := w.pollOperation(opID)
	if errno != 0 || !done {
		return errno
	}
	memory := module.Memory()
	if !memory.WriteUint32Le(dataLenOut, uint32(len(result.data))) ||
		!memory.WriteUint32Le(addrLenOut, uint32(len(result.addr))) ||
		!memory.WriteUint32Le(handleOut, result.handle) {
		w.finishOperation(opID)
		if result.handle != 0 {
			_ = w.closeHandle(result.handle)
		}
		return wasiErrnoFault
	}
	if uint32(len(result.data)) > dataCap || uint32(len(result.addr)) > addrCap {
		return wasiErrnoNoBuffer
	}
	if len(result.data) > 0 && !memory.Write(dataPtr, result.data) {
		w.finishOperation(opID)
		if result.handle != 0 {
			_ = w.closeHandle(result.handle)
		}
		return wasiErrnoFault
	}
	if result.addr != "" && !memory.WriteString(addrPtr, result.addr) {
		w.finishOperation(opID)
		if result.handle != 0 {
			_ = w.closeHandle(result.handle)
		}
		return wasiErrnoFault
	}
	if len(result.data) == 0 && !memory.WriteUint32Le(dataLenOut, result.n) {
		w.finishOperation(opID)
		if result.handle != 0 {
			_ = w.closeHandle(result.handle)
		}
		return wasiErrnoFault
	}
	w.finishOperation(opID)
	// EAGAIN and ENOBUFS are reserved by op_poll for pending operations and
	// output-buffer resizing. Do not let a terminal host error masquerade as
	// either protocol state.
	if result.errno == wasiErrnoAgain || result.errno == wasiErrnoNoBuffer {
		return wasiErrnoIO
	}
	return result.errno
}

func (w *Host) hostOperationCancel(_ context.Context, opID uint32) uint32 {
	return w.cancelOperation(opID)
}

func (w *Host) hostShutdown(_ context.Context, handle, how uint32) uint32 {
	socket, errno := w.socket(handle)
	if errno != 0 {
		return errno
	}
	if socket.conn == nil {
		return wasiErrnoNotConnected
	}
	var err error
	switch how {
	case 0:
		if conn, ok := socket.conn.(interface{ CloseRead() error }); ok {
			err = conn.CloseRead()
		} else {
			return wasiErrnoNotSupported
		}
	case 1:
		if conn, ok := socket.conn.(interface{ CloseWrite() error }); ok {
			err = conn.CloseWrite()
		} else {
			return wasiErrnoNotSupported
		}
	case 2:
		err = socket.conn.Close()
	default:
		return wasiErrnoInvalid
	}
	return wasmNetworkErrno(err)
}

func (w *Host) closeHandle(handle uint32) uint32 {
	socket, errno := w.removeHandle(handle)
	if errno != 0 {
		return errno
	}
	return wasmNetworkErrno(socket.close())
}

func (w *Host) hostClose(_ context.Context, handle uint32) uint32 {
	return w.closeHandle(handle)
}

func (w *Host) hostGetAddr(
	_ context.Context,
	module api.Module,
	handle, which, addrPtr, addrCap, addrLenOut uint32,
) uint32 {
	socket, errno := w.socket(handle)
	if errno != 0 {
		return errno
	}
	var addr net.Addr
	switch which {
	case 0:
		addr = socket.localAddr()
	case 1:
		if socket.conn != nil {
			addr = socket.conn.RemoteAddr()
		}
	default:
		return wasiErrnoInvalid
	}
	if addr == nil {
		return wasiErrnoNotConnected
	}
	value := addr.String()
	if !module.Memory().WriteUint32Le(addrLenOut, uint32(len(value))) {
		return wasiErrnoFault
	}
	if uint32(len(value)) > addrCap {
		return wasiErrnoNoBuffer
	}
	if !module.Memory().WriteString(addrPtr, value) {
		return wasiErrnoFault
	}
	return wasiErrnoSuccess
}

func (w *Host) hostSetDeadline(_ context.Context, handle, which uint32, deadline int64) uint32 {
	socket, errno := w.socket(handle)
	if errno != 0 {
		return errno
	}
	var err error
	switch {
	case socket.conn != nil:
		err = setConnDeadline(socket.conn, which, deadline)
	case socket.packet != nil:
		err = setPacketDeadline(socket.packet, which, deadline)
	case socket.listener != nil && (which == 0 || which == 1):
		err = setListenerDeadline(socket.listener, deadline)
	default:
		return wasiErrnoNotSupported
	}
	return wasmNetworkErrno(err)
}

func wasmNetworkReadString(module api.Module, ptr, length uint32) (string, uint32) {
	if length > wasmNetworkMaxAddress {
		return "", wasiErrnoInvalid
	}
	if length == 0 {
		return "", wasiErrnoSuccess
	}
	value, ok := module.Memory().Read(ptr, length)
	if !ok {
		return "", wasiErrnoFault
	}
	return string(value), wasiErrnoSuccess
}

func wasmNetworkReadBytes(module api.Module, ptr, length uint32) ([]byte, uint32) {
	if length > wasmNetworkMaxBuffer {
		return nil, wasiErrnoMessageSize
	}
	if length == 0 {
		return nil, wasiErrnoSuccess
	}
	value, ok := module.Memory().Read(ptr, length)
	if !ok {
		return nil, wasiErrnoFault
	}
	return append([]byte(nil), value...), wasiErrnoSuccess
}

func validateWasmNetwork(network string, listen bool) uint32 {
	switch network {
	case "tcp", "tcp4", "tcp6", "udp", "udp4", "udp6":
		return wasiErrnoSuccess
	default:
		if listen && (network == "") {
			return wasiErrnoInvalid
		}
		return wasiErrnoProtocolNotSupported
	}
}

func resolveWasmNetworkAddr(network, address string) (net.Addr, error) {
	switch {
	case strings.HasPrefix(network, "tcp"):
		return net.ResolveTCPAddr(network, address)
	case strings.HasPrefix(network, "udp"):
		return net.ResolveUDPAddr(network, address)
	default:
		return nil, syscall.EAFNOSUPPORT
	}
}

func wasmNetworkDeadlineContext(ctx context.Context, deadline int64) (context.Context, context.CancelFunc) {
	if deadline <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithDeadline(ctx, time.Unix(0, deadline))
}

func wasmNetworkDeadline(deadline int64) time.Time {
	if deadline <= 0 {
		return time.Time{}
	}
	return time.Unix(0, deadline)
}

func wasmNetworkInterrupt(ctx context.Context, interrupt func()) func() {
	finished := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		select {
		case <-ctx.Done():
			interrupt()
		case <-finished:
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			close(finished)
			<-stopped
		})
	}
}

func setConnDeadline(conn net.Conn, which uint32, deadline int64) error {
	value := wasmNetworkDeadline(deadline)
	switch which {
	case 0:
		return conn.SetDeadline(value)
	case 1:
		return conn.SetReadDeadline(value)
	case 2:
		return conn.SetWriteDeadline(value)
	default:
		return syscall.EINVAL
	}
}

func setPacketDeadline(packet net.PacketConn, which uint32, deadline int64) error {
	value := wasmNetworkDeadline(deadline)
	switch which {
	case 0:
		return packet.SetDeadline(value)
	case 1:
		return packet.SetReadDeadline(value)
	case 2:
		return packet.SetWriteDeadline(value)
	default:
		return syscall.EINVAL
	}
}

func setListenerDeadline(listener net.Listener, deadline int64) error {
	deadlineListener, ok := listener.(interface{ SetDeadline(time.Time) error })
	if !ok {
		return syscall.ENOTSUP
	}
	return deadlineListener.SetDeadline(wasmNetworkDeadline(deadline))
}

func wasmNetworkErrno(err error) uint32 {
	if err == nil || errors.Is(err, io.EOF) {
		return wasiErrnoSuccess
	}
	switch {
	case errors.Is(err, context.Canceled):
		return wasiErrnoCanceled
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, os.ErrDeadlineExceeded):
		return wasiErrnoTimedOut
	case errors.Is(err, net.ErrClosed):
		return wasiErrnoBadFile
	}
	var dnsError *net.DNSError
	if errors.As(err, &dnsError) && dnsError.IsNotFound {
		return wasiErrnoNoEntry
	}
	if errno, ok := wasmNetworkSystemErrno(err); ok {
		return errno
	}
	var netError net.Error
	if errors.As(err, &netError) {
		if netError.Timeout() {
			return wasiErrnoTimedOut
		}
		return wasiErrnoHostUnreachable
	}
	return wasiErrnoIO
}
