## Changes

### 4.1.0
---------

Many new features and improvements in this version:
- New keybindings (working on Emacs, and in `Vim Insert Mode`):
    * `CtrlW` to cut the previous word at the cursor
    * `CtrlA` to go back to the beginning of the line
    * `CtrlY` to paste the laste copy/paste buffer (see Registers)
    * `CtrlU` to cut the whole line.

- More precise Vim iterations:
    * Iterations can now be applied to some Vim actions (`y4w`, `d3b`)

- Implemented Vim registers:
    * Yank/paste operations of any sort can occur and be assigned to registers.
    * The default `""` register
    * 10 numbered registers, to which bufffers are automatically added
    * 26 lettered registers (lowercase), to which you can append with `"D` (D being the uppercase of the `"d` register)
    * Triggered in Insert Mode with `Alt"` (buggy sometimes: goes back to Normal mode selecting a register, will have to fix this)

- Unified iterations and registers:
    * To copy to the `d` register the next 4 words: `"d y4w`
    * To append to this `d` register the cuttend end of line: `"D d$"`
    * In this example, the `d` register buffer is also the buffer in the default register `""`
    * You could either:
        - Paste 3 times this buffer while in Normal mode: `3p`
        - Paste the buffer once in Insert mode: `CtrlY`

- History completions:
    * The binding for the alternative history changed to `AltR` (the normal remains `CtrlR`)
    * By defaul the history filters only against the search pattern.
    * If there are matches for this patten, the first occurence is insert (virtually)
    * This is refreshed as the pattern changes
    * `CtrlG` to exit the comps, while leaving the current candidate 
    * `CtrlC` to exit and delete the current candidate

- Completions:
    * When a candidate is inserted virtually, `CtrlC` to abort both completions and the candidate
    * Implemented global printing size: If the overall number of completions is biffer, will roll over them.

**Notes:**
    * The `rl.Readline()` function dispatch has some big cases, maybe a bit of refactoring would be nice 
    * The way the buffer storing bytes from key strokes sometimes gives weird results (like `Alt"` for showing Vim registers)
    * Some defer/cancel calls related to DelayedTabContext that should have been merged from lmorg/readline are still missing.


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
