package universal

type DyldCacheHeader struct {
	Magic                         [16]byte
	MappingOffset                 uint32
	MappingCount                  uint32
	ImagesOffsetOld               uint32
	ImagesCountOld                uint32
	DyldBaseAddress               uint64
	CodeSignatureOffset           uint64
	CodeSignatureSize             uint64
	SlideInfoOffsetUnused         uint64
	SlideInfoSizeUnused           uint64
	LocalSymbolsOffset            uint64
	LocalSymbolsSize              uint64
	UUID                          [16]byte
	CacheType                     uint64
	BranchPoolOffset              uint32
	BranchPoolCount               uint32
	AccelerateInfoAddr            uint64
	AccelerateInfoSize            uint64
	ImagesTextOffset              uint64
	ImagesTextCount               uint64
	PatchInfoAddr                 uint64
	PatchInfoSize                 uint64
	OtherImageGroupAddrUnused     uint64
	OtherImageGroupSizeUnused     uint64
	ProgClosuresAddr              uint64
	ProgClosuresSize              uint64
	ProgClosuresTrieAddr          uint64
	ProgClosuresTrieSize          uint64
	Platform                      uint32
	FormatVersion                 uint32
	SharedRegionStart             uint64
	SharedRegionSize              uint64
	MaxSlide                      uint64
	DylibsImageArrayAddr          uint64
	DylibsImageArraySize          uint64
	DylibsTrieAddr                uint64
	DylibsTrieSize                uint64
	OtherImageArrayAddr           uint64
	OtherImageArraySize           uint64
	OtherTrieAddr                 uint64
	OtherTrieSize                 uint64
	MappingWithSlideOffset        uint32
	MappingWithSlideCount         uint32
	DylibsPBLStateArrayAddrUnused uint64
	DylibsPBLSetAddr              uint64
	ProgramsPBLSetPoolAddr        uint64
	ProgramsPBLSetPoolSize        uint64
	ProgramTrieAddr               uint64
	ProgramTrieSize               uint32
	OsVersion                     uint32
	AltPlatform                   uint32
	AltOsVersion                  uint32
	SwiftOptsOffset               uint64
	SwiftOptsSize                 uint64
	SubCacheArrayOffset           uint32
	SubCacheArrayCount            uint32
	SymbolFileUUID                [16]byte
	RosettaReadOnlyAddr           uint64
	RosettaReadOnlySize           uint64
	RosettaReadWriteAddr          uint64
	RosettaReadWriteSize          uint64
	ImagesOffset                  uint32
	ImagesCount                   uint32
}

type DyldCacheMappingInfo struct {
	Address    uint64
	Size       uint64
	FileOffset uint64
	MaxProt    uint32
	InitProt   uint32
}

type DyldCacheImageInfo struct {
	Address        uint64
	ModTime        uint64
	Inode          uint64
	PathFileOffset uint32
	Pad            uint32
}

type SharedFileMapping struct {
	Address    uint64
	Size       uint64
	FileOffset uint64
	MaxProt    uint32
	InitProt   uint32
}
