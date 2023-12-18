The `loot` command is a server-side store of looted files and credentials. Since this is implemented server-side, all files and credentials in the loot store are shared amongst all operators in multiplayer mode. See `loot --help` for a complete list of sub-commands.

### Adding Loot

You can add loot to the loot store several different ways:

- The `loot remote` command can be used to pull a file directly from a remote system to the loot store (requires an active session).
- The `loot local` command can be used to add files from the operator's local machine to the loot store.
- Several commands (e.g. `sideload`, and `execute-assembly`) have the `--loot` flag, which can be used to save the output of the command to the loot store.

### Fetching Loot

To fetch (i.e., look at) a piece of loot use the `loot fetch` command, it will present an interactive menu for you to select from. Textual loot will be displayed directly, file loot can be saved to a local file.

### Remove Loot

The `loot rm` command is used to remove loot from the server.
