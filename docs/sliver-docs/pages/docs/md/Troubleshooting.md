### Server logs

Server related logs are saved to: `~/.sliver/logs/`

- `sliver.log` is the main log file for server related events
- `console.log` is the log of the server's built-in "client" console
- `sliver.json` JSON formatted log (includes timestamps)
- `audit.json` a JSON formatted history of commands/activity

The default log level for the server is `INFO` when troubleshooting it may be helpful to increase this to `DEBUG` (5), which can be done by editing the [server configuration file](/docs?name=Configuration+Files)

### Client logs

`~/.sliver-client/sliver-client.log`

## Implant Troubleshooting

If you need to trouble shoot an issue (connectivity, etc) with an implant try generating it with the `--debug` flag.
