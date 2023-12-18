Starting in v1.5.30 Sliver supports "external builders," which allow a Sliver server to offload implant builds onto other systems. This can be used to increase platform support (e.g. connecting a MacBook to a Linux server to enable additional MacOS compiler targets) or increasing performance (e.g. having a low powered cloud host offload a local PC).

External builders can also be used to create custom modifications to the implant source code, or potentially replace the default Sliver implant entirely.

```
          MacOS .dylib Implant Builds
      ┌─────────────────────────────────────┐
      │                                     │
      ▼                                     │
┌───────────┐                         ┌─────┴─────┐
│ MacOS     │ Multiplayer             │ Linux     │
│ Builder   ├────────────────────────►│ Server    │
│           │                         │           │
└───────────┘                         └───────────┘
                                          ▲
┌───────────┐                             │
│Windows    │ Multiplayer                 │
│ Operator  ├─────────────────────────────┘
│           │
└───────────┘
```

## Builders

#### Setup

Any `sliver-server` binary can be started as a builder process using [operator configuration files from multiplayer-mode](/docs?name=Multi-player+Mode) from the server you want to connect the builder to, for example:

```
./sliver-server builder -c operator-multiplayer.cfg
```

When started as a builder, the Sliver process will mirror log output to stdout by default, however this can be disabled (see `sliver-server builder --help`).

**⚠️ IMPORTANT:** Make sure the builder and server have identical `http-c2.json` configuration files to avoid incompatibility problems.

**⚠️ IMPORTANT:** Builders must have unique names, by default the builder's hostname will be used, but this can be changed using the `--name` cli flag.

#### External Builds

Any operator can see which builders are connected to the server using the `builders` command. This command will also show what templates, formats, and compiler targets each builder supports:

```
sliver > builders

 Name                            Operator   Templates   Platform       Compiler Targets
=============================== ========== =========== ============== ==========================
 molochs-MacBook-Pro-111.local   moloch     sliver      darwin/arm64   EXECUTABLE:linux/386
                                                                       EXECUTABLE:linux/amd64
                                                                       EXECUTABLE:windows/386
                                                                       EXECUTABLE:windows/amd64
                                                                       EXECUTABLE:darwin/amd64
                                                                       EXECUTABLE:darwin/arm64
                                                                       SHARED_LIB:windows/386
                                                                       SHARED_LIB:windows/amd64
                                                                       SHARED_LIB:darwin/amd64
                                                                       SHARED_LIB:darwin/arm64
                                                                       SHARED_LIB:linux/amd64
                                                                       SERVICE:windows/386
                                                                       SERVICE:windows/amd64
                                                                       SHELLCODE:windows/386
                                                                       SHELLCODE:windows/amd64
```

Use the `--external-builder` flag to offload a `generate` or `generate beacon` command onto an external builder:

```
sliver > generate --mtls localhost --os mac --arch arm64 --external-builder

[*] Using external builder: molochs-MacBook-Pro-111.local
[*] Externally generating new darwin/arm64 implant binary
[*] Symbol obfuscation is enabled
[*] Creating external build ... done
[*] Build completed in 1m19s
```

If a given format/target combination is supported by multiple external builders you will be prompted to select one for the build.

#### Limitations

Currently external builds do not support DNS canaries.

## Implant Customization

You are welcome to customize the implant source code under the terms of Sliver's [GPLv3 license](https://github.com/BishopFox/sliver/blob/master/LICENSE). While we plan to improve the workflow over time, currently the easiest way to operationalize changes to the implant source code is:

1. Fork the main Sliver Github repository
1. Make modifications to the source code
1. [Compile a Sliver server binary](/docs?name=Compile+from+Source)
1. Connect the customized Sliver server binary to any other C2 server (including mainline servers) as an external builder
1. Operators can generate the customized implant builds via the `generate --external-builder` flag
1. Avoid making any changes to `/server` to make merging upstream easier if changes are introduced to the builder APIs
