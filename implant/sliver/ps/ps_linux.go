//go:build linux
// +build linux

package ps

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"
)

// UnixProcess is an implementation of Process
// that contains Unix-specific fields and information.
type UnixProcess struct {
	pid     int
	ppid    int
	state   rune
	pgrp    int
	sid     int
	arch    string
	cmdLine []string

	binary string
	owner  string
}

// Pid returns the process identifier
func (p *UnixProcess) Pid() int {
	return p.pid
}

// PPid returns the parent process identifier
func (p *UnixProcess) PPid() int {
	return p.ppid
}

// Executable returns the process name
func (p *UnixProcess) Executable() string {
	return p.binary
}

// Owner returns the username the process belongs to
func (p *UnixProcess) Owner() string {
	return p.owner
}

func (p *UnixProcess) CmdLine() []string {
	return p.cmdLine
}

func (p *UnixProcess) Architecture() string {
	return p.arch
}

func getProcessOwnerUid(pid int) (uint32, error) {
	filename := fmt.Sprintf("/proc/%d/task", pid)
	f, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	fileStat := &syscall.Stat_t{}
	err = syscall.Fstat(int(f.Fd()), fileStat)
	if err != nil {
		return 0, err
	}
	return fileStat.Uid, nil
}

func getProcessOwner(pid int) (string, error) {
	uid, err := getProcessOwnerUid(pid)
	if err != nil {
		return "", err
	}
	usr, err := user.LookupId(fmt.Sprintf("%d", uid))
	if err != nil {
		// return the UID in case LookupId fails
		return fmt.Sprintf("%d", uid), nil
	}
	return usr.Username, err
}

func getProcessCmdLine(pid int) ([]string, error) {
	cmdLinePath := fmt.Sprintf("/proc/%d/cmdline", pid)
	data, err := os.ReadFile(cmdLinePath)
	if err != nil {
		return []string{""}, err
	}
	argv := strings.Split(string(data), "\x00")
	return argv, nil
}

func getProcessArchitecture(pid int) (string, error) {
	exePath := fmt.Sprintf("/proc/%d/exe", pid)

	f, err := os.Open(exePath)
	if err != nil {
		return "", err
	}
	_, err = f.Seek(0x12, 0)
	if err != nil {
		return "", err
	}
	mach := make([]byte, 2)
	n, err := io.ReadAtLeast(f, mach, 2)

	f.Close()

	if err != nil || n < 2 {
		return "", nil
	}

	if mach[0] == 0xb3 {
		return "aarch64", nil
	}
	if mach[0] == 0x03 {
		return "x86", nil
	}
	if mach[0] == 0x3e {
		return "x86_64", nil
	}
	return "", err
}

// Refresh reloads all the data associated with this process.
func (p *UnixProcess) Refresh() error {
	statPath := fmt.Sprintf("/proc/%d/stat", p.pid)
	dataBytes, err := os.ReadFile(statPath)
	if err != nil {
		return err
	}

	// First, parse out the image name
	data := string(dataBytes)
	binStart := strings.IndexRune(data, '(') + 1
	binEnd := strings.IndexRune(data[binStart:], ')')
	p.binary = data[binStart : binStart+binEnd]

	// Move past the image name and start parsing the rest
	data = data[binStart+binEnd+2:]
	_, err = fmt.Sscanf(data,
		"%c %d %d %d",
		&p.state,
		&p.ppid,
		&p.pgrp,
		&p.sid)

	return err
}

func findProcess(pid int, fullInfo bool) (Process, error) {
	ps, err := processes(fullInfo)
	if err != nil {
		return nil, err
	}
	for _, p := range ps {
		if p.Pid() == pid {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no process found for pid %d", pid)
}

func processes(fullInfo bool) ([]Process, error) {
	d, err := os.Open("/proc")
	if err != nil {
		return nil, err
	}
	defer d.Close()

	results := make([]Process, 0, 50)
	for {
		fis, err := d.Readdir(10)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		for _, fi := range fis {
			// We only care about directories, since all pids are dirs
			if !fi.IsDir() {
				continue
			}
			// We only care if the name starts with a numeric
			name := fi.Name()
			if name[0] < '0' || name[0] > '9' {
				continue
			}
			// From this point forward, any errors we just ignore, because
			// it might simply be that the process doesn't exist anymore.
			pid, err := strconv.ParseInt(name, 10, 0)
			if err != nil {
				continue
			}
			p, err := newUnixProcess(int(pid))
			if err != nil {
				continue
			}
			if !fullInfo {
				results = append(results, p)
				continue
			}
			p.owner, err = getProcessOwner(int(pid))
			if err != nil {
				continue
			}
			argv, err := getProcessCmdLine(int(pid))
			if err == nil {
				p.cmdLine = argv
				if argv[0] != "" && len(argv[0]) > 0 {
					p.binary = argv[0]
				}
			}
			p.arch, err = getProcessArchitecture(int(pid))
			results = append(results, p)
		}
	}
	return results, nil
}

func newUnixProcess(pid int) (*UnixProcess, error) {
	p := &UnixProcess{pid: pid}
	return p, p.Refresh()
}
