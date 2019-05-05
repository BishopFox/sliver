package sha1

import "encoding/binary"

// Size of a SHA-1 checksum in bytes.
const Size = 20

// BlockSize is the block size of SHA-1 in bytes.
const BlockSize = 64

// Sum returns the SHA-1 checksum of data.
func Sum(data []byte) [Size]byte {
	n := len(data)
	h := [5]uint32{0x67452301, 0xefcdab89, 0x98badcfe, 0x10325476, 0xc3d2e1f0}

	// Consume full blocks.
	for len(data) >= BlockSize {
		block(&h, data)
		data = data[BlockSize:]
	}

	// Final block.
	tmp := make([]byte, BlockSize)
	copy(tmp, data)
	tmp[len(data)] = 0x80

	if len(data) >= 56 {
		block(&h, tmp)
		for i := 0; i < BlockSize; i++ {
			tmp[i] = 0
		}
	}

	binary.BigEndian.PutUint64(tmp[56:], uint64(8*n))
	block(&h, tmp)

	// Write into byte array.
	var digest [Size]byte
	for i := 0; i < 5; i++ {
		binary.BigEndian.PutUint32(digest[4*i:], h[i])
	}

	return digest
}
