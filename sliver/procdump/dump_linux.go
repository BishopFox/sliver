// +linux

package procdump

// LinuxDump - Structure implementing the ProcessDump
// interface for linux processes
type LinuxDump struct {
	data []byte
}

// Data - Returns the byte array corresponding to a process memory
func (d *LinuxDump) Data() []byte {
	return d.data
}

func dumpProcess(pid int32) (ProcessDump, error) {
	res := &LinuxDump{}
	return res, nil
}
