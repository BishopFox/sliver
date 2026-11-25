# M6 server-side build profiles

The server stores each validated `BuildProfile` as canonical YAML in its
`<SLIVER_ROOT_DIR>/build-profiles/profiles.json` index. The index is written
atomically with owner-only permissions. A profile names an immutable core spec
and ordered transform specs, with the desired final artifact type. Generated
keys and per-build input bytes are supplied separately. The server resolves the
profile against its current catalog and freezes the resolved plan and spec
digests in a new build record before execution.

## Author and run a profile

Start with an example under [`examples/`](examples/) and replace its C2
endpoint and target settings. The example endpoints are documentation
addresses. A profile revision increases when its definition changes.

```text
profiles validate profile.yaml
profiles import profile.yaml
profiles info my-profile
profiles export my-profile --save exported.yaml
profiles generate my-profile --save ./payloads/
```

`validate` checks the YAML and the resolved pipeline without saving it.
`import` stores the canonical document. Importing the same document again is
idempotent; changing it requires a higher `metadata.revision`. The output of
`export` of an authored profile can be imported again as the same revision.
`profiles new` and `profiles new beacon` continue to accept flags or the
interactive form. The server saves their legacy configuration and attempts a
canonical YAML mirror.
The command reports the YAML revision on success or a migration diagnostic
when the legacy settings cannot be represented faithfully.
`profiles new` and `profiles new beacon` reject a name already used by an
authored YAML profile; export and edit that document, then import its next
revision.

For a profile declaring `perBuildInputs`, pass each required input exactly
once. Input names and kinds must match the declaration:

```text
profiles generate staged-profile --input-artifact donor-pe=./donor.exe \
  --secret-file aes-key=./key.bin --secret-file aes-iv=./iv.bin
```

The client bounds input-file reads; the server checks the declaration again.
Input bytes are used during the build and are excluded from the stored plan
and profile. For authored YAML, the server revalidates `--name`, `--builder`,
`--external-builder`, and `--shellcode-encoder` generation overrides against
the resolved pipeline. A migrated profile that cannot be represented as YAML
builds locally through the resolved `Generate` pipeline; requesting a remote
builder for that profile returns a compatibility error. Remote profile builds
require an authored YAML profile.
The completed build's file, type, size, and SHA-256 must agree with its frozen
record before the client saves it.

## Legacy profiles and deletion

Listing or accessing profiles backfills existing `ImplantConfig` profiles
without deleting their database rows. Backfill is idempotent and does not
replace an authored YAML document. Migrated profiles retain a link to their
original complete configuration and execute locally through the resolved
`Generate` pipeline, so settings that the neutral profile request cannot yet
express keep their previous meaning. New authored YAML profiles execute the
ordered neutral pipeline.

Importing a higher revision of authored YAML under a migrated profile name
changes what the new profile API builds. Older clients still see the retained
legacy database configuration under that name and build its previous output.
Use a new name when operators on older and newer clients must see the same
definition during migration.

`profiles rm NAME` removes the definition and records a tombstone so a stale
legacy database row cannot recreate it. The legacy profile row is removed;
completed builds retain their own configurations, frozen plans, and artifacts
for retrieval and regeneration. `implants rm BUILD` explicitly removes a
completed build. Editing or deleting a profile does not rewrite an existing
build record.

## Profile consumers

Consumers check the profile's declared final artifact before building and
check the completed record's final artifact again before using its bytes.
These consumer commands do not take `perBuildInputs` flags; a profile with
required input slots cannot be used through them until that command supplies
the inputs.

| Consumer | Accepted final artifact |
| --- | --- |
| TCP stage listener | `shellcode/raw` or `stage/raw` |
| PsExec | Windows `service-executable/pe` |
| Backdoor | Windows `shellcode/raw` |
| DLL hijack | Windows `shared-library/pe` |

For `stage/raw`, the profile defines its payload transforms; the listener
does not add another set of compression or encryption transforms. The TCP
listener always adds its own outer four-byte length prefix to the final
payload bytes. A profile intended for that listener should leave any
internal `stage-frame` size prefix disabled. `profiles stage` executes an
authored `stage/raw` profile and saves its recorded bytes. It accepts
`--input-artifact name=path` and `--secret-file name=path` for declared
per-build inputs. Compression, encryption, and size-prefix options must be
declared as YAML transforms; the command rejects those flags for authored
profiles. Migrated profiles retain the old `GenerateStage` behavior for
existing per-invocation stage flags.

For `shellcode/raw`, the listener still accepts its legacy compression and
encryption flags and applies those to the verified final shellcode before
adding the TCP prefix. Those listener-specific served bytes are not a
separate stored build artifact.
