package main

// ProcessDump is the generic interfaces that provides access to a process dump
type ProcessDump interface {
	Data() []byte
}

// DumpProcess returns the minidump of a process
func DumpProcess(pid int32) (ProcessDump, error) {
	return dumpProcess(pid)
}
