package donut

import (
	"bufio"
	"bytes"
	"encoding/binary"
)

// Chaskey Implementation ported from donut

const (
	// CipherBlockLen - Chaskey Block Length
	CipherBlockLen = uint32(128 / 8)
	// CipherKeyLen - Chaskey Key Length
	CipherKeyLen = uint32(128 / 8)
)

// ROTR32 - rotates a byte right (same as (32 - n) left)
func ROTR32(v uint32, n uint32) uint32 {
	return (v >> n) | (v << (32 - n))
}

// Chaskey Encryption Function
func Chaskey(masterKey []byte, data []byte) []byte {
	// convert inputs to []uint32
	mk := BytesToUint32s(masterKey)
	p := BytesToUint32s(data)

	// add 128-bit master key
	for i := 0; i < 4; i++ {
		p[i] ^= mk[i]
	}
	// apply 16 rounds of permutation
	for i := 0; i < 16; i++ {
		p[0] += p[1]
		p[1] = ROTR32(p[1], 27) ^ p[0]
		p[2] += p[3]
		p[3] = ROTR32(p[3], 24) ^ p[2]
		p[2] += p[1]
		p[0] = ROTR32(p[0], 16) + p[3]
		p[3] = ROTR32(p[3], 19) ^ p[0]
		p[1] = ROTR32(p[1], 25) ^ p[2]
		p[2] = ROTR32(p[2], 16)
	}
	// add 128-bit master key
	for i := 0; i < 4; i++ {
		p[i] ^= mk[i]
	}
	// convert to []byte for XOR phase
	b := bytes.NewBuffer([]byte{})
	w := bufio.NewWriter(b)
	for _, v := range p {
		binary.Write(w, binary.LittleEndian, v)
	}
	w.Flush()
	return b.Bytes()
}

// BytesToUint32s - converts a Byte array to an array of uint32s
func BytesToUint32s(inbytes []byte) []uint32 {
	mb := bytes.NewBuffer(inbytes)
	r := bufio.NewReader(mb)
	var outints []uint32
	for i := 0; i < len(inbytes); i = i + 4 {
		var tb uint32
		binary.Read(r, binary.LittleEndian, &tb)
		outints = append(outints, tb)
	}
	return outints
}

// Encrypt - encrypt/decrypt data in counter mode
func Encrypt(mk []byte, ctr []byte, data []byte) []byte {
	length := uint32(len(data))
	x := make([]byte, CipherBlockLen)
	p := uint32(0) // data blocks counter
	retval := make([]byte, length)
	for length > 0 {
		// copy counter+nonce to local buffer
		copy(x[:CipherBlockLen], ctr[:CipherBlockLen])

		// donut_encrypt x
		x = Chaskey(mk, x)

		// XOR plaintext with ciphertext
		r := uint32(0)
		if length > CipherBlockLen {
			r = CipherBlockLen
		} else {
			r = length
		}
		for i := uint32(0); i < r; i++ {
			retval[i+p] = data[i+p] ^ x[i]
		}
		// update length + position
		length -= r
		p += r

		// update counter
		for i := CipherBlockLen - 1; i >= 0; i-- {
			ctr[i]++
			if ctr[i] != 0 {
				break
			}
		}
	}
	return retval
}

// Speck 64/128
func Speck(mk []byte, p uint64) uint64 {
	w := make([]uint32, 2)
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, p)
	binary.Read(buf, binary.LittleEndian, &w[0])
	binary.Read(buf, binary.LittleEndian, &w[1])
	k := make([]uint32, 4)

	r := bytes.NewBuffer(mk)
	for c := 0; c < 4; c++ {
		binary.Read(r, binary.LittleEndian, &k[c])
	}

	for i := uint32(0); i < 27; i++ {
		// encrypt 64-bit plaintext
		w[0] = (ROTR32(w[0], 8) + w[1]) ^ k[0]
		w[1] = ROTR32(w[1], 29) ^ w[0]

		// create next 32-bit subkey
		t := k[3]
		k[3] = (ROTR32(k[1], 8) + k[0]) ^ i
		k[0] = ROTR32(k[0], 29) ^ k[3]
		k[1] = k[2]
		k[2] = t
	}

	// return 64-bit ciphertext
	b := bytes.NewBuffer([]byte{})
	binary.Write(b, binary.LittleEndian, w[0])
	binary.Write(b, binary.LittleEndian, w[1])
	num := binary.LittleEndian.Uint64(b.Bytes())
	return num
}

// Maru hash
func Maru(input []byte, iv uint64) uint64 { // todo: iv and return must be 8 bytes

	// set H to initial value
	//h := binary.LittleEndian.Uint64(iv)
	h := iv
	b := make([]byte, MARU_BLK_LEN)

	idx, length, end := 0, 0, 0
	for {
		if end > 0 {
			break
		}
		// end of string or max len?
		if length == len(input) || input[length] == 0 || length == MARU_MAX_STR {
			// zero remainder of M
			for j := idx; j < MARU_BLK_LEN; /*-idx*/ j++ {
				b[j] = 0
			}
			// store the end bit
			b[idx] = 0x80
			// have we space in M for api length?
			if idx >= MARU_BLK_LEN-4 {
				// no, update H with E
				h ^= Speck(b, h)
				// zero M
				b = make([]byte, MARU_BLK_LEN)
			}
			// store total length in bits
			binary.LittleEndian.PutUint32(b[MARU_BLK_LEN-4:], uint32(length)*8)
			idx = MARU_BLK_LEN
			end++
		} else {
			// store character from api string
			b[idx] = input[length]
			idx++
			length++
		}
		if idx == MARU_BLK_LEN {
			// update H with E
			h ^= Speck(b, h)
			// reset idx
			idx = 0
		}
	}
	return h
}
