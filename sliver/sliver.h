#ifdef __WIN32
#include <windows.h>

void RunSliver();

BOOL WINAPI DllMain(
    HINSTANCE _hinstDLL, // handle to DLL module
    DWORD _fdwReason,    // reason for calling function
    LPVOID _lpReserved   // reserved
);
#endif