
<div align="center">
  <br> <h1> Console </h1>

  <p>  Closed-loop application library for Cobra commands  </p>
  <p>  (ready-to-use menus, prompts, completions, and more)  </p>
</div>


<!-- Badges -->
<p align="center">
  <a href="https://github.com/reeflective/console/actions/workflows/go.yml">
    <img src="https://github.com/reeflective/console/actions/workflows/go.yml/badge.svg?branch=main"
      alt="Github Actions (workflows)" />
  </a>

  <a href="https://github.com/reeflective/console">
    <img src="https://img.shields.io/github/go-mod/go-version/reeflective/console.svg"
      alt="Go module version" />
  </a>

  <a href="https://pkg.go.dev/github.com/reeflective/console">
    <img src="https://img.shields.io/badge/godoc-reference-blue.svg"
      alt="GoDoc reference" />
  </a>

  <a href="https://goreportcard.com/report/github.com/reeflective/console">
    <img src="https://goreportcard.com/badge/github.com/reeflective/console"
      alt="Go Report Card" />
  </a>

  <a href="https://codecov.io/gh/reeflective/console">
    <img src="https://codecov.io/gh/reeflective/console/branch/main/graph/badge.svg"
      alt="codecov" />
  </a>

  <a href="https://opensource.org/licenses/BSD-3-Clause">
    <img src="https://img.shields.io/badge/License-BSD_3--Clause-blue.svg"
      alt="License: BSD-3" />
  </a>
</p>

Console is an all-in-one console application library built on top of a [readline](https://github.com/reeflective/readline) shell and using [Cobra](https://github.com/spf13/cobra) commands. 
It aims to provide users with a modern interface at at minimal cost while allowing them to focus on developing 
their commands and application core: the console will then transparently interface with these commands, and provide
the various features below almost for free.


## Features

### Menus & Commands 
- Bind cobra commands to provide the core functionality.
- Multiple menus with their own command tree, prompt engines and special handlers.
- All cobra settings can be modified, set and used freely, like in normal CLI workflows.
- Bind handlers to special interrupt errors (eg. `CtrlC`/`CtrlD`), per menu.

### Shell interface
- Shell is powered by a [readline](https://github.com/reeflective/readline) instance, with full `inputrc` support and extended functionality.
- All features of readline are supported in the console. It also allows the console to give:
- Configurable bind keymaps, commands and options, sane defaults, and per-application configuration.
- Out-of-the-box, advanced completions for commands, flags, positional and flag arguments.
- Provided by readline and [carapace](https://github.com/rsteube/carapace): automatic usage & validation command/flags/args hints.
- Syntax highlighting for commands (might be extended in the future).

### Others
- Support for an arbitrary number of history sources, per menu.
- Support for [oh-my-posh](https://github.com/JanDeDobbeleer/oh-my-posh) prompts, per menu and with custom configuration files for each.
- Also with oh-my-posh, write and bind application/menu-specific prompt segments.
- Set of ready-to-use commands (`commands/` directory) for readline binds/options manipulation.


## Documentation

You can install and use the [example application console](https://github.com/reeflective/console/tree/main/example). This example application 
will give you a taste of the behavior and supported features. The following documentation 
is also available in the [wiki](https://github.com/reeflective/console/wiki):

* [Getting started](https://github.com/reeflective/console/wiki/Getting-Started) 
* [Menus](https://github.com/reeflective/console/wiki/Menus)
* [Prompts](https://github.com/reeflective/console/wiki/Prompts)
* [Binding commands](https://github.com/reeflective/console/wiki/Binding-Commands)
* [Interrupt handlers](https://github.com/reeflective/console/wiki/Interrupt-Handlers)
* [History Sources](https://github.com/reeflective/console/wiki/History-Sources)
* [Logging](https://github.com/reeflective/console/wiki/Logging)
* [Readline shell](https://github.com/reeflective/readline/wiki)
* [Other utilities](https://github.com/reeflective/console/wiki/Other-Utililites)


## Showcase
![console](https://github.com/reeflective/console/blob/assets/console.gif)


## Status 

The library is in a pre-release candidate status:
- Although quite simple and small, it has not been tested heavily.
- There are probably some features/improvements to be made.
- The API is quite stable. It is unlikely to change much in future versions.

Please open a PR or an issue if you wish to bring enhancements to it. 
Other contributions, as well as bug fixes and reviews are also welcome.


## Possible Improvements

The following is a currently moving list of possible enhancements to be made in order to reach `v1.0`:
- [ ] Ensure to the best extent possible a thread-safe access to the command API.
- [ ] Clearer integration/alignment of the various I/O references between readline and commands.
- [ ] Clearer and sane model for asynchronous control/cancel of commands.
- [ ] Allow users to run the console command trees in one-exec style, with identical behavior.
- [ ] Test suite for most important or risky code paths.
- [ ] Set of helper functions for application-related directories.
