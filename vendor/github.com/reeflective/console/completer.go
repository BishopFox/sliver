package console

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rsteube/carapace"
	"github.com/rsteube/carapace/pkg/style"
	completer "github.com/rsteube/carapace/pkg/x"
	"github.com/rsteube/carapace/pkg/xdg"

	"github.com/reeflective/readline"
)

func (c *Console) complete(line []rune, pos int) readline.Completions {
	menu := c.activeMenu()

	// Ensure the carapace library is called so that the function
	// completer.Complete() variable is correctly initialized before use.
	carapace.Gen(menu.Command)

	// Split the line as shell words, only using
	// what the right buffer (up to the cursor)
	args, prefixComp, prefixLine := splitArgs(line, pos)

	// Prepare arguments for the carapace completer
	// (we currently need those two dummies for avoiding a panic).
	args = append([]string{c.name, "_carapace"}, args...)

	// Call the completer with our current command context.
	completions, err := completer.Complete(menu.Command, args...)

	// The completions are never nil: fill out our own object
	// with everything it contains, regardless of errors.
	raw := make([]readline.Completion, len(completions.Values))

	for idx, val := range completions.Values.Decolor() {
		raw[idx] = readline.Completion{
			Value:       unescapeValue(prefixComp, prefixLine, val.Value),
			Display:     val.Display,
			Description: val.Description,
			Style:       val.Style,
			Tag:         val.Tag,
		}

		if !completions.Nospace.Matches(val.Value) {
			raw[idx].Value = val.Value + " "
		}
	}

	// Assign both completions and command/flags/args usage strings.
	comps := readline.CompleteRaw(raw)
	comps = comps.Usage(completions.Usage)
	comps = c.justifyCommandComps(comps)

	// If any errors arose from the completion call itself.
	if err != nil {
		comps = readline.CompleteMessage("failed to load config: " + err.Error())
	}

	// Completion status/errors
	for _, msg := range completions.Messages.Get() {
		comps = comps.Merge(readline.CompleteMessage(msg))
	}

	// Suffix matchers for the completions if any.
	suffixes, err := completions.Nospace.MarshalJSON()
	if len(suffixes) > 0 && err == nil {
		comps = comps.NoSpace([]rune(string(suffixes))...)
	}

	// If we have a quote/escape sequence unaccounted
	// for in our completions, add it to all of them.
	comps = comps.Prefix(prefixComp)
	comps.PREFIX = prefixLine

	// Finally, reset our command tree for the next call.
	completer.ClearStorage()
	menu.resetPreRun()
	menu.hideFilteredCommands(menu.Command)

	return comps
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

// splitArgs splits the line in valid words, prepares them in various ways before calling
// the completer with them, and also determines which parts of them should be used as
// prefixes, in the completions and/or in the line.
func splitArgs(line []rune, pos int) (args []string, prefixComp, prefixLine string) {
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

func adjustQuotedPrefix(remain string, err error) (arg, comp, line string) {
	arg = remain

	switch {
	case errors.Is(err, errUnterminatedDoubleQuote):
		comp = "\""
		line = comp + arg
	case errors.Is(err, errUnterminatedSingleQuote):
		comp = "'"
		line = comp + arg
	case errors.Is(err, errUnterminatedEscape):
		arg = strings.ReplaceAll(arg, "\\", "")
	}

	return arg, comp, line
}

// sanitizeArg unescapes a restrained set of characters.
func sanitizeArgs(args []string) (sanitized []string) {
	for _, arg := range args {
		arg = replacer.Replace(arg)
		sanitized = append(sanitized, arg)
	}

	return sanitized
}

// when the completer has returned us some completions, we sometimes
// needed to post-process them a little before passing them to our shell.
func unescapeValue(prefixComp, prefixLine, val string) string {
	quoted := strings.HasPrefix(prefixLine, "\"") ||
		strings.HasPrefix(prefixLine, "'")

	if quoted {
		val = strings.ReplaceAll(val, "\\ ", " ")
	}

	return val
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
