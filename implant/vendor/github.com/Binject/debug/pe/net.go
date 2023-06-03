package pe

import (
	"encoding/binary"
	"io"
)

type IMAGE_COR20_HEADER struct {
	Cb                        uint32
	MajorRuntimeVersion       uint16
	MinorRuntimeVersion       uint16
	MetaDataRVA, MetaDataSize uint32
	Flags                     uint32 //todo: define flags
	EntryPointToken           uint32
	ResourcesRVA, ResourcesSize,
	StrongNameSignatureRVA, StrongNameSignatureSize,
	CodeManagerTableRVA, CodeManagerTableSize,
	VTableFixupsRVA, VTableFixupsSize,
	ExportAddressTableJumpsRVA, ExportAddressTableJumpsSize,
	ManagedNativeHeaderRVA, ManagedNativeHeaderSize uint32
}

//Net provides a public interface for getting at some net info.
type Net struct {
	NetDirectory IMAGE_COR20_HEADER //Net directory information
	MetaData     NetMetaData        //MetaData Header
}

type NetMetaData struct {
	Signature       [4]byte //should be 0x424a4542
	MajorVersion    uint16
	MinorVersion    uint16
	Reserved        uint32
	VersionLength   uint32
	VersionString   []byte
	Flags           uint16 //todo: define flags betterer
	NumberOfStreams uint16
}

func newMetadataHeader(i io.Reader) (NetMetaData, error) {
	r := NetMetaData{}

	//todo: error checks/sanity checks
	binary.Read(i, binary.LittleEndian, &r.Signature)
	binary.Read(i, binary.LittleEndian, &r.MajorVersion)
	binary.Read(i, binary.LittleEndian, &r.MinorVersion)
	binary.Read(i, binary.LittleEndian, &r.Reserved)

	// it appears that this value is terminated by two nulls.. that might be important at some point
	binary.Read(i, binary.LittleEndian, &r.VersionLength)
	r.VersionString = make([]byte, r.VersionLength)
	i.Read(r.VersionString)

	binary.Read(i, binary.LittleEndian, r.Flags)

	return r, nil
}

//NetCLRVersion returns the CLR version specified by the binary. Returns an empty string if not a net binary. String has had trailing nulls stripped.
func (f File) NetCLRVersion() string {
	b := f.Net.MetaData.VersionString
	for i, x := range b {
		if x == 0x00 {
			b = b[:i]
			break
		}
	}
	return string(b)
}
