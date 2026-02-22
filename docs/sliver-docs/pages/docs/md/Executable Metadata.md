Executable metadata spoofing lets you post-process a generated build and copy metadata/resource details from a donor executable.

## Overview

The feature is enabled with the `--spoof-metadata` flag on generate commands.

- Current format support: `pe` (Windows PE)
- Config file: `<client app dir>/spoof-metadata.yaml`
- Default client app dir: `~/.sliver-client`

On first client startup, Sliver creates `spoof-metadata.yaml` if it does not exist.

## Flag Usage

`--spoof-metadata` is off by default.

- No flag: metadata spoofing is not applied.
- `--spoof-metadata` (no argument): use `<client app dir>/spoof-metadata.yaml`.
- `--spoof-metadata /path/to/donor.exe`: use the provided donor file directly.

Path mode validation:

- Currently only supported for Windows PE targets.
- Target output format must be executable, service, or shared library.
- Donor PE machine type must match target architecture (`386`, `amd64`, `arm64`).
- For shared library targets, donor must be a DLL; for executable/service targets, donor must not be a DLL.

When path mode is used, Sliver only uses the donor file bytes as `pe.source` and does not read override fields from `spoof-metadata.yaml`.

## Quick Start

1. Start the client once to generate `~/.sliver-client/spoof-metadata.yaml`.
2. Edit `pe.source.path` to point to a donor PE file.
3. Generate with `--spoof-metadata`.

Example:

```bash
sliver > generate --os windows --arch amd64 --format exe --mtls 10.0.0.5 --spoof-metadata -N win-agent
```

Direct donor path example:

```bash
sliver > generate --os windows --arch amd64 --format exe --mtls 10.0.0.5 --spoof-metadata /opt/samples/notepad.exe -N win-agent
```

## Config Format

Top-level config:

```yaml
pe:
  source:
    name: metadata-source.exe
    path: /absolute/path/to/donor.exe
```

`pe.source` is required when using `--spoof-metadata`.

Fields that can carry binary content (such as `source` and `icon`) support:

- `name`: logical filename sent in protobuf metadata
- `path`: local file path read by the client
- `base64`: inline base64 bytes

`name` and `path` are not duplicates:

- `path` controls where bytes are loaded from on the client machine.
- `name` is the filename value sent to the server.

If `path` is set and `name` is omitted, Sliver uses `basename(path)` automatically.

Set only one of `path` or `base64` for each object.

## Examples

### Minimal Path-Based Config

```yaml
pe:
  source:
    name: metadata-source.exe
    path: /opt/samples/notepad.exe
```

### Include Icon Donor Bytes

```yaml
pe:
  source:
    name: metadata-source.exe
    path: /opt/samples/notepad.exe
  icon:
    name: icon.ico
    path: /opt/samples/app.ico
```

### Inline Base64 Config

```yaml
pe:
  source:
    name: metadata-source.exe
    base64: TVqQAAMAAAAEAAAA...
  icon:
    name: icon.ico
    base64: AAABAAEAICAAAAEAIACoEAAAFgAAACgAAAAgAAAAQAAAAAEAIAAAAAAA...
```

### Advanced Optional PE Structure Fields

These fields are optional and act as PE metadata overrides:

```yaml
pe:
  source:
    name: metadata-source.exe
    path: /opt/samples/notepad.exe

  resource_directory:
    major_version: 1
    minor_version: 0
    time_date_stamp: 1712345678
    number_of_named_entries: 0
    number_of_id_entries: 0

  resource_directory_entries:
    - name: 1
      offset_to_data: 0

  resource_data_entries:
    - offset_to_data: 0
      size: 0
      code_page: 0
      reserved: 0

  export_directory:
    major_version: 1
    minor_version: 0
    time_date_stamp: 1712345678
    number_of_functions: 0
    number_of_names: 0
```

## Override Behavior

Override order for `pe`:

1. Sliver clones metadata/resources from `pe.source`.
2. Sliver applies explicit override fields from your config.

This means:

- If an override field is set, it replaces the cloned value.
- If an override field is omitted, the cloned donor value is kept.

Resource override details:

- `resource_directory` overrides the root `IMAGE_RESOURCE_DIRECTORY` fields.
- `resource_directory_entries` overrides the first N root directory entries in order.
- `resource_data_entries` overrides the first N resource data entries discovered in resource-tree order.

Export override details:

- `export_directory` overrides fields in the PE export directory when that directory exists.
- If a PE has no export directory, this override has no effect.

Important:

- In protobuf/YAML, scalar zero values are real override values when the parent block is present.
- When setting partial `resource_directory` values, include `number_of_named_entries` and `number_of_id_entries` from the original header unless you intentionally want to change them.

## Notes

- The client reads local file paths and sends bytes to the server over gRPC.
- Spoofing is applied only after a build completes successfully.
- The server applies spoofing to the selected existing build artifact.
- PE support is implemented first; the config is structured for future executable formats.
