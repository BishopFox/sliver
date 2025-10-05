// Copyright (c) 2024 Karl Gaissmaier
// SPDX-License-Identifier: MIT

// Package art summarizes the functions and inverse functions
// for mapping between a prefix and a baseIndex.
//
//	can inline OctetToIdx with cost 5
//	can inline PfxToIdx with cost 10
//	can inline IdxToPfx with cost 30
//	can inline IdxToRange with cost 54
//	can inline PfxBits with cost 21
//	can inline NetMask with cost 7
//
// Please read the ART paper ./doc/artlookup.pdf
// to understand the baseIndex algorithm.
package art

import "math/bits"

// PfxToIdx maps 8bit prefixes to numbers. The prefixes range from 0/0 to 255/7
// The return values range from 1 to 255.
//
//	  [0x0000_00001 .. 0x1111_1111] = [1 .. 255]
//
//		example: octet/pfxLen: 160/3 = 0b1010_0000/3 => IdxToPfx(160/3) => 13
//
//		                0b1010_0000 => 0b0000_0101
//		                  ^^^ >> (8-3)         ^^^
//
//		                0b0000_0001 => 0b0000_1000
//		                          ^ << 3      ^
//		                 + -----------------------
//		                               0b0000_1101 = 13
func PfxToIdx(octet, pfxLen uint8) uint8 {
	return octet>>(8-pfxLen) + 1<<pfxLen
}

// OctetToIdx maps octet/8 prefixes to numbers. The return values range from 256 to 511.
// OctetToIdx is a special case of PfxToIdx.
func OctetToIdx(octet uint8) uint {
	return uint(octet) + 256
}

// IdxToPfx returns the octet and prefix len of baseIdx.
// It's the inverse to pfxToIdx256.
func IdxToPfx(idx uint8) (octet, pfxLen uint8) {
	// The prefix length corresponds to the number of leading bits in idx.
	// bits.Len8 returns the number of bits needed to represent idx as binary,
	// so we subtract 1 to recover the prefix length (which is always >= 0).
	pfxLen = uint8(bits.Len8(idx)) - 1

	// Compute the number of bits to shift back to obtain the original octet.
	shiftBits := 8 - pfxLen

	// Extract the original prefix bits from idx by masking out the low bits.
	mask := uint8(0xff) >> shiftBits

	// Shift the masked prefix bits back to their original position.
	octet = (idx & mask) << shiftBits

	return
}

// PfxBits returns the bit position of a prefix represented by a base index at a given trie depth.
//
// Each trie level represents an 8-bit stride (one octet).
// The base index contains enough information to recover the prefix length.
// This function returns the full bit offset of the prefix in the address space.
//
// For example:
//
//	depth = 2 (i.e. third trie level)
//	idx = 13 (which encodes a prefix of length 3 bits within that stride)
//
//	=> PfxBits = 2*8 + 3 = 19
func PfxBits(depth int, idx uint8) uint8 {
	// bits.Len8(idx) gives the number of significant bits (i.e. prefix length + 1)
	// subtract 1 to get the actual prefix length used in this stride
	pfxLenInStride := bits.Len8(idx) - 1

	// Each trie level represents 8 bits (an octet)
	baseBits := depth << 3 // same as depth * 8

	// Total prefix length in bits = full bytes before + prefix bits in this byte
	return uint8(baseBits + pfxLenInStride)
}

// IdxToRange returns the first and last octets covered by a base index.
//
// The base index encodes a prefix of up to 8 bits inside a single stride (octet).
// This function computes the numerical start and end of the value range for that prefix.
//
// For example:
//
//   - A 0-bit prefix (idx == 1) covers the full range:   0..255
//
//   - A 3-bit prefix like 0b101xxxx (idx == 13) covers:  160..191
//
//     idx := PfxToIdx(0b10100000, 3) // 13
//     first, last := IdxToRange(13)  // 160, 191
//
// Internally, this function decodes (octet, prefixLength) via IdxToPfx,
// then computes the broadcast address (last octet) by masking all bits below the prefix length.
func IdxToRange(idx uint8) (first, last uint8) {
	// Decode the prefix base (octet) and length (number of fixed bits)
	first, pfxLen := IdxToPfx(idx)

	// Compute the "broadcast" value by filling trailing (host) bits with 1s.
	// This gives the maximum octet value that still matches the prefix.
	last = first | ^NetMask(pfxLen)

	return
}

// NetMask returns an 8-bit left-aligned network mask for the given number of prefix bits.
//
// For example:
//
//	bits = 0  -> 0b00000000
//	bits = 1  -> 0b10000000
//	bits = 2  -> 0b11000000
//	bits = 3  -> 0b11100000
//	...
//	bits = 8  -> 0b11111111
//
// This mask is used to extract or identify the fixed (prefix) portion of an octet.
// The rightmost (8 - bits) bits are cleared (set to zero), and the upper 'bits' are set to 1.
func NetMask(bits uint8) uint8 {
	// Use a full 8-bit mask and shift left to clear trailing (host) bits.
	// Convert 'bits' to uint16 to avoid overflow in shift for bits == 8.
	return 0b11111111 << (8 - uint16(bits))
}
