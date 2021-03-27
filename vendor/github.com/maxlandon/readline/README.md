
# Readline - Console library in Go

![Demo](../assets/readline-demo.gif)
*This demo GIF has been made with a Sliver project client.*


## Introduction

**This project is actually the merging of an original project (github.com/lmorg/readline) and one of its
forks (github.com/maxlandon/readline): both introductions are thus here given, in chronological order.**

#### lmorg

This project began a few years prior to this git commit history as an API for
_[murex](https://github.com/lmorg/murex)_, an alternative UNIX shell, because
I wasn't satisfied with the state of existing Go packages for readline (at that
time they were either bugger and/or poorly maintained, or lacked features I
desired). The state of things for readline in Go may have changed since then
however own package had also matured and grown to include many more features
that has arisen during the development of _murex_. So it seemed only fair to
give back to the community considering it was other peoples readline libraries
that allowed me rapidly prototype _murex_ during it's early stages of
development.

#### maxlandon

This project started out of the wish to make an enhanced console for a security tool (Sliver, see below).
There are already several readline libraries available in Go ([github.com/chzyer/readline](https://github.com/chzyer/readline), 
and [github.com/lmorg/readline](https://github.com/lmorg/readline)), but being stricter readline implementations, their completion 


## Features Summary

This project is not an integrated REPL/command-line application, which means it does not automatically understand nor executes any commands.
However, having been developed in a project using the CLI [github.com/jessevdk/go-flags](https://github.com/jessevdk/go-flags) library,
it also includes some default utilities (completers) that are made to work with this library, which I humbly but highly recommend.
Please see the [Wiki](https://github.com/bishopfox/sliver/client/readline/wiki) (or the `examples/` directory) for information on how to use these utilities.

A summarized list of features supported by this library is the following:

### Input & Editing 
- Vim / Emacs input and editing modes.
- Optional, live-refresh Vim status.
- line editing using `$EDITOR` (`vi` in the example - enabled by pressing `[ESC]` followed by `[v]`)
- `readline`'s warning before pasting multiple lines of data into the buffer

### Completion engine
- 3 types of completion categories (`Grid`, `List` and `Map`)
- Stackable, combinable completions (completion groups of any type & size can be proposed simultaneously).
- Controlable completion group sizes (if size is greater than completions, the completions will roll automatically)
- Virtual insertion of the current candidate, like in Z-shell.
- In `List` completion groups, ability to have alternative candidates (used for displaying `--long` and `-s` (short) options, with descriptions)
- Completions working anywhere in the input line (your cursor can be anywhere)
- Completions are searchable with *Ctrl-F*, like in lmorg's library.

### Prompt system & Colors
- 1-line and 2-line prompts, both being customizable.
- Function for refreshing the prompt, with optional behavior settings.
- Optional colors (can be disabled).

### Hints & Syntax highlighting
- A hint line can be printed below the input line, with any type of information. See utilities for a default one.
- The Hint system is now refreshed depending on the cursor position as well, like completions.
- A syntax highlighting system. A default one is also available.

### Command history
- Ability to have 2 different history sources (I used this for clients connected to a server, used by a single user).
- History is searchable like completions.
- Default history is an in-memory list.
- Quick history navigation with *Up*/*Down* arrow keys in Emacs mode, and *j*/*k* keys in Vim mode.

### Utilities
- Default Tab completer, Hint formatter and Syntax highlighter provided, using [github.com/jessevdk/go-flags](https://github.com/jessevdk/go-flags) 
command parser to build themselves. These are in the  `completers/` directory. Please look at the [Wiki page](https://github.com/bishopfox/sliver/client/readline/wiki) 
for how to use them. Also feel free to use them as an inspiration source to make your owns.
- Colors mercilessly copied from [github.com/evilsocket/islazy/](https://github.com/evilsocket/islazy) `tui/` package.
- Also in the `completers` directory, completion functions for environment variables (using Go's std lib for getting them), and dir/file path completions.


## Installation & Usage

As usual with Go, installation:
```
go get github.com/bishopfox/sliver/client/readline
```
Please see either the `examples` directory, or the Wiki for detailed instructions on how to use this library.


## Documentation

The complete documentation for this library can be found in the repo's [Wiki](https://github.com/bishopfox/sliver/client/readline/wiki). Below is the Table of Contents:

**Getting started**
* [ Embedding readline in a project ](https://github.com/bishopfox/sliver/client/readline/wiki/Embedding-Readline-In-A-Project)
* [ Input Modes ](https://github.com/bishopfox/sliver/client/readline/wiki/Input-Modes)

**Prompt system**
* [ Setting the Prompts](https://github.com/bishopfox/sliver/client/readline/wiki/Prompt-Setup)
* [ Prompt Refresh ](https://github.com/bishopfox/sliver/client/readline/wiki/Prompt-Refresh)

**Completion Engine**
* [ Completion Groups ](https://github.com/bishopfox/sliver/client/readline/wiki/Completion-Groups)
* [ Completion Search ](https://github.com/bishopfox/sliver/client/readline/wiki/Completion-Search)

**Hint Formatter & Syntax Highlighter**
* [ Live Refresh Demonstration ](https://github.com/bishopfox/sliver/client/readline/wiki/Live-Refresh-Demonstration)

**Command History**
* [ Main & Alternative Sources ](https://github.com/bishopfox/sliver/client/readline/wiki/Main-&-Alternative-Sources)
* [ Navigation & Search ](https://github.com/bishopfox/sliver/client/readline/wiki/Navigation-&-Search)

#### Command & Completion utilities
* [ Interfacing with the go-flags library](https://github.com/bishopfox/sliver/client/readline/wiki/Interfacing-With-Go-Flags)
* [ Declaring go-flags commands](https://github.com/bishopfox/sliver/client/readline/wiki/Declaring-Commands)
* [ Colors/Effects Usage ](https://github.com/bishopfox/sliver/client/readline/wiki/Colors-&-Effects-Usage)


## Project Status & Support

Being alone working on this project and having only one lifetime (anyone able to solve this please call me), I can engage myself over the following:
- Support for any issue opened.
- Answering any questions related.
- Being available for any blame you'd like to make for my humble but passioned work. I don't mind, I need to go up.


## Fixes & Planned Enhancements

### Completers package (working with go-flags)

Things I do intend to add in a more or less foreseeable future:
- A better recursive command/subcommand default completer (see utilities), because the current one supports only `command subcommand --options` patterns, not `command subcommand subsubcommand`.
- A recursive option group completer: tools like `nmap` will use options like `-PA`, or `-sT`, etc. These are not supported.


## Version Information

Because the last thing a developer wants is to do is fix breaking changes after
updating modules, I will make the following guarantees:

* The version string will be based on Semantic Versioning. ie version numbers
  will be formatted `(major).(minor).(patch)` - for example `2.0.1`

* `major` releases _will_ have breaking changes. Be sure to read CHANGES.md for
  upgrade instructions

* `minor` releases will contain new APIs or introduce new user facing features
  which may affect useability from an end user perspective. However `minor`
  releases will not break backwards compatibility at the source code level and
  nor will it break existing expected user-facing behavior. These changes will
  be documented in CHANGES.md too

* `patch` releases will be bug fixes and such like. Where the code has changed
  but both API endpoints and user experience will remain the same (except where
  expected user experience was broken due to a bug, then that would be bumped
  to either a `minor` or `major` depending on the significance of the bug and
  the significance of the change to the user experience)

* Any updates to documentation, comments within code or the example code will
  not result in a version bump because they will not affect the output of the
  go compiler. However if this concerns you then I recommend pinning your
  project to the git commit hash rather than a `patch` release

My recommendation is to pin to either the `minor` or `patch` release and I will
endeavour to keep breaking changes to an absolute minimum.


## License Information

The `readline` library is distributed under the Apache License (Version 2.0, January 2004) (http://www.apache.org/licenses/). 
All the example code and documentation in `/examples`, `/completers` is public domain.


## Warmest Thanks

- The [Sliver](https://github.com/BishopFox/sliver) implant framework project, which I used as a basis to make, test and refine this library. as well as all the GIFs and documentation pictures !
- [evilsocket](https://github.com/evilsocket) for his TUI library !

