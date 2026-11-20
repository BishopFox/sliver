# M5 remote step execution

M5 lets the server assign a resolved core or transform step to a connected
builder. The server remains responsible for planning, Sliver build identity and
cryptographic material, accepting each typed artifact, running any following
steps, and storing the final bytes. The installed remote core integration in
this milestone is `builtin/sliver`; another core needs the prepared material
contract planned for M7.

## Register and inspect builders

A builder advertises the exact name, version, and SHA-256 digest of each
installed spec, plus the target, mode, protocol, input/output artifact, and
toolchain tuples it can run. The server compares those claims with the
advertised immutable YAML. A status update can change capacity and toolchain
availability, but cannot add or replace a spec or capability. The builder
sends available memory, effective logical CPU cores, and target-specific
toolchain status at registration and every 10 seconds. The server timestamps
reports when received and measures freshness with a monotonic clock; an
automatic selection requires a report no older than 30 seconds and an open
registration stream.

`builders` displays each builder's CPU count, available decimal MB, report
age, availability, exact spec references, and compiler targets. Offline and
stale builders remain visible for diagnosis but receive no new assigned
steps. `cores` and `transforms` show the immutable spec definitions and
provider status.

The worker measures Linux `MemAvailable` and caps it by the tightest remaining
memory allowance at its process cgroup or an applicable ancestor; macOS uses
free, inactive, and speculative `vm_stat` pages;
Windows uses available physical memory. A failed measurement reports zero,
which prevents an assignment based on invented capacity. Effective CPU cores
are the smallest known limit from the process-visible CPU count,
`GOMAXPROCS`, Linux affinity/cpuset, Windows process affinity, and applicable
Linux cgroup CPU quota.
A positive fractional quota rounds down to at least one core; an explicit
zero quota makes the builder unavailable.

## Selection and reservations

The planner first checks the exact resolved spec digest and capability tuple,
including target, Sliver mode and protocols, input and output artifact types,
and live toolchain availability. The server ranks compatible remote builders
by available memory rounded to the nearest decimal MB, then effective CPU
cores, then ascending stable name. A builder pin bypasses that ordering but
still has to pass compatibility, freshness, and capacity checks.

All authenticated operators share the connected builder pool, including for
pinned requests. M5 has no per-builder requester ACL. The operator certificate
used to register a builder controls its worker RPCs; it does not limit who may
request a build on that builder.

At dispatch, the scheduler atomically reserves the larger of the spec's
estimated peak memory and the installed executor's estimate. If neither is
known, it reserves the builder exclusively for that step. It ranks using
reported memory less existing reservations, so concurrent assignments cannot
both spend the same capacity. It releases the reservation when the step
finishes, fails, times out, or is cancelled. A later status report does not
free an active exclusive reservation. A build record stores the selected
builder, the report time, reported and previously reserved bytes, rounded
available MB, CPU count, required peak bytes, selection rule, and ordered
eligible candidates with the step.

## Assigned step exchange

Registration establishes a random 32-byte session credential bound to the
builder's authenticated operator TLS identity. `Builders` never returns the
credential. For an assigned step, the server sends a `build-step-assigned`
notification containing only the builder name and an unpredictable assignment
ID. Assigned notifications use a bounded queue for each builder. A full queue
rejects dispatch promptly, so a reserved step is never silently dropped. The
named builder fetches the request with `BuilderGetStep` and returns a
typed output through `BuilderSubmitStep`. Both RPCs require the operator
certificate and the current builder session credential. A disconnected
registration cancels its pending assignments. A timed-out worker is
quarantined: its stream closes, outstanding work is cancelled, and the worker
reconnects with a fresh status report after its current work stops.

The fetched request carries the frozen step, exact spec digest, target,
toolchain and options. For a Sliver core, the server prepares the build keys,
config, and HTTP C2 material before dispatch. For a transform, it sends the
accepted prior artifact and only the per-build inputs referenced by that
step's options. The worker checks its local installed spec and capability
before execution. The server accepts an upload only from its assigned builder
and only when the spec, target, artifact type, and reported toolchain identity
agree with the assignment. The server checks size bounds, format, target, and
the SHA-256 supplied with the bytes. ZIP and archive outputs
go through the same bounded artifact validation as local builds. The next
step cannot start until the uploaded bytes pass that validation.

## Commands and compatibility

For configurations represented by the neutral build request,
`generate -E` and `generate beacon -E` use assigned steps and download the
server's recorded final artifact. `--builder NAME` pins the builder; the
`--plan` preview shows the selected provider. `--remote-step` explicitly
requires this path and reports an unsupported legacy setting instead of
changing its meaning. The client checks the final artifact's size and
SHA-256 against the build record before saving it.

Some existing settings are not yet neutral core options, including custom
beacon timing, WireGuard peer settings, a custom HTTP C2 profile, selected
connection/retry settings, traffic encoder assets, and legacy shellcode
configuration. `-E` keeps the existing external build compatibility path for
those requests and prints that choice. `profiles generate --builder NAME`
pins that legacy profile build; M6 will move profiles to full server-side YAML
pipelines. The compatibility `GenerateExternal` RPCs remain until the M7
replacement and migration gates pass.

## Verification scope

`./go-tests.sh --offline-build-pipeline` covers scheduler ordering and ties,
pinning, stale and disconnected builders, concurrent reservations, worker
capability and typed input checks, registration and token privacy, rejected
foreign or mismatched uploads, command routing, and builder inventory. Its
`TestTwoBuilderAssignedStepStreams` opens two authenticated registration
streams, dispatches concurrent typed steps, verifies that the second
reservation changes the preferred builder, accepts both uploads, checks the
recorded candidate order, and excludes disconnected builders. That stream
test uses simulated worker uploads. `TestTwoBuilderLiveWorkersExecuteAssignedTransforms`
starts two real builder workers in-process. Each fetches an assignment,
executes XOR, and submits a validated typed output; an exclusive reservation
on the first worker routes the second assignment to the other. A two-process
builder/server core compile and remote output retrieval also passed in an
isolated private-network Docker exercise on the final source: the server
assigned `builtin/sliver` to a separate builder process, then the client
verified the Linux ELF magic, 15,503,587-byte size, and SHA-256
`71cb76bfb62255378521a71d6f704f716c70b64b9c53e18f3f3a8fdfc10ce0d5`
against the successful build record. Remote validation in CI remains
unverified.
