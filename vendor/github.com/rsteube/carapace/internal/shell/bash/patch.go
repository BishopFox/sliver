package bash

import (
	"os"
	"strconv"

	shlex "github.com/rsteube/carapace-shlex"
)

// RedirectError current position is a redirect like `echo test >[TAB]`.
type RedirectError struct{}

func (r RedirectError) Error() string {
	return "current position is a redirect like `echo test >[TAB]`"
}

// TODO yuck! - set by Patch which also unsets bash comp environment variables so that they don't affect further completion
// introduces state and hides what is happening but works for now
var wordbreakPrefix string = ""
var compType = ""

const (
	COMP_TYPE_NORMAL               = "9"  // TAB, for normal completion
	COMP_TYPE_LIST_PARTIAL_WORD    = "33" // ‘!’, for listing alternatives on partial word completion,
	COMP_TYPE_MENU_COMPLETION      = "37" // ‘%’, for menu completion
	COMP_TYPE_LIST_SUCCESSIVE_TABS = "63" // ‘?’, for listing completions after successive tabs,
	COMP_TYPE_LIST_NOT_UNMODIFIED  = "64" // ‘@’, to list completions if the word is not unmodified
)

func CompLine() (string, bool) {
	line, ok := os.LookupEnv("COMP_LINE")
	if !ok {
		return "", false
	}

	point, ok := os.LookupEnv("COMP_POINT")
	if !ok {
		return "", false
	}

	pointI, err := strconv.Atoi(point)
	if err != nil || len(line) < pointI {
		return "", false
	}

	return line[:pointI], true
}

// Patch patches args if `COMP_LINE` environment variable is set.
//
// Bash passes redirects to the completion function so these need to be filtered out.
//
//	`example action >/tmp/stdout.txt --values 2>/tmp/stderr.txt fi[TAB]`
//	["example", "action", ">", "/tmp/stdout.txt", "--values", "2", ">", "/tmp/stderr.txt", "fi"]
//	["example", "action", "--values", "fi"]
func Patch(args []string) ([]string, error) { // TODO document and fix wordbreak splitting (e.g. `:`)
	compline, ok := CompLine()
	if !ok {
		return args, nil
	}

	if compline == "" {
		return args, nil
	}

	tokens, err := shlex.Split(compline)
	if err != nil {
		return nil, err
	}

	if len(tokens) > 1 {
		if previous := tokens[len(tokens)-2]; previous.WordbreakType.IsRedirect() {
			return append(args[:1], tokens[len(tokens)-1].Value), RedirectError{}
		}
	}
	args = append(args[:1], tokens.CurrentPipeline().FilterRedirects().Words().Strings()...)

	// TODO find a better solution to pass the wordbreakprefix to bash/action.go
	wordbreakPrefix = tokens.CurrentPipeline().WordbreakPrefix()
	compType = os.Getenv("COMP_TYPE")
	unsetBashCompEnv()

	return args, nil
}

func unsetBashCompEnv() {
	for _, key := range []string{
		// https://www.gnu.org/software/bash/manual/html_node/Bash-Variables.html
		"COMP_CWORD",
		"COMP_LINE",
		"COMP_POINT",
		"COMP_TYPE",
		"COMP_KEY",
		"COMP_WORDBREAKS",
		"COMP_WORDS",
	} {
		os.Unsetenv(key)
	}
}
