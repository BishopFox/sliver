package tun

import (
	"encoding/binary"
	"math/bits"
)

// TODO: Explore SIMD and/or other assembly optimizations.
func checksumNoFold(b []byte, initial uint64) uint64 {
	tmp := make([]byte, 8)
	binary.NativeEndian.PutUint64(tmp, initial)
	ac := binary.BigEndian.Uint64(tmp)
	var carry uint64

	for len(b) >= 128 {
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[:8]), 0)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[8:16]), carry)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[16:24]), carry)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[24:32]), carry)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[32:40]), carry)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[40:48]), carry)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[48:56]), carry)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[56:64]), carry)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[64:72]), carry)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[72:80]), carry)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[80:88]), carry)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[88:96]), carry)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[96:104]), carry)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[104:112]), carry)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[112:120]), carry)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[120:128]), carry)
		ac += carry
		b = b[128:]
	}
	if len(b) >= 64 {
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[:8]), 0)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[8:16]), carry)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[16:24]), carry)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[24:32]), carry)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[32:40]), carry)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[40:48]), carry)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[48:56]), carry)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[56:64]), carry)
		ac += carry
		b = b[64:]
	}
	if len(b) >= 32 {
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[:8]), 0)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[8:16]), carry)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[16:24]), carry)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[24:32]), carry)
		ac += carry
		b = b[32:]
	}
	if len(b) >= 16 {
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[:8]), 0)
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[8:16]), carry)
		ac += carry
		b = b[16:]
	}
	if len(b) >= 8 {
		ac, carry = bits.Add64(ac, binary.NativeEndian.Uint64(b[:8]), 0)
		ac += carry
		b = b[8:]
	}
	if len(b) >= 4 {
		ac, carry = bits.Add64(ac, uint64(binary.NativeEndian.Uint32(b[:4])), 0)
		ac += carry
		b = b[4:]
	}
	if len(b) >= 2 {
		ac, carry = bits.Add64(ac, uint64(binary.NativeEndian.Uint16(b[:2])), 0)
		ac += carry
		b = b[2:]
	}
	if len(b) == 1 {
		tmp := binary.NativeEndian.Uint16([]byte{b[0], 0})
		ac, carry = bits.Add64(ac, uint64(tmp), 0)
		ac += carry
	}

	binary.NativeEndian.PutUint64(tmp, ac)
	return binary.BigEndian.Uint64(tmp)
}

func checksum(b []byte, initial uint64) uint16 {
	ac := checksumNoFold(b, initial)
	ac = (ac >> 16) + (ac & 0xffff)
	ac = (ac >> 16) + (ac & 0xffff)
	ac = (ac >> 16) + (ac & 0xffff)
	ac = (ac >> 16) + (ac & 0xffff)
	return uint16(ac)
}

func pseudoHeaderChecksumNoFold(protocol uint8, srcAddr, dstAddr []byte, totalLen uint16) uint64 {
	sum := checksumNoFold(srcAddr, 0)
	sum = checksumNoFold(dstAddr, sum)
	sum = checksumNoFold([]byte{0, protocol}, sum)
	tmp := make([]byte, 2)
	binary.BigEndian.PutUint16(tmp, totalLen)
	return checksumNoFold(tmp, sum)
}
