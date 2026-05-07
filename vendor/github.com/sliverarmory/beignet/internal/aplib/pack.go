package aplib

import (
	"encoding/binary"
	"errors"
	"math/bits"
)

var (
	ErrEmptyInput = errors.New("aplib: empty input")
	ErrTooLarge   = errors.New("aplib: input too large")
)

// Pack compresses src into the raw aPLib bitstream format expected by aP_depack().
//
// This implements a compatible (not necessarily optimal) encoder which is sufficient
// for the loader to depack staged dylibs.
func Pack(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, ErrEmptyInput
	}
	if uint64(len(src)) > uint64(^uint32(0)) {
		return nil, ErrTooLarge
	}

	w := newBitWriter(src[0])
	lastWasMatch := false // LWM=0 at stream start

	const (
		hashBits    = 16
		minMatchLen = 4
	)

	hashTable := make([]int, 1<<hashBits) // stores pos+1

	hash4 := func(b []byte) uint32 {
		v := binary.LittleEndian.Uint32(b)
		return (v * 2654435761) >> (32 - hashBits)
	}

	// Seed with position 0 if we can hash it.
	if len(src) >= 4 {
		h := hash4(src[:4])
		hashTable[h] = 0 + 1
	}

	pos := 1
	for pos < len(src) {
		bestOff := 0
		bestLen := 0

		if pos+minMatchLen <= len(src) {
			h := hash4(src[pos : pos+4])
			candidate := hashTable[h] - 1
			hashTable[h] = pos + 1

			if candidate >= 0 && candidate < pos {
				off := pos - candidate
				if off > 0 {
					l := 0
					for pos+l < len(src) && src[candidate+l] == src[pos+l] {
						l++
					}
					if l >= minMatchLen {
						bestOff = off
						bestLen = l
					}
				}
			}
		} else if pos+4 <= len(src) {
			h := hash4(src[pos : pos+4])
			hashTable[h] = pos + 1
		}

		if bestLen >= minMatchLen {
			w.writeBit(1)
			w.writeBit(0)

			base := uint32(3)
			if lastWasMatch {
				base = 2
			}

			hi := uint32(bestOff >> 8)
			lo := byte(bestOff & 0xff)

			w.writeGamma(hi + base)
			w.writeByte(lo)

			adj := uint32(0)
			if bestOff >= 32000 {
				adj++
			}
			if bestOff >= 1280 {
				adj++
			}
			if bestOff < 128 {
				adj += 2
			}

			gammaLen := uint32(bestLen) - adj
			if gammaLen < 2 {
				// Should not happen with minMatchLen>=4, but fall back safely.
				w.writeBit(0)
				w.writeByte(src[pos])
				lastWasMatch = false
				pos++
				continue
			}
			w.writeGamma(gammaLen)

			lastWasMatch = true

			end := pos + bestLen
			if end > len(src) {
				end = len(src)
			}
			for p := pos; p < end; p++ {
				if p+4 <= len(src) {
					h := hash4(src[p : p+4])
					hashTable[h] = p + 1
				}
			}
			pos = end
			continue
		}

		w.writeBit(0)
		w.writeByte(src[pos])
		lastWasMatch = false
		pos++
	}

	// End marker: 1 1 0 then offs byte = 0.
	w.writeBit(1)
	w.writeBit(1)
	w.writeBit(0)
	w.writeByte(0)

	return w.out, nil
}

type bitWriter struct {
	out      []byte
	tagIndex int
	bitPos   int
}

func newBitWriter(firstLiteral byte) *bitWriter {
	return &bitWriter{
		out:      []byte{firstLiteral},
		tagIndex: -1,
		bitPos:   -1,
	}
}

func (w *bitWriter) startTag() {
	w.out = append(w.out, 0)
	w.tagIndex = len(w.out) - 1
	w.bitPos = 7
}

func (w *bitWriter) ensureTag() {
	if w.tagIndex < 0 || w.bitPos < 0 {
		w.startTag()
	}
}

func (w *bitWriter) writeBit(bit uint32) {
	w.ensureTag()
	if bit != 0 {
		w.out[w.tagIndex] |= 1 << w.bitPos
	}
	w.bitPos--
}

func (w *bitWriter) writeByte(b byte) {
	w.out = append(w.out, b)
}

// writeGamma writes the "gamma2" coding used by aPLib.
//
// Valid values are >= 2. Values < 2 are not representable.
func (w *bitWriter) writeGamma(v uint32) {
	// aP_getgamma() starts at 1 and runs the body at least once, so the minimum
	// representable value is 2.
	if v < 2 {
		v = 2
	}

	// For each bit after the MSB, output (dataBit, continueBit).
	n := bits.Len32(v)
	for shift := n - 2; shift >= 0; shift-- {
		dataBit := (v >> shift) & 1
		w.writeBit(dataBit)
		continueBit := uint32(0)
		if shift != 0 {
			continueBit = 1
		}
		w.writeBit(continueBit)
	}
}
