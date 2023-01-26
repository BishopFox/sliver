const std = @import("std");
const os = std.os;
const allocator = std.heap.page_allocator;
const stdout = std.io.getStdOut().writer();

pub fn main() !void {
    // Allocate arguments from the the operating system.
    const args = try std.process.argsAlloc(allocator);
    defer std.process.argsFree(allocator, args);

    if (std.mem.eql(u8, args[1], "ls")) {
        var dir = std.fs.cwd().openIterableDir(args[2], .{}) catch |err| switch (err) {
            error.NotDir => {
                try stdout.print("ENOTDIR\n", .{});
                return;
            },
            else => {
                try stdout.print("./{}\n", .{err});
                return;
            },
        };
        var iter = dir.iterate();
        while (try iter.next()) |entry| {
            try stdout.print("./{s}\n", .{entry.name});
        }
    } else if (std.mem.eql(u8, args[1], "stat")) {
        try stdout.print("stdin isatty: {}\n", .{os.isatty(0)});
        try stdout.print("stdout isatty: {}\n", .{os.isatty(1)});
        try stdout.print("stderr isatty: {}\n", .{os.isatty(2)});
        try stdout.print("/ isatty: {}\n", .{os.isatty(3)});
    }
}
