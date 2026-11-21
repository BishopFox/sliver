package generate

import (
	"errors"
	"fmt"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

// SpoofMetadataConfigFromProto converts the wire configuration into the
// existing PE mutator's typed input. Generation-time and compatibility RPCs
// use the same bounds and field mapping.
func SpoofMetadataConfigFromProto(req *clientpb.SpoofMetadataConfig) (*SpoofMetadataConfig, error) {
	if req == nil || req.GetPE() == nil {
		return nil, errors.New("spoof metadata has no supported executable format")
	}
	peReq := req.GetPE()
	resourceDirectory, err := protoResourceDirectory(peReq.GetResourceDirectory())
	if err != nil {
		return nil, err
	}
	exportDirectory, err := protoExportDirectory(peReq.GetExportDirectory())
	if err != nil {
		return nil, err
	}
	return &SpoofMetadataConfig{PE: &PESpoofMetadataConfig{
		Source: protoSpoofFile(peReq.GetSource()), Icon: protoSpoofFile(peReq.GetIcon()),
		ResourceDirectory:        resourceDirectory,
		ResourceDirectoryEntries: protoResourceDirectoryEntries(peReq.GetResourceDirectoryEntries()),
		ResourceDataEntries:      protoResourceDataEntries(peReq.GetResourceDataEntries()),
		ExportDirectory:          exportDirectory,
	}}, nil
}

func protoSpoofFile(req *clientpb.SpoofMetadataFile) *SpoofMetadataFile {
	if req == nil {
		return nil
	}
	return &SpoofMetadataFile{Name: req.GetName(), Data: req.GetData()}
}

func protoResourceDirectory(req *clientpb.IMAGE_RESOURCE_DIRECTORY) (*IMAGE_RESOURCE_DIRECTORY, error) {
	if req == nil {
		return nil, nil
	}
	major, err := protoUint16("pe.resource_directory.major_version", req.GetMajorVersion())
	if err != nil {
		return nil, err
	}
	minor, err := protoUint16("pe.resource_directory.minor_version", req.GetMinorVersion())
	if err != nil {
		return nil, err
	}
	named, err := protoUint16("pe.resource_directory.number_of_named_entries", req.GetNumberOfNamedEntries())
	if err != nil {
		return nil, err
	}
	ids, err := protoUint16("pe.resource_directory.number_of_id_entries", req.GetNumberOfIdEntries())
	if err != nil {
		return nil, err
	}
	return &IMAGE_RESOURCE_DIRECTORY{
		Characteristics: req.GetCharacteristics(), TimeDateStamp: req.GetTimeDateStamp(),
		MajorVersion: major, MinorVersion: minor, NumberOfNamedEntries: named,
		NumberOfIdEntries: ids,
	}, nil
}

func protoResourceDirectoryEntries(req []*clientpb.IMAGE_RESOURCE_DIRECTORY_ENTRY) []*IMAGE_RESOURCE_DIRECTORY_ENTRY {
	if len(req) == 0 {
		return nil
	}
	entries := make([]*IMAGE_RESOURCE_DIRECTORY_ENTRY, 0, len(req))
	for _, entry := range req {
		if entry != nil {
			entries = append(entries, &IMAGE_RESOURCE_DIRECTORY_ENTRY{Name: entry.GetName(), OffsetToData: entry.GetOffsetToData()})
		}
	}
	return entries
}

func protoResourceDataEntries(req []*clientpb.IMAGE_RESOURCE_DATA_ENTRY) []*IMAGE_RESOURCE_DATA_ENTRY {
	if len(req) == 0 {
		return nil
	}
	entries := make([]*IMAGE_RESOURCE_DATA_ENTRY, 0, len(req))
	for _, entry := range req {
		if entry != nil {
			entries = append(entries, &IMAGE_RESOURCE_DATA_ENTRY{
				OffsetToData: entry.GetOffsetToData(), Size: entry.GetSize(),
				CodePage: entry.GetCodePage(), Reserved: entry.GetReserved(),
			})
		}
	}
	return entries
}

func protoExportDirectory(req *clientpb.IMAGE_EXPORT_DIRECTORY) (*IMAGE_EXPORT_DIRECTORY, error) {
	if req == nil {
		return nil, nil
	}
	major, err := protoUint16("pe.export_directory.major_version", req.GetMajorVersion())
	if err != nil {
		return nil, err
	}
	minor, err := protoUint16("pe.export_directory.minor_version", req.GetMinorVersion())
	if err != nil {
		return nil, err
	}
	return &IMAGE_EXPORT_DIRECTORY{
		Characteristics: req.GetCharacteristics(), TimeDateStamp: req.GetTimeDateStamp(),
		MajorVersion: major, MinorVersion: minor, Name: req.GetName(), Base: req.GetBase(),
		NumberOfFunctions: req.GetNumberOfFunctions(), NumberOfNames: req.GetNumberOfNames(),
		AddressOfFunctions: req.GetAddressOfFunctions(), AddressOfNames: req.GetAddressOfNames(),
		AddressOfNameOrdinals: req.GetAddressOfNameOrdinals(),
	}, nil
}

func protoUint16(field string, value uint32) (uint16, error) {
	if value > 0xffff {
		return 0, fmt.Errorf("%s value %d exceeds uint16", field, value)
	}
	return uint16(value), nil
}
