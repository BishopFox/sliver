const std = @import("std");
const CrossTarget = std.zig.CrossTarget;

pub fn build(b: *std.build.Builder) void {
    // Standard release options allow the person running `zig build` to select
    // between Debug, ReleaseSafe, ReleaseFast, and ReleaseSmall.
    const mode = b.standardReleaseOptions();

    const exe = b.addExecutable("wasi", "wasi.zig");
    exe.setTarget(CrossTarget{ .cpu_arch = .wasm32, .os_tag = .wasi });
    exe.setBuildMode(mode);
    exe.install();
}
