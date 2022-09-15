// +build darwin freebsd linux netbsd openbsd

package jibberjabber

import (
	"os"
	"strings"
)

func getLangFromEnv() string {
	for _, env := range []string{"LC_MESSAGES", "LC_ALL", "LANG"} {
		locale := os.Getenv(env)
		if len(locale) > 0 {
			return locale
		}
	}
	return ""
}

func getUnixLocale() (string, error) {
	locale := getLangFromEnv()
	if len(locale) <= 0 {
		return "", ErrLangDetectFail
	}
	return locale, nil
}

// DetectIETF detects and returns the IETF language tag of UNIX systems, like Linux and macOS.
// If a territory is defined, the returned value will be in the format of `[language]-[territory]`,
// e.g. `en-GB`.
func DetectIETF() (string, error) {
	locale, err := getUnixLocale()
	if err != nil {
		return "", err
	}

	language, territory := splitLocale(locale)
	locale = language
	if len(territory) > 0 {
		locale = strings.Join([]string{language, territory}, "-")
	}

	return locale, nil
}