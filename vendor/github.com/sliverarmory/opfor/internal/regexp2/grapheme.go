package regexp2

// The grapheme engine follows the Unicode 17 implementation used by the
// pinned OpenJDK Grapheme helper. Its generated classifications deliberately
// do not depend on the Go toolchain's Unicode tables.

const (
	graphemeOther = iota
	graphemeCR
	graphemeLF
	graphemeControl
	graphemeExtend
	graphemeZWJ
	graphemeRI
	graphemePrepend
	graphemeSpacingMark
	graphemeL
	graphemeV
	graphemeT
	graphemeLV
	graphemeLVT
	graphemeExtendedPictographic
)

const (
	indicNone = iota
	indicExtend
	indicLinker
	indicConsonant
)

type javaGraphemeRange struct {
	lo    rune
	hi    rune
	value uint8
}

type graphemeTimeoutCheck struct {
	steps int
	check func() error
}

func (c *graphemeTimeoutCheck) pulse() error {
	if c == nil || c.check == nil {
		return nil
	}
	c.steps++
	if c.steps&1023 == 0 {
		return c.check()
	}
	return nil
}

func javaGraphemeClassOf(cp rune) uint8 {
	return javaGraphemeRangeValue(javaGraphemeClassRanges, cp, graphemeOther)
}

func javaIndicConjunctClassOf(cp rune) uint8 {
	return javaGraphemeRangeValue(javaIndicConjunctRanges, cp, indicNone)
}

func javaGraphemeRangeValue(ranges []javaGraphemeRange, cp rune, fallback uint8) uint8 {
	left, right := 0, len(ranges)
	for left < right {
		middle := left + (right-left)/2
		if ranges[middle].hi < cp {
			left = middle + 1
		} else {
			right = middle
		}
	}
	if left < len(ranges) && ranges[left].lo <= cp {
		return ranges[left].value
	}
	return fallback
}

func nextJavaGraphemeBoundary(src []rune, off, limit int, check func() error) (int, error) {
	if off < 0 || off >= limit || limit > len(src) {
		return off, nil
	}
	timeout := graphemeTimeoutCheck{check: check}
	ch0 := src[off]
	t0 := javaGraphemeClassOf(ch0)
	ret := off + 1
	riCount := 0
	if t0 == graphemeRI {
		riCount = 1
	}
	gb11 := t0 == graphemeExtendedPictographic

	for ret < limit {
		if err := timeout.pulse(); err != nil {
			return 0, err
		}
		ch1 := src[ret]
		t1 := javaGraphemeClassOf(ch1)

		// GB9c: Consonant (Extend|Linker)* Linker
		//       (Extend|Linker)* x Consonant.
		if javaIndicConjunctClassOf(ch0) == indicConsonant {
			advance, ok, err := javaIndicConjunctAdvance(src, ret, limit, &timeout)
			if err != nil {
				return 0, err
			}
			if ok {
				ret += advance
				continue
			}
		}

		switch {
		case gb11 && t0 == graphemeZWJ && t1 == graphemeExtendedPictographic:
			// GB11 continues the cluster.
		case riCount%2 == 1 && t0 == graphemeRI && t1 == graphemeRI:
			// GB12/GB13 pair regional indicators from the cluster start.
		case javaGraphemeBreaks(t0, t1):
			return ret, nil
		}

		if t1 == graphemeRI {
			riCount++
		}
		ch0 = ch1
		t0 = t1
		ret++
	}
	return ret, nil
}

func javaIndicConjunctAdvance(src []rune, index, limit int, timeout *graphemeTimeoutCheck) (int, bool, error) {
	linkerFound := false
	advance := 0
	for index+advance < limit {
		if err := timeout.pulse(); err != nil {
			return 0, false, err
		}
		class := javaIndicConjunctClassOf(src[index+advance])
		advance++
		switch class {
		case indicLinker:
			linkerFound = true
		case indicConsonant:
			return advance, linkerFound, nil
		case indicExtend:
			// Continue looking for the linking consonant.
		default:
			return 0, false, nil
		}
	}
	return 0, false, nil
}

func javaGraphemeBreaks(left, right uint8) bool {
	// GB3.
	if left == graphemeCR && right == graphemeLF {
		return false
	}
	// GB4 and GB5 take precedence over the continuation rules.
	if isJavaGraphemeControl(left) || isJavaGraphemeControl(right) {
		return true
	}
	// GB6.
	if left == graphemeL &&
		(right == graphemeL || right == graphemeV || right == graphemeLV || right == graphemeLVT) {
		return false
	}
	// GB7.
	if (left == graphemeLV || left == graphemeV) && (right == graphemeV || right == graphemeT) {
		return false
	}
	// GB8.
	if (left == graphemeLVT || left == graphemeT) && right == graphemeT {
		return false
	}
	// GB9, GB9a, and GB9b.
	if right == graphemeExtend || right == graphemeZWJ || right == graphemeSpacingMark ||
		left == graphemePrepend {
		return false
	}
	return true // GB999.
}

func isJavaGraphemeControl(class uint8) bool {
	return class == graphemeCR || class == graphemeLF || class == graphemeControl
}
