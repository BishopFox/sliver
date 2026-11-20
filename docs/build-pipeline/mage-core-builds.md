# Core source and Mage build handoff

An installed core executor supplies source files and a Magefile bundle. The
generic workspace stage copies source bytes into a unique private directory
under the Sliver app root:

```text
slivers/<target-os>/<target-arch>/<name>/<build-id>/
  src/        copied core source
  magefiles/  the installed core's Magefile source
  inputs/     private material and build manifests
  bin/        compiler outputs and result.json
```

Copying never interprets `{{ ... }}` or any other template delimiters. A core
that requires source rendering declares an installed renderer. Sliver's
renderer runs on its staged source copy; a core without a renderer compiles
the copied files unchanged. The built-in Sliver path retains the existing
source layout, vendor dependencies, two template passes, and module/import
renaming after the copy.

The server prepares Sliver's implant mTLS, Age, and optional WireGuard
material. It writes a versioned `inputs/sliver-material.json` with owner-only
permissions. Mage receives the material file path in a private manifest;
secret values are not command-line arguments or environment variables. Only
installed and digest-pinned Magefile bundles can be executed. CoreSpec
`builder.magefile` is a relative directory within that installed executor's
bundle, not a host filesystem path.

Each Mage target accepts a manifest path as its one argument. The built-in
Sliver target uses `Build(ctx context.Context, manifestPath string) error` and
preserves the previous Go, Zig, and Garble flags and environment. Mage itself
is compiled for the server host; the manifest selects the implant target. The
server compiles a verified Magefile bundle once into a cached native runner,
then runs the target with a cancellable process and an explicit environment.
The same staged Magefile can be called with the Mage CLI. Use the
`builder.magefile` directory under the staged `magefiles/` tree:

```sh
mage -d <workspace>/magefiles/<builder.magefile> -w <workspace> build <workspace>/inputs/build.json
```

For Sliver's built-in bundle, the Magefile is directly in `magefiles/` and the
input is `inputs/go-build-....json`.

The target writes one primary output under `bin/` and publishes `bin/result.json`
after the output is complete:

```json
{"primary":"sliver.zip","displayName":"sliver.zip"}
```

The result file gives a location and display name only. The server reads the
primary output from `bin/`, checks that it is a regular file with no path
escape, validates its format and target against the frozen plan, and computes
the size and SHA-256 itself. The resulting typed bytes can be passed to the
next transform. Archives are inspected without extraction: ZIP and AR inputs
have bounded member counts and sizes, path and link checks, and ZIP CRC checks.
Installed core source archives are extracted into an empty private workspace
with bounded ZIP, TAR, and TAR.GZ readers that reject traversal, links, and
duplicate names.

The RPC executor validates the core output, then applies each resolved local
transform in order. Sliver shellcode builds expose their Windows PIE, Linux
shared object, or macOS shared library as the core step; Donut, Malasada, and
Beignet each run as a separate transform. Shellcode encoders, PE metadata
spoofing, and stage framing, compression, and encryption use the same step
contract. A failed step is recorded with the prior steps' observed hashes.

Intermediate bytes live only in the executing process. Malasada's temporary
shared object and the PE metadata scratch files are private and removed after
their steps. The Sliver core workspace and cryptographic material are removed
after the final artifact is saved. Each completed step records its actual
artifact type, filename, size, and SHA-256; only the final bytes are retained
in the build record and legacy Sliver storage. `regenerate` reads the recorded
final stream. Stage keys and PE donor/configuration bytes enter through
per-build inputs; the plan stores typed references, not their contents.

Legacy direct build helpers continue to return a file path. The RPC build
path consumes and removes its private workspace after saving the output.
