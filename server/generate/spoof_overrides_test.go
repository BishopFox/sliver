package generate

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestApplyResourceSectionOverrides(t *testing.T) {
	sectionData := makeTestResourceSectionData()

	updatedData, err := applyResourceSectionOverrides(
		sectionData,
		&IMAGE_RESOURCE_DIRECTORY{
			Characteristics:      0x10203040,
			TimeDateStamp:        0x50607080,
			MajorVersion:         9,
			MinorVersion:         7,
			NumberOfNamedEntries: 0,
			NumberOfIdEntries:    1,
		},
		[]*IMAGE_RESOURCE_DIRECTORY_ENTRY{
			{Name: 77, OffsetToData: 24},
		},
		[]*IMAGE_RESOURCE_DATA_ENTRY{
			{OffsetToData: 0x12345678, Size: 0x22, CodePage: 65001, Reserved: 1},
		},
	)
	if err != nil {
		t.Fatalf("applyResourceSectionOverrides() error: %v", err)
	}

	if got := binary.LittleEndian.Uint32(updatedData[0:4]); got != 0x10203040 {
		t.Fatalf("resource directory characteristics mismatch: got=0x%x", got)
	}
	if got := binary.LittleEndian.Uint32(updatedData[4:8]); got != 0x50607080 {
		t.Fatalf("resource directory timestamp mismatch: got=0x%x", got)
	}
	if got := binary.LittleEndian.Uint16(updatedData[8:10]); got != 9 {
		t.Fatalf("resource directory major_version mismatch: got=%d", got)
	}
	if got := binary.LittleEndian.Uint16(updatedData[10:12]); got != 7 {
		t.Fatalf("resource directory minor_version mismatch: got=%d", got)
	}
	if got := binary.LittleEndian.Uint32(updatedData[16:20]); got != 77 {
		t.Fatalf("resource directory entry name mismatch: got=%d", got)
	}
	if got := binary.LittleEndian.Uint32(updatedData[20:24]); got != 24 {
		t.Fatalf("resource directory entry offset mismatch: got=%d", got)
	}
	if got := binary.LittleEndian.Uint32(updatedData[24:28]); got != 0x12345678 {
		t.Fatalf("resource data entry offset_to_data mismatch: got=0x%x", got)
	}
	if got := binary.LittleEndian.Uint32(updatedData[28:32]); got != 0x22 {
		t.Fatalf("resource data entry size mismatch: got=%d", got)
	}
	if got := binary.LittleEndian.Uint32(updatedData[32:36]); got != 65001 {
		t.Fatalf("resource data entry code_page mismatch: got=%d", got)
	}
	if got := binary.LittleEndian.Uint32(updatedData[36:40]); got != 1 {
		t.Fatalf("resource data entry reserved mismatch: got=%d", got)
	}
}

func TestApplyResourceSectionOverridesRootEntryBounds(t *testing.T) {
	sectionData := makeTestResourceSectionData()

	_, err := applyResourceSectionOverrides(
		sectionData,
		nil,
		[]*IMAGE_RESOURCE_DIRECTORY_ENTRY{
			{Name: 1, OffsetToData: 24},
			{Name: 2, OffsetToData: 24},
		},
		nil,
	)
	if err == nil {
		t.Fatal("expected error for root resource directory entry bounds")
	}
	if !strings.Contains(err.Error(), "root directory only has 1 entries") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyExportDirectoryOverrides(t *testing.T) {
	sectionData := make([]byte, 80)
	sectionOffset := uint32(12)
	updatedData, err := applyExportDirectoryOverrides(sectionData, sectionOffset, &IMAGE_EXPORT_DIRECTORY{
		Characteristics:       0x11111111,
		TimeDateStamp:         0x22222222,
		MajorVersion:          3,
		MinorVersion:          4,
		Name:                  0x55555555,
		Base:                  0x66666666,
		NumberOfFunctions:     0x77777777,
		NumberOfNames:         0x88888888,
		AddressOfFunctions:    0x99999999,
		AddressOfNames:        0xaaaaaaaa,
		AddressOfNameOrdinals: 0xbbbbbbbb,
	})
	if err != nil {
		t.Fatalf("applyExportDirectoryOverrides() error: %v", err)
	}

	base := int(sectionOffset)
	if got := binary.LittleEndian.Uint32(updatedData[base : base+4]); got != 0x11111111 {
		t.Fatalf("export characteristics mismatch: got=0x%x", got)
	}
	if got := binary.LittleEndian.Uint32(updatedData[base+4 : base+8]); got != 0x22222222 {
		t.Fatalf("export timestamp mismatch: got=0x%x", got)
	}
	if got := binary.LittleEndian.Uint16(updatedData[base+8 : base+10]); got != 3 {
		t.Fatalf("export major_version mismatch: got=%d", got)
	}
	if got := binary.LittleEndian.Uint16(updatedData[base+10 : base+12]); got != 4 {
		t.Fatalf("export minor_version mismatch: got=%d", got)
	}
	if got := binary.LittleEndian.Uint32(updatedData[base+12 : base+16]); got != 0x55555555 {
		t.Fatalf("export name mismatch: got=0x%x", got)
	}
	if got := binary.LittleEndian.Uint32(updatedData[base+16 : base+20]); got != 0x66666666 {
		t.Fatalf("export base mismatch: got=0x%x", got)
	}
	if got := binary.LittleEndian.Uint32(updatedData[base+20 : base+24]); got != 0x77777777 {
		t.Fatalf("export number_of_functions mismatch: got=0x%x", got)
	}
	if got := binary.LittleEndian.Uint32(updatedData[base+24 : base+28]); got != 0x88888888 {
		t.Fatalf("export number_of_names mismatch: got=0x%x", got)
	}
	if got := binary.LittleEndian.Uint32(updatedData[base+28 : base+32]); got != 0x99999999 {
		t.Fatalf("export address_of_functions mismatch: got=0x%x", got)
	}
	if got := binary.LittleEndian.Uint32(updatedData[base+32 : base+36]); got != 0xaaaaaaaa {
		t.Fatalf("export address_of_names mismatch: got=0x%x", got)
	}
	if got := binary.LittleEndian.Uint32(updatedData[base+36 : base+40]); got != 0xbbbbbbbb {
		t.Fatalf("export address_of_name_ordinals mismatch: got=0x%x", got)
	}
}

func makeTestResourceSectionData() []byte {
	data := make([]byte, 40)

	// Root IMAGE_RESOURCE_DIRECTORY (16 bytes)
	binary.LittleEndian.PutUint32(data[0:4], 0x01020304) // Characteristics
	binary.LittleEndian.PutUint32(data[4:8], 0x05060708) // TimeDateStamp
	binary.LittleEndian.PutUint16(data[8:10], 1)         // MajorVersion
	binary.LittleEndian.PutUint16(data[10:12], 2)        // MinorVersion
	binary.LittleEndian.PutUint16(data[12:14], 0)        // NumberOfNamedEntries
	binary.LittleEndian.PutUint16(data[14:16], 1)        // NumberOfIdEntries

	// One IMAGE_RESOURCE_DIRECTORY_ENTRY at offset 16.
	binary.LittleEndian.PutUint32(data[16:20], 10) // Name
	binary.LittleEndian.PutUint32(data[20:24], 24) // OffsetToData (points to IMAGE_RESOURCE_DATA_ENTRY)

	// One IMAGE_RESOURCE_DATA_ENTRY at offset 24.
	binary.LittleEndian.PutUint32(data[24:28], 0x2000) // OffsetToData
	binary.LittleEndian.PutUint32(data[28:32], 0x40)   // Size
	binary.LittleEndian.PutUint32(data[32:36], 0)      // CodePage
	binary.LittleEndian.PutUint32(data[36:40], 0)      // Reserved

	return data
}
