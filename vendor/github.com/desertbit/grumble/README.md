# Grumble - A powerful modern CLI and SHELL

[![GoDoc](https://godoc.org/github.com/desertbit/grumble?status.svg)](https://godoc.org/github.com/desertbit/grumble)
[![Go Report Card](https://goreportcard.com/badge/github.com/desertbit/grumble)](https://goreportcard.com/report/github.com/desertbit/grumble)

There are a handful of powerful go CLI libraries available ([spf13/cobra](https://github.com/spf13/cobra), [urfave/cli](https://github.com/urfave/cli)).
However sometimes an integrated shell interface is a great and useful extension for the actual application.
This library offers a simple API to create powerful CLI applications and automatically starts
an **integrated interactive shell**, if the application is started without any command arguments.

**Hint:** The API might change slightly, until a first 1.0 release is published.

[![asciicast](https://asciinema.org/a/155332.png)](https://asciinema.org/a/155332?t=5)

## Introduction

Create a grumble APP.

```go
var app = grumble.New(&grumble.Config{
	Name:        "app",
	Description: "short app description",

	Flags: func(f *grumble.Flags) {
		f.String("d", "directory", "DEFAULT", "set an alternative directory path")
		f.Bool("v", "verbose", false, "enable verbose mode")
	},
})
```

Register a top-level command. *Note: Sub commands are also supported...*

```go
app.AddCommand(&grumble.Command{
    Name:      "daemon",
    Help:      "run the daemon",
    Aliases:   []string{"run"},
    Usage:     "daemon [OPTIONS]",
    AllowArgs: true,

    Flags: func(f *grumble.Flags) {
        f.Duration("t", "timeout", time.Second, "timeout duration")
    },

    Run: func(c *grumble.Context) error {
        fmt.Println("timeout:", c.Flags.Duration("timeout"))
        fmt.Println("directory:", c.Flags.String("directory"))
        fmt.Println("verbose:", c.Flags.Bool("verbose"))

        // Handle args.
        fmt.Println("args:")
        fmt.Println(strings.Join(c.Args, "\n"))

        return nil
    },
})
```

Run the application.

```go
err := app.Run()
```

Or use the builtin *grumble.Main* function to handle errors automatically.

```go
func main() {
	grumble.Main(app)
}
```

## Shell Multiline Input

Builtin support for multiple lines.

```
>>> This is \
... a multi line \
... command
```

## Samples

Check out the [sample directory](/sample) for some detailed examples.

The [grml project](https://github.com/desertbit/grml) uses grumble.

## Additional Useful Packages

- https://github.com/AlecAivazis/survey
- https://github.com/tj/go-spin

## Credits

This project is based on ideas from the great [ishell](https://github.com/abiosoft/ishell) library.

## License

MIT
