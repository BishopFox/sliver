package farm

import (
	"strconv"
	"testing"
)

var res32 uint32
var res64 uint64
var res64lo, res64hi uint64

// 256-bytes random string
var buf = make([]byte, 8192)

var sizes = []int{8, 16, 32, 40, 60, 64, 72, 80, 100, 150, 200, 250, 512, 1024, 8192}

func BenchmarkHash32(b *testing.B) {
	var r uint32

	for _, n := range sizes {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			b.SetBytes(int64(n))
			for i := 0; i < b.N; i++ {
				// record the result to prevent the compiler eliminating the function call
				r = Hash32(buf[:n])
			}
			// store the result to a package level variable so the compiler cannot eliminate the Benchmark itself
			res32 = r
		})
	}

}

func BenchmarkFingerprint32(b *testing.B) {
	var r uint32

	for _, n := range sizes {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			b.SetBytes(int64(n))
			for i := 0; i < b.N; i++ {
				// record the result to prevent the compiler eliminating the function call
				r = Fingerprint32(buf[:n])
			}
			// store the result to a package level variable so the compiler cannot eliminate the Benchmark itself
			res32 = r
		})
	}

}

func BenchmarkHash64(b *testing.B) {
	var r uint64

	for _, n := range sizes {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				r = Hash64(buf[:n])
			}
			res64 = r
		})
	}

}

func BenchmarkFingerprint64(b *testing.B) {
	var r uint64

	for _, n := range sizes {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			b.SetBytes(int64(n))
			for i := 0; i < b.N; i++ {
				r = Fingerprint64(buf[:n])
			}
			res64 = r
		})
	}
}

func BenchmarkHash128(b *testing.B) {
	var rlo, rhi uint64

	for _, n := range sizes {
		b.Run(strconv.Itoa(n), func(b *testing.B) {

			for i := 0; i < b.N; i++ {
				rlo, rhi = Hash128(buf)
			}
			res64lo = rlo
			res64hi = rhi

		})
	}
}
