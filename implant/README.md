Implant
=======

This directory contains the code for Sliver's implant, implant source code is dynamically
rendered at runtime via the `generate` command. The code generation inserts the per-binary
values such as X.509 certificates, etc. and compiles it to produce a binary.

The implant code contains a lot of platform specific code too, which varies the features
that will be supported on different platforms.

Platform agnostic code is implemented in `_generic.go` files, and can be compiled for any
valid Go compiler target but only contains very generic commands/features.

Development
===========

Before committing any changes to any implant files, run `go generate` in this directory. This
will ensure the vendor directory is kept up to date so offline implant builds will function
correctly.
