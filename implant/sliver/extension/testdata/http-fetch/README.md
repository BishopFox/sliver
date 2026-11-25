# Sliver WASI HTTP fetch example

This Go `wasip1` module fetches `https://example.com/` by default and accepts a
different URL as its first argument. Go's `wasip1` port has no operating-system
certificate store, so the module embeds curl's complete Mozilla public CA
extract using `//go:embed`.

`public-roots.pem` is the 119-certificate Mozilla snapshot dated 2026-07-16
from <https://curl.se/docs/caextract.html>. Its SHA-256 digest is
`3ff344e30b9b1ed2971044eabb438a08f2e2245ddb5f8ab1a3ad8b63ab4eaf91`.

Build it with the Sliver-specific compiler wrapper:

```console
$SLIVER_ROOT_DIR/go/bin/sliver-wasm-go build -o http-fetch.wasm .
```

The resulting module imports `sliver_wasi_net_v1`, so it must run in a Sliver
Wasm extension runtime (or another runtime implementing the same versioned
host ABI).
