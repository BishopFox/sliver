# Comprehensive E2E command coverage

The static source of truth is `coverage.ComprehensiveCatalog()` in [coverage/catalog.go](coverage/catalog.go). Every scenario is expanded over three transports (`mtls`, `wg`, `http`) and two implant modes (`session`, `beacon`) for each supported target.

The executable catalog contains 56 scenarios across 41 of the 67 finite implant-command RPC methods and expands to 2,016 matrix cells:

| Catalog group | Scenarios | Supported targets per scenario | Required recorded cells | Expected platform `SKIP` cells |
|---|---:|---:|---:|---:|
| Portable | 29 | 6 | 1,044 | 0 |
| Darwin and Linux | 2 | 4 | 48 | 24 |
| Linux-only | 3 | 2 | 36 | 72 |
| Windows-only | 10 | 2 | 120 | 240 |
| Signed Armory | 4 | 1 | 24 | 120 |
| Native OPFOR CNA | 8 | 1 | 48 | 240 |
| **Total** | **56** | — | **1,320** | **696** |

Status semantics are strict:

- A supported cell must have a recorded result. Only `PASS` satisfies the comprehensive gate; recorded `FAIL` and runtime `SKIP` results both fail it. An absent supported cell is `NOT RUN` and also fails it.
- Unsupported platform cells are synthesized as `SKIP` with the catalog reason; the harness does not send that RPC to an unsupported implant.
- The report command exits nonzero when any record is `FAIL`, any supported cell has a recorded runtime `SKIP`, or any required cell is `NOT RUN`. Catalog-generated platform skips do not fail aggregation.

## SliverRPC disposition denominator

`coverage.ComprehensiveRPCDispositions()` classifies every method in the generated `rpcpb.SliverRPC` descriptor. Unit tests fail when a generated method is missing from the registry, when a stale registry entry no longer exists, or when the executable catalog and the `COVERED` method set differ. The aggregate JSON report includes this registry in `rpc_dispositions`, and the Markdown report renders every finite command as `COVERED` or `DEFERRED`.

| Disposition | Unique gRPC methods | Meaning |
|---|---:|---|
| Server-only control plane | 102 | Does not dispatch an implant command |
| Finite implant command — covered | 41 | Has one or more executable scenarios below |
| Finite implant command — deferred | 26 | Implant-facing, but deliberately outside the executable matrix for the stated reason |
| Implant lifecycle | 3 | Kills or changes an implant connection mode |
| Tunnel or interactive | 21 | Pivot, forwarding, SOCKS, shell, or tunnel protocol rather than a finite command |
| **Generated SliverRPC total** | **193** | Exhaustive service-method registry |

The finite implant-command coverage denominator is therefore **41 covered / 67 total unique methods**. Scenario variants and platform/transport/mode cells measure depth for those 41 methods; they do not increase the method numerator.

### Deferred finite implant commands

| gRPC method | Reason |
|---|---|
| `Reconfigure` | Changes callback behavior; reconnect transition assertions are not yet isolated |
| `ProcessDump` | Captures process memory and has no bounded disposable fixture |
| `RunAs` | Requires managed credentials and platform privileges |
| `Impersonate` | Requires a managed token source and identity rollback |
| `RevToSelf` | Requires a paired isolated impersonation fixture |
| `GetSystem` | Performs privilege escalation and generated payload execution |
| `Task` | Executes arbitrary shellcode |
| `Msf` | Requires external msfvenom and executes generated shellcode |
| `MsfRemote` | Requires external msfvenom and remote-process injection |
| `ExecuteAssembly` | Executes an arbitrary managed assembly |
| `Migrate` | Injects a new implant and changes the connection lifecycle |
| `ExecuteWindows` | Windows token-specific execution fixture is not yet implemented |
| `Sideload` | Loads an arbitrary native library in a remote process |
| `SpawnDll` | Injects and executes an arbitrary DLL |
| `Screenshot` | Captures desktop contents and needs a controlled display fixture |
| `StartServiceByName` | Changes host service state |
| `StartService` | Creates and starts a host service |
| `StopService` | Changes host service state |
| `RemoveService` | Deletes host service configuration |
| `MakeToken` | Requires managed credentials and changes the active token |
| `Backdoor` | Rewrites a remote executable |
| `RegistryReadHive` | Reads a registry hive artifact and has no disposable fixture |
| `RunSSHCommand` | Requires an isolated SSH endpoint and managed credentials |
| `HijackDLL` | Rewrites and uploads a remote DLL |
| `RegisterWasmExtension` | No pinned signed cross-platform WASM fixture is selected |
| `ExecWasmExtension` | No pinned signed cross-platform WASM fixture is selected |

## Portable scenarios

Each row below is required on all six targets: `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/arm64`, `windows/amd64`, and `windows/arm64`. That is 36 required cells per scenario.

| gRPC method | Scenario |
|---|---|
| `Ping` | exact nonce round trip |
| `Pwd` | initial working directory |
| `Mkdir` | recursive nested directory |
| `Ls` | directory and wildcard metadata |
| `Cd` | relative, parent, and rejected missing path |
| `Upload` | gzip file and overwrite |
| `Upload` | tar directory recursive overwrite truncation |
| `Download` | file, byte/line limits, and recursive directory |
| `Grep` | context and recursive regex |
| `Cp` | copy exact bytes |
| `Mv` | rename within test root |
| `Chtimes` | exact access and modification time |
| `Rm` | file then recursive directory with force |
| `Ifconfig` | loopback interface and parseable addresses |
| `Netstat` | TCP IPv4 listening |
| `Netstat` | TCP IPv4 established |
| `Netstat` | UDP-only IPv4 |
| `GetEnv` | full inherited environment |
| `SetEnv` | set unique process variable |
| `GetEnv` | named variable after set |
| `UnsetEnv` | unset unique process variable |
| `Ps` | `FullInfo=false` and implant PID |
| `Ps` | `FullInfo=true` and implant PID |
| `Execute` | captured stdout stderr status and explicit environment |
| `Execute` | tracked background child |
| `ExecuteChildren` | find tracked live child |
| `Terminate` | kill only tracked test child |
| `Mount` | nonempty read-only mount inventory |
| `ListWasmExtensions` | empty initial extension inventory |

## Darwin and Linux scenarios

These rows are required on `darwin/amd64`, `darwin/arm64`, `linux/amd64`, and `linux/arm64` (24 required cells per scenario). All Windows cells are expected `SKIP` with the reason `supported only on Darwin and Linux`.

| gRPC method | Scenario |
|---|---|
| `Chmod` | recursive mode change inside test root |
| `Chown` | recursive no-op to current owner |

## Linux-only scenarios

These rows are required on `linux/amd64` and `linux/arm64` (12 required cells per scenario). All Darwin and Windows cells are expected `SKIP` with the reason `supported only on Linux`.

| gRPC method | Scenario |
|---|---|
| `MemfilesAdd` | create anonymous memfd |
| `MemfilesList` | list exact anonymous memfd |
| `MemfilesRm` | close only created anonymous memfd |

## Windows-only scenarios

These rows are required on `windows/amd64` and `windows/arm64` (12 required cells per scenario). All Darwin and Linux cells are expected `SKIP` with the reason `supported only on Windows`.

| gRPC method | Scenario |
|---|---|
| `CurrentTokenOwner` | nonempty current token identity |
| `GetPrivs` | read-only process privilege inventory |
| `Services` | read-only local service inventory |
| `ServiceDetail` | exact detail for inventoried service |
| `RegistryCreateKey` | unique HKCU subtree and child |
| `RegistryWrite` | string binary DWORD and QWORD values |
| `RegistryRead` | exact typed value round trips |
| `RegistryListSubKeys` | exact disposable child inventory |
| `RegistryListValues` | exact typed value inventory |
| `RegistryDeleteKey` | remove only disposable child and root |

## Signed Armory scenarios

These rows are required only on `windows/amd64` (six required cells per scenario). Every other target is expected `SKIP` because the pinned signed packages have no exact artifact for that OS/architecture.

| gRPC method | Scenario |
|---|---|
| `RegisterExtension` | signed COFFLoader exact target |
| `ListExtensions` | registered COFFLoader digest |
| `CallExtension` | signed `sa-env` BOF through COFFLoader |
| `CallExtension` | signed `sa-whoami` BOF through COFFLoader |

## Native OPFOR CNA scenarios

Every supported target below must run each row through the OPFOR scripting engine over all three transports (`mtls`, `wg`, and `http`) as both a session and a beacon. That is six required cells per supported target; a supported runtime `SKIP` is not accepted as coverage.

| gRPC method | Scenario | `windows/amd64` | `windows/arm64` |
|---|---|---|---|
| `CallExtension` | OPFOR Cat CNA reads isolated test file | Required: signed upstream x64 object | `SKIP`: OPFOR provider has no arm64 BOF support |
| `CallExtension` | OPFOR FirefoxDump CNA finds no host profiles | Required: signed upstream x64 object after empty-profile safety gate | `SKIP`: OPFOR provider has no arm64 BOF support |
| `CallExtension` | OPFOR FindDotnet CNA read-only process inventory | Required: pinned upstream object | `SKIP`: OperatorsKit has no arm64 object and OPFOR provider has no arm64 BOF support |
| `CallExtension` | OPFOR FindSysmon CNA read-only registry probe | Required: pinned upstream object after absent-Sysmon safety gate; invoke only `reg` | `SKIP`: OperatorsKit has no arm64 object and OPFOR provider has no arm64 BOF support |
| `CallExtension` | OPFOR callback preserves ordered typed binary channels | Required: repository-owned synthetic fixture | `SKIP`: OPFOR provider has no arm64 BOF support |
| `CallExtension` | typed BOF partial output retained on callback error | Required: repository-owned synthetic fixture | `SKIP`: OPFOR provider has no arm64 BOF support |
| `CallExtension` | malformed BOF returns bounded loader error | Required: deterministic malformed fixture followed by recovery proof | `SKIP`: OPFOR provider has no arm64 BOF support |
| `CallExtension` | finite BOF deadline returns and target recovers | Required: bounded fixture timeout followed by `Ping` | `SKIP`: OPFOR provider has no arm64 BOF support |

All Darwin and Linux cells are expected `SKIP` because these CNA scripts execute Windows COFF BOFs. The aggregate therefore adds 48 required native-OPFOR cells and 240 catalog-generated skips.

## Known request-field boundary

The `Download` scenario covers complete files, positive and negative byte/line limits, and recursive directory archives. `DownloadReq.Start` and `DownloadReq.Stop` are not exercised because the current implant download handler does not implement those fields; setting them would not test distinct behavior.
