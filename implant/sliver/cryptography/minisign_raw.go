package cryptography

import (
	"encoding/binary"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/ed25519"
)

// MinisignVerifyRaw - Verify a fixed-length raw minisign signature (binary) using
// the server's bundled minisign public key.
func MinisignVerifyRaw(message []byte, rawSig []byte) bool {
	if len(rawSig) != RawSigSize {
		return false
	}

	serverPublicKey, err := DecodeMinisignPublicKey(serverMinisignPublicKey)
	if err != nil {
		return false
	}

	algorithm := binary.LittleEndian.Uint16(rawSig[:2])
	keyID := binary.LittleEndian.Uint64(rawSig[2:10])
	if keyID != serverPublicKey.ID() {
		return false
	}

	switch algorithm {
	case EdDSA:
		// Verify as-is.
	case HashEdDSA:
		digest := blake2b.Sum512(message)
		message = digest[:]
	default:
		return false
	}

	signature := rawSig[10:]
	if len(signature) != ed25519.SignatureSize {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(serverPublicKey.PublicKey[:]), message, signature)
}
