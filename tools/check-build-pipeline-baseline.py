#!/usr/bin/env python3
"""Check the M0 acceptance inventory against current source-level contracts.

This intentionally does not run builds or Go unit tests. Later milestones must
execute the listed target/mode combinations and record build and live results.
"""

import json
import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
MANIFEST = ROOT / "docs/build-pipeline/legacy-acceptance.json"


def fail(message):
    raise ValueError(message)


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            fail(f"duplicate JSON field: {key}")
        result[key] = value
    return result


def source(path):
    return (ROOT / path).read_text(encoding="utf-8")


def require(text, fragment, context):
    if fragment not in text:
        fail(f"{context}: missing source fragment {fragment!r}")


def function_body(text, name):
    match = re.search(rf"(?m)^func (?:\([^)]*\)\s+)?{re.escape(name)}\([^\n]*", text)
    if not match:
        fail(f"missing function {name}")
    next_func = re.search(r"(?m)^func ", text[match.end() :])
    return text[match.start() : match.end() + next_func.start() if next_func else len(text)]


def target_map(text):
    match = re.search(r"SupportedCompilerTargets\s*=\s*map\[string\]bool\s*\{([^}]*)\}", text, re.S)
    if not match:
        fail("missing SupportedCompilerTargets map")
    return set(re.findall(r'"(darwin|linux|windows)/(amd64|arm64|386)"\s*:\s*true', match.group(1)))


def check_source():
    client = source("client/command/generate/generate.go")
    commands = source("client/command/generate/commands.go")
    beacon = source("client/command/generate/generate-beacon.go")
    binaries = source("server/generate/binaries.go")
    adapter = source("server/generate/sliver_adapter.go")
    core_step = source("server/generate/core_step.go")
    pipeline = source("server/rpc/build-pipeline-execute.go")
    legacy_plan = source("server/rpc/legacy-build-plan.go")
    rpc = source("server/rpc/rpc-generate.go")
    runner = source("implant/sliver/runner/runner.go")
    encoder = source("server/encoders/shellcode/shellcode.go")
    transport = source("implant/sliver/transports/beacon.go")

    expected = {
        ("darwin", "amd64"), ("darwin", "arm64"),
        ("linux", "386"), ("linux", "amd64"), ("linux", "arm64"),
        ("windows", "386"), ("windows", "amd64"), ("windows", "arm64"),
    }
    if target_map(client) != expected or target_map(binaries) != expected:
        fail("client and server static target maps differ from M0 baseline")

    for fragment in (
        'f.StringP("os", "o", "windows"',
        'f.StringP("arch", "a", "amd64"',
        'f.StringP("format", "f", "exe"',
        'f.StringP("canary", "c", ""',
        'f.Uint32P("key-exchange", "X", DefaultWGKeyExPort',
        'f.Uint32P("tcp-comms", "T", DefaultWGNPort',
        'f.Int64P("reconnect", "j", DefaultReconnect',
        'f.Int64P("poll-timeout", "P", DefaultPollTimeout',
        'f.Uint32P("max-errors", "k", DefaultMaxErrors',
        'f.Int64P("seconds", "S", 60',
        'f.Int64P("jitter", "J", 30',
    ):
        require(commands, fragment, "console defaults")
    for fragment in ('DefaultReconnect = 60', 'DefaultPollTimeout = 360', 'DefaultMaxErrors = 1000',
                     'DefaultWGKeyExPort = 1337', 'DefaultWGNPort = 8888'):
        require(client, fragment, "connection defaults")
    require(client, 'canaryDomain += "."', "canary FQDN normalization")
    for fragment in ('case "exe":', 'case "shared":', 'case "archive":', 'case "shellcode":', 'case "service":',
                     'no core capability for output format',
                     'selectOutputFormat(format)',
                     'Must specify at least one of --mtls',
                     'Named pipe pivoting can only be used in Windows.',
                     'if targetArch != "amd64" && targetArch != "386"',
                     'if targetArch != "arm64"',
                     'if targetArch != "amd64" && targetArch != "arm64"'):
        require(client, fragment, "client route and rejection rules")
    require(beacon, 'interval < minBeaconInterval', "minimum beacon interval")

    generate_rpc = rpc[rpc.index("func (rpc *Server) Generate(") : rpc.index("// Regenerate -")]
    for fragment in (
        'adapter.Normalize(config)', 'legacyBuildRequest(name, config)',
        'legacyMetadataBuildRequest(', 'rpc.executeResolvedPlan(ctx, plan, adapter, config, resolveInput)',
    ):
        require(generate_rpc, fragment, "Generate RPC")
    for fragment in ('BuildCoreWithConfig(', 'compileShellcodeCore(', 'artifactaccept.Accept(',
                     'SaveFinal: saveFinal', 'Close: closeProduct'):
        require(core_step, fragment, "Sliver core step")
    for fragment in ('buildtransform.Convert(', 'buildtransform.EncodeShellcode(',
                     'buildtransform.ApplyStageStep(', 'buildtransform.ApplyPEMetadataConfig(',
                     'artifactaccept.InspectBytes(', 'store.Complete('):
        require(pipeline, fragment, "ordered transform executor")
    for fragment in ('builtin/donut', 'builtin/beignet', 'builtin/malasada',
                     'builtin/shikata-ga-nai', 'builtin/xor', 'builtin/xor-dynamic'):
        require(legacy_plan, fragment, "legacy typed transform mapping")
    for fragment in (
        'case clientpb.OutputFormat_EXECUTABLE, clientpb.OutputFormat_SERVICE:',
        'adapter.Executable(', 'adapter.SharedLibrary(', 'adapter.Archive(',
        'SliverShellcodeContext(', 'ImplantBuildSave(',
    ):
        require(adapter, fragment, "Sliver core adapter dispatch")
    require(runner, 'svc.Run("{{if .Config.ServiceName}}{{.Config.ServiceName}}{{else}}{{end}}"', "service startup")

    build_modes = {
        "Executable": 'Destination: dest,',
        "SharedLibrary": 'BuildMode: "c-shared"',
        "Archive": 'BuildMode: "c-archive"',
        "windowsShellcode": 'BuildMode: "pie"',
        "darwinShellcode": 'BuildMode: "c-shared"',
        "linuxShellcode": 'BuildMode: "c-shared"',
    }
    for function, fragment in build_modes.items():
        require(function_body(binaries, function), fragment, f"{function} Go mode")
    for fragment in ('zipFiles(zipDest, dest, headerDest, mainHeaderDest)',
                     'DonutShellcodeFromFile(', 'beignet.DylibToShellcode(',
                     'malasada.ConvertSharedObject('):
        require(binaries, fragment, "artifact conversion")
    require(binaries, 'if runtime.GOOS == DARWIN && platform[0] == LINUX && platform[1] == "386"',
            "Darwin host Linux/386 CGO exclusion")
    for fragment in ('"amd64": {', '"arm64": {', '"386": {',
                     '"shikata_ga_nai":', '"xor":', '"xor_dynamic":'):
        require(encoder, fragment, "encoder map")
    if 'case "namedpipe":' in transport or 'case "tcppivot":' in transport:
        fail("beacon pivot transport support changed; update M0 baseline")

    compile_body = function_body(client, "compileWithWorkspaceRetention")
    ordered = ('con.Rpc.Generate(', 'verifiedGenerateOutput(', 'os.WriteFile(saveTo, file.Data')
    positions = [compile_body.find(item) for item in ordered]
    if any(position < 0 for position in positions) or positions != sorted(positions):
        fail("server-final Generate/save order changed")
    if 'applySpoofMetadataToBuild(' in compile_body or 'con.Rpc.ShellcodeEncoder(' in compile_body:
        fail("local compile reintroduced post-build artifact mutation")
    stage = function_body(rpc, "GenerateStage")
    for fragment in ('legacyStageBuildRequest(', 'rpc.executeResolvedPlan(', 'Data: result.Data'):
        require(stage, fragment, "GenerateStage ordered transforms")

    # M6 moved consumers to the canonical profile and final-artifact path.
    # Keep the legacy client fallback in the inventory while checking that
    # callers accepting authored profiles use the frozen artifact type.
    consumers = {
        "client/command/jobs/stage.go": (
            "generate.GetProfileDefinitionOrLegacy(", "stageListenerProfileArtifact(",
            "stageListenerProfilePayload(", "generate.ExecuteProfile(", "generate.GetSliverBinary("),
        "client/command/exec/psexec.go": (
            "generate.GetProfileDefinitionOrLegacy(", "psexecProfileArtifact(",
            "psexecProfileBinary(", "generate.ExecuteProfile(", "generate.GetSliverBinary("),
        "server/rpc/rpc-backdoor.go": (
            'buildProfileBytesForConsumer(ctx, req.ProfileName, name, "shellcode", "raw", "windows")',),
        "server/rpc/rpc-hijack.go": (
            'buildProfileBytesForConsumer(ctx, req.ProfileName, name, "shared-library", "pe", "windows")',),
        "client/command/clean/clean.go": ("DeleteImplantProfile(",),
        "client/command/generate/profiles-generate.go": (
            "canonicalProfileGenerate(", "GetImplantProfileByName(name, con)"),
        "client/command/generate/profiles-stage.go": (
            "canonicalProfileStage(", "con.Rpc.GenerateStage("),
    }
    for path, fragments in consumers.items():
        for fragment in fragments:
            require(source(path), fragment, f"profile consumer {path}")
    return expected


def check_manifest(data, targets):
    if data.get("schemaVersion") != 1:
        fail("unsupported legacy acceptance schema version")
    routes = data.get("routes")
    edges = data.get("edgeCases")
    protocols = data.get("protocols")
    if not isinstance(routes, list) or not isinstance(edges, list) or not isinstance(protocols, list):
        fail("routes, edgeCases, and protocols must be lists")
    ids = set()
    rows = set()
    formats = {"exe", "service", "shared", "archive", "shellcode"}
    for item in routes:
        case_id = item.get("id")
        if not isinstance(case_id, str) or case_id in ids:
            fail(f"missing or duplicate route ID: {case_id}")
        ids.add(case_id)
        target = tuple(item.get("target", "").split("/"))
        fmt = item.get("format")
        if target not in targets or fmt not in formats or item.get("modes") != ["session", "beacon"]:
            fail(f"invalid target, format, or modes in {case_id}")
        os_name, arch = target
        if fmt == "service" and os_name != "windows":
            fail(f"non-Windows service advertised in {case_id}")
        if fmt == "shellcode" and target not in {
            ("windows", "386"), ("windows", "amd64"), ("darwin", "arm64"),
            ("linux", "amd64"), ("linux", "arm64")
        }:
            fail(f"unsupported shellcode tuple advertised in {case_id}")
        expected_route = {"exe": "SliverExecutable", "service": "SliverExecutable",
                          "shared": "SliverSharedLibrary", "archive": "SliverArchive",
                          "shellcode": "SliverShellcode"}[fmt]
        expected_mode = {"exe": "default", "service": "default", "shared": "c-shared",
                         "archive": "c-archive", "shellcode": "pie" if os_name == "windows" else "c-shared"}[fmt]
        converter = {"exe": "none", "service": "none", "shared": "none",
                     "archive": "zip-archive-headers", "shellcode": {"windows": "donut", "darwin": "beignet", "linux": "malasada"}[os_name]}[fmt]
        if item.get("rpc") != expected_route or item.get("goBuildMode") != expected_mode or item.get("converter") != converter:
            fail(f"incorrect route, Go mode, or converter in {case_id}")
        toolchain = "go" if fmt in ("exe", "service") or (fmt == "shellcode" and os_name == "windows") else "cgo-host-or-cross"
        status = "route" if toolchain == "go" else "route-if-toolchain"
        if item.get("toolchain") != toolchain or item.get("expected") != status:
            fail(f"incorrect availability in {case_id}")
        extension = ".zip" if fmt == "archive" else ".bin" if fmt == "shellcode" else ".exe" if os_name == "windows" and fmt in ("exe", "service") else {"windows": ".dll", "darwin": ".dylib", "linux": ".so"}[os_name] if fmt == "shared" else ""
        artifact = (f"shellcode-{arch}" if fmt == "shellcode" else
                    "c-archive-bundle" if fmt == "archive" else
                    "pe-service-executable" if fmt == "service" else
                    {"windows": "pe", "darwin": "macho", "linux": "elf"}[os_name] +
                    ("-shared-library" if fmt == "shared" else "-executable"))
        if item.get("extension") != extension or item.get("artifact") != artifact:
            fail(f"incorrect final artifact in {case_id}")
        rows.add((fmt, target))
    expected_rows = ({("exe", target) for target in targets} |
                     {("shared", target) for target in targets} |
                     {("archive", target) for target in targets} |
                     {("service", target) for target in targets if target[0] == "windows"} |
                     {("shellcode", target) for target in {
                         ("windows", "386"), ("windows", "amd64"), ("darwin", "arm64"),
                         ("linux", "amd64"), ("linux", "arm64")}})
    if rows != expected_rows or len(routes) != len(expected_rows):
        fail("routes omit or duplicate a static target/output combination")
    expected_edges = {
        "shellcode-windows-arm64": ("windows/arm64", "session", "shellcode", None, None, "client-reject"),
        "shellcode-darwin-amd64": ("darwin/amd64", "session", "shellcode", None, None, "client-reject"),
        "shellcode-linux-386": ("linux/386", "session", "shellcode", None, None, "client-reject"),
        "named-pipe-linux": ("linux/amd64", "session", "exe", "named-pipe", None, "client-reject"),
        "no-c2": ("windows/amd64", "session", "exe", "none", None, "client-reject"),
        "unknown-format": ("windows/amd64", "session", "unknown-format", None, None, "legacy-fallback-exe"),
        "service-nonwindows": ("linux/amd64", "session", "service", None, None, "not-advertised-legacy-rpc-route"),
        "darwin-host-linux-386-cgo": ("linux/386", "session", "shared", None, "darwin", "not-advertised-even-with-cross-compiler"),
        "beacon-named-pipe": ("windows/amd64", "beacon", "exe", "named-pipe", None, "client-accepts-runtime-unsupported"),
        "beacon-tcp-pivot": ("linux/amd64", "beacon", "exe", "tcp-pivot", None, "client-accepts-runtime-unsupported"),
    }
    edge_ids = {item.get("id") for item in edges}
    if len(edge_ids) != len(edges) or edge_ids != set(expected_edges):
        fail("missing, duplicate, or unexpected legacy edge case")
    for item in edges:
        actual = tuple(item.get(key) for key in ("target", "mode", "format", "protocol", "host", "expected"))
        if actual != expected_edges[item["id"]] or not item.get("reason"):
            fail(f"incorrect legacy edge case: {item['id']}")
    protocol_ids = {item.get("id") for item in protocols}
    if len(protocol_ids) != len(protocols) or protocol_ids != {
        "mtls", "wg", "http", "https", "dns", "named-pipe", "tcp-pivot"
    }:
        fail("missing, duplicate, or unexpected C2 protocol case")
    for item in protocols:
        if item.get("expected") != "supported" or item.get("modes") != (["session"] if item["id"] in ("named-pipe", "tcp-pivot") else ["session", "beacon"]):
            fail(f"incorrect protocol mode in {item.get('id')}")
        if item.get("targetRule") != ("windows-only" if item["id"] == "named-pipe" else "all-static-targets"):
            fail(f"incorrect protocol target rule in {item.get('id')}")
    return len(routes), len(routes) * 2, len(edges), len(protocols)


def main():
    data = json.loads(MANIFEST.read_text(encoding="utf-8"), object_pairs_hook=unique_object)
    counts = check_manifest(data, check_source())
    print(f"M0 baseline checked: {counts[0]} routes ({counts[1]} mode cases), {counts[2]} edge cases, {counts[3]} protocols; no builds executed")


if __name__ == "__main__":
    try:
        main()
    except (KeyError, OSError, ValueError) as exc:
        print(f"M0 baseline check failed: {exc}", file=sys.stderr)
        sys.exit(1)
