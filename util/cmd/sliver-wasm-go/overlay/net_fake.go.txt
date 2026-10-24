// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build wasip1

package net

import (
	"context"
	"io"
	"os"
	"runtime"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	wasiNetAddressBufferSize = 256
	wasiNetMaxUint32         = uint64(1<<32 - 1)

	wasiNetDeadlineAll   = 0
	wasiNetDeadlineRead  = 1
	wasiNetDeadlineWrite = 2

	wasiNetAddressLocal  = 0
	wasiNetAddressRemote = 1

	wasiNetShutdownRead  = 0
	wasiNetShutdownWrite = 1
	wasiNetShutdownBoth  = 2
)

// socket returns a network file descriptor backed by Sliver's networking host
// module. The host performs potentially blocking calls asynchronously; the
// guest polls operation handles so that other Go goroutines can keep running.
func socket(ctx context.Context, network string, family, sotype, proto int, ipv6only bool, laddr, raddr sockaddr, ctrlCtxFn func(context.Context, string, string, syscall.RawConn) error) (*netFD, error) {
	if ctrlCtxFn != nil {
		return nil, os.NewSyscallError("socket", syscall.ENOTSUP)
	}
	switch network {
	case "tcp", "tcp4", "tcp6":
		if sotype != syscall.SOCK_STREAM {
			return nil, os.NewSyscallError("socket", syscall.EPROTOTYPE)
		}
	case "udp", "udp4", "udp6":
		if sotype != syscall.SOCK_DGRAM {
			return nil, os.NewSyscallError("socket", syscall.EPROTOTYPE)
		}
	default:
		return nil, os.NewSyscallError("socket", syscall.EAFNOSUPPORT)
	}

	local := wasiNetSockaddrString(laddr)
	remote := wasiNetSockaddrString(raddr)
	var handle uint32
	if raddr == nil {
		if errno := wasiNetListen(network, local, &handle); errno != 0 {
			return nil, wasiNetError("listen", errno)
		}
	} else {
		var operation uint32
		errno := wasiNetDialStart(network, local, remote, wasiNetContextDeadline(ctx), &operation)
		if errno != 0 {
			return nil, wasiNetError("connect", errno)
		}
		result, err := wasiNetWaitOperation(ctx, operation, nil, wasiNetAddressBufferSize)
		if err != nil {
			return nil, err
		}
		handle = result.handle
		if handle == 0 {
			return nil, os.NewSyscallError("connect", syscall.EIO)
		}
	}

	fd, err := wasiNetNewFD(handle, network, family, sotype, raddr != nil)
	if err != nil {
		_ = wasiNetClose(handle)
		return nil, err
	}
	return fd, nil
}

func wasiNetNewFD(handle uint32, network string, family, sotype int, connected bool) (*netFD, error) {
	localText, err := wasiNetGetAddress(handle, wasiNetAddressLocal)
	if err != nil {
		return nil, err
	}
	local, err := wasiNetParseAddress(network, localText)
	if err != nil {
		return nil, err
	}

	var remote Addr
	if connected {
		remoteText, err := wasiNetGetAddress(handle, wasiNetAddressRemote)
		if err != nil {
			return nil, err
		}
		remote, err = wasiNetParseAddress(network, remoteText)
		if err != nil {
			return nil, err
		}
	}

	fd := &netFD{
		family:      family,
		sotype:      sotype,
		isConnected: connected,
		net:         network,
		laddr:       local,
		raddr:       remote,
	}
	fd.fakeNetFD = &fakeNetFD{
		fd:     fd,
		handle: handle,
	}
	fd.setAddr(local, remote)
	return fd, nil
}

type fakeNetFD struct {
	fd     *netFD
	handle uint32

	closed        atomic.Bool
	readDeadline  atomic.Int64
	writeDeadline atomic.Int64
}

func (ffd *fakeNetFD) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if ffd.closed.Load() {
		return 0, ErrClosed
	}
	if uint64(len(p)) > wasiNetMaxUint32 {
		return 0, os.NewSyscallError("read", syscall.EINVAL)
	}

	var operation uint32
	errno := wasiNetReadStart(ffd.handle, uint32(len(p)), ffd.readDeadline.Load(), &operation)
	if errno != 0 {
		return 0, wasiNetError("read", errno)
	}
	result, err := wasiNetWaitOperation(context.Background(), operation, p, 0)
	if len(result.data) > len(p) {
		return 0, os.NewSyscallError("read", syscall.EIO)
	}
	n := copy(p, result.data)
	if err != nil {
		return n, err
	}
	if n == 0 && ffd.fd.sotype == syscall.SOCK_STREAM {
		return 0, io.EOF
	}
	return n, nil
}

func (ffd *fakeNetFD) Write(p []byte) (int, error) {
	if len(p) == 0 && ffd.fd.sotype == syscall.SOCK_STREAM {
		return 0, nil
	}
	if ffd.closed.Load() {
		return 0, ErrClosed
	}
	if uint64(len(p)) > wasiNetMaxUint32 {
		return 0, os.NewSyscallError("write", syscall.EINVAL)
	}

	var operation uint32
	errno := wasiNetWriteStart(ffd.handle, wasiNetBytePointer(p), uint32(len(p)), ffd.writeDeadline.Load(), &operation)
	runtime.KeepAlive(p)
	if errno != 0 {
		return 0, wasiNetError("write", errno)
	}
	result, err := wasiNetWaitOperation(context.Background(), operation, nil, 0)
	if result.n > uint32(len(p)) {
		return 0, os.NewSyscallError("write", syscall.EIO)
	}
	return int(result.n), err
}

func (ffd *fakeNetFD) Close() error {
	if !ffd.closed.CompareAndSwap(false, true) {
		return ErrClosed
	}
	runtime.SetFinalizer(ffd.fd, nil)
	return wasiNetError("close", wasiNetClose(ffd.handle))
}

func (ffd *fakeNetFD) closeRead() error {
	if ffd.closed.Load() {
		return ErrClosed
	}
	return wasiNetError("shutdown", wasiNetShutdown(ffd.handle, wasiNetShutdownRead))
}

func (ffd *fakeNetFD) closeWrite() error {
	if ffd.closed.Load() {
		return ErrClosed
	}
	return wasiNetError("shutdown", wasiNetShutdown(ffd.handle, wasiNetShutdownWrite))
}

func (ffd *fakeNetFD) accept(_ Addr) (*netFD, error) {
	if ffd.closed.Load() {
		return nil, ErrClosed
	}
	var operation uint32
	errno := wasiNetAcceptStart(ffd.handle, ffd.readDeadline.Load(), &operation)
	if errno != 0 {
		return nil, wasiNetError("accept", errno)
	}
	result, err := wasiNetWaitOperation(context.Background(), operation, nil, 0)
	if err != nil {
		return nil, err
	}
	if result.handle == 0 {
		return nil, os.NewSyscallError("accept", syscall.EIO)
	}
	fd, err := wasiNetNewFD(result.handle, ffd.fd.net, ffd.fd.family, ffd.fd.sotype, true)
	if err != nil {
		_ = wasiNetClose(result.handle)
		return nil, err
	}
	return fd, nil
}

func (ffd *fakeNetFD) SetDeadline(deadline time.Time) error {
	value := wasiNetTimeDeadline(deadline)
	if errno := wasiNetSetDeadline(ffd.handle, wasiNetDeadlineAll, value); errno != 0 {
		return wasiNetError("setdeadline", errno)
	}
	ffd.readDeadline.Store(value)
	ffd.writeDeadline.Store(value)
	return nil
}

func (ffd *fakeNetFD) SetReadDeadline(deadline time.Time) error {
	value := wasiNetTimeDeadline(deadline)
	if errno := wasiNetSetDeadline(ffd.handle, wasiNetDeadlineRead, value); errno != 0 {
		return wasiNetError("setreaddeadline", errno)
	}
	ffd.readDeadline.Store(value)
	return nil
}

func (ffd *fakeNetFD) SetWriteDeadline(deadline time.Time) error {
	value := wasiNetTimeDeadline(deadline)
	if errno := wasiNetSetDeadline(ffd.handle, wasiNetDeadlineWrite, value); errno != 0 {
		return wasiNetError("setwritedeadline", errno)
	}
	ffd.writeDeadline.Store(value)
	return nil
}

func (ffd *fakeNetFD) readFrom(p []byte) (n int, sa syscall.Sockaddr, err error) {
	n, from, err := ffd.wasiNetRecvFrom(p)
	if err != nil || from == nil {
		return n, nil, err
	}
	sa, addrErr := from.(sockaddr).sockaddr(ffd.fd.family)
	if addrErr != nil {
		return n, nil, addrErr
	}
	return n, sa, nil
}

func (ffd *fakeNetFD) readFromInet4(p []byte, sa *syscall.SockaddrInet4) (int, error) {
	n, from, err := ffd.readFrom(p)
	if err != nil {
		return n, err
	}
	from4, ok := from.(*syscall.SockaddrInet4)
	if !ok {
		return n, os.NewSyscallError("recvfrom", syscall.EAFNOSUPPORT)
	}
	*sa = *from4
	return n, nil
}

func (ffd *fakeNetFD) readFromInet6(p []byte, sa *syscall.SockaddrInet6) (int, error) {
	n, from, err := ffd.readFrom(p)
	if err != nil {
		return n, err
	}
	from6, ok := from.(*syscall.SockaddrInet6)
	if !ok {
		return n, os.NewSyscallError("recvfrom", syscall.EAFNOSUPPORT)
	}
	*sa = *from6
	return n, nil
}

func (ffd *fakeNetFD) wasiNetRecvFrom(p []byte) (int, Addr, error) {
	if ffd.closed.Load() {
		return 0, nil, ErrClosed
	}
	if uint64(len(p)) > wasiNetMaxUint32 {
		return 0, nil, os.NewSyscallError("recvfrom", syscall.EINVAL)
	}

	var operation uint32
	errno := wasiNetRecvFromStart(ffd.handle, uint32(len(p)), ffd.readDeadline.Load(), &operation)
	if errno != 0 {
		return 0, nil, wasiNetError("recvfrom", errno)
	}
	result, err := wasiNetWaitOperation(context.Background(), operation, p, wasiNetAddressBufferSize)
	if len(result.data) > len(p) {
		return 0, nil, os.NewSyscallError("recvfrom", syscall.EIO)
	}
	n := copy(p, result.data)
	if err != nil {
		return n, nil, err
	}
	from, err := wasiNetParseAddress(ffd.fd.net, string(result.address))
	if err != nil {
		return n, nil, err
	}
	return n, from, nil
}

func (ffd *fakeNetFD) readMsg(p []byte, _ []byte, flags int) (n, oobn, retflags int, sa syscall.Sockaddr, err error) {
	if flags != 0 {
		return 0, 0, 0, nil, os.NewSyscallError("recvmsg", syscall.ENOTSUP)
	}
	n, sa, err = ffd.readFrom(p)
	return n, 0, 0, sa, err
}

func (ffd *fakeNetFD) readMsgInet4(p []byte, oob []byte, flags int, sa *syscall.SockaddrInet4) (n, oobn, retflags int, err error) {
	if flags != 0 {
		return 0, 0, 0, os.NewSyscallError("recvmsg", syscall.ENOTSUP)
	}
	n, err = ffd.readFromInet4(p, sa)
	return n, 0, 0, err
}

func (ffd *fakeNetFD) readMsgInet6(p []byte, oob []byte, flags int, sa *syscall.SockaddrInet6) (n, oobn, retflags int, err error) {
	if flags != 0 {
		return 0, 0, 0, os.NewSyscallError("recvmsg", syscall.ENOTSUP)
	}
	n, err = ffd.readFromInet6(p, sa)
	return n, 0, 0, err
}

func (ffd *fakeNetFD) writeTo(p []byte, sa syscall.Sockaddr) (int, error) {
	if ffd.closed.Load() {
		return 0, ErrClosed
	}
	if sa == nil {
		if !ffd.fd.isConnected {
			return 0, os.NewSyscallError("sendto", syscall.EDESTADDRREQ)
		}
		return ffd.Write(p)
	}
	if ffd.fd.isConnected {
		return 0, os.NewSyscallError("sendto", syscall.EISCONN)
	}
	if uint64(len(p)) > wasiNetMaxUint32 {
		return 0, os.NewSyscallError("sendto", syscall.EINVAL)
	}

	remoteAddr := ffd.fd.addrFunc()(sa)
	if remoteAddr == nil {
		return 0, os.NewSyscallError("sendto", syscall.EINVAL)
	}
	var operation uint32
	errno := wasiNetSendToStart(
		ffd.handle,
		wasiNetBytePointer(p),
		uint32(len(p)),
		remoteAddr.String(),
		ffd.writeDeadline.Load(),
		&operation,
	)
	runtime.KeepAlive(p)
	if errno != 0 {
		return 0, wasiNetError("sendto", errno)
	}
	result, err := wasiNetWaitOperation(context.Background(), operation, nil, 0)
	if result.n > uint32(len(p)) {
		return 0, os.NewSyscallError("sendto", syscall.EIO)
	}
	return int(result.n), err
}

func (ffd *fakeNetFD) writeToInet4(p []byte, sa *syscall.SockaddrInet4) (int, error) {
	return ffd.writeTo(p, sa)
}

func (ffd *fakeNetFD) writeToInet6(p []byte, sa *syscall.SockaddrInet6) (int, error) {
	return ffd.writeTo(p, sa)
}

func (ffd *fakeNetFD) writeMsg(p []byte, oob []byte, sa syscall.Sockaddr) (n, oobn int, err error) {
	if len(oob) != 0 {
		return 0, 0, os.NewSyscallError("sendmsg", syscall.ENOTSUP)
	}
	n, err = ffd.writeTo(p, sa)
	return n, 0, err
}

func (ffd *fakeNetFD) writeMsgInet4(p []byte, oob []byte, sa *syscall.SockaddrInet4) (n, oobn int, err error) {
	var sockaddr syscall.Sockaddr
	if sa != nil {
		sockaddr = sa
	}
	return ffd.writeMsg(p, oob, sockaddr)
}

func (ffd *fakeNetFD) writeMsgInet6(p []byte, oob []byte, sa *syscall.SockaddrInet6) (n, oobn int, err error) {
	var sockaddr syscall.Sockaddr
	if sa != nil {
		sockaddr = sa
	}
	return ffd.writeMsg(p, oob, sockaddr)
}

func (ffd *fakeNetFD) dup() (*os.File, error) {
	return nil, os.NewSyscallError("dup", syscall.ENOSYS)
}

func (ffd *fakeNetFD) setReadBuffer(int) error {
	return os.NewSyscallError("setreadbuffer", syscall.ENOTSUP)
}

func (ffd *fakeNetFD) setWriteBuffer(int) error {
	return os.NewSyscallError("setwritebuffer", syscall.ENOTSUP)
}

func (ffd *fakeNetFD) setLinger(int) error {
	return os.NewSyscallError("setlinger", syscall.ENOTSUP)
}

func sysSocket(family, sotype, proto int) (int, error) {
	return 0, os.NewSyscallError("socket", syscall.ENOSYS)
}

type wasiNetOperationResult struct {
	data    []byte
	address []byte
	handle  uint32
	n       uint32
}

func wasiNetWaitOperation(ctx context.Context, operation uint32, data []byte, addressCapacity int) (result wasiNetOperationResult, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	address := make([]byte, addressCapacity)
	complete := false
	defer func() {
		if !complete {
			_ = wasiNetOpCancel(operation)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		var dataLen, addressLen, handle uint32
		errno := wasiNetOpPoll(
			operation,
			wasiNetBytePointer(data),
			uint32(len(data)),
			&dataLen,
			wasiNetBytePointer(address),
			uint32(len(address)),
			&addressLen,
			&handle,
		)
		runtime.KeepAlive(data)
		runtime.KeepAlive(address)
		switch errno {
		case uint32(syscall.EAGAIN):
			runtime.Gosched()
			time.Sleep(time.Millisecond)
			continue
		case uint32(syscall.ENOBUFS):
			resized := false
			if dataLen > uint32(len(data)) {
				data = make([]byte, dataLen)
				resized = true
			}
			if addressLen > uint32(len(address)) {
				address = make([]byte, addressLen)
				resized = true
			}
			if !resized {
				return result, wasiNetError("op_poll", errno)
			}
			continue
		}
		if (len(data) > 0 && dataLen > uint32(len(data))) || addressLen > uint32(len(address)) {
			return result, os.NewSyscallError("op_poll", syscall.EIO)
		}
		complete = true
		var resultData []byte
		if len(data) > 0 {
			resultData = data[:dataLen]
		}
		result = wasiNetOperationResult{
			data:    resultData,
			address: address[:addressLen],
			handle:  handle,
			n:       dataLen,
		}
		return result, wasiNetError("op_poll", errno)
	}
}

func wasiNetGetAddress(handle, which uint32) (string, error) {
	buffer := make([]byte, wasiNetAddressBufferSize)
	for {
		var size uint32
		errno := wasiNetGetAddr(handle, which, wasiNetBytePointer(buffer), uint32(len(buffer)), &size)
		runtime.KeepAlive(buffer)
		switch errno {
		case 0:
			if size > uint32(len(buffer)) {
				return "", os.NewSyscallError("get_addr", syscall.EIO)
			}
			return string(buffer[:size]), nil
		case uint32(syscall.ENOBUFS):
			if size <= uint32(len(buffer)) {
				return "", wasiNetError("get_addr", errno)
			}
			buffer = make([]byte, size)
		default:
			return "", wasiNetError("get_addr", errno)
		}
	}
}

func wasiNetParseAddress(network, address string) (Addr, error) {
	if address == "" {
		return nil, nil
	}
	host, portText, err := SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 0 || port >= 1<<16 {
		return nil, &AddrError{Err: "invalid port", Addr: address}
	}
	host, zone := splitHostZone(host)
	ip := ParseIP(host)
	if host != "" && ip == nil {
		return nil, &AddrError{Err: "invalid IP address", Addr: address}
	}
	switch network {
	case "tcp", "tcp4", "tcp6":
		return &TCPAddr{IP: ip, Port: port, Zone: zone}, nil
	case "udp", "udp4", "udp6":
		return &UDPAddr{IP: ip, Port: port, Zone: zone}, nil
	default:
		return nil, &AddrError{Err: syscall.EAFNOSUPPORT.Error(), Addr: address}
	}
}

func wasiNetSockaddrString(address sockaddr) string {
	switch address := address.(type) {
	case nil:
		return ""
	case *TCPAddr:
		if address == nil {
			return ""
		}
	case *UDPAddr:
		if address == nil {
			return ""
		}
	case *UnixAddr:
		if address == nil {
			return ""
		}
	}
	return address.String()
}

func wasiNetTimeDeadline(deadline time.Time) int64 {
	if deadline.IsZero() {
		return 0
	}
	return deadline.UnixNano()
}

func wasiNetContextDeadline(ctx context.Context) int64 {
	if ctx == nil {
		return 0
	}
	if deadline, ok := ctx.Deadline(); ok {
		return wasiNetTimeDeadline(deadline)
	}
	return 0
}

var wasiNetZeroByte byte

func wasiNetBytePointer(buffer []byte) *byte {
	if len(buffer) == 0 {
		return &wasiNetZeroByte
	}
	return &buffer[0]
}

func wasiNetError(operation string, errno uint32) error {
	if errno == 0 {
		return nil
	}
	if errno == uint32(syscall.EBADF) {
		return ErrClosed
	}
	return os.NewSyscallError(operation, syscall.Errno(errno))
}

//go:wasmimport sliver_wasi_net_v1 dial_start
//go:noescape
func wasiNetDialStart(network, local, remote string, deadlineNS int64, operation *uint32) uint32

//go:wasmimport sliver_wasi_net_v1 listen
//go:noescape
func wasiNetListen(network, local string, handle *uint32) uint32

//go:wasmimport sliver_wasi_net_v1 accept_start
//go:noescape
func wasiNetAcceptStart(handle uint32, deadlineNS int64, operation *uint32) uint32

//go:wasmimport sliver_wasi_net_v1 read_start
//go:noescape
func wasiNetReadStart(handle, maximum uint32, deadlineNS int64, operation *uint32) uint32

//go:wasmimport sliver_wasi_net_v1 write_start
//go:noescape
func wasiNetWriteStart(handle uint32, data *byte, dataLen uint32, deadlineNS int64, operation *uint32) uint32

//go:wasmimport sliver_wasi_net_v1 recv_from_start
//go:noescape
func wasiNetRecvFromStart(handle, maximum uint32, deadlineNS int64, operation *uint32) uint32

//go:wasmimport sliver_wasi_net_v1 send_to_start
//go:noescape
func wasiNetSendToStart(handle uint32, data *byte, dataLen uint32, remote string, deadlineNS int64, operation *uint32) uint32

//go:wasmimport sliver_wasi_net_v1 op_poll
//go:noescape
func wasiNetOpPoll(operation uint32, data *byte, dataCap uint32, dataLen *uint32, address *byte, addressCap uint32, addressLen, handle *uint32) uint32

//go:wasmimport sliver_wasi_net_v1 op_cancel
//go:noescape
func wasiNetOpCancel(operation uint32) uint32

//go:wasmimport sliver_wasi_net_v1 shutdown
//go:noescape
func wasiNetShutdown(handle, how uint32) uint32

//go:wasmimport sliver_wasi_net_v1 close
//go:noescape
func wasiNetClose(handle uint32) uint32

//go:wasmimport sliver_wasi_net_v1 get_addr
//go:noescape
func wasiNetGetAddr(handle, which uint32, address *byte, addressCap uint32, addressLen *uint32) uint32

//go:wasmimport sliver_wasi_net_v1 set_deadline
//go:noescape
func wasiNetSetDeadline(handle, which uint32, deadlineNS int64) uint32

//go:wasmimport sliver_wasi_net_v1 lookup_start
//go:noescape
func wasiNetLookupStart(network, name string, deadlineNS int64, operation *uint32) uint32
