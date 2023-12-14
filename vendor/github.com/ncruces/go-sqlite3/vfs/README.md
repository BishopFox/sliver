# Go SQLite VFS API

This package implements the SQLite [OS Interface](https://sqlite.org/vfs.html) (aka VFS).

It replaces the default SQLite VFS with a pure Go implementation.

It also exposes interfaces that should allow you to implement your own custom VFSes.