package d3d

type _DXGI_RATIONAL struct {
	Numerator   uint32
	Denominator uint32
}
type _DXGI_MODE_DESC struct {
	Width            uint32
	Height           uint32
	Rational         _DXGI_RATIONAL
	Format           uint32 // DXGI_FORMAT
	ScanlineOrdering uint32 // DXGI_MODE_SCANLINE_ORDER
	Scaling          uint32 // DXGI_MODE_SCALING
}

type _DXGI_OUTDUPL_DESC struct {
	ModeDesc                   _DXGI_MODE_DESC
	Rotation                   uint32 // DXGI_MODE_ROTATION
	DesktopImageInSystemMemory uint32 // BOOL
}

type _DXGI_SAMPLE_DESC struct {
	Count   uint32
	Quality uint32
}

type POINT struct {
	X int32
	Y int32
}
type RECT struct {
	Left, Top, Right, Bottom int32
}

type _DXGI_OUTDUPL_MOVE_RECT struct {
	Src  POINT
	Dest RECT
}
type _DXGI_OUTDUPL_POINTER_POSITION struct {
	Position POINT
	Visible  uint32
}
type _DXGI_OUTDUPL_FRAME_INFO struct {
	LastPresentTime           int64
	LastMouseUpdateTime       int64
	AccumulatedFrames         uint32
	RectsCoalesced            uint32
	ProtectedContentMaskedOut uint32
	PointerPosition           _DXGI_OUTDUPL_POINTER_POSITION
	TotalMetadataBufferSize   uint32
	PointerShapeBufferSize    uint32
}
type DXGI_MAPPED_RECT struct {
	Pitch int32
	PBits uintptr
}

const (
	DXGI_FORMAT_R8G8B8A8_UNORM DXGI_FORMAT = 28
	DXGI_FORMAT_B8G8R8A8_UNORM DXGI_FORMAT = 87
)
