# Build pipeline contract examples (M0)

The schemas in [`schemas/`](schemas/) define `CoreSpec`, `TransformSpec`, and
`BuildProfile` version `sliver.bishopfox.com/v1alpha1`. They are JSON Schema
Draft 2020-12 files for a restricted YAML input: one mapping document, string
mapping keys, JSON-compatible scalar values, and no duplicate keys, aliases,
merge keys, or custom tags. Quote ambiguous scalars such as architecture
`'386'`. Numeric values must be finite I-JSON numbers; integers outside
`-9007199254740991` through `9007199254740991` are rejected before import.
Every defined object rejects unknown fields; the intentionally open
option **values** are checked against the referenced spec's typed option
declarations. Profile revisions are mutable editing versions; spec versions and
digests identify immutable definitions.

## Immutable spec digest

For a `CoreSpec` or `TransformSpec`, parse the restricted YAML and validate the
schema. Remove only `metadata.digest` from that parsed document. Do not inject
schema defaults, source, provider, or live availability fields. Serialize the
remaining JSON data model with [RFC 8785 JSON Canonicalization
Scheme](https://www.rfc-editor.org/rfc/rfc8785), hash those UTF-8 bytes with
SHA-256, and store the lowercase hex value as `sha256:<64 hex digits>`. The
example digest checker covers the examples' integer/string/boolean data subset;
production import must implement full RFC 8785 for finite non-integer numbers.
An imported `(name, version)` cannot be replaced by different content. A
content change needs a new version. A disconnected provider does not change a
spec digest.

For example, [`core-sliver.yaml`](examples/core-sliver.yaml) has digest
`sha256:91dec1eb4394dbc34ab9e0594d7297424cff0d5523e6ab3b3f6d4ba4f90a1af0`.
The value is calculated from the parsed definition, not from YAML formatting.

## Examples and capability meaning

[`core-sliver.yaml`](examples/core-sliver.yaml) lists the current Sliver core's
target/mode/protocol/output groups. Windows service output is a PE executable;
the operator's `archive` output is a ZIP of a Go `c-archive` and headers.
Windows shellcode takes an intermediate PIE executable through Donut. macOS
arm64 and Linux amd64/arm64 shellcode take a `c-shared` library through Beignet
or Malasada. Encoder examples cover the implemented architecture map: Shikata
Ga Nai on `386`/`amd64`, XOR and dynamic XOR on `amd64`/`arm64`.

The Sliver example declares supported *definitions*, including conditional
CGo outputs. A provider is available for one of those tuples only after its
installed toolchain, compatible compiler, and live capacity are checked. The
example itself is not evidence that each target compiled on every host.
Transport rules are separate: HTTP and HTTPS are distinct, named pipes are
Windows-only, and named pipes and TCP pivot are session-only.

[`core-go-modes-example.yaml`](examples/core-go-modes-example.yaml) demonstrates
all eight Go build modes: `archive`, `c-archive`, `c-shared`, `default`, `exe`,
`pie`, `plugin`, and `shared`. [`core-alternate-example.yaml`](examples/core-alternate-example.yaml)
demonstrates a non-Go core with an explicit subset of `sliver-v1` commands, a
copy-only source policy, and its own installed Magefile target. Its executable
has no `goBuildMode`. Both `example/*` specs use hypothetical
executor IDs and **must be unavailable** until matching trusted integrations
are installed. They do not add those modes to `builtin/sliver`.

The profile examples cover direct executable, service, shared-library, and C
archive bundle output plus Windows, macOS, and Linux shellcode conversions.
[`profile-windows-shellcode.yaml`](examples/profile-windows-shellcode.yaml), for
example, resolves a Windows `pie` executable to Donut shellcode, then applies
Shikata Ga Nai. [`profile-linux-shellcode.yaml`](examples/profile-linux-shellcode.yaml)
uses a documentation-only WireGuard IP. These are contract fixtures, not live
listeners or working C2 endpoints.

The [`profile-windows-stage.yaml`](examples/profile-windows-stage.yaml) fixture
shows donor PE metadata copy, optional four-byte little-endian size prefix,
compression, then AES encryption as ordered transforms. The stage RPC and
installed M4 wrappers apply those operations in that order. `stage/raw` is
the transformed byte stream; a stage listener is a separate consumer. The
`example/*` executor IDs in the fixture remain hypothetical; use installed
`builtin/*` references for a runnable profile.

The profile declares a donor PE as a per-build input artifact slot and AES key
and IV as per-build secret slots. It stores references only. The existing
`--spoof-metadata` compatibility command still uses a separate
`GenerateSpoofMetadata` RPC, which records a new final artifact revision.
Authored YAML profiles receive donor bytes and secret values through per-build
inputs. Migrated `profiles stage` commands retain per-invocation encryption
flags; authored YAML profiles declare those transforms in the document. The
schemas allow neither a command string nor a source bundle to become an
executor. Existing empty legacy connection strategy maps to `sequential`
when migrating profiles.

## Validation

For the implemented M1/M2 catalog, planner, RPCs, and console commands, see
the [implementation guide](m1-m2-implementation.md).

With Python, PyYAML, and `jsonschema` installed, run:

```sh
python3 docs/build-pipeline/validate_examples.py
```

The validator checks restricted YAML parsing, each schema, real spec digests,
duplicate option names, option defaults and assignments, conflicting
`(name, version)` definitions, spec references, and each profile's artifact
edges, including per-build input slot bindings. Structural schemas also reject
nested unknown fields and invalid artifact kind/container/build-mode pairs.
The M1 catalog import verifies installed executor IDs, the allowed runtime
commands, capability tuples, and provider claims. The M2 planner validates
route selection, transform edges, requested final format, and installed
toolchains before execution. The [M5 implementation guide](m5-remote-execution.md)
describes remote capacity selection and assigned step execution. The
[M6 profile guide](m6-yaml-profiles.md) describes canonical YAML profiles,
legacy migration, per-build inputs, and consumer artifact rules.
