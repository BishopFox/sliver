// +build linux

package ps

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"
)

// UnixProcess is an implementation of Process
// that contains Unix-specific fields and information.
type UnixProcess struct {
	pid   int
	ppid  int
	state rune
	pgrp  int
	sid   int

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

// Refresh reloads all the data associated with this process.
func (p *UnixProcess) Refresh() error {
	statPath := fmt.Sprintf("/proc/%d/stat", p.pid)
	dataBytes, err := ioutil.ReadFile(statPath)
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
	return nil, fmt.Errorf("no process found for pid %d", pid)
}

func processes() ([]Process, error) {
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
			p.owner, err = getProcessOwner(int(pid))
			if err != nil {
				continue
			}
			results = append(results, p)
		}
	}
	return results, nil
}

func newUnixProcess(pid int) (*UnixProcess, error) {
	p := &UnixProcess{pid: pid}
	return p, p.Refresh()
}
