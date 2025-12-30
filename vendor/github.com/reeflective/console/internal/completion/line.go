package completion

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/reeflective/console/internal/line"
)

// when the completer has returned us some completions, we sometimes
// needed to post-process them a little before passing them to our shell.
func UnescapeValue(prefixComp, prefixLine, val string) string {
	quoted := strings.HasPrefix(prefixLine, "\"") ||
		strings.HasPrefix(prefixLine, "'")

	if quoted {
		val = strings.ReplaceAll(val, "\\ ", " ")
	}

	return val
}

// SplitArgs splits the line in valid words, prepares them in various ways before calling
// the completer with them, and also determines which parts of them should be used as
// prefixes, in the completions and/or in the line.
func SplitArgs(line []rune, pos int) (args []string, prefixComp, prefixLine string) {
	line = line[:pos]

	// Remove all colors from the string
	line = []rune(strip(string(line)))

	// Split the line as shellwords, return them if all went fine.
	args, remain, err := splitCompWords(string(line))

	// We might have either no error and args, or no error and
	// the cursor ready to complete a new word (last character
	// in line is a space).
	// In some of those cases we append a single dummy argument
	// for the completer to understand we want a new word comp.
	mustComplete, args, remain := mustComplete(line, args, remain, err)
	if mustComplete {
		return sanitizeArgs(args), "", remain
	}

	// But the completion candidates themselves might need slightly
	// different prefixes, for an optimal completion experience.
	arg, prefixComp, prefixLine := adjustQuotedPrefix(remain, err)

	// The remainder is everything following the open charater.
	// Pass it as is to the carapace completion engine.
	args = append(args, arg)

	return sanitizeArgs(args), prefixComp, prefixLine
}

func mustComplete(line []rune, args []string, remain string, err error) (bool, []string, string) {
	dummyArg := ""

	// Empty command line, complete the root command.
	if len(args) == 0 || len(line) == 0 {
		return true, append(args, dummyArg), remain
	}

	// If we have an error, we must handle it later.
	if err != nil {
		return false, args, remain
	}

	lastChar := line[len(line)-1]

	// No remain and a trailing space means we want to complete
	// for the next word, except when this last space was escaped.
	if remain == "" && unicode.IsSpace(lastChar) {
		if strings.HasSuffix(string(line), "\\ ") {
			return true, args, args[len(args)-1]
		}

		return true, append(args, dummyArg), remain
	}

	// Else there is a character under the cursor, which means we are
	// in the middle/at the end of a posentially completed word.
	return true, args, remain
}

func adjustQuotedPrefix(remain string, err error) (arg, comp, input string) {
	arg = remain

	switch {
	case errors.Is(err, line.ErrUnterminatedDoubleQuote):
		comp = "\""
		input = comp + arg
	case errors.Is(err, line.ErrUnterminatedSingleQuote):
		comp = "'"
		input = comp + arg
	case errors.Is(err, line.ErrUnterminatedEscape):
		arg = strings.ReplaceAll(arg, "\\", "")
	}

	return arg, comp, input 
}

// sanitizeArg unescapes a restrained set of characters.
func sanitizeArgs(args []string) (sanitized []string) {
	for _, arg := range args {
		arg = replacer.Replace(arg)
		sanitized = append(sanitized, arg)
	}

	return sanitized
}

// split has been copied from go-shellquote and slightly modified so as to also
// return the remainder when the parsing failed because of an unterminated quote.
func splitCompWords(input string) (words []string, remainder string, err error) {
	var buf bytes.Buffer
	words = make([]string, 0)

	for len(input) > 0 {
		// skip any splitChars at the start
		char, read := utf8.DecodeRuneInString(input)
		if strings.ContainsRune(line.SplitChars, char) {
			input = input[read:]
			continue
		} else if char == line.EscapeChar {
			// Look ahead for escaped newline so we can skip over it
			next := input[read:]
			if len(next) == 0 {
				remainder = string(line.EscapeChar)
				err = line.ErrUnterminatedEscape

				return words, remainder, err
			}

			c2, l2 := utf8.DecodeRuneInString(next)
			if c2 == '\n' {
				input = next[l2:]
				continue
			}
		}

		var word string

		word, input, err = splitCompWord(input, &buf)
		if err != nil {
			return words, word + input, err
		}

		words = append(words, word)
	}

	return words, remainder, nil
}

// splitWord has been modified to return the remainder of the input (the part that has not been
// added to the buffer) even when an error is returned.
func splitCompWord(input string, buf *bytes.Buffer) (word string, remainder string, err error) {
	buf.Reset()

raw:
	{
		cur := input
		for len(cur) > 0 {
			char, read := utf8.DecodeRuneInString(cur)
			cur = cur[read:]
			switch {
			case char == line.SingleChar:
				buf.WriteString(input[0 : len(input)-len(cur)-read])
				input = cur
				goto single
			case char == line.DoubleChar:
				buf.WriteString(input[0 : len(input)-len(cur)-read])
				input = cur
				goto double
			case char == line.EscapeChar:
				buf.WriteString(input[0 : len(input)-len(cur)-read])
				buf.WriteRune(char)
				input = cur
				goto escape
			case strings.ContainsRune(line.SplitChars, char):
				buf.WriteString(input[0 : len(input)-len(cur)-read])
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
			input = buf.String() + input
			return "", input, line.ErrUnterminatedEscape
		}
		c, l := utf8.DecodeRuneInString(input)
		if c != '\n' {
			buf.WriteString(input[:l])
		}
		input = input[l:]
	}

	goto raw

single:
	{
		i := strings.IndexRune(input, line.SingleChar)
		if i == -1 {
			return "", input, line.ErrUnterminatedSingleQuote
		}
		buf.WriteString(input[0:i])
		input = input[i+1:]
		goto raw
	}

double:
	{
		cur := input
		for len(cur) > 0 {
			c, read := utf8.DecodeRuneInString(cur)
			cur = cur[read:]
			switch c {
			case line.DoubleChar:
				buf.WriteString(input[0 : len(input)-len(cur)-read])
				input = cur
				goto raw
			case line.EscapeChar:
				// bash only supports certain escapes in double-quoted strings
				char2, l2 := utf8.DecodeRuneInString(cur)
				cur = cur[l2:]
				if strings.ContainsRune(line.DoubleEscapeChars, char2) {
					buf.WriteString(input[0 : len(input)-len(cur)-read-l2])

					if char2 != '\n' {
						buf.WriteRune(char2)
					}
					input = cur
				}
			}
		}

		return "", input, line.ErrUnterminatedDoubleQuote
	}

done:
	return buf.String(), input, nil
}

const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

var re = regexp.MustCompile(ansi)

// strip removes all ANSI escaped color sequences in a string.
func strip(str string) string {
	return re.ReplaceAllString(str, "")
}

var replacer = strings.NewReplacer(
	"\n", ` `,
	"\t", ` `,
	"\\ ", " ", // User-escaped spaces in words.
)
