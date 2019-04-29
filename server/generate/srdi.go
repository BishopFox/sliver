package generate

/*
	This is port of SRDI by Leo Loobeek, that we've made a few modifications to

	Originals:
	https://gist.github.com/leoloobeek/c726719d25d7e7953d4121bd93dd2ed3
	https://silentbreaksecurity.com/srdi-shellcode-reflective-dll-injection/
*/

import (
	"encoding/binary"
	"flag"
	"io/ioutil"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ShellcodeRDIToFile generates a sRDI shellcode and writes it to a file
func ShellcodeRDIToFile(dllPath string, functionName string) (shellcodePath string, err error) {
	shellcode, err := ShellcodeRDI(dllPath, functionName)
	if err != nil {
		return "", err
	}
	dir := path.Dir(dllPath)
	filename := strings.Replace(path.Base(dllPath), ".dll", ".bin", 1)
	filepath := filepath.Join(dir, filename)
	ioutil.WriteFile(filepath, shellcode, os.ModePerm)
	return filepath, nil
}

// ShellcodeRDI generates a reflective shellcode based on a DLL file
func ShellcodeRDI(dllPath string, functionName string) (shellcode []byte, err error) {
	// handle command line arguments, -h or -help shows the menu
	userDataStr := ""
	clearHeader := true
	flag.Parse()

	dllBytes, err := ioutil.ReadFile(dllPath)
	if err != nil {
		panic(err)
	}

	// functionHash is 0x10 by default, otherwise get the hash and convert to bytes
	var hashFunction []byte
	if functionName != "" {
		hashFunctionUint32 := hashFunctionName(functionName)
		hashFunction = pack(hashFunctionUint32)
	} else {
		hashFunction = pack(uint32(0x10))
	}

	flags := 0
	if clearHeader {
		flags |= 0x1
	}
	var userData []byte
	if userDataStr != "" {
		userData = []byte(userDataStr)
	}
	shellcode = convertToShellcode(dllBytes, hashFunction, userData, flags)
	//	err = os.RemoveAll(path.Clean(path.Dir(dllPath) + "/../"))
	return shellcode, nil

}

func convertToShellcode(dllBytes, functionHash, userData []byte, flags int) []byte {

	if userData == nil {
		userData = []byte("None")
	}

	var final []byte

	if is64BitDLL(dllBytes) {
		// do 64 bit things

		bootstrapSize := 64

		// call next instruction (Pushes next instruction address to stack)
		bootstrap := []byte{0xe8, 0x00, 0x00, 0x00, 0x00}

		// Set the offset to our DLL from pop result
		dllOffset := bootstrapSize - len(bootstrap) + len(rdiShellcode64)

		// pop rcx - Capture our current location in memory
		bootstrap = append(bootstrap, 0x59)

		// mov r8, rcx - copy our location in memory to r8 before we start modifying RCX
		bootstrap = append(bootstrap, 0x49, 0x89, 0xc8)

		// add rcx, <Offset of the DLL>
		bootstrap = append(bootstrap, 0x48, 0x81, 0xc1)

		bootstrap = append(bootstrap, pack(uint32(dllOffset))...)

		// mov edx, <Hash of function>
		bootstrap = append(bootstrap, 0xba)
		bootstrap = append(bootstrap, functionHash...)

		// Setup the location of our user data
		// add r8, <Offset of the DLL> + <Length of DLL>
		bootstrap = append(bootstrap, 0x49, 0x81, 0xc0)
		userDataLocation := dllOffset + len(dllBytes)
		bootstrap = append(bootstrap, pack(uint32(userDataLocation))...)

		// mov r9d, <Length of User Data>
		bootstrap = append(bootstrap, 0x41, 0xb9)
		bootstrap = append(bootstrap, pack(uint32(len(userData)))...)

		// push rsi - save original value
		bootstrap = append(bootstrap, 0x56)

		// mov rsi, rsp - store our current stack pointer for later
		bootstrap = append(bootstrap, 0x48, 0x89, 0xe6)

		// and rsp, 0x0FFFFFFFFFFFFFFF0 - Align the stack to 16 bytes
		bootstrap = append(bootstrap, 0x48, 0x83, 0xe4, 0xf0)

		// sub rsp, 0x30 - Create some breathing room on the stack
		bootstrap = append(bootstrap, 0x48, 0x83, 0xec)
		bootstrap = append(bootstrap, 0x30) // 32 bytes for shadow space + 8 bytes for last arg + 8 bytes for stack alignment

		// mov dword ptr [rsp + 0x20], <Flags> - Push arg 5 just above shadow space
		bootstrap = append(bootstrap, 0xC7, 0x44, 0x24)
		bootstrap = append(bootstrap, 0x20)
		bootstrap = append(bootstrap, pack(uint32(flags))...)

		// call - Transfer execution to the RDI
		bootstrap = append(bootstrap, 0xe8)
		bootstrap = append(bootstrap, byte(bootstrapSize-len(bootstrap)-4)) // Skip over the remainder of instructions
		bootstrap = append(bootstrap, 0x00, 0x00, 0x00)

		// mov rsp, rsi - Reset our original stack pointer
		bootstrap = append(bootstrap, 0x48, 0x89, 0xf4)

		// pop rsi - Put things back where we left them
		bootstrap = append(bootstrap, 0x5e)

		// ret - return to caller
		bootstrap = append(bootstrap, 0xc3)

		final = append(bootstrap, rdiShellcode64...)
		final = append(final, dllBytes...)
		final = append(final, userData...)

	} else {
		// do 32 bit things

		bootstrapSize := 45

		// call next instruction (Pushes next instruction address to stack)
		bootstrap := []byte{0xe8, 0x00, 0x00, 0x00, 0x00}

		// Set the offset to our DLL from pop result
		dllOffset := bootstrapSize - len(bootstrap) + len(rdiShellcode32)

		// pop eax - Capture our current location in memory
		bootstrap = append(bootstrap, 0x58)

		// mov ebx, eax - copy our location in memory to ebx before we start modifying eax
		bootstrap = append(bootstrap, 0x89, 0xc3)

		// add eax, <Offset to the DLL>
		bootstrap = append(bootstrap, 0x05)
		bootstrap = append(bootstrap, pack(uint32(dllOffset))...)

		// add ebx, <Offset to the DLL> + <Size of DLL>
		bootstrap = append(bootstrap, 0x81, 0xc3)
		userDataLocation := dllOffset + len(dllBytes)
		bootstrap = append(bootstrap, pack(uint32(userDataLocation))...)

		// push <Flags>
		bootstrap = append(bootstrap, 0x68)
		bootstrap = append(bootstrap, pack(uint32(flags))...)

		// push <Length of User Data>
		bootstrap = append(bootstrap, 0x68)
		bootstrap = append(bootstrap, pack(uint32(len(userData)))...)

		// push ebx
		bootstrap = append(bootstrap, 0x53)

		// push <hash of function>
		bootstrap = append(bootstrap, 0x68)
		bootstrap = append(bootstrap, functionHash...)

		// push eax
		bootstrap = append(bootstrap, 0x50)

		// call - Transfer execution to the RDI
		bootstrap = append(bootstrap, 0xe8)
		bootstrap = append(bootstrap, byte(bootstrapSize-len(bootstrap)-4)) // Skip over the remainder of instructions
		bootstrap = append(bootstrap, 0x00, 0x00, 0x00)

		// add esp, 0x14 - correct the stack pointer
		bootstrap = append(bootstrap, 0x83, 0xc4, 0x14)

		// ret - return to caller
		bootstrap = append(bootstrap, 0xc3)

		final = append(bootstrap, rdiShellcode32...)
		final = append(final, dllBytes...)
		final = append(final, userData...)
	}

	return final

}

// helper function similar to struct.pack from python3
func pack(val uint32) []byte {
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, val)
	return bytes
}

// "HelloWorld" = 3571859646
func hashFunctionName(name string) uint32 {
	function := []byte(name)
	function = append(function, 0x00)

	functionHash := uint32(0)

	for _, b := range function {
		functionHash = ror(functionHash, 13, 32)
		functionHash += uint32(b)
	}

	return functionHash
}

// ROR-13 implementation
func ror(val uint32, rBits uint32, maxBits uint32) uint32 {
	exp := uint32(math.Exp2(float64(maxBits))) - 1
	return ((val & exp) >> (rBits % maxBits)) | (val << (maxBits - (rBits % maxBits)) & exp)
}

func is64BitDLL(dllBytes []byte) bool {
	machineIA64 := uint16(512)
	machineAMD64 := uint16(34404)

	headerOffset := binary.LittleEndian.Uint32(dllBytes[60:64])
	machine := binary.LittleEndian.Uint16(dllBytes[headerOffset+4 : headerOffset+4+2])

	// 64 bit
	if machine == machineIA64 || machine == machineAMD64 {
		return true
	}
	return false
}
