package opfor

import (
	"context"
	"fmt"
	"math"
)

// The line and escape algorithms in this file are pinned to OpenJDK 17u
// commit 352633b5cef98ef3de7e562751222c38d76bb319. They intentionally operate
// on Java UTF-16 code units rather than Go runes.

type portableJavaStringLine struct {
	start int
	end   int
}

type portableJavaStringWork struct {
	ctx        context.Context
	iterations uint64
}

func (work *portableJavaStringWork) advance(count int) error {
	for count > 0 {
		const chunkSize = uint64(portableJavaStringNativeLoopChunk)
		if work.iterations%chunkSize == 0 {
			if err := executionContextError(work.ctx); err != nil {
				return err
			}
			if err := consumeInstruction(work.ctx); err != nil {
				return err
			}
		}
		chunk := int(chunkSize - work.iterations%chunkSize)
		if chunk > count {
			chunk = count
		}
		work.iterations += uint64(chunk)
		count -= chunk
	}
	return nil
}

func (work *portableJavaStringWork) finish() error {
	return executionContextError(work.ctx)
}

func portableJavaStringIndent(
	ctx context.Context,
	invocation ObjectInvocation,
	target Value,
) (Value, bool, error) {
	if len(invocation.Arguments) != 1 {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	units := sleepStringUnits(target)
	if len(units) == 0 {
		return String(""), true, nil
	}
	raw := sleepStringRawMask(target)

	work := &portableJavaStringWork{ctx: ctx}
	lines, err := portableJavaStringScanLines(work, units)
	if err != nil {
		return Null(), true, err
	}
	n := sleepInt32(invocation.Arg(0))
	resultLength := int64(len(lines))
	for _, line := range lines {
		if err := work.advance(1); err != nil {
			return Null(), true, err
		}
		resultLength += int64(line.end - line.start)
	}
	if n > 0 {
		resultLength += int64(n) * int64(len(lines))
	}
	if resultLength > math.MaxInt32 {
		return Null(), true, fmt.Errorf("java.lang.OutOfMemoryError: Required length exceeds implementation limit")
	}

	builder := newPortableJavaStringBuilder(int(resultLength))
	for _, line := range lines {
		start := line.start
		switch {
		case n > 0:
			if err := portableJavaStringAppendSpaces(work, builder, int(n)); err != nil {
				return Null(), true, err
			}
		case n < 0:
			leading, err := portableJavaStringLeadingWhitespace(work, units, line.start, line.end)
			if err != nil {
				return Null(), true, err
			}
			remove := leading - line.start
			if n != math.MinInt32 && remove > int(-n) {
				remove = int(-n)
			}
			start += remove
		}
		if err := portableJavaStringAppendRange(work, builder, units, raw, start, line.end); err != nil {
			return Null(), true, err
		}
		if err := portableJavaStringAppendGenerated(work, builder, []uint16{'\n'}); err != nil {
			return Null(), true, err
		}
	}
	if err := work.finish(); err != nil {
		return Null(), true, err
	}
	return builder.value(), true, nil
}

func portableJavaStringStripIndent(
	ctx context.Context,
	invocation ObjectInvocation,
	target Value,
) (Value, bool, error) {
	if len(invocation.Arguments) != 0 {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	units := sleepStringUnits(target)
	if len(units) == 0 {
		return String(""), true, nil
	}
	raw := sleepStringRawMask(target)

	work := &portableJavaStringWork{ctx: ctx}
	lines, err := portableJavaStringScanLines(work, units)
	if err != nil {
		return Null(), true, err
	}
	type bounds struct {
		first int
		last  int
	}
	whitespace := make([]bounds, len(lines))
	for index, line := range lines {
		if err := work.advance(1); err != nil {
			return Null(), true, err
		}
		first, last, err := portableJavaStringWhitespaceBounds(work, units, line.start, line.end)
		if err != nil {
			return Null(), true, err
		}
		whitespace[index] = bounds{first: first, last: last}
	}

	lastUnit := units[len(units)-1]
	terminalLineEnding := lastUnit == '\n' || lastUnit == '\r'
	outdent := 0
	if !terminalLineEnding {
		outdent = len(units)
		for index, line := range lines {
			if err := work.advance(1); err != nil {
				return Null(), true, err
			}
			if whitespace[index].first != line.end {
				outdent = min(outdent, whitespace[index].first-line.start)
			}
		}
		lastLine := lines[len(lines)-1]
		if whitespace[len(lines)-1].first == lastLine.end {
			outdent = min(outdent, lastLine.end-lastLine.start)
		}
	}

	builder := newPortableJavaStringBuilder(len(units))
	for index, line := range lines {
		if err := work.advance(1); err != nil {
			return Null(), true, err
		}
		if index != 0 {
			if err := portableJavaStringAppendGenerated(work, builder, []uint16{'\n'}); err != nil {
				return Null(), true, err
			}
		}
		first, last := whitespace[index].first, whitespace[index].last
		if first > last {
			continue
		}
		start := line.start + min(outdent, first-line.start)
		if err := portableJavaStringAppendRange(work, builder, units, raw, start, last); err != nil {
			return Null(), true, err
		}
	}
	if terminalLineEnding {
		if err := portableJavaStringAppendGenerated(work, builder, []uint16{'\n'}); err != nil {
			return Null(), true, err
		}
	}
	if err := work.finish(); err != nil {
		return Null(), true, err
	}
	return builder.value(), true, nil
}

func portableJavaStringTranslateEscapes(
	ctx context.Context,
	invocation ObjectInvocation,
	target Value,
) (Value, bool, error) {
	if len(invocation.Arguments) != 0 {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	units := sleepStringUnits(target)
	if len(units) == 0 {
		return String(""), true, nil
	}
	raw := sleepStringRawMask(target)
	resultUnits := make([]uint16, 0, len(units))
	resultRaw := make([]bool, 0, len(units))
	work := &portableJavaStringWork{ctx: ctx}
	changed := false
	for from := 0; from < len(units); {
		if err := work.advance(1); err != nil {
			return Null(), true, err
		}
		character := units[from]
		characterRaw := raw[from]
		from++
		if character != '\\' {
			resultUnits = append(resultUnits, character)
			resultRaw = append(resultRaw, characterRaw)
			continue
		}

		changed = true
		character = 0
		if from < len(units) {
			if err := work.advance(1); err != nil {
				return Null(), true, err
			}
			character = units[from]
			from++
		}
		switch character {
		case 'b':
			character = '\b'
		case 'f':
			character = '\f'
		case 'n':
			character = '\n'
		case 'r':
			character = '\r'
		case 's':
			character = ' '
		case 't':
			character = '\t'
		case '\'', '"', '\\':
			// The escaped character is retained as-is.
		case '0', '1', '2', '3', '4', '5', '6', '7':
			limit := from + 1
			if character <= '3' {
				limit++
			}
			limit = min(limit, len(units))
			code := int(character - '0')
			for from < limit {
				next := units[from]
				if next < '0' || next > '7' {
					break
				}
				if err := work.advance(1); err != nil {
					return Null(), true, err
				}
				from++
				code = code<<3 | int(next-'0')
			}
			character = uint16(code)
		case '\n':
			continue
		case '\r':
			if from < len(units) && units[from] == '\n' {
				if err := work.advance(1); err != nil {
					return Null(), true, err
				}
				from++
			}
			continue
		default:
			escape := sleepRenderStringUnits([]uint16{character}, []bool{false})
			return Null(), true, fmt.Errorf(
				"java.lang.IllegalArgumentException: Invalid escape sequence: \\%s \\\\u%04X",
				escape,
				character,
			)
		}
		resultUnits = append(resultUnits, character)
		resultRaw = append(resultRaw, false)
	}
	if err := work.finish(); err != nil {
		return Null(), true, err
	}
	if !changed {
		return target, true, nil
	}
	return sleepStringValueFromUnits(resultUnits, resultRaw), true, nil
}

func portableJavaStringScanLines(work *portableJavaStringWork, units []uint16) ([]portableJavaStringLine, error) {
	lines := make([]portableJavaStringLine, 0)
	for cursor := 0; cursor < len(units); {
		start := cursor
		for cursor < len(units) && units[cursor] != '\n' && units[cursor] != '\r' {
			if err := work.advance(1); err != nil {
				return nil, err
			}
			cursor++
		}
		lines = append(lines, portableJavaStringLine{start: start, end: cursor})
		if cursor == len(units) {
			break
		}
		separator := units[cursor]
		if err := work.advance(1); err != nil {
			return nil, err
		}
		cursor++
		if separator == '\r' && cursor < len(units) && units[cursor] == '\n' {
			if err := work.advance(1); err != nil {
				return nil, err
			}
			cursor++
		}
	}
	return lines, nil
}

func portableJavaStringLeadingWhitespace(
	work *portableJavaStringWork,
	units []uint16,
	start, end int,
) (int, error) {
	index := start
	for index < end {
		codePoint, width := sleepUTF16CodePointAt(units[:end], index)
		if err := work.advance(width); err != nil {
			return 0, err
		}
		if codePoint != ' ' && codePoint != '\t' && !sleepJavaIsWhitespace(rune(codePoint)) {
			break
		}
		index += width
	}
	return index, nil
}

func portableJavaStringWhitespaceBounds(
	work *portableJavaStringWork,
	units []uint16,
	start, end int,
) (int, int, error) {
	first, err := portableJavaStringLeadingWhitespace(work, units, start, end)
	if err != nil {
		return 0, 0, err
	}
	last := end
	for last > start {
		codePoint, width := sleepUTF16CodePointBefore(units, last)
		if err := work.advance(width); err != nil {
			return 0, 0, err
		}
		if codePoint != ' ' && codePoint != '\t' && !sleepJavaIsWhitespace(rune(codePoint)) {
			break
		}
		last -= width
	}
	return first, last, nil
}

func portableJavaStringAppendRange(
	work *portableJavaStringWork,
	builder *portableJavaStringBuilder,
	units []uint16,
	raw []bool,
	start, end int,
) error {
	for cursor := start; cursor < end; {
		next := min(cursor+portableJavaStringNativeLoopChunk, end)
		if err := work.advance(next - cursor); err != nil {
			return err
		}
		if err := builder.append(units[cursor:next], raw[cursor:next]); err != nil {
			return err
		}
		cursor = next
	}
	return nil
}

func portableJavaStringAppendGenerated(
	work *portableJavaStringWork,
	builder *portableJavaStringBuilder,
	units []uint16,
) error {
	if err := work.advance(len(units)); err != nil {
		return err
	}
	return builder.append(units, nil)
}

func portableJavaStringAppendSpaces(
	work *portableJavaStringWork,
	builder *portableJavaStringBuilder,
	count int,
) error {
	chunk := make([]uint16, min(count, 4096))
	for index := range chunk {
		chunk[index] = ' '
	}
	for count > 0 {
		amount := min(count, len(chunk))
		if err := portableJavaStringAppendGenerated(work, builder, chunk[:amount]); err != nil {
			return err
		}
		count -= amount
	}
	return nil
}
