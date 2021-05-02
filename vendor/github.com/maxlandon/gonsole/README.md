
Gonsole - Integrated Console Application library
=========

This package rests on a [readline](https://github.com/maxlandon/readline) console library, (giving advanced completion, hint, input and history system), 
and the [go-flags](https://github.com/jessievdk/go-flags) commands library. Also added, a bit of optional boilerplate for better user experience.

![readme-main-gif](https://github.com/maxlandon/gonsole/blob/assets/readme-main.gif)

The purpose of this library is to offer a complete off-the-shelf console application, with some key aspects: 
- Better overall features than what is seen in most projects, including those not written in Go.
- A simple but powerful way of transforming code (structs and anything they might embed), into commands.
- Easily provide completions for any command/subcommand, any arguments, or any option arguments.
- If you get how to declare a go-flags compatible command, you can declare commands for this console.


## Simple Usage

The library is made to work with sane but powerful defaults. Paste the following, run it,
and take a look around to get a feel, without your commands. Default editing mode is Vim.
THis example doesn't have an exit command: you'll need to close your terminal.

```go

func main() {

	// Instantiate a new console, with a single, default menu.
	// All defaults are set, and nothing is needed to make it work
	console := gonsole.NewConsole()

	// By default the shell as created a single menu and
	// made it current, so you can access it and set it up.
	menu := console.CurrentMenu()

	// Set the prompt (config, for usability purposes). Each menu has its own.
	// See the documentation for more prompt setup possibilities.
	prompt := menu.PromptConfig()
	prompt.Left = "application-name"
	prompt.Multiline = false

	// Add a default help command, that can be used with any command, however nested:
	// 'help <command> <subcommand> <subcommand'
	// The console creates it and attaches it to all existing contexts.
	// "core" is the name of the group in which we will put this command.
	console.AddHelpCommand("core")

	// Add a configuration command if you want your users to be able
	// to modify it on the fly, export it as files or as JSON.
	// Please see the documentation and/or use this example to
	// see what can be done with this.
	console.AddConfigCommand("config", "core")

	// Everything is ready for a tour.
	// Run the console and take a look around.
	console.Run()
}
```

If you're still here, at least you want to declare and bind commands. Just as everything else possible with
this library, it is explained in the [Wiki](https://github.com/maxlandon/gonsole/wiki), although with more 
pictures than text (I like pictures), because the code is heavily documented (I don't like to repeat myself).
Using the library, as usual:
```
go get -u github.com/maxlandon/gonsole
```

---- 
## Documentation Contents

### Developers
* [Menus](https://github.com/maxlandon/gonsole/wiki/Menus)
* [Configurations Overview](https://github.com/maxlandon/gonsole/wiki/Configurations-Overview)
* [Setting Prompts & Input Modes](https://github.com/maxlandon/gonsole/wiki/Prompts-&-Input-Modes)
* [Default commands](https://github.com/maxlandon/gonsole/wiki/Default-Commands)
* [Declaring commands](https://github.com/maxlandon/gonsole/wiki/Declaring-Commands)
* [Querying state from commands](https://github.com/maxlandon/gonsole/wiki/Querying-State-From-Commands)
* [Completions (writing and binding)](https://github.com/maxlandon/gonsole/wiki/Completions)
* [Additional Expansion completions](https://github.com/maxlandon/gonsole/wiki/Expansion-Completers)
* [History Sources Declaration](https://github.com/maxlandon/gonsole/wiki/History-Sources-Declaration)
* [Asynchronous Logs & Prompt Refresh](https://github.com/maxlandon/gonsole/wiki/Prompt-Refresh)

### Users
- [Vim Keys & Shortcuts](https://github.com/maxlandon/gonsole/wiki/Vim-Keys-&-Shortcuts)
- [History sources](https://github.com/maxlandon/gonsole/wiki/History-Sources)
- [Completions & Tab Search](https://github.com/maxlandon/gonsole/wiki/Completions-&-Tab-Search)
- [Help and config commands](https://github.com/maxlandon/gonsole/wiki/Help-&-Config-Commands)


----
## Features 

The list of features supported or provided by this library can fall into 2 different categories:
the shell/console interface part, and the commands/parsing logic part.  Some of the features below
are simply extrated from my [readline](https://github.com/maxlandon/readline) library (everything below **Shell Details**).

#### Menus & Commands
- Declare different "menus" to which you can bind commands, prompt and shell settings.
- The library is fundamentally a wrapper around the [go-flags](https://github.com/jessievdk/go-flags) commands/options, etc.
- This go-flag library allows you to create commands out of structs (however populated), and gonsole asks you to pass these structs to it.
- This works for any level of command nesting. 
- Also allows you to declare as many option groups you want, all of that will work.
- All commands have methods for adding completions either to their arguments, or to their options.

#### Shell details
- Vim / Emacs input and editing modes.
- Vim modes (Insert, Normal, Replace, Delete) with visual prompt Vim status indicator
- Line editing using `$EDITOR` (`vi` in the example - enabled by pressing `[ESC]` followed by `[v]`)
- Vim registers (one default, 10 numbered, and 26 lettered) and Vim iterations

#### Completion engine
- Rather easy declaration of completion generators, which some level of customization.
- 3 types of completion categories (`Grid`, `List` and `Map`)
- In `List` completion groups, ability to have alternative candidates (used for displaying `--long` and `-s` (short) options, with descriptions)
- Completions working anywhere in the input line (your cursor can be anywhere)
- Completions are searchable with *Ctrl-F*, 

#### Others
- You can pass special completers that will be triggered if the rune (like `$` or `@`) is detected, anywhere in the line. These variables are expanded at command execution time, and work in completions as well.
- You can export the configuration for your application, its menus, and add some custom subcommands to the root one, for specialized actions over it.
- Also, an optional `help` command can be bound to the console, in additional to default `-h`/`--help` flags for every command.
- History sources can also be bound per menu/menu.

#### Prompt system & Colors
- 1-line and 2-line prompts, both being customizable.
- Function for refreshing the prompt, with optional behavior settings.

#### Hints & Syntax highlighting
- A hint line can be printed below the input line, with any type of information. See utilities for a default one.
- The Hint system is now refreshed depending on the cursor position as well, like completions.
- A syntax highlighting system. 

#### Command history 
- Ability to have 2 different history sources (I used this for clients connected to a server, used by a single user).
- History is searchable like completions.
- Default history is an in-memory list.
- Quick history navigation with *Up*/*Down* arrow keys in Emacs mode, and *j*/*k* keys in Vim mode.


## Status & Support 

#### Support:
- Support for any issue opened.
- Answering any questions related.
- Taking blames if things are done wrong.

#### TO DO:
- [ ] Recursive option completion (`-sP`, `-oL`, etc)
- [ ] `config load` command
- [ ] Analyse args and pull out from comps if in line
- [ ] Add color for strings in input line (this will need a good part of murex parser code) 
- [ ] Add token parsing code from murex (this must be well thought out, like the quotes stuff, because it must also not interfere with other commands in special menus, the command parsing code, etc...)
- [ ] Let the user pass keypresses and their associated behavior, like we could do in readline.


## Version Information

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


## License

The `gonsole` library is distributed under the Apache License (Version 2.0, January 2004) (http://www.apache.org/licenses/). 
All the example code and documentation in `/examples`, `/completers` is public domain.

