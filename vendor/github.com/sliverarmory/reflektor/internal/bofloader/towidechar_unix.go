//go:build (darwin && !ios && (amd64 || arm64)) || (freebsd && (amd64 || arm64)) || (linux && !android && (386 || amd64 || (arm && arm.7) || arm64 || ppc64le || riscv64))

package bofloader

import (
	"encoding/binary"
	"unicode/utf16"
	"unicode/utf8"
)

const beaconWideCharSize = 2

// toWideChar converts a NUL-terminated UTF-8 string to the Beacon API's
// UTF-16LE representation. Portable BOFs should use a uint16_t destination;
// native Unix wchar_t values are deliberately not substituted for the wire ABI.
func toWideChar(sourceAddress, destinationAddress, maximumValue uintptr) (result uintptr) {
	defer callbackPanic("toWideChar")
	maximum := int64(int32(maximumValue))
	if sourceAddress == 0 || destinationAddress == 0 || maximum < beaconWideCharSize || maximum > maxCallbackData {
		return 0
	}
	text, err := readCString(sourceAddress, maxFormatString)
	if err != nil {
		callbackError("toWideChar", "%v", err)
		return 0
	}
	if !utf8.ValidString(text) {
		return 0
	}
	codeUnits := utf16.Encode([]rune(text))
	required := (len(codeUnits) + 1) * beaconWideCharSize
	if required > int(maximum) {
		return 0
	}
	destination := pointerBytes(destinationAddress, required)
	for index, value := range codeUnits {
		binary.LittleEndian.PutUint16(destination[index*beaconWideCharSize:], value)
	}
	binary.LittleEndian.PutUint16(destination[len(codeUnits)*beaconWideCharSize:], 0)
	return 1
}
