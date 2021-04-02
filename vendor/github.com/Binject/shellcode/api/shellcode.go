package api

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"log"
	"net"
)

// Os - Operating System Options Flag
type Os string

// Arch - Architecture Options Flag
type Arch string

// Bits - Bit Width Options Flag
type Bits string

const (

	// Operating System Options

	// Windows flag for Windows OS
	Windows Os = "windows"
	// Linux flag for Linux OS
	Linux Os = "linux"
	// FreeBSD flag for FreeBSD OS
	FreeBSD Os = "freebsd"
	// Darwin flag for Darwin / Mac OS
	Darwin Os = "darwin"

	// Architecture Options

	// Intel32 flag for Intel/AMD 32 bit architectures
	Intel32 Arch = "x32"
	// Intel64 flag for Intel/AMD 64 bit architectures
	Intel64 Arch = "x64"
	// Intel32y64 flag for Intel/AMD 32+64 bit combo shellcodes
	Intel32y64 Arch = "x32x64"
	// Arm flag for Arm 32 bit shellcodes
	Arm Arch = "arm"
)

var (
	// Arches - list of human readable architecture names
	Arches []string = []string{"x32", "x64", "x32x64", "arm"}

	// Oses - list of human readable OS names
	Oses []string = []string{"windows", "linux", "darwin"}
)

// Generator - type for a shellcode generator
type Generator struct {
	Os       Os
	Arch     Arch
	Bit      Bits
	Name     string
	Function func(Parameters) ([]byte, error)
}

var generators []Generator

// RegisterShellCode - registers a shellcode generating function with the registry
func RegisterShellCode(
	os Os,
	arch Arch,
	name string,
	fx func(Parameters) ([]byte, error)) {

	generators = append(generators, Generator{Os: os, Arch: arch, Name: name, Function: fx})
}

// LookupShellCode - looks up shellcode by OS and architecture
func LookupShellCode(os Os, arch Arch) []Generator {
	var ret []Generator
	for _, g := range generators {
		if g.Os == os && g.Arch == arch {
			ret = append(ret, g)
		}
	}
	return ret
}

// PrintShellCodes - looks up shellcode by OS and architecture and prints the output
func PrintShellCodes(os Os, arch Arch) {
	gens := LookupShellCode(os, arch)
	for _, g := range gens {
		log.Printf("%+v\n", g)
	}
}

// PackUint16 - packs a jump address
func PackUint16(addr uint16) (string, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, addr)
	if err != nil {
		return "", err
	}
	return string(buf.Bytes()), nil
}

// PackUint32 - packs a jump address
func PackUint32(addr uint32) (string, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, addr)
	if err != nil {
		return "", err
	}
	return string(buf.Bytes()), nil
}

// PackUint64 - packs a jump address
func PackUint64(addr uint64) (string, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, addr)
	if err != nil {
		return "", err
	}
	return string(buf.Bytes()), nil
}

// PackPort - packs a port
func PackPort(port uint16) (string, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, port)
	if err != nil {
		return "", err
	}
	return string(buf.Bytes()), nil
}

// PackIP - packs an IP
func PackIP(ip string) string {
	ipaddr := net.ParseIP(ip).To4()
	return string(ipaddr)
}

// ApplyPrefixForkIntel64 - Prepends instructions to fork and have the parent jump to a relative 32-bit address (the entryJump argument)
//							Intel x64 Linux version
//
//							Returns the resulting shellcode
func ApplyPrefixForkIntel64(shellcode []byte, entryJump uint32, byteOrder binary.ByteOrder) []byte {
	/*
		Disassembly:
		0:  b8 02 00 00 00          mov    eax,0x2
		5:  cd 80                   int    0x80
		7:  83 f8 00                cmp    eax,0x0
		a:  0f 85 xx xx xx xx       jne    <entryJump>
	*/
	prefix := bytes.NewBuffer([]byte{0xB8, 0x02, 0x00, 0x00, 0x00, 0xCD, 0x80, 0x83, 0xF8,
		0x00, 0x0F, 0x85})
	w := bufio.NewWriter(prefix)
	binary.Write(w, byteOrder, entryJump)
	binary.Write(w, byteOrder, shellcode)
	w.Flush()
	return prefix.Bytes()
}

// ApplySuffixJmpIntel64 - Appends instructions to jump to the original entryPoint (the parameter)
//							Intel x64 Linux version
//
//							Returns the resulting shellcode
func ApplySuffixJmpIntel64(shellcode []byte, shellcodeVaddr uint32, entryPoint uint32, byteOrder binary.ByteOrder) []byte {
	/*
		Disassembly:
		0:  e9 00 00 00 00          jmp    <entryJump>
	*/

	retval := append(shellcode, 0xe9)
	buf := bytes.NewBuffer(retval)
	w := bufio.NewWriter(buf)
	entryJump := entryPoint - (shellcodeVaddr + 5) - uint32(len(shellcode))
	binary.Write(w, byteOrder, entryJump)
	w.Flush()
	return buf.Bytes()
}

// ApplySuffixJmpIntel32 - Appends instructions to jump to the original entryPoint (the parameter)
//							Intel x32 Windows version
//
//							Returns the resulting shellcode
func ApplySuffixJmpIntel32(shellcode []byte, shellcodeVaddr uint32, entryPoint uint32, byteOrder binary.ByteOrder) []byte {
	/*
		Disassembly:
		0:  68 					push
		1:  xx xx xx xx			original entry point (make sure to add the imageBase)
		5:  ff 24 24            jmp [esp]
	*/

	retval := append(shellcode, 0x68)
	buf := bytes.NewBuffer(retval)
	w := bufio.NewWriter(buf)
	binary.Write(w, byteOrder, entryPoint)
	w.Flush()
	return append(buf.Bytes(), 0xff, 0x24, 0x24)
}
