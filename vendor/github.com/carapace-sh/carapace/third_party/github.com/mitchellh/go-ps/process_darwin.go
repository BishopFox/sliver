//go:build darwin

package ps

import (
	"bytes"
	"encoding/binary"
	"syscall"
	"unsafe"
)

type DarwinProcess struct {
	pid    int
	ppid   int
	binary string
}

func (p *DarwinProcess) Pid() int {
	return p.pid
}

func (p *DarwinProcess) PPid() int {
	return p.ppid
}

func (p *DarwinProcess) Executable() string {
	return p.binary
}

func findProcess(pid int) (Process, error) {
	buf, err := darwinSysctlPid(pid)
	if err != nil {
		return nil, err
	}

	proc := &kinfoProc{}
	err = binary.Read(buf, binary.LittleEndian, proc)
	if err != nil {
		return nil, err
	}

	if proc.Pid == 0 { // no process with this pid
		return nil, nil
	}

	return &DarwinProcess{
		pid:    int(proc.Pid),
		ppid:   int(proc.PPid),
		binary: darwinCstring(proc.Comm),
	}, nil
}

func processes() ([]Process, error) {
	buf, err := darwinSysctlAll()
	if err != nil {
		return nil, err
	}

	procs := make([]*kinfoProc, 0, 50)
	k := 0
	for i := _KINFO_STRUCT_SIZE; i < buf.Len(); i += _KINFO_STRUCT_SIZE {
		proc := &kinfoProc{}
		err = binary.Read(bytes.NewBuffer(buf.Bytes()[k:i]), binary.LittleEndian, proc)
		if err != nil {
			return nil, err
		}

		k = i
		procs = append(procs, proc)
	}

	darwinProcs := make([]Process, len(procs))
	for i, p := range procs {
		darwinProcs[i] = &DarwinProcess{
			pid:    int(p.Pid),
			ppid:   int(p.PPid),
			binary: darwinCstring(p.Comm),
		}
	}

	return darwinProcs, nil
}

func darwinSysctlPid(pid int) (*bytes.Buffer, error) {
	mib := [4]int32{_CTRL_KERN, _KERN_PROC, _KERN_PROC_PID, int32(pid)}
	size := uintptr(0)

	_, _, errno := syscall.Syscall6(
		syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		4,
		0,
		uintptr(unsafe.Pointer(&size)),
		0,
		0)

	if errno != 0 {
		return nil, errno
	}

	bs := make([]byte, size)
	_, _, errno = syscall.Syscall6(
		syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		4,
		uintptr(unsafe.Pointer(&bs[0])),
		uintptr(unsafe.Pointer(&size)),
		0,
		0)

	if errno != 0 {
		return nil, errno
	}

	if size < _KINFO_STRUCT_SIZE {
		return bytes.NewBuffer(nil), nil // no process found
	}

	return bytes.NewBuffer(bs[0:size]), nil
}

func darwinSysctlAll() (*bytes.Buffer, error) {
	mib := [4]int32{_CTRL_KERN, _KERN_PROC, _KERN_PROC_ALL, 0}
	size := uintptr(0)

	_, _, errno := syscall.Syscall6(
		syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		4,
		0,
		uintptr(unsafe.Pointer(&size)),
		0,
		0)

	if errno != 0 {
		return nil, errno
	}

	bs := make([]byte, size)
	_, _, errno = syscall.Syscall6(
		syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		4,
		uintptr(unsafe.Pointer(&bs[0])),
		uintptr(unsafe.Pointer(&size)),
		0,
		0)

	if errno != 0 {
		return nil, errno
	}

	return bytes.NewBuffer(bs[0:size]), nil
}

func darwinCstring(s [16]byte) string {
	i := 0
	for _, b := range s {
		if b != 0 {
			i++
		} else {
			break
		}
	}

	return string(s[:i])
}

const (
	_CTRL_KERN         = 1
	_KERN_PROC         = 14
	_KERN_PROC_ALL     = 0
	_KERN_PROC_PID     = 1
	_KINFO_STRUCT_SIZE = 648
)

type kinfoProc struct {
	_    [40]byte
	Pid  int32
	_    [199]byte
	Comm [16]byte
	_    [301]byte
	PPid int32
	_    [84]byte
}
