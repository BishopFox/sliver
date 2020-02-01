package syscalls

//go:generate go run $GOPATH/src/golang.org/x/sys/windows/mkwinsyscall -output zsyscall_windows.go syscall_windows.go

//sys InitializeProcThreadAttributeList(lpAttributeList *PROC_THREAD_ATTRIBUTE_LIST, dwAttributeCount uint32, dwFlags uint32, lpSize *uintptr) (err error) = kernel32.InitializeProcThreadAttributeList
//sys GetProcessHeap() (procHeap windows.Handle, err error) = kernel32.GetProcessHeap
//sys HeapAlloc(hHeap windows.Handle, dwFlags uint32, dwBytes uintptr) (lpMem uintptr, err error) = kernel32.HeapAlloc
//sys UpdateProcThreadAttribute(lpAttributeList *PROC_THREAD_ATTRIBUTE_LIST, dwFlags uint32, attribute uintptr, lpValue *uintptr, cbSize uintptr, lpPreviousValue uintptr, lpReturnSize *uintptr) (err error) = kernel32.UpdateProcThreadAttribute
//sys CreateProcess(appName *uint16, commandLine *uint16, procSecurity *windows.SecurityAttributes, threadSecurity *windows.SecurityAttributes, inheritHandles bool, creationFlags uint32, env *uint16, currentDir *uint16, startupInfo *StartupInfoEx, outProcInfo *windows.ProcessInformation) (err error) = kernel32.CreateProcessW
//sys VirtualAllocEx(hProcess windows.Handle, lpAddress uintptr, dwSize uintptr, flAllocationType uint32, flProtect uint32) (addr uintptr, err error) = kernel32.VirtualAllocEx
//sys WriteProcessMemory(hProcess windows.Handle, lpBaseAddress uintptr, lpBuffer *byte, nSize uintptr, lpNumberOfBytesWritten *uintptr) (err error) = kernel32.WriteProcessMemory
//sys VirtualProtectEx(hProcess windows.Handle, lpAddress uintptr, dwSize uintptr, flNewProtect uint32, lpflOldProtect *uint32) (err error) = kernel32.VirtualProtectEx
//sys QueueUserAPC(pfnAPC uintptr, hThread windows.Handle, dwData uintptr) (err error) = kernel32.QueueUserAPC
//sys DeleteProcThreadAttributeList(lpAttributeList *PROC_THREAD_ATTRIBUTE_LIST) = kernel32.DeleteProcThreadAttributeList
//sys HeapFree(hHeap windows.Handle, dwFlags uint32, lpMem uintptr) (err error) = kernel32.HeapFree
//sys CreateRemoteThread(hProcess windows.Handle, lpThreadAttributes *windows.SecurityAttributes, dwStackSize uint32, lpStartAddress uintptr, lpParameter uintptr, dwCreationFlags uint32, lpThreadId *uint32)(threadHandle windows.Handle, err error) = kernel32.CreateRemoteThread
//sys CreateThread(lpThreadAttributes *windows.SecurityAttributes, dwStackSize uint32, lpStartAddress uintptr, lpParameter uintptr, dwCreationFlags uint32, lpThreadId *uint32)(threadHandle windows.Handle, err error) = kernel32.CreateThread
//sys GetExitCodeThread(hTread windows.Handle, lpExitCode *uint32) (err error) = kernel32.GetExitCodeThread

//sys MiniDumpWriteDump(hProcess windows.Handle, pid uint32, hFile uintptr, dumpType uint32, exceptionParam uintptr, userStreamParam uintptr, callbackParam uintptr) (err error) = DbgHelp.MiniDumpWriteDump
//sys ImpersonateLoggedOnUser(hToken windows.Token) (err error) = advapi32.ImpersonateLoggedOnUser
