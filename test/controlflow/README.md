# Control-flow obfuscation integration harness

This opt-in harness builds and runs native Darwin/arm64 session and beacon
executables through the public multiplayer RPC API. It verifies:

- the server advertises `garble.control-flow/balanced-v1`;
- both builds use symbol obfuscation and `CONTROL_FLOW_BALANCED_V1`;
- the three balanced-v1 directives are attached to the expected rendered
  functions;
- each build is captured with Garble's debug output and produces one parsed,
  flattened `GARBLE_controlflow.go` file for each of the three selected
  packages;
- the session produces a matching `SessionOpened` event and answers a
  synchronous Ping with the requested nonce;
- both implants use randomized C2 selection, exercising the transformed
  `randomCCDomain` before their single mTLS endpoint is selected;
- the session successfully lists a harness-owned temporary proof file through
  a wildcard path, exercising the transformed `determineDirPathFilter`; and
- the beacon produces a matching `BeaconRegistered` event and completes a
  nonempty asynchronous Ping task with the requested nonce, exercising the
  transformed `registerSliver` during registration.

The server, operator files, generated source, compiler cache, logs, and
artifacts live under a unique temporary directory. Multiplayer and implant
mTLS listeners bind dynamic loopback-only ports. Every long-lived child is put
in its own process group, and cleanup signals only those owned groups.

## Run with a prebuilt server

Build the current branch using the repository's normal command, then pass the
result to the harness:

```sh
make
go run -tags=sliver_controlflow_e2e ./test/controlflow \
  --server /absolute/path/to/sliver-server
```

## Let the harness build the server

If the embedded assets are already populated (run `make` once if needed), omit
`--server`. The harness performs an isolated `go build` of the current checkout
and does not write its server binary or Go build cache into the repository:

```sh
go run -tags=sliver_controlflow_e2e ./test/controlflow
```

Use `--keep-work-dir` to retain successful artifacts and logs. Failed runs are
always preserved and print their work-directory path. The harness refuses to
run on a host other than native Darwin/arm64 because callback verification
executes the generated binaries directly.
