Contributing to Sliver
======================

## General

* Contributions to core code must be GPLv3 (but not libraries)
* If you'd like to work on a feature, please open a ticket and assign it to yourself
* Changes should be made in a new branch
* Commits [must be signed](https://docs.github.com/en/github/authenticating-to-github/signing-commits) for any PR to master
* Please provide meaningful commit messages
* Ensure code passes existing unit tests, or provide updated test(s)
* `gofmt` your code
* Any changes to `vendor/` should be in a distinct commit
* Avoid use of `CGO` (limits cross-platform support)
* Avoid use of empty interfaces
* Never import anything from the `server` package in the `client` package.

## Security

* _Never_ trust the user, applied in a common-sense way.
* __Secure by default__, please ensure any contributed code follows this methodology to the best of your ability. It should be difficult to insecurely configure features/servers.
    - It is better to fail securely than operate in an insecure manner.
* _Avoid_ incorporating user controlled values when constructing file/directory paths. Ensure any values that must be incorporated into paths are properly canonicalized.
* _Never_ use homegrown or non-peer reviewed encryption or random number generation algorithms.
* Whenever possible, use the following algorithms/encryption modes:
    - AES-GCM-256
    - SHA2-256 / HMAC-SHA2-256 or higher (e.g. SHA2-384)
    - Curves P521, P384, P256
    - Curve25519, XSalsa20, and Poly1305 (Nacl)
    - ChaCha20Poly1305
* _Never_ use the following in a security context, and _avoid_ use even in a non-security context:
    - MD5
    - SHA1
    - AES-ECB
    - AES-CBC, AES-CTR, etc. -without use case justification
* `math/random` should _always_ be imported as `insecureRand` and _never_ used to generate values related to a security context.
* Always apply the most restrictive file permissions possible.
* Apply obfuscation techniques when possible, but _do not rely upon_ obfuscation for security.
