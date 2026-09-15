package opfor

import "errors"

// The sorting structure in this file is adapted from the MIT-licensed Go
// TimSort implementation by Mike Kroutikov and its maintained v2 fork at
// github.com/psilva261/timsort, commit
// 790c195bc04e0b7150839446cae66a703631a949. OPFOR uses a typed,
// error-returning comparator so script cancellation and bridge failures remain
// ordinary Go errors. See third_party_licenses/psilva261-timsort-MIT.txt.

const (
	sleepTimSortMinMerge  = 32
	sleepTimSortMinGallop = 7
)

var errSleepComparatorContract = errors.New("Comparison method violates its general contract!")

// sleepStableTimSort is the stable, adaptive merge sort used by the Java
// Collections.sort path underneath Sleep 2.1's BasicStrings sort functions.
// In particular, its merge exhaustion checks preserve the observable
// IllegalArgumentException raised for comparison functions that violate their
// contract. The input is copied so a comparison or cancellation failure cannot
// partially commit a reordered script collection.
func sleepStableTimSort[T any](input []T, compare func(T, T) (int, error)) ([]T, error) {
	values := append([]T(nil), input...)
	remaining := len(values)
	if remaining < 2 {
		return values, nil
	}
	if remaining < sleepTimSortMinMerge {
		run, err := sleepTimSortCountRun(values, 0, remaining, compare)
		if err != nil {
			return nil, err
		}
		if err := sleepTimSortBinaryInsertion(values, 0, remaining, run, compare); err != nil {
			return nil, err
		}
		return values, nil
	}

	sorter := &sleepTimSorter[T]{
		values:    values,
		compare:   compare,
		minGallop: sleepTimSortMinGallop,
	}
	minimumRun := sleepTimSortMinimumRun(remaining)
	start := 0
	for remaining != 0 {
		run, err := sleepTimSortCountRun(values, start, len(values), compare)
		if err != nil {
			return nil, err
		}
		if run < minimumRun {
			forced := minimumRun
			if remaining < forced {
				forced = remaining
			}
			if err := sleepTimSortBinaryInsertion(values, start, start+forced, start+run, compare); err != nil {
				return nil, err
			}
			run = forced
		}
		sorter.pushRun(start, run)
		if err := sorter.mergeCollapse(); err != nil {
			return nil, err
		}
		start += run
		remaining -= run
	}
	if err := sorter.mergeForceCollapse(); err != nil {
		return nil, err
	}
	return values, nil
}

func sleepTimSortBinaryInsertion[T any](
	values []T,
	low int,
	high int,
	start int,
	compare func(T, T) (int, error),
) error {
	if start == low {
		start++
	}
	for ; start < high; start++ {
		pivot := values[start]
		left, right := low, start
		for left < right {
			middle := int(uint(left+right) >> 1)
			order, err := compare(pivot, values[middle])
			if err != nil {
				return err
			}
			if order < 0 {
				right = middle
			} else {
				left = middle + 1
			}
		}
		copy(values[left+1:start+1], values[left:start])
		values[left] = pivot
	}
	return nil
}

func sleepTimSortCountRun[T any](
	values []T,
	low int,
	high int,
	compare func(T, T) (int, error),
) (int, error) {
	runHigh := low + 1
	if runHigh == high {
		return 1, nil
	}
	order, err := compare(values[runHigh], values[low])
	if err != nil {
		return 0, err
	}
	runHigh++
	if order < 0 {
		for runHigh < high {
			order, err = compare(values[runHigh], values[runHigh-1])
			if err != nil {
				return 0, err
			}
			if order >= 0 {
				break
			}
			runHigh++
		}
		for left, right := low, runHigh-1; left < right; left, right = left+1, right-1 {
			values[left], values[right] = values[right], values[left]
		}
	} else {
		for runHigh < high {
			order, err = compare(values[runHigh], values[runHigh-1])
			if err != nil {
				return 0, err
			}
			if order < 0 {
				break
			}
			runHigh++
		}
	}
	return runHigh - low, nil
}

func sleepTimSortMinimumRun(length int) int {
	remainder := 0
	for length >= sleepTimSortMinMerge {
		remainder |= length & 1
		length >>= 1
	}
	return length + remainder
}

type sleepTimRun struct {
	base   int
	length int
}

type sleepTimSorter[T any] struct {
	values    []T
	compare   func(T, T) (int, error)
	minGallop int
	runs      []sleepTimRun
}

func (sorter *sleepTimSorter[T]) pushRun(base, length int) {
	sorter.runs = append(sorter.runs, sleepTimRun{base: base, length: length})
}

func (sorter *sleepTimSorter[T]) mergeCollapse() error {
	for len(sorter.runs) > 1 {
		index := len(sorter.runs) - 2
		if index > 0 && sorter.runs[index-1].length <= sorter.runs[index].length+sorter.runs[index+1].length ||
			index > 1 && sorter.runs[index-2].length <= sorter.runs[index].length+sorter.runs[index-1].length {
			if sorter.runs[index-1].length < sorter.runs[index+1].length {
				index--
			}
		} else if sorter.runs[index].length > sorter.runs[index+1].length {
			break
		}
		if err := sorter.mergeAt(index); err != nil {
			return err
		}
	}
	return nil
}

func (sorter *sleepTimSorter[T]) mergeForceCollapse() error {
	for len(sorter.runs) > 1 {
		index := len(sorter.runs) - 2
		if index > 0 && sorter.runs[index-1].length < sorter.runs[index+1].length {
			index--
		}
		if err := sorter.mergeAt(index); err != nil {
			return err
		}
	}
	return nil
}

func (sorter *sleepTimSorter[T]) mergeAt(index int) error {
	first, second := sorter.runs[index], sorter.runs[index+1]
	sorter.runs[index].length = first.length + second.length
	copy(sorter.runs[index+1:], sorter.runs[index+2:])
	sorter.runs = sorter.runs[:len(sorter.runs)-1]

	skip, err := sleepTimSortGallopRight(
		sorter.values[second.base], sorter.values, first.base, first.length, 0, sorter.compare,
	)
	if err != nil {
		return err
	}
	first.base += skip
	first.length -= skip
	if first.length == 0 {
		return nil
	}
	second.length, err = sleepTimSortGallopLeft(
		sorter.values[first.base+first.length-1],
		sorter.values,
		second.base,
		second.length,
		second.length-1,
		sorter.compare,
	)
	if err != nil || second.length == 0 {
		return err
	}
	if first.length <= second.length {
		return sorter.mergeLow(first.base, first.length, second.base, second.length)
	}
	return sorter.mergeHigh(first.base, first.length, second.base, second.length)
}

func sleepTimSortGallopLeft[T any](
	key T,
	values []T,
	base int,
	length int,
	hint int,
	compare func(T, T) (int, error),
) (int, error) {
	lastOffset, offset := 0, 1
	order, err := compare(key, values[base+hint])
	if err != nil {
		return 0, err
	}
	if order > 0 {
		maximum := length - hint
		for offset < maximum {
			order, err = compare(key, values[base+hint+offset])
			if err != nil {
				return 0, err
			}
			if order <= 0 {
				break
			}
			lastOffset = offset
			offset = (offset << 1) + 1
			if offset <= 0 {
				offset = maximum
			}
		}
		if offset > maximum {
			offset = maximum
		}
		lastOffset += hint
		offset += hint
	} else {
		maximum := hint + 1
		for offset < maximum {
			order, err = compare(key, values[base+hint-offset])
			if err != nil {
				return 0, err
			}
			if order > 0 {
				break
			}
			lastOffset = offset
			offset = (offset << 1) + 1
			if offset <= 0 {
				offset = maximum
			}
		}
		if offset > maximum {
			offset = maximum
		}
		lastOffset, offset = hint-offset, hint-lastOffset
	}
	lastOffset++
	for lastOffset < offset {
		middle := lastOffset + int(uint(offset-lastOffset)>>1)
		order, err = compare(key, values[base+middle])
		if err != nil {
			return 0, err
		}
		if order > 0 {
			lastOffset = middle + 1
		} else {
			offset = middle
		}
	}
	return offset, nil
}

func sleepTimSortGallopRight[T any](
	key T,
	values []T,
	base int,
	length int,
	hint int,
	compare func(T, T) (int, error),
) (int, error) {
	offset, lastOffset := 1, 0
	order, err := compare(key, values[base+hint])
	if err != nil {
		return 0, err
	}
	if order < 0 {
		maximum := hint + 1
		for offset < maximum {
			order, err = compare(key, values[base+hint-offset])
			if err != nil {
				return 0, err
			}
			if order >= 0 {
				break
			}
			lastOffset = offset
			offset = (offset << 1) + 1
			if offset <= 0 {
				offset = maximum
			}
		}
		if offset > maximum {
			offset = maximum
		}
		lastOffset, offset = hint-offset, hint-lastOffset
	} else {
		maximum := length - hint
		for offset < maximum {
			order, err = compare(key, values[base+hint+offset])
			if err != nil {
				return 0, err
			}
			if order < 0 {
				break
			}
			lastOffset = offset
			offset = (offset << 1) + 1
			if offset <= 0 {
				offset = maximum
			}
		}
		if offset > maximum {
			offset = maximum
		}
		lastOffset += hint
		offset += hint
	}
	lastOffset++
	for lastOffset < offset {
		middle := lastOffset + int(uint(offset-lastOffset)>>1)
		order, err = compare(key, values[base+middle])
		if err != nil {
			return 0, err
		}
		if order < 0 {
			offset = middle
		} else {
			lastOffset = middle + 1
		}
	}
	return offset, nil
}

func (sorter *sleepTimSorter[T]) mergeLow(base1, length1, base2, length2 int) error {
	values := sorter.values
	temporary := append([]T(nil), values[base1:base1+length1]...)
	cursor1, cursor2, destination := 0, base2, base1
	values[destination] = values[cursor2]
	destination++
	cursor2++
	length2--
	if length2 == 0 {
		copy(values[destination:destination+length1], temporary[cursor1:cursor1+length1])
		return nil
	}
	if length1 == 1 {
		copy(values[destination:destination+length2], values[cursor2:cursor2+length2])
		values[destination+length2] = temporary[cursor1]
		return nil
	}

	minimumGallop := sorter.minGallop
	for {
		count1, count2 := 0, 0
		for {
			order, err := sorter.compare(values[cursor2], temporary[cursor1])
			if err != nil {
				return err
			}
			if order < 0 {
				values[destination] = values[cursor2]
				destination++
				cursor2++
				count2++
				count1 = 0
				length2--
				if length2 == 0 {
					goto merged
				}
			} else {
				values[destination] = temporary[cursor1]
				destination++
				cursor1++
				count1++
				count2 = 0
				length1--
				if length1 == 1 {
					goto merged
				}
			}
			if (count1 | count2) >= minimumGallop {
				break
			}
		}

		for {
			var err error
			count1, err = sleepTimSortGallopRight(values[cursor2], temporary, cursor1, length1, 0, sorter.compare)
			if err != nil {
				return err
			}
			if count1 != 0 {
				copy(values[destination:destination+count1], temporary[cursor1:cursor1+count1])
				destination += count1
				cursor1 += count1
				length1 -= count1
				if length1 <= 1 {
					goto merged
				}
			}
			values[destination] = values[cursor2]
			destination++
			cursor2++
			length2--
			if length2 == 0 {
				goto merged
			}

			count2, err = sleepTimSortGallopLeft(temporary[cursor1], values, cursor2, length2, 0, sorter.compare)
			if err != nil {
				return err
			}
			if count2 != 0 {
				copy(values[destination:destination+count2], values[cursor2:cursor2+count2])
				destination += count2
				cursor2 += count2
				length2 -= count2
				if length2 == 0 {
					goto merged
				}
			}
			values[destination] = temporary[cursor1]
			destination++
			cursor1++
			length1--
			if length1 == 1 {
				goto merged
			}
			minimumGallop--
			if count1 < sleepTimSortMinGallop && count2 < sleepTimSortMinGallop {
				break
			}
		}
		if minimumGallop < 0 {
			minimumGallop = 0
		}
		minimumGallop += 2
	}

merged:
	if minimumGallop < 1 {
		sorter.minGallop = 1
	} else {
		sorter.minGallop = minimumGallop
	}
	if length1 == 1 {
		copy(values[destination:destination+length2], values[cursor2:cursor2+length2])
		values[destination+length2] = temporary[cursor1]
		return nil
	}
	if length1 == 0 {
		return errSleepComparatorContract
	}
	copy(values[destination:destination+length1], temporary[cursor1:cursor1+length1])
	return nil
}

func (sorter *sleepTimSorter[T]) mergeHigh(base1, length1, base2, length2 int) error {
	values := sorter.values
	temporary := append([]T(nil), values[base2:base2+length2]...)
	cursor1 := base1 + length1 - 1
	cursor2 := length2 - 1
	destination := base2 + length2 - 1
	values[destination] = values[cursor1]
	destination--
	cursor1--
	length1--
	if length1 == 0 {
		copy(values[destination-(length2-1):destination+1], temporary[:length2])
		return nil
	}
	if length2 == 1 {
		destination -= length1
		cursor1 -= length1
		copy(values[destination+1:destination+1+length1], values[cursor1+1:cursor1+1+length1])
		values[destination] = temporary[cursor2]
		return nil
	}

	minimumGallop := sorter.minGallop
	for {
		count1, count2 := 0, 0
		for {
			order, err := sorter.compare(temporary[cursor2], values[cursor1])
			if err != nil {
				return err
			}
			if order < 0 {
				values[destination] = values[cursor1]
				destination--
				cursor1--
				count1++
				count2 = 0
				length1--
				if length1 == 0 {
					goto merged
				}
			} else {
				values[destination] = temporary[cursor2]
				destination--
				cursor2--
				count2++
				count1 = 0
				length2--
				if length2 == 1 {
					goto merged
				}
			}
			if (count1 | count2) >= minimumGallop {
				break
			}
		}

		for {
			position, err := sleepTimSortGallopRight(temporary[cursor2], values, base1, length1, length1-1, sorter.compare)
			if err != nil {
				return err
			}
			count1 = length1 - position
			if count1 != 0 {
				destination -= count1
				cursor1 -= count1
				length1 -= count1
				copy(values[destination+1:destination+1+count1], values[cursor1+1:cursor1+1+count1])
				if length1 == 0 {
					goto merged
				}
			}
			values[destination] = temporary[cursor2]
			destination--
			cursor2--
			length2--
			if length2 == 1 {
				goto merged
			}

			position, err = sleepTimSortGallopLeft(values[cursor1], temporary, 0, length2, length2-1, sorter.compare)
			if err != nil {
				return err
			}
			count2 = length2 - position
			if count2 != 0 {
				destination -= count2
				cursor2 -= count2
				length2 -= count2
				copy(values[destination+1:destination+1+count2], temporary[cursor2+1:cursor2+1+count2])
				if length2 <= 1 {
					goto merged
				}
			}
			values[destination] = values[cursor1]
			destination--
			cursor1--
			length1--
			if length1 == 0 {
				goto merged
			}
			minimumGallop--
			if count1 < sleepTimSortMinGallop && count2 < sleepTimSortMinGallop {
				break
			}
		}
		if minimumGallop < 0 {
			minimumGallop = 0
		}
		minimumGallop += 2
	}

merged:
	if minimumGallop < 1 {
		sorter.minGallop = 1
	} else {
		sorter.minGallop = minimumGallop
	}
	if length2 == 1 {
		destination -= length1
		cursor1 -= length1
		copy(values[destination+1:destination+1+length1], values[cursor1+1:cursor1+1+length1])
		values[destination] = temporary[cursor2]
		return nil
	}
	if length2 == 0 {
		return errSleepComparatorContract
	}
	copy(values[destination-(length2-1):destination+1], temporary[:length2])
	return nil
}
