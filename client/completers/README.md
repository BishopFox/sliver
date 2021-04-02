Completers
======

This package contains the full completion/hint/syntax processing engine of Sliver, wired to its readline library dependency 
Currently this package is not fully on the shoulders of readline, but might be in the future. Many completions are offered, in all contexts.
Not all files are explained below, only the most importants.

**Engines & Core**

 * `tab-completer.go`       - The tab completion entrypoint, dispatching the input line toward its intended completers.
 * `command-arguments.go`   - Most of commands with arguments have completers in there.
 * `option-arguments.go`    - Same as command arguments, but for their --options.
 * `hint-completer.go`      - The hint system engine, similar in working to the tab completer.
 * `load-completions.go`    - Some commands binds additional lists of completions from other packages. They call here.
 * `cache.go`               - In order to limit the completion-incurred traffic with implants, we cache when possible.

 **Completions**

 * `argument-lists.go`      - Some fixed list of arguments (archs, protocols, etc)
 * `local-filesystem.go`    - The local (client) filesystem completer.
 * `local-net.go`           - Completions for the client network interfaces (address and/or CIDR, IPv4 & IPv6)
 * `msf.go`                 - Full list of MSFVenom payloads, and formats.
 * `prompt.go`              - Completions for the prompt system, which can be set from the console (colors & items).
 * `sliver-filesystem.go`   - The remote (Sliver session) filesystem completer. Is used by the cache.
 * `sliver-net.go`          - Completions for the session's network interfaces (address and/or CIDR, IPv4 & IPv6)
 * `url.go`                 - A URL completer (scheme://address), for some commands that require full URL values.
