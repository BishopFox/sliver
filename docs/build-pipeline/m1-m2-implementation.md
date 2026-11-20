# M1 and M2 implementation guide

The server ships an immutable `builtin/sliver` core definition and separate
Donut, Beignet, Malasada, Shikata Ga Nai, XOR, and dynamic XOR transform
definitions. The catalog tracks a spec's identity and sources separately from
its live providers. A disconnected builder's last advertised YAML remains
visible with an offline provider.

## Inspect the catalog

From a connected client:

```text
cores list --target windows/amd64 --availability available
cores info builtin/sliver --yaml
transforms list --target linux/amd64
transforms info builtin/malasada
```

`list` can filter by target and `all`, `available`, or `unavailable` status.
The `cores` and `transforms` tables omit digests by default; add
`--show-digests` to either command or its `list` subcommand to include the
`Digest` column.
`info` shows capability tuples, toolchain requirements, source, digest, and
provider status. Names and targets complete from the server catalog. The RPCs
`ListBuildSpecs` and `GetBuildSpec` expose the same read-only inventory.
Availability means an installed executor has an online provider; whether a
specific compiler tuple is currently usable is checked during planning.

To add an administrator spec, place a validated `.yaml` or `.yml` file in
`<server app root>/build-catalog/admin/` and restart the server. The catalog
loads these files on first access after startup and writes its snapshot index to
`<server app root>/build-catalog/snapshots.json`. This release only accepts
installed executor IDs; YAML cannot run arbitrary commands. Imports reject
floating-point values until the canonical JSON number serializer is extended.

Remote builders can send a complete set of spec YAML documents in
`Builder.SpecYAML`. Registration validates the set atomically. The server
persists accepted definitions and updates provider status on disconnect.
Older builders that do not send `SpecYAML` retain their legacy build protocol
without creating a catalog provider claim.

## Preview and execute a route

`generate --plan` calls `PlanBuild` and returns a route without compiling or
allocating a WireGuard peer IP. For example:

```text
generate --mtls 192.0.2.10:8888 --os windows --arch amd64 --format shellcode --plan
```

The preview resolves `builtin/sliver` to a Windows PIE core output followed by
`builtin/donut`, and shows each exact spec digest, selected provider, toolchain,
and final artifact type. A Linux amd64 shellcode preview instead resolves a
shared-library core output followed by `builtin/malasada`. Use
`generate beacon ... --plan` for beacon mode. Unknown formats, unsupported
target/mode/protocol combinations, missing toolchains, and ambiguous routes
fail before a compiler starts.

`ExecuteBuild` accepts a neutral `BuildRequest` and selects the registered
core adapter named by the resolved spec's executor ID. The installed Sliver
adapter executes direct outputs and the existing platform shellcode converters.
Other valid transform routes return a specific unsupported-execution error
until the M4 step executor is installed. Administrator and remote spec
definitions can be inspected and planned; remote execution arrives in M5.
Remote step execution is reserved for M5. The existing `Generate` RPC and
operator commands remain compatibility entry points. A compiler failure from
`ExecuteBuild` returns a persisted failed record; the monolithic compiler
cannot identify its exact failing internal step, which the record states.

Completed builds store a frozen neutral request, resolved steps, spec digests,
provider and toolchain information, step states, final artifact type, size,
SHA-256, and exact returned bytes under `<server app root>/build-records/`.
`GetBuildRecord` and `GetBuildArtifact` expose the record and checksum-checked
final file. `regenerate` reads these stored bytes for new builds, so a later
catalog or profile edit cannot change the output. Existing builds without a
record still use the database-backed artifact. Legacy PE metadata spoofing
creates a numbered final-byte revision while keeping the plan unchanged.

The legacy shellcode encoder still runs after generation via its existing RPC;
the completed record covers the server-returned bytes before that optional
encoding step. M4 moves encoding and other post-processing into the recorded
server pipeline. The compatibility compiler currently reports completed route
steps after its monolithic call finishes, so only the final step has an
intermediate-independent byte hash.
