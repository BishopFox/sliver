package syscalls

import (
	"golang.org/x/sys/windows"
)

const (
	PSS_CAPTURE_NONE = iota
	PSS_CAPTURE_VA_CLONE
	PSS_CAPTURE_RESERVED_00000002
	PSS_CAPTURE_HANDLES
	PSS_CAPTURE_HANDLE_NAME_INFORMATION
	PSS_CAPTURE_HANDLE_BASIC_INFORMATION
	PSS_CAPTURE_HANDLE_TYPE_SPECIFIC_INFORMATION
	PSS_CAPTURE_HANDLE_TRACE
	PSS_CAPTURE_THREADS
	PSS_CAPTURE_THREAD_CONTEXT
	PSS_CAPTURE_THREAD_CONTEXT_EXTENDED
	PSS_CAPTURE_RESERVED_00000400
	PSS_CAPTURE_VA_SPACE
	PSS_CAPTURE_VA_SPACE_SECTION_INFORMATION
	PSS_CAPTURE_IPT_TRACE
	PSS_CAPTURE_RESERVED_00004000
	PSS_CREATE_BREAKAWAY_OPTIONAL
	PSS_CREATE_BREAKAWAY
	PSS_CREATE_FORCE_BREAKAWAY
	PSS_CREATE_USE_VM_ALLOCATIONS
	PSS_CREATE_MEASURE_PERFORMANCE
	PSS_CREATE_RELEASE_SECTION

	CONTEXT_i386               = 0x10000
	CONTEXT_i486               = 0x10000             //  same as i386
	CONTEXT_CONTROL            = CONTEXT_i386 | 0x01 // SS:SP, CS:IP, FLAGS, BP
	CONTEXT_INTEGER            = CONTEXT_i386 | 0x02 // AX, BX, CX, DX, SI, DI
	CONTEXT_SEGMENTS           = CONTEXT_i386 | 0x0  // DS, ES, FS, GS
	CONTEXT_FLOATING_POINT     = CONTEXT_i386 | 0x08 // 387 state
	CONTEXT_DEBUG_REGISTERS    = CONTEXT_i386 | 0x10 // DB 0-3,6,7
	CONTEXT_EXTENDED_REGISTERS = CONTEXT_i386 | 0x20 // cpu specific extensions
	CONTEXT_FULL               = CONTEXT_CONTROL | CONTEXT_INTEGER | CONTEXT_SEGMENTS
	CONTEXT_ALL                = CONTEXT_CONTROL | CONTEXT_INTEGER | CONTEXT_SEGMENTS | CONTEXT_FLOATING_POINT | CONTEXT_DEBUG_REGISTERS | CONTEXT_EXTENDED_REGISTERS

	PROC_THREAD_ATTRIBUTE_PARENT_PROCESS = 0x00020000
	LOGON32_LOGON_INTERACTIVE            = 2
	LOGON32_LOGON_NETWORK                = 3
	LOGON32_LOGON_BATCH                  = 4
	LOGON32_LOGON_SERVICE                = 5
	LOGON32_LOGON_UNLOCK                 = 7
	LOGON32_LOGON_NETWORK_CLEARTEXT      = 8
	LOGON32_LOGON_NEW_CREDENTIALS        = 9

	LOGON32_PROVIDER_DEFAULT = 0
	LOGON32_PROVIDER_WINNT35 = 1
	LOGON32_PROVIDER_WINNT40 = 2
	LOGON32_PROVIDER_WINNT50 = 3
	PROCESS_DUP_HANDLE       = 0x0040
	DUPLICATE_SAME_ACCESS    = 0x00000002
	STILL_ACTIVE             = 259
)

type StartupInfoEx struct {
	windows.StartupInfo
	AttributeList *PROC_THREAD_ATTRIBUTE_LIST
}

type PROC_THREAD_ATTRIBUTE_LIST struct {
	dwFlags  uint32
	size     uint64
	count    uint64
	reserved uint64
	unknown  *uint64
	entries  []*PROC_THREAD_ATTRIBUTE_ENTRY
}

type PROC_THREAD_ATTRIBUTE_ENTRY struct {
	attribute *uint32
	cbSize    uintptr
	lpValue   uintptr
}

type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type RGBQUAD struct {
	RgbBlue     byte
	RgbGreen    byte
	RgbRed      byte
	RgbReserved byte
}

type BITMAPINFO struct {
	BmiHeader BITMAPINFOHEADER
	BmiColors *RGBQUAD
}

type RECT struct {
	Left, Top, Right, Bottom int32
}

type POINT struct {
	X, Y int32
}

type MONITORINFO struct {
	CbSize    uint32
	RcMonitor RECT
	RcWork    RECT
	DwFlags   uint32
}

type ProcessMemoryCounters struct {
	Cb                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         int
	WorkingSetSize             int
	QuotaPeakPagedPoolUsage    int
	QuotaPagedPoolUsage        int
	QuotaPeakNonPagedPoolUsage int
	QuotaNonPagedPoolUsage     int
	PagefileUsage              int
	PeakPagefileUsage          int
}

const (
	CCHDEVICENAME = 32
	CCHFORMNAME   = 32
)

// Ternary raster operations
const (
	SRCCOPY        = 0x00CC0020
	SRCPAINT       = 0x00EE0086
	SRCAND         = 0x008800C6
	SRCINVERT      = 0x00660046
	SRCERASE       = 0x00440328
	NOTSRCCOPY     = 0x00330008
	NOTSRCERASE    = 0x001100A6
	MERGECOPY      = 0x00C000CA
	MERGEPAINT     = 0x00BB0226
	PATCOPY        = 0x00F00021
	PATPAINT       = 0x00FB0A09
	PATINVERT      = 0x005A0049
	DSTINVERT      = 0x00550009
	BLACKNESS      = 0x00000042
	WHITENESS      = 0x00FF0062
	NOMIRRORBITMAP = 0x80000000
	CAPTUREBLT     = 0x40000000
)

// GlobalAlloc flags
const (
	GHND          = 0x0042
	GMEM_FIXED    = 0x0000
	GMEM_MOVEABLE = 0x0002
	GMEM_ZEROINIT = 0x0040
	GPTR          = GMEM_FIXED | GMEM_ZEROINIT
)

const (
	MAX_MODULE_NAME32 = 255
	MAX_PATH          = 260
)

type MODULEENTRY32W struct {
	DwSize        uint32
	Th32ModuleID  uint32
	Th32ProcessID uint32
	GlblcntUsage  uint32
	ProccntUsage  uint32
	ModBaseAddr   uintptr
	ModBaseSize   uint32
	HModule       uintptr
	SzModule      [MAX_MODULE_NAME32 + 1]uint16
	SzExePath     [MAX_PATH]uint16
}
