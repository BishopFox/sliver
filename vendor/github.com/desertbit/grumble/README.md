# Grumble - A powerful modern CLI and SHELL

[![GoDoc](https://godoc.org/github.com/desertbit/grumble?status.svg)](https://godoc.org/github.com/desertbit/grumble)
[![Go Report Card](https://goreportcard.com/badge/github.com/desertbit/grumble)](https://goreportcard.com/report/github.com/desertbit/grumble)

There are a handful of powerful go CLI libraries available ([spf13/cobra](https://github.com/spf13/cobra), [urfave/cli](https://github.com/urfave/cli)).
However sometimes an integrated shell interface is a great and useful extension for the actual application.
This library offers a simple API to create powerful CLI applications and automatically starts
an **integrated interactive shell**, if the application is started without any command arguments.

**Hint:** We do not guarantee 100% backwards compatiblity between minor versions (1.x). However, the API is mostly stable and should not change much.

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

    Flags: func(f *grumble.Flags) {
        f.Duration("t", "timeout", time.Second, "timeout duration")
    },

    Args: func(a *grumble.Args) {
        a.String("service", "which service to start", grumble.Default("server"))
    },

    Run: func(c *grumble.Context) error {
        // Parent Flags.
        c.App.Println("directory:", c.Flags.String("directory"))
        c.App.Println("verbose:", c.Flags.Bool("verbose"))
        // Flags.
        c.App.Println("timeout:", c.Flags.Duration("timeout"))
        // Args.
        c.App.Println("service:", c.Args.String("service"))
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

## Separate flags and args specifically
If you need to pass a flag-like value as positional argument, you can do so by using a double dash:  
`>>> command --flag1=something -- --myPositionalArg`

## Remote shell access with readline
By calling RunWithReadline() rather than Run() you can pass instance of readline.Instance. 
One of interesting usages is having a possibility of remote access to your shell:

```go
handleFunc := func(rl *readline.Instance) {

    var app = grumble.New(&grumble.Config{
        // override default interrupt handler to avoid remote shutdown
        InterruptHandler: func(a *grumble.App, count int) {
            // do nothing
        },
		
        // your usual grumble configuration
    })  
    
    // add commands
	
    app.RunWithReadline(rl)

}

cfg := &readline.Config{}
readline.ListenRemote("tcp", ":5555", cfg, handleFunc)
```

In the client code just use readline built in DialRemote function:

```go
if err := readline.DialRemote("tcp", ":5555"); err != nil {
    fmt.Errorf("An error occurred: %s \n", err.Error())
}
```

## Samples

Check out the [sample directory](/sample) for some detailed examples.

## Projects using Grumble

- grml - A simple build automation tool written in Go: https://github.com/desertbit/grml
- orbit - A RPC-like networking backend written in Go: https://github.com/desertbit/orbit

## Known issues
- Windows unicode not fully supported ([issue](https://github.com/desertbit/grumble/issues/48))

## Additional Useful Packages

- https://github.com/AlecAivazis/survey
- https://github.com/tj/go-spin

## Credits

This project is based on ideas from the great [ishell](https://github.com/abiosoft/ishell) library.

## License

MIT
