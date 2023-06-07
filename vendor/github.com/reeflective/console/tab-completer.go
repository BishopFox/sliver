package console

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/reeflective/readline"
	"github.com/rsteube/carapace"
	"github.com/rsteube/carapace/pkg/style"
	"github.com/rsteube/carapace/pkg/xdg"
)

func (c *Console) complete(line []rune, pos int) readline.Completions {
	menu := c.activeMenu()

	// Split the line as shell words, only using
	// what the right buffer (up to the cursor)
	rbuffer := line[:pos]
	args, prefix := splitArgs(rbuffer)
	args = sanitizeArgs(rbuffer, args)

	// Prepare arguments for the carapace completer
	// (we currently need those two dummies for avoiding a panic).
	args = append([]string{"examples", "_carapace"}, args...)

	// Call the completer with our current command context.
	values, meta := carapace.Complete(menu.Command, args, c.completeCommands(menu))

	// Tranfer all completion results to our readline shell completions.
	raw := make([]readline.Completion, len(values))

	for idx, val := range values {
		value := readline.Completion{
			Value:       val.Value,
			Display:     val.Display,
			Description: val.Description,
			Style:       val.Style,
			Tag:         val.Tag,
		}
		raw[idx] = value
	}

	// Assign both completions and command/flags/args usage strings.
	comps := readline.CompleteRaw(raw)
	comps = comps.Usage(meta.Usage)
	comps = c.justifyCommandComps(comps)

	// Suffix matchers for the completions if any.
	if meta.Nospace.String() != "" {
		comps = comps.NoSpace([]rune(meta.Nospace.String())...)
	}

	// If we have a quote/escape sequence unaccounted
	// for in our completions, add it to all of them.
	if prefix != "" {
		comps = comps.Prefix(prefix)
	}

	return comps
}

func splitArgs(line []rune) (args []string, prefix string) {
	// Split the line as shellwords, return them if all went fine.
	args, remain, err := splitCompWords(string(line))
	if err == nil {
		return args, remain
	}

	// If we had an error, it's because we have an unterminated quote/escape sequence.
	// In this case we split the remainder again, as the completer only ever considers
	// words as space-separated chains of characters.
	if errors.Is(err, errUnterminatedDoubleQuote) {
		remain = strings.Trim(remain, "\"")
		prefix = "\""
	} else if errors.Is(err, errUnterminatedSingleQuote) {
		remain = strings.Trim(remain, "'")
		prefix = "'"
	}

	args = append(args, strings.Split(remain, " ")...)

	return
}

func sanitizeArgs(rbuffer []rune, args []string) (sanitized []string) {
	// Like in classic system shells, we need to add an empty
	// argument if the last character is a space: the args
	// returned from the previous call don't account for it.
	if strings.HasSuffix(string(rbuffer), " ") || len(args) == 0 {
		args = append(args, "")
	} else if strings.HasSuffix(string(rbuffer), "\n") {
		args = append(args, "")
	}

	if len(args) == 0 {
		return
	}

	sanitized = args[:len(args)-1]
	last := args[len(args)-1]

	// The last word should not comprise newlines.
	last = strings.ReplaceAll(last, "\n", " ")
	last = strings.ReplaceAll(last, "\\ ", " ")
	sanitized = append(sanitized, last)

	return sanitized
}

// Regenerate commands and apply any filters.
func (c *Console) completeCommands(menu *Menu) func() {
	commands := func() {
		menu.resetCommands()
		c.hideFilteredCommands()
	}

	return commands
}

func (c *Console) justifyCommandComps(comps readline.Completions) readline.Completions {
	justified := []string{}

	comps.EachValue(func(comp readline.Completion) readline.Completion {
		if !strings.HasSuffix(comp.Tag, "commands") {
			return comp
		}

		justified = append(justified, comp.Tag)

		return comp
	})

	if len(justified) > 0 {
		return comps.JustifyDescriptions(justified...)
	}

	return comps
}

func (c *Console) defaultStyleConfig() {
	// If carapace config file is found, just return.
	if dir, err := xdg.UserConfigDir(); err == nil {
		_, err := os.Stat(fmt.Sprintf("%v/carapace/styles.json", dir))
		if err == nil {
			return
		}
	}

	// Overwrite all default styles for color
	for i := 1; i < 13; i++ {
		styleStr := fmt.Sprintf("carapace.Highlight%d", i)
		style.Set(styleStr, "bright-white")
	}

	// Overwrite all default styles for flags
	style.Set("carapace.FlagArg", "bright-white")
	style.Set("carapace.FlagMultiArg", "bright-white")
	style.Set("carapace.FlagNoArg", "bright-white")
	style.Set("carapace.FlagOptArg", "bright-white")
}

// split has been copied from go-shellquote and slightly modified so as to also
// return the remainder when the parsing failed because of an unterminated quote.
func splitCompWords(input string) (words []string, remainder string, err error) {
	var buf bytes.Buffer
	words = make([]string, 0)

	for len(input) > 0 {
		// skip any splitChars at the start
		char, read := utf8.DecodeRuneInString(input)
		if strings.ContainsRune(splitChars, char) {
			input = input[read:]
			continue
		} else if char == escapeChar {
			// Look ahead for escaped newline so we can skip over it
			next := input[read:]
			if len(next) == 0 {
				remainder = string(escapeChar)
				err = errUnterminatedEscape
				return
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
			remainder = input
			return
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
			case char == singleChar:
				buf.WriteString(input[0 : len(input)-len(cur)-read])
				input = cur
				goto single
			case char == doubleChar:
				buf.WriteString(input[0 : len(input)-len(cur)-read])
				input = cur
				goto double
			case char == escapeChar:
				buf.WriteString(input[0 : len(input)-len(cur)-read])
				buf.WriteRune(char)
				input = cur
				goto escape
			case strings.ContainsRune(splitChars, char):
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
			return "", input, errUnterminatedEscape
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
		i := strings.IndexRune(input, singleChar)
		if i == -1 {
			return "", input, errUnterminatedSingleQuote
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
			case doubleChar:
				buf.WriteString(input[0 : len(input)-len(cur)-read])
				input = cur
				goto raw
			case escapeChar:
				// bash only supports certain escapes in double-quoted strings
				char2, l2 := utf8.DecodeRuneInString(cur)
				cur = cur[l2:]
				if strings.ContainsRune(doubleEscapeChars, char2) {
					buf.WriteString(input[0 : len(input)-len(cur)-read-l2])

					if char2 != '\n' {
						buf.WriteRune(char2)
					}
					input = cur
				}
			}
		}

		return "", input, errUnterminatedDoubleQuote
	}

done:
	return buf.String(), input, nil
}
