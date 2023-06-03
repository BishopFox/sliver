// Copyright 2010 The win Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package win

import (
	"golang.org/x/sys/windows"
	"syscall"
	"unsafe"
)

type CSIDL uint32
type HDROP HANDLE

const (
	CSIDL_DESKTOP                 = 0x00
	CSIDL_INTERNET                = 0x01
	CSIDL_PROGRAMS                = 0x02
	CSIDL_CONTROLS                = 0x03
	CSIDL_PRINTERS                = 0x04
	CSIDL_PERSONAL                = 0x05
	CSIDL_FAVORITES               = 0x06
	CSIDL_STARTUP                 = 0x07
	CSIDL_RECENT                  = 0x08
	CSIDL_SENDTO                  = 0x09
	CSIDL_BITBUCKET               = 0x0A
	CSIDL_STARTMENU               = 0x0B
	CSIDL_MYDOCUMENTS             = 0x0C
	CSIDL_MYMUSIC                 = 0x0D
	CSIDL_MYVIDEO                 = 0x0E
	CSIDL_DESKTOPDIRECTORY        = 0x10
	CSIDL_DRIVES                  = 0x11
	CSIDL_NETWORK                 = 0x12
	CSIDL_NETHOOD                 = 0x13
	CSIDL_FONTS                   = 0x14
	CSIDL_TEMPLATES               = 0x15
	CSIDL_COMMON_STARTMENU        = 0x16
	CSIDL_COMMON_PROGRAMS         = 0x17
	CSIDL_COMMON_STARTUP          = 0x18
	CSIDL_COMMON_DESKTOPDIRECTORY = 0x19
	CSIDL_APPDATA                 = 0x1A
	CSIDL_PRINTHOOD               = 0x1B
	CSIDL_LOCAL_APPDATA           = 0x1C
	CSIDL_ALTSTARTUP              = 0x1D
	CSIDL_COMMON_ALTSTARTUP       = 0x1E
	CSIDL_COMMON_FAVORITES        = 0x1F
	CSIDL_INTERNET_CACHE          = 0x20
	CSIDL_COOKIES                 = 0x21
	CSIDL_HISTORY                 = 0x22
	CSIDL_COMMON_APPDATA          = 0x23
	CSIDL_WINDOWS                 = 0x24
	CSIDL_SYSTEM                  = 0x25
	CSIDL_PROGRAM_FILES           = 0x26
	CSIDL_MYPICTURES              = 0x27
	CSIDL_PROFILE                 = 0x28
	CSIDL_SYSTEMX86               = 0x29
	CSIDL_PROGRAM_FILESX86        = 0x2A
	CSIDL_PROGRAM_FILES_COMMON    = 0x2B
	CSIDL_PROGRAM_FILES_COMMONX86 = 0x2C
	CSIDL_COMMON_TEMPLATES        = 0x2D
	CSIDL_COMMON_DOCUMENTS        = 0x2E
	CSIDL_COMMON_ADMINTOOLS       = 0x2F
	CSIDL_ADMINTOOLS              = 0x30
	CSIDL_CONNECTIONS             = 0x31
	CSIDL_COMMON_MUSIC            = 0x35
	CSIDL_COMMON_PICTURES         = 0x36
	CSIDL_COMMON_VIDEO            = 0x37
	CSIDL_RESOURCES               = 0x38
	CSIDL_RESOURCES_LOCALIZED     = 0x39
	CSIDL_COMMON_OEM_LINKS        = 0x3A
	CSIDL_CDBURN_AREA             = 0x3B
	CSIDL_COMPUTERSNEARME         = 0x3D
	CSIDL_FLAG_CREATE             = 0x8000
	CSIDL_FLAG_DONT_VERIFY        = 0x4000
	CSIDL_FLAG_NO_ALIAS           = 0x1000
	CSIDL_FLAG_PER_USER_INIT      = 0x8000
	CSIDL_FLAG_MASK               = 0xFF00
)

// NotifyIcon flags
const (
	NIF_MESSAGE  = 0x00000001
	NIF_ICON     = 0x00000002
	NIF_TIP      = 0x00000004
	NIF_STATE    = 0x00000008
	NIF_INFO     = 0x00000010
	NIF_GUID     = 0x00000020
	NIF_REALTIME = 0x00000040
	NIF_SHOWTIP  = 0x00000080
)

// NotifyIcon messages
const (
	NIM_ADD        = 0x00000000
	NIM_MODIFY     = 0x00000001
	NIM_DELETE     = 0x00000002
	NIM_SETFOCUS   = 0x00000003
	NIM_SETVERSION = 0x00000004
)

// NotifyIcon states
const (
	NIS_HIDDEN     = 0x00000001
	NIS_SHAREDICON = 0x00000002
)

// NotifyIcon info flags
const (
	NIIF_NONE               = 0x00000000
	NIIF_INFO               = 0x00000001
	NIIF_WARNING            = 0x00000002
	NIIF_ERROR              = 0x00000003
	NIIF_USER               = 0x00000004
	NIIF_NOSOUND            = 0x00000010
	NIIF_LARGE_ICON         = 0x00000020
	NIIF_RESPECT_QUIET_TIME = 0x00000080
)

// NotifyIcon notifications
const (
	NIN_SELECT           = WM_USER + 0
	NIN_KEYSELECT        = WM_USER + 1
	NIN_BALLOONSHOW      = WM_USER + 2
	NIN_BALLOONHIDE      = WM_USER + 3
	NIN_BALLOONTIMEOUT   = WM_USER + 4
	NIN_BALLOONUSERCLICK = WM_USER + 5
	NIN_POPUPOPEN        = WM_USER + 6
	NIN_POPUPCLOSE       = WM_USER + 7
)

// NotifyIcon versions
const (
	NOTIFYICON_VERSION   = 3
	NOTIFYICON_VERSION_4 = 4
)

// SHGetFileInfo flags
const (
	SHGFI_LARGEICON         = 0x000000000
	SHGFI_SMALLICON         = 0x000000001
	SHGFI_OPENICON          = 0x000000002
	SHGFI_SHELLICONSIZE     = 0x000000004
	SHGFI_PIDL              = 0x000000008
	SHGFI_USEFILEATTRIBUTES = 0x000000010
	SHGFI_ADDOVERLAYS       = 0x000000020
	SHGFI_OVERLAYINDEX      = 0x000000040
	SHGFI_ICON              = 0x000000100
	SHGFI_DISPLAYNAME       = 0x000000200
	SHGFI_TYPENAME          = 0x000000400
	SHGFI_ATTRIBUTES        = 0x000000800
	SHGFI_ICONLOCATION      = 0x000001000
	SHGFI_EXETYPE           = 0x000002000
	SHGFI_SYSICONINDEX      = 0x000004000
	SHGFI_LINKOVERLAY       = 0x000008000
	SHGFI_SELECTED          = 0x000010000
	SHGFI_ATTR_SPECIFIED    = 0x000020000
)

// SHGetStockIconInfo flags
const (
	SHGSI_ICONLOCATION  = 0
	SHGSI_ICON          = 0x000000100
	SHGSI_SYSICONINDEX  = 0x000004000
	SHGSI_LINKOVERLAY   = 0x000008000
	SHGSI_SELECTED      = 0x000010000
	SHGSI_LARGEICON     = 0x000000000
	SHGSI_SMALLICON     = 0x000000001
	SHGSI_SHELLICONSIZE = 0x000000004
)

// SHSTOCKICONID values
const (
	SIID_DOCNOASSOC        = 0
	SIID_DOCASSOC          = 1
	SIID_APPLICATION       = 2
	SIID_FOLDER            = 3
	SIID_FOLDEROPEN        = 4
	SIID_DRIVE525          = 5
	SIID_DRIVE35           = 6
	SIID_DRIVEREMOVE       = 7
	SIID_DRIVEFIXED        = 8
	SIID_DRIVENET          = 9
	SIID_DRIVENETDISABLED  = 10
	SIID_DRIVECD           = 11
	SIID_DRIVERAM          = 12
	SIID_WORLD             = 13
	SIID_SERVER            = 15
	SIID_PRINTER           = 16
	SIID_MYNETWORK         = 17
	SIID_FIND              = 22
	SIID_HELP              = 23
	SIID_SHARE             = 28
	SIID_LINK              = 29
	SIID_SLOWFILE          = 30
	SIID_RECYCLER          = 31
	SIID_RECYCLERFULL      = 32
	SIID_MEDIACDAUDIO      = 40
	SIID_LOCK              = 47
	SIID_AUTOLIST          = 49
	SIID_PRINTERNET        = 50
	SIID_SERVERSHARE       = 51
	SIID_PRINTERFAX        = 52
	SIID_PRINTERFAXNET     = 53
	SIID_PRINTERFILE       = 54
	SIID_STACK             = 55
	SIID_MEDIASVCD         = 56
	SIID_STUFFEDFOLDER     = 57
	SIID_DRIVEUNKNOWN      = 58
	SIID_DRIVEDVD          = 59
	SIID_MEDIADVD          = 60
	SIID_MEDIADVDRAM       = 61
	SIID_MEDIADVDRW        = 62
	SIID_MEDIADVDR         = 63
	SIID_MEDIADVDROM       = 64
	SIID_MEDIACDAUDIOPLUS  = 65
	SIID_MEDIACDRW         = 66
	SIID_MEDIACDR          = 67
	SIID_MEDIACDBURN       = 68
	SIID_MEDIABLANKCD      = 69
	SIID_MEDIACDROM        = 70
	SIID_AUDIOFILES        = 71
	SIID_IMAGEFILES        = 72
	SIID_VIDEOFILES        = 73
	SIID_MIXEDFILES        = 74
	SIID_FOLDERBACK        = 75
	SIID_FOLDERFRONT       = 76
	SIID_SHIELD            = 77
	SIID_WARNING           = 78
	SIID_INFO              = 79
	SIID_ERROR             = 80
	SIID_KEY               = 81
	SIID_SOFTWARE          = 82
	SIID_RENAME            = 83
	SIID_DELETE            = 84
	SIID_MEDIAAUDIODVD     = 85
	SIID_MEDIAMOVIEDVD     = 86
	SIID_MEDIAENHANCEDCD   = 87
	SIID_MEDIAENHANCEDDVD  = 88
	SIID_MEDIAHDDVD        = 89
	SIID_MEDIABLURAY       = 90
	SIID_MEDIAVCD          = 91
	SIID_MEDIADVDPLUSR     = 92
	SIID_MEDIADVDPLUSRW    = 93
	SIID_DESKTOPPC         = 94
	SIID_MOBILEPC          = 95
	SIID_USERS             = 96
	SIID_MEDIASMARTMEDIA   = 97
	SIID_MEDIACOMPACTFLASH = 98
	SIID_DEVICECELLPHONE   = 99
	SIID_DEVICECAMERA      = 100
	SIID_DEVICEVIDEOCAMERA = 101
	SIID_DEVICEAUDIOPLAYER = 102
	SIID_NETWORKCONNECT    = 103
	SIID_INTERNET          = 104
	SIID_ZIPFILE           = 105
	SIID_SETTINGS          = 106
	SIID_DRIVEHDDVD        = 132
	SIID_DRIVEBD           = 133
	SIID_MEDIAHDDVDROM     = 134
	SIID_MEDIAHDDVDR       = 135
	SIID_MEDIAHDDVDRAM     = 136
	SIID_MEDIABDROM        = 137
	SIID_MEDIABDR          = 138
	SIID_MEDIABDRE         = 139
	SIID_CLUSTEREDDRIVE    = 140
	SIID_MAX_ICONS         = 175
)

type NOTIFYICONDATA struct {
	CbSize           uint32
	HWnd             HWND
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            HICON
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         syscall.GUID
	HBalloonIcon     HICON
}

type SHFILEINFO struct {
	HIcon         HICON
	IIcon         int32
	DwAttributes  uint32
	SzDisplayName [MAX_PATH]uint16
	SzTypeName    [80]uint16
}

type BROWSEINFO struct {
	HwndOwner      HWND
	PidlRoot       uintptr
	PszDisplayName *uint16
	LpszTitle      *uint16
	UlFlags        uint32
	Lpfn           uintptr
	LParam         uintptr
	IImage         int32
}

type SHSTOCKICONINFO struct {
	CbSize         uint32
	HIcon          HICON
	ISysImageIndex int32
	IIcon          int32
	SzPath         [MAX_PATH]uint16
}

var (
	// Library
	libshell32 *windows.LazyDLL

	// Functions
	dragAcceptFiles        *windows.LazyProc
	dragFinish             *windows.LazyProc
	dragQueryFile          *windows.LazyProc
	extractIcon            *windows.LazyProc
	shBrowseForFolder      *windows.LazyProc
	shDefExtractIcon       *windows.LazyProc
	shGetFileInfo          *windows.LazyProc
	shGetPathFromIDList    *windows.LazyProc
	shGetSpecialFolderPath *windows.LazyProc
	shParseDisplayName     *windows.LazyProc
	shGetStockIconInfo     *windows.LazyProc
	shellExecute           *windows.LazyProc
	shell_NotifyIcon       *windows.LazyProc
)

func init() {
	// Library
	libshell32 = windows.NewLazySystemDLL("shell32.dll")

	// Functions
	dragAcceptFiles = libshell32.NewProc("DragAcceptFiles")
	dragFinish = libshell32.NewProc("DragFinish")
	dragQueryFile = libshell32.NewProc("DragQueryFileW")
	extractIcon = libshell32.NewProc("ExtractIconW")
	shBrowseForFolder = libshell32.NewProc("SHBrowseForFolderW")
	shDefExtractIcon = libshell32.NewProc("SHDefExtractIconW")
	shGetFileInfo = libshell32.NewProc("SHGetFileInfoW")
	shGetPathFromIDList = libshell32.NewProc("SHGetPathFromIDListW")
	shGetSpecialFolderPath = libshell32.NewProc("SHGetSpecialFolderPathW")
	shGetStockIconInfo = libshell32.NewProc("SHGetStockIconInfo")
	shellExecute = libshell32.NewProc("ShellExecuteW")
	shell_NotifyIcon = libshell32.NewProc("Shell_NotifyIconW")
	shParseDisplayName = libshell32.NewProc("SHParseDisplayName")
}

func DragAcceptFiles(hWnd HWND, fAccept bool) bool {
	ret, _, _ := syscall.Syscall(dragAcceptFiles.Addr(), 2,
		uintptr(hWnd),
		uintptr(BoolToBOOL(fAccept)),
		0)

	return ret != 0
}

func DragQueryFile(hDrop HDROP, iFile uint, lpszFile *uint16, cch uint) uint {
	ret, _, _ := syscall.Syscall6(dragQueryFile.Addr(), 4,
		uintptr(hDrop),
		uintptr(iFile),
		uintptr(unsafe.Pointer(lpszFile)),
		uintptr(cch),
		0,
		0)

	return uint(ret)
}

func DragFinish(hDrop HDROP) {
	syscall.Syscall(dragAcceptFiles.Addr(), 1,
		uintptr(hDrop),
		0,
		0)
}

func ExtractIcon(hInst HINSTANCE, exeFileName *uint16, iconIndex int32) HICON {
	ret, _, _ := syscall.Syscall(extractIcon.Addr(), 3,
		uintptr(hInst),
		uintptr(unsafe.Pointer(exeFileName)),
		uintptr(iconIndex))

	return HICON(ret)
}

func SHBrowseForFolder(lpbi *BROWSEINFO) uintptr {
	ret, _, _ := syscall.Syscall(shBrowseForFolder.Addr(), 1,
		uintptr(unsafe.Pointer(lpbi)),
		0,
		0)

	return ret
}

func SHDefExtractIcon(pszIconFile *uint16, iIndex int32, uFlags uint32, phiconLarge, phiconSmall *HICON, nIconSize uint32) HRESULT {
	ret, _, _ := syscall.Syscall6(shDefExtractIcon.Addr(), 6,
		uintptr(unsafe.Pointer(pszIconFile)),
		uintptr(iIndex),
		uintptr(uFlags),
		uintptr(unsafe.Pointer(phiconLarge)),
		uintptr(unsafe.Pointer(phiconSmall)),
		uintptr(nIconSize))

	return HRESULT(ret)
}

func SHGetFileInfo(pszPath *uint16, dwFileAttributes uint32, psfi *SHFILEINFO, cbFileInfo, uFlags uint32) uintptr {
	ret, _, _ := syscall.Syscall6(shGetFileInfo.Addr(), 5,
		uintptr(unsafe.Pointer(pszPath)),
		uintptr(dwFileAttributes),
		uintptr(unsafe.Pointer(psfi)),
		uintptr(cbFileInfo),
		uintptr(uFlags),
		0)

	return ret
}

func SHGetPathFromIDList(pidl uintptr, pszPath *uint16) bool {
	ret, _, _ := syscall.Syscall(shGetPathFromIDList.Addr(), 2,
		pidl,
		uintptr(unsafe.Pointer(pszPath)),
		0)

	return ret != 0
}

func SHGetSpecialFolderPath(hwndOwner HWND, lpszPath *uint16, csidl CSIDL, fCreate bool) bool {
	ret, _, _ := syscall.Syscall6(shGetSpecialFolderPath.Addr(), 4,
		uintptr(hwndOwner),
		uintptr(unsafe.Pointer(lpszPath)),
		uintptr(csidl),
		uintptr(BoolToBOOL(fCreate)),
		0,
		0)

	return ret != 0
}

func SHParseDisplayName(pszName *uint16, pbc uintptr, ppidl *uintptr, sfgaoIn uint32, psfgaoOut *uint32) HRESULT {
	ret, _, _ := syscall.Syscall6(shParseDisplayName.Addr(), 5,
		uintptr(unsafe.Pointer(pszName)),
		pbc,
		uintptr(unsafe.Pointer(ppidl)),
		0,
		uintptr(unsafe.Pointer(psfgaoOut)),
		0)

	return HRESULT(ret)
}

func SHGetStockIconInfo(stockIconId int32, uFlags uint32, stockIcon *SHSTOCKICONINFO) HRESULT {
	if shGetStockIconInfo.Find() != nil {
		return HRESULT(0)
	}
	ret, _, _ := syscall.Syscall6(shGetStockIconInfo.Addr(), 3,
		uintptr(stockIconId),
		uintptr(uFlags),
		uintptr(unsafe.Pointer(stockIcon)),
		0,
		0,
		0,
	)
	return HRESULT(ret)
}

func ShellExecute(hWnd HWND, verb *uint16, file *uint16, args *uint16, cwd *uint16, showCmd int) bool {
	ret, _, _ := syscall.Syscall6(shellExecute.Addr(), 6,
		uintptr(hWnd),
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		uintptr(unsafe.Pointer(args)),
		uintptr(unsafe.Pointer(cwd)),
		uintptr(showCmd),
	)
	return ret != 0
}

func Shell_NotifyIcon(dwMessage uint32, lpdata *NOTIFYICONDATA) bool {
	ret, _, _ := syscall.Syscall(shell_NotifyIcon.Addr(), 2,
		uintptr(dwMessage),
		uintptr(unsafe.Pointer(lpdata)),
		0)

	return ret != 0
}
