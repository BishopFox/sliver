package sysfs

import (
	"net"
	"os"

	experimentalsys "github.com/tetratelabs/wazero/experimental/sys"
	socketapi "github.com/tetratelabs/wazero/internal/sock"
	"github.com/tetratelabs/wazero/sys"
)

// NewTCPListenerFile creates a socketapi.TCPSock for a given *net.TCPListener.
func NewTCPListenerFile(tl *net.TCPListener) socketapi.TCPSock {
	return newTCPListenerFile(tl)
}

// baseSockFile implements base behavior for all TCPSock, TCPConn files,
// regardless the platform.
type baseSockFile struct {
	experimentalsys.UnimplementedFile
}

var _ experimentalsys.File = (*baseSockFile)(nil)

// IsDir implements the same method as documented on File.IsDir
func (*baseSockFile) IsDir() (bool, experimentalsys.Errno) {
	// We need to override this method because WASI-libc prestats the FD
	// and the default impl returns ENOSYS otherwise.
	return false, 0
}

// Stat implements the same method as documented on File.Stat
func (f *baseSockFile) Stat() (fs sys.Stat_t, errno experimentalsys.Errno) {
	// The mode is not really important, but it should be neither a regular file nor a directory.
	fs.Mode = os.ModeIrregular
	return
}
