<div align="center">
  <br> <h1> Readline </h1>
</div>

<!-- Badges -->
<p align="center">
  <a href="https://github.com/reeflective/readline/actions/workflows/go.yml">
    <img src="https://github.com/reeflective/readline/actions/workflows/go.yml/badge.svg?branch=master"
      alt="Github Actions (workflows)" />
  </a>

  <a href="https://github.com/reeflective/readline">
    <img src="https://img.shields.io/github/go-mod/go-version/reeflective/readline.svg"
      alt="Go module version" />
  </a>

  <a href="https://pkg.go.dev/github.com/reeflective/readline">
    <img src="https://img.shields.io/badge/godoc-reference-blue.svg"
      alt="GoDoc reference" />
  </a>

  <a href="https://goreportcard.com/report/github.com/reeflective/readline">
    <img src="https://goreportcard.com/badge/github.com/reeflective/readline"
      alt="Go Report Card" />
  </a>

  <a href="https://codecov.io/gh/reeflective/readline">
    <img src="https://codecov.io/gh/reeflective/readline/branch/master/graph/badge.svg"
      alt="codecov" />
  </a>

  <a href="https://opensource.org/licenses/BSD-3-Clause">
    <img src="https://img.shields.io/badge/License-BSD_3--Clause-blue.svg"
      alt="License: BSD-3" />
  </a>
</p>

This library is a modern, pure Go `readline` shell implementation, with full `.inputrc` and legacy
readline command/option support, and extended with various commands, options and tools commonly
found in modern shells. Its architecture and completion system is heavily inspired from Z-Shell.
It is used, between others, to power the [console](https://github.com/reeflective/console) library.

## Features

### Core

- Pure Go, almost-only standard library
- Cross-platform (Linux / MacOS / Windows)
- Full `.inputrc` support (all commands/options)
- Extensive test suite and almost full coverage of core code
- [Extended list](https://github.com/reeflective/readline/wiki/Keymaps-&-Commands) of additional commands/options (edition/completion/history)
- Complete [multiline edition/movement support](https://github.com/reeflective/readline/wiki/Multiline)
- Command-line edition in `$EDITOR`/`$VISUAL` support
- [Programmable API](https://github.com/reeflective/readline/wiki/Programmable-Commands), with failure-safe access to core components
- Support for an [arbitrary number of history sources](https://github.com/reeflective/readline/wiki/History-Sources)

### Emacs / Standard

- Native Emacs commands
- Emacs-style [macro engine](https://github.com/reeflective/readline/wiki/Macros#emacs) (not working across multiple calls)
- Keywords [switching](https://github.com/reeflective/readline/wiki/Keymaps-&-Commands#modifying-text) (operators, booleans, hex/binary/digit) with iterations
- Command/mode cursor status indicator
- Complete undo/redo history
- Command status/arg/iterations hint display

### Vim

- Near-native Vim mode
- Vim [text objects](https://github.com/reeflective/readline/wiki/Keymaps-&-Commands#text-objects) (code blocks, words/blank/shellwords)
- Extended surround select/change/add functionality, with highlighting
- Vim Visual/Operator pending mode & cursor styles indications
- Vim Insert and Replace (once/many)
- All Vim registers, with completion support
- [Vim-style](https://github.com/reeflective/readline/wiki/Macros#vim) macro recording (`q<a>`) and invocation (`@<a>`)

### Interface

- Support for PS1/PS2/RPROMPT/transient/tooltip [prompts](https://github.com/reeflective/readline/wiki/Prompts) (compatible with [oh-my-posh](https://github.com/JanDeDobbeleer/oh-my-posh))
- Extended completion system, [keymap-based and configurable](https://github.com/reeflective/readline/wiki/Keymaps-&-Commands#completion), easy to populate & use
- Multiple completion display styles, with color support.
- Completion & History incremental search system & highlighting (fuzzy-search).
- Automatic & context-aware suffix removal for efficient flags/path/list completion.
- Optional asynchronous autocomplete
- Builtin & programmable [syntax highlighting](https://github.com/reeflective/readline/wiki/Syntax-Highlighting)

## Documentation

Readline is used by the [console library](https://github.com/reeflective/console) and its [example binary](https://github.com/reeflective/console/tree/main/example). To get a grasp of the
functionality provided by readline and its default configuration, install and start the binary.

The documentation is available on the [repository wiki](https://github.com/reeflective/readline/wiki), for both users and developers.

## Showcases

<details>
  <summary>- Emacs edition</summary>
 <dd><em>(This extract is quite a pity, because its author is not using Emacs and does not know many of its shortcuts)</em></dd>
<img src="https://github.com/reeflective/readline/blob/assets/emacs.gif"/>
</details>
<details>
  <summary>- Vim edition</summary>
<img src="https://github.com/reeflective/readline/blob/assets/vim.gif"/>
</details>
<details>
  <summary>- Undo/redo line history </summary>
<img src="https://github.com/reeflective/readline/blob/assets/undo.gif"/>
</details>
<details>
  <summary>- Keyword switching & selection </summary>
 <dd><em>Switching various keywords</em></dd>
<img src="https://github.com/reeflective/readline/blob/assets/switch-keywords.gif"/>
 <dd><em>Using regexp-based selection to grab parts of words (here, URL components)</em></dd>
<img src="https://github.com/reeflective/readline/blob/assets/select-keywords.gif"/>
</details>
<details>
  <summary>- Vim selection & movements (basic) </summary>
<img src="https://github.com/reeflective/readline/blob/assets/vim-selection.gif"/>
</details>
<details>
  <summary>- Vim surround (selection and change) </summary>
 <dd><em>Selecting/adding/changing surround regions</em></dd>
<img src="https://github.com/reeflective/readline/blob/assets/vim-surround.gif"/>
 <dd><em>Surround and change in shellwords, matching brackets, etc.</em></dd>
<img src="https://github.com/reeflective/readline/blob/assets/vim-surround-2.gif"/>
</details>
<details>
  <summary>- Vim registers (with completion) </summary>
<img src="https://github.com/reeflective/readline/blob/assets/registers.gif"/>
</details>
<details>
  <summary>- History movements/completion/use/search </summary>
 <dd><em>History movement, completion and some other other widgets</em></dd>
<img src="https://github.com/reeflective/readline/blob/assets/history.gif"/>
</details>
<details>
  <summary>- Completion </summary>
 <dd><em>Classic mode & incremental search mode</em></dd>
<img src="https://github.com/reeflective/readline/blob/assets/completion.gif"/>
 <dd><em>Suffix-autoremoval </em></dd>
<img src="https://github.com/reeflective/readline/blob/assets/suffix-autoremoval.gif"/>
</details>
<details>
  <summary>- Prompts </summary>
<img src="https://github.com/reeflective/readline/blob/assets/prompts.gif"/>
</details>
<details>
  <summary>- Logging </summary>
<img src="https://github.com/reeflective/readline/blob/assets/logging.gif"/>
</details>
<details>
  <summary>- Inputrc init file reload </summary>
<img src="https://github.com/reeflective/readline/blob/assets/config-reload.gif"/>
</details>
<details>
  <summary>- Multiline edition </summary>
<img src="https://github.com/reeflective/readline/blob/assets/multiline.gif"/>
</details>
<details>
  <summary>- Macros </summary>
 <dd><em>Emacs</em></dd>
<img src="https://github.com/reeflective/readline/blob/assets/emacs-macros.gif"/>
 <dd><em>Vim</em></dd>
<img src="https://github.com/reeflective/readline/blob/assets/vim-macros.gif"/>
</details>

## Status

This library is now in a release status, as it has underwent several major rewrites and is now considered mostly
feature-complete, with a solid testing suite to ensure safe and smooth operation to the best extent possible.
New releases will be regularly pushed when bugs are found and corrected.

Additionally:

- Key dispatch/flushing, meta-key enable, etc might still contain some bugs/wrong behavior:
  30 years of legacy support for 3000 different terminal emulators cannot be done right by me alone.
- Please open a PR or an issue if you face any bug, and it will be promptly resolved.
- Don't hesitate proposing a new feature or a PR if you deem it to be useful to most users.

## Credits

- @kenshaw for his `.inputrc` parsing package, which brings much wider compatibility to this library.
- `chzyer/readline` for the Windows I/O code and everything related.
- Some of the Vim code is inspired or translated from [zsh-vi-mode](https://github.com/jeffreytse/zsh-vi-mode).
- [lmorg/readline](https://github.com/lmorg/readline), for the line tokenizers.
