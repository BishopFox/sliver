#pragma once

#include <windows.h>

#ifdef BOF

/* Beacon API */
void   BeaconPrintf(int type, char* fmt, ...);
void   BeaconOutput(int type, char* data, int len);
void   BeaconDataParse(datap* parser, char* buffer, int size);
char*  BeaconDataExtract(datap* parser, int* size);
int    BeaconDataInt(datap* parser);
short  BeaconDataShort(datap* parser);
int    BeaconDataLength(datap* parser);
BOOL   BeaconUseToken(HANDLE token);
void   BeaconRevertToken(void);
BOOL   BeaconIsAdmin(void);

/* Output types */
#define CALLBACK_OUTPUT      0x0
#define CALLBACK_OUTPUT_OEM  0x1e
#define CALLBACK_ERROR       0x0d
#define CALLBACK_OUTPUT_UTF8 0x20

/* Data parser */
typedef struct {
    char* original;
    char* buffer;
    int   length;
    int   size;
} datap;

/* Dynamic function resolution macros */
#define DECLSPEC_IMPORT __declspec(dllimport)

/* Secur32.dll */
DECLSPEC_IMPORT NTSTATUS WINAPI SECUR32$LsaConnectUntrusted(PHANDLE);
DECLSPEC_IMPORT NTSTATUS WINAPI SECUR32$LsaLookupAuthenticationPackage(HANDLE, PLSA_STRING, PULONG);
DECLSPEC_IMPORT NTSTATUS WINAPI SECUR32$LsaCallAuthenticationPackage(HANDLE, ULONG, PVOID, ULONG, PVOID*, PULONG, PNTSTATUS);
DECLSPEC_IMPORT NTSTATUS WINAPI SECUR32$LsaDeregisterLogonProcess(HANDLE);
DECLSPEC_IMPORT NTSTATUS WINAPI SECUR32$LsaFreeReturnBuffer(PVOID);
DECLSPEC_IMPORT NTSTATUS WINAPI SECUR32$LsaRegisterLogonProcess(PLSA_STRING, PHANDLE, PLSA_OPERATIONAL_MODE);
DECLSPEC_IMPORT NTSTATUS WINAPI SECUR32$LsaEnumerateLogonSessions(PULONG, PLUID*);
DECLSPEC_IMPORT NTSTATUS WINAPI SECUR32$LsaGetLogonSessionData(PLUID, PVOID*);

/* Advapi32.dll */
DECLSPEC_IMPORT BOOL WINAPI ADVAPI32$OpenProcessToken(HANDLE, DWORD, PHANDLE);
DECLSPEC_IMPORT BOOL WINAPI ADVAPI32$GetTokenInformation(HANDLE, TOKEN_INFORMATION_CLASS, LPVOID, DWORD, PDWORD);
DECLSPEC_IMPORT BOOL WINAPI ADVAPI32$LookupPrivilegeValueA(LPCSTR, LPCSTR, PLUID);
DECLSPEC_IMPORT BOOL WINAPI ADVAPI32$AdjustTokenPrivileges(HANDLE, BOOL, PTOKEN_PRIVILEGES, DWORD, PTOKEN_PRIVILEGES, PDWORD);
DECLSPEC_IMPORT BOOL WINAPI ADVAPI32$ImpersonateSelf(SECURITY_IMPERSONATION_LEVEL);
DECLSPEC_IMPORT BOOL WINAPI ADVAPI32$RevertToSelf(void);
DECLSPEC_IMPORT BOOL WINAPI ADVAPI32$DuplicateTokenEx(HANDLE, DWORD, LPSECURITY_ATTRIBUTES, SECURITY_IMPERSONATION_LEVEL, TOKEN_TYPE, PHANDLE);

/* Kernel32.dll */
DECLSPEC_IMPORT HANDLE WINAPI KERNEL32$GetCurrentProcess(void);
DECLSPEC_IMPORT BOOL   WINAPI KERNEL32$CloseHandle(HANDLE);
DECLSPEC_IMPORT DWORD  WINAPI KERNEL32$GetLastError(void);
DECLSPEC_IMPORT LPVOID WINAPI KERNEL32$HeapAlloc(HANDLE, DWORD, SIZE_T);
DECLSPEC_IMPORT BOOL   WINAPI KERNEL32$HeapFree(HANDLE, DWORD, LPVOID);
DECLSPEC_IMPORT HANDLE WINAPI KERNEL32$GetProcessHeap(void);
DECLSPEC_IMPORT void   WINAPI KERNEL32$RtlZeroMemory(PVOID, SIZE_T);
DECLSPEC_IMPORT int    WINAPI KERNEL32$MultiByteToWideChar(UINT, DWORD, LPCCH, int, LPWSTR, int);
DECLSPEC_IMPORT int    WINAPI KERNEL32$WideCharToMultiByte(UINT, DWORD, LPCWCH, int, LPSTR, int, LPCCH, LPBOOL);
DECLSPEC_IMPORT void*  WINAPI KERNEL32$VirtualAlloc(LPVOID, SIZE_T, DWORD, DWORD);
DECLSPEC_IMPORT BOOL   WINAPI KERNEL32$VirtualFree(LPVOID, SIZE_T, DWORD);

/* Msvcrt */
DECLSPEC_IMPORT int    MSVCRT$sprintf(char*, const char*, ...);
DECLSPEC_IMPORT int    MSVCRT$_snprintf(char*, size_t, const char*, ...);
DECLSPEC_IMPORT void*  MSVCRT$memcpy(void*, const void*, size_t);
DECLSPEC_IMPORT void*  MSVCRT$memset(void*, int, size_t);
DECLSPEC_IMPORT int    MSVCRT$memcmp(const void*, const void*, size_t);
DECLSPEC_IMPORT size_t MSVCRT$strlen(const char*);
DECLSPEC_IMPORT size_t MSVCRT$wcslen(const wchar_t*);
DECLSPEC_IMPORT int    MSVCRT$strcmp(const char*, const char*);
DECLSPEC_IMPORT int    MSVCRT$_stricmp(const char*, const char*);
DECLSPEC_IMPORT int    MSVCRT$wcscmp(const wchar_t*, const wchar_t*);
DECLSPEC_IMPORT char*  MSVCRT$strcpy(char*, const char*);
DECLSPEC_IMPORT char*  MSVCRT$strcat(char*, const char*);
DECLSPEC_IMPORT unsigned long MSVCRT$strtoul(const char*, char**, int);
DECLSPEC_IMPORT int    MSVCRT$atoi(const char*);
DECLSPEC_IMPORT int    MSVCRT$_vsnprintf(char*, size_t, const char*, va_list);
DECLSPEC_IMPORT wchar_t* MSVCRT$wcscpy(wchar_t*, const wchar_t*);
DECLSPEC_IMPORT wchar_t* MSVCRT$wcscat(wchar_t*, const wchar_t*);

/* Remapping macros for BOF */
#define LsaConnectUntrusted          SECUR32$LsaConnectUntrusted
#define LsaLookupAuthenticationPackage SECUR32$LsaLookupAuthenticationPackage
#define LsaCallAuthenticationPackage SECUR32$LsaCallAuthenticationPackage
#define LsaDeregisterLogonProcess    SECUR32$LsaDeregisterLogonProcess
#define LsaFreeReturnBuffer          SECUR32$LsaFreeReturnBuffer
#define LsaRegisterLogonProcess      SECUR32$LsaRegisterLogonProcess
#define LsaEnumerateLogonSessions    SECUR32$LsaEnumerateLogonSessions
#define LsaGetLogonSessionData       SECUR32$LsaGetLogonSessionData

#define OpenProcessToken       ADVAPI32$OpenProcessToken
#define GetTokenInformation    ADVAPI32$GetTokenInformation
#define LookupPrivilegeValueA  ADVAPI32$LookupPrivilegeValueA
#define AdjustTokenPrivileges  ADVAPI32$AdjustTokenPrivileges
#define ImpersonateSelf        ADVAPI32$ImpersonateSelf
#define RevertToSelf           ADVAPI32$RevertToSelf
#define DuplicateTokenEx       ADVAPI32$DuplicateTokenEx

#define GetCurrentProcess  KERNEL32$GetCurrentProcess
#define CloseHandle        KERNEL32$CloseHandle
#define GetLastError       KERNEL32$GetLastError
#define HeapAlloc          KERNEL32$HeapAlloc
#define HeapFree           KERNEL32$HeapFree
#define GetProcessHeap     KERNEL32$GetProcessHeap
#define RtlZeroMemory      KERNEL32$RtlZeroMemory
#define MultiByteToWideChar KERNEL32$MultiByteToWideChar
#define WideCharToMultiByte KERNEL32$WideCharToMultiByte
#define VirtualAlloc       KERNEL32$VirtualAlloc
#define VirtualFree        KERNEL32$VirtualFree

#define sprintf    MSVCRT$sprintf
#define _snprintf  MSVCRT$_snprintf
#define memcpy     MSVCRT$memcpy
#define memset     MSVCRT$memset
#define memcmp     MSVCRT$memcmp
#define strlen     MSVCRT$strlen
#define wcslen     MSVCRT$wcslen
#define strcmp      MSVCRT$strcmp
#define _stricmp   MSVCRT$_stricmp
#define wcscmp     MSVCRT$wcscmp
#define strcpy     MSVCRT$strcpy
#define strcat     MSVCRT$strcat
#define strtoul    MSVCRT$strtoul
#define atoi       MSVCRT$atoi
#define _vsnprintf MSVCRT$_vsnprintf
#define wcscpy     MSVCRT$wcscpy
#define wcscat     MSVCRT$wcscat

#else /* !BOF - normal compilation */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#pragma comment(lib, "secur32.lib")
#pragma comment(lib, "advapi32.lib")

#endif /* BOF */
