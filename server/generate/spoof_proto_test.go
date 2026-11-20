package generate

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bishopfox/sliver/protobuf/clientpb"
)

func TestSpoofMetadataConfigFromProtoPreservesOverridesAndBounds(t *testing.T) {
	req := &clientpb.SpoofMetadataConfig{PE: &clientpb.PESpoofMetadataConfig{
		Source: &clientpb.SpoofMetadataFile{Name: "donor.exe", Data: []byte("donor")},
		Icon:   &clientpb.SpoofMetadataFile{Name: "icon.ico", Data: []byte("icon")},
		ResourceDirectory: &clientpb.IMAGE_RESOURCE_DIRECTORY{
			TimeDateStamp: 1234, MajorVersion: 4, NumberOfNamedEntries: 2,
		},
		ResourceDirectoryEntries: []*clientpb.IMAGE_RESOURCE_DIRECTORY_ENTRY{{Name: 7, OffsetToData: 9}},
		ResourceDataEntries:      []*clientpb.IMAGE_RESOURCE_DATA_ENTRY{{Size: 11, CodePage: 1200}},
		ExportDirectory:          &clientpb.IMAGE_EXPORT_DIRECTORY{TimeDateStamp: 5678, NumberOfNames: 3},
	}}
	got, err := SpoofMetadataConfigFromProto(req)
	if err != nil {
		t.Fatal(err)
	}
	if got.PE == nil || !bytes.Equal(got.PE.Source.Data, []byte("donor")) ||
		got.PE.Icon.Name != "icon.ico" || got.PE.ResourceDirectory.TimeDateStamp != 1234 ||
		len(got.PE.ResourceDirectoryEntries) != 1 || got.PE.ResourceDirectoryEntries[0].OffsetToData != 9 ||
		len(got.PE.ResourceDataEntries) != 1 || got.PE.ResourceDataEntries[0].CodePage != 1200 ||
		got.PE.ExportDirectory.TimeDateStamp != 5678 || got.PE.ExportDirectory.NumberOfNames != 3 {
		t.Fatalf("metadata conversion dropped an override: %+v", got.PE)
	}
	req.PE.ResourceDirectory.MajorVersion = 1 << 16
	if _, err := SpoofMetadataConfigFromProto(req); err == nil || !strings.Contains(err.Error(), "major_version") {
		t.Fatalf("out-of-range PE field was accepted: %v", err)
	}
}
