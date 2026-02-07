package aplib

import "errors"

var ErrCorrupt = errors.New("aplib: corrupt data")

// Depack decompresses a raw aPLib stream (without any safe header).
func Depack(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, ErrCorrupt
	}

	r := bitReader{src: src, pos: 1}
	out := make([]byte, 0, len(src))
	out = append(out, src[0])

	var (
		r0  uint32 = ^uint32(0)
		lwm uint32 = 0
	)

	for {
		bit, ok := r.getBit()
		if !ok {
			return nil, ErrCorrupt
		}

		if bit == 0 {
			b, ok := r.getByte()
			if !ok {
				return nil, ErrCorrupt
			}
			out = append(out, b)
			lwm = 0
			continue
		}

		bit, ok = r.getBit()
		if !ok {
			return nil, ErrCorrupt
		}

		if bit == 0 {
			offs, ok := r.getGamma()
			if !ok {
				return nil, ErrCorrupt
			}

			if lwm == 0 && offs == 2 {
				offs = r0
				ln, ok := r.getGamma()
				if !ok {
					return nil, ErrCorrupt
				}
				if offs == 0 || int(offs) > len(out) {
					return nil, ErrCorrupt
				}
				for i := uint32(0); i < ln; i++ {
					out = append(out, out[len(out)-int(offs)])
				}
			} else {
				if lwm == 0 {
					if offs < 3 {
						return nil, ErrCorrupt
					}
					offs -= 3
				} else {
					if offs < 2 {
						return nil, ErrCorrupt
					}
					offs -= 2
				}

				lo, ok := r.getByte()
				if !ok {
					return nil, ErrCorrupt
				}
				offs = (offs << 8) + uint32(lo)

				ln, ok := r.getGamma()
				if !ok {
					return nil, ErrCorrupt
				}
				if offs >= 32000 {
					ln++
				}
				if offs >= 1280 {
					ln++
				}
				if offs < 128 {
					ln += 2
				}
				if offs == 0 || int(offs) > len(out) {
					return nil, ErrCorrupt
				}
				for i := uint32(0); i < ln; i++ {
					out = append(out, out[len(out)-int(offs)])
				}
				r0 = offs
			}

			lwm = 1
			continue
		}

		bit, ok = r.getBit()
		if !ok {
			return nil, ErrCorrupt
		}
		if bit == 0 {
			b, ok := r.getByte()
			if !ok {
				return nil, ErrCorrupt
			}
			ln := uint32(2 + (b & 1))
			offs := uint32(b >> 1)
			if offs == 0 {
				break
			}
			if int(offs) > len(out) {
				return nil, ErrCorrupt
			}
			for i := uint32(0); i < ln; i++ {
				out = append(out, out[len(out)-int(offs)])
			}
			r0 = offs
			lwm = 1
			continue
		}

		// bit == 1: 1 1 1
		offs := uint32(0)
		for i := 0; i < 4; i++ {
			bit, ok := r.getBit()
			if !ok {
				return nil, ErrCorrupt
			}
			offs = (offs << 1) + uint32(bit)
		}
		if offs != 0 {
			if int(offs) > len(out) {
				return nil, ErrCorrupt
			}
			out = append(out, out[len(out)-int(offs)])
		} else {
			out = append(out, 0)
		}
		lwm = 0
	}

	return out, nil
}

type bitReader struct {
	src      []byte
	pos      int
	tag      byte
	bitCount int
}

func (r *bitReader) getByte() (byte, bool) {
	if r.pos >= len(r.src) {
		return 0, false
	}
	b := r.src[r.pos]
	r.pos++
	return b, true
}

func (r *bitReader) getBit() (uint8, bool) {
	if r.bitCount == 0 {
		if r.pos >= len(r.src) {
			return 0, false
		}
		r.tag = r.src[r.pos]
		r.pos++
		r.bitCount = 8
	}
	bit := (r.tag >> 7) & 1
	r.tag <<= 1
	r.bitCount--
	return bit, true
}

func (r *bitReader) getGamma() (uint32, bool) {
	v := uint32(1)
	for {
		bit, ok := r.getBit()
		if !ok {
			return 0, false
		}
		v = (v << 1) + uint32(bit)
		bit, ok = r.getBit()
		if !ok {
			return 0, false
		}
		if bit == 0 {
			return v, true
		}
	}
}
