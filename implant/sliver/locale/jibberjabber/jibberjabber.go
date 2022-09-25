package jibberjabber

import (
	"errors"
//	"fmt"
//	"sort"
	"strings"
//	"sync"

//	"golang.org/x/text/language"
//	"golang.org/x/text/language/display"
)

var (
	ErrLangDetectFail          = errors.New("could not detect Language")
	ErrLangFallbackUndefined   = errors.New("no fallback language defined")
	ErrLangFallbackUnsupported = errors.New("defined fallback language is not supported")
	ErrLangUnsupported         = errors.New("language not supported")
	ErrLangParse               = errors.New("language identifier cannot be parsed")
)

func splitLocale(locale string) (string, string) {
	formattedLocale := strings.Split(locale, ".")[0]
	formattedLocale = strings.Replace(formattedLocale, "-", "_", -1)

	pieces := strings.Split(formattedLocale, "_")
	language := pieces[0]
	territory := ""
	if len(pieces) > 1 {
		territory = strings.Split(formattedLocale, "_")[1]
	}
	return language, territory
}

/**
 * languageServer
 */


// IsError checks an error you received from one of jibberjabber's funcs for a jibberjabber error like `ErrLangDetectFail`.
// Reason you cannot use e.g. `errors.Is()`: currently, golang does not allow native chain-wrapping errors. Therefore, `errors.Unwrap()`, `errors.Is()` & Co. won't return `true` for jibberjabber errors.
func IsError(err error, jjError error) bool {
	return strings.HasPrefix(err.Error(), jjError.Error())
}
