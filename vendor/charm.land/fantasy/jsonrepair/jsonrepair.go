// Package jsonrepair provides utilities to repair malformed JSON.
package jsonrepair

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
)

// Option is a function that configures the JSON repairer.
type Option func(*options)

type options struct {
	ensureASCII   *bool
	skipJSONLoads bool
	streamStable  bool
	strict        bool
}

// LogEntry represents a log entry with context and text.
type LogEntry struct {
	Context string `json:"context"`
	Text    string `json:"text"`
}

type numberValue struct {
	raw string
}

type objectEntry struct {
	key   string
	value any
}

type orderedObject struct {
	entries []objectEntry
	index   map[string]int
}

func newOrderedObject() *orderedObject {
	return &orderedObject{index: map[string]int{}}
}

func (o *orderedObject) set(key string, value any) {
	if idx, ok := o.index[key]; ok {
		o.entries[idx].value = value
		return
	}
	o.index[key] = len(o.entries)
	o.entries = append(o.entries, objectEntry{key: key, value: value})
}

func (o *orderedObject) get(key string) (any, bool) {
	idx, ok := o.index[key]
	if !ok {
		return nil, false
	}
	return o.entries[idx].value, true
}

func (o *orderedObject) lastKey() (string, bool) {
	if len(o.entries) == 0 {
		return "", false
	}
	return o.entries[len(o.entries)-1].key, true
}

func (o *orderedObject) hasKey(key string) bool {
	_, ok := o.index[key]
	return ok
}

func (o *orderedObject) merge(other *orderedObject) {
	for _, entry := range other.entries {
		o.set(entry.key, entry.value)
	}
}

type contextValue int

const (
	contextObjectKey contextValue = iota
	contextObjectValue
	contextArray
)

type jsonContext struct {
	context []contextValue
	current *contextValue
	empty   bool
}

func newJSONContext() *jsonContext {
	return &jsonContext{empty: true}
}

func (c *jsonContext) set(value contextValue) {
	c.context = append(c.context, value)
	c.current = &c.context[len(c.context)-1]
	c.empty = false
}

func (c *jsonContext) reset() {
	if len(c.context) > 0 {
		c.context = c.context[:len(c.context)-1]
	}
	if len(c.context) == 0 {
		c.current = nil
		c.empty = true
		return
	}
	c.current = &c.context[len(c.context)-1]
}

func (c *jsonContext) contains(value contextValue) bool {
	return slices.Contains(c.context, value)
}

type parser struct {
	jsonStr      []rune
	index        int
	context      *jsonContext
	logging      bool
	logger       []LogEntry
	streamStable bool
	strict       bool
	log          func(string)
}

func newParser(input string, logging bool, streamStable bool, strict bool) *parser {
	p := &parser{
		jsonStr:      []rune(input),
		context:      newJSONContext(),
		logging:      logging,
		streamStable: streamStable,
		strict:       strict,
	}
	if logging {
		p.log = p.addLog
	} else {
		p.log = func(string) {}
	}
	return p
}

func (p *parser) addLog(text string) {
	window := 10
	start := max(p.index-window, 0)
	end := min(p.index+window, len(p.jsonStr))
	context := string(p.jsonStr[start:end])
	p.logger = append(p.logger, LogEntry{Text: text, Context: context})
}

func (p *parser) parse() (any, []LogEntry, error) {
	jsonValue, err := p.parseJSON()
	if err != nil {
		return nil, nil, err
	}
	if p.index < len(p.jsonStr) {
		p.log("The parser returned early, checking if there's more json elements")
		values := []any{jsonValue}
		for p.index < len(p.jsonStr) {
			p.context.reset()
			j, parseErr := p.parseJSON()
			if parseErr != nil {
				return nil, nil, parseErr
			}
			if isTruthy(j) {
				if len(values) > 0 && isSameObject(values[len(values)-1], j) {
					values = values[:len(values)-1]
				} else if len(values) > 0 && !isTruthy(values[len(values)-1]) {
					values = values[:len(values)-1]
				}
				values = append(values, j)
			} else {
				if len(values) > 1 {
					_, ok := p.getCharAt(0)
					if !ok {
						break
					}
					if len(values) > 1 {
						values = values[:len(values)-1]
					}
					p.index = len(p.jsonStr)
					break
				}
				p.index++
			}
		}
		if len(values) == 1 {
			p.log("There were no more elements, returning the element without the array")
			jsonValue = values[0]
		} else if p.strict {
			p.log("Multiple top-level JSON elements found in strict mode, raising an error")
			return nil, nil, errors.New("multiple top-level JSON elements found in strict mode")
		} else {
			jsonValue = values
		}
	}
	return jsonValue, p.logger, nil
}

func (p *parser) parseJSON() (any, error) {
	for {
		char, ok := p.getCharAt(0)
		if !ok {
			return "", nil
		}
		if char == '{' {
			p.index++
			return p.parseObject()
		}
		if char == '[' {
			p.index++
			return p.parseArray()
		}
		if !p.context.empty && (isStringDelimiter(char) || unicode.IsLetter(char)) {
			return p.parseString()
		}
		if !p.context.empty && (unicode.IsDigit(char) || char == '-' || char == '.') {
			return p.parseNumber()
		}
		if p.context.empty && (unicode.IsDigit(char) || char == '-' || char == '.') {
			if onlyWhitespaceBefore(p) {
				return p.parseNumber()
			}
		}
		if char == '#' || char == '/' {
			return p.parseComment()
		}
		if !p.context.empty && (char == 't' || char == 'f' || char == 'n') {
			value := p.parseBooleanOrNull()
			if value != "" {
				return value, nil
			}
			return p.parseString()
		}
		if p.context.empty && (char == 't' || char == 'f' || char == 'n') {
			if onlyWhitespaceBefore(p) {
				value := p.parseBooleanOrNull()
				if value != "" {
					return value, nil
				}
			}
		}
		if p.context.empty && char == ':' {
			return "", nil
		}
		p.index++
	}
}

func (p *parser) getCharAt(offset int) (rune, bool) {
	idx := p.index + offset
	if idx < 0 || idx >= len(p.jsonStr) {
		return 0, false
	}
	return p.jsonStr[idx], true
}

func (p *parser) skipWhitespaces() {
	for {
		char, ok := p.getCharAt(0)
		if !ok || !unicode.IsSpace(char) {
			return
		}
		p.index++
	}
}

func (p *parser) scrollWhitespaces(idx int) int {
	for {
		char, ok := p.getCharAt(idx)
		if !ok || !unicode.IsSpace(char) {
			return idx
		}
		idx++
	}
}

func (p *parser) skipToCharacter(character rune, idx int) int {
	targets := map[rune]struct{}{character: {}}
	return p.skipToCharacters(targets, idx)
}

func (p *parser) skipToCharacters(targets map[rune]struct{}, idx int) int {
	i := p.index + idx
	backslashes := 0
	for i < len(p.jsonStr) {
		ch := p.jsonStr[i]
		if ch == '\\' {
			backslashes++
			i++
			continue
		}
		if _, ok := targets[ch]; ok && backslashes%2 == 0 {
			return i - p.index
		}
		backslashes = 0
		i++
	}
	return len(p.jsonStr) - p.index
}

func (p *parser) parseArray() (any, error) {
	arr := []any{}
	p.context.set(contextArray)
	char, ok := p.getCharAt(0)
	for ok && char != ']' && char != '}' {
		p.skipWhitespaces()
		var value any
		if isStringDelimiter(char) {
			i := 1
			i = p.skipToCharacter(char, i)
			i = p.scrollWhitespaces(i + 1)
			if nextChar, ok := p.getCharAt(i); ok && nextChar == ':' {
				value, _ = p.parseObject()
			} else {
				value, _ = p.parseString()
			}
		} else {
			var err error
			value, err = p.parseJSON()
			if err != nil {
				return nil, err
			}
		}

		if isStrictlyEmpty(value) {
			if nextChar, ok := p.getCharAt(0); !ok || (nextChar != ']' && nextChar != ',') {
				p.index++
			} else {
				arr = append(arr, value)
			}
		} else if strVal, ok := value.(string); ok && strVal == "..." {
			if prev, ok := p.getCharAt(-1); ok && prev == '.' {
				p.log("While parsing an array, found a stray '...'; ignoring it")
			} else {
				arr = append(arr, value)
			}
		} else {
			arr = append(arr, value)
		}

		char, ok = p.getCharAt(0)
		for ok && char != ']' && (unicode.IsSpace(char) || char == ',') {
			p.index++
			char, ok = p.getCharAt(0)
		}
	}

	if char != ']' {
		p.log("While parsing an array we missed the closing ], ignoring it")
	}

	p.index++
	p.context.reset()
	return arr, nil
}

func (p *parser) parseComment() (any, error) {
	char, ok := p.getCharAt(0)
	if !ok {
		return "", nil
	}
	termination := map[rune]struct{}{'\n': {}, '\r': {}}
	if p.context.contains(contextArray) {
		termination[']'] = struct{}{}
	}
	if p.context.contains(contextObjectValue) {
		termination['}'] = struct{}{}
	}
	if p.context.contains(contextObjectKey) {
		termination[':'] = struct{}{}
	}
	if char == '#' {
		comment := []rune{}
		for ok {
			if _, hit := termination[char]; hit {
				break
			}
			comment = append(comment, char)
			p.index++
			char, ok = p.getCharAt(0)
		}
		p.log("Found line comment: " + string(comment) + ", ignoring")
	} else if char == '/' {
		nextChar, ok := p.getCharAt(1)
		if ok && nextChar == '/' {
			comment := []rune{'/', '/'}
			p.index += 2
			char, ok = p.getCharAt(0)
			for ok {
				if _, hit := termination[char]; hit {
					break
				}
				comment = append(comment, char)
				p.index++
				char, ok = p.getCharAt(0)
			}
			p.log("Found line comment: " + string(comment) + ", ignoring")
		} else if ok && nextChar == '*' {
			comment := []rune{'/', '*'}
			p.index += 2
			for {
				char, ok = p.getCharAt(0)
				if !ok {
					p.log("Reached end-of-string while parsing block comment; unclosed block comment.")
					break
				}
				comment = append(comment, char)
				p.index++
				if len(comment) >= 2 && comment[len(comment)-2] == '*' && comment[len(comment)-1] == '/' {
					break
				}
			}
			p.log("Found block comment: " + string(comment) + ", ignoring")
		} else {
			p.index++
		}
	}
	if p.context.empty {
		return p.parseJSON()
	}
	return "", nil
}

func (p *parser) parseNumber() (any, error) {
	numberChars := "0123456789-.eE/,_"
	numberStr := ""
	char, ok := p.getCharAt(0)
	isArray := p.context.current != nil && *p.context.current == contextArray
	for ok && strings.ContainsRune(numberChars, char) && (!isArray || char != ',' || strings.Contains(numberStr, "/")) {
		if char != '_' {
			numberStr += string(char)
		}
		p.index++
		char, ok = p.getCharAt(0)
	}
	if nextChar, ok := p.getCharAt(0); ok && unicode.IsLetter(nextChar) {
		p.index -= len([]rune(numberStr))
		return p.parseString()
	}
	if len(numberStr) > 0 {
		last := numberStr[len(numberStr)-1]
		if last == '-' || last == 'e' || last == 'E' || last == '/' || last == ',' {
			numberStr = numberStr[:len(numberStr)-1]
			p.index--
		}
	}
	if strings.Contains(numberStr, "/") || strings.Contains(numberStr, "-") || strings.Contains(numberStr, ",") {
		if numberStr == "-" {
			return "", nil
		}
		if strings.ContainsAny(numberStr, "eE") {
			floatVal, err := strconv.ParseFloat(numberStr, 64)
			if err == nil {
				formatted := formatFloat(floatVal)
				return numberValue{raw: formatted}, nil
			}
			return numberStr, nil
		}
		return numberStr, nil
	}
	if strings.ContainsAny(numberStr, ".eE") {
		floatVal, err := strconv.ParseFloat(numberStr, 64)
		if err == nil {
			formatted := formatFloat(floatVal)
			return numberValue{raw: formatted}, nil
		}
		return numberStr, nil
	}
	if numberStr == "" {
		return "", nil
	}
	return numberValue{raw: numberStr}, nil
}

func (p *parser) parseObject() (any, error) {
	obj := newOrderedObject()
	startIndex := p.index
	for {
		p.skipWhitespaces()
		char, ok := p.getCharAt(0)
		if !ok || char == '}' {
			break
		}
		if current, ok := p.getCharAt(0); ok && current == ':' {
			p.log("While parsing an object we found a : before a key, ignoring")
			p.index++
		}
		p.context.set(contextObjectKey)
		rollbackIndex := p.index
		key := ""
		for {
			current, ok := p.getCharAt(0)
			if !ok {
				break
			}
			rollbackIndex = p.index
			if current == '[' && key == "" {
				prevKey, ok := obj.lastKey()
				if ok {
					prevValue, _ := obj.get(prevKey)
					if prevArray, ok := prevValue.([]any); ok && !p.strict {
						p.index++
						newArrayValue, err := p.parseArray()
						if err != nil {
							return nil, err
						}
						if newArray, ok := newArrayValue.([]any); ok {
							listLengths := []int{}
							for _, item := range prevArray {
								if nested, ok := item.([]any); ok {
									listLengths = append(listLengths, len(nested))
								}
							}
							expectedLen := 0
							if len(listLengths) > 0 {
								same := true
								for _, length := range listLengths {
									if length != listLengths[0] {
										same = false
										break
									}
								}
								if same {
									expectedLen = listLengths[0]
								}
							}
							if expectedLen > 0 {
								tail := []any{}
								for len(prevArray) > 0 {
									if _, ok := prevArray[len(prevArray)-1].([]any); ok {
										break
									}
									tail = append(tail, prevArray[len(prevArray)-1])
									prevArray = prevArray[:len(prevArray)-1]
								}
								if len(tail) > 0 {
									reverseAny(tail)
									if len(tail)%expectedLen == 0 {
										p.log("While parsing an object we found row values without an inner array, grouping them into rows")
										for i := 0; i < len(tail); i += expectedLen {
											prevArray = append(prevArray, tail[i:i+expectedLen])
										}
									} else {
										prevArray = append(prevArray, tail...)
									}
								}
								if len(newArray) > 0 {
									allLists := true
									for _, item := range newArray {
										if _, ok := item.([]any); !ok {
											allLists = false
											break
										}
									}
									if allLists {
										p.log("While parsing an object we found additional rows, appending them without flattening")
										prevArray = append(prevArray, newArray...)
									} else {
										prevArray = append(prevArray, newArray)
									}
								}
							} else {
								if len(newArray) == 1 {
									if nested, ok := newArray[0].([]any); ok {
										prevArray = append(prevArray, nested...)
									} else {
										prevArray = append(prevArray, newArray...)
									}
								} else {
									prevArray = append(prevArray, newArray...)
								}
							}
							obj.set(prevKey, prevArray)
						}
						p.skipWhitespaces()
						if nextChar, ok := p.getCharAt(0); ok && nextChar == ',' {
							p.index++
						}
						p.skipWhitespaces()
						continue
					}
				}
			}
			rawKeyValue, err := p.parseString()
			if err != nil {
				return nil, err
			}
			rawKey, _ := rawKeyValue.(string)
			key = rawKey
			if key == "" {
				p.skipWhitespaces()
			}
			if key != "" || (key == "" && func() bool { ch, ok := p.getCharAt(0); return ok && (ch == ':' || ch == '}') }()) {
				if key == "" && p.strict {
					p.log("Empty key found in strict mode while parsing object, raising an error")
					return nil, errors.New("empty key found in strict mode while parsing object")
				}
				break
			}
		}
		if p.context.contains(contextArray) && obj.hasKey(key) {
			if p.strict {
				p.log("Duplicate key found in strict mode while parsing object, raising an error")
				return nil, errors.New("duplicate key found in strict mode while parsing object")
			}
			p.log("While parsing an object we found a duplicate key, closing the object here and rolling back the index")
			p.index = rollbackIndex - 1
			p.insertRune(p.index+1, '{')
			break
		}
		p.skipWhitespaces()
		if current, ok := p.getCharAt(0); !ok || current == '}' {
			continue
		}
		p.skipWhitespaces()
		if current, ok := p.getCharAt(0); ok && current != ':' {
			if p.strict {
				p.log("Missing ':' after key in strict mode while parsing object, raising an error")
				return nil, errors.New("missing ':' after key in strict mode while parsing object")
			}
			p.log("While parsing an object we missed a : after a key")
		}
		p.index++
		p.context.reset()
		p.context.set(contextObjectValue)
		p.skipWhitespaces()
		value := any("")
		if current, ok := p.getCharAt(0); ok && (current == ',' || current == '}') {
			p.log("While parsing an object value we found a stray " + string(current) + ", ignoring it")
		} else {
			var err error
			value, err = p.parseJSON()
			if err != nil {
				return nil, err
			}
		}
		if value == "" && p.strict {
			if prev, ok := p.getCharAt(-1); !ok || !isStringDelimiter(prev) {
				p.log("Parsed value is empty in strict mode while parsing object, raising an error")
				return nil, errors.New("parsed value is empty in strict mode while parsing object")
			}
		}
		p.context.reset()
		obj.set(key, value)
		if current, ok := p.getCharAt(0); ok && (current == ',' || current == '\'' || current == '"') {
			p.index++
		}
		if current, ok := p.getCharAt(0); ok && current == ']' && p.context.contains(contextArray) {
			p.log("While parsing an object we found a closing array bracket, closing the object here and rolling back the index")
			p.index--
			break
		}
		p.skipWhitespaces()
	}
	p.index++
	if len(obj.entries) == 0 && p.index-startIndex > 2 {
		if p.strict {
			p.log("Parsed object is empty but contains extra characters in strict mode, raising an error")
			return nil, errors.New("parsed object is empty but contains extra characters in strict mode")
		}
		if p.context.empty && p.index-startIndex <= 3 {
			return obj, nil
		}
		if p.context.empty {
			prefix := string(p.jsonStr[:startIndex-1])
			if strings.TrimSpace(prefix) == "" {
				return obj, nil
			}
		}
		p.log("Parsed object is empty, we will try to parse this as an array instead")
		p.index = startIndex
		return p.parseArray()
	}
	if len(obj.entries) == 0 && p.index-startIndex <= 2 {
		return obj, nil
	}
	if !p.context.empty {
		if current, ok := p.getCharAt(0); ok && current == '}' {
			if p.context.current == nil || (*p.context.current != contextObjectKey && *p.context.current != contextObjectValue) {
				p.log("Found an extra closing brace that shouldn't be there, skipping it")
				p.index++
			}
		}
		return obj, nil
	}
	p.skipWhitespaces()
	if current, ok := p.getCharAt(0); !ok || current != ',' {
		return obj, nil
	}
	p.index++
	p.skipWhitespaces()
	if current, ok := p.getCharAt(0); !ok || !isStringDelimiter(current) {
		return obj, nil
	}
	if !p.strict {
		p.log("Found a comma and string delimiter after object closing brace, checking for additional key-value pairs")
		additionalValue, err := p.parseObject()
		if err != nil {
			return nil, err
		}
		if additionalObj, ok := additionalValue.(*orderedObject); ok {
			obj.merge(additionalObj)
		}
	}
	return obj, nil
}

func (p *parser) parseString() (any, error) {
	missingQuotes := false
	doubledQuotes := false
	ldelim := '"'
	rdelim := '"'

	char, ok := p.getCharAt(0)
	if ok && (char == '#' || char == '/') {
		return p.parseComment()
	}
	for ok && !isStringDelimiter(char) && !isAlphaNum(char) {
		p.index++
		char, ok = p.getCharAt(0)
	}
	if !ok {
		return "", nil
	}
	if char == '\'' {
		ldelim = '\''
		rdelim = '\''
	} else if char == '“' {
		ldelim = '“'
		rdelim = '”'
	} else if isAlphaNum(char) {
		if (char == 't' || char == 'f' || char == 'n') && (p.context.current == nil || *p.context.current != contextObjectKey) {
			value := p.parseBooleanOrNull()
			if value != "" {
				return value, nil
			}
		}
		if (char == 'T' || char == 'F' || char == 'N') && (p.context.current == nil || *p.context.current != contextObjectKey) {
			value := p.parseBooleanOrNull()
			if value != "" {
				return value, nil
			}
		}
		p.log("While parsing a string, we found a literal instead of a quote")
		missingQuotes = true
	}

	if !missingQuotes {
		p.index++
	}
	if next, ok := p.getCharAt(0); ok && next == '`' {
		if value, ok := p.parseJSONLLMBlock(); ok {
			return value, nil
		}
		if p.context.empty {
			return "", nil
		}
		p.log("While parsing a string, we found code fences but they did not enclose valid JSON, continuing parsing the string")
	}

	if next, ok := p.getCharAt(0); ok && next == ldelim {
		if (p.context.current != nil && *p.context.current == contextObjectKey && func() bool { ch, ok := p.getCharAt(1); return ok && ch == ':' }()) ||
			(p.context.current != nil && *p.context.current == contextObjectValue && func() bool { ch, ok := p.getCharAt(1); return ok && (ch == ',' || ch == '}') }()) ||
			(p.context.current != nil && *p.context.current == contextArray && func() bool { ch, ok := p.getCharAt(1); return ok && (ch == ',' || ch == ']') }()) {
			p.index++
			return "", nil
		}
		if p.context.current != nil && *p.context.current == contextObjectKey {
			i := p.scrollWhitespaces(1)
			if ch, ok := p.getCharAt(i); ok && ch == ':' {
				p.index++
				return "", nil
			}
		}
		if next2, ok := p.getCharAt(1); ok && next2 == ldelim {
			p.log("While parsing a string, we found a doubled quote and then a quote again, ignoring it")
			if p.strict {
				return nil, errors.New("found doubled quotes followed by another quote")
			}
			return "", nil
		}
		i := p.skipToCharacter(rdelim, 1)
		if nextChar, ok := p.getCharAt(i + 1); ok && nextChar == rdelim {
			p.log("While parsing a string, we found a valid starting doubled quote")
			doubledQuotes = true
			p.index++
		} else {
			i = p.scrollWhitespaces(1)
			nextChar, ok := p.getCharAt(i)
			if ok && (isStringDelimiter(nextChar) || nextChar == '{' || nextChar == '[') {
				p.log("While parsing a string, we found a doubled quote but also another quote afterwards, ignoring it")
				if p.strict {
					return nil, errors.New("found doubled quotes followed by another quote while parsing a string")
				}
				p.index++
				return "", nil
			}
			if !ok || (nextChar != ',' && nextChar != ']' && nextChar != '}') {
				p.log("While parsing a string, we found a doubled quote but it was a mistake, removing one quote")
				p.index++
			}
		}
	}

	stringAcc := []rune{}
	char, ok = p.getCharAt(0)
	unmatchedDelimiter := false
	for ok && char != rdelim {
		if missingQuotes {
			if p.context.current != nil && *p.context.current == contextObjectKey {
				if char == ':' || unicode.IsSpace(char) {
					p.log("While parsing a string missing the left delimiter in object key context, we found a :, stopping here")
					break
				}
			}
			if p.context.current != nil && *p.context.current == contextArray {
				if char == ']' || char == ',' {
					p.log("While parsing a string missing the left delimiter in array context, we found a ] or ,, stopping here")
					break
				}
			}
		}
		if !p.streamStable && p.context.current != nil && *p.context.current == contextObjectValue {
			if (char == ',' || char == '}') && (len(stringAcc) == 0 || stringAcc[len(stringAcc)-1] != rdelim) {
				rstringDelimiterMissing := true
				next := rune(0)
				p.skipWhitespaces()
				if next, ok := p.getCharAt(1); ok && next == '\\' {
					rstringDelimiterMissing = false
				}
				i := p.skipToCharacter(rdelim, 1)
				if _, ok := p.getCharAt(i); ok {
					i++
					i = p.scrollWhitespaces(i)
					next, _ = p.getCharAt(i)
					if next == ',' || next == '}' {
						rstringDelimiterMissing = false
					} else {
						i = p.skipToCharacter(ldelim, i)
						if _, ok := p.getCharAt(i); !ok {
							rstringDelimiterMissing = false
						} else {
							i = p.scrollWhitespaces(i + 1)
							next, _ = p.getCharAt(i)
							if next != ':' {
								rstringDelimiterMissing = false
							}
						}
					}
				} else {
					i = p.skipToCharacter(':', 1)
					if _, ok := p.getCharAt(i); ok {
						break
					}
					i = p.scrollWhitespaces(1)
					j := p.skipToCharacter('}', i)
					if j-i > 1 {
						rstringDelimiterMissing = false
					} else if _, ok := p.getCharAt(j); ok {
						for k := len(stringAcc) - 1; k >= 0; k-- {
							if stringAcc[k] == '{' {
								rstringDelimiterMissing = false
								break
							}
						}
					}
				}
				if rstringDelimiterMissing {
					p.log("While parsing a string missing the left delimiter in object value context, we found a , or } and we couldn't determine that a right delimiter was present. Stopping here")
					break
				}
			}
		}
		if !p.streamStable && p.context.contains(contextArray) && char == ']' {
			i := p.skipToCharacter(rdelim, 0)
			if _, ok := p.getCharAt(i); !ok {
				break
			}
		}
		if p.context.current != nil && *p.context.current == contextObjectValue && char == '}' {
			i := p.scrollWhitespaces(1)
			nextChar, ok := p.getCharAt(i)
			if ok && nextChar == '`' {
				if c1, ok := p.getCharAt(i + 1); ok && c1 == '`' {
					if c2, ok := p.getCharAt(i + 2); ok && c2 == '`' {
						p.log("While parsing a string in object value context, we found a } that closes the object before code fences, stopping here")
						break
					}
				}
			}
			if !ok {
				p.log("While parsing a string in object value context, we found a } that closes the object, stopping here")
				break
			}
		}
		stringAcc = append(stringAcc, char)
		p.index++
		char, ok = p.getCharAt(0)
		if !ok {
			if p.streamStable && len(stringAcc) > 0 && stringAcc[len(stringAcc)-1] == '\\' {
				stringAcc = stringAcc[:len(stringAcc)-1]
			}
			break
		}
		if len(stringAcc) > 0 && stringAcc[len(stringAcc)-1] == '\\' {
			p.log("Found a stray escape sequence, normalizing it")
			if char == rdelim || char == 't' || char == 'n' || char == 'r' || char == 'b' || char == '\\' {
				stringAcc = stringAcc[:len(stringAcc)-1]
				escapeSeqs := map[rune]rune{'t': '\t', 'n': '\n', 'r': '\r', 'b': '\b'}
				if replacement, ok := escapeSeqs[char]; ok {
					stringAcc = append(stringAcc, replacement)
				} else {
					stringAcc = append(stringAcc, char)
				}
				p.index++
				char, ok = p.getCharAt(0)
				for ok && len(stringAcc) > 0 && stringAcc[len(stringAcc)-1] == '\\' && (char == rdelim || char == '\\') {
					stringAcc = append(stringAcc[:len(stringAcc)-1], char)
					p.index++
					char, ok = p.getCharAt(0)
				}
				continue
			}
			if char == 'u' || char == 'x' {
				numChars := 4
				if char == 'x' {
					numChars = 2
				}
				nextChars := p.sliceRunes(p.index+1, p.index+1+numChars)
				if len(nextChars) == numChars && isHexString(string(nextChars)) {
					p.log("Found a unicode escape sequence, normalizing it")
					parsed, _ := strconv.ParseInt(string(nextChars), 16, 32)
					stringAcc = append(stringAcc[:len(stringAcc)-1], rune(parsed))
					p.index += 1 + numChars
					char, ok = p.getCharAt(0)
					continue
				}
			} else if isStringDelimiter(char) && char != rdelim {
				p.log("Found a delimiter that was escaped but shouldn't be escaped, removing the escape")
				stringAcc = append(stringAcc[:len(stringAcc)-1], char)
				p.index++
				char, ok = p.getCharAt(0)
				continue
			}
		}
		if char == ':' && !missingQuotes && p.context.current != nil && *p.context.current == contextObjectKey {
			i := p.skipToCharacter(ldelim, 1)
			if _, ok := p.getCharAt(i); ok {
				i++
				i = p.skipToCharacter(rdelim, i)
				if _, ok := p.getCharAt(i); ok {
					i++
					i = p.scrollWhitespaces(i)
					ch, ok := p.getCharAt(i)
					if ok && (ch == ',' || ch == '}') {
						p.log("While parsing a string missing the right delimiter in object key context, we found a " + string(ch) + " stopping here")
						break
					}
				}
			} else {
				p.log("While parsing a string missing the right delimiter in object key context, we found a :, stopping here")
				break
			}
		}
		if char == rdelim && (len(stringAcc) == 0 || stringAcc[len(stringAcc)-1] != '\\') {
			if doubledQuotes {
				if next, ok := p.getCharAt(1); ok && next == rdelim {
					p.log("While parsing a string, we found a doubled quote, ignoring it")
					p.index++
				}
			} else if missingQuotes && p.context.current != nil && *p.context.current == contextObjectValue {
				i := 1
				nextChar, ok := p.getCharAt(i)
				for ok && nextChar != rdelim && nextChar != ldelim {
					i++
					nextChar, ok = p.getCharAt(i)
				}
				if ok {
					i++
					i = p.scrollWhitespaces(i)
					if ch, ok := p.getCharAt(i); ok && ch == ':' {
						p.index--
						char, _ = p.getCharAt(0)
						p.log("In a string with missing quotes and object value context, I found a delimeter but it turns out it was the beginning on the next key. Stopping here.")
						break
					}
				}
			} else if unmatchedDelimiter {
				unmatchedDelimiter = false
				stringAcc = append(stringAcc, char)
				p.index++
				char, ok = p.getCharAt(0)
			} else {
				i := 1
				nextChar, ok := p.getCharAt(i)
				checkCommaInObjectValue := true
				for ok && nextChar != rdelim && nextChar != ldelim {
					if checkCommaInObjectValue && unicode.IsLetter(nextChar) {
						checkCommaInObjectValue = false
					}
					if (p.context.contains(contextObjectKey) && (nextChar == ':' || nextChar == '}')) ||
						(p.context.contains(contextObjectValue) && nextChar == '}') ||
						(p.context.contains(contextArray) && (nextChar == ']' || nextChar == ',')) ||
						(checkCommaInObjectValue && p.context.current != nil && *p.context.current == contextObjectValue && nextChar == ',') {
						break
					}
					i++
					nextChar, ok = p.getCharAt(i)
				}
				if nextChar == ',' && p.context.current != nil && *p.context.current == contextObjectValue {
					i++
					i = p.skipToCharacter(rdelim, i)
					i++
					i = p.scrollWhitespaces(i)
					nextChar, _ = p.getCharAt(i)
					if nextChar == '}' || nextChar == ',' {
						p.log("While parsing a string, we found a misplaced quote that would have closed the string but has a different meaning here, ignoring it")
						stringAcc = append(stringAcc, char)
						p.index++
						char, ok = p.getCharAt(0)
						if !ok {
							break
						}
						continue
					}
				} else if nextChar == rdelim && func() bool { prev, ok := p.getCharAt(i - 1); return ok && prev != '\\' }() {
					if onlyWhitespaceUntil(p, i) {
						break
					}
					if p.context.current != nil && *p.context.current == contextObjectValue {
						i = p.scrollWhitespaces(i + 1)
						if ch, ok := p.getCharAt(i); ok && ch == ',' {
							i = p.skipToCharacter(ldelim, i+1)
							i++
							i = p.skipToCharacter(rdelim, i+1)
							i++
							i = p.scrollWhitespaces(i)
							if ch, ok := p.getCharAt(i); ok && ch == ':' {
								p.log("While parsing a string, we found a misplaced quote that would have closed the string but has a different meaning here, ignoring it")
								stringAcc = append(stringAcc, char)
								p.index++
								char, ok = p.getCharAt(0)
								if !ok {
									break
								}
								continue
							}
						}
						i = p.skipToCharacter(rdelim, i+1)
						i++
						nextChar, ok = p.getCharAt(i)
						for ok && nextChar != ':' {
							if nextChar == ',' || nextChar == ']' || nextChar == '}' || (nextChar == rdelim && func() bool { prev, ok := p.getCharAt(i - 1); return ok && prev != '\\' }()) {
								break
							}
							i++
							nextChar, ok = p.getCharAt(i)
						}
						if nextChar != ':' {
							p.log("While parsing a string, we found a misplaced quote that would have closed the string but has a different meaning here, ignoring it")
							unmatchedDelimiter = !unmatchedDelimiter
							stringAcc = append(stringAcc, char)
							p.index++
							char, ok = p.getCharAt(0)
							if !ok {
								break
							}
						}
					} else if p.context.current != nil && *p.context.current == contextArray {
						evenDelimiters := nextChar == rdelim
						for nextChar == rdelim {
							i = p.skipToCharacters(map[rune]struct{}{rdelim: {}, ']': {}}, i+1)
							nextChar, ok = p.getCharAt(i)
							if !ok || nextChar != rdelim {
								evenDelimiters = false
								break
							}
							i = p.skipToCharacters(map[rune]struct{}{rdelim: {}, ']': {}}, i+1)
							nextChar, _ = p.getCharAt(i)
						}
						if evenDelimiters {
							p.log("While parsing a string in Array context, we detected a quoted section that would have closed the string but has a different meaning here, ignoring it")
							unmatchedDelimiter = !unmatchedDelimiter
							stringAcc = append(stringAcc, char)
							p.index++
							char, ok = p.getCharAt(0)
							if !ok {
								break
							}
						} else {
							break
						}
					} else if p.context.current != nil && *p.context.current == contextObjectKey {
						p.log("While parsing a string in Object Key context, we detected a quoted section that would have closed the string but has a different meaning here, ignoring it")
						stringAcc = append(stringAcc, char)
						p.index++
						char, ok = p.getCharAt(0)
						if !ok {
							break
						}
					}
				}
			}
		}
	}
	if ok && missingQuotes && p.context.current != nil && *p.context.current == contextObjectKey && unicode.IsSpace(char) {
		p.log("While parsing a string, handling an extreme corner case in which the LLM added a comment instead of valid string, invalidate the string and return an empty value")
		p.skipWhitespaces()
		if ch, ok := p.getCharAt(0); ok {
			if ch != ':' && ch != ',' {
				p.index--
				return "", nil
			}
			if ch == ',' {
				p.index--
				return "", nil
			}
		}
	}
	if missingQuotes && p.context.current != nil && *p.context.current == contextObjectKey {
		if !onlyWhitespaceUntil(p, p.scrollWhitespaces(0)) {
			stringAcc = trimRightWhitespace(stringAcc)
			if len(stringAcc) == 0 {
				return "", nil
			}
		}
	}
	if !ok || char != rdelim {
		if !p.streamStable {
			p.log("While parsing a string, we missed the closing quote, ignoring")
			stringAcc = trimRightWhitespace(stringAcc)
		}
	} else {
		p.index++
	}
	if !p.streamStable && (missingQuotes || (len(stringAcc) > 0 && stringAcc[len(stringAcc)-1] == '\n')) {
		stringAcc = trimRightWhitespace(stringAcc)
	}
	if missingQuotes && p.context.empty {
		next := p.scrollWhitespaces(0)
		if ch, ok := p.getCharAt(next); ok && (ch == '{' || ch == '[' || ch == '`') {
			return "", nil
		}
		if !p.streamStable {
			stringAcc = trimRightWhitespace(stringAcc)
		}
		if len(stringAcc) == 0 {
			return "", nil
		}
	}
	if p.context.empty {
		next := p.scrollWhitespaces(0)
		if ch, ok := p.getCharAt(next); ok && (ch == '{' || ch == '[' || ch == '`') {
			return "", nil
		}
	}
	if len(stringAcc) == 1 && stringAcc[0] == rdelim {
		return "", nil
	}
	if p.context.empty && missingQuotes {
		if len(stringAcc) == 1 && stringAcc[0] == '"' {
			return "", nil
		}
	}
	return string(stringAcc), nil
}

func (p *parser) parseBooleanOrNull() any {
	char, ok := p.getCharAt(0)
	if !ok {
		return ""
	}
	valueMap := map[rune]struct {
		token string
		value any
	}{
		't': {"true", true},
		'f': {"false", false},
		'n': {"null", nil},
	}
	lower := unicode.ToLower(char)
	value, ok := valueMap[lower]
	if !ok {
		return ""
	}
	matchUpper := unicode.IsUpper(char)
	i := 0
	startingIndex := p.index
	current := lower
	for ok && i < len(value.token) && current == rune(value.token[i]) {
		i++
		p.index++
		char, ok = p.getCharAt(0)
		if ok {
			if unicode.IsUpper(char) {
				matchUpper = true
			}
			current = unicode.ToLower(char)
		}
	}
	if i == len(value.token) {
		if matchUpper && p.context.empty {
			p.index = startingIndex
			return ""
		}
		return value.value
	}
	p.index = startingIndex
	return ""
}

func (p *parser) parseJSONLLMBlock() (any, bool) {
	if p.sliceString(p.index, p.index+7) == "```json" {
		i := p.skipToCharacter('`', 7)
		if p.sliceString(p.index+i, p.index+i+3) == "```" {
			p.index += 7
			value, err := p.parseJSON()
			if err != nil {
				return nil, false
			}
			return value, true
		}
	}
	return nil, false
}

func (p *parser) sliceRunes(start int, end int) []rune {
	if start < 0 {
		start = 0
	}
	if end > len(p.jsonStr) {
		end = len(p.jsonStr)
	}
	if start > end {
		return []rune{}
	}
	return p.jsonStr[start:end]
}

func (p *parser) sliceString(start int, end int) string {
	return string(p.sliceRunes(start, end))
}

func (p *parser) insertRune(pos int, r rune) {
	if pos < 0 {
		pos = 0
	}
	if pos > len(p.jsonStr) {
		pos = len(p.jsonStr)
	}
	p.jsonStr = append(p.jsonStr[:pos], append([]rune{r}, p.jsonStr[pos:]...)...)
}

func onlyWhitespaceUntil(p *parser, end int) bool {
	for j := 1; j < end; j++ {
		c, ok := p.getCharAt(j)
		if ok && !unicode.IsSpace(c) {
			return false
		}
	}
	return true
}

func onlyWhitespaceBefore(p *parser) bool {
	for i := p.index - 1; i >= 0; i-- {
		c := p.jsonStr[i]
		if !unicode.IsSpace(c) {
			return false
		}
	}
	return true
}

func reverseAny(values []any) {
	for i, j := 0, len(values)-1; i < j; i, j = i+1, j-1 {
		values[i], values[j] = values[j], values[i]
	}
}

func isStrictlyEmpty(value any) bool {
	switch v := value.(type) {
	case string:
		return len(v) == 0
	case []any:
		return len(v) == 0
	case *orderedObject:
		return len(v.entries) == 0
	default:
		return false
	}
}

func isSameObject(obj1 any, obj2 any) bool {
	switch v1 := obj1.(type) {
	case *orderedObject:
		v2, ok := obj2.(*orderedObject)
		if !ok {
			return false
		}
		if len(v1.entries) != len(v2.entries) {
			return false
		}
		for _, entry := range v1.entries {
			val2, ok := v2.get(entry.key)
			if !ok {
				return false
			}
			if !isSameObject(entry.value, val2) {
				return false
			}
		}
		return true
	case []any:
		v2, ok := obj2.([]any)
		if !ok {
			return false
		}
		if len(v1) != len(v2) {
			return false
		}
		for i := range v1 {
			if !isSameObject(v1[i], v2[i]) {
				return false
			}
		}
		return true
	default:
		if obj1 == nil || obj2 == nil {
			return obj1 == obj2
		}
		return reflect.TypeOf(obj1) == reflect.TypeOf(obj2)
	}
}

func isTruthy(value any) bool {
	switch v := value.(type) {
	case string:
		return v != ""
	case []any:
		return len(v) > 0
	case *orderedObject:
		return len(v.entries) > 0
	case bool:
		return v
	case numberValue:
		return v.raw != ""
	case nil:
		return false
	default:
		return true
	}
}

func isStringDelimiter(char rune) bool {
	switch char {
	case '"', '\'', '“', '”':
		return true
	default:
		return false
	}
}

func isAlphaNum(char rune) bool {
	return unicode.IsLetter(char) || unicode.IsDigit(char)
}

func trimRightWhitespace(values []rune) []rune {
	for len(values) > 0 {
		if !unicode.IsSpace(values[len(values)-1]) {
			break
		}
		values = values[:len(values)-1]
	}
	return values
}

func isHexString(value string) bool {
	if value == "" {
		return false
	}
	for _, c := range value {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

func formatFloat(value float64) string {
	formatted := strconv.FormatFloat(value, 'f', -1, 64)
	if !strings.Contains(formatted, ".") {
		formatted += ".0"
	}
	return formatted
}

func applyOptions(opts []Option) options {
	cfg := options{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return cfg
}

func ensureASCIIValue(cfg options) bool {
	if cfg.ensureASCII == nil {
		return true
	}
	return *cfg.ensureASCII
}

// WithEnsureASCII sets whether to escape non-ASCII characters.
func WithEnsureASCII(value bool) Option {
	return func(o *options) {
		o.ensureASCII = &value
	}
}

// WithSkipJSONLoads skips JSON parsing during load.
func WithSkipJSONLoads() Option {
	return func(o *options) {
		o.skipJSONLoads = true
	}
}

// WithStreamStable enables streaming-stable parsing.
func WithStreamStable() Option {
	return func(o *options) {
		o.streamStable = true
	}
}

// WithStrict enables strict parsing mode.
func WithStrict() Option {
	return func(o *options) {
		o.strict = true
	}
}

// RepairJSON takes a potentially malformed JSON string output from LLMs and
// attempts to repair it into a valid JSON string. It returns the repaired JSON
// string or an error if the input cannot be repaired.
func RepairJSON(input string, opts ...Option) (string, error) {
	cfg := applyOptions(opts)
	p := newParser(input, false, cfg.streamStable, cfg.strict)
	value, _, err := p.parse()
	if err != nil {
		return "", err
	}
	if str, ok := value.(string); ok {
		trimmed := strings.TrimSpace(str)
		if str == "" || trimmed == "" {
			return "", nil
		}
		return "", nil
	}
	if value == "" {
		return "", nil
	}
	return serialize(value, ensureASCIIValue(cfg)), nil
}

// Loads takes a potentially malformed JSON string output from LLMs and attempts
// to repair it and parse it into a Go value.
func Loads(input string, opts ...Option) (any, error) {
	cfg := applyOptions(opts)
	p := newParser(input, false, cfg.streamStable, cfg.strict)
	value, _, err := p.parse()
	if err != nil {
		return nil, err
	}
	if value == "" {
		return "", nil
	}
	return normalizeValue(value), nil
}

// RepairJSONWithLog takes a potentially malformed JSON string output from LLMs
// and attempts to repair it into a valid JSON string, while also returning logs
// of the repair process.
func RepairJSONWithLog(input string, opts ...Option) (any, []LogEntry, error) {
	cfg := applyOptions(opts)
	p := newParser(input, true, cfg.streamStable, cfg.strict)
	value, logs, err := p.parse()
	if err != nil {
		return nil, nil, err
	}
	if logs == nil {
		logs = []LogEntry{}
	}
	if value == "" {
		return "", logs, nil
	}
	return normalizeValue(value), logs, nil
}

func normalizeValue(value any) any {
	switch v := value.(type) {
	case *orderedObject:
		result := map[string]any{}
		for _, entry := range v.entries {
			result[entry.key] = normalizeValue(entry.value)
		}
		return result
	case []any:
		items := make([]any, 0, len(v))
		for _, item := range v {
			items = append(items, normalizeValue(item))
		}
		return items
	case numberValue:
		return json.Number(v.raw)
	default:
		return v
	}
}

func serialize(value any, ensureASCII bool) string {
	var buf bytes.Buffer
	writeValue(&buf, value, ensureASCII)
	return buf.String()
}

func writeValue(buf *bytes.Buffer, value any, ensureASCII bool) {
	switch v := value.(type) {
	case string:
		buf.WriteByte('"')
		writeEscapedString(buf, v, ensureASCII)
		buf.WriteByte('"')
	case numberValue:
		buf.WriteString(v.raw)
	case json.Number:
		buf.WriteString(v.String())
	case bool:
		if v {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case nil:
		buf.WriteString("null")
	case []any:
		buf.WriteByte('[')
		for i, item := range v {
			if i > 0 {
				buf.WriteString(", ")
			}
			writeValue(buf, item, ensureASCII)
		}
		buf.WriteByte(']')
	case *orderedObject:
		buf.WriteByte('{')
		for i, entry := range v.entries {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteByte('"')
			writeEscapedString(buf, entry.key, ensureASCII)
			buf.WriteByte('"')
			buf.WriteString(": ")
			writeValue(buf, entry.value, ensureASCII)
		}
		buf.WriteByte('}')
	case float64:
		buf.WriteString(formatFloat(v))
	case int:
		buf.WriteString(strconv.Itoa(v))
	case int64:
		buf.WriteString(strconv.FormatInt(v, 10))
	case uint64:
		buf.WriteString(strconv.FormatUint(v, 10))
	case map[string]any:
		buf.WriteByte('{')
		idx := 0
		for key, item := range v {
			if idx > 0 {
				buf.WriteString(", ")
			}
			buf.WriteByte('"')
			writeEscapedString(buf, key, ensureASCII)
			buf.WriteByte('"')
			buf.WriteString(": ")
			writeValue(buf, item, ensureASCII)
			idx++
		}
		buf.WriteByte('}')
	default:
		buf.WriteString("null")
	}
}

func writeEscapedString(buf *bytes.Buffer, value string, ensureASCII bool) {
	for _, r := range value {
		switch r {
		case '\\':
			buf.WriteString("\\\\")
		case '"':
			buf.WriteString("\\\"")
		case '\b':
			buf.WriteString("\\b")
		case '\f':
			buf.WriteString("\\f")
		case '\n':
			buf.WriteString("\\n")
		case '\r':
			buf.WriteString("\\r")
		case '\t':
			buf.WriteString("\\t")
		default:
			if r < 0x20 {
				buf.WriteString("\\u")
				buf.WriteString(hex4(r))
				continue
			}
			if ensureASCII && r > 0x7f {
				if r > 0xFFFF {
					for _, rr := range utf16.Encode([]rune{r}) {
						buf.WriteString("\\u")
						buf.WriteString(hex4(rune(rr)))
					}
					continue
				}
				buf.WriteString("\\u")
				buf.WriteString(hex4(r))
				continue
			}
			buf.WriteRune(r)
		}
	}
}

func hex4(r rune) string {
	value := int(r)
	result := strconv.FormatInt(int64(value), 16)
	return strings.Repeat("0", 4-len(result)) + strings.ToLower(result)
}
