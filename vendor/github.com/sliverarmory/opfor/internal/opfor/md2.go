package opfor

import "hash"

// MD2 is still part of the Java MessageDigest surface used by Sleep 2.1. It
// is not provided by Go's standard crypto packages, so OPFOR carries this
// small RFC 1319 implementation rather than silently substituting a different
// digest. MD2 is retained solely for language compatibility; it is not a
// secure digest for new protocols.
type sleepMD2Digest struct {
	state    [48]byte
	checksum [16]byte
	buffer   [16]byte
	buffered int
}

var _ hash.Hash = (*sleepMD2Digest)(nil)

func newSleepMD2() hash.Hash { return &sleepMD2Digest{} }

func (*sleepMD2Digest) Size() int      { return 16 }
func (*sleepMD2Digest) BlockSize() int { return 16 }

func (digest *sleepMD2Digest) Reset() {
	*digest = sleepMD2Digest{}
}

func (digest *sleepMD2Digest) Write(data []byte) (int, error) {
	written := len(data)
	for len(data) != 0 {
		copied := copy(digest.buffer[digest.buffered:], data)
		digest.buffered += copied
		data = data[copied:]
		if digest.buffered == len(digest.buffer) {
			digest.process(digest.buffer[:], true)
			digest.buffered = 0
		}
	}
	return written, nil
}

func (digest *sleepMD2Digest) Sum(prefix []byte) []byte {
	clone := *digest
	paddingLength := len(clone.buffer) - clone.buffered
	var padding [16]byte
	for index := 0; index < paddingLength; index++ {
		padding[index] = byte(paddingLength)
	}
	_, _ = clone.Write(padding[:paddingLength])

	// The checksum appended to the padded message is the value produced after
	// processing the padding block. Processing that final checksum block must
	// not feed it back into the checksum calculation.
	checksum := clone.checksum
	clone.process(checksum[:], false)
	return append(prefix, clone.state[:16]...)
}

func (digest *sleepMD2Digest) process(block []byte, updateChecksum bool) {
	if updateChecksum {
		last := digest.checksum[15]
		for index, value := range block[:16] {
			digest.checksum[index] ^= sleepMD2Permutation[value^last]
			last = digest.checksum[index]
		}
	}

	for index, value := range block[:16] {
		digest.state[16+index] = value
		digest.state[32+index] = value ^ digest.state[index]
	}
	t := byte(0)
	for round := 0; round < 18; round++ {
		for index := range digest.state {
			digest.state[index] ^= sleepMD2Permutation[t]
			t = digest.state[index]
		}
		t += byte(round)
	}
}

var sleepMD2Permutation = [256]byte{
	41, 46, 67, 201, 162, 216, 124, 1, 61, 54, 84, 161,
	236, 240, 6, 19, 98, 167, 5, 243, 192, 199, 115, 140,
	152, 147, 43, 217, 188, 76, 130, 202, 30, 155, 87, 60,
	253, 212, 224, 22, 103, 66, 111, 24, 138, 23, 229, 18,
	190, 78, 196, 214, 218, 158, 222, 73, 160, 251, 245, 142,
	187, 47, 238, 122, 169, 104, 121, 145, 21, 178, 7, 63,
	148, 194, 16, 137, 11, 34, 95, 33, 128, 127, 93, 154,
	90, 144, 50, 39, 53, 62, 204, 231, 191, 247, 151, 3,
	255, 25, 48, 179, 72, 165, 181, 209, 215, 94, 146, 42,
	172, 86, 170, 198, 79, 184, 56, 210, 150, 164, 125, 182,
	118, 252, 107, 226, 156, 116, 4, 241, 69, 157, 112, 89,
	100, 113, 135, 32, 134, 91, 207, 101, 230, 45, 168, 2,
	27, 96, 37, 173, 174, 176, 185, 246, 28, 70, 97, 105,
	52, 64, 126, 15, 85, 71, 163, 35, 221, 81, 175, 58,
	195, 92, 249, 206, 186, 197, 234, 38, 44, 83, 13, 110,
	133, 40, 132, 9, 211, 223, 205, 244, 65, 129, 77, 82,
	106, 220, 55, 200, 108, 193, 171, 250, 36, 225, 123, 8,
	12, 189, 177, 74, 120, 136, 149, 139, 227, 99, 232, 109,
	233, 203, 213, 254, 59, 0, 29, 57, 242, 239, 183, 14,
	102, 88, 208, 228, 166, 119, 114, 248, 235, 117, 75, 10,
	49, 68, 80, 180, 143, 237, 31, 26, 219, 153, 141, 51,
	159, 17, 131, 20,
}
