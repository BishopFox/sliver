// +build darwin

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
	owner  string
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

func (p *DarwinProcess) Owner() string {
	return p.owner
}

func findProcess(pid int) (Process, error) {
	ps, err := processes()
	if err != nil {
		return nil, err
	}

	for _, p := range ps {
		if p.Pid() == pid {
			return p, nil
		}
	}

	return nil, nil
}

func processes() ([]Process, error) {
	buf, err := darwinSyscall()
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
		binPath, err := getPathFromPid(int(p.Pid))
		if err != nil {
			binPath = darwinCstring(p.Comm)
		}
		darwinProcs[i] = &DarwinProcess{
			pid:    int(p.Pid),
			ppid:   int(p.PPid),
			binary: binPath,
			owner:  "",
		}
	}

	return darwinProcs, nil
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

func darwinSyscall() (*bytes.Buffer, error) {
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

const (
	_CTRL_KERN               = 1
	_KERN_PROC               = 14
	_KERN_PROC_ALL           = 0
	_KINFO_STRUCT_SIZE       = 648
	MAXPATHLEN               = 1024
	PROC_PIDPATHINFO_MAXSIZE = MAXPATHLEN * 4
	PROC_INFO_CALL_PIDINFO   = 0x2
	PROC_PIDPATHINFO         = 11
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

func errnoErr(e syscall.Errno) error {
	switch e {
	case 0:
		return nil
	case syscall.EAGAIN:
		return syscall.EAGAIN
	case syscall.EINVAL:
		return syscall.EINVAL
	case syscall.ENOENT:
		return syscall.ENOENT
	}
	return e

}

func readCString(data []byte) string {
	i := 0
	for _, b := range data {
		if b != 0 {
			i++
		} else {
			break
		}
	}
	return string(data[:i])
}

func getPathFromPid(pid int) (path string, err error) {
	bufSize := PROC_PIDPATHINFO_MAXSIZE
	buf := make([]byte, bufSize)

	// https://opensource.apple.com/source/xnu/xnu-1504.3.12/bsd/kern/syscalls.master
	// int proc_info(int32_t callnum,int32_t pid,uint32_t flavor, uint64_t arg,user_addr_t buffer,int32_t buffersize)
	_, _, err = syscall.Syscall6(
		syscall.SYS_PROC_INFO,
		PROC_INFO_CALL_PIDINFO,
		uintptr(pid),
		PROC_PIDPATHINFO,
		uintptr(uint64(0)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(bufSize),
	)
	if errno, ok := err.(syscall.Errno); ok {
		if err = errnoErr(errno); err != nil {
			return
		}
	}
	path = readCString(buf)
	return
}
