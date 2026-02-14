This page summarizes implant payload format compatibility by target OS/architecture (`GOOS/GOARCH`).

## Common Platforms

| Target OS/Arch  | Executable | Shared Library | Shellcode | Service |
| --------------- | ---------- | -------------- | --------- | ------- |
| `windows/386`   | ✅         | ✅             | ✅        | ✅      |
| `windows/amd64` | ✅         | ✅             | ✅        | ✅      |
| `windows/arm64` | ⚠️         | ⚠️             | ❌        | ⚠️      |
| `linux/386`     | ✅         | ✅             | ❌        | ❌      |
| `linux/amd64`   | ✅         | ✅             | ✅        | ❌      |
| `linux/arm64`   | ✅         | ✅             | ✅        | ❌      |
| `darwin/amd64`  | ✅         | ✅             | ❌        | ❌      |
| `darwin/arm64`  | ✅         | ✅             | ✅        | ❌      |

### Important Notes

- `✅` = first-class support in Sliver's built-in target matrix.
- `⚠️` = generic/experimental target (not first-class; may fail depending on toolchain/target).
- `❌` = not currently supported for that payload format.
- `Service` is a Windows-only format.
- `Shellcode` is currently supported for `windows/{386,amd64}`, `linux/{amd64,arm64}`, and `darwin/arm64`.
- `Shellcode` and `Shared Library` for MacOS may require a [cross-compiler](/docs?name=Cross-compiling+Implants) or an [external builder](/docs?name=External+Builders) depending on your platform.
- On macOS hosts, targeting `linux/386` for shared library/shellcode builds is currently not reliable.

Use `generate info` in the Sliver console to see what your current server can build with its configured toolchains.

## All GOOS/GOARCH Targets

The table below covers all `GOOS/GOARCH` entries from `go tool dist list` (Go 1.24+), including common platforms:

| Target GOOS/GOARCH | Common Platform | Executable | Shared Library | Shellcode | Service |
| ------------------ | --------------- | ---------- | -------------- | --------- | ------- |
| `aix/ppc64` | No | ⚠️ | ❌ | ❌ | ❌ |
| `android/386` | No | ⚠️ | ❌ | ❌ | ❌ |
| `android/amd64` | No | ⚠️ | ❌ | ❌ | ❌ |
| `android/arm` | No | ⚠️ | ❌ | ❌ | ❌ |
| `android/arm64` | No | ⚠️ | ❌ | ❌ | ❌ |
| `darwin/amd64` | Yes | ✅ | ✅ | ❌ | ❌ |
| `darwin/arm64` | Yes | ✅ | ✅ | ✅ | ❌ |
| `dragonfly/amd64` | No | ⚠️ | ❌ | ❌ | ❌ |
| `freebsd/386` | No | ⚠️ | ❌ | ❌ | ❌ |
| `freebsd/amd64` | No | ⚠️ | ❌ | ❌ | ❌ |
| `freebsd/arm` | No | ⚠️ | ❌ | ❌ | ❌ |
| `freebsd/arm64` | No | ⚠️ | ❌ | ❌ | ❌ |
| `freebsd/riscv64` | No | ⚠️ | ❌ | ❌ | ❌ |
| `illumos/amd64` | No | ⚠️ | ❌ | ❌ | ❌ |
| `ios/amd64` | No | ⚠️ | ❌ | ❌ | ❌ |
| `ios/arm64` | No | ⚠️ | ❌ | ❌ | ❌ |
| `js/wasm` | No | ⚠️ | ❌ | ❌ | ❌ |
| `linux/386` | Yes | ✅ | ✅ | ❌ | ❌ |
| `linux/amd64` | Yes | ✅ | ✅ | ✅ | ❌ |
| `linux/arm` | No | ⚠️ | ❌ | ❌ | ❌ |
| `linux/arm64` | Yes | ✅ | ✅ | ✅ | ❌ |
| `linux/loong64` | No | ⚠️ | ❌ | ❌ | ❌ |
| `linux/mips` | No | ⚠️ | ❌ | ❌ | ❌ |
| `linux/mips64` | No | ⚠️ | ❌ | ❌ | ❌ |
| `linux/mips64le` | No | ⚠️ | ❌ | ❌ | ❌ |
| `linux/mipsle` | No | ⚠️ | ❌ | ❌ | ❌ |
| `linux/ppc64` | No | ⚠️ | ❌ | ❌ | ❌ |
| `linux/ppc64le` | No | ⚠️ | ❌ | ❌ | ❌ |
| `linux/riscv64` | No | ⚠️ | ❌ | ❌ | ❌ |
| `linux/s390x` | No | ⚠️ | ❌ | ❌ | ❌ |
| `netbsd/386` | No | ⚠️ | ❌ | ❌ | ❌ |
| `netbsd/amd64` | No | ⚠️ | ❌ | ❌ | ❌ |
| `netbsd/arm` | No | ⚠️ | ❌ | ❌ | ❌ |
| `netbsd/arm64` | No | ⚠️ | ❌ | ❌ | ❌ |
| `openbsd/386` | No | ⚠️ | ❌ | ❌ | ❌ |
| `openbsd/amd64` | No | ⚠️ | ❌ | ❌ | ❌ |
| `openbsd/arm` | No | ⚠️ | ❌ | ❌ | ❌ |
| `openbsd/arm64` | No | ⚠️ | ❌ | ❌ | ❌ |
| `openbsd/ppc64` | No | ⚠️ | ❌ | ❌ | ❌ |
| `openbsd/riscv64` | No | ⚠️ | ❌ | ❌ | ❌ |
| `plan9/386` | No | ⚠️ | ❌ | ❌ | ❌ |
| `plan9/amd64` | No | ⚠️ | ❌ | ❌ | ❌ |
| `plan9/arm` | No | ⚠️ | ❌ | ❌ | ❌ |
| `solaris/amd64` | No | ⚠️ | ❌ | ❌ | ❌ |
| `wasip1/wasm` | No | ⚠️ | ❌ | ❌ | ❌ |
| `windows/386` | Yes | ✅ | ✅ | ✅ | ✅ |
| `windows/amd64` | Yes | ✅ | ✅ | ✅ | ✅ |
| `windows/arm64` | Yes | ⚠️ | ⚠️ | ❌ | ⚠️ |
