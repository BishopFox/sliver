package opfor

import (
	"context"
	"errors"
	"fmt"
)

const (
	sleepTrDelete = 1 << iota
	sleepTrComplement
	sleepTrSqueeze
	sleepTrDone = uint16(0xffff)
)

type sleepTrElement struct {
	item        uint16
	replacement uint16
	replaceRaw  bool
	special     bool
}

// sleepTrPatternSyntaxError mirrors the message exposed by
// java.util.regex.PatternSyntaxException. Transliteration uses that exception
// for its pattern grammar even though it does not use java.util.regex.Pattern.
type sleepTrPatternSyntaxError struct {
	description string
	pattern     string
	index       int
}

func (err *sleepTrPatternSyntaxError) Error() string {
	if err == nil {
		return ""
	}
	return formatSleepJavaPatternSyntaxMessage(err.description, err.pattern, err.index)
}

type sleepTrRangeSyntaxError struct {
	index int
}

func (err *sleepTrRangeSyntaxError) Error() string {
	return fmt.Sprintf("Dangling range operator '-' near index %d", err.index)
}

func builtinSleepTr(ctx context.Context, invocation Invocation) (Value, error) {
	if err := ctx.Err(); err != nil {
		return Null(), err
	}
	text := sleepStringCoercion(invocation.Arg(0))
	pattern := sleepStringCoercion(invocation.Arg(1))
	mapper := sleepStringCoercion(invocation.Arg(2))
	optionsText := invocation.Arg(3).String()
	options := 0
	for _, option := range optionsText {
		switch option {
		case 'c':
			options |= sleepTrComplement
		case 'd':
			options |= sleepTrDelete
		case 's':
			options |= sleepTrSqueeze
		}
	}
	elements, err := compileSleepTransliteration(ctx, pattern, mapper, options)
	if err != nil {
		if ctx.Err() != nil {
			return Null(), ctx.Err()
		}
		var syntaxError *sleepTrPatternSyntaxError
		if errors.As(err, &syntaxError) {
			return Null(), sleepBridgeIllegalArgument(syntaxError.Error())
		}
		return Null(), fmt.Errorf("&%s: %w", builtinName(invocation.Name), err)
	}
	return applySleepTransliteration(ctx, text, elements, options)
}

func compileSleepTransliteration(ctx context.Context, pattern, mapper Value, options int) ([]sleepTrElement, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	matches, err := expandSleepTrRanges(ctx, sleepStringUnits(pattern))
	if err != nil {
		var rangeError *sleepTrRangeSyntaxError
		if errors.As(err, &rangeError) {
			return nil, &sleepTrPatternSyntaxError{
				description: "Dangling range operator '-'",
				pattern:     sleepCanonicalString(pattern),
				index:       rangeError.index,
			}
		}
		return nil, err
	}
	replacements, replacementRaw, err := expandSleepTrRangesWithRaw(ctx, sleepStringUnits(mapper), sleepStringRawMask(mapper))
	if err != nil {
		var rangeError *sleepTrRangeSyntaxError
		if errors.As(err, &rangeError) {
			return nil, &sleepTrPatternSyntaxError{
				description: "Dangling range operator '-'",
				pattern:     sleepCanonicalString(mapper),
				index:       rangeError.index,
			}
		}
		return nil, err
	}

	elements := make([]sleepTrElement, 0, len(matches))
	replacementIndex := 0
	for matchIndex := 0; matchIndex < len(matches); matchIndex++ {
		if err := sleepTrContextErr(ctx, matchIndex); err != nil {
			return nil, err
		}
		item := matches[matchIndex]
		if item == sleepTrDone {
			break
		}
		replacement := sleepTrIteratorCurrent(replacements, replacementIndex)
		element := sleepTrElement{
			item:        item,
			replacement: replacement,
			replaceRaw:  replacementIndex >= 0 && replacementIndex < len(replacementRaw) && replacementRaw[replacementIndex],
			special:     item == '.',
		}
		if item == '\\' {
			matchIndex++
			if matchIndex >= len(matches) || matches[matchIndex] == sleepTrDone {
				return nil, &sleepTrPatternSyntaxError{
					description: "attempting to escape end of pattern string",
					pattern:     sleepCanonicalString(pattern),
					index:       len(matches) - 1,
				}
			}
			element.item = matches[matchIndex]
			if !sleepTrEscapeAllowed(element.item) {
				character := sleepCanonicalString(sleepStringValueFromUnits([]uint16{element.item}, nil))
				return nil, &sleepTrPatternSyntaxError{
					description: fmt.Sprintf("unrecognized escaped meta-character '%s'", character),
					pattern:     sleepCanonicalString(pattern),
					index:       matchIndex,
				}
			}
			element.special = element.item != '\\' && element.item != '.' && element.item != '-'
		}
		elements = append(elements, element)

		replacementIndex++
		if sleepTrIteratorCurrent(replacements, replacementIndex) == sleepTrDone && options&sleepTrDelete == 0 {
			if len(replacements) == 0 {
				replacementIndex = 0
			} else {
				replacementIndex = len(replacements) - 1
			}
		}
	}
	return elements, nil
}

func expandSleepTrRangesWithRaw(ctx context.Context, value []uint16, raw []bool) ([]uint16, []bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	result := append([]uint16(nil), value...)
	resultRaw := append([]bool(nil), raw...)
	if len(resultRaw) != len(result) {
		resultRaw = make([]bool, len(result))
	}
	for index := 0; index < len(result); index++ {
		if err := sleepTrContextErr(ctx, index); err != nil {
			return nil, nil, err
		}
		if result[index] == '\\' {
			index++
			continue
		}
		if result[index] != '-' {
			continue
		}
		if index <= 0 || index >= len(result)-1 {
			return nil, nil, &sleepTrRangeSyntaxError{index: len(value) - 1}
		}
		rangeValues, err := sleepTrRange(ctx, result[index-1], result[index+1])
		if err != nil {
			return nil, nil, err
		}
		rangeRaw := make([]bool, len(rangeValues))
		if resultRaw[index-1] && resultRaw[index+1] {
			for position := range rangeRaw {
				rangeRaw[position] = true
			}
		}
		expanded := make([]uint16, 0, len(result)-2+len(rangeValues))
		expanded = append(expanded, result[:index-1]...)
		expanded = append(expanded, rangeValues...)
		expanded = append(expanded, result[index+1:]...)
		expandedRaw := make([]bool, 0, len(resultRaw)-2+len(rangeRaw))
		expandedRaw = append(expandedRaw, resultRaw[:index-1]...)
		expandedRaw = append(expandedRaw, rangeRaw...)
		expandedRaw = append(expandedRaw, resultRaw[index+1:]...)
		result, resultRaw = expanded, expandedRaw
		index += len(rangeValues) - 2
	}
	return result, resultRaw, nil
}

func sleepTrIteratorCurrent(value []uint16, index int) uint16 {
	if index < 0 || index >= len(value) || value[index] == sleepTrDone {
		return sleepTrDone
	}
	return value[index]
}

func expandSleepTrRanges(ctx context.Context, value []uint16) ([]uint16, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	result := append([]uint16(nil), value...)
	for index := 0; index < len(result); index++ {
		if err := sleepTrContextErr(ctx, index); err != nil {
			return nil, err
		}
		if result[index] == '\\' {
			index++
			continue
		}
		if result[index] != '-' {
			continue
		}
		if index <= 0 || index >= len(result)-1 {
			return nil, &sleepTrRangeSyntaxError{index: len(value) - 1}
		}
		rangeValues, err := sleepTrRange(ctx, result[index-1], result[index+1])
		if err != nil {
			return nil, err
		}
		expanded := make([]uint16, 0, len(result)-2+len(rangeValues))
		expanded = append(expanded, result[:index-1]...)
		expanded = append(expanded, rangeValues...)
		expanded = append(expanded, result[index+1:]...)
		result = expanded
		index += len(rangeValues) - 2
	}
	return result, nil
}

func sleepTrRange(ctx context.Context, start, end uint16) ([]uint16, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if start == end {
		return nil, nil
	}
	result := make([]uint16, 0, sleepTrRangeCapacity(start, end))
	if start < end {
		for current := start; current < end; current++ {
			if err := sleepTrContextErr(ctx, int(current-start)); err != nil {
				return nil, err
			}
			result = append(result, current)
		}
		return result, nil
	}
	for current := start; current > end; current-- {
		if err := sleepTrContextErr(ctx, int(start-current)); err != nil {
			return nil, err
		}
		result = append(result, current)
	}
	return result, nil
}

func sleepTrRangeCapacity(start, end uint16) int {
	if start < end {
		return int(end - start)
	}
	return int(start - end)
}

func sleepTrEscapeAllowed(value uint16) bool {
	switch value {
	case 'd', 'D', 's', 'S', 'w', 'W', '.', '\\', '-':
		return true
	default:
		return false
	}
}

func applySleepTransliteration(ctx context.Context, text Value, elements []sleepTrElement, options int) (Value, error) {
	if err := ctx.Err(); err != nil {
		return Null(), err
	}
	input := sleepStringUnits(text)
	inputRaw := sleepStringRawMask(text)
	output := make([]uint16, 0, len(input))
	outputRaw := make([]bool, 0, len(inputRaw))
	for index := 0; index < len(input); index++ {
		if err := sleepTrContextErr(ctx, index); err != nil {
			return Null(), err
		}
		current := input[index]
		matched := false
		for elementIndex, element := range elements {
			if err := sleepTrContextErr(ctx, elementIndex); err != nil {
				return Null(), err
			}
			if !sleepTrMatches(current, element, options) {
				continue
			}
			if element.replacement != sleepTrDone {
				output = append(output, element.replacement)
				outputRaw = append(outputRaw, element.replaceRaw)
			}
			if options&sleepTrSqueeze != 0 {
				for index+1 < len(input) && input[index+1] == current {
					index++
					if err := sleepTrContextErr(ctx, index); err != nil {
						return Null(), err
					}
				}
			}
			matched = true
			break
		}
		if !matched {
			output = append(output, current)
			outputRaw = append(outputRaw, inputRaw[index])
		}
	}
	return sleepStringValueFromUnits(output, outputRaw), nil
}

func sleepTrMatches(current uint16, element sleepTrElement, options int) bool {
	matched := false
	if element.special {
		character := rune(current)
		switch element.item {
		case '.':
			matched = true
		case 'd':
			matched = javaRegexRuneInRanges(character, javaRegexPropertyRanges["DIGIT"])
		case 'D':
			matched = !javaRegexRuneInRanges(character, javaRegexPropertyRanges["DIGIT"])
		case 's':
			matched = sleepJavaIsWhitespace(character)
		case 'S':
			matched = !sleepJavaIsWhitespace(character)
		case 'w':
			matched = javaRegexRuneInRanges(character, javaRegexPropertyRanges["LETTER"])
		case 'W':
			matched = !javaRegexRuneInRanges(character, javaRegexPropertyRanges["LETTER"])
		}
	} else {
		matched = element.item == current
	}
	if options&sleepTrComplement != 0 {
		matched = !matched
	}
	return matched
}

func sleepTrContextErr(ctx context.Context, index int) error {
	if ctx == nil || index&0xff != 0 {
		return nil
	}
	return ctx.Err()
}

func sleepJavaIsWhitespace(value rune) bool {
	return javaRegexRuneInRanges(value, javaRegexPropertyRanges["JAVA_WHITESPACE"])
}
