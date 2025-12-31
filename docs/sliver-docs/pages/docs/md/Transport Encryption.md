**⚠️ NOTE:** This document does not apply when mTLS or WireGuard are used.

# Sliver v1.6.x

The following keys are embedded in each implant at compile time; the server also stores these values in its database, along with the SHA2-256 hash of the implant's peer public key:

1. Age public key of server
2. Age implant "peer" public key
3. Age implant "peer" private key
4. A minisign signature of the implant's age peer public key (signed by server's private key)
5. The server's minisign public key

### Server to Implant Key Exchange

1. Implant generates random 256-bit symmetric "session key"
2. Implant generates:

- Calculate the HMAC-SHA2-256 of the session key, where the HMAC key is the SHA2-256 hash of the peer private key
- Uses Age to encrypt the session key and HMAC-SHA256 with the server's public key

3. Implant sends `[SHA2-256 Hash of Public Key | Age Ciphertext ]` to server.
4. Server decrypts message with its private key
5. Server generates a session ID, encrypts it with the session key using ChaCha20Poly1305, and sends it back
6. All messages are encrypted with the session key using ChaCha20Poly1305 and associated with the session ID
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
11. The initiator then performs a key exchange with the server using the server's embedded Age public key and the same pattern described above. This prevents upstream implants (i.e. the listener) from being able to decrypt any traffic sent between the initiator and the server.

### Known Limitations

There are some known limitations, if you spot any or have ideas on improvements please file a ticket or contact us.

1. Perfect Forward Secrecy: We do get some forward secrecy, since only the public-key-encrypted version of the session key is sent over the wire; therefore, recovery of the hard-coded implant keys from the binary should not result in recovery of the session key. Only when the server's private key is compromised can the session key be recovered, which we currently don't consider to be a problem.
2. Implants can potentially be tracked via the hash of their public key. However, this value is implant-specific, so in order to track the implant this way you'd have to already have a copy of the specific implant you want to track. At which point more effective tracking mechanisms like YARA rules could be employed.
3. While messages cannot be replayed, valid messages can potentially be re-ordered.
