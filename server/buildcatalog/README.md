# Server build spec catalog

`buildcatalog.Open(appRoot, installedExecutorIDs)` embeds the shipped Sliver
core and transform specs. The server reads administrator specs from
`<appRoot>/build-catalog/admin/*.yaml` or `*.yml` at startup. Place a complete
`CoreSpec` or `TransformSpec` there, with a digest calculated by the
[M0 contract](../../docs/build-pipeline/README.md), and restart the server.
An unknown executor ID or conflicting name/version prevents startup.
`installedExecutorIDs` extends the built-in allowlist only for integrations
that the server has actually installed.

Administrator and remote snapshots are stored in
`<appRoot>/build-catalog/snapshots.json`. Remote provider status is separate
from the immutable YAML. Restarting the server loads remote definitions as
offline until they advertise again. Built-in and administrator specs use the
`server` provider. An online provider means its executor is present; exact
target compiler and capacity checks are performed during planning.

The importer preserves raw YAML and checks its canonical digest. It rejects
aliases, duplicate keys, unknown fields, ambiguous scalars, unsafe integers,
and floats. Float values need a full RFC 8785 number serializer before import
can support them; the shipped specs do not use float values.
