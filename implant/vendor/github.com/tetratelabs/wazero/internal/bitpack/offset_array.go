package bitpack

import (
	"math"
)

// OffsetArray is an interface representing read-only views of arrays of 64 bits
// offsets.
type OffsetArray interface {
	// Returns the value at index i.
	//
	// The method complexity may be anywhere between O(1) and O(N).
	Index(i int) uint64
	// Returns the number of offsets in the array.
	//
	// The method complexity must be O(1).
	Len() int
}

// OffsetArrayLen is a helper function to access the length of an offset array.
// It is similar to calling Len on the array but handles the special case where
// the array is nil, in which case it returns zero.
func OffsetArrayLen(array OffsetArray) int {
	if array != nil {
		return array.Len()
	}
	return 0
}

// NewOffsetArray constructs a new array of offsets from the slice of values
// passed as argument. The slice is not retained, the returned array always
// holds a copy of the values.
//
// The underlying implementation of the offset array applies a compression
// mechanism derived from Frame-of-Reference and Delta Encoding to minimize
// the memory footprint of the array. This compression model works best when
// the input is made of ordered values, otherwise the deltas between values
// are likely to be too large to benefit from delta encoding.
//
// See https://lemire.me/blog/2012/02/08/effective-compression-using-frame-of-reference-and-delta-coding/
func NewOffsetArray(values []uint64) OffsetArray {
	if len(values) == 0 {
		return emptyOffsetArray{}
	}
	if len(values) <= smallOffsetArrayCapacity {
		return newSmallOffsetArray(values)
	}

	maxDelta := uint64(0)
	lastValue := values[0]
	// TODO: the pre-processing we perform here can be optimized using SIMD
	// instructions.
	for _, value := range values[1:] {
		if delta := value - lastValue; delta > maxDelta {
			maxDelta = delta
		}
		lastValue = value
	}

	switch {
	case maxDelta > math.MaxUint32:
		return newOffsetArray(values)
	case maxDelta > math.MaxUint16:
		return newDeltaArray[uint32](values)
	case maxDelta > math.MaxUint8:
		return newDeltaArray[uint16](values)
	case maxDelta > 15:
		return newDeltaArray[uint8](values)
	default:
		return newDeltaArrayUint4(values)
	}
}

type offsetArray struct {
	values []uint64
}

func newOffsetArray(values []uint64) *offsetArray {
	a := &offsetArray{
		values: make([]uint64, len(values)),
	}
	copy(a.values, values)
	return a
}

func (a *offsetArray) Index(i int) uint64 {
	return a.values[i]
}

func (a *offsetArray) Len() int {
	return len(a.values)
}

type emptyOffsetArray struct{}

func (emptyOffsetArray) Index(int) uint64 {
	panic("index out of bounds")
}

func (emptyOffsetArray) Len() int {
	return 0
}

const smallOffsetArrayCapacity = 7

type smallOffsetArray struct {
	length int
	values [smallOffsetArrayCapacity]uint64
}

func newSmallOffsetArray(values []uint64) *smallOffsetArray {
	a := &smallOffsetArray{length: len(values)}
	copy(a.values[:], values)
	return a
}

func (a *smallOffsetArray) Index(i int) uint64 {
	if i < 0 || i >= a.length {
		panic("index out of bounds")
	}
	return a.values[i]
}

func (a *smallOffsetArray) Len() int {
	return a.length
}

type uintType interface {
	uint8 | uint16 | uint32 | uint64
}

type deltaArray[T uintType] struct {
	deltas     []T
	firstValue uint64
}

func newDeltaArray[T uintType](values []uint64) *deltaArray[T] {
	a := &deltaArray[T]{
		deltas:     make([]T, len(values)-1),
		firstValue: values[0],
	}
	lastValue := values[0]
	for i, value := range values[1:] {
		a.deltas[i] = T(value - lastValue)
		lastValue = value
	}
	return a
}

func (a *deltaArray[T]) Index(i int) uint64 {
	if i < 0 || i >= a.Len() {
		panic("index out of bounds")
	}
	value := a.firstValue
	// TODO: computing the prefix sum can be vectorized;
	// see https://en.algorithmica.org/hpc/algorithms/prefix/
	for _, delta := range a.deltas[:i] {
		value += uint64(delta)
	}
	return value
}

func (a *deltaArray[T]) Len() int {
	return len(a.deltas) + 1
}

// deltaArrayUint4 is a specialization of deltaArray which packs 4 bits integers
// to hold deltas between 0 and 15; based on the analysis of compiling Python,
// it appeared that most source offset deltas were under 16, so using this
// data structure cuts by 50% the memory needed compared to deltaArray[uint8].
//
// Here is the distribution of source offset deltas for Python 3.13:
//
// - <=15    : 10240
// - <=255   : 9565
// - <=65535 : 1163
//
// Memory profiles showed that using deltaArrayUint4 (compared to deltaArray[T])
// dropped the memory footprint of source mappings for Python from 6MB to 4.5MB.
type deltaArrayUint4 struct {
	deltas     []byte
	numValues  int
	firstValue uint64
}

func newDeltaArrayUint4(values []uint64) *deltaArrayUint4 {
	a := &deltaArrayUint4{
		deltas:     make([]byte, len(values)/2+1),
		numValues:  len(values),
		firstValue: values[0],
	}
	lastValue := values[0]
	for i, value := range values[1:] {
		a.assign(i, value-lastValue)
		lastValue = value
	}
	return a
}

func (a *deltaArrayUint4) assign(i int, v uint64) {
	index, shift := uint(i)>>1, 4*(uint(i)&1)
	a.deltas[index] &= ^(0xF << shift)
	a.deltas[index] |= byte(v) << shift
}

func (a *deltaArrayUint4) index(i int) uint64 {
	index, shift := uint(i)>>1, 4*(uint(i)&1)
	return uint64((a.deltas[index] >> shift) & 0xF)
}

func (a *deltaArrayUint4) Index(i int) uint64 {
	if i < 0 || i >= a.Len() {
		panic("index out of bounds")
	}
	value := a.firstValue
	for j := 0; j < i; j++ {
		value += a.index(j)
	}
	return value
}

func (a *deltaArrayUint4) Len() int {
	return a.numValues
}
