Command
=======

This package contains all of the client command implementations that are shared between the client and the server console.

### Developers

General guidance on the structure of this package:
 * One command per file and the name of the file should reflect the command it implements (or there about, trivial commands can go into the same file). This makes it easy to find where a command implementation lives within a package or to search by file name if it is not clear which package implements a given command.
 * The root command, and reused code should go into the file named after the command
   * For example, code shared between the `generate` and the `regenerate` command should go in `generate.go`, and any `regenerate` specific code should go in `regenerate.go`
 * Command entrypoint functions should have the suffix `Cmd` e.g., `GenerateCmd` is the entrypoint for `generate`
   * Command entrypoints should always a function signature of `func (ctx *grumble.Context, con *console.SliverConsoleClient)`
 * Functions that are only ever exported for other commands to use should go in a `helpers.go`, if the function is used internally and exported follow the guidance above on shared code.
