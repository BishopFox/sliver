
## Introduction & Notes

The documentation and examples pictures have been realized with a [Sliver](https://github.com/BishopFox/sliver) 
client console, so that one can see what can be done with the readline library. Huge thanks to Sliver maintainers for their agreement !
As well, the subset of commands/options/arguments bound the example console in the `examples/commands.go` file is from the Sliver project.

In order to use the readline library at the best of its possibilities and with the smallest number of problems, **it is strongly advised**
to read the documentation in the order below: **it will guide you step-by-step, from instantiating a shell into your project all the way to
adding completion groups, going through prompt systems, etc**, in between. Have fun !


-----
## Table of Contents 

#### Getting started
* [ Embedding readline in a project ](https://github.com/bishopfox/sliver/client/readline/wiki/Embedding-Readline-In-A-Project)
* [ Input Modes ](https://github.com/bishopfox/sliver/client/readline/wiki/Input-Modes)
* [ Command History ](https://github.com/bishopfox/sliver/client/readline/wiki/Command-History)

### Prompt system
* [ Setting the Prompts](https://github.com/bishopfox/sliver/client/readline/wiki/Prompt-Setup)
* [ Prompt Refresh ](https://github.com/bishopfox/sliver/client/readline/wiki/Prompt-Refresh)

### Completion Engine 
* [ Completion Groups ](https://github.com/bishopfox/sliver/client/readline/wiki/Completion-Groups)
* [ Completion Search & Movements ](https://github.com/bishopfox/sliver/client/readline/wiki/Completion-Search)
* [ Writing a Completer ](https://github.com/bishopfox/sliver/client/readline/wiki/Writing-A-Completer)

### Hint Formatter & Syntax Highlighter 
* [ Live Refresh Demonstration ](https://github.com/bishopfox/sliver/client/readline/wiki/Live-Refresh-Demonstration)

### Command & Completion utilities
* [ Interfacing with the go-flags library](https://github.com/bishopfox/sliver/client/readline/wiki/Interfacing-With-Go-Flags)
* [ Default Completion Engine (with go-flags) ](https://github.com/bishopfox/sliver/client/readline/wiki/Default-Completion-Engine)
* [ Colors/Effects Usage ](https://github.com/bishopfox/sliver/client/readline/wiki/Colors-&-Effects-Usage)


-----
## File Contents

Below is a summary of the content for most files. The list is not complete, as I did not have the occasion yet to dig in some of them. The list is structured by topic/domain.

#### Readline Shell, Prompt and Main loop
* `instance.go`     - Contains the `Instance` type, which is the Shell itself, and all its operating parameters.
* `readline.go`     - The readline main loop.
* `prompt.go`       - Functions computing the prompts at each loop, or on demand with custom behavior with the `RefreshPrompt()`  function.

#### Vim editing
* `vim.go`          - Processes all keystrokes used in Vim input mode, for all Vim editing modes.
* `vimdelete.go`    - Equivalent code for deleting items of the line in Vim Delete mode.
* `undo.go`         - Handles all undo operations in Vim editing mode.

#### Completion engine
* `tab.go`          - The central dispatcher for all completion requests and handlers, for all completion modes.
* `tab-virtual.go`  - Handles insertion and replacement of the currently selected completion candidate, in the input line.
* `tabfind.go`      - Handles everything related to searching completions (with Ctrl-F)
* `comp-group.go`   - Definition of the `Completiongroup` type, and all methods used by completion groups, regardless of their display.
* `comp-grid.go`    - Initialization, movement and printing for **Grid** display completion groups.
* `comp-list.go`    - Initialization, movement and printing for **List** display completion groups.
* `comp-map.go`     - Initialization, movement and printing for **Map** display completion groups.

#### Hints & syntax
* `hint.go`         - Handles printing and refreshing the console hints.
* `syntax.go`       - Handles syntax parsing of the input line.

#### Helpers
* `update.go`       - All helpers used to refresh input line, completion/hint printing, and cursor positions.
* `tokenise.go`     - Splits and processes the input line. Used by Vim editing code, as well as virtual completion helpers.

#### Codes, Colors & Effects
* `codes.go`        - All key, colors, effects and other terminal sequences. Also contains TUI colors and related methods.

#### Others
* `history.go`      - Defines the `History` type, and handles history completion/refreshing and writing, for main and alternative sources.
