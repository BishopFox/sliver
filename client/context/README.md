Context
======

The `context` package fills several roles:
 * `console.go`     - It holds some console state that many packages must have access to.
 * `commands.go`    - It adds a bit of command group structuring (for printing help, or structuring completion groups)
 * `session.go`     - It wraps Sliver session objects for more easily populating their requests.
