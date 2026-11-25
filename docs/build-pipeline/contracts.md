# Build pipeline contracts (M0)

This document fixes the meanings of artifacts, the initial runtime protocol, and
remote capacity reports before the catalog and planner are implemented. The
versioned YAML shapes and concrete examples live in [`schemas/`](schemas/) and
[`examples/`](examples/). These are target contracts, not claims that the current
generation RPCs already enforce them.

## Artifact identity

An artifact has a `kind`, `container`, target OS and architecture, byte size,
SHA-256 of its complete stored bytes, and, when applicable, a Go build mode and
exported `entrySymbols`. The kind describes what a consumer can do with the
bytes. The container describes their binary or package format. Neither is
inferred from a filename extension. A transform input must match both fields,
the target, and any required entry symbols. Architecture-specific shellcode
must also match the architecture exactly.

| Kind | Container | Meaning |
| --- | --- | --- |
| `executable` | `pe`, `elf`, or `macho` | Direct process executable. |
| `service-executable` | `pe` | Windows executable built with service registration and startup semantics; it is not interchangeable with an ordinary PE executable. |
| `shared-library` | `pe`, `elf`, or `macho` | Dynamic library with its declared exports and load behavior. |
| `go-archive` | `ar` | Raw Go `-buildmode=archive` output; it is not a C archive bundle. |
| `c-archive` | `ar` | Raw Go `-buildmode=c-archive` library, with a separately tracked generated header. |
| `c-archive-bundle` | `zip` | Distribution package containing the C archive and generated headers. The checksum covers the ZIP bytes, not just its `.a` member. |
| `go-plugin` | `elf` or `macho` | Go `-buildmode=plugin` output, distinct from a C ABI shared library. |
| `go-shared-library` | `elf` or `macho` | Go `-buildmode=shared` output, distinct from `c-shared`; the target still needs an executor that supports this mode. |
| `shellcode` | `raw` | Unpackaged machine code for one target architecture. |
| `stage` | `raw` | A staged payload after requested compression, encryption, or size framing; its bytes must not be treated as the unprocessed executable or shellcode. |

The Go build mode is a separate dimension. A spec may describe `archive`,
`c-archive`, `c-shared`, `default`, `exe`, `pie`, `plugin`, and `shared`. A mode
does not by itself grant an output capability: the spec must list a valid
target/mode/protocol/output tuple, and the installed executor must implement
it. In particular, `default` can produce different kinds depending on the Go
package. The built-in Sliver core should advertise only its demonstrated
tuples. Current console `--format archive` means the **`c-archive-bundle` ZIP**
built with Go `c-archive`, not Go `archive`. Current `--format service` means
`service-executable` on Windows and uses the executable compilation route.

For the initial shellcode paths, the typed edges are:

| Target | Core artifact | Transform | Final artifact |
| --- | --- | --- | --- |
| Windows 386/amd64 | PE `executable`, Go `pie` | Donut | `shellcode/raw` for the same architecture |
| Darwin arm64 | Mach-O `shared-library`, Go `c-shared` | Beignet | `shellcode/raw` for arm64 |
| Linux amd64/arm64 | ELF `shared-library`, Go `c-shared` | Malasada | `shellcode/raw` for the same architecture |

Shellcode encoders are subsequent `shellcode/raw` to `shellcode/raw` edges.
Each transform declares the architectures it accepts. The planner must not
coerce a PE executable to an ELF library, a raw archive to a ZIP bundle, or
shellcode for one architecture to another. At execution, the server checks
the file signature where the format has one, validates the typed output
descriptor, and computes the complete-byte checksum before accepting the
output; the recorded final checksum must match the bytes it later returns.

## `sliver-v1` runtime

`sliver-v1` identifies the existing implant/server wire contract, not the
Sliver source tree or the Go compiler. A core claiming it must use the
`protobuf/sliverpb/sliver.proto` messages and the stable numeric envelope
types in `protobuf/sliverpb/constants.go`, including the framing, security
handshake, and request/response behavior of each C2 transport it advertises.
It must not substitute its own command schema or protocol adapter. The
`sliverpb.Envelope` carries the numeric message type, request ID, payload,
and `UnknownMessageType` response indicator.

A session core registers with `MsgRegister` and `sliverpb.Register`, then
correlates replies with the incoming envelope ID. A beacon core registers
with `MsgBeaconRegister` and exchanges `MsgBeaconTasks` lists, preserving task
IDs in responses. A core may advertise either mode or both. It must implement
the transport lifecycle, registration, and response handling necessary for
each mode/transport tuple it advertises. The protocol's `Register.Capabilities`
bits remain feature flags for specific server behaviors; they are not a
complete command inventory.

The CoreSpec chooses one command source. With
`runtime.commandSource: installed-catalog`, its `runtime.commandCatalog` ID
selects a trusted inventory supplied by the installed executor. The built-in
Sliver catalog resolves that inventory for each advertised OS, architecture,
and session or beacon mode from its actual handlers. It does **not** claim
that every `sliverpb` request exists on every target. YAML cannot add commands
to an installed catalog. With `runtime.commandSource: declared-subset`, the
nonempty `runtime.commands` list names the **request message types** the core
actually handles, using stable `Msg...` names from
`protobuf/sliverpb/constants.go` (for example `MsgPing` and `MsgLsReq`). An
importer resolves every name against an installed allowlist and rejects
unknown names, response-only types, and duplicates. A declared subset applies
to every capability tuple in that spec; a core with target-specific command
differences needs separate specs until per-tuple subset declarations exist.

The planner and console can use the resolved request inventory to disable
unsupported operations. For each supported request type, the existing
`sliver-v1` mapping fixes the protobuf request and response message types, the
envelope ID correlation, and the `commonpb.Response` error semantics where
that message defines them; YAML cannot redefine those mappings. If an older
client still sends an unsupported request, the core returns an envelope with
the same ID and `UnknownMessageType: true`, matching the current server error
path. A core must not claim a command, feature bit, C2 transport, or output
unless it implements it. For example, an alternate core that only handles
registration, `MsgPing`, and `MsgLsReq` can join the same server but does not
claim shell, BOF, or tunnel support.

## Remote capacity and deterministic selection

The builder reports capacity at registration and every **10 seconds** while
connected. The server records its own receipt time and considers a report
fresh when its age is at most **30 seconds** at the instant of assignment.
Clock comparisons use the server's monotonic time; a builder-supplied clock
cannot extend freshness. A disconnected builder, missing report, report with
invalid measurements, or report older than 30 seconds is unavailable for new
assignments, including a pinned assignment. A report is a builder claim,
validated for shape and bounds by the server, not proof of physical capacity.

`availableMemoryBytes` is a non-negative integer estimate of memory currently
available to the builder process. On a constrained host, the builder uses the
smaller of host available memory and the applicable process/container limit
minus its current usage, clamped to zero. It must not report total installed
RAM as available memory. An unavailable or untrustworthy measurement makes
the builder ineligible until a valid report arrives. The server displays and
ranks memory as integer **decimal MB**, where 1 MB is 1,000,000 bytes and
rounding is nearest with a half-MB rounded up:
`memoryMB = floor((availableMemoryBytes + 500000) / 1000000)` (implemented
without integer overflow). The original byte count is retained in the report
and build record.

`effectiveLogicalCores` is a positive integer count of logical CPUs the
builder can use. It is bounded by the host logical CPU count, process
affinity/cpuset, an applicable CPU quota, and the process concurrency limit.
For a fractional quota, take the floor and use one core for a positive quota
below one core; a zero usable quota makes the report invalid. A missing limit
is ignored, not treated as zero. The builder reports the effective count,
and the server rejects zero or impossible values. The reported value ranks
builders but does not authorize execution of an incompatible toolchain.

Before ranking, the server filters connected builders by fresh capacity,
exact spec digest, target, mode, protocol, input/output artifact types,
toolchain availability, and operator authorization. It then ranks by the
**remaining unreserved available memory rounded to decimal MB**, descending;
effective logical cores, descending; and stable builder name, ascending by
UTF-8 byte order. Stable names are unique among connected builders and do not
change during a connection. A pin bypasses ranking but must pass the same
filters and capacity check. Equal values after MB rounding use CPU and then
name even if the original byte counts differ.

In M5, operator authorization uses the server's existing shared builder pool:
any authenticated operator may request a build on any eligible connected
builder. There is no per-builder requester ACL. The certificate identity that
registered a builder, together with its current private session credential,
authorizes that builder's status updates and assigned-step fetch and upload;
it does not restrict which authenticated operator may select the builder.

Each core or transform capability tuple may declare a positive
`resources.estimatedPeakMemoryBytes`, a conservative per-step estimate rather
than the smallest amount of free memory on which a job might start. The
scheduler reserves the larger of that value and the server-installed
executor's own per-step estimate. If neither has a known estimate, it
reserves the builder's entire reported available memory for that step, a
conservative single-step fallback. Under one server-side atomic lock or
transaction, a candidate must have
`reportedAvailableBytes - activeReservationsBytes >= reservationBytes`;
selection and reservation happen together. Active reservations remain charged
against every later report until their steps finish or fail. A completion
releases its reservation, but stale reports still cannot qualify for a new
assignment. Reservations are not persisted as new spec content. The build
record freezes the chosen builder, report receipt time and measurements,
reservation, and decision order. This makes concurrent requests see the same
committed capacity ledger rather than repeatedly selecting an uncommitted
high-memory builder.

For example, a builder with 4,500,000,000 reported bytes and eight cores ties
on rounded MB with one reporting 4,499,600,000 bytes and twelve cores; the
twelve-core builder wins. If that builder already has a 1,000,000,000-byte
reservation, its remaining 3,499,600,000 bytes round to 3,500 MB, so the
other builder wins. If memory and CPU also tie, the lexicographically smaller
stable name wins.

At dispatch and result upload, the server rechecks the authenticated builder
identity, assigned step, exact spec digest, target, artifact kind/container,
size, and SHA-256 before advancing the plan. YAML names an installed executor
and capabilities; it never supplies executable commands or source bundles.
