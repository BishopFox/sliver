package core

import (
	"crypto/rand"
	"encoding/binary"
)

// EnvelopeID - Generate random ID of randomIDSize bytes
func EnvelopeID() uint64 {
	randBuf := make([]byte, 8) // 64 bytes of randomness
	rand.Read(randBuf)
	return binary.LittleEndian.Uint64(randBuf)
}
