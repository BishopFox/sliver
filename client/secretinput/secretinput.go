package secretinput

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

	Three-tier secret resolution for CLI flags that accept sensitive
	values (HMAC secrets, encryption keys, etc.).

	Resolution order:
	  1. --flag-env ENVVAR   – read value from the named env var (no
	     secret in argv; preferred).
	  2. --flag VALUE        – literal value on the command line. A
	     stderr warning is emitted because the value is visible in
	     `ps` output.
	  3. stdin prompt        – if neither of the above provides a
	     value AND stdin is a TTY, prompt with masked input
	     (term.ReadPassword). If stdin is a pipe, read a single
	     line (whitespace preserved).

	Callers get a single Resolve() call that returns the secret bytes
	or a user-facing error.
*/

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// Resolve implements the three-tier secret resolution.
//
// Parameters:
//   - envFlagVal: the value of the --*-env flag (env var NAME, not the secret itself).
//     Empty string means the flag was not provided.
//   - directFlagVal: the value of the direct --*-secret flag (literal secret).
//     Empty string means the flag was not provided.
//   - directFlagChanged: whether the direct flag was explicitly set by the user
//     (distinguishes "" from "not provided").
//   - flagLabel: human-readable label for error messages (e.g. "--trigger-wake-secret").
//   - promptLabel: label shown in the stdin prompt (e.g. "trigger-wake HMAC secret").
//   - warnFn: function to emit warnings (typically con.PrintWarnf or fmt.Fprintf(os.Stderr, ...)).
//
// Returns the resolved secret bytes or an error.
func Resolve(
	envFlagVal string,
	directFlagVal string,
	directFlagChanged bool,
	flagLabel string,
	promptLabel string,
	warnFn func(format string, args ...any),
) ([]byte, error) {
	// Tier 1: env var indirection
	envFlagVal = strings.TrimSpace(envFlagVal)
	if envFlagVal != "" {
		secret, ok := os.LookupEnv(envFlagVal)
		if !ok {
			return nil, fmt.Errorf(
				"%s-env: environment variable %q is not set on this host\n"+
					"  hint: export %s=<your-secret> before running this command,\n"+
					"        or use %s <value> to pass the secret directly (visible in ps)",
				flagLabel, envFlagVal, envFlagVal, flagLabel,
			)
		}
		if strings.TrimSpace(secret) == "" {
			return nil, fmt.Errorf("%s-env: environment variable %q is set but empty", flagLabel, envFlagVal)
		}
		return []byte(secret), nil
	}

	// Tier 2: direct CLI value
	if directFlagChanged && directFlagVal != "" {
		if warnFn != nil {
			warnFn("WARNING: %s value is visible in `ps` output; prefer %s-env for production use\n", flagLabel, flagLabel)
		}
		return []byte(directFlagVal), nil
	}
	if directFlagChanged && directFlagVal == "" {
		return nil, fmt.Errorf("%s: flag was set but value is empty", flagLabel)
	}

	// Tier 3: stdin prompt
	return readFromStdin(promptLabel)
}

// ValidateForTemplate checks that a resolved secret does not contain
// characters that would break Go template compilation when the secret
// is embedded in the implant source.  Specifically it rejects:
//   - Backtick characters (used for raw string literals in Go)
//   - Go template directives ({{ and }})
//
// This mirrors the guard in server/configs/http-c2.go for UserAgent.
func ValidateForTemplate(secret []byte) error {
	s := string(secret)
	if strings.Contains(s, "`") {
		return fmt.Errorf("secret contains characters unsafe for implant template embedding (backtick or template directive); use a different secret")
	}
	if strings.Contains(s, "{{") || strings.Contains(s, "}}") {
		return fmt.Errorf("secret contains characters unsafe for implant template embedding (backtick or template directive); use a different secret")
	}
	return nil
}

// readFromStdin reads a secret from stdin. If stdin is a TTY, it uses
// term.ReadPassword for masked/no-echo input. If stdin is a pipe, it
// reads a single line (preserving leading/trailing whitespace).
func readFromStdin(promptLabel string) ([]byte, error) {
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		fmt.Fprintf(os.Stderr, "Enter %s: ", promptLabel)
		secret, err := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr) // newline after masked input
		if err != nil {
			return nil, fmt.Errorf("failed to read secret from terminal: %w", err)
		}
		if strings.TrimSpace(string(secret)) == "" {
			return nil, fmt.Errorf("secret input was empty (stdin prompt for %s)", promptLabel)
		}
		return secret, nil
	}

	// Piped stdin: read single line (preserve whitespace)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			return nil, fmt.Errorf("secret input was empty (piped stdin for %s)", promptLabel)
		}
		return []byte(line), nil
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read secret from piped stdin: %w", err)
	}
	return nil, fmt.Errorf("no input received on stdin for %s", promptLabel)
}
