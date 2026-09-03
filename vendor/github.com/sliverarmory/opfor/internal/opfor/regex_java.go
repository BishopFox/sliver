package opfor

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/sliverarmory/opfor/internal/regexp2"
)

// sleepRegexMatchTimeout bounds Java-compatible backtracking. java.util.regex
// itself has no timeout, but leaving attacker-controlled Aggressor patterns
// unbounded would let one library call wedge the embedding Go process.
const sleepRegexMatchTimeout = 2 * time.Second

const sleepRegexCacheCapacity = 128

type sleepRegexCacheKey struct {
	pattern string
	whole   bool
}

type sleepRegexCacheEntry struct {
	key        sleepRegexCacheKey
	expression *sleepRegex
}

// sleepRegexCache bounds translated/compiled Java-compatible patterns per
// Runtime. Compiled regexp2 programs are immutable and their runner pool is
// concurrency-safe, so entries may be shared by simultaneous executions.
type sleepRegexCache struct {
	mu      sync.Mutex
	entries map[sleepRegexCacheKey]*list.Element
	recent  list.List
}

func newSleepRegexCache() *sleepRegexCache {
	return &sleepRegexCache{entries: make(map[sleepRegexCacheKey]*list.Element, sleepRegexCacheCapacity)}
}

func (cache *sleepRegexCache) get(key sleepRegexCacheKey) (*sleepRegex, bool) {
	if cache == nil {
		return nil, false
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	element := cache.entries[key]
	if element == nil {
		return nil, false
	}
	cache.recent.MoveToFront(element)
	entry := element.Value.(sleepRegexCacheEntry)
	return entry.expression, true
}

func (cache *sleepRegexCache) put(key sleepRegexCacheKey, expression *sleepRegex) *sleepRegex {
	if cache == nil || expression == nil {
		return expression
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if element := cache.entries[key]; element != nil {
		cache.recent.MoveToFront(element)
		return element.Value.(sleepRegexCacheEntry).expression
	}
	element := cache.recent.PushFront(sleepRegexCacheEntry{key: key, expression: expression})
	cache.entries[key] = element
	if cache.recent.Len() > sleepRegexCacheCapacity {
		oldest := cache.recent.Back()
		entry := oldest.Value.(sleepRegexCacheEntry)
		delete(cache.entries, entry.key)
		cache.recent.Remove(oldest)
	}
	return expression
}

func (cache *sleepRegexCache) clear() {
	if cache == nil {
		return
	}
	cache.mu.Lock()
	clear(cache.entries)
	cache.recent.Init()
	cache.mu.Unlock()
}

func (cache *sleepRegexCache) len() int {
	if cache == nil {
		return 0
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return cache.recent.Len()
}

// sleepRegex is the narrow adapter used by RegexBridge-compatible operations.
// regexp2 supplies the backtracking constructs that Go's standard RE2 engine
// deliberately omits. The adapter keeps byte offsets at its boundary so the
// rest of the runtime can continue slicing Go strings without corrupting UTF-8.
type sleepRegex struct {
	expression *regexp2.Regexp
	names      map[string]int
	groups     int
	whole      bool
}

func compileSleepRegex(pattern string, whole bool) (*sleepRegex, error) {
	translated, names, groups, err := translateSleepRegexPattern(pattern)
	if err != nil {
		return nil, fmt.Errorf("opfor: invalid regular expression %q: %w", pattern, err)
	}
	if whole {
		translated = `\A(?:` + translated + `)\z`
		if err := validateJavaRegexTranslatedPatternSize(translated); err != nil {
			return nil, fmt.Errorf("opfor: invalid regular expression %q: %w", pattern, err)
		}
	}

	// RE2 mode only adjusts regexp2's predefined classes and a few parsing
	// details; it does not disable lookaround, backreferences, or atomic groups.
	// Its ASCII \d/\s/\w behavior is the closest regexp2 option to an unflagged
	// java.util.regex.Pattern.
	expression, err := regexp2.Compile(translated, regexp2.RE2)
	if err != nil {
		return nil, fmt.Errorf("opfor: invalid regular expression %q: %w", pattern, err)
	}
	expression.MatchTimeout = sleepRegexMatchTimeout
	return &sleepRegex{expression: expression, names: names, groups: groups, whole: whole}, nil
}

// compileSleepRegexBridge models RegexBridge.getPattern. Pattern.compile
// raises PatternSyntaxException (an IllegalArgumentException), so Sleep's
// Block reports the exception message as a warning and aborts only the active
// block. OPFOR's translation-size guard is an embedding safety boundary, not
// a Java syntax exception, and deliberately remains a fatal runtime error.
func compileSleepRegexBridge(pattern string, whole bool) (*sleepRegex, error) {
	expression, err := compileSleepRegex(pattern, whole)
	if err == nil {
		return expression, nil
	}
	if sleepRegexCompileFailureIsFatal(err) {
		return nil, err
	}
	return nil, sleepBridgeIllegalArgument(sleepJavaPatternSyntaxMessage(pattern, err))
}

func (runtime *Runtime) compileSleepRegexBridge(pattern string, whole bool) (*sleepRegex, error) {
	expression, err := runtime.compileSleepRegexCached(pattern, whole)
	if err == nil {
		return expression, nil
	}
	if sleepRegexCompileFailureIsFatal(err) {
		return nil, err
	}
	return nil, sleepBridgeIllegalArgument(sleepJavaPatternSyntaxMessage(pattern, err))
}

func (runtime *Runtime) compileSleepRegexCached(pattern string, whole bool) (*sleepRegex, error) {
	if runtime == nil || runtime.regexCache == nil {
		return compileSleepRegex(pattern, whole)
	}
	key := sleepRegexCacheKey{pattern: pattern, whole: whole}
	if expression, ok := runtime.regexCache.get(key); ok {
		return expression, nil
	}
	expression, err := compileSleepRegex(pattern, whole)
	if err != nil {
		return nil, err
	}
	return runtime.regexCache.put(key, expression), nil
}

func sleepRegexCompileFailureIsFatal(err error) bool {
	if err == nil {
		return false
	}
	var guard *javaRegexTranslationGuardError
	return errors.As(err, &guard) || strings.Contains(err.Error(), "regexp/syntax: internal error")
}

func sleepJavaPatternSyntaxMessage(pattern string, err error) string {
	var syntaxError *javaRegexPatternSyntaxError
	if errors.As(err, &syntaxError) {
		return formatSleepJavaPatternSyntaxMessage(syntaxError.description, pattern, syntaxError.index)
	}
	cause := err
	for cause != nil {
		next := errors.Unwrap(cause)
		if next == nil {
			break
		}
		cause = next
	}
	detail := cause.Error()
	if description, index, ok := javaRegexEngineDiagnostic(pattern, detail); ok {
		return formatSleepJavaPatternSyntaxMessage(description, pattern, index)
	}
	length := javaRegexPatternCodePointLength(pattern)
	switch {
	case strings.Contains(detail, "unterminated character class"):
		return formatSleepJavaPatternSyntaxMessage("Unclosed character class", pattern, length-1)
	case strings.Contains(detail, "missing closing )"), strings.Contains(detail, "unterminated group"):
		return formatSleepJavaPatternSyntaxMessage("Unclosed group", pattern, length)
	default:
		// The adapter's parser can reject Java-only forms before regexp2 has a
		// Java diagnostic equivalent. Preserve warning recovery and include the
		// rejected pattern instead of leaking OPFOR's wrapper text.
		return formatSleepJavaPatternSyntaxMessage(detail, pattern, -1)
	}
}

// javaRegexPatternSyntaxError carries the two fields exposed by
// PatternSyntaxException. The index is intentionally a Pattern parser cursor
// index: OpenJDK parses supplementary pairs as one code point even though
// PatternSyntaxException renders the caret against UTF-16 code units.
type javaRegexPatternSyntaxError struct {
	description string
	index       int
}

func (err *javaRegexPatternSyntaxError) Error() string {
	if err == nil {
		return ""
	}
	return err.description
}

func newJavaRegexPatternSyntaxError(pattern, description string, byteIndex int) error {
	index := byteIndex
	if byteIndex >= 0 {
		if byteIndex > len(pattern) {
			byteIndex = len(pattern)
		}
		index = javaRegexPatternCodePointLength(pattern[:byteIndex])
	}
	return &javaRegexPatternSyntaxError{description: description, index: index}
}

func javaRegexPatternCodePointLength(pattern string) int {
	units := sleepStringUnits(sleepStringValueFromCanonical(pattern))
	length := 0
	for index := 0; index < len(units); index++ {
		if units[index] >= 0xd800 && units[index] <= 0xdbff && index+1 < len(units) &&
			units[index+1] >= 0xdc00 && units[index+1] <= 0xdfff {
			index++
		}
		length++
	}
	return length
}

func formatSleepJavaPatternSyntaxMessage(description, pattern string, index int) string {
	var message strings.Builder
	message.WriteString(description)
	if index >= 0 {
		fmt.Fprintf(&message, " near index %d", index)
	}
	message.WriteByte('\n')
	message.WriteString(pattern)
	units := sleepStringUnits(sleepStringValueFromCanonical(pattern))
	if index >= 0 && index < len(units) {
		message.WriteByte('\n')
		for _, unit := range units[:index] {
			if unit == '\t' {
				message.WriteByte('\t')
			} else {
				message.WriteByte(' ')
			}
		}
		message.WriteByte('^')
	}
	return message.String()
}

// sleepRegexBridgeReplacementError applies Block's exception dispatch to the
// OpenJDK-style errors produced by appendPortableJavaReplacement. Host errors
// such as cancellation, instruction limits, and allocation guards pass
// through unchanged.
func sleepRegexBridgeReplacementError(err error) error {
	if err == nil {
		return nil
	}
	const illegalArgument = "java.lang.IllegalArgumentException: "
	const invalidIndex = "java.lang.IndexOutOfBoundsException: "
	message := err.Error()
	switch {
	case strings.HasPrefix(message, illegalArgument):
		return sleepBridgeIllegalArgument(strings.TrimPrefix(message, illegalArgument))
	case strings.HasPrefix(message, invalidIndex):
		return sleepBridgeInvalidIndex(strings.TrimPrefix(message, invalidIndex))
	default:
		return err
	}
}

func (r *sleepRegex) NumSubexp() int {
	if r == nil {
		return 0
	}
	return r.groups
}

func (r *sleepRegex) SubexpIndex(name string) int {
	if r == nil {
		return -1
	}
	group, ok := r.names[name]
	if !ok {
		return -1
	}
	return group
}

func (r *sleepRegex) FindStringSubmatchIndex(input string) ([]int, error) {
	return r.FindStringSubmatchIndexContext(context.Background(), input)
}

func (r *sleepRegex) FindStringSubmatchIndexContext(ctx context.Context, input string) ([]int, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return nil, err
	}
	text := newSleepRegexText(input)
	// ismatch compiles an absolute whole-input expression. With no capture
	// groups, RegexBridge only observes success: Sleep's matcher group zero is
	// not included in matched(), so avoid materializing a regexp2 Match and its
	// capture arrays on this common predicate path.
	if r.whole && r.groups == 0 {
		matched, err := r.expression.MatchRunesContext(ctx, text.runes)
		if err != nil || !matched {
			return nil, err
		}
		if err := executionContextError(ctx); err != nil {
			return nil, err
		}
		if err := consumeInstruction(ctx); err != nil {
			return nil, err
		}
		return []int{0, len(input)}, nil
	}
	match, err := r.expression.FindRunesMatchContext(ctx, text.runes)
	if err != nil || match == nil {
		return nil, err
	}
	if err := executionContextError(ctx); err != nil {
		return nil, err
	}
	if err := consumeInstruction(ctx); err != nil {
		return nil, err
	}
	return r.byteIndices(text, match), nil
}

func (r *sleepRegex) FindStringSubmatchIndexAt(input string, byteStart int) ([]int, error) {
	return r.FindStringSubmatchIndexAtContext(context.Background(), input, byteStart)
}

func (r *sleepRegex) FindStringSubmatchIndexAtContext(ctx context.Context, input string, byteStart int) ([]int, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return nil, err
	}
	text := newSleepRegexText(input)
	runeStart, ok := text.runeIndexAtByteOffset(byteStart)
	if !ok {
		return nil, fmt.Errorf("startAt must align to a canonical string element")
	}
	match, err := r.expression.FindRunesMatchStartingAtContext(ctx, text.runes, runeStart)
	if err != nil || match == nil {
		return nil, err
	}
	if err := executionContextError(ctx); err != nil {
		return nil, err
	}
	if err := consumeInstruction(ctx); err != nil {
		return nil, err
	}
	return r.byteIndices(text, match), nil
}

func (r *sleepRegex) FindAllStringSubmatchIndex(input string, limit int) ([][]int, error) {
	return r.FindAllStringSubmatchIndexContext(context.Background(), input, limit)
}

func (r *sleepRegex) FindAllStringSubmatchIndexContext(ctx context.Context, input string, limit int) ([][]int, error) {
	if limit == 0 {
		return nil, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return nil, err
	}
	text := newSleepRegexText(input)
	match, err := r.expression.FindRunesMatchContext(ctx, text.runes)
	if err != nil {
		return nil, err
	}
	matches := make([][]int, 0)
	for match != nil && (limit < 0 || len(matches) < limit) {
		if err := executionContextError(ctx); err != nil {
			return nil, err
		}
		if err := consumeInstruction(ctx); err != nil {
			return nil, err
		}
		matches = append(matches, r.byteIndices(text, match))
		match, err = r.expression.FindNextMatchContext(ctx, match)
		if err != nil {
			return nil, err
		}
	}
	return matches, nil
}

// FindAllStringSubmatchUTF16IndexContext returns Java String offsets and also
// models Matcher.find's one-UTF-16-code-unit advance after an empty match.
// regexp2 deliberately works in Unicode code points, so the ordinary byte
// index API cannot represent the position between a supplementary character's
// surrogate halves. Java can visit that position after a zero-width match.
func (r *sleepRegex) FindAllStringSubmatchUTF16IndexContext(ctx context.Context, input string, limit int) ([][]int, error) {
	if limit == 0 {
		return nil, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return nil, err
	}
	text := newSleepRegexText(input)
	normal, err := r.expression.FindRunesMatchContext(ctx, text.runes)
	if err != nil {
		return nil, err
	}
	boundaries := text.supplementaryBoundaries()
	boundaryIndex := 0
	nextUnit := 0
	matches := make([][]int, 0)

	for limit < 0 || len(matches) < limit {
		if err := executionContextError(ctx); err != nil {
			return nil, err
		}
		for boundaryIndex < len(boundaries) && boundaries[boundaryIndex].unitOffset < nextUnit {
			boundaryIndex++
		}

		normalStart := int(^uint(0) >> 1)
		var normalIndices []int
		if normal != nil {
			normalIndices = r.utf16Indices(text.utf16Offsets, normal)
			normalStart = normalIndices[0]
		}

		var internal []int
		for boundaryIndex < len(boundaries) && boundaries[boundaryIndex].unitOffset < normalStart {
			if err := consumeInstruction(ctx); err != nil {
				return nil, err
			}
			candidate := boundaries[boundaryIndex]
			boundaryIndex++
			internal, err = r.zeroWidthSupplementaryBoundaryMatch(ctx, text, candidate)
			if err != nil {
				return nil, err
			}
			if internal != nil {
				break
			}
		}
		if internal != nil {
			matches = append(matches, internal)
			nextUnit = internal[0] + 1
			continue
		}
		if normal == nil {
			break
		}
		if err := consumeInstruction(ctx); err != nil {
			return nil, err
		}
		matches = append(matches, normalIndices)
		if normalIndices[0] == normalIndices[1] {
			nextUnit = normalIndices[0] + 1
		} else {
			nextUnit = normalIndices[1]
		}
		normal, err = r.expression.FindNextMatchContext(ctx, normal)
		if err != nil {
			return nil, err
		}
	}
	return matches, nil
}

type sleepRegexSupplementaryBoundary struct {
	runeIndex  int
	unitOffset int
}

func (r *sleepRegex) zeroWidthSupplementaryBoundaryMatch(
	ctx context.Context,
	text sleepRegexText,
	boundary sleepRegexSupplementaryBoundary,
) ([]int, error) {
	character := text.runes[boundary.runeIndex]
	high, low := utf16.EncodeRune(character)
	probe := make([]rune, 0, len(text.runes)+1)
	probe = append(probe, text.runes[:boundary.runeIndex]...)
	probe = append(probe, high, low)
	probe = append(probe, text.runes[boundary.runeIndex+1:]...)
	start := boundary.runeIndex + 1
	match, err := r.expression.FindRunesMatchStartingAtWithReverseFloorContext(ctx, probe, start, start)
	if err != nil || match == nil || match.Index != start || match.Length != 0 {
		return nil, err
	}
	offsets := make([]int, len(probe)+1)
	units := 0
	for index, current := range probe {
		offsets[index] = units
		units++
		if current > 0xffff {
			units++
		}
	}
	offsets[len(probe)] = units
	return r.utf16Indices(offsets, match), nil
}

func (r *sleepRegex) utf16Indices(offsets []int, match *regexp2.Match) []int {
	indices := make([]int, (r.groups+1)*2)
	for index := range indices {
		indices[index] = -1
	}
	for groupNumber := 0; groupNumber <= r.groups; groupNumber++ {
		group := match.GroupByNumber(groupNumber)
		if group == nil || len(group.Captures) == 0 {
			continue
		}
		start := group.Index
		end := group.Index + group.Length
		if start < 0 || end < start || end >= len(offsets) {
			// ASCII text needs no offset table: rune, byte, and UTF-16 indices
			// are identical. newSleepRegexText deliberately leaves it nil.
			if offsets != nil || start < 0 || end < start {
				continue
			}
			indices[groupNumber*2] = start
			indices[groupNumber*2+1] = end
			continue
		}
		indices[groupNumber*2] = offsets[start]
		indices[groupNumber*2+1] = offsets[end]
	}
	return indices
}

func (r *sleepRegex) byteIndices(text sleepRegexText, match *regexp2.Match) []int {
	indices := make([]int, (r.groups+1)*2)
	for index := range indices {
		indices[index] = -1
	}
	for groupNumber := 0; groupNumber <= r.groups; groupNumber++ {
		group := match.GroupByNumber(groupNumber)
		if group == nil || len(group.Captures) == 0 {
			continue
		}
		start := group.Index
		end := group.Index + group.Length
		if text.ascii {
			indices[groupNumber*2] = start
			indices[groupNumber*2+1] = end
			continue
		}
		if start < 0 || end < start || end >= len(text.byteOffsets) {
			continue
		}
		indices[groupNumber*2] = text.byteOffsets[start]
		indices[groupNumber*2+1] = text.byteOffsets[end]
	}
	return indices
}

// sleepRegexText is regexp2's rune-oriented view of a canonical Sleep string.
// Unpaired UTF-16 surrogates are represented by reversible WTF-8 at the Go
// boundary; treating that spelling with range would produce three RuneErrors.
// Keeping exact byte offsets lets callers continue slicing the canonical input
// without exposing regexp2's rune indices.
type sleepRegexText struct {
	runes        []rune
	byteOffsets  []int
	utf16Offsets []int
	ascii        bool
}

func newSleepRegexText(value string) sleepRegexText {
	if sleepRegexASCII(value) {
		runes := make([]rune, len(value))
		for index := range value {
			runes[index] = rune(value[index])
		}
		return sleepRegexText{runes: runes, ascii: true}
	}
	runeCount := utf8.RuneCountInString(value)
	offsets := make([]int, (runeCount+1)*2)
	text := sleepRegexText{
		runes:        make([]rune, 0, runeCount),
		byteOffsets:  offsets[: 0 : runeCount+1],
		utf16Offsets: offsets[runeCount+1 : runeCount+1 : (runeCount+1)*2],
	}
	units := 0
	for index := 0; index < len(value); {
		text.byteOffsets = append(text.byteOffsets, index)
		text.utf16Offsets = append(text.utf16Offsets, units)
		if unit, ok := sleepWTF8SurrogateAt(value, index); ok {
			text.runes = append(text.runes, rune(unit))
			units++
			index += 3
			continue
		}
		character, size := utf8.DecodeRuneInString(value[index:])
		text.runes = append(text.runes, character)
		units++
		if character > 0xffff {
			units++
		}
		index += size
	}
	text.byteOffsets = append(text.byteOffsets, len(value))
	text.utf16Offsets = append(text.utf16Offsets, units)
	return text
}

func sleepRegexASCII(value string) bool {
	for index := range value {
		if value[index] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

func (t sleepRegexText) supplementaryBoundaries() []sleepRegexSupplementaryBoundary {
	boundaries := make([]sleepRegexSupplementaryBoundary, 0)
	for index, character := range t.runes {
		if character > 0xffff {
			boundaries = append(boundaries, sleepRegexSupplementaryBoundary{
				runeIndex:  index,
				unitOffset: t.utf16Offsets[index] + 1,
			})
		}
	}
	return boundaries
}

func (t sleepRegexText) runeIndexAtByteOffset(byteOffset int) (int, bool) {
	if t.ascii {
		return byteOffset, byteOffset >= 0 && byteOffset <= len(t.runes)
	}
	left, right := 0, len(t.byteOffsets)
	for left < right {
		middle := left + (right-left)/2
		if t.byteOffsets[middle] < byteOffset {
			left = middle + 1
		} else {
			right = middle
		}
	}
	return left, left < len(t.byteOffsets) && t.byteOffsets[left] == byteOffset
}

func sleepRegexUTF16Length(value string) int {
	length := 0
	for _, character := range newSleepRegexText(value).runes {
		length++
		if character > 0xffff {
			length++
		}
	}
	return length
}

func sleepRegexUTF16ToByteIndex(value string, target int) (int, bool) {
	if target < 0 {
		return 0, false
	}
	text := newSleepRegexText(value)
	if text.ascii {
		return target, target <= len(value)
	}
	units := 0
	for index, character := range text.runes {
		if units == target {
			return text.byteOffsets[index], true
		}
		width := 1
		if character > 0xffff {
			width = 2
		}
		if units+width > target {
			// Java exposes UTF-16 offsets, but regexp2 cannot begin between the
			// two halves of a supplementary code point.
			return 0, false
		}
		units += width
	}
	if units == target {
		return len(value), true
	}
	return 0, false
}

// Split implements java.util.regex.Pattern.split. In particular, a
// zero-width match at the beginning never creates a leading empty element,
// while a positive-width match there does.
func (r *sleepRegex) Split(input string, limit int) ([]string, error) {
	return r.SplitContext(context.Background(), input, limit)
}

func (r *sleepRegex) SplitContext(ctx context.Context, input string, limit int) ([]string, error) {
	return r.splitContext(ctx, input, limit, nil)
}

func (r *sleepRegex) splitContext(ctx context.Context, input string, limit int, beforeAppend func() error) ([]string, error) {
	if limit == 1 {
		if beforeAppend != nil {
			if err := beforeAppend(); err != nil {
				return nil, err
			}
		}
		return []string{input}, nil
	}
	reservePieces := func(pieces []string) error {
		if beforeAppend == nil {
			return nil
		}
		for range pieces {
			if err := beforeAppend(); err != nil {
				return err
			}
		}
		return nil
	}
	matches, err := r.FindAllStringSubmatchUTF16IndexContext(ctx, input, -1)
	if err != nil {
		return nil, err
	}
	inputValue := sleepStringValueFromCanonical(input)
	inputLength := sleepStringLength(inputValue)
	pieces := make([]string, 0, len(matches)+1)
	last := 0
	used := false
	for _, match := range matches {
		if limit > 0 && len(pieces) == limit-1 {
			break
		}
		start, end := match[0], match[1]
		if !used && start == 0 && end == 0 {
			continue
		}
		pieces = append(pieces, sleepCanonicalString(sleepStringValueSlice(inputValue, last, start)))
		last = end
		used = true
	}
	if !used {
		pieces = append(pieces, input)
		if err := reservePieces(pieces); err != nil {
			return nil, err
		}
		return pieces, nil
	}
	pieces = append(pieces, sleepCanonicalString(sleepStringValueSlice(inputValue, last, inputLength)))
	if limit == 0 {
		for len(pieces) > 0 && pieces[len(pieces)-1] == "" {
			pieces = pieces[:len(pieces)-1]
		}
	}
	if err := reservePieces(pieces); err != nil {
		return nil, err
	}
	return pieces, nil
}

// translateSleepRegex is retained as a small test seam. The returned pattern
// uses regexp2/.NET syntax after Java-specific quoting, class algebra, named
// group numbering, and possessive quantifiers have been normalized.
func translateSleepRegex(pattern string) (string, error) {
	translated, _, _, err := translateSleepRegexPattern(pattern)
	return translated, err
}

func translateSleepRegexPattern(pattern string) (string, map[string]int, int, error) {
	if err := validateJavaRegexPattern(pattern); err != nil {
		return "", nil, 0, err
	}
	quoted, err := translateJavaRegexQuotes(pattern)
	if err != nil {
		return "", nil, 0, err
	}
	modes, properties, err := translateJavaRegexModes(quoted)
	if err != nil {
		return "", nil, 0, err
	}
	classes, err := translateJavaRegexClasses(modes)
	if err != nil {
		return "", nil, 0, err
	}
	classes, err = properties.expand(classes)
	if err != nil {
		return "", nil, 0, err
	}
	named, names, groups, err := translateJavaNamedGroups(classes)
	if err != nil {
		return "", nil, 0, err
	}
	possessive, err := translateJavaPossessiveQuantifiers(named)
	if err != nil {
		return "", nil, 0, err
	}
	if err := validateJavaRegexTranslatedPatternSize(possessive); err != nil {
		return "", nil, 0, err
	}
	return possessive, names, groups, nil
}

func translateJavaRegexQuotes(pattern string) (string, error) {
	var result strings.Builder
	for index := 0; index < len(pattern); {
		if index+1 < len(pattern) && pattern[index] == '\\' && pattern[index+1] == 'Q' {
			end := strings.Index(pattern[index+2:], `\E`)
			if end < 0 {
				result.WriteString(escapeSleepRegexQuoted(pattern[index+2:]))
				break
			}
			end += index + 2
			result.WriteString(escapeSleepRegexQuoted(pattern[index+2 : end]))
			index = end + 2
			continue
		}
		if index+1 < len(pattern) && pattern[index] == '\\' {
			if unit, ok := sleepWTF8SurrogateAt(pattern, index+1); ok {
				writeSleepRegexSurrogateEscape(&result, unit)
				index += 4
				continue
			}
			result.WriteString(pattern[index : index+2])
			index += 2
			continue
		}
		if unit, ok := sleepWTF8SurrogateAt(pattern, index); ok {
			writeSleepRegexSurrogateEscape(&result, unit)
			index += 3
			continue
		}
		result.WriteByte(pattern[index])
		index++
	}
	return result.String(), nil
}

func escapeSleepRegexQuoted(value string) string {
	var result strings.Builder
	plainStart := 0
	for index := 0; index < len(value); {
		unit, ok := sleepWTF8SurrogateAt(value, index)
		if !ok {
			_, size := utf8.DecodeRuneInString(value[index:])
			index += size
			continue
		}
		result.WriteString(regexp2.Escape(value[plainStart:index]))
		writeSleepRegexSurrogateEscape(&result, unit)
		index += 3
		plainStart = index
	}
	result.WriteString(regexp2.Escape(value[plainStart:]))
	return result.String()
}

func writeSleepRegexSurrogateEscape(result *strings.Builder, unit uint16) {
	fmt.Fprintf(result, `\u%04X`, unit)
}

func translateJavaRegexClasses(pattern string) (string, error) {
	var result strings.Builder
	for index := 0; index < len(pattern); {
		if pattern[index] == '\\' && index+1 < len(pattern) {
			result.WriteString(pattern[index : index+2])
			index += 2
			continue
		}
		if pattern[index] != '[' {
			result.WriteByte(pattern[index])
			index++
			continue
		}
		end, err := javaRegexClassEnd(pattern, index)
		if err != nil {
			return "", err
		}
		fragment, err := renderJavaClass(pattern[index+1 : end])
		if err != nil {
			return "", err
		}
		result.WriteString(fragment)
		index = end + 1
	}
	return result.String(), nil
}

func javaRegexClassEnd(pattern string, start int) (int, error) {
	type classFrame struct {
		firstElement    bool
		leadingNegation bool
	}
	frames := []classFrame{{firstElement: true}}
	for index := start + 1; index < len(pattern); index++ {
		if pattern[index] == '\\' {
			frames[len(frames)-1].firstElement = false
			index++
			continue
		}
		switch pattern[index] {
		case '[':
			frames[len(frames)-1].firstElement = false
			frames = append(frames, classFrame{firstElement: true})
		case ']':
			if frames[len(frames)-1].firstElement {
				// Java treats ] as a literal when it is the first class
				// element (also after a leading negation marker).
				frames[len(frames)-1].firstElement = false
				continue
			}
			frames = frames[:len(frames)-1]
			if len(frames) == 0 {
				return index, nil
			}
		case '^':
			frame := &frames[len(frames)-1]
			if frame.firstElement && !frame.leadingNegation {
				frame.leadingNegation = true
			} else {
				frame.firstElement = false
			}
		default:
			frames[len(frames)-1].firstElement = false
		}
	}
	return 0, fmt.Errorf("unterminated character class")
}

// renderJavaClass turns Java's implicit class union and && intersection into
// one-character regexp fragments. Lookaheads express intersections generally,
// including positive intersections that .NET-style subtraction cannot encode.
func renderJavaClass(content string) (string, error) {
	negated := strings.HasPrefix(content, "^")
	if negated {
		content = content[1:]
		if strings.HasPrefix(content, "^") {
			content = `\^` + content[1:]
		}
	}
	parts, err := splitJavaClassIntersection(content)
	if err != nil {
		return "", err
	}
	operands := make([]string, 0, len(parts))
	for index, part := range parts {
		if index > 0 && strings.HasPrefix(part, "^") {
			part = `\^` + part[1:]
		}
		operand, operandErr := renderJavaClassUnion(part)
		if operandErr != nil {
			return "", operandErr
		}
		operands = append(operands, operand)
	}
	var positive string
	if len(operands) == 1 {
		positive = operands[0]
	} else {
		var result strings.Builder
		result.WriteString("(?:")
		for _, operand := range operands {
			result.WriteString("(?=")
			result.WriteString(operand)
			result.WriteByte(')')
		}
		result.WriteString(`[\s\S])`)
		positive = result.String()
	}
	if !negated {
		return positive, nil
	}
	return `(?:(?!` + positive + `)[\s\S])`, nil
}

func splitJavaClassIntersection(content string) ([]string, error) {
	parts := make([]string, 0, 2)
	start := 0
	depth := 0
	leadingEmpty := false
	for index := 0; index < len(content); index++ {
		if content[index] == '\\' {
			index++
			continue
		}
		switch content[index] {
		case '[':
			depth++
		case ']':
			depth--
			if depth < 0 {
				return nil, fmt.Errorf("malformed nested character class")
			}
		case '&':
			if depth == 0 && index+1 < len(content) && content[index+1] == '&' {
				if index > start {
					parts = append(parts, content[start:index])
				} else if len(parts) == 0 {
					if leadingEmpty {
						return nil, fmt.Errorf("empty character-class intersection")
					}
					leadingEmpty = true
				}
				index++
				start = index + 1
			}
		}
	}
	if depth != 0 {
		return nil, fmt.Errorf("unterminated nested character class")
	}
	if start < len(content) {
		parts = append(parts, content[start:])
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty character-class intersection")
	}
	return parts, nil
}

func renderJavaClassUnion(content string) (string, error) {
	components := make([]string, 0, 2)
	plainStart := 0
	flushPlain := func(end int) {
		if end > plainStart {
			components = append(components, "["+content[plainStart:end]+"]")
		}
	}
	for index := 0; index < len(content); {
		if content[index] == '\\' {
			index += 2
			continue
		}
		if content[index] != '[' {
			index++
			continue
		}
		flushPlain(index)
		end, err := javaRegexClassEnd(content, index)
		if err != nil {
			return "", err
		}
		fragment, err := renderJavaClass(content[index+1 : end])
		if err != nil {
			return "", err
		}
		components = append(components, fragment)
		index = end + 1
		plainStart = index
	}
	flushPlain(len(content))
	if len(components) == 0 {
		return "", fmt.Errorf("empty character class")
	}
	if len(components) == 1 {
		return components[0], nil
	}
	return "(?:" + strings.Join(components, "|") + ")", nil
}

func translateJavaNamedGroups(pattern string) (string, map[string]int, int, error) {
	names := make(map[string]int)
	groups := 0
	for index := 0; index < len(pattern); {
		if pattern[index] == '\\' {
			index += min(2, len(pattern)-index)
			continue
		}
		if pattern[index] == '[' {
			end, err := javaRegexClassEnd(pattern, index)
			if err != nil {
				return "", nil, 0, err
			}
			index = end + 1
			continue
		}
		if pattern[index] != '(' {
			index++
			continue
		}
		if index+2 >= len(pattern) || pattern[index+1] != '?' {
			groups++
			index++
			continue
		}
		if pattern[index+2] != '<' || index+3 >= len(pattern) || pattern[index+3] == '=' || pattern[index+3] == '!' {
			index++
			continue
		}
		end := strings.IndexByte(pattern[index+3:], '>')
		if end < 0 {
			return "", nil, 0, fmt.Errorf("unterminated named capture")
		}
		end += index + 3
		name := pattern[index+3 : end]
		if !validJavaRegexGroupName(name) {
			return "", nil, 0, fmt.Errorf("invalid named capture %q", name)
		}
		if _, exists := names[name]; exists {
			return "", nil, 0, fmt.Errorf("duplicate named capture %q", name)
		}
		groups++
		names[name] = groups
		index = end + 1
	}

	var result strings.Builder
	for index := 0; index < len(pattern); {
		if index+2 < len(pattern) && pattern[index] == '\\' && pattern[index+1] == 'k' && pattern[index+2] == '<' {
			end := strings.IndexByte(pattern[index+3:], '>')
			if end < 0 {
				return "", nil, 0, fmt.Errorf("unterminated named backreference")
			}
			end += index + 3
			name := pattern[index+3 : end]
			group, ok := names[name]
			if !ok {
				return "", nil, 0, fmt.Errorf("unknown named backreference %q", name)
			}
			fmt.Fprintf(&result, `\k<%d>`, group)
			index = end + 1
			continue
		}
		if pattern[index] == '\\' && index+1 < len(pattern) {
			result.WriteString(pattern[index : index+2])
			index += 2
			continue
		}
		if index+3 < len(pattern) && pattern[index:index+3] == "(?<" && pattern[index+3] != '=' && pattern[index+3] != '!' {
			end := strings.IndexByte(pattern[index+3:], '>')
			if end < 0 {
				return "", nil, 0, fmt.Errorf("unterminated named capture")
			}
			end += index + 3
			result.WriteByte('(')
			index = end + 1
			continue
		}
		result.WriteByte(pattern[index])
		index++
	}
	return result.String(), names, groups, nil
}

func validJavaRegexGroupName(name string) bool {
	if name == "" || !asciiRegexLetter(name[0]) {
		return false
	}
	for index := 1; index < len(name); index++ {
		if !asciiRegexLetter(name[index]) && (name[index] < '0' || name[index] > '9') {
			return false
		}
	}
	return true
}

func asciiRegexLetter(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}

func translateJavaPossessiveQuantifiers(pattern string) (string, error) {
	var result []byte
	groupStarts := make([]int, 0, 4)
	lastAtom := -1
	for index := 0; index < len(pattern); {
		start := len(result)
		switch pattern[index] {
		case '\\':
			if index+1 >= len(pattern) {
				return "", fmt.Errorf("escape at end of pattern")
			}
			result = append(result, pattern[index:index+2]...)
			index += 2
			lastAtom = start
		case '[':
			end, err := javaRegexClassEnd(pattern, index)
			if err != nil {
				return "", err
			}
			result = append(result, pattern[index:end+1]...)
			index = end + 1
			lastAtom = start
		case '(':
			result = append(result, '(')
			groupStarts = append(groupStarts, start)
			index++
			lastAtom = -1
		case ')':
			result = append(result, ')')
			index++
			if len(groupStarts) == 0 {
				return "", fmt.Errorf("unmatched closing parenthesis")
			}
			lastAtom = groupStarts[len(groupStarts)-1]
			groupStarts = groupStarts[:len(groupStarts)-1]
		case '|':
			result = append(result, '|')
			index++
			lastAtom = -1
		case '*', '+', '?':
			quantifier := pattern[index : index+1]
			index++
			modifier := byte(0)
			if index < len(pattern) && (pattern[index] == '+' || pattern[index] == '?') {
				modifier = pattern[index]
				index++
			}
			if modifier == '+' && lastAtom >= 0 {
				result = wrapPossessiveAtom(result, lastAtom, quantifier)
			} else {
				result = append(result, quantifier...)
				if modifier != 0 {
					result = append(result, modifier)
				}
			}
		case '{':
			end, ok := sleepCountedQuantifier(pattern, index)
			if !ok {
				result = append(result, pattern[index])
				index++
				lastAtom = start
				continue
			}
			quantifier := pattern[index : end+1]
			index = end + 1
			modifier := byte(0)
			if index < len(pattern) && (pattern[index] == '+' || pattern[index] == '?') {
				modifier = pattern[index]
				index++
			}
			if lastAtom < 0 {
				// Pattern.atom returns an empty slice when a counted closure is
				// the first item in a sequence, so forms such as {2} are valid
				// zero-width patterns. regexp2 rejects a leading quantifier;
				// omitting it preserves Java's observable match behavior.
				continue
			}
			if modifier == '+' {
				result = wrapPossessiveAtom(result, lastAtom, quantifier)
			} else {
				result = append(result, quantifier...)
				if modifier != 0 {
					result = append(result, modifier)
				}
			}
		default:
			result = append(result, pattern[index])
			if !strings.ContainsRune("^$", rune(pattern[index])) && !(pattern[index] == '?' && len(groupStarts) > 0) {
				lastAtom = start
			}
			index++
		}
	}
	if len(groupStarts) != 0 {
		return "", fmt.Errorf("unterminated group")
	}
	return string(result), nil
}

func wrapPossessiveAtom(result []byte, atomStart int, quantifier string) []byte {
	atom := append([]byte(nil), result[atomStart:]...)
	result = result[:atomStart]
	result = append(result, "(?>"...)
	result = append(result, atom...)
	result = append(result, quantifier...)
	result = append(result, ')')
	return result
}
