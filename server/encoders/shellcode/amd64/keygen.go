package amd64

import (
	"crypto/rand"
	"fmt"
)

const xorDynamicKeySize = 8

// XorKeyGen returns a random key suitable for the XOR encoder.
func XorKeyGen() ([]byte, error) {
	return randomBytes(xorKeySize)
}

// XorDynamicKeyGen returns a random key suitable for the XOR dynamic encoder.
func XorDynamicKeyGen() ([]byte, error) {
	return randomBytesFiltered(xorDynamicKeySize, xorDynamicBadchars)
}

// XorDynamicKeyGenForPayload returns a payload-aware key whose encoded bytes
// avoid badChars. Empty badChars use the xor_dynamic defaults.
func XorDynamicKeyGenForPayload(data []byte, badChars []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("xor_dynamic keygen: empty payload")
	}

	badcharSet := xorDynamicBadcharSet(badChars)
	allowed := allowedDynamicCharsFor(badcharSet)
	if len(allowed) < 2 {
		return nil, fmt.Errorf("xor_dynamic keygen: not enough allowed key bytes")
	}
	// Keep one allowed byte out of the key so the decoder can use it as the
	// key terminator even when a large payload requires a large key.
	keyCandidates := allowed[1:]

	badcharList := make([]byte, 0, len(badcharSet))
	for badchar := range badcharSet {
		badcharList = append(badcharList, badchar)
	}

	keyLength := xorDynamicKeySize
	if len(badcharList) > 0 {
		// A residue class containing at most maxGroupSize payload bytes is
		// guaranteed to leave at least one key byte available: every payload
		// byte can rule out at most len(badcharList) candidates.
		maxGroupSize := (len(keyCandidates) - 1) / len(badcharList)
		if maxGroupSize > 0 {
			guaranteedLength := (len(data) + maxGroupSize - 1) / maxGroupSize
			if guaranteedLength > keyLength {
				keyLength = guaranteedLength
			}
		} else if len(data) > keyLength {
			keyLength = len(data)
		}
	}

	randomStarts := make([]byte, keyLength)
	if _, err := rand.Read(randomStarts); err != nil {
		return nil, err
	}

	key := make([]byte, keyLength)
	for keyIndex := range key {
		var forbidden [256]bool
		for payloadIndex := keyIndex; payloadIndex < len(data); payloadIndex += keyLength {
			for _, badchar := range badcharList {
				forbidden[data[payloadIndex]^badchar] = true
			}
		}

		start := int(randomStarts[keyIndex]) % len(keyCandidates)
		found := false
		for offset := range len(keyCandidates) {
			candidate := keyCandidates[(start+offset)%len(keyCandidates)]
			if forbidden[candidate] {
				continue
			}
			key[keyIndex] = candidate
			found = true
			break
		}
		if !found {
			return nil, fmt.Errorf("xor_dynamic keygen: no key byte for position %d", keyIndex)
		}
	}

	return key, nil
}

func randomBytes(length int) ([]byte, error) {
	if length <= 0 {
		return nil, fmt.Errorf("invalid random length %d", length)
	}
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func randomBytesFiltered(length int, badchars map[byte]bool) ([]byte, error) {
	if length <= 0 {
		return nil, fmt.Errorf("invalid random length %d", length)
	}
	buf := make([]byte, length)
	for i := range buf {
		for {
			if _, err := rand.Read(buf[i : i+1]); err != nil {
				return nil, err
			}
			if !badchars[buf[i]] {
				break
			}
		}
	}
	return buf, nil
}
