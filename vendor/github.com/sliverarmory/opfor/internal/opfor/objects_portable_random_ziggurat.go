package opfor

import "math"

// This modified-ziggurat sampler is a Go adaptation of the Apache-2.0 Commons
// RNG sampler at commit c961b76e7109cc40216b4f46d49b28c3d5fdd3c2,
// specialized to the OpenJDK RandomGenerator contract and its exact
// BSD-3-Clause generator output attributed in
// objects_portable_random_ziggurat_tables_openjdk.go. Keeping it separate from
// portableJavaRandom's classic cached polar nextGaussian() is important:
// java.util.Random overrides the zero-argument method, but inherits the
// RandomGenerator two-argument Gaussian and exponential defaults.

const (
	portableExponentialLayers       = int64(252)
	portableExponentialX0           = 7.56927469414806264
	portableExponentialConvexMargin = int64(853965788476313645)

	portableNormalLayers        = int64(253)
	portableNormalX0            = 3.63600662550094578
	portableNormalInflection    = 204
	portableNormalConvexMargin  = int64(760463704284035183)
	portableNormalConcaveMargin = int64(2269182951627976012)
)

func (random *portableJavaRandom) nextExponential() float64 {
	return random.nextExponentialSoftCapped(math.Ldexp(portableExponentialX0, 63))
}

func (random *portableJavaRandom) nextExponentialSoftCapped(maxValue float64) float64 {
	u1 := random.nextLong()
	i := int64(uint64(u1) & 0xff)
	if i < portableExponentialLayers {
		return portableExponentialX[i] * float64(uint64(u1)>>1)
	}
	if maxValue <= 0 {
		return 0
	}

	maxExponential := math.Ldexp(portableExponentialX0, 63)
	maxExtraMinusOne := int64(math.MaxInt64)
	if maxValue < maxExponential {
		maxExtraMinusOne = int64(maxValue / portableExponentialX0)
	}
	for extra := int64(0); ; {
		ua := random.nextLong()
		j := int(uint64(ua) & 0xff)
		if ua >= portableExponentialAliasThreshold[j] {
			j = int(portableExponentialAliasMap[j])
		}
		if j > 0 {
			u1 = int64(uint64(u1) >> 1)
			for {
				u2 := int64(uint64(random.nextLong()) >> 1)
				difference := u2 - u1
				if difference < 0 {
					difference = -difference
					u2 = u1
					u1 -= difference
				}
				xBase := portableExponentialX[j] * 0x1p63
				// OpenJDK rounds the interpolation product before adding the
				// scaled base; the conversion forbids Go's permitted FMA fusion.
				xOffset := float64((portableExponentialX[j-1] - portableExponentialX[j]) * float64(u1))
				x := xBase + xOffset
				if difference >= portableExponentialConvexMargin {
					return math.FMA(float64(extra), portableExponentialX0, x)
				}
				yBase := portableExponentialY[j] * 0x1p63
				yOffset := float64((portableExponentialY[j-1] - portableExponentialY[j]) * float64(u2))
				y := yBase + yOffset
				if y <= math.Exp(-x) {
					return math.FMA(float64(extra), portableExponentialX0, x)
				}
				u1 = int64(uint64(random.nextLong()) >> 1)
			}
		}
		if extra == maxExtraMinusOne {
			return maxValue
		}
		extra++
		u1 = random.nextLong()
		i = int64(uint64(u1) & 0xff)
		if i < portableExponentialLayers {
			base := portableExponentialX[i] * float64(uint64(u1)>>1)
			return math.FMA(float64(extra), portableExponentialX0, base)
		}
	}
}

func (random *portableJavaRandom) nextGeneratorGaussian() float64 {
	u1 := random.nextLong()
	i := int64(uint64(u1) & 0xff)
	if i < portableNormalLayers {
		return portableNormalX[i] * float64(u1)
	}

	sign := 1.0
	if u1 < 0 {
		sign = -1.0
	}
	u1 = int64((uint64(u1) << 1) >> 1)
	ua := random.nextLong()
	j := int(uint64(ua) & 0xff)
	if ua >= portableNormalAliasThreshold[j] {
		j = int(portableNormalAliasMap[j])
	}

	var x float64
	switch {
	case j > portableNormalInflection:
		for {
			u2 := int64(uint64(random.nextLong()) >> 1)
			xBase := portableNormalX[j] * 0x1p63
			xOffset := float64((portableNormalX[j-1] - portableNormalX[j]) * float64(u1))
			x = xBase + xOffset
			difference := u2 - u1
			if difference >= 0 {
				break
			}
			if difference > -portableNormalConcaveMargin {
				yBase := portableNormalY[j] * 0x1p63
				yOffset := float64((portableNormalY[j-1] - portableNormalY[j]) * float64(u2))
				if yBase+yOffset <= math.Exp(-0.5*x*x) {
					break
				}
			}
			u1 = int64(uint64(random.nextLong()) >> 1)
		}
	case j == 0:
		for {
			// Preserve OpenJDK's two strict-double operations. The explicit
			// conversions prevent an implementation from retaining extra
			// precision across the reciprocal and multiplication.
			inverseX0 := float64(1.0 / float64(portableNormalX0))
			x = float64(inverseX0 * random.nextExponential())
			limit := 0.5 * x * x
			if random.nextExponentialSoftCapped(limit) >= limit {
				break
			}
		}
		x += portableNormalX0
	case j < portableNormalInflection:
		for {
			u2 := int64(uint64(random.nextLong()) >> 1)
			difference := u2 - u1
			if difference < 0 {
				difference = -difference
				u2 = u1
				u1 -= difference
			}
			xBase := portableNormalX[j] * 0x1p63
			xOffset := float64((portableNormalX[j-1] - portableNormalX[j]) * float64(u1))
			x = xBase + xOffset
			if difference >= portableNormalConvexMargin {
				break
			}
			yBase := portableNormalY[j] * 0x1p63
			yOffset := float64((portableNormalY[j-1] - portableNormalY[j]) * float64(u2))
			if yBase+yOffset <= math.Exp(-0.5*x*x) {
				break
			}
			u1 = int64(uint64(random.nextLong()) >> 1)
		}
	default:
		for {
			u2 := int64(uint64(random.nextLong()) >> 1)
			x = math.FMA(portableNormalX[j-1]-portableNormalX[j], float64(u1), portableNormalX[j]*0x1p63)
			difference := u2 - u1
			if difference >= portableNormalConvexMargin {
				break
			}
			if difference > -portableNormalConcaveMargin {
				y := math.FMA(portableNormalY[j-1]-portableNormalY[j], float64(u2), portableNormalY[j]*0x1p63)
				if y <= math.Exp(-0.5*x*x) {
					break
				}
			}
			u1 = int64(uint64(random.nextLong()) >> 1)
		}
	}
	return sign * x
}
