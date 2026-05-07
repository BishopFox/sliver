// Package accessibility provides accessible functions to capture user input.
package accessibility

import (
	"bufio"
	"cmp"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/x/term"
)

func atoi(s string) (int, error) {
	if strings.TrimSpace(s) == "" {
		return -1, nil
	}
	return strconv.Atoi(s) //nolint:wrapcheck
}

// PromptInt prompts a user for an integer between a certain range.
//
// Given invalid input (non-integers, integers outside of the range), the user
// will continue to be reprompted until a valid input is given, ensuring that
// the return value is always valid.
func PromptInt(
	out io.Writer,
	in io.Reader,
	prompt string,
	low, high int,
	defaultValue *int,
) int {
	var choice int

	validInt := func(s string) error {
		if strings.TrimSpace(s) == "" && defaultValue != nil {
			return nil
		}
		i, err := atoi(s)
		if err != nil || i < low || i > high {
			if low == high {
				return fmt.Errorf("Invalid: must be %d", low) //nolint:staticcheck
			}
			return fmt.Errorf("Invalid: must be a number between %d and %d", low, high) //nolint:staticcheck
		}
		return nil
	}

	input := PromptString(
		out,
		in,
		prompt,
		ptrToStr(defaultValue, strconv.Itoa),
		validInt,
	)
	choice, _ = strconv.Atoi(input)
	return choice
}

func parseBool(s string) (bool, error) {
	s = strings.ToLower(s)

	if slices.Contains([]string{"y", "yes"}, s) {
		return true, nil
	}

	// As a special case, we default to "" to no since the usage of this
	// function suggests N is the default.
	if slices.Contains([]string{"n", "no"}, s) {
		return false, nil
	}

	return false, errors.New("invalid input. please try again")
}

// PromptBool prompts a user for a boolean value.
//
// Given invalid input (non-boolean), the user will continue to be reprompted
// until a valid input is given, ensuring that the return value is always valid.
func PromptBool(
	out io.Writer,
	in io.Reader,
	prompt string,
	defaultValue bool,
) bool {
	validBool := func(s string) error {
		if strings.TrimSpace(s) == "" {
			return nil
		}
		_, err := parseBool(s)
		return err
	}

	input := PromptString(
		out, in, prompt,
		boolToStr(defaultValue),
		validBool,
	)
	b, _ := parseBool(input)
	return b
}

// PromptPassword allows to prompt for a password.
// In must be the fd of a tty.
func PromptPassword(
	out io.Writer,
	in uintptr,
	prompt string,
	validator func(input string) error,
) (string, error) {
	for {
		_, _ = fmt.Fprint(out, prompt)
		pwd, err := term.ReadPassword(in)
		if err != nil {
			return "", err //nolint:wrapcheck
		}
		_, _ = fmt.Fprintln(out)
		if err := validator(string(pwd)); err != nil {
			_, _ = fmt.Fprintln(out, err)
			continue
		}
		return string(pwd), nil
	}
}

// PromptString prompts a user for a string value and validates it against a
// validator function. It re-prompts the user until a valid input is given.
func PromptString(
	out io.Writer,
	in io.Reader,
	prompt string,
	defaultValue string,
	validator func(input string) error,
) string {
	scanner := bufio.NewScanner(in)

	var (
		valid bool
		input string
	)

	for !valid {
		_, _ = fmt.Fprint(out, prompt)
		if !scanner.Scan() {
			// no way to bubble up errors or signal cancellation
			// but the program is probably not continuing if
			// stdin sent EOF
			_, _ = fmt.Fprintln(out)
			break
		}
		input = scanner.Text()

		if err := validator(input); err != nil {
			_, _ = fmt.Fprintln(out, err)
			continue
		}

		break
	}

	return cmp.Or(strings.TrimSpace(input), defaultValue)
}

func ptrToStr[T any](t *T, fn func(t T) string) string {
	if t == nil {
		return ""
	}
	return fn(*t)
}

func boolToStr(b bool) string {
	if b {
		return "y"
	}
	return "N"
}
