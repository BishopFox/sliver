
## Getting Started: Embedding the readline library in a console application


This page shows how to start using the library in your console application. Please read carefully, and then 
proceed with the other pages of the documentation in order. Hopefully, by the end of them, you should have a 
shell leveraging most, if not all, of the features offered by this library. A few notes before anything else:

1) As mentionned in the README, this library is not an integrated console application. This means **it does not natively 
handles a REPL-style loop**, but the examples along the documentation will show you how to define one and use the library accordingly.
2) The variables declared in the example below will be reused in the following sections.
3) If some things are not entirely clear to you, they will probably be explained in the following sections of this documentation.
5) The code is pulled out of the [examples](https://github.com/bishopfox/sliver/client/readline/tree/master/examples) directory. Check it out if needed.


### Embedding the shell readline

For the sake of an example project, we define a `console` type embedding our readline instance, as well
as a [go-flags](https://github.com/jessevdk/go-flags) command parser. As explained before and detailed later, the
go-flags parser plays well with our library: the latter has default completers working with such a `Parser` type.
This example involves a relatively simple struct, but feel free to add more involved logic if needed.

```go
// console - A simple console example.
type console struct {
	shell  *readline.Instance   // Our shell instance.
	parser *flags.Parser        // The go-flags command parser, will be explained in further sections.
}
```

We then instantiate the console with constructor (being serious persons):
We will modify/add parameters along the documentation. 

```go
func newConsole(commandParser *flags.Parser) *console {
	console := &console{
		shell:  readline.NewInstance(), // The readline instance is constructed with some default behavior.
		parser: commandParser,          
	}
	return console
}
```

Everything related to the shell's setup will be declared in the following function, for the sake of examples.

```go
// setup - The console sets up various elements such as the completion system, hints,
// syntax highlighting, prompt system, commands binding, and client environment loading.
func (c *console) setup() (err error) {

    // Input modes and details

    // Prompt system settings

    // Completers

    // History

    return
}
```

We then setup the [shell's input mode](https://github.com/bishopfox/sliver/client/readline/wiki/Input-Modes)
