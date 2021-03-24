## Changes


### 4.0.0-beta
---------

This version is the merge of [maxlandon/readline](https://github.com/maxlandon/readline) 
and [lmorg/readline](https://github.com/lmorg/readline). Therefore it both integrates parts
from both libraries, but also adds a few features, with some API breaking changes (ex: completions),
thus the new 4.0.0 version. Remains a beta because maxlandon/readline code has not been thoroughly
test neither in nor of itself, and no more against `lmorg/murex`, it's main consumer until now.

#### Code
- Enhance delete/copy buffer in Vim mode
- DelayedTabContext now works with completion groups

#### Packages
- Added a `completers` package, with a default tab/hint/syntax completer working with 
 the [go-flags](https://github.com/jessevdk/go-flags) library.
- The `examples` package has been enhanced with a more complete -base- application code. See the wiki

#### Documentation 
- Merged relevant parts of both READMEs
- Use documentation from maxlandon/readline

#### New features / bindings
- CtrlL now clears the screen and reprints the prompt
- Added evilsocket's tui colors/effects, for ease of use & integration with shell. Has not yet replaced the current `seqColor` variables everywhere though

#### Changes I'm not sure of
- is the function leftMost() in cursor.go useful ?
- is the function getCursorPos() in cursor.go useful ?


### 3.0.0
---------

- Added test (input line, prompt, correct refresh, etc)
- Added multiline support
- Added `DelayedTabContext` and `DelayedSyntaxWorker`


### 2.1.0
---------

Error returns from `readline` have been created as error a variable, which is
more idiomatic to Go than the err constants that existed previously. Currently
both are still available to use however I will be deprecating the the constants
in a latter release.

**Deprecated constants:**
```go
const (
	// ErrCtrlC is returned when ctrl+c is pressed
	ErrCtrlC = "Ctrl+C"

	// ErrEOF is returned when ctrl+d is pressed
	ErrEOF = "EOF"
)
```

**New error variables:**
```go
var (
	// CtrlC is returned when ctrl+c is pressed
	CtrlC = errors.New("Ctrl+C")

	// EOF is returned when ctrl+d is pressed
	// (this is actually the same value as io.EOF)
	EOF = errors.New("EOF")
)
```

## Version Information

`readline`'s version numbers are based on Semantic Versioning. More details can
be found in the [README.md](README.md#version-information).
