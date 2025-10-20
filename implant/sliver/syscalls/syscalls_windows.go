package syscalls

//go:generate go run $GOPATH/src/golang.org/x/sys/windows/mkwinsyscall -output zsyscalls_windows.go syscalls_windows.go

//sys InitializeProcThreadAttributeList(lpAttributeList *PROC_THREAD_ATTRIBUTE_LIST, dwAttributeCount uint32, dwFlags uint32, lpSize *uintptr) (err error) = kernel32.InitializeProcThreadAttributeList
//sys GetProcessHeap() (procHeap windows.Handle, err error) = kernel32.GetProcessHeap
//sys HeapAlloc(hHeap windows.Handle, dwFlags uint32, dwBytes uintptr) (lpMem uintptr, err error) = kernel32.HeapAlloc
//sys HeapReAlloc(hHeap windows.Handle, dwFlags uint32, lpMem uintptr, dwBytes uintptr) (lpRes uintptr, err error) = kernel32.HeapReAlloc
//sys HeapSize(hHeap windows.Handle, dwFlags uint32, lpMem uintptr) (res uint32, err error) = kernel32.HeapSize
//sys UpdateProcThreadAttribute(lpAttributeList *PROC_THREAD_ATTRIBUTE_LIST, dwFlags uint32, attribute uintptr, lpValue *uintptr, cbSize uintptr, lpPreviousValue uintptr, lpReturnSize *uintptr) (err error) = kernel32.UpdateProcThreadAttribute
//sys CreateProcess(appName *uint16, commandLine *uint16, procSecurity *windows.SecurityAttributes, threadSecurity *windows.SecurityAttributes, inheritHandles bool, creationFlags uint32, env *uint16, currentDir *uint16, startupInfo *StartupInfoEx, outProcInfo *windows.ProcessInformation) (err error) = kernel32.CreateProcessW
//sys CreateProcessWithLogonW(username *uint16, domain *uint16, password *uint16, logonFlags uint32, appName *uint16, commandLine *uint16, creationFlags uint32, env *uint16, currentDir *uint16, startupInfo *StartupInfoEx, outProcInfo *windows.ProcessInformation) (err error) = advapi32.CreateProcessWithLogonW
//sys VirtualAllocEx(hProcess windows.Handle, lpAddress uintptr, dwSize uintptr, flAllocationType uint32, flProtect uint32) (addr uintptr, err error) = kernel32.VirtualAllocEx
//sys WriteProcessMemory(hProcess windows.Handle, lpBaseAddress uintptr, lpBuffer *byte, nSize uintptr, lpNumberOfBytesWritten *uintptr) (err error) = kernel32.WriteProcessMemory
//sys VirtualProtectEx(hProcess windows.Handle, lpAddress uintptr, dwSize uintptr, flNewProtect uint32, lpflOldProtect *uint32) (err error) = kernel32.VirtualProtectEx
//sys QueueUserAPC(pfnAPC uintptr, hThread windows.Handle, dwData uintptr) (err error) = kernel32.QueueUserAPC
//sys DeleteProcThreadAttributeList(lpAttributeList *PROC_THREAD_ATTRIBUTE_LIST) = kernel32.DeleteProcThreadAttributeList
//sys HeapFree(hHeap windows.Handle, dwFlags uint32, lpMem uintptr) (err error) = kernel32.HeapFree
//sys CreateRemoteThread(hProcess windows.Handle, lpThreadAttributes *windows.SecurityAttributes, dwStackSize uint32, lpStartAddress uintptr, lpParameter uintptr, dwCreationFlags uint32, lpThreadId *uint32)(threadHandle windows.Handle, err error) = kernel32.CreateRemoteThread
//sys CreateThread(lpThreadAttributes *windows.SecurityAttributes, dwStackSize uint32, lpStartAddress uintptr, lpParameter uintptr, dwCreationFlags uint32, lpThreadId *uint32)(threadHandle windows.Handle, err error) = kernel32.CreateThread
//sys GetExitCodeThread(hTread windows.Handle, lpExitCode *uint32) (err error) = kernel32.GetExitCodeThread

//sys MiniDumpWriteDump(hProcess windows.Handle, pid uint32, hFile uintptr, dumpType uint32, exceptionParam uintptr, userStreamParam uintptr, callbackParam uintptr) (err error) = DbgCore.MiniDumpWriteDump
//sys ImpersonateLoggedOnUser(hToken windows.Token) (err error) = advapi32.ImpersonateLoggedOnUser
//sys LogonUser(lpszUsername *uint16, lpszDomain *uint16, lpszPassword *uint16, dwLogonType uint32, dwLogonProvider uint32, phToken *windows.Token) (err error) = advapi32.LogonUserW

//sys GetDC(HWND windows.Handle) (HDC windows.Handle, err error) = User32.GetDC
//sys ReleaseDC(hWnd windows.Handle, hDC windows.Handle) (int uint32, err error) = User32.ReleaseDC
//sys CreateCompatibleDC(hdc windows.Handle) (HDC windows.Handle, err error) =  Gdi32.CreateCompatibleDC
//sys GetDesktopWindow() (HWND windows.Handle, err error) = User32.GetDesktopWindow
//sys DeleteDC(hdc windows.Handle) (BOOL uint32, err error) = Gdi32.DeleteDC
//sys CreateCompatibleBitmap(hdc windows.Handle, cx int, cy int) (HBITMAP windows.Handle, err error) = Gdi32.CreateCompatibleBitmap
//sys DeleteObject(ho windows.Handle) (BOOL uint32, err error) = Gdi32.DeleteObject
//sys GlobalAlloc(uFlags uint, dwBytes uintptr) (HGLOBAL windows.Handle, err error) = Kernel32.GlobalAlloc
//sys GlobalFree(hMem windows.Handle) (HGLOBAL windows.Handle, err error) = Kernel32.GlobalFree
//sys GlobalLock(hMem windows.Handle) (LPVOID uintptr, err error) = Kernel32.GlobalLock
//sys GlobalUnlock(hMem windows.Handle) (BOOL uint32, err error) = Kernel32.GlobalUnlock
//sys SelectObject(hdc windows.Handle, h windows.Handle) (HGDIOBJ windows.Handle, err error) = Gdi32.SelectObject
//sys BitBlt(hdc windows.Handle, x uint32, y uint32, cx uint32, cy uint32, hdcSrc windows.Handle, x1 uint32, y1 uint32, rop int32) (BOOL int, err error) = Gdi32.BitBlt
//sys GetDIBits(hdc windows.Handle, hbm windows.Handle, start uint32, cLines uint32, lpvBits uintptr, lpbmi uintptr, usage int) (ret int, err error) = Gdi32.GetDIBits
//sys PssCaptureSnapshot(processHandle windows.Handle, captureFlags uint32, threadContextFlags uint32, snapshotHandle *windows.Handle) (err error) = kernel32.PssCaptureSnapshot

//sys RtlCopyMemory(dest uintptr, src uintptr, dwSize uint32) = ntdll.RtlCopyMemory
//sys GetProcessMemoryInfo(process windows.Handle, ppsmemCounters *ProcessMemoryCounters, cb uint32) (err error) = psapi.GetProcessMemoryInfo
//sys LookupPrivilegeNameW(systemName string, luid *uint64, buffer *uint16, size *uint32) (err error) = advapi32.LookupPrivilegeNameW
//sys LookupPrivilegeDisplayNameW(systemName string, privilegeName *uint16, buffer *uint16, size *uint32, languageId *uint32) (err error) = advapi32.LookupPrivilegeDisplayNameW

//sys Module32FirstW(hSnapshot windows.Handle, lpme *MODULEENTRY32W) (err error) = kernel32.Module32FirstW
//sys RegSaveKeyW(hKey windows.Handle, lpFile *uint16, lpSecurityAttributes *windows.SecurityAttributes) (err error) = advapi32.RegSaveKeyW
