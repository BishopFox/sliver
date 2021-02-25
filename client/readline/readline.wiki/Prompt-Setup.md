
## Single-line prompts

The single-line prompt is the default one. The following combination, with Vim mode and status enabled,
```go
c.shell.Multiline = false
c.shell.SetPrompt("readline")
```
will give you the following prompt (note that the Vim status indicator is always appended at the beginning)
```
[I] readline > 
```
Even in single-mode, you specify the prompt sign (the arrow above is a default):
```go
c.shell.MultilinePrompt = " $"
```
```
[I] readline $
```

Emacs will produce the same result, without the Vim status indicator.


## 2-line prompts

Setting up a 2-line prompt has only one difference:
```go
c.shell.Multiline = true
```

Combined with the settings above, will give this:
```
readline
[I] $
```

Therefore `SetPrompt()` will only control either the first or second line, and `MultilinePrompt` and `ShowVimMode` will only control
behavior for the second line. You can obviously set them at any moment: the next readline execution loop will recompute.

Once done, you can go on setting up the Completion system. Two paths can be taken, depending on your use case:
- *"I don't want to write a completer function, and I intend to use this library with one or more go-flags commands"*: you can directly check [how to interface the shell with go-flags and its default completers](https://github.com/bishopfox/sliver/client/readline/wiki/Interfacing-With-Go-Flags)
- *"I either want to write a completer or learn about it"*: you can go on [to the next section](https://github.com/bishopfox/sliver/client/readline/wiki/Completion-Groups)
