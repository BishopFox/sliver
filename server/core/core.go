package core

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

const (
	// randomIDSize - Size of the TunnelID in bytes
	randomIDSize = 8
)

// RandomID - Generate random ID of randomIDSize bytes
func RandomID() string {
	randBuf := make([]byte, 64) // 64 bytes of randomness
	rand.Read(randBuf)
	digest := sha256.Sum256(randBuf)
	return fmt.Sprintf("%x", digest[:randomIDSize])
}

// EnvelopeID - Generate random ID of randomIDSize bytes
func EnvelopeID() uint64 {
	randBuf := make([]byte, 8) // 64 bytes of randomness
	rand.Read(randBuf)
	return binary.LittleEndian.Uint64(randBuf)
}
