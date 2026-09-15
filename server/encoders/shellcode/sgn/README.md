# SGN shellcode encoder adapter

This package adapts [`github.com/moloch--/sgn`](https://github.com/moloch--/sgn)
for Sliver's shellcode-encoder interface. Sliver's public RPC and Go APIs are
unchanged; the adapter translates `SGNConfig` into the upstream encoder and
uses its deterministic seeded encoding entry point with fresh cryptographic
randomness for every production attempt.

## Configuration

- `Architecture` selects 386 or amd64 encoding.
- `Iterations` controls the number of encoding layers.
- `MaxObfuscation` is the decoder obfuscation-byte budget.
- `PlainDecoder` leaves the decoder stub unencoded.
- `Safe` preserves general-purpose registers after the payload falls through
  with its stack pointer balanced. It does not preserve RFLAGS, XMM, or x87
  state; SGN v0.1.2 also preserves the caller's x64 stack alignment.
- `BadChars` and `Asci` enable output constraints.

Unconstrained encoding is attempted exactly once. Constraint searches may try
up to 64 independently seeded outputs, but an upstream encoder error or empty
output fails immediately; retries never mask an encoder defect.

## Tests and fixtures

The tests cover configuration mapping, strict constraint handling, fixed-seed
replay on 386 and amd64, concurrent seeded encoding, and multiple deterministic
variants of three embedded `msfvenom` fixtures. The comprehensive shellcode E2E
workflow adds native execution coverage: every SGN matrix cell must produce and
execute four distinct randomized outputs, and any failed sample fails the cell.

Fixtures under `testdata/` can be regenerated when intentionally updating them:

```bash
go generate ./server/encoders/shellcode/sgn
```

This requires Metasploit's `msfvenom` on `PATH`. Run repository unit tests only
through the project wrapper:

```bash
./go-tests.sh --unit-only
```
