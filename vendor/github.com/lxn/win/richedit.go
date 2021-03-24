// Copyright 2010 The win Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package win

const (
	// NOTE:  MSFTEDIT.DLL only registers MSFTEDIT_CLASS.  If an application wants
	// to use the following RichEdit classes, it needs to load riched20.dll.
	// Otherwise, CreateWindow with RICHEDIT_CLASS will fail.
	// This also applies to any dialog that uses RICHEDIT_CLASS
	// RichEdit 2.0 Window Class
	MSFTEDIT_CLASS = "RICHEDIT50W"
	RICHEDIT_CLASS = "RichEdit20W"
)

// RichEdit messages
const (
	EM_CANPASTE           = WM_USER + 50
	EM_DISPLAYBAND        = WM_USER + 51
	EM_EXGETSEL           = WM_USER + 52
	EM_EXLIMITTEXT        = WM_USER + 53
	EM_EXLINEFROMCHAR     = WM_USER + 54
	EM_EXSETSEL           = WM_USER + 55
	EM_FINDTEXT           = WM_USER + 56
	EM_FORMATRANGE        = WM_USER + 57
	EM_GETCHARFORMAT      = WM_USER + 58
	EM_GETEVENTMASK       = WM_USER + 59
	EM_GETOLEINTERFACE    = WM_USER + 60
	EM_GETPARAFORMAT      = WM_USER + 61
	EM_GETSELTEXT         = WM_USER + 62
	EM_HIDESELECTION      = WM_USER + 63
	EM_PASTESPECIAL       = WM_USER + 64
	EM_REQUESTRESIZE      = WM_USER + 65
	EM_SELECTIONTYPE      = WM_USER + 66
	EM_SETBKGNDCOLOR      = WM_USER + 67
	EM_SETCHARFORMAT      = WM_USER + 68
	EM_SETEVENTMASK       = WM_USER + 69
	EM_SETOLECALLBACK     = WM_USER + 70
	EM_SETPARAFORMAT      = WM_USER + 71
	EM_SETTARGETDEVICE    = WM_USER + 72
	EM_STREAMIN           = WM_USER + 73
	EM_STREAMOUT          = WM_USER + 74
	EM_GETTEXTRANGE       = WM_USER + 75
	EM_FINDWORDBREAK      = WM_USER + 76
	EM_SETOPTIONS         = WM_USER + 77
	EM_GETOPTIONS         = WM_USER + 78
	EM_FINDTEXTEX         = WM_USER + 79
	EM_GETWORDBREAKPROCEX = WM_USER + 80
	EM_SETWORDBREAKPROCEX = WM_USER + 81
)

// RichEdit 2.0 messages
const (
	EM_SETUNDOLIMIT    = WM_USER + 82
	EM_REDO            = WM_USER + 84
	EM_CANREDO         = WM_USER + 85
	EM_GETUNDONAME     = WM_USER + 86
	EM_GETREDONAME     = WM_USER + 87
	EM_STOPGROUPTYPING = WM_USER + 88

	EM_SETTEXTMODE = WM_USER + 89
	EM_GETTEXTMODE = WM_USER + 90
)

type TEXTMODE int32

const (
	TM_PLAINTEXT       TEXTMODE = 1
	TM_RICHTEXT                 = 2 // Default behavior
	TM_SINGLELEVELUNDO          = 4
	TM_MULTILEVELUNDO           = 8 // Default behavior
	TM_SINGLECODEPAGE           = 16
	TM_MULTICODEPAGE            = 32 // Default behavior
)

const (
	EM_AUTOURLDETECT = WM_USER + 91
)

// RichEdit 8.0 messages
const (
	AURL_ENABLEURL          = 1
	AURL_ENABLEEMAILADDR    = 2
	AURL_ENABLETELNO        = 4
	AURL_ENABLEEAURLS       = 8
	AURL_ENABLEDRIVELETTERS = 16
	AURL_DISABLEMIXEDLGC    = 32 // Disable mixed Latin Greek Cyrillic IDNs
)

const (
	EM_GETAUTOURLDETECT = WM_USER + 92
	EM_SETPALETTE       = WM_USER + 93
	EM_GETTEXTEX        = WM_USER + 94
	EM_GETTEXTLENGTHEX  = WM_USER + 95
	EM_SHOWSCROLLBAR    = WM_USER + 96
	EM_SETTEXTEX        = WM_USER + 97
)

// East Asia specific messages
const (
	EM_SETPUNCTUATION  = WM_USER + 100
	EM_GETPUNCTUATION  = WM_USER + 101
	EM_SETWORDWRAPMODE = WM_USER + 102
	EM_GETWORDWRAPMODE = WM_USER + 103
	EM_SETIMECOLOR     = WM_USER + 104
	EM_GETIMECOLOR     = WM_USER + 105
	EM_SETIMEOPTIONS   = WM_USER + 106
	EM_GETIMEOPTIONS   = WM_USER + 107
	EM_CONVPOSITION    = WM_USER + 108
)

const (
	EM_SETLANGOPTIONS = WM_USER + 120
	EM_GETLANGOPTIONS = WM_USER + 121
	EM_GETIMECOMPMODE = WM_USER + 122

	EM_FINDTEXTW   = WM_USER + 123
	EM_FINDTEXTEXW = WM_USER + 124
)

// RE3.0 FE messages
const (
	EM_RECONVERSION   = WM_USER + 125
	EM_SETIMEMODEBIAS = WM_USER + 126
	EM_GETIMEMODEBIAS = WM_USER + 127
)

// BiDi specific messages
const (
	EM_SETBIDIOPTIONS = WM_USER + 200
	EM_GETBIDIOPTIONS = WM_USER + 201

	EM_SETTYPOGRAPHYOPTIONS = WM_USER + 202
	EM_GETTYPOGRAPHYOPTIONS = WM_USER + 203
)

// Extended edit style specific messages
const (
	EM_SETEDITSTYLE = WM_USER + 204
	EM_GETEDITSTYLE = WM_USER + 205
)

// Extended edit style masks
const (
	SES_EMULATESYSEDIT    = 1
	SES_BEEPONMAXTEXT     = 2
	SES_EXTENDBACKCOLOR   = 4
	SES_MAPCPS            = 8 // Obsolete (never used)
	SES_HYPERLINKTOOLTIPS = 8
	SES_EMULATE10         = 16 // Obsolete (never used)
	SES_DEFAULTLATINLIGA  = 16
	SES_USECRLF           = 32 // Obsolete (never used)
	SES_NOFOCUSLINKNOTIFY = 32
	SES_USEAIMM           = 64
	SES_NOIME             = 128

	SES_ALLOWBEEPS         = 256
	SES_UPPERCASE          = 512
	SES_LOWERCASE          = 1024
	SES_NOINPUTSEQUENCECHK = 2048
	SES_BIDI               = 4096
	SES_SCROLLONKILLFOCUS  = 8192
	SES_XLTCRCRLFTOCR      = 16384
	SES_DRAFTMODE          = 32768

	SES_USECTF               = 0x00010000
	SES_HIDEGRIDLINES        = 0x00020000
	SES_USEATFONT            = 0x00040000
	SES_CUSTOMLOOK           = 0x00080000
	SES_LBSCROLLNOTIFY       = 0x00100000
	SES_CTFALLOWEMBED        = 0x00200000
	SES_CTFALLOWSMARTTAG     = 0x00400000
	SES_CTFALLOWPROOFING     = 0x00800000
	SES_LOGICALCARET         = 0x01000000
	SES_WORDDRAGDROP         = 0x02000000
	SES_SMARTDRAGDROP        = 0x04000000
	SES_MULTISELECT          = 0x08000000
	SES_CTFNOLOCK            = 0x10000000
	SES_NOEALINEHEIGHTADJUST = 0x20000000
	SES_MAX                  = 0x20000000
)

// Options for EM_SETLANGOPTIONS and EM_GETLANGOPTIONS
const (
	IMF_AUTOKEYBOARD        = 0x0001
	IMF_AUTOFONT            = 0x0002
	IMF_IMECANCELCOMPLETE   = 0x0004 // High completes comp string when aborting, low cancels
	IMF_IMEALWAYSSENDNOTIFY = 0x0008
	IMF_AUTOFONTSIZEADJUST  = 0x0010
	IMF_UIFONTS             = 0x0020
	IMF_NOIMPLICITLANG      = 0x0040
	IMF_DUALFONT            = 0x0080
	IMF_NOKBDLIDFIXUP       = 0x0200
	IMF_NORTFFONTSUBSTITUTE = 0x0400
	IMF_SPELLCHECKING       = 0x0800
	IMF_TKBPREDICTION       = 0x1000
	IMF_IMEUIINTEGRATION    = 0x2000
)

// Values for EM_GETIMECOMPMODE
const (
	ICM_NOTOPEN    = 0x0000
	ICM_LEVEL3     = 0x0001
	ICM_LEVEL2     = 0x0002
	ICM_LEVEL2_5   = 0x0003
	ICM_LEVEL2_SUI = 0x0004
	ICM_CTF        = 0x0005
)

// Options for EM_SETTYPOGRAPHYOPTIONS
const (
	TO_ADVANCEDTYPOGRAPHY   = 0x0001
	TO_SIMPLELINEBREAK      = 0x0002
	TO_DISABLECUSTOMTEXTOUT = 0x0004
	TO_ADVANCEDLAYOUT       = 0x0008
)

// Pegasus outline mode messages (RE 3.0)
const (
	// Outline mode message
	EM_OUTLINE = WM_USER + 220

	// Message for getting and restoring scroll pos
	EM_GETSCROLLPOS = WM_USER + 221
	EM_SETSCROLLPOS = WM_USER + 222

	// Change fontsize in current selection by wParam
	EM_SETFONTSIZE = WM_USER + 223
	EM_GETZOOM     = WM_USER + 224
	EM_SETZOOM     = WM_USER + 225
	EM_GETVIEWKIND = WM_USER + 226
	EM_SETVIEWKIND = WM_USER + 227
)

// RichEdit 4.0 messages
const (
	EM_GETPAGE          = WM_USER + 228
	EM_SETPAGE          = WM_USER + 229
	EM_GETHYPHENATEINFO = WM_USER + 230
	EM_SETHYPHENATEINFO = WM_USER + 231

	EM_GETPAGEROTATE    = WM_USER + 235
	EM_SETPAGEROTATE    = WM_USER + 236
	EM_GETCTFMODEBIAS   = WM_USER + 237
	EM_SETCTFMODEBIAS   = WM_USER + 238
	EM_GETCTFOPENSTATUS = WM_USER + 240
	EM_SETCTFOPENSTATUS = WM_USER + 241
	EM_GETIMECOMPTEXT   = WM_USER + 242
	EM_ISIME            = WM_USER + 243
	EM_GETIMEPROPERTY   = WM_USER + 244
)

// These messages control what rich edit does when it comes accross
// OLE objects during RTF stream in.  Normally rich edit queries the client
// application only after OleLoad has been called.  With these messages it is possible to
// set the rich edit control to a mode where it will query the client application before
// OleLoad is called
const (
	EM_GETQUERYRTFOBJ = WM_USER + 269
	EM_SETQUERYRTFOBJ = WM_USER + 270
)

// EM_SETPAGEROTATE wparam values
const (
	EPR_0   = 0 // Text flows left to right and top to bottom
	EPR_270 = 1 // Text flows top to bottom and right to left
	EPR_180 = 2 // Text flows right to left and bottom to top
	EPR_90  = 3 // Text flows bottom to top and left to right
	EPR_SE  = 5 // Text flows top to bottom and left to right (Mongolian text layout)
)

// EM_SETCTFMODEBIAS wparam values
const (
	CTFMODEBIAS_DEFAULT               = 0x0000
	CTFMODEBIAS_FILENAME              = 0x0001
	CTFMODEBIAS_NAME                  = 0x0002
	CTFMODEBIAS_READING               = 0x0003
	CTFMODEBIAS_DATETIME              = 0x0004
	CTFMODEBIAS_CONVERSATION          = 0x0005
	CTFMODEBIAS_NUMERIC               = 0x0006
	CTFMODEBIAS_HIRAGANA              = 0x0007
	CTFMODEBIAS_KATAKANA              = 0x0008
	CTFMODEBIAS_HANGUL                = 0x0009
	CTFMODEBIAS_HALFWIDTHKATAKANA     = 0x000A
	CTFMODEBIAS_FULLWIDTHALPHANUMERIC = 0x000B
	CTFMODEBIAS_HALFWIDTHALPHANUMERIC = 0x000C
)

// EM_SETIMEMODEBIAS lparam values
const (
	IMF_SMODE_PLAURALCLAUSE = 0x0001
	IMF_SMODE_NONE          = 0x0002
)

// EM_GETIMECOMPTEXT wparam structure
type IMECOMPTEXT struct {
	// count of bytes in the output buffer.
	Cb int32

	// value specifying the composition string type.
	//	Currently only support ICT_RESULTREADSTR
	Flags uint32
}

const ICT_RESULTREADSTR = 1

// Outline mode wparam values
const (
	// Enter normal mode,  lparam ignored
	EMO_EXIT = 0

	// Enter outline mode, lparam ignored
	EMO_ENTER = 1

	// LOWORD(lparam) == 0 ==>
	//	promote  to body-text
	// LOWORD(lparam) != 0 ==>
	//	promote/demote current selection
	//	by indicated number of levels
	EMO_PROMOTE = 2

	// HIWORD(lparam) = EMO_EXPANDSELECTION
	//	-> expands selection to level
	//	indicated in LOWORD(lparam)
	//	LOWORD(lparam) = -1/+1 corresponds
	//	to collapse/expand button presses
	//	in winword (other values are
	//	equivalent to having pressed these
	//	buttons more than once)
	//	HIWORD(lparam) = EMO_EXPANDDOCUMENT
	//	-> expands whole document to
	//	indicated level
	EMO_EXPAND = 3

	// LOWORD(lparam) != 0 -> move current
	//	selection up/down by indicated amount
	EMO_MOVESELECTION = 4

	// Returns VM_NORMAL or VM_OUTLINE
	EMO_GETVIEWMODE = 5
)

// EMO_EXPAND options
const (
	EMO_EXPANDSELECTION = 0
	EMO_EXPANDDOCUMENT  = 1
)

const (
	// Agrees with RTF \viewkindN
	VM_NORMAL = 4

	VM_OUTLINE = 2

	// Screen page view (not print layout)
	VM_PAGE = 9
)

// New messages as of Win8
const (
	EM_INSERTTABLE = WM_USER + 232
)

// Data type defining table rows for EM_INSERTTABLE
// Note: The Richedit.h is completely #pragma pack(4)-ed
type TABLEROWPARMS struct { // EM_INSERTTABLE wparam is a (TABLEROWPARMS *)
	CbRow        uint32 // Count of bytes in this structure
	CbCell       uint32 // Count of bytes in TABLECELLPARMS
	CCell        uint32 // Count of cells
	CRow         uint32 // Count of rows
	DxCellMargin int32  // Cell left/right margin (\trgaph)
	DxIndent     int32  // Row left (right if fRTL indent (similar to \trleft)
	DyHeight     int32  // Row height (\trrh)

	// nAlignment:3   Row alignment (like PARAFORMAT::bAlignment, \trql, trqr, \trqc)
	// fRTL:1         Display cells in RTL order (\rtlrow)
	// fKeep:1        Keep row together (\trkeep}
	// fKeepFollow:1  Keep row on same page as following row (\trkeepfollow)
	// fWrap:1        Wrap text to right/left (depending on bAlignment) (see \tdfrmtxtLeftN, \tdfrmtxtRightN)
	// fIdentCells:1  lparam points at single struct valid for all cells
	Flags uint32

	CpStartRow  int32  // cp where to insert table (-1 for selection cp) (can be used for either TRD by EM_GETTABLEPARMS)
	BTableLevel uint32 // Table nesting level (EM_GETTABLEPARMS only)
	ICell       uint32 // Index of cell to insert/delete (EM_SETTABLEPARMS only)
}

// Data type defining table cells for EM_INSERTTABLE
// Note: The Richedit.h is completely #pragma pack(4)-ed
type TABLECELLPARMS struct { // EM_INSERTTABLE lparam is a (TABLECELLPARMS *)
	DxWidth int32 // Cell width (\cellx)

	// nVertAlign:2   Vertical alignment (0/1/2 = top/center/bottom \clvertalt (def), \clvertalc, \clvertalb)
	// fMergeTop:1    Top cell for vertical merge (\clvmgf)
	// fMergePrev:1   Merge with cell above (\clvmrg)
	// fVertical:1    Display text top to bottom, right to left (\cltxtbrlv)
	// fMergeStart:1  Start set of horizontally merged cells (\clmgf)
	// fMergeCont:1   Merge with previous cell (\clmrg)
	Flags uint32

	WShading uint32 // Shading in .01%		(\clshdng) e.g., 10000 flips fore/back

	DxBrdrLeft   int32 // Left border width	(\clbrdrl\brdrwN) (in twips)
	DyBrdrTop    int32 // Top border width 	(\clbrdrt\brdrwN)
	DxBrdrRight  int32 // Right border width	(\clbrdrr\brdrwN)
	DyBrdrBottom int32 // Bottom border width	(\clbrdrb\brdrwN)

	CrBrdrLeft   COLORREF // Left border color	(\clbrdrl\brdrcf)
	CrBrdrTop    COLORREF // Top border color 	(\clbrdrt\brdrcf)
	CrBrdrRight  COLORREF // Right border color	(\clbrdrr\brdrcf)
	CrBrdrBottom COLORREF // Bottom border color	(\clbrdrb\brdrcf)
	CrBackPat    COLORREF // Background color 	(\clcbpat)
	CrForePat    COLORREF // Foreground color 	(\clcfpat)
}

const (
	EM_GETAUTOCORRECTPROC  = WM_USER + 233
	EM_SETAUTOCORRECTPROC  = WM_USER + 234
	EM_CALLAUTOCORRECTPROC = WM_USER + 255
)

// AutoCorrect callback
type AutoCorrectProc func(langid LANGID, pszBefore *uint16, pszAfter *uint16, cchAfter int32, pcchReplaced *int32) int

const (
	ATP_NOCHANGE       = 0
	ATP_CHANGE         = 1
	ATP_NODELIMITER    = 2
	ATP_REPLACEALLTEXT = 4
)

const (
	EM_GETTABLEPARMS = WM_USER + 265

	EM_SETEDITSTYLEEX = WM_USER + 275
	EM_GETEDITSTYLEEX = WM_USER + 276
)

// wparam values for EM_SETEDITSTYLEEX/EM_GETEDITSTYLEEX
// All unused bits are reserved.
const (
	SES_EX_NOTABLE            = 0x00000004
	SES_EX_NOMATH             = 0x00000040
	SES_EX_HANDLEFRIENDLYURL  = 0x00000100
	SES_EX_NOTHEMING          = 0x00080000
	SES_EX_NOACETATESELECTION = 0x00100000
	SES_EX_USESINGLELINE      = 0x00200000
	SES_EX_MULTITOUCH         = 0x08000000 // Only works under Win8+
	SES_EX_HIDETEMPFORMAT     = 0x10000000
	SES_EX_USEMOUSEWPARAM     = 0x20000000 // Use wParam when handling WM_MOUSEMOVE message and do not call GetAsyncKeyState
)

const (
	EM_GETSTORYTYPE = WM_USER + 290
	EM_SETSTORYTYPE = WM_USER + 291

	EM_GETELLIPSISMODE = WM_USER + 305
	EM_SETELLIPSISMODE = WM_USER + 306
)

// uint32: *lparam for EM_GETELLIPSISMODE, lparam for EM_SETELLIPSISMODE
const (
	ELLIPSIS_MASK = 0x00000003 // all meaningful bits
	ELLIPSIS_NONE = 0x00000000 // ellipsis disabled
	ELLIPSIS_END  = 0x00000001 // ellipsis at the end (forced break)
	ELLIPSIS_WORD = 0x00000003 // ellipsis at the end (word break)
)

const (
	EM_SETTABLEPARMS = WM_USER + 307

	EM_GETTOUCHOPTIONS  = WM_USER + 310
	EM_SETTOUCHOPTIONS  = WM_USER + 311
	EM_INSERTIMAGE      = WM_USER + 314
	EM_SETUIANAME       = WM_USER + 320
	EM_GETELLIPSISSTATE = WM_USER + 322
)

// Values for EM_SETTOUCHOPTIONS/EM_GETTOUCHOPTIONS
const (
	RTO_SHOWHANDLES    = 1
	RTO_DISABLEHANDLES = 2
	RTO_READINGMODE    = 3
)

// lparam for EM_INSERTIMAGE
type RICHEDIT_IMAGE_PARAMETERS struct {
	XWidth            int32 // Units are HIMETRIC
	YHeight           int32 // Units are HIMETRIC
	Ascent            int32 // Units are HIMETRIC
	Type              int32 // Valid values are TA_TOP, TA_BOTTOM and TA_BASELINE
	PwszAlternateText *uint16
	PIStream          uintptr
}

// New notifications
const (
	EN_MSGFILTER         = 0x0700
	EN_REQUESTRESIZE     = 0x0701
	EN_SELCHANGE         = 0x0702
	EN_DROPFILES         = 0x0703
	EN_PROTECTED         = 0x0704
	EN_CORRECTTEXT       = 0x0705 // PenWin specific
	EN_STOPNOUNDO        = 0x0706
	EN_IMECHANGE         = 0x0707 // East Asia specific
	EN_SAVECLIPBOARD     = 0x0708
	EN_OLEOPFAILED       = 0x0709
	EN_OBJECTPOSITIONS   = 0x070a
	EN_LINK              = 0x070b
	EN_DRAGDROPDONE      = 0x070c
	EN_PARAGRAPHEXPANDED = 0x070d
	EN_PAGECHANGE        = 0x070e
	EN_LOWFIRTF          = 0x070f
	EN_ALIGNLTR          = 0x0710 // BiDi specific notification
	EN_ALIGNRTL          = 0x0711 // BiDi specific notification
	EN_CLIPFORMAT        = 0x0712
	EN_STARTCOMPOSITION  = 0x0713
	EN_ENDCOMPOSITION    = 0x0714
)

// Notification structure for EN_ENDCOMPOSITION
type ENDCOMPOSITIONNOTIFY struct {
	Nmhdr  NMHDR
	DwCode uint32
}

// Constants for ENDCOMPOSITIONNOTIFY dwCode
const (
	ECN_ENDCOMPOSITION = 0x0001
	ECN_NEWTEXT        = 0x0002
)

// Event notification masks
const (
	ENM_NONE              = 0x00000000
	ENM_CHANGE            = 0x00000001
	ENM_UPDATE            = 0x00000002
	ENM_SCROLL            = 0x00000004
	ENM_SCROLLEVENTS      = 0x00000008
	ENM_DRAGDROPDONE      = 0x00000010
	ENM_PARAGRAPHEXPANDED = 0x00000020
	ENM_PAGECHANGE        = 0x00000040
	ENM_CLIPFORMAT        = 0x00000080
	ENM_KEYEVENTS         = 0x00010000
	ENM_MOUSEEVENTS       = 0x00020000
	ENM_REQUESTRESIZE     = 0x00040000
	ENM_SELCHANGE         = 0x00080000
	ENM_DROPFILES         = 0x00100000
	ENM_PROTECTED         = 0x00200000
	ENM_CORRECTTEXT       = 0x00400000 // PenWin specific
	ENM_IMECHANGE         = 0x00800000 // Used by RE1.0 compatibility
	ENM_LANGCHANGE        = 0x01000000
	ENM_OBJECTPOSITIONS   = 0x02000000
	ENM_LINK              = 0x04000000
	ENM_LOWFIRTF          = 0x08000000
	ENM_STARTCOMPOSITION  = 0x10000000
	ENM_ENDCOMPOSITION    = 0x20000000
	ENM_GROUPTYPINGCHANGE = 0x40000000
	ENM_HIDELINKTOOLTIP   = 0x80000000
)

// New edit control styles
const (
	ES_SAVESEL         = 0x00008000
	ES_SUNKEN          = 0x00004000
	ES_DISABLENOSCROLL = 0x00002000
	ES_SELECTIONBAR    = 0x01000000 // Same as WS_MAXIMIZE, but that doesn't make sense so we re-use the value
	ES_NOOLEDRAGDROP   = 0x00000008 // Same as ES_UPPERCASE, but re-used to completely disable OLE drag'n'drop
)

// Obsolete Edit Style
const (
	ES_EX_NOCALLOLEINIT = 0x00000000 // Not supported in RE 2.0/3.0
)

// These flags are used in FE Windows
const (
	ES_VERTICAL = 0x00400000 // Not supported in RE 2.0/3.0
	ES_NOIME    = 0x00080000
	ES_SELFIME  = 0x00040000
)

// Edit control options
const (
	ECO_AUTOWORDSELECTION = 0x00000001
	ECO_AUTOVSCROLL       = 0x00000040
	ECO_AUTOHSCROLL       = 0x00000080
	ECO_NOHIDESEL         = 0x00000100
	ECO_READONLY          = 0x00000800
	ECO_WANTRETURN        = 0x00001000
	ECO_SAVESEL           = 0x00008000
	ECO_SELECTIONBAR      = 0x01000000
	ECO_VERTICAL          = 0x00400000 // FE specific
)

// ECO operations
const (
	ECOOP_SET = 0x0001
	ECOOP_OR  = 0x0002
	ECOOP_AND = 0x0003
	ECOOP_XOR = 0x0004
)

// New word break function actions
const (
	WB_CLASSIFY      = 3
	WB_MOVEWORDLEFT  = 4
	WB_MOVEWORDRIGHT = 5
	WB_LEFTBREAK     = 6
	WB_RIGHTBREAK    = 7
)

// East Asia specific flags
const (
	WB_MOVEWORDPREV = 4
	WB_MOVEWORDNEXT = 5
	WB_PREVBREAK    = 6
	WB_NEXTBREAK    = 7

	PC_FOLLOWING  = 1
	PC_LEADING    = 2
	PC_OVERFLOW   = 3
	PC_DELIMITER  = 4
	WBF_WORDWRAP  = 0x010
	WBF_WORDBREAK = 0x020
	WBF_OVERFLOW  = 0x040
	WBF_LEVEL1    = 0x080
	WBF_LEVEL2    = 0x100
	WBF_CUSTOM    = 0x200
)

// East Asia specific flags
const (
	IMF_FORCENONE         = 0x0001
	IMF_FORCEENABLE       = 0x0002
	IMF_FORCEDISABLE      = 0x0004
	IMF_CLOSESTATUSWINDOW = 0x0008
	IMF_VERTICAL          = 0x0020
	IMF_FORCEACTIVE       = 0x0040
	IMF_FORCEINACTIVE     = 0x0080
	IMF_FORCEREMEMBER     = 0x0100
	IMF_MULTIPLEEDIT      = 0x0400
)

// Word break flags (used with WB_CLASSIFY)
const (
	WBF_CLASS      byte = 0x0F
	WBF_ISWHITE    byte = 0x10
	WBF_BREAKLINE  byte = 0x20
	WBF_BREAKAFTER byte = 0x40
)

type CHARFORMAT struct {
	CbSize          uint32
	DwMask          uint32
	DwEffects       uint32
	YHeight         int32
	YOffset         int32
	CrTextColor     COLORREF
	BCharSet        byte
	BPitchAndFamily byte
	SzFaceName      [LF_FACESIZE]uint16
}

type CHARFORMAT2 struct {
	CHARFORMAT
	WWeight         uint16   // Font weight (LOGFONT value)
	SSpacing        int16    // Amount to space between letters
	CrBackColor     COLORREF // Background color
	Lcid            LCID     // Locale ID
	DwCookie        uint32   // Client cookie opaque to RichEdit
	SStyle          int16    // Style handle
	WKerning        uint16   // Twip size above which to kern char pair
	BUnderlineType  byte     // Underline type
	BAnimation      byte     // Animated text like marching ants
	BRevAuthor      byte     // Revision author index
	BUnderlineColor byte     // Underline color
}

// CHARFORMAT masks
const (
	CFM_BOLD      = 0x00000001
	CFM_ITALIC    = 0x00000002
	CFM_UNDERLINE = 0x00000004
	CFM_STRIKEOUT = 0x00000008
	CFM_PROTECTED = 0x00000010
	CFM_LINK      = 0x00000020 // Exchange hyperlink extension
	CFM_SIZE      = 0x80000000
	CFM_COLOR     = 0x40000000
	CFM_FACE      = 0x20000000
	CFM_OFFSET    = 0x10000000
	CFM_CHARSET   = 0x08000000
)

// CHARFORMAT effects
const (
	CFE_BOLD      = 0x00000001
	CFE_ITALIC    = 0x00000002
	CFE_UNDERLINE = 0x00000004
	CFE_STRIKEOUT = 0x00000008
	CFE_PROTECTED = 0x00000010
	CFE_LINK      = 0x00000020
	CFE_AUTOCOLOR = 0x40000000 // NOTE: this corresponds to CFM_COLOR, which controls it

	// Masks and effects defined for CHARFORMAT2 -- an (*) indicates that the data is stored by RichEdit 2.0/3.0, but not displayed
	CFM_SMALLCAPS = 0x00000040 // (*)
	CFM_ALLCAPS   = 0x00000080 // Displayed by 3.0
	CFM_HIDDEN    = 0x00000100 // Hidden by 3.0
	CFM_OUTLINE   = 0x00000200 // (*)
	CFM_SHADOW    = 0x00000400 // (*)
	CFM_EMBOSS    = 0x00000800 // (*)
	CFM_IMPRINT   = 0x00001000 // (*)
	CFM_DISABLED  = 0x00002000
	CFM_REVISED   = 0x00004000

	CFM_REVAUTHOR     = 0x00008000
	CFE_SUBSCRIPT     = 0x00010000 // Superscript and subscript are
	CFE_SUPERSCRIPT   = 0x00020000 //	mutually exclusive
	CFM_ANIMATION     = 0x00040000 // (*)
	CFM_STYLE         = 0x00080000 // (*)
	CFM_KERNING       = 0x00100000
	CFM_SPACING       = 0x00200000 // Displayed by 3.0
	CFM_WEIGHT        = 0x00400000
	CFM_UNDERLINETYPE = 0x00800000 // Many displayed by 3.0
	CFM_COOKIE        = 0x01000000 // RE 6.0
	CFM_LCID          = 0x02000000
	CFM_BACKCOLOR     = 0x04000000 // Higher mask bits defined above

	CFM_SUBSCRIPT   = (CFE_SUBSCRIPT | CFE_SUPERSCRIPT)
	CFM_SUPERSCRIPT = CFM_SUBSCRIPT

	// CHARFORMAT "ALL" masks
	CFM_EFFECTS  = CFM_BOLD | CFM_ITALIC | CFM_UNDERLINE | CFM_COLOR | CFM_STRIKEOUT | CFE_PROTECTED | CFM_LINK
	CFM_ALL      = CFM_EFFECTS | CFM_SIZE | CFM_FACE | CFM_OFFSET | CFM_CHARSET
	CFM_EFFECTS2 = CFM_EFFECTS | CFM_DISABLED | CFM_SMALLCAPS | CFM_ALLCAPS | CFM_HIDDEN | CFM_OUTLINE | CFM_SHADOW | CFM_EMBOSS | CFM_IMPRINT | CFM_REVISED | CFM_SUBSCRIPT | CFM_SUPERSCRIPT | CFM_BACKCOLOR
	CFM_ALL2     = CFM_ALL | CFM_EFFECTS2 | CFM_BACKCOLOR | CFM_LCID | CFM_UNDERLINETYPE | CFM_WEIGHT | CFM_REVAUTHOR | CFM_SPACING | CFM_KERNING | CFM_STYLE | CFM_ANIMATION | CFM_COOKIE

	CFE_SMALLCAPS = CFM_SMALLCAPS
	CFE_ALLCAPS   = CFM_ALLCAPS
	CFE_HIDDEN    = CFM_HIDDEN
	CFE_OUTLINE   = CFM_OUTLINE
	CFE_SHADOW    = CFM_SHADOW
	CFE_EMBOSS    = CFM_EMBOSS
	CFE_IMPRINT   = CFM_IMPRINT
	CFE_DISABLED  = CFM_DISABLED
	CFE_REVISED   = CFM_REVISED

	// CFE_AUTOCOLOR and CFE_AUTOBACKCOLOR correspond to CFM_COLOR and
	// CFM_BACKCOLOR, respectively, which control them
	CFE_AUTOBACKCOLOR = CFM_BACKCOLOR

	CFM_FONTBOUND     = 0x00100000
	CFM_LINKPROTECTED = 0x00800000 // Word hyperlink field
	CFM_EXTENDED      = 0x02000000
	CFM_MATHNOBUILDUP = 0x08000000
	CFM_MATH          = 0x10000000
	CFM_MATHORDINARY  = 0x20000000

	CFM_ALLEFFECTS = (CFM_EFFECTS2 | CFM_FONTBOUND | CFM_EXTENDED | CFM_MATHNOBUILDUP | CFM_MATH | CFM_MATHORDINARY)

	CFE_FONTBOUND     = 0x00100000 // Font chosen by binder, not user
	CFE_LINKPROTECTED = 0x00800000
	CFE_EXTENDED      = 0x02000000
	CFE_MATHNOBUILDUP = 0x08000000
	CFE_MATH          = 0x10000000
	CFE_MATHORDINARY  = 0x20000000

	// Underline types. RE 1.0 displays only CFU_UNDERLINE
	CFU_CF1UNDERLINE             = 0xFF // Map charformat's bit underline to CF2
	CFU_INVERT                   = 0xFE // For IME composition fake a selection
	CFU_UNDERLINETHICKLONGDASH   = 18   // (*) display as dash
	CFU_UNDERLINETHICKDOTTED     = 17   // (*) display as dot
	CFU_UNDERLINETHICKDASHDOTDOT = 16   // (*) display as dash dot dot
	CFU_UNDERLINETHICKDASHDOT    = 15   // (*) display as dash dot
	CFU_UNDERLINETHICKDASH       = 14   // (*) display as dash
	CFU_UNDERLINELONGDASH        = 13   // (*) display as dash
	CFU_UNDERLINEHEAVYWAVE       = 12   // (*) display as wave
	CFU_UNDERLINEDOUBLEWAVE      = 11   // (*) display as wave
	CFU_UNDERLINEHAIRLINE        = 10   // (*) display as single
	CFU_UNDERLINETHICK           = 9
	CFU_UNDERLINEWAVE            = 8
	CFU_UNDERLINEDASHDOTDOT      = 7
	CFU_UNDERLINEDASHDOT         = 6
	CFU_UNDERLINEDASH            = 5
	CFU_UNDERLINEDOTTED          = 4
	CFU_UNDERLINEDOUBLE          = 3 // (*) display as single
	CFU_UNDERLINEWORD            = 2 // (*) display as single
	CFU_UNDERLINE                = 1
	CFU_UNDERLINENONE            = 0
)

const YHeightCharPtsMost = 1638

const (
	// EM_SETCHARFORMAT wParam masks
	SCF_SELECTION       = 0x0001
	SCF_WORD            = 0x0002
	SCF_DEFAULT         = 0x0000 // Set default charformat or paraformat
	SCF_ALL             = 0x0004 // Not valid with SCF_SELECTION or SCF_WORD
	SCF_USEUIRULES      = 0x0008 // Modifier for SCF_SELECTION; says that came from a toolbar, etc., and  UI formatting rules should be instead of literal formatting
	SCF_ASSOCIATEFONT   = 0x0010 // Associate fontname with bCharSet (one possible for each of Western, ME, FE, Thai)
	SCF_NOKBUPDATE      = 0x0020 // Do not update KB layout for this change even if autokeyboard is on
	SCF_ASSOCIATEFONT2  = 0x0040 // Associate plane-2 (surrogate) font
	SCF_SMARTFONT       = 0x0080 // Apply font only if it can handle script (5.0)
	SCF_CHARREPFROMLCID = 0x0100 // Get character repertoire from lcid (5.0)

	SPF_DONTSETDEFAULT = 0x0002 // Suppress setting default on empty control
	SPF_SETDEFAULT     = 0x0004 // Set the default paraformat
)

type CHARRANGE struct {
	CpMin int32
	CpMax int32
}

type TEXTRANGE struct {
	Chrg      CHARRANGE
	LpstrText *uint16 // Allocated by caller, zero terminated by RichEdit
}

type EDITSTREAM struct {
	DwCookie    uintptr // User value passed to callback as first parameter
	DwError     uint32  // Last error
	PfnCallback uintptr
}

const (
	// Stream formats. Flags are all in low word, since high word gives possible codepage choice.
	SF_TEXT      = 0x0001
	SF_RTF       = 0x0002
	SF_RTFNOOBJS = 0x0003 // Write only
	SF_TEXTIZED  = 0x0004 // Write only

	SF_UNICODE        = 0x0010 // Unicode file (UCS2 little endian)
	SF_USECODEPAGE    = 0x0020 // CodePage given by high word
	SF_NCRFORNONASCII = 0x40   // Output \uN for nonASCII
	SFF_WRITEXTRAPAR  = 0x80   // Output \par at end

	// Flag telling stream operations to operate on selection only
	// EM_STREAMIN	replaces current selection
	// EM_STREAMOUT streams out current selection
	SFF_SELECTION = 0x8000

	// Flag telling stream operations to ignore some FE control words having to do with FE word breaking and horiz vs vertical text.
	// Not used in RichEdit 2.0 and later
	SFF_PLAINRTF = 0x4000

	// Flag telling file stream output (SFF_SELECTION flag not set) to persist // \viewscaleN control word.
	SFF_PERSISTVIEWSCALE = 0x2000

	// Flag telling file stream input with SFF_SELECTION flag not set not to // close the document
	SFF_KEEPDOCINFO = 0x1000

	// Flag telling stream operations to output in Pocket Word format
	SFF_PWD = 0x0800

	// 3-bit field specifying the value of N - 1 to use for \rtfN or \pwdN
	SF_RTFVAL = 0x0700
)

type FINDTEXT struct {
	Chrg      CHARRANGE
	LpstrText *uint16
}

type FINDTEXTEX struct {
	chrg      CHARRANGE
	lpstrText *uint16
	chrgText  CHARRANGE
}

type FORMATRANGE struct {
	hdc       HDC
	hdcTarget HDC
	rc        RECT
	rcPage    RECT
	chrg      CHARRANGE
}

// All paragraph measurements are in twips
const (
	MAX_TAB_STOPS   = 32
	LDefaultTab     = 720
	MAX_TABLE_CELLS = 63
)

type PARAFORMAT struct {
	CbSize        uint32
	DwMask        uint32
	WNumbering    uint16
	WEffects      uint16
	DxStartIndent int32
	DxRightIndent int32
	DxOffset      int32
	WAlignment    uint16
	CTabCount     int16
	RgxTabs       [MAX_TAB_STOPS]int32
}

type PARAFORMAT2 struct {
	PARAFORMAT
	DySpaceBefore    int32  // Vertical spacing before para
	DySpaceAfter     int32  // Vertical spacing after para
	DyLineSpacing    int32  // Line spacing depending on Rule
	SStyle           int16  // Style handle
	BLineSpacingRule byte   // Rule for line spacing (see tom.doc)
	BOutlineLevel    byte   // Outline level
	WShadingWeight   uint16 // Shading in hundredths of a per cent
	WShadingStyle    uint16 // Nibble 0: style, 1: cfpat, 2: cbpat
	WNumberingStart  uint16 // Starting value for numbering
	WNumberingStyle  uint16 // Alignment, roman/arabic, (), ), ., etc.
	WNumberingTab    uint16 // Space bet FirstIndent & 1st-line text
	WBorderSpace     uint16 // Border-text spaces (nbl/bdr in pts)
	WBorderWidth     uint16 // Pen widths (nbl/bdr in half pts)
	WBorders         uint16 // Border styles (nibble/border)
}

const (
	// PARAFORMAT mask values
	PFM_STARTINDENT  = 0x00000001
	PFM_RIGHTINDENT  = 0x00000002
	PFM_OFFSET       = 0x00000004
	PFM_ALIGNMENT    = 0x00000008
	PFM_TABSTOPS     = 0x00000010
	PFM_NUMBERING    = 0x00000020
	PFM_OFFSETINDENT = 0x80000000

	// PARAFORMAT 2.0 masks and effects
	PFM_SPACEBEFORE    = 0x00000040
	PFM_SPACEAFTER     = 0x00000080
	PFM_LINESPACING    = 0x00000100
	PFM_STYLE          = 0x00000400
	PFM_BORDER         = 0x00000800 // (*)
	PFM_SHADING        = 0x00001000 // (*)
	PFM_NUMBERINGSTYLE = 0x00002000 // RE 3.0
	PFM_NUMBERINGTAB   = 0x00004000 // RE 3.0
	PFM_NUMBERINGSTART = 0x00008000 // RE 3.0

	PFM_RTLPARA         = 0x00010000
	PFM_KEEP            = 0x00020000 // (*)
	PFM_KEEPNEXT        = 0x00040000 // (*)
	PFM_PAGEBREAKBEFORE = 0x00080000 // (*)
	PFM_NOLINENUMBER    = 0x00100000 // (*)
	PFM_NOWIDOWCONTROL  = 0x00200000 // (*)
	PFM_DONOTHYPHEN     = 0x00400000 // (*)
	PFM_SIDEBYSIDE      = 0x00800000 // (*)

	// The following two paragraph-format properties are read only
	PFM_COLLAPSED         = 0x01000000 // RE 3.0
	PFM_OUTLINELEVEL      = 0x02000000 // RE 3.0
	PFM_BOX               = 0x04000000 // RE 3.0
	PFM_RESERVED2         = 0x08000000 // RE 4.0
	PFM_TABLEROWDELIMITER = 0x10000000 // RE 4.0
	PFM_TEXTWRAPPINGBREAK = 0x20000000 // RE 3.0
	PFM_TABLE             = 0x40000000 // RE 3.0

	// PARAFORMAT "ALL" masks
	PFM_ALL = PFM_STARTINDENT | PFM_RIGHTINDENT | PFM_OFFSET | PFM_ALIGNMENT | PFM_TABSTOPS | PFM_NUMBERING | PFM_OFFSETINDENT | PFM_RTLPARA

	// Note: PARAFORMAT has no effects (BiDi RichEdit 1.0 does have PFE_RTLPARA)
	PFM_EFFECTS = PFM_RTLPARA | PFM_KEEP | PFM_KEEPNEXT | PFM_TABLE | PFM_PAGEBREAKBEFORE | PFM_NOLINENUMBER | PFM_NOWIDOWCONTROL | PFM_DONOTHYPHEN | PFM_SIDEBYSIDE | PFM_TABLE | PFM_TABLEROWDELIMITER

	PFM_ALL2 = PFM_ALL | PFM_EFFECTS | PFM_SPACEBEFORE | PFM_SPACEAFTER | PFM_LINESPACING | PFM_STYLE | PFM_SHADING | PFM_BORDER | PFM_NUMBERINGTAB | PFM_NUMBERINGSTART | PFM_NUMBERINGSTYLE

	PFE_RTLPARA           = PFM_RTLPARA >> 16
	PFE_KEEP              = PFM_KEEP >> 16              // (*)
	PFE_KEEPNEXT          = PFM_KEEPNEXT >> 16          // (*)
	PFE_PAGEBREAKBEFORE   = PFM_PAGEBREAKBEFORE >> 16   // (*)
	PFE_NOLINENUMBER      = PFM_NOLINENUMBER >> 16      // (*)
	PFE_NOWIDOWCONTROL    = PFM_NOWIDOWCONTROL >> 16    // (*)
	PFE_DONOTHYPHEN       = PFM_DONOTHYPHEN >> 16       // (*)
	PFE_SIDEBYSIDE        = PFM_SIDEBYSIDE >> 16        // (*)
	PFE_TEXTWRAPPINGBREAK = PFM_TEXTWRAPPINGBREAK >> 16 // (*)

	// The following four effects are read only
	PFE_COLLAPSED         = PFM_COLLAPSED >> 16         // (+)
	PFE_BOX               = PFM_BOX >> 16               // (+)
	PFE_TABLE             = PFM_TABLE >> 16             // Inside table row. RE 3.0
	PFE_TABLEROWDELIMITER = PFM_TABLEROWDELIMITER >> 16 // Table row start. RE 4.0

	// PARAFORMAT numbering options
	PFN_BULLET = 1 // tomListBullet

	// PARAFORMAT2 wNumbering options
	PFN_ARABIC   = 2 // tomListNumberAsArabic:	0, 1, 2,	...
	PFN_LCLETTER = 3 // tomListNumberAsLCLetter: a, b, c,	...
	PFN_UCLETTER = 4 // tomListNumberAsUCLetter: A, B, C,	...
	PFN_LCROMAN  = 5 // tomListNumberAsLCRoman:	i, ii, iii, ...
	PFN_UCROMAN  = 6 // tomListNumberAsUCRoman:	I, II, III, ...

	// PARAFORMAT2 wNumberingStyle options
	PFNS_PAREN    = 0x000 // default, e.g.,				  1)
	PFNS_PARENS   = 0x100 // tomListParentheses/256, e.g., (1)
	PFNS_PERIOD   = 0x200 // tomListPeriod/256, e.g., 	  1.
	PFNS_PLAIN    = 0x300 // tomListPlain/256, e.g.,		  1
	PFNS_NONUMBER = 0x400 // Used for continuation w/o number

	PFNS_NEWNUMBER = 0x8000 // Start new number with wNumberingStart
	// (can be combined with other PFNS_xxx)
	// PARAFORMAT alignment options
	PFA_LEFT   = 1
	PFA_RIGHT  = 2
	PFA_CENTER = 3

	// PARAFORMAT2 alignment options
	PFA_JUSTIFY        = 4 // New paragraph-alignment option 2.0 (*)
	PFA_FULL_INTERWORD = 4 // These are supported in 3.0 with advanced
)

type MSGFILTER struct {
	Nmhdr  NMHDR
	Msg    uint32
	WParam uintptr
	LParam uintptr
}

type REQRESIZE struct {
	Nmhdr NMHDR
	Rc    RECT
}

type SELCHANGE struct {
	Nmhdr  NMHDR
	Chrg   CHARRANGE
	Seltyp uint16
}

type GROUPTYPINGCHANGE struct {
	Nmhdr        NMHDR
	FGroupTyping BOOL
}

type CLIPBOARDFORMAT struct {
	Nmhdr NMHDR
	Cf    CLIPFORMAT
}

const (
	SEL_EMPTY       = 0x0000
	SEL_TEXT        = 0x0001
	SEL_OBJECT      = 0x0002
	SEL_MULTICHAR   = 0x0004
	SEL_MULTIOBJECT = 0x0008
)

const (
	// Used with IRichEditOleCallback::GetContextMenu, this flag will be passed as a "selection type".  It indicates that a context menu for a right-mouse drag drop should be generated.  The IOleObject parameter will really be the IDataObject for the drop
	GCM_RIGHTMOUSEDROP = 0x8000
)

type GETCONTEXTMENUEX struct {
	Chrg       CHARRANGE
	DwFlags    uint32
	Pt         POINT
	PvReserved uintptr
}

const (
	// bits for GETCONTEXTMENUEX::dwFlags
	GCMF_GRIPPER   = 0x00000001
	GCMF_SPELLING  = 0x00000002 // pSpellingSuggestions is valid and points to the list of spelling suggestions
	GCMF_TOUCHMENU = 0x00004000
	GCMF_MOUSEMENU = 0x00002000
)

type ENDROPFILES struct {
	Nmhdr      NMHDR
	HDrop      HANDLE
	Cp         int32
	FProtected BOOL
}

type ENPROTECTED struct {
	Nmhdr  NMHDR
	Msg    uint32
	WParam uintptr
	LParam uintptr
	Chrg   CHARRANGE
}

type ENSAVECLIPBOARD struct {
	Nmhdr        NMHDR
	CObjectCount int32
	Cch          int32
}

type ENOLEOPFAILED struct {
	Nmhdr NMHDR
	Iob   int32
	LOper int32
	Hr    HRESULT
}

const OLEOP_DOVERB = 1

type OBJECTPOSITIONS struct {
	Nmhdr        NMHDR
	CObjectCount int32
	PcpPositions *int32
}

type ENLINK struct {
	Nmhdr  NMHDR
	Msg    uint32
	WParam uintptr
	LParam uintptr
	Chrg   CHARRANGE
}

type ENLOWFIRTF struct {
	Nmhdr     NMHDR
	SzControl *byte
}

// PenWin specific
type ENCORRECTTEXT struct {
	Nmhdr  NMHDR
	Chrg   CHARRANGE
	Seltyp uint16
}

// East Asia specific
type PUNCTUATION struct {
	ISize         uint32
	SzPunctuation *byte
}

// East Asia specific
type COMPCOLOR struct {
	CrText       COLORREF
	CrBackground COLORREF
	DwEffects    uint32
}

const (
	// Clipboard formats - use as parameter to RegisterClipboardFormat()
	CF_RTF       = "Rich Text Format"
	CF_RTFNOOBJS = "Rich Text Format Without Objects"
	CF_RETEXTOBJ = "RichEdit Text and Objects"
)

// Paste Special
type REPASTESPECIAL struct {
	DwAspect uint32
	DwParam  uintptr
}

//	UndoName info
type UNDONAMEID int32

const (
	UID_UNKNOWN   UNDONAMEID = 0
	UID_TYPING               = 1
	UID_DELETE               = 2
	UID_DRAGDROP             = 3
	UID_CUT                  = 4
	UID_PASTE                = 5
	UID_AUTOTABLE            = 6
)

const (
	// Flags for the SETEXTEX data structure
	ST_DEFAULT   = 0
	ST_KEEPUNDO  = 1
	ST_SELECTION = 2
	ST_NEuint16S = 4
	ST_UNICODE   = 8
)

// EM_SETTEXTEX info; this struct is passed in the wparam of the message
type SETTEXTEX struct {
	Flags    uint32 // Flags (see the ST_XXX defines)
	Codepage uint32 // Code page for translation (CP_ACP for sys default, 1200 for Unicode, -1 for control default)
}

const (
	// Flags for the GETEXTEX data structure
	GT_DEFAULT      = 0
	GT_USECRLF      = 1
	GT_SELECTION    = 2
	GT_RAWTEXT      = 4
	GT_NOHIDDENTEXT = 8
)

// EM_GETTEXTEX info; this struct is passed in the wparam of the message
type GETTEXTEX struct {
	Cb            uint32 // Count of bytes in the string
	Flags         uint32 // Flags (see the GT_XXX defines
	Codepage      uint32 // Code page for translation (CP_ACP for sys default, 1200 for Unicode, -1 for control default)
	LpDefaultChar *byte  // Replacement for unmappable chars
	LpUsedDefChar *BOOL  // Pointer to flag set when def char used
}

const (
	// Flags for the GETTEXTLENGTHEX data structure
	GTL_DEFAULT  = 0  // Do default (return # of chars)
	GTL_USECRLF  = 1  // Compute answer using CRLFs for paragraphs
	GTL_PRECISE  = 2  // Compute a precise answer
	GTL_CLOSE    = 4  // Fast computation of a "close" answer
	GTL_NUMCHARS = 8  // Return number of characters
	GTL_NUMBYTES = 16 // Return number of _bytes_
)

// EM_GETTEXTLENGTHEX info; this struct is passed in the wparam of the msg
type GETTEXTLENGTHEX struct {
	Flags    uint32 // Flags (see GTL_XXX defines)
	Codepage uint32 // Code page for translation (CP_ACP for default, 1200 for Unicode)
}

// BiDi specific features
type BIDIOPTIONS struct {
	CbSize   uint32
	WMask    uint16
	WEffects uint16
}

const (
	// BIDIOPTIONS masks
	BOM_NEUTRALOVERRIDE  = 0x0004 // Override neutral layout (obsolete)
	BOM_CONTEXTREADING   = 0x0008 // Context reading order
	BOM_CONTEXTALIGNMENT = 0x0010 // Context alignment
	BOM_LEGACYBIDICLASS  = 0x0040 // Legacy Bidi classification (obsolete)
	BOM_UNICODEBIDI      = 0x0080 // Use Unicode BiDi algorithm

	// BIDIOPTIONS effects
	BOE_NEUTRALOVERRIDE  = 0x0004 // Override neutral layout (obsolete)
	BOE_CONTEXTREADING   = 0x0008 // Context reading order
	BOE_CONTEXTALIGNMENT = 0x0010 // Context alignment
	BOE_FORCERECALC      = 0x0020 // Force recalc and redraw
	BOE_LEGACYBIDICLASS  = 0x0040 // Legacy Bidi classification (obsolete)
	BOE_UNICODEBIDI      = 0x0080 // Use Unicode BiDi algorithm

	// Additional EM_FINDTEXT[EX] flags
	FR_MATCHDIAC      = 0x20000000
	FR_MATCHKASHIDA   = 0x40000000
	FR_MATCHALEFHAMZA = 0x80000000

	// UNICODE embedding character
	WCH_EMBEDDING uint16 = 0xFFFC
)

// khyph - Kind of hyphenation
type KHYPH int32

const (
	KhyphNil          KHYPH = iota // No Hyphenation
	KhyphNormal                    // Normal Hyphenation
	KhyphAddBefore                 // Add letter before hyphen
	KhyphChangeBefore              // Change letter before hyphen
	KhyphDeleteBefore              // Delete letter before hyphen
	KhyphChangeAfter               // Change letter after hyphen
	KhyphDelAndChange              // Delete letter before hyphen and change letter preceding hyphen
)

type HYPHRESULT struct {
	Khyph   KHYPH  // Kind of hyphenation
	IchHyph int32  // Character which was hyphenated
	ChHyph  uint16 // Depending on hyphenation type, character added, changed, etc.
}

type HYPHENATEINFO struct {
	CbSize          int16 // Size of HYPHENATEINFO structure
	DxHyphenateZone int16 // If a space character is closer to the margin than this value, don't hyphenate (in TWIPs)
	PfnHyphenate    uintptr
}

const (
	// Additional class for Richedit 6.0
	RICHEDIT60_CLASS = "RICHEDIT60W"
)
