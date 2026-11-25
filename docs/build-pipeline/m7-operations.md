# Build pipeline operator and integration guide

This guide covers the installed `sliver-v1` build system. The server owns the
catalog, plan, final artifact, and build record. A core integration supplies
trusted source and a compiler; a transform consumes one typed artifact and
returns another. YAML describes installed capabilities. It cannot upload
source code, select an arbitrary command, or install an executor.

## Author an immutable spec

Start with the [CoreSpec and TransformSpec examples](examples/) and the
[versioned schemas](schemas/). A spec's `metadata.name`, semantic `version`,
and SHA-256 `digest` identify one immutable definition. Calculate the digest
from the restricted parsed YAML with only `metadata.digest` removed, then
serialize it using RFC 8785 and hash those bytes. The [contract guide](README.md)
gives the exact rule. A changed definition needs a new version; importing
different content under an existing name and version fails. Current catalog
import rejects floating-point YAML values.

For a core, declare:

| Field | Meaning |
| --- | --- |
| `runtime.protocol` | `sliver-v1`, the existing Sliver server wire contract. |
| `runtime.commandSource` | `installed-catalog` only for the installed Sliver command catalog, or `declared-subset` with the request messages the core actually handles. |
| `executor.id` | An integration compiled into the server and, for remote builds, into the worker. |
| `source` and `builder` | `copy` or `none` source mode and an installed Magefile directory and target. An optional renderer must also be installed. |
| `capabilities` | Only proven target, mode, C2 protocol, output artifact, operator format, and toolchain tuples. |
| `options` | Typed, named inputs to the installed executor; secret and donor inputs use per-build references. |

For a transform, declare its exact input and output artifact types and the
targets and options it supports. A matching kind alone is insufficient: the
planner checks container, target, build mode, entry symbols, and option types.
For example, a Windows service is `service-executable/pe`; an operator
`archive` is a `c-archive-bundle/zip`. Do not advertise the product of every
target, mode, protocol, and output as a Cartesian set.

## Install and inspect an executor

An integration implements `buildcore.Adapter` and registers a stable executor
ID with `buildcore.RegisterDefault`. An installed core may implement
`buildcore.CoreStepAdapter` for the ordered step pipeline. `MageCoreAdapter`
provides a reusable copy, optional render, private input, Mage execution,
and typed artifact handoff. The installed binding must match the CoreSpec's
source and builder declarations. The Magefile path is a relative directory
inside the installed bundle; the operator cannot choose a host path. Its
target reads a private JSON manifest and writes one primary file plus
`bin/result.json`. The server checks regular-file paths, type, size, format,
target, and SHA-256 before the next step. See the [Mage handoff](mage-core-builds.md).

Place a `CoreSpec` YAML file in `$SLIVER_ROOT_DIR/catalog/cores/` or a
`TransformSpec` YAML file in `$SLIVER_ROOT_DIR/catalog/transforms/`, then restart
the server. The catalog rejects a spec in the wrong directory, an unknown
executor ID, or a name/version conflict. Built-in specs
ship in `server/buildcatalog/builtin/`; administrator and remote snapshots
persist in `<SLIVER_ROOT_DIR>/build-catalog/snapshots.json`. The console shows
definition and availability separately:

```text
cores list --target linux/amd64 --availability all
cores info builtin/sliver --yaml
transforms list --availability available
transforms info builtin/malasada
```

The `cores` and `transforms` list tables omit digests by default. Add
`--show-digests` to include the `Digest` column; `info` continues to show the
selected spec's digest.

An offline remote spec remains visible. Availability on the catalog means an
executor/provider is present; planning still checks the requested compiler,
target, and toolchain tuple.

For a remote provider, install the same executor and its Mage/source bundle on
the worker. The worker advertises the exact spec YAML and digest, then sends
target-specific toolchain and capacity status. The server never sends an
operator-provided Magefile or a source bundle to the worker. The server
assigns one frozen step; the worker returns a typed artifact that is checked
again before another step starts. See [remote execution](m5-remote-execution.md).

## Author and run a profile

A `BuildProfile` fixes the target, session or beacon mode, C2 endpoints, exact
core and ordered transform references, final artifact, and automatic or
pinned provider policy. It contains no generated private keys. Importing a
changed profile requires a higher `metadata.revision`; a build stores its
resolved YAML, revision, step graph, spec digests, provider decisions, and
final artifact separately.

The [minimal core profile example](examples/profile-minimal-sliver-v1.yaml)
selects the installed alternate core for a Linux/amd64 session. Replace its
example mTLS address with a reachable listener before generating. This core
accepts one mTLS endpoint, sequential connection order, a positive reconnect
interval, the default HTTP profile, and zero polling timeout and connection
error cap; it rejects other C2 settings during execution.

```text
profiles validate profile.yaml
profiles import profile.yaml
profiles info my-profile
profiles export my-profile --save exported.yaml
profiles generate my-profile --save ./payloads/
```

`profiles generate my-profile --builder builder-a` requests that specific
compatible remote builder. A profile can instead set `provider.policy:
pinned` and `provider.builder: builder-a`. Automatic policy uses an available
server provider first. `--external-builder` requests remote-only selection;
among compatible online remote builders, ranking uses available memory
rounded to decimal MB, then effective logical CPU cores, then stable builder
name. The status must be fresh and the builder must have enough unreserved
capacity. The `builders` command shows
CPU, available MB, report age, availability, exact spec support, and compiler
targets. Use `generate --plan` or `generate beacon --plan` to preview a legacy
command's resolved steps without compiling; authored profiles are validated
at `profiles validate` and revalidated before execution.

A profile may declare `perBuildInputs`. Provide every required slot once with
the matching kind; secret and input bytes are passed for that build and are
not written into the profile or frozen plan:

```text
profiles generate staged-profile --input-artifact donor-pe=./donor.exe \
  --secret-file aes-key=./key.bin --secret-file aes-iv=./iv.bin
```

Legacy `ImplantConfig` profiles are backfilled without deleting the database
row. `profiles info` shows a migration diagnostic for a legacy configuration
that cannot be represented faithfully. Such profiles retain the old client
generation path. An authored YAML profile under a migrated name is
authoritative to new clients; older clients can still see the retained
legacy row. `profiles rm NAME` removes the definition and its legacy profile
row while retaining completed builds, snapshots, and final bytes.
`implants rm BUILD` explicitly removes a completed build. See the
[profile guide](m6-yaml-profiles.md) for stage and consumer artifact rules.

## Diagnose an unavailable or failed build

1. Run `cores info NAME --yaml` or `transforms info NAME --yaml` to check the
   exact digest, capability tuple, and providers. A name/version conflict
   needs a new immutable version. An unknown executor needs a trusted
   installation on the provider, not another YAML edit.
2. Run `builders` and check online status, report age, available MB, CPU,
   spec digest, and compiler targets. A disconnected or stale builder cannot
   receive a new assignment. A pin still needs the matching tuple and
   capacity.
3. Use `generate --plan` for a legacy command, or `profiles validate` for
   authored YAML. An unsupported target, unavailable toolchain, ambiguous
   route, invalid option, or transform type mismatch fails before compile.
4. Inspect the build record after execution. It freezes each step's provider,
   selection reason, state, artifact type, size, SHA-256, and failure detail.
   `GetBuildArtifact` returns only a checksum-checked final artifact;
   `regenerate` reads the stored final bytes. A completed build is unchanged
   by later spec or profile edits.

The [acceptance matrix](m7-acceptance.md) separates static checks, compiled
artifacts, two-process remote builds, and live implant connections.

For a local Sliver `generate` or `generate beacon` build, pass `--cleanup
false` to keep the rendered source, Magefile, per-build inputs, and compiler
output in the server's private workspace. The command prints the workspace
path. The retained input and source files contain cryptographic material;
see [source and Mage handoff](mage-core-builds.md) for the layout. The default
`--cleanup true` removes the workspace after saving the final artifact.

`implants` shows the core name and version from each build's frozen record.
Builds created before records were introduced show `Unknown (legacy)`.
Core identity no longer comes from the former template configuration field.
