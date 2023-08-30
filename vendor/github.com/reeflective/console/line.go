package console

import (
	"bytes"
	"errors"
	"strings"
	"unicode/utf8"
)

var (
	splitChars        = " \n\t"
	singleChar        = '\''
	doubleChar        = '"'
	escapeChar        = '\\'
	doubleEscapeChars = "$`\"\n\\"
)

var (
	errUnterminatedSingleQuote = errors.New("unterminated single-quoted string")
	errUnterminatedDoubleQuote = errors.New("unterminated double-quoted string")
	errUnterminatedEscape      = errors.New("unterminated backslash-escape")
)

// acceptMultiline determines if the line just accepted is complete (in which case
// we should execute it), or incomplete (in which case we must read in multiline).
func (c *Console) acceptMultiline(line []rune) (accept bool) {
	// Errors are either: unterminated quotes, or unterminated escapes.
	_, _, err := split(string(line), false)
	if err == nil {
		return true
	}

	// Currently, unterminated quotes are obvious to treat: keep reading.
	switch err {
	case errUnterminatedDoubleQuote, errUnterminatedSingleQuote:
		return false
	case errUnterminatedEscape:
		if len(line) > 0 && line[len(line)-1] == '\\' {
			return false
		}

		return true
	}

	return true
}

// split has been copied from go-shellquote and slightly modified so as to also
// return the remainder when the parsing failed because of an unterminated quote.
func split(input string, hl bool) (words []string, remainder string, err error) {
	var buf bytes.Buffer
	words = make([]string, 0)

	for len(input) > 0 {
		// skip any splitChars at the start
		c, l := utf8.DecodeRuneInString(input)
		if strings.ContainsRune(splitChars, c) {
			// Keep these characters in the result when higlighting the line.
			if hl {
				if len(words) == 0 {
					words = append(words, string(c))
				} else {
					words[len(words)-1] += string(c)
				}
			}

			input = input[l:]
			continue
		} else if c == escapeChar {
			// Look ahead for escaped newline so we can skip over it
			next := input[l:]
			if len(next) == 0 {
				if hl {
					remainder = string(escapeChar)
				}
				err = errUnterminatedEscape
				return
			}
			c2, l2 := utf8.DecodeRuneInString(next)
			if c2 == '\n' {
				if hl {
					if len(words) == 0 {
						words = append(words, string(c)+string(c2))
					} else {
						words[len(words)-1] += string(c) + string(c2)
					}
				}
				input = next[l2:]
				continue
			}
		}

		var word string
		word, input, err = splitWord(input, &buf, hl)
		if err != nil {
			remainder = input
			return
		}
		words = append(words, word)
	}
	return
}

// splitWord has been modified to return the remainder of the input (the part that has not been
// added to the buffer) even when an error is returned.
func splitWord(input string, buf *bytes.Buffer, hl bool) (word string, remainder string, err error) {
	buf.Reset()

raw:
	{
		cur := input
		for len(cur) > 0 {
			c, l := utf8.DecodeRuneInString(cur)
			cur = cur[l:]
			if c == singleChar {
				buf.WriteString(input[0 : len(input)-len(cur)-l])
				input = cur
				goto single
			} else if c == doubleChar {
				buf.WriteString(input[0 : len(input)-len(cur)-l])
				input = cur
				goto double
			} else if c == escapeChar {
				buf.WriteString(input[0 : len(input)-len(cur)-l])
				if hl {
					buf.WriteRune(c)
				}
				input = cur
				goto escape
			} else if strings.ContainsRune(splitChars, c) {
				buf.WriteString(input[0 : len(input)-len(cur)-l])
				if hl {
					buf.WriteRune(c)
				}

				return buf.String(), cur, nil
			}
		}
		if len(input) > 0 {
			buf.WriteString(input)
			input = ""
		}
		goto done
	}

escape:
	{
		if len(input) == 0 {
			if hl {
				input = buf.String() + input
			}
			return "", input, errUnterminatedEscape
		}
		c, l := utf8.DecodeRuneInString(input)
		if c == '\n' {
			// a backslash-escaped newline is elided from the output entirely
		} else {
			buf.WriteString(input[:l])
		}
		input = input[l:]
	}
	goto raw

single:
	{
		i := strings.IndexRune(input, singleChar)
		if i == -1 {
			if hl {
				input = buf.String() + seqFgYellow + string(singleChar) + input
			}
			return "", input, errUnterminatedSingleQuote
		}
		// Catch up opening quote
		if hl {
			buf.WriteString(seqFgYellow)
			buf.WriteRune(singleChar)
		}

		buf.WriteString(input[0:i])
		input = input[i+1:]

		if hl {
			buf.WriteRune(singleChar)
			buf.WriteString(seqFgReset)
		}
		goto raw
	}

double:
	{
		cur := input
		for len(cur) > 0 {
			c, l := utf8.DecodeRuneInString(cur)
			cur = cur[l:]
			if c == doubleChar {
				// Catch up opening quote
				if hl {
					buf.WriteString(seqFgYellow)
					buf.WriteRune(c)
				}

				buf.WriteString(input[0 : len(input)-len(cur)-l])

				if hl {
					buf.WriteRune(c)
					buf.WriteString(seqFgReset)
				}
				input = cur
				goto raw
			} else if c == escapeChar && !hl {
				// bash only supports certain escapes in double-quoted strings
				c2, l2 := utf8.DecodeRuneInString(cur)
				cur = cur[l2:]
				if strings.ContainsRune(doubleEscapeChars, c2) {
					buf.WriteString(input[0 : len(input)-len(cur)-l-l2])
					if c2 == '\n' {
						// newline is special, skip the backslash entirely
					} else {
						buf.WriteRune(c2)
					}
					input = cur
				}
			}
		}

		if hl {
			input = buf.String() + seqFgYellow + string(doubleChar) + input
		}

		return "", input, errUnterminatedDoubleQuote
	}

done:
	return buf.String(), input, nil
}

func trimSpacesMatch(remain []string) (trimmed []string) {
	for _, word := range remain {
		trimmed = append(trimmed, strings.TrimSpace(word))
	}

	return
}
