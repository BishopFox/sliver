# Go SQLite VFS API

This package implements the SQLite [OS Interface](https://www.sqlite.org/vfs.html) (aka VFS).

It replaces the default VFS with a pure Go implementation,
that is tested on Linux, macOS and Windows,
but which should also work on illumos and the various BSDs.

It also exposes interfaces that should allow you to implement your own custom VFSes.