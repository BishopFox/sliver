package line

import (
	"bytes"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/kballard/go-shellquote"
	"mvdan.cc/sh/v3/syntax"
)

var (
	SplitChars        = " \n\t"
	SingleChar        = '\''
	DoubleChar        = '"'
	EscapeChar        = '\\'
	DoubleEscapeChars = "$`\"\n\\"
)

var (
	ErrUnterminatedSingleQuote = errors.New("unterminated single-quoted string")
	ErrUnterminatedDoubleQuote = errors.New("unterminated double-quoted string")
	ErrUnterminatedEscape      = errors.New("unterminated backslash-escape")
)

// Parse is in charge of removing all comments from the input line
// before execution, and if successfully parsed, split into words.
func Parse(line string) (args []string, err error) {
	lineReader := strings.NewReader(line)
	parser := syntax.NewParser(syntax.KeepComments(false))

	// Parse the shell string a syntax, removing all comments.
	stmts, err := parser.Parse(lineReader, "")
	if err != nil {
		return nil, err
	}

	var parsedLine bytes.Buffer

	err = syntax.NewPrinter().Print(&parsedLine, stmts)
	if err != nil {
		return nil, err
	}

	// Split the line into shell words.
	return shellquote.Split(parsedLine.String())
}

// acceptMultiline determines if the line just accepted is complete (in which case
// we should execute it), or incomplete (in which case we must read in multiline).
func AcceptMultiline(line []rune) (accept bool) {
	// Errors are either: unterminated quotes, or unterminated escapes.
	_, _, err := Split(string(line), false)
	if err == nil {
		return true
	}

	// Currently, unterminated quotes are obvious to treat: keep reading.
	switch err {
	case ErrUnterminatedDoubleQuote, ErrUnterminatedSingleQuote:
		return false
	case ErrUnterminatedEscape:
		if len(line) > 0 && line[len(line)-1] == '\\' {
			return false
		}

		return true
	}

	return true
}

// IsEmpty checks if a given input line is empty.
// It accepts a list of characters that we consider to be irrelevant,
// that is, if the given line only contains these characters, it will
// be considered empty.
func IsEmpty(line string, emptyChars ...rune) bool {
	empty := true

	for _, r := range line {
		if !strings.ContainsRune(string(emptyChars), r) {
			empty = false
			break
		}
	}

	return empty
}

// UnescapeValue is used When the completer has returned us some completions, 
// we sometimes need to post-process them a little before passing them to our shell.
func UnescapeValue(prefixComp, prefixLine, val string) string {
	quoted := strings.HasPrefix(prefixLine, "\"") ||
		strings.HasPrefix(prefixLine, "'")

	if quoted {
		val = strings.ReplaceAll(val, "\\ ", " ")
	}

	return val
}

// TrimSpaces removes all leading/trailing spaces from words
func TrimSpaces(remain []string) (trimmed []string) {
	for _, word := range remain {
		trimmed = append(trimmed, strings.TrimSpace(word))
	}

	return
}

// Split has been copied from go-shellquote and slightly modified so as to also
// return the remainder when the parsing failed because of an unterminated quote.
func Split(input string, hl bool) (words []string, remainder string, err error) {
	var buf bytes.Buffer
	words = make([]string, 0)

	for len(input) > 0 {
		// skip any splitChars at the start
		c, l := utf8.DecodeRuneInString(input)
		if strings.ContainsRune(SplitChars, c) {
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
		} else if c == EscapeChar {
			// Look ahead for escaped newline so we can skip over it
			next := input[l:]
			if len(next) == 0 {
				if hl {
					remainder = string(EscapeChar)
				}

				err = ErrUnterminatedEscape

				return words, remainder, err
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
			return words, remainder, err
		}

		words = append(words, word)
	}

	return words, remainder, err
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
			if c == SingleChar {
				buf.WriteString(input[0 : len(input)-len(cur)-l])
				input = cur
				goto single
			} else if c == DoubleChar {
				buf.WriteString(input[0 : len(input)-len(cur)-l])
				input = cur
				goto double
			} else if c == EscapeChar {
				buf.WriteString(input[0 : len(input)-len(cur)-l])
				if hl {
					buf.WriteRune(c)
				}
				input = cur
				goto escape
			} else if strings.ContainsRune(SplitChars, c) {
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
			return "", input, ErrUnterminatedEscape
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
		i := strings.IndexRune(input, SingleChar)
		if i == -1 {
			if hl {
				input = buf.String() + YellowFG + string(SingleChar) + input
			}
			return "", input, ErrUnterminatedSingleQuote
		}
		// Catch up opening quote
		if hl {
			buf.WriteString(YellowFG)
			buf.WriteRune(SingleChar)
		}

		buf.WriteString(input[0:i])
		input = input[i+1:]

		if hl {
			buf.WriteRune(SingleChar)
			buf.WriteString(ResetFG)
		}
		goto raw
	}

double:
	{
		cur := input
		for len(cur) > 0 {
			c, l := utf8.DecodeRuneInString(cur)
			cur = cur[l:]
			if c == DoubleChar {
				// Catch up opening quote
				if hl {
					buf.WriteString(YellowFG)
					buf.WriteRune(c)
				}

				buf.WriteString(input[0 : len(input)-len(cur)-l])

				if hl {
					buf.WriteRune(c)
					buf.WriteString(ResetFG)
				}
				input = cur
				goto raw
			} else if c == EscapeChar && !hl {
				// bash only supports certain escapes in double-quoted strings
				c2, l2 := utf8.DecodeRuneInString(cur)
				cur = cur[l2:]
				if strings.ContainsRune(DoubleEscapeChars, c2) {
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
			input = buf.String() + YellowFG + string(DoubleChar) + input
		}

		return "", input, ErrUnterminatedDoubleQuote
	}

done:
	return buf.String(), input, nil
}

