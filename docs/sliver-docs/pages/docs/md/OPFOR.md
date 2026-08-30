[OPFOR](https://github.com/sliverarmory/opfor) runs a BOF-focused subset of CNA/Aggressor Script directly inside `sliver-client`. It lets operators use projects that already ship a `.cna` wrapper and BOF object files without first translating the script into a Sliver extension manifest.

OPFOR is not a complete Cobalt Strike client or a complete Aggressor Script implementation. Sliver currently connects the parts needed by common BOF scripts: local script resources, Beacon alias registration and help, target metadata queries, BOF argument packing, transcript output, `beacon_inline_execute`, result callbacks, and basic terminal prompts.

For manifest-based BOF packages and Armory installation, see [BOF and COFF Support](/docs?name=BOF+and+COFF+Support).

## Requirements

- Keep the `.cna` file and its resource tree on the computer running `sliver-client`. Calls such as `script_resource()` resolve files from that local script tree.
- BOF aliases require an active session or beacon. The target must advertise Sliver's `bof_v1` capability.
- The tested BOF execution path supports Windows x64 and x86 targets. A CNA may perform its own architecture selection with functions such as `barch()`.
- Review third-party scripts before loading them. Top-level CNA code runs locally in the Sliver client process.

Loaded scripts and their registrations belong only to the current client process. They are not installed in Armory, copied to the server, or shared with other operators, and they must be loaded again after restarting the client.

## Commands

| Command | Behavior |
| --- | --- |
| `opfor check <script.cna>` | Compile the script without executing its top-level code. |
| `opfor run <script.cna> [arguments...]` | Execute top-level code once with trailing values in `@ARGV`, then discard registrations. |
| `opfor load <script.cna>` | Execute initialization and retain aliases, hooks, and callbacks. |
| `opfor <script.cna>` | Shorthand for `opfor load <script.cna>`. |
| `opfor list` | List scripts retained by this client process. |
| `opfor help <alias>` | Display the description and detail supplied by `beacon_command_register()`. |
| `opfor <alias> [arguments...]` | Invoke a loaded Beacon alias against the active target. |
| `opfor unload <script.cna>` | Retire a loaded script and its registrations. |

`opfor run` does not invoke aliases declared by the script. Most BOF CNAs define an `alias` at top level, so running one only creates that alias in a temporary runtime and then retires it. Use `opfor load`, select a target, and invoke the registered alias instead.

The default operation timeout is 600 seconds. Put a timeout override before the alias name so Sliver consumes it instead of passing it to the CNA command:

```
opfor --timeout 900 firefoxdump /all
```

Arguments after the alias name are passed to the CNA as written, including flag-like values. If the CNA itself needs a leading `--timeout` or `-t` argument, place `--` before it so the OPFOR host stops parsing timeout overrides:

```
opfor example -- --timeout
```

## Loading and running FirefoxDump

Keep the script beside the object files expected by its `script_resource()` calls. For example:

```
firefoxdump/
├── firefoxdump.cna
└── bin/
    ├── firefoxdump.x64.o
    └── firefoxdump.x86.o
```

Check the script before executing its initialization code:

```
sliver > opfor check ./firefoxdump/firefoxdump.cna
./firefoxdump/firefoxdump.cna: ok
```

Load it persistently:

```
sliver > opfor load ./firefoxdump/firefoxdump.cna

[*] Loaded CNA script /absolute/path/firefoxdump/firefoxdump.cna
[*] Registered CNA aliases:
  opfor firefoxdump
[*] View CNA alias help:
  opfor help firefoxdump
```

Inspect the command metadata without running the BOF:

```
sliver > opfor help firefoxdump

Command: opfor firefoxdump [arguments...]

Extract Firefox secrets (cookies, passwords) from Firefox profile.
```

Select a compatible target and invoke the alias:

```
sliver > use <session-or-beacon>
sliver (TARGET) > opfor firefoxdump /all
```

OPFOR sends the selected object file and packed arguments through Sliver's native BOF execution path. Session results are rendered immediately; beacon tasks are correlated and polled until the result or command deadline arrives. Typed BOF output channels and CNA callbacks are preserved when the script uses them.

Unload the script when it is no longer needed:

```
sliver > opfor unload ./firefoxdump/firefoxdump.cna
```

## Compatibility and troubleshooting

- If an alias is missing after `opfor run`, load the script instead. One-shot registrations are intentionally discarded.
- If Sliver reports that no target is active, run `use <session-or-beacon>` before invoking the alias.
- If the target does not advertise `bof_v1`, update the server, client, and implant to compatible builds and establish a new session or beacon.
- If `script_resource()` cannot find an object file, preserve the upstream directory layout relative to the `.cna` file.
- If a script calls an unsupported Aggressor function or prompt type, OPFOR returns an explicit error. The current integration prioritizes BOF execution rather than every Aggressor callback or client feature.
- CNA aliases are namespaced beneath `opfor`, so they do not replace built-in Sliver commands. The management names `check`, `help`, `list`, `load`, `run`, and `unload` remain reserved.
