
## Hint system

The library shell includes an optional hint sentence placed just behind the input line. This hint can span on multiple lines.
It can be deactivated (if set to nil) or customized, by registering a hint function `HintText()`.

Also, the hint refreshes itself depending on the position of your cursor, so that you can have a live description of many different things:
- The current command
- The current argument required
- The current option description
- The current argument to the option, if any

*Note*: that all of this is included in the [default tab/hint completer], because it interfaces with the go-falgs command library. 
If you plan to write your own completer from scratch (which means not even using go-flags or the default completer), this will not work.

![hint](https://github.com/bishopfox/sliver/client/readline/blob/assets/hint.gif)

## Syntax highlighter

The syntax highlighter works with the same kind of function, and the default syntax 
highlighter provided in the library has the same kind of analysis based on go-flags. 

*This example will only highlight (bold) commands and subcommands, but we could do about anything with good line parsing.*
![syntax](https://github.com/bishopfox/sliver/client/readline/blob/assets/syntax.gif)
