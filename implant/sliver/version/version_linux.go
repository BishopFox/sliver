package version

import (
	"fmt"
	"log"
	"strings"
	"syscall"
)

func getString(input [65]int8) string {
	var buf [65]byte
	for i, b := range input {
		buf[i] = byte(b)
	}
	ver := string(buf[:])
	if i := strings.Index(ver, "\x00"); i != -1 {
		ver = ver[:i]
	}
	return ver
}

// GetVersion returns the os version information
func GetVersion() string {
	var uname syscall.Utsname
	if err := syscall.Uname(&uname); err != nil {
		log.Fatal(err)
	}
	return fmt.Sprintf("%s %s %s", getString(uname.Sysname), getString(uname.Nodename), getString(uname.Release))
}
