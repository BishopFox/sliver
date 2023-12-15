**⚠️ NOTE:** This document does not apply when mTLS or WireGuard are used.

# Versions 1.5.40+

The following keys are embedded in each implant at compile time, the server also stores these values in its database in addition to the SHA2-256 hash of the implant's peer public key:

1. Age public key of server
2. Age implant "peer" public key
3. Age implant "peer" private key
4. A minisign signature of the implant's age peer public key (signed by server's private key)
5. The server's minisign public key

### Server to Implant Key Exchange

1. Implant generates random 256-bit symmetric "session key"
2. Implant generates:

- Calculate the HMAC-SHA2-256 of the session key, where the HMAC key is the SHA2-256 hash of the peer private key
- Uses Age to encrypt session key and HMAC-SHA256 with server's public key

3. Implant sends `[SHA2-256 Hash of Public Key | Age Ciphertext ]` to server.
4. Server decrypts message with its private key
5. Server generates a session ID, encrypts it with the session key using ChaCha20Poly1305, and sends it back
6. All messages are encrypted with the session key using ChaCha20Poly1305 and associated with via the session ID
7. Each side stores a SHA2-256 hash of each message's ciphertext to detect replayed messages

### Implant to Implant Key Exchange (Pivots)

1. An implant starts a pivot listener (the listener)
2. Another implant connects to the listener (the initiator)
3. The initiator sends its Age public key and the minisign signature of its public key
4. The listener verifies the initiator's public key is signed by the listener's server's minisign public key
5. The listener generates a random session key and encrypts it with the initiator's verified public key
6. The listener sends its Age public key, the minisign signature of its public key, and the encrypted session key back to the initiator
7. The initiator verifies the listener's public key is signed by the initiator's server's minisign public key
8. The initiator decrypts the session key using Age
9. All messages are encrypted with the session key using ChaCha20Poly1305
10. Each side stores a SHA2-256 hash of each message's ciphertext to detect replayed messages
11. The initiator then performs a key exchange with the server, using the server's embedded ECC public key, using the same pattern as described above. This prevents upstream implants (i.e. the listener) from being able to decrypt any traffic sent between the initiator and the server.

### Known Limitations

There are some known limitations, if you spot any or have ideas on improvements please file a ticket or contact us.

1. Perfect Forward Secrecy: We do get some forward secrecy, since only the public key encrypted version of the session key is sent over the wire; therefore recovery of the hard coded implant keys from the binary should not result in recovery of the session key. Only when the server's private key is compromised can the session key be recovered, which we currently don't consider to be a problem.
2. Implants can potentially be tracked via the hash of their public key. However, this value is implant specific, so in order to track the implant this way you'd have to already have a copy of the specific implant you want to track. At which point more effective tracking mechanisms like YARA rules could be employed.
3. While messages cannot be replayed, valid messages can potentially be re-ordered.

# Versions 1.5.0+

⚠️ This version of the key exchange is [vulnerable to MitM attack](https://github.com/BishopFox/sliver/security/advisories/GHSA-8jxm-xp43-qh3q)

The following keys are embedded in each implant at compile time, the server also stores these values in its database in addition to the SHA2-256 hash of the implant's public key:

1. ECC public key of server
2. ECC implant public key
3. ECC implant private key
4. A minisign signature of the implant ECC public key (signed by server's private key)
5. The server's minisign public key
6. TOTP shared secret (server-wide shared secret)

### Server to Implant Key Exchange

1. Implant generates random 256-bit symmetric "session key"
2. Implant generates:

- Current TOTP code using SHA2-256, Unix UTC, 8-digit numeric code
- SHA2-256 hash of its own ECC public key
- Uses Nacl Box (Curve25519, XSalsa20, and Poly1305) to encrypt session key with server's public ECC key

3. Implant sends `[ TOTP Code | SHA2-256 Hash of Public Key | Nacl Box Ciphertext ]` to server, note: in this scheme no ECC keys (even public keys) are ever sent over the wire, instead we only send the hash of the public key.
4. Server verifies TOTP Code (can optionally be disabled per listener server-side).
5. Server uses the SHA2-256 hash of public key to look up the implant's full ECC public key in its database
6. Decrypts Nacl with sender public key + server private key
7. Server generates a session ID, encrypts it with the session key using ChaCha20Poly1305, and sends it back
8. All messages are encrypted with the session key using ChaCha20Poly1305 and associated with via the session ID
9. Each side stores a SHA2-256 hash of each message's ciphertext to detect replayed messages

### Implant to Implant Key Exchange (Pivots)

1. An implant starts a pivot listener (the listener)
2. Another implant connects to the listener (the initiator)
3. The initiator sends its ECC public key and the minisign signature of its public key
4. The listener verifies the initiator's public key is signed by the listener's server's minisign public key
5. The listener generates a random session key and encrypts it with the initiator's verified public key
6. The listener sends its ECC public key, the minisign signature of its public key, and the encrypted session key back to the initiator
7. The initiator verifies the listener's public key is signed by the initiator's server's minisign public key
8. The initiator decrypts the session key using Nacl Box (Curve25519, XSalsa20, and Poly1305)
9. All messages are encrypted with the session key using ChaCha20Poly1305
10. Each side stores a SHA2-256 hash of each message's ciphertext to detect replayed messages
11. The initiator then performs a key exchange with the server, using the server's embedded ECC public key, using the same pattern as described above but without the TOTP code. This prevents upstream implants (i.e. the listener) from being able to decrypt any traffic sent between the initiator and the server.

### Known Limitations

There are some known limitations, if you spot any or have ideas on improvements please file a ticket or contact us.

1. Perfect Forward Secrecy: We do get some forward secrecy, since only the public key encrypted version of the session key is sent over the wire; therefore recovery of the hard coded implant keys from the binary should not result in recovery of the session key. Only when the server's private key is compromised can the session key be recovered, which we currently don't consider to be a problem.
2. Implants can potentially be tracked via the hash of their public key. However, this value is implant specific, so in order to track the implant this way you'd have to already have a copy of the specific implant you want to track. At which point more effective tracking mechanisms like YARA rules could be employed.
3. Session initialization messages can be replayed within the validity period of the TOTP value. TOTP values are valid for 30 seconds + 30 second margin of error, so the session initialization message can be replayed within about ~60 second period without obtaining a new TOTP code. However, the implant must use the session key to register itself post-key exchange so replayed session initialization does not appear to be a security risk even outside of the restrictive window.
4. While messages cannot be replayed, valid messages can potentially be re-ordered.
