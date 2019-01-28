// +darwin
package main

type DarwinDump struct {
	data []byte
}

func (d *DarwinDump) Data() []byte {
	return d.data
}

func dumpProcess(pid int32) (ProcessDump, error) {
	dump := &DarwinDump{}
	return dump, nil
}
