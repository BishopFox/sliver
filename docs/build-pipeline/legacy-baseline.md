# Legacy implant build baseline (M0)

This is the historical compatibility baseline for the build-pipeline roadmap,
read from commit `17724ef9b8de89ed53e64bdccb9852592a651599`. It describes
the M0 behavior, including the template selector and external builder path
that have since been removed. The companion
[`legacy-acceptance.json`](legacy-acceptance.json) enumerates target/output
routes and negative cases. Run `python3 tools/check-build-pipeline-baseline.py`
from the repository root to check the list and source-backed invariants. That
check does **not** compile an implant or prove a live connection; release gates
still require `./go-tests.sh` and target-specific E2E evidence.

## Console entry points and flags

`generate` and `generate beacon` parse the shared flags below; `profiles new`
and `profiles new beacon` use the same parser and save one `ImplantConfig`.
Interactive forms appear only in the console when no argument or flag is set;
the CLI path uses the defaults. The ordinary build defaults to a session; the
beacon subcommand sets `IsBeacon` and resolves its interval and jitter.
[Commands](../../client/command/generate/commands.go),
[parser](../../client/command/generate/generate.go), and
[beacon parser](../../client/command/generate/generate-beacon.go) are the source.

| Group | Flags (short name) | Current default or handling |
| --- | --- | --- |
| Target and output | `--os` (`-o`), `--arch` (`-a`), `--format` (`-f`), `--service-name` | `windows`, `amd64`, `exe`, empty service name. Formats: `exe`, `shared`, `archive`, `service`, `shellcode`. At the M0 snapshot, an **unknown** format silently selected `exe`; M2 now returns a capability error. |
| Build identity | `--name` (`-N`), `--template` (`-I`), `--save` (`-s`), `--external-builder` (`-E`) | Empty name makes the server choose a codename. Template defaults to `sliver`; save defaults to the current directory; external builder defaults off. `--template`, `--save`, and `--external-builder` are bound on `profiles new` through shared flag setup, though creating a profile only stores the parsed config. |
| Compilation | `--debug` (`-d`), `--debug-file` (`-O`), `--evasion` (`-e`), `--skip-symbols` (`-l`), `--exports` (`-v`), `--run-at-load` (`-R`), `--netgo` (`-q`), `--canary` (`-c`) | Debug/evasion/run-at-load/netgo off, debug file and canary domains empty, symbol obfuscation on unless debug or skip-symbols, exports `StartW,VoidFunc,DllInstall,DllRegisterServer,DllUnregisterServer`. Run-at-load applies to shared libraries. Canary domains are comma separated and normalized with a trailing dot. |
| C2 URLs | `--mtls` (`-m`), `--wg` (`-g`), `--http` (`-b`), `--dns` (`-n`), `--named-pipe` (`-p`), `--tcp-pivot` (`-i`), `--key-exchange` (`-X`), `--tcp-comms` (`-T`) | URLs default empty. At least one parsed URL is required. Named pipe is rejected on non-Windows targets. HTTP accepts HTTP and HTTPS. WireGuard key exchange defaults to port `1337`; TCP comms defaults to `8888`. |
| C2 inclusion | `--include-mtls`, `--include-http`, `--include-wg`, `--include-dns`, `--include-named-pipe`, `--include-tcp-pivot`, `--all-protocols` | All false. These force transport code inclusion; the server also enables transports used by C2 URLs. `--all-protocols` forces every inclusion flag, but does not remove the at-least-one-URL requirement. |
| C2 behavior | `--strategy` (`-Z`), `--reconnect` (`-j`), `--poll-timeout` (`-P`), `--max-errors` (`-k`), `--c2profile` (`-C`), `--traffic-encoders` (`-A`) | Empty strategy (which runs sequentially; `s`, `r`, `rd` accepted when specified), 60 seconds, 360 seconds, 1000 errors, `default`, empty encoders. Traffic encoders apply only to HTTP C2. |
| Execution limits | `--limit-datetime` (`-w`), `--limit-domainjoined` (`-x`), `--limit-username` (`-y`), `--limit-hostname` (`-z`), `--limit-fileexists` (`-F`), `--limit-locale` (`-L`) | Empty or false. |
| PE metadata | `--spoof-metadata` | Optional donor path or config file; handled after Generate by a separate RPC that updates the stored build. Applies only to supported PE output. |
| Shellcode | `--shellcode-encoder`, `--shellcode-entropy`, `--shellcode-compress`, `--shellcode-exitopt`, `--shellcode-bypass`, `--shellcode-headers`, `--shellcode-thread`, `--shellcode-unicode`, `--shellcode-oep` | Empty encoder; Windows Donut defaults `1`, false, `1`, `3`, `1`, false, false, `0`. If no encoder is specified, the client offers a compatible encoder choice. Only compression is used by macOS and Linux converters. The hidden/deprecated aliases are `--donut-entropy`, `--donut-compress`, `--donut-exitopt`, `--donut-bypass`, `--donut-headers`, `--donut-thread`, `--donut-unicode`, and `--donut-oep`; an explicitly set `--shellcode-*` option wins over its alias. |
| Beacon | `--days` (`-D`), `--hours` (`-H`), `--minutes` (`-M`), `--seconds` (`-S`), `--jitter` (`-J`) | `0`, `0`, `0`, `60`, `30` seconds. When days/hours/minutes are set and seconds is not explicitly set, the default 60 seconds is omitted. Intervals below five seconds are rejected. |

Command-only flags: `generate` and `profiles` each have `--timeout` (`-t`),
defaulting to 60 seconds. `profiles generate <name>` accepts `--save` (`-s`),
`--name` (`-n`), `--external-builder` (`-E`), and a per-build
`--shellcode-encoder` override (`none` disables). `profiles stage <name>`
accepts `--save` (`-s`), `--name` (`-n`), `--aes-encrypt-key` (`-k`),
`--aes-encrypt-iv` (`-i`), `--rc4-encrypt-key` (`-r`), `--prepend-size` (`-p`),
and `--compress` (`-c`; zlib, gzip, deflate9, deflate).

OS aliases are `mac`/`macos`/`osx` to `darwin`, `win` to `windows`, and `lin`
to `linux`; the parser also accepts the historical `shit` alias for Windows.
Architecture aliases are `x64` or a `64` prefix to `amd64`, and `x86` or a `32`
prefix to `386`. Unsupported targets prompt for an attempt at a generic build
rather than failing automatically. See [target parser](../../client/command/generate/generate.go).

## Target, mode, protocol, and artifact behavior

The eight static targets are Darwin amd64/arm64, Linux 386/amd64/arm64, and
Windows 386/amd64/arm64. `GetCompilerTargets` always advertises executables,
Windows service outputs, and Windows 386/amd64 shellcode. Shared libraries,
archives, Darwin arm64 shellcode, and Linux amd64/arm64 shellcode are
advertised only on a native host or when a suitable cross compiler is
detected. Darwin hosts
omit Linux/386 CGo outputs even when a cross compiler is found. The route list
records the RPC path that would be attempted, not a claim that every tuple is
advertised on every host or successfully compiled. Successful execution of a
conditional CGo route still depends on the actual toolchain.
The [server target list](../../server/generate/binaries.go) and
[client target list](../../client/command/generate/generate.go) are separate
today; both are checked by the companion script.

| Operator format | Existing core build | Final artifact | Important distinction |
| --- | --- | --- | --- |
| `exe` | Go default mode, CGO disabled | PE `.exe`, ELF executable, or Mach-O executable | Same build function for sessions and beacons. |
| `service` | Go default mode, CGO disabled | Windows PE `.exe` | Routed through `SliverExecutable`; `IsService` renders Windows `svc.Run` startup, and `--service-name` is embedded there. Not a distinct Go build mode. |
| `shared` | Go `c-shared`, CGO enabled | Windows `.dll`, macOS `.dylib`, Linux `.so` | Export symbols and optional `run-at-load` matter. |
| `archive` | Go `c-archive`, CGO enabled | `.zip` with `.a`, generated `.h`, and `main.h` | The protobuf enum is `GO_ARCHIVE`, but the output is a **C archive bundle**, not Go's `archive` build mode or a bare `.a`. |
| `shellcode` | Windows: Go `pie` then Donut; macOS arm64: Go `c-shared` then Beignet; Linux amd64/arm64: Go `c-shared` then Malasada | Architecture-specific raw `.bin` | Converter options and available toolchains vary by OS. Optional Shikata Ga Nai/XOR/dynamic XOR then run in a separate encoder RPC. |

The console accepts mTLS, WireGuard, HTTP, HTTPS, and DNS URLs for sessions
and beacons. Named pipes require Windows and are implemented by the session
transport; TCP pivot is also implemented by the session transport. The current
parser can accept these pivot URLs in beacon configurations, but the beacon
transport has no corresponding handler. M2 planning must reject those runtime
combinations. Encoder availability is architecture-specific: amd64 supports
Shikata Ga Nai, XOR, and dynamic XOR; arm64 supports XOR and dynamic XOR; 386
supports Shikata Ga Nai. These are separate from converter support.

## RPC and persistence baseline

`Generate(GenerateReq)` chooses or validates a name, resolves an existing
profile config by ID when supplied, enables requested C2 transports, builds
one of the four direct paths (`service` shares executable; `shellcode` includes
its converter), saves the artifact and checksums, and returns its bytes.
`Regenerate` reads the stored file. Client `--spoof-metadata` invokes
`GenerateSpoofMetadata` after `Generate`, then calls `Regenerate` before the
client download. Optional shellcode encoding is later still: the client calls
`ShellcodeEncoder` and writes the returned bytes locally, while the server's
saved build and hash remain the pre-encoding bytes. This is an M4 parity gap.
See [Generate RPC](../../server/rpc/rpc-generate.go),
[client compile](../../client/command/generate/generate.go), and
[build storage](../../server/generate/implants.go).

At M0, `GenerateExternal` required a named, connected builder and sent a whole
Sliver config via an event. The client filtered builders by target and output
format, then prompted when more than one matched; the server received one
final file. This was the M5 migration baseline. The event and RPC route have
since been removed in favor of assigned build steps.

`SaveImplantProfile` stores one `ImplantConfig`; saving the same name updates
it. `ImplantProfiles` returns the current list. `profiles generate` uses that
config, with optional name, encoder override, and external builder. `profiles
stage` calls `GenerateStage`, which applies size framing, compression, and
encryption. Currently `GenerateStage` stores the transformed `stage2` but
returns `fileData` after size framing, before compression/encryption; downloaded
and stored bytes can differ. Deleting a profile currently deletes its linked
builds and files. These are explicit M4/M6 migration gaps.
The shared `GetSliverBinary` helper used by stage listeners and `psexec`
optionally encodes shellcode after `Generate` and saves the profile again.
[Profile RPC](../../server/rpc/rpc-generate.go),
[profile storage](../../server/generate/profiles.go), and
[profile command](../../client/command/generate/profiles-generate.go), and
[shared helper](../../client/command/generate/helpers.go) are the source.

## Profile consumers to migrate

| Consumer | Current path and artifact assumption |
| --- | --- |
| `profiles generate` / `profiles stage` | Generate from a saved config; stage applies separate framing/compression/encryption. |
| `jobs stage` | Reads a profile, generates bytes client-side (including optional shellcode encoding), applies listener staging options, and starts the TCP stager listener. |
| `psexec` | Reads a profile and generates bytes, then uploads them as an `.exe` service binary without a profile-format check. |
| `backdoor` | Server requires `OutputFormat_SHELLCODE` and generates shellcode for injection. |
| `dllhijack` | Server requires `OutputFormat_SHARED_LIB` and generates a shared library. |
| `clean`, `profiles rm` | Delete profiles through `DeleteImplantProfile`; completed builds linked to the profile are removed today. |
| `profiles`, `profiles info`, completions | Read `ImplantProfiles` and display the legacy enum and config fields. |

Sources: [stage listener](../../client/command/jobs/stage.go),
[psexec](../../client/command/exec/psexec.go),
[backdoor RPC](../../server/rpc/rpc-backdoor.go),
[DLL hijack RPC](../../server/rpc/rpc-hijack.go),
[clean](../../client/command/clean/clean.go), and
[profile commands](../../client/command/generate/profiles.go).

## Acceptance use

`legacy-acceptance.json` expands 32 target/output rows across session and
beacon modes, plus explicit negative/quirk and protocol rows. The checker
validates every row's expected route, mode, artifact, and source references.
The list is an acceptance input for M2 through M7; it is **not** evidence that
all target builds or live transports pass on the current host. When running
the later matrix, record build-only results and live session/beacon results
separately, including host OS, cross compiler, and exact command.
