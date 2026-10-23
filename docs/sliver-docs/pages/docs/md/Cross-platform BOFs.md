Sliver's built-in [Reflektor](https://github.com/sliverarmory/reflektor/tree/v0.0.8) executor runs Beacon Object Files (BOFs) on Windows, macOS, and Linux. A BOF is a relocatable native object with an entry point such as `go(char *args, int length)` and Beacon callbacks for arguments and output. It executes inside the implant process.

This page covers writing and compiling basic BOFs. For Armory installation, argument manifests, and compatibility with older components, see [BOF and COFF Support](/docs?name=BOF+and+COFF+Support). Projects with CNA wrappers can also use [OPFOR](/docs?name=OPFOR).

## Platform support

The following targets support built-in BOF execution in Sliver:

| Implant OS | Implant architecture (`GOARCH`) | Object format | Typical compiler |
| --- | --- | --- | --- |
| Windows (`windows`) | `386`, `amd64`, `arm64` | COFF relocatable object | MinGW-w64 GCC or Clang targeting Windows |
| macOS (`darwin`) | `amd64`, `arm64` | Mach-O relocatable object (`MH_OBJECT`) | Apple Clang |
| Linux (`linux`) | `386`, `amd64`, `arm64` | ELF relocatable object (`ET_REL`) | GCC or Clang targeting Linux |

The object must match the **implant's** operating system and CPU architecture, regardless of the client or server platform. `amd64` means x86-64; `386` means x86; `arm64` means AArch64. The `.o` suffix alone does not identify the format. Compile a separate artifact for each OS/architecture pair; renaming a Windows COFF object does not turn it into a Linux or macOS BOF.

Reflektor also accepts legacy ELF BOFs on macOS when they match the host architecture and ABI. Use native Mach-O for new macOS BOFs. Reflektor's standalone library supports additional targets, but the table above describes Sliver's built-in integration.

Use compatible client, server, and implant versions with the `bof_v1` capability. Setting `"bof_executor": "reflektor"` explicitly selects built-in execution when that capability is available. The examples below require this support and do not need a COFF Loader dependency. See [BOF and COFF Support](/docs?name=BOF+and+COFF+Support) for the optional legacy fallback.

## Basic examples

These three examples take no arguments and emit one message through `BeaconOutput`. Save the source files in a directory named `hello-bof`. Run the compiler commands from that directory; each command produces an object with `-c` rather than linking an executable or shared library.

The declarations are included directly to keep the examples self-contained. Windows imports use `__declspec(dllimport)`; Unix platforms use normal external symbols. The supported targets use 32-bit `int` for the entry-point length and Beacon callback parameters.

### Windows: COFF

Save as `hello-windows.c`:

```c
__declspec(dllimport) void BeaconOutput(int type, char *data, int length);

void go(char *args, int length) {
    static char message[] = "Hello from a Windows BOF!\n";
    (void)args;
    (void)length;
    BeaconOutput(0, message, (int)(sizeof(message) - 1));
}
```

With a MinGW-w64 cross compiler, build the Windows x86-64 object:

```bash
x86_64-w64-mingw32-gcc -Os -c -fno-stack-protector \
  -fno-unwind-tables -fno-asynchronous-unwind-tables \
  hello-windows.c -o hello.windows.amd64.o
```

For a Windows x86 implant, use `i686-w64-mingw32-gcc` with the same flags and name the result `hello.windows.386.o`. For Windows ARM64, Clang can compile this header-free example:

```bash
clang --target=aarch64-w64-windows-gnu -Os -c -fno-stack-protector \
  -fno-unwind-tables -fno-asynchronous-unwind-tables \
  hello-windows.c -o hello.windows.arm64.o
```

Existing Windows BOFs may import system functions using names such as `KERNEL32$GetCurrentProcessId`. Those imports and their calling conventions remain Windows-specific; port the source and imports before compiling for another OS.

### macOS: Mach-O

Save as `hello-macos.c`:

```c
extern void BeaconOutput(int type, char *data, int length);

void go(char *args, int length) {
    static char message[] = "Hello from a macOS BOF!\n";
    (void)args;
    (void)length;
    BeaconOutput(0, message, (int)(sizeof(message) - 1));
}
```

On macOS, install the Xcode Command Line Tools (`xcode-select --install`) if Clang is unavailable. Build separate objects for Apple Silicon and Intel implants:

```bash
clang -arch arm64 -Os -c -fno-stack-protector \
  -fno-unwind-tables -fno-asynchronous-unwind-tables \
  hello-macos.c -o hello.darwin.arm64.o

clang -arch x86_64 -Os -c -fno-stack-protector \
  -fno-unwind-tables -fno-asynchronous-unwind-tables \
  hello-macos.c -o hello.darwin.amd64.o
```

Keep these as separate thin Mach-O objects. Reflektor does not accept a universal/fat BOF object. Clang prefixes the C symbols with `_` in Mach-O; the default `go` entry-point selection handles `_go` automatically.

**Apple Silicon:** Native Mach-O `arm64` BOFs must use `BeaconOutput` for output. Reflektor rejects `BeaconPrintf` and `BeaconFormatPrintf` imports for this format because Apple's variadic calling convention differs from the callback bridge. The example above works on both macOS architectures.

### Linux: ELF

Save as `hello-linux.c`:

```c
extern void BeaconOutput(int type, char *data, int length);

void go(char *args, int length) {
    static char message[] = "Hello from a Linux BOF!\n";
    (void)args;
    (void)length;
    BeaconOutput(0, message, (int)(sizeof(message) - 1));
}
```

On an x86-64 Linux build host, use GCC to produce an ELF object:

```bash
gcc -m64 -Os -c -fPIC -fno-stack-protector \
  -fno-unwind-tables -fno-asynchronous-unwind-tables \
  hello-linux.c -o hello.linux.amd64.o
```

For a Linux x86 implant, replace `-m64` with `-m32` and name the result `hello.linux.386.o`; this requires a compiler with 32-bit target support. For ARM64, use a Linux AArch64 cross compiler:

```bash
aarch64-linux-gnu-gcc -Os -c -fPIC -fno-stack-protector \
  -fno-unwind-tables -fno-asynchronous-unwind-tables \
  hello-linux.c -o hello.linux.arm64.o
```

On a native ARM64 Linux build host, `gcc` with the same flags as the ARM64 command produces the ARM64 object. Do not use a macOS host's default `clang` or `gcc` command to build a Linux BOF without selecting a Linux target.

## Load and run in Sliver

Save the following `extension.json` beside the compiled objects in `hello-bof`. The command uses Sliver's modern manifest format, with execution settings inside `commands`. Keep only the `files` entries for artifacts you have built; add `386` or Windows `arm64` entries if you built those variants.

```json
{
  "name": "hello-bof",
  "version": "1.0.0",
  "extension_author": "Example Author",
  "commands": [
    {
      "command_name": "hello-bof",
      "help": "Print a greeting from a platform-native BOF",
      "bof_executor": "reflektor",
      "entrypoint": "go",
      "files": [
        {
          "os": "windows",
          "arch": "amd64",
          "path": "hello.windows.amd64.o"
        },
        {
          "os": "darwin",
          "arch": "amd64",
          "path": "hello.darwin.amd64.o"
        },
        {
          "os": "darwin",
          "arch": "arm64",
          "path": "hello.darwin.arm64.o"
        },
        {
          "os": "linux",
          "arch": "amd64",
          "path": "hello.linux.amd64.o"
        },
        {
          "os": "linux",
          "arch": "arm64",
          "path": "hello.linux.arm64.o"
        }
      ],
      "arguments": []
    }
  ]
}
```

Copy the directory to your operator machine if you compiled on another host, then load it into the Sliver client:

```text
sliver > extensions load /absolute/path/to/hello-bof
sliver > use <session-or-beacon-id>
sliver (selected-target) > hello-bof
```

Sliver selects the artifact using the active session or beacon's exact `os`/`arch` pair. A macOS ARM64 target prints `Hello from a macOS BOF!`; Windows and Linux targets print the corresponding greeting. Sessions return output immediately; beacon output arrives with the asynchronous task result.

For BOFs with arguments, declare them in the manifest in the same order and types consumed by `BeaconDataParse`, `BeaconDataInt`, `BeaconDataShort`, and `BeaconDataExtract`. Sliver packs the arguments into the length-prefixed Beacon format. See the argument type table and conversion example in [BOF and COFF Support](/docs?name=BOF+and+COFF+Support).

## Compatibility and troubleshooting

Reflektor implements the Beacon data, format, and output callbacks, including `BeaconOutput` and supported `BeaconPrintf` calls. It preserves output channel values such as normal output (`0`) and errors (`13`). It does not implement every Beacon integration API: token manipulation, temporary-process, and process-injection callbacks require host integration and fail explicitly when unavailable. Check the [Reflektor BOF documentation](https://github.com/sliverarmory/reflektor/blob/v0.0.8/README.md#beacon-object-files) before importing additional Beacon APIs.

- **No artifact for the target:** Add the exact `files` entry with Go OS/architecture names, such as `darwin`/`arm64`, and the matching object. Paths are relative to `extension.json`; parent-directory traversal is not allowed.
- **Missing built-in support:** Upgrade the client and server and regenerate the implant with BOF support. An older running implant does not gain the capability when only the client is upgraded.
- **Wrong format or machine:** Inspect the object with `file`, `llvm-readobj`, or `objdump`. Check both the OS format and architecture, then rebuild for the implant.
- **Unresolved import:** Use the correct platform ABI and symbol spelling. Windows DLL imports do not carry over to Unix. The examples avoid standard-library and system-library dependencies.
- **Entry point not found:** Export a non-static `go` function with the signature shown above. C++ sources must use `extern "C"` to avoid name mangling. Custom entry points use the exact object symbol name, including Mach-O's leading underscore.

BOFs run native code in the implant process. Keep the entry point short and return normally; crashes or process-exit calls can terminate the implant. Worker threads using Beacon callbacks must finish before the entry point returns because output capture and object lifetime are tied to that execution.
