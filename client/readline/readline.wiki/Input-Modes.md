
## Input Modes

Two different input (or editing) modes are available: the usual suspects, Vim or Emacs.

------
## Vim input mode

This will make the shell to recognize Vim-bindings (the same way other known shells -ZSH between others- do).

```go
console.InputMode = readline.Vim
```

The shell understands the following Vim editing modes:
- `INSERT`
- `NORMAL`
- `REPLACE` (once)
- `REPLACE` (many)
- `DELETE`

### Showing/Hiding the current Vim status

By default, Vim status will be printed in the prompt (either at beginning if 2-line prompt, or at the end if 1-line).
To set/unset it:
```go
c.shell.ShowVimMode = true
c.shell.ShowVimMode = false 
```

You can also require the different editing modes statuses to be colorized with the following (colors cannot be customized at the moment):
```go
c.shell.VimModeColorize = true
```

As explained in the various [Prompts](https://github.com/bishopfox/sliver/client/readline/wiki/Vim-Prompt) sections, you can combine any of 
these parameters with custom prompt strings, and for either 1-line or 2-line prompts. You can go on setting it up at this point.


----
### Available key strokes & patterns 

In the `NORMAL` mode, the following keys are supported:

- `h`  - move one character backward
- `l`  - move one character forward 
- `a`  - pass in Insert mode after next character.
- `A`  - pass in Insert mode at end of line.
- `d`  - pass in Delete mode
- `D`  - delete to end of line (can do the same with `dd`)
- `i`  - pass in Insert mode at cursor position
- `I`  - pass in Insert mode at beginning of line
- `y`  - copy the entire line (needs)
- `Y`  - same as `y` 
- `p`  - paste after cursor
- `P`  - paste before cursor
- `r`  - pass into Replace Once mode (replace a single character)
- `R`  - pass into Replace Many mode (replace all following characters until exiting this mode) 
- `u`  - Undo last action
- `v`  - Open the current line buffer with `$EDITOR`

Other movement key strokes include:
- `[`          - Move cursor to the first previous **brace** encountered
- `]`          - Move cursor to the first next **brace** encountered
- `%`          - Move cursor to the first next **bracket** encountered
- `j`          - Walk the command history (next item)
- `k`          - Walk the command history (previous item)
- `0` to `9`   - Number of Vim iterations to apply on the next keystroke


Keys applying in both the `NORMAL` and `DELETE` modes.

- `w`  - (move/apply action) to the beginning of next word (but up to the first non-letter character encountered)
- `W`  - (move/apply action) to the beginning of next word (but up to the next space character encountered)
- `b`  - (move/apply action) to beginning of current word (but up to the first non-letter character encountered)
- `B`  - (move to beginning) of current word (but up to the next space character encountered)
- `e`  - (move/apply action) to the end of current word (but up to the first non-letter character encountered)
- `E`  - (move/apply action) to the end of current word (but up to the next space character encountered)
- `x`  - Delete the character under cursor, or the number of characters as defined by the current number of Vim iterations.
- `$`  - (apply) to everything until the end of line.


------
## Emacs input mode

The emacs binding are obviously more simple. Setup and usage resume to this:
```go
c.shell.InputMode = readline.Emacs
```
