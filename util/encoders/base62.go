package encoders

/*
	MIT License

	Copyright (c) 2019 Shawn Wang <jxskiss@126.com>

	Permission is hereby granted, free of charge, to any person obtaining a copy
	of this software and associated documentation files (the "Software"), to deal
	in the Software without restriction, including without limitation the rights
	to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
	copies of the Software, and to permit persons to whom the Software is
	furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice shall be included in all
	copies or substantial portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
	IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
	AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
	LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
	OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
	SOFTWARE.
	-------------------------------------------------------------------------------
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
	-------------------------------------------------------------------------------

	Modifications by moloch to work with Sliver

*/

import (
	"math/bits"
	"strconv"
)

const (
	base        = 62
	compactMask = 0x1E // 00011110
	mask5bits   = 0x1F // 00011111
)

// Base62EncoderID - EncoderID
const Base62EncoderID = 43

// Base62 Encoder
type Base62 struct{}

const base62Alphabet = "aBCDEFGHIJKLMNOPQRSTUVWXYZAbcdefghijklmnopqrstuvwxyz0123456789"

// Encode - Base62 Encode
func (e Base62) Encode(data []byte) []byte {
	return []byte(NewEncoding(base62Alphabet).EncodeToString(data))
}

// Decode - Base62 Decode
func (e Base62) Decode(data []byte) ([]byte, error) {
	return NewEncoding(base62Alphabet).DecodeString(string(data))
}

type Base62Encoder struct {
	encode    [base]byte
	decodeMap [256]byte
}

func NewEncoding(encoder string) *Base62Encoder {
	if len(encoder) != base {
		panic("encoding alphabet is not 62-bytes long")
	}
	for i := 0; i < len(encoder); i++ {
		if encoder[i] == '\n' || encoder[i] == '\r' {
			panic("encoding alphabet contains newline character")
		}
	}

	e := new(Base62Encoder)
	copy(e.encode[:], encoder)
	for i := 0; i < len(e.decodeMap); i++ {
		e.decodeMap[i] = 0xFF
	}
	for i := 0; i < len(encoder); i++ {
		e.decodeMap[encoder[i]] = byte(i)
	}
	return e
}

func (enc *Base62Encoder) Encode(src []byte) []byte {
	if len(src) == 0 {
		return []byte{}
	}
	encoder := newEncoder(src)
	return encoder.encode(enc.encode[:])
}

func (enc *Base62Encoder) EncodeToString(src []byte) string {
	ret := enc.Encode(src)
	return string(ret)
}

type encoder struct {
	b   []byte
	pos int
}

func newEncoder(src []byte) *encoder {
	return &encoder{
		b:   src,
		pos: len(src) * 8,
	}
}

func (enc *encoder) next() (byte, bool) {
	var i, pos int
	var j, blen byte
	pos = enc.pos - 6
	if pos <= 0 {
		pos = 0
		blen = byte(enc.pos)
	} else {
		i = pos / 8
		j = byte(pos % 8)
		blen = byte((i+1)*8 - pos)
		if blen > 6 {
			blen = 6
		}
	}
	shift := 8 - j - blen
	b := enc.b[i] >> shift & (1<<blen - 1)

	if blen < 6 && pos > 0 {
		blen1 := 6 - blen
		b = b<<blen1 | enc.b[i+1]>>(8-blen1)
	}
	if b&compactMask == compactMask {
		if pos > 0 || b > mask5bits {
			pos++
		}
		b &= mask5bits
	}
	enc.pos = pos

	return b, pos > 0
}

func (enc *encoder) encode(encTable []byte) []byte {
	ret := make([]byte, 0, len(enc.b)*8/5+1)
	x, hasMore := enc.next()
	for {
		ret = append(ret, encTable[x])
		if !hasMore {
			break
		}
		x, hasMore = enc.next()
	}
	return ret
}

type CorruptInputError int64

func (e CorruptInputError) Error() string {
	return "illegal base62 data at input byte " + strconv.FormatInt(int64(e), 10)
}

func (enc *Base62Encoder) Decode(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return []byte{}, nil
	}
	dec := decoder(src)
	return dec.decode(enc.decodeMap[:])
}

func (enc *Base62Encoder) DecodeString(src string) ([]byte, error) {
	return enc.Decode([]byte(src))
}

type decoder []byte

func (dec decoder) decode(decTable []byte) ([]byte, error) {
	ret := make([]byte, len(dec)*6/8+1)
	idx := len(ret)
	pos := byte(0)
	b := 0
	for i, c := range dec {
		x := decTable[c]
		if x == 0xFF {
			return nil, CorruptInputError(i)
		}
		if i == len(dec)-1 {
			b |= int(x) << pos
			pos += byte(bits.Len8(x))
		} else if x&compactMask == compactMask {
			b |= int(x) << pos
			pos += 5
		} else {
			b |= int(x) << pos
			pos += 6
		}
		if pos >= 8 {
			idx--
			ret[idx] = byte(b)
			pos %= 8
			b >>= 8
		}
	}
	if pos > 0 {
		idx--
		ret[idx] = byte(b)
	}

	return ret[idx:], nil
}
