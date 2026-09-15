<p align="center">
  <img src=".github/images/logo.png" alt="OPFOR logo" width="220">
</p>

OPFOR is an independent, pure-Go runtime for the Sleep language and Aggressor
Script (`.cna`). The same engine is available as an embeddable Go package and
as the offline `opfor` CLI, so applications can host scripts and operators can
evaluate, validate, and run them without a JVM.

OPFOR implements parsing, compilation, execution, portable Sleep built-ins,
script lifecycle, events, hooks, and callbacks. Embedding applications supply
host-specific data and effects through explicit Aggressor Script extension and
provider APIs.

## Embed OPFOR

Create a runtime and evaluate a source string:

```go
package main

import (
	"context"
	"log"

	"github.com/sliverarmory/opfor"
)

func main() {
	ctx := context.Background()
	runtime, err := opfor.New()
	if err != nil {
		log.Fatal(err)
	}
	defer runtime.Close(ctx)

	if _, err := runtime.Eval(ctx, "hello.sl", `println("hello from OPFOR");`); err != nil {
		log.Fatal(err)
	}
}
```

This prints `hello from OPFOR`. `Eval` compiles and runs source in one call; use
`CompileString` and `Runtime.Execute` when a program will be reused. `println`
writes to standard output by default. Use `WithStdin`, `WithStdout`, and
`WithStderr` to replace the process streams.

See the [Aggressor Script extension and provider API
reference](docs/aggressor-script-extensions.md) to connect application-owned
state, actions, UI, callbacks, catalogs, and custom functions.

### Call between Sleep and Go

`WithFunction` exposes a Go function to Sleep. Keep the loaded `Script` to call
a Sleep `sub` from Go with `Script.Call`:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/sliverarmory/opfor"
)

func main() {
	ctx := context.Background()
	runtime, err := opfor.New(
		opfor.WithFunction("hello_from_go", func(_ context.Context, call opfor.Invocation) (opfor.Value, error) {
			name := call.Arg(0).String()
			return opfor.String("hello " + name + " from Go"), nil
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer runtime.Close(ctx)

	program, err := runtime.CompileString("bridge.sl", `
sub hello_from_sleep {
    return "hello " . $1 . " from Sleep";
}

println(hello_from_go("Sleep"));
`)
	if err != nil {
		log.Fatal(err)
	}

	script, err := runtime.Load(ctx, program)
	if err != nil {
		log.Fatal(err)
	}

	reply, err := script.Call(ctx, "hello_from_sleep", opfor.String("Go"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(reply.String())
}
```

Loading the script calls `hello_from_go` and prints `hello Sleep from Go`.
The later `Script.Call` prints `hello Go from Sleep`. `Runtime.Close` unloads
the retained script and its functions.

## CLI

Install the `opfor` interpreter with Go 1.24 or later:

```sh
go install github.com/sliverarmory/opfor/cmd/opfor@latest
```

From a source checkout, install that checkout with:

```sh
go install ./cmd/opfor
```

Or build it in the repository root:

```sh
make
```

Then evaluate an expression, validate a script, or run it:

```sh
./opfor eval '2 + 2'
./opfor check examples/01-hello.sl
./opfor run examples/01-hello.sl operator
./opfor repl
```

The REPL displays a colored `opfor > ` prompt and red evaluation errors in a
terminal. Redirected input and output remain prompt-free for line-oriented
pipelines.

See [`examples/`](examples/) for runnable scripts that use only stock Sleep
syntax and built-ins.

Running `./opfor` without arguments prints the complete command help. The CLI
is an offline interpreter; it does not connect or authenticate to an external
Aggressor Script host.

## Scope

OPFOR implements Sleep and Aggressor Script, not Java. A small pure-Go
compatibility shim covers the Java-shaped string, collection, file, random,
and UUID behavior needed by supported scripts. Other object behavior can be
provided by the embedding application.

Java serialization is optional compatibility support for scripts that
explicitly use it; it is not required for normal embedding, execution, or
callbacks.

See the [detailed compatibility and embedding reference](docs/README.md) for
the provider catalog, lifecycle contracts, limits, conformance kit, and current
compatibility coverage.

## License

Apache License 2.0. See [LICENSE](LICENSE).
