//go:build darwin
// +build darwin

package ps

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os/user"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

type DarwinProcess struct {
	pid     int
	ppid    int
	binary  string
	owner   string
	arch    string
	cmdLine []string
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

func (p *DarwinProcess) CmdLine() []string {
	return p.cmdLine
}

func (p *DarwinProcess) Architecture() string {
	return p.arch
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
	var owner string
	buf, _, err := procInfoSyscall()
	if err != nil {
		return nil, err
	}

	procs := make([]*KinfoProc, 0, 50)
	k := 0
	for i := _KINFO_STRUCT_SIZE; i < buf.Len(); i += _KINFO_STRUCT_SIZE {
		// Super unsafe but faster than doing manual parsing
		proc := (*KinfoProc)(unsafe.Pointer((&buf.Bytes()[k:i][0])))

		k = i
		procs = append(procs, proc)
	}

	darwinProcs := make([]Process, len(procs))
	for i, p := range procs {
		binPath, err := getPathFromPid(int(p.Proc.P_pid))
		if err != nil {
			pComm := make([]byte, 17)
			for i, x := range p.Proc.P_comm {
				pComm[i] = byte(x)
			}
			pCommReader := bytes.NewBuffer(pComm)
			binPath, _ = pCommReader.ReadString(0x00)
		}
		binPath = strings.TrimSuffix(binPath, "\x00") // Trim the null byte
		// Discard the error: if the call errors out, we'll just have an empty argv slice
		cmdLine, _ := getArgvFromPid(int(p.Proc.P_pid))

		uid := fmt.Sprintf("%d", p.Eproc.Ucred.Uid)
		u, err := user.LookupId(uid)
		if err != nil {
			owner = uid
		} else {
			owner = u.Username
		}
		if owner == "" {
			owner = uid
		}
		arch := ""

		darwinProcs[i] = &DarwinProcess{
			pid:     int(p.Proc.P_pid),
			ppid:    int(p.Eproc.Ppid),
			binary:  binPath,
			owner:   owner,
			cmdLine: cmdLine,
			arch:    arch,
		}
	}

	return darwinProcs, nil
}

func procInfoSyscall() (*bytes.Buffer, uint64, error) {
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
		return nil, 0, errno
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
		return nil, 0, errno
	}

	return bytes.NewBuffer(bs[0:size]), uint64(size), nil
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
	_KERN_PROCARGS2          = 49
)

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
	buffer := bytes.NewBuffer(buf)
	path, err = buffer.ReadString(0x00)
	return
}

func getArgvFromPid(pid int) ([]string, error) {
	systemMaxArgs, err := unix.SysctlUint32("kern.argmax")
	if err != nil {
		return []string{""}, err
	}
	processArgs := make([]byte, systemMaxArgs)
	var size uint = uint(systemMaxArgs)
	mib := [4]int32{_CTRL_KERN, _KERN_PROCARGS2, int32(pid), 0}
	_, _, errno := syscall.Syscall6(
		syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		4,
		uintptr(unsafe.Pointer(&processArgs[0])),
		uintptr(unsafe.Pointer(&size)),
		0,
		0,
	)
	if errno != 0 {
		errStr := unix.ErrnoName(errno)
		return []string{""}, fmt.Errorf("%s", errStr)
	}
	buffer := bytes.NewBuffer(processArgs[0:size])
	numberOfArgsBytes := buffer.Next(4)
	numberOfArgs := binary.LittleEndian.Uint32(numberOfArgsBytes)
	argv := make([]string, numberOfArgs+1) // executable name is present twice

	// There's probably a way to optimize that loop.
	// processArgs is a buffer of "things" delimited by null bytes.
	// The structure looks like this:
	// number of args: int32
	// executable name: null terminated string
	// empty: null terminated string
	// empty: null terminated string
	// empty: null terminated string
	// argv[]: table of null termiated strings, containing the executable name in argv[0]
	i := 0
	for {
		arg, err := buffer.ReadString(0x00)
		if err != nil {
			break
		}
		if strings.ReplaceAll(arg, "\x00", "") != "" {
			argv[i] = arg
			i++
		}
		if i == int(numberOfArgs+1) {
			break
		}
	}
	if len(argv) >= 2 {
		argv = argv[1:]
	}
	return argv, nil
}
