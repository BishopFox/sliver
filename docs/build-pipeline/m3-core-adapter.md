# M3 core and compiler adapters

The server resolves a neutral `BuildRequest` into exact catalog specs and a
plan. `ExecuteBuild` reads the core spec's executor ID and calls its registered
`buildcore.Adapter`. The generic planner, executor, and record writer receive
typed plans and artifact bytes; they do not read Sliver source or
`ImplantConfig`. The `builtin-sliver-go-v1` adapter owns conversion of plan
options into `ImplantConfig`, C2 defaults, credential preparation, source
rendering, compilation, legacy database save, and output naming.

The existing `Generate` RPC uses the same Sliver adapter through `BuildLegacy`
and stores its returned bytes with `buildcore.CompleteRecord`. Existing
`SliverExecutable`, `SliverSharedLibrary`, and `SliverArchive` exports remain
wrappers around adapter methods. The adapter's shellcode branch still combines
Go compilation with the platform converter; M4 will record those as separate
steps.

Every Go compile invocation in `server/generate`, including shellcode, calls
the registered `buildcompiler` Go driver. It preserves the current Go, Zig,
and Garble settings, modes, tags, flags, and destination paths. The
context-aware entry point lets a canceled `ExecuteBuild` stop compiler work.
Additional compiler drivers can register under a stable ID before their core
specs are advertised.

The completed record currently identifies the bundled Go toolchain. It does
not yet record the selected C compiler or Garble binary identity separately;
the plan retains the declared CGo requirements and the compiler invocation
still uses the selected tools.

A new local core provides a catalog `CoreSpec` with an installed executor ID
and registers an adapter implementing `ExecutorID`, `Supports`, and `Build`.
For example, a `test/core` spec with `executor.id: test-core-adapter-m3` selects
the test adapter without edits to Sliver generation code or planner routing.
The RPC integration test executes that synthetic core and checks the exact
bytes returned by `GetBuildArtifact`.

Client and server transport timing now comes from shared `sliverpb` constants,
and client file matching uses `util/matcher`. Generated implant source copies
the shared timing file alongside the other `sliverpb` files.

`GetCompilerTargets` now checks that a cross-OS C compiler command exists
before advertising CGo outputs. In particular, the default osxcross path on a
Linux server does not advertise Darwin shared libraries or archives unless
osxcross is installed. The spec still describes those supported routes; its
availability is determined by the installed compiler inventory.

`./go-tests.sh --offline-build-pipeline` covers the registry, neutral planning,
adapter selection, direct-format configuration, compiler driver commands,
recorded bytes, and generated source copy without unpacked compiler assets.
With the Linux/arm64 Docker toolchains unpacked, targeted `--generate-only`
tests passed 46 available session/beacon builds across executable, service,
shared-library, and C archive outputs. They inspect file names, PE/ELF/Mach-O
formats, service and `StartW` entrypoints, and archive ZIP entries and CRCs.
The tagged RPC generation test also compiles through `ExecuteBuild` and
compares its recorded checksum and stored bytes with legacy `regenerate`.
The preferred Docker `test` target passed with `GO_TESTS_FLAGS=--skip-generate`.
The existing comprehensive E2E driver passed on Linux/arm64 with generated
mTLS session and beacon implants, verifying live connections and command
execution. That run uses the legacy `Generate` RPC, which now calls the same
Sliver adapter; the targeted RPC test separately exercises `ExecuteBuild`.
`python3 tools/check-build-pipeline-baseline.py` separately checks the 32
documented routes and static dispatch; it does not compile an implant.
