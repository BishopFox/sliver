#ifndef SLIVER_OPFOR_E2E_WINDOWS_H
#define SLIVER_OPFOR_E2E_WINDOWS_H

/*
 * Sliver's bundled Zig intentionally does not include MinGW's Windows SDK
 * headers. This compatibility header contains only the ABI declarations used
 * by bof_collection v0.0.1 cat/entry.c and its exact tagged beacon.h. It is
 * copied into the isolated E2E build directory as windows.h; the pinned
 * upstream sources remain byte-for-byte unchanged.
 */

#include <stddef.h>
#include <stdint.h>

#define IN
#define OUT
#define OPTIONAL
#define WINAPI __attribute__((stdcall))
#define DECLSPEC_IMPORT __declspec(dllimport)

#ifndef NULL
#define NULL ((void *)0)
#endif

typedef void VOID;
typedef int BOOL;
typedef uint8_t BYTE;
typedef uint8_t BOOLEAN;
typedef uint16_t WORD;
typedef uint32_t DWORD;
typedef uint32_t ULONG;
typedef uint32_t UINT;
typedef uint64_t DWORD64;
typedef int32_t LONG;
typedef uintptr_t SIZE_T;

typedef char CHAR;
typedef CHAR *PCHAR;
typedef const CHAR *LPCSTR;
typedef void *PVOID;
typedef void *LPVOID;
typedef const void *LPCVOID;
typedef void *HANDLE;
typedef HANDLE *LPHANDLE;
typedef HANDLE HMODULE;
typedef DWORD *PDWORD;

typedef struct _CONTEXT CONTEXT;
typedef CONTEXT *PCONTEXT;
typedef struct _MEMORY_BASIC_INFORMATION MEMORY_BASIC_INFORMATION;
typedef MEMORY_BASIC_INFORMATION *PMEMORY_BASIC_INFORMATION;
typedef struct _PROCESS_INFORMATION PROCESS_INFORMATION;
typedef struct _STARTUPINFOA STARTUPINFO;
typedef struct _SECURITY_ATTRIBUTES SECURITY_ATTRIBUTES;
typedef SECURITY_ATTRIBUTES *LPSECURITY_ATTRIBUTES;

typedef union _LARGE_INTEGER {
	struct {
		DWORD LowPart;
		LONG HighPart;
	};
	int64_t QuadPart;
} LARGE_INTEGER;

#define INVALID_HANDLE_VALUE ((HANDLE)(intptr_t)-1)
#define GENERIC_READ ((DWORD)0x80000000)
#define FILE_SHARE_READ ((DWORD)0x00000001)
#define OPEN_EXISTING ((DWORD)3)
#define FILE_ATTRIBUTE_NORMAL ((DWORD)0x00000080)
#define HEAP_ZERO_MEMORY ((DWORD)0x00000008)
#define CP_UTF8 ((UINT)65001)

DECLSPEC_IMPORT HANDLE WINAPI CreateFileA(
	LPCSTR filename,
	DWORD desired_access,
	DWORD share_mode,
	LPSECURITY_ATTRIBUTES security_attributes,
	DWORD creation_disposition,
	DWORD flags_and_attributes,
	HANDLE template_file
);
DECLSPEC_IMPORT BOOL WINAPI GetFileSizeEx(HANDLE file, LARGE_INTEGER *size);
DECLSPEC_IMPORT BOOL WINAPI CloseHandle(HANDLE object);
DECLSPEC_IMPORT DWORD WINAPI GetLastError(void);
DECLSPEC_IMPORT HANDLE WINAPI GetProcessHeap(void);
DECLSPEC_IMPORT LPVOID WINAPI HeapAlloc(HANDLE heap, DWORD flags, SIZE_T bytes);
DECLSPEC_IMPORT BOOL WINAPI HeapFree(HANDLE heap, DWORD flags, LPVOID memory);
DECLSPEC_IMPORT BOOL WINAPI ReadFile(
	HANDLE file,
	LPVOID buffer,
	DWORD bytes_to_read,
	PDWORD bytes_read,
	LPVOID overlapped
);
DECLSPEC_IMPORT int WINAPI WideCharToMultiByte(
	UINT code_page,
	DWORD flags,
	const wchar_t *wide,
	int wide_length,
	CHAR *multibyte,
	int multibyte_length,
	LPCSTR default_character,
	BOOL *used_default_character
);

#endif
