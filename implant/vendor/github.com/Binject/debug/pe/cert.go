package pe

import (
	"errors"
	"fmt"
	"io"
)

// CERTIFICATE_TABLE is the index of the Certificate Table info in the Data Directory structure
// in the PE header
const CERTIFICATE_TABLE = 4

func readCertTable(f *File, r io.ReadSeeker) ([]byte, error) {
	if f.OptionalHeader == nil { // Optional header is optional, might not exist
		return nil, nil
	}

	var certTableOffset, certTableSize uint32

	switch f.FileHeader.Machine {
	case IMAGE_FILE_MACHINE_I386:
		certTableOffset = f.OptionalHeader.(*OptionalHeader32).DataDirectory[CERTIFICATE_TABLE].VirtualAddress
		certTableSize = f.OptionalHeader.(*OptionalHeader32).DataDirectory[CERTIFICATE_TABLE].Size
	case IMAGE_FILE_MACHINE_AMD64:
		certTableOffset = f.OptionalHeader.(*OptionalHeader64).DataDirectory[CERTIFICATE_TABLE].VirtualAddress
		certTableSize = f.OptionalHeader.(*OptionalHeader64).DataDirectory[CERTIFICATE_TABLE].Size
	default:
		return nil, errors.New("architecture not supported")
	}

	// check if certificate table exists
	if certTableOffset == 0 || certTableSize == 0 {
		return nil, nil
	}

	var err error
	_, err = r.Seek(int64(certTableOffset), seekStart)
	if err != nil {
		return nil, fmt.Errorf("fail to seek to certificate table: %v", err)
	}

	// grab the cert
	cert := make([]byte, certTableSize)
	_, err = io.ReadFull(r, cert)
	if err != nil {
		return nil, fmt.Errorf("fail to read certificate table: %v", err)
	}

	return cert, nil
}
