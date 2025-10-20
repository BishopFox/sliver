<!-- markdownlint-disable MD013 MD033 -->
<p align="center"><img width="90%" src="kong.png" /></p>

# Kong is a command-line parser for Go

[![](https://godoc.org/github.com/alecthomas/kong?status.svg)](http://godoc.org/github.com/alecthomas/kong) [![CircleCI](https://img.shields.io/circleci/project/github/alecthomas/kong.svg)](https://circleci.com/gh/alecthomas/kong) [![Go Report Card](https://goreportcard.com/badge/github.com/alecthomas/kong)](https://goreportcard.com/report/github.com/alecthomas/kong) [![Slack chat](https://img.shields.io/static/v1?logo=slack&style=flat&label=slack&color=green&message=gophers)](https://gophers.slack.com/messages/CN9DS8YF3)

<!-- TOC depthfrom:2 depthto:3 -->

- [Introduction](#introduction)
- [Help](#help)
  - [Help as a user of a Kong application](#help-as-a-user-of-a-kong-application)
  - [Defining help in Kong](#defining-help-in-kong)
- [Command handling](#command-handling)
  - [Switch on the command string](#switch-on-the-command-string)
  - [Attach a Run... error method to each command](#attach-a-run-error-method-to-each-command)
- [Hooks: BeforeReset, BeforeResolve, BeforeApply, AfterApply and the Bind option](#hooks-beforereset-beforeresolve-beforeapply-afterapply-and-the-bind-option)
- [Flags](#flags)
- [Commands and sub-commands](#commands-and-sub-commands)
- [Branching positional arguments](#branching-positional-arguments)
- [Positional arguments](#positional-arguments)
- [Slices](#slices)
- [Maps](#maps)
- [Pointers](#pointers)
- [Nested data structure](#nested-data-structure)
- [Custom named decoders](#custom-named-decoders)
- [Supported field types](#supported-field-types)
- [Custom decoders mappers](#custom-decoders-mappers)
- [Supported tags](#supported-tags)
- [Plugins](#plugins)
- [Dynamic Commands](#dynamic-commands)
- [Variable interpolation](#variable-interpolation)
- [Validation](#validation)
- [Modifying Kong's behaviour](#modifying-kongs-behaviour)
  - [Namehelp and Descriptionhelp - set the application name description](#namehelp-and-descriptionhelp---set-the-application-name-description)
  - [Configurationloader, paths... - load defaults from configuration files](#configurationloader-paths---load-defaults-from-configuration-files)
  - [Resolver... - support for default values from external sources](#resolver---support-for-default-values-from-external-sources)
  - [\*Mapper... - customising how the command-line is mapped to Go values](#mapper---customising-how-the-command-line-is-mapped-to-go-values)
  - [ConfigureHelpHelpOptions and HelpHelpFunc - customising help](#configurehelphelpoptions-and-helphelpfunc---customising-help)
  - [Bind... - bind values for callback hooks and Run methods](#bind---bind-values-for-callback-hooks-and-run-methods)
  - [Other options](#other-options)

<!-- /TOC -->

## Introduction

Kong aims to support arbitrarily complex command-line structures with as little developer effort as possible.

To achieve that, command-lines are expressed as Go types, with the structure and tags directing how the command line is mapped onto the struct.

For example, the following command-line:

    shell rm [-f] [-r] <paths> ...
    shell ls [<paths> ...]

Can be represented by the following command-line structure:

```go
package main

import "github.com/alecthomas/kong"

var CLI struct {
  Rm struct {
    Force     bool `help:"Force removal."`
    Recursive bool `help:"Recursively remove files."`

    Paths []string `arg:"" name:"path" help:"Paths to remove." type:"path"`
  } `cmd:"" help:"Remove files."`

  Ls struct {
    Paths []string `arg:"" optional:"" name:"path" help:"Paths to list." type:"path"`
  } `cmd:"" help:"List paths."`
}

func main() {
  ctx := kong.Parse(&CLI)
  switch ctx.Command() {
  case "rm <path>":
  case "ls":
  default:
    panic(ctx.Command())
  }
}
```

## Help

### Help as a user of a Kong application

Every Kong application includes a `--help` flag that will display auto-generated help.

eg.

    $ shell --help
    usage: shell <command>

    A shell-like example app.

    Flags:
      --help   Show context-sensitive help.
      --debug  Debug mode.

    Commands:
      rm <path> ...
        Remove files.

      ls [<path> ...]
        List paths.

If a command is provided, the help will show full detail on the command including all available flags.

eg.

    $ shell --help rm
    usage: shell rm <paths> ...

    Remove files.

    Arguments:
      <paths> ...  Paths to remove.

    Flags:
          --debug        Debug mode.

      -f, --force        Force removal.
      -r, --recursive    Recursively remove files.

### Defining help in Kong

Help is automatically generated from the command-line structure itself,
including `help:""` and other tags. [Variables](#variable-interpolation) will
also be interpolated into the help string.

Finally, any command, or argument type implementing the interface
`Help() string` will have this function called to retrieve more detail to
augment the help tag. This allows for much more descriptive text than can
fit in Go tags. [See \_examples/shell/help](./_examples/shell/help)

#### Showing the _command_'s detailed help

A command's additional help text is _not_ shown from top-level help, but can be displayed within contextual help:

**Top level help**

```bash
 $ go run ./_examples/shell/help --help
Usage: help <command>

An app demonstrating HelpProviders

Flags:
  -h, --help    Show context-sensitive help.
      --flag    Regular flag help

Commands:
  echo    Regular command help
```

**Contextual**

```bash
 $ go run ./_examples/shell/help echo --help
Usage: help echo <msg>

Regular command help

🚀 additional command help

Arguments:
  <msg>    Regular argument help

Flags:
  -h, --help    Show context-sensitive help.
      --flag    Regular flag help
```

#### Showing an _argument_'s detailed help

Custom help will only be shown for _positional arguments with named fields_ ([see the README section on positional arguments for more details on what that means](../../../README.md#branching-positional-arguments))

**Contextual argument help**

```bash
 $ go run ./_examples/shell/help msg --help
Usage: help echo <msg>

Regular argument help

📣 additional argument help

Flags:
  -h, --help    Show context-sensitive help.
      --flag    Regular flag help
```

## Command handling

There are two ways to handle commands in Kong.

### Switch on the command string

When you call `kong.Parse()` it will return a unique string representation of the command. Each command branch in the hierarchy will be a bare word and each branching argument or required positional argument will be the name surrounded by angle brackets. Here's an example:

There's an example of this pattern [here](https://github.com/alecthomas/kong/blob/master/_examples/shell/commandstring/main.go).

eg.

```go
package main

import "github.com/alecthomas/kong"

var CLI struct {
  Rm struct {
    Force     bool `help:"Force removal."`
    Recursive bool `help:"Recursively remove files."`

    Paths []string `arg:"" name:"path" help:"Paths to remove." type:"path"`
  } `cmd:"" help:"Remove files."`

  Ls struct {
    Paths []string `arg:"" optional:"" name:"path" help:"Paths to list." type:"path"`
  } `cmd:"" help:"List paths."`
}

func main() {
  ctx := kong.Parse(&CLI)
  switch ctx.Command() {
  case "rm <path>":
  case "ls":
  default:
    panic(ctx.Command())
  }
}
```

This has the advantage that it is convenient, but the downside that if you modify your CLI structure, the strings may change. This can be fragile.

### Attach a `Run(...) error` method to each command

A more robust approach is to break each command out into their own structs:

1. Break leaf commands out into separate structs.
2. Attach a `Run(...) error` method to all leaf commands.
3. Call `kong.Kong.Parse()` to obtain a `kong.Context`.
4. Call `kong.Context.Run(bindings...)` to call the selected parsed command.

Once a command node is selected by Kong it will search from that node back to the root. Each
encountered command node with a `Run(...) error` will be called in reverse order. This allows
sub-trees to be re-used fairly conveniently.

In addition to values bound with the `kong.Bind(...)` option, any values
passed through to `kong.Context.Run(...)` are also bindable to the target's
`Run()` arguments.

Finally, hooks can also contribute bindings via `kong.Context.Bind()` and `kong.Context.BindTo()`.

There's a full example emulating part of the Docker CLI [here](https://github.com/alecthomas/kong/tree/master/_examples/docker).

eg.

```go
type Context struct {
  Debug bool
}

type RmCmd struct {
  Force     bool `help:"Force removal."`
  Recursive bool `help:"Recursively remove files."`

  Paths []string `arg:"" name:"path" help:"Paths to remove." type:"path"`
}

func (r *RmCmd) Run(ctx *Context) error {
  fmt.Println("rm", r.Paths)
  return nil
}

type LsCmd struct {
  Paths []string `arg:"" optional:"" name:"path" help:"Paths to list." type:"path"`
}

func (l *LsCmd) Run(ctx *Context) error {
  fmt.Println("ls", l.Paths)
  return nil
}

var cli struct {
  Debug bool `help:"Enable debug mode."`

  Rm RmCmd `cmd:"" help:"Remove files."`
  Ls LsCmd `cmd:"" help:"List paths."`
}

func main() {
  ctx := kong.Parse(&cli)
  // Call the Run() method of the selected parsed command.
  err := ctx.Run(&Context{Debug: cli.Debug})
  ctx.FatalIfErrorf(err)
}

```

## Hooks: BeforeReset(), BeforeResolve(), BeforeApply(), AfterApply() and the Bind() option

If a node in the grammar has a `BeforeReset(...)`, `BeforeResolve
(...)`, `BeforeApply(...) error` and/or `AfterApply(...) error` method, those
methods will be called before values are reset, before validation/assignment,
and after validation/assignment, respectively.

The `--help` flag is implemented with a `BeforeReset` hook.

Arguments to hooks are provided via the `Run(...)` method or `Bind(...)` option. `*Kong`, `*Context` and `*Path` are also bound and finally, hooks can also contribute bindings via `kong.Context.Bind()` and `kong.Context.BindTo()`.

eg.

```go
// A flag with a hook that, if triggered, will set the debug loggers output to stdout.
type debugFlag bool

func (d debugFlag) BeforeApply(logger *log.Logger) error {
  logger.SetOutput(os.Stdout)
  return nil
}

var cli struct {
  Debug debugFlag `help:"Enable debug logging."`
}

func main() {
  // Debug logger going to discard.
  logger := log.New(ioutil.Discard, "", log.LstdFlags)

  ctx := kong.Parse(&cli, kong.Bind(logger))

  // ...
}
```

Another example of using hooks is load the env-file:

```go
package main

import (
  "fmt"
  "github.com/alecthomas/kong"
  "github.com/joho/godotenv"
)

type EnvFlag string

// BeforeResolve loads env file.
func (c EnvFlag) BeforeReset(ctx *kong.Context, trace *kong.Path) error {
  path := string(ctx.FlagValue(trace.Flag).(EnvFlag)) // nolint
  path = kong.ExpandPath(path)
  if err := godotenv.Load(path); err != nil {
    return err
  }
  return nil
}

var CLI struct {
  EnvFile EnvFlag
  Flag `env:"FLAG"`
}

func main() {
  _ = kong.Parse(&CLI)
  fmt.Println(CLI.Flag)
}
```

## Flags

Any [mapped](#mapper---customising-how-the-command-line-is-mapped-to-go-values) field in the command structure _not_ tagged with `cmd` or `arg` will be a flag. Flags are optional by default.

eg. The command-line `app [--flag="foo"]` can be represented by the following.

```go
type CLI struct {
  Flag string
}
```

## Commands and sub-commands

Sub-commands are specified by tagging a struct field with `cmd`. Kong supports arbitrarily nested commands.

eg. The following struct represents the CLI structure `command [--flag="str"] sub-command`.

```go
type CLI struct {
  Command struct {
    Flag string

    SubCommand struct {
    } `cmd`
  } `cmd`
}
```

If a sub-command is tagged with `default:"1"` it will be selected if there are no further arguments. If a sub-command is tagged with `default:"withargs"` it will be selected even if there are further arguments or flags and those arguments or flags are valid for the sub-command. This allows the user to omit the sub-command name on the CLI if its arguments/flags are not ambiguous with the sibling commands or flags.

## Branching positional arguments

In addition to sub-commands, structs can also be configured as branching positional arguments.

This is achieved by tagging an [unmapped](#mapper---customising-how-the-command-line-is-mapped-to-go-values) nested struct field with `arg`, then including a positional argument field inside that struct _with the same name_. For example, the following command structure:

    app rename <name> to <name>

Can be represented with the following:

```go
var CLI struct {
  Rename struct {
    Name struct {
      Name string `arg` // <-- NOTE: identical name to enclosing struct field.
      To struct {
        Name struct {
          Name string `arg`
        } `arg`
      } `cmd`
    } `arg`
  } `cmd`
}
```

This looks a little verbose in this contrived example, but typically this will not be the case.

## Positional arguments

If a field is tagged with `arg:""` it will be treated as the final positional
value to be parsed on the command line. By default positional arguments are
required, but specifying `optional:""` will alter this.

If a positional argument is a slice, all remaining arguments will be appended
to that slice.

## Slices

Slice values are treated specially. First the input is split on the `sep:"<rune>"` tag (defaults to `,`), then each element is parsed by the slice element type and appended to the slice. If the same value is encountered multiple times, elements continue to be appended.

To represent the following command-line:

    cmd ls <file> <file> ...

You would use the following:

```go
var CLI struct {
  Ls struct {
    Files []string `arg:"" type:"existingfile"`
  } `cmd`
}
```

## Maps

Maps are similar to slices except that only one key/value pair can be assigned per value, and the `sep` tag denotes the assignment character and defaults to `=`.

To represent the following command-line:

    cmd config set <key>=<value> <key>=<value> ...

You would use the following:

```go
var CLI struct {
  Config struct {
    Set struct {
      Config map[string]float64 `arg:"" type:"file:"`
    } `cmd`
  } `cmd`
}
```

For flags, multiple key+value pairs should be separated by `mapsep:"rune"` tag (defaults to `;`) eg. `--set="key1=value1;key2=value2"`.

## Pointers

Pointers work like the underlying type, except that you can differentiate between the presence of the zero value and no value being supplied.

For example:

```go
var CLI struct {
	Foo *int
}
```

Would produce a nil value for `Foo` if no `--foo` argument is supplied, but would have a pointer to the value 0 if the argument `--foo=0` was supplied.

## Nested data structure

Kong support a nested data structure as well with `embed:""`. You can combine `embed:""` with `prefix:""`:

```go
var CLI struct {
  Logging struct {
    Level string `enum:"debug,info,warn,error" default:"info"`
    Type string `enum:"json,console" default:"console"`
  } `embed:"" prefix:"logging."`
}
```

This configures Kong to accept flags `--logging.level` and `--logging.type`.

## Custom named decoders

Kong includes a number of builtin custom type mappers. These can be used by
specifying the tag `type:"<type>"`. They are registered with the option
function `NamedMapper(name, mapper)`.

| Name           | Description                                                                                                            |
| -------------- | ---------------------------------------------------------------------------------------------------------------------- |
| `path`         | A path. ~ expansion is applied. `-` is accepted for stdout, and will be passed unaltered.                              |
| `existingfile` | An existing file. ~ expansion is applied. `-` is accepted for stdin, and will be passed unaltered.                     |
| `existingdir`  | An existing directory. ~ expansion is applied.                                                                         |
| `counter`      | Increment a numeric field. Useful for `-vvv`. Can accept `-s`, `--long` or `--long=N`.                                 |
| `filecontent`  | Read the file at path into the field. ~ expansion is applied. `-` is accepted for stdin, and will be passed unaltered. |

Slices and maps treat type tags specially. For slices, the `type:""` tag
specifies the element type. For maps, the tag has the format
`tag:"[<key>]:[<value>]"` where either may be omitted.

## Supported field types

## Custom decoders (mappers)

Any field implementing `encoding.TextUnmarshaler` or `json.Unmarshaler` will use those interfaces
for decoding values. Kong also includes builtin support for many common Go types:

| Type            | Description                                                                                                 |
| --------------- | ----------------------------------------------------------------------------------------------------------- |
| `time.Duration` | Populated using `time.ParseDuration()`.                                                                     |
| `time.Time`     | Populated using `time.Parse()`. Format defaults to RFC3339 but can be overridden with the `format:"X"` tag. |
| `*os.File`      | Path to a file that will be opened, or `-` for `os.Stdin`. File must be closed by the user.                 |
| `*url.URL`      | Populated with `url.Parse()`.                                                                               |

For more fine-grained control, if a field implements the
[MapperValue](https://godoc.org/github.com/alecthomas/kong#MapperValue)
interface it will be used to decode arguments into the field.

## Supported tags

Tags can be in two forms:

1. Standard Go syntax, eg. `kong:"required,name='foo'"`.
2. Bare tags, eg. `required:"" name:"foo"`

Both can coexist with standard Tag parsing.

| Tag                  | Description                                                                                                                                                                                                                                                                                                                    |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `cmd:""`             | If present, struct is a command.                                                                                                                                                                                                                                                                                               |
| `arg:""`             | If present, field is an argument. Required by default.                                                                                                                                                                                                                                                                         |
| `env:"X,Y,..."`      | Specify envars to use for default value. The envs are resolved in the declared order. The first value found is used.                                                                                                                                                                                                           |
| `name:"X"`           | Long name, for overriding field name.                                                                                                                                                                                                                                                                                          |
| `help:"X"`           | Help text.                                                                                                                                                                                                                                                                                                                     |
| `type:"X"`           | Specify [named types](#custom-named-decoders) to use.                                                                                                                                                                                                                                                                          |
| `placeholder:"X"`    | Placeholder text.                                                                                                                                                                                                                                                                                                              |
| `default:"X"`        | Default value.                                                                                                                                                                                                                                                                                                                 |
| `default:"1"`        | On a command, make it the default.                                                                                                                                                                                                                                                                                             |
| `default:"withargs"` | On a command, make it the default and allow args/flags from that command                                                                                                                                                                                                                                                       |
| `short:"X"`          | Short name, if flag.                                                                                                                                                                                                                                                                                                           |
| `aliases:"X,Y"`      | One or more aliases (for cmd).                                                                                                                                                                                                                                                                                                 |
| `required:""`        | If present, flag/arg is required.                                                                                                                                                                                                                                                                                              |
| `optional:""`        | If present, flag/arg is optional.                                                                                                                                                                                                                                                                                              |
| `hidden:""`          | If present, command or flag is hidden.                                                                                                                                                                                                                                                                                         |
| `negatable:""`       | If present on a `bool` field, supports prefixing a flag with `--no-` to invert the default value                                                                                                                                                                                                                               |
| `format:"X"`         | Format for parsing input, if supported.                                                                                                                                                                                                                                                                                        |
| `sep:"X"`            | Separator for sequences (defaults to ","). May be `none` to disable splitting.                                                                                                                                                                                                                                                 |
| `mapsep:"X"`         | Separator for maps (defaults to ";"). May be `none` to disable splitting.                                                                                                                                                                                                                                                      |
| `enum:"X,Y,..."`     | Set of valid values allowed for this flag. An enum field must be `required` or have a valid `default`.                                                                                                                                                                                                                         |
| `group:"X"`          | Logical group for a flag or command.                                                                                                                                                                                                                                                                                           |
| `xor:"X,Y,..."`      | Exclusive OR groups for flags. Only one flag in the group can be used which is restricted within the same command. When combined with `required`, at least one of the `xor` group will be required.                                                                                                                            |
| `prefix:"X"`         | Prefix for all sub-flags.                                                                                                                                                                                                                                                                                                      |
| `envprefix:"X"`      | Envar prefix for all sub-flags.                                                                                                                                                                                                                                                                                                |
| `set:"K=V"`          | Set a variable for expansion by child elements. Multiples can occur.                                                                                                                                                                                                                                                           |
| `embed:""`           | If present, this field's children will be embedded in the parent. Useful for composition.                                                                                                                                                                                                                                      |
| `passthrough:""`     | If present on a positional argument, it stops flag parsing when encountered, as if `--` was processed before. Useful for external command wrappers, like `exec`. On a command it requires that the command contains only one argument of type `[]string` which is then filled with everything following the command, unparsed. |
| `-`                  | Ignore the field. Useful for adding non-CLI fields to a configuration struct. e.g `` `kong:"-"` ``                                                                                                                                                                                                                             |

## Plugins

Kong CLI's can be extended by embedding the `kong.Plugin` type and populating it with pointers to Kong annotated structs. For example:

```go
var pluginOne struct {
  PluginOneFlag string
}
var pluginTwo struct {
  PluginTwoFlag string
}
var cli struct {
  BaseFlag string
  kong.Plugins
}
cli.Plugins = kong.Plugins{&pluginOne, &pluginTwo}
```

Additionally if an interface type is embedded, it can also be populated with a Kong annotated struct.

## Dynamic Commands

While plugins give complete control over extending command-line interfaces, Kong
also supports dynamically adding commands via `kong.DynamicCommand()`.

## Variable interpolation

Kong supports limited variable interpolation into help strings, enum lists and
default values.

Variables are in the form:

    ${<name>}
    ${<name>=<default>}

Variables are set with the `Vars{"key": "value", ...}` option. Undefined
variable references in the grammar without a default will result in an error at
construction time.

Variables can also be set via the `set:"K=V"` tag. In this case, those variables will be available for that
node and all children. This is useful for composition by allowing the same struct to be reused.

When interpolating into flag or argument help strings, some extra variables
are defined from the value itself:

    ${default}
    ${enum}

For flags with associated environment variables, the variable `${env}` can be
interpolated into the help string. In the absence of this variable in the
help string, Kong will append `($$${env})` to the help string.

eg.

```go
type cli struct {
  Config string `type:"path" default:"${config_file}"`
}

func main() {
  kong.Parse(&cli,
    kong.Vars{
      "config_file": "~/.app.conf",
    })
}
```

## Validation

Kong does validation on the structure of a command-line, but also supports
extensible validation. Any node in the tree may implement the following
interface:

```go
type Validatable interface {
    Validate() error
 }
```

If one of these nodes is in the active command-line it will be called during
normal validation.

## Modifying Kong's behaviour

Each Kong parser can be configured via functional options passed to `New(cli interface{}, options...Option)`.

The full set of options can be found [here](https://godoc.org/github.com/alecthomas/kong#Option).

### `Name(help)` and `Description(help)` - set the application name description

Set the application name and/or description.

The name of the application will default to the binary name, but can be overridden with `Name(name)`.

As with all help in Kong, text will be wrapped to the terminal.

### `Configuration(loader, paths...)` - load defaults from configuration files

This option provides Kong with support for loading defaults from a set of configuration files. Each file is opened, if possible, and the loader called to create a resolver for that file.

eg.

```go
kong.Parse(&cli, kong.Configuration(kong.JSON, "/etc/myapp.json", "~/.myapp.json"))
```

[See the tests](https://github.com/alecthomas/kong/blob/master/resolver_test.go#L206) for an example of how the JSON file is structured.

#### List of Configuration Loaders

- [YAML](https://github.com/alecthomas/kong-yaml)
- [HCL](https://github.com/alecthomas/kong-hcl)
- [TOML](https://github.com/alecthomas/kong-toml)
- [JSON](https://github.com/alecthomas/kong)

### `Resolver(...)` - support for default values from external sources

Resolvers are Kong's extension point for providing default values from external sources. As an example, support for environment variables via the `env` tag is provided by a resolver. There's also a builtin resolver for JSON configuration files.

Example resolvers can be found in [resolver.go](https://github.com/alecthomas/kong/blob/master/resolver.go).

### `*Mapper(...)` - customising how the command-line is mapped to Go values

Command-line arguments are mapped to Go values via the Mapper interface:

```go
// A Mapper represents how a field is mapped from command-line values to Go.
//
// Mappers can be associated with concrete fields via pointer, reflect.Type, reflect.Kind, or via a "type" tag.
//
// Additionally, if a type implements the MapperValue interface, it will be used.
type Mapper interface {
	// Decode ctx.Value with ctx.Scanner into target.
	Decode(ctx *DecodeContext, target reflect.Value) error
}
```

All builtin Go types (as well as a bunch of useful stdlib types like `time.Time`) have mappers registered by default. Mappers for custom types can be added using `kong.??Mapper(...)` options. Mappers are applied to fields in four ways:

1. `NamedMapper(string, Mapper)` and using the tag key `type:"<name>"`.
2. `KindMapper(reflect.Kind, Mapper)`.
3. `TypeMapper(reflect.Type, Mapper)`.
4. `ValueMapper(interface{}, Mapper)`, passing in a pointer to a field of the grammar.

### `ConfigureHelp(HelpOptions)` and `Help(HelpFunc)` - customising help

The default help output is usually sufficient, but if not there are two solutions.

1. Use `ConfigureHelp(HelpOptions)` to configure how help is formatted (see [HelpOptions](https://godoc.org/github.com/alecthomas/kong#HelpOptions) for details).
2. Custom help can be wired into Kong via the `Help(HelpFunc)` option. The `HelpFunc` is passed a `Context`, which contains the parsed context for the current command-line. See the implementation of `PrintHelp` for an example.
3. Use `ValueFormatter(HelpValueFormatter)` if you want to just customize the help text that is accompanied by flags and arguments.
4. Use `Groups([]Group)` if you want to customize group titles or add a header.

### `Bind(...)` - bind values for callback hooks and Run() methods

See the [section on hooks](#hooks-beforeresolve-beforeapply-afterapply-and-the-bind-option) for details.

### Other options

The full set of options can be found [here](https://godoc.org/github.com/alecthomas/kong#Option).
