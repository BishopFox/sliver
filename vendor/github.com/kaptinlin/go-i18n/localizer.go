package i18n

import (
	"fmt"

	mf "github.com/kaptinlin/messageformat-go/v1"
	"golang.org/x/text/language"
)

// Localizer provides translation methods for a specific locale. Create one
// via [I18n.NewLocalizer].
type Localizer struct {
	bundle *I18n
	locale string
}

// Locale returns the resolved locale name for this localizer.
func (l *Localizer) Locale() string {
	return l.locale
}

// Get returns the translation for name with optional MessageFormat variables.
// Returns name as fallback if no translation is found.
func (l *Localizer) Get(name string, data ...Vars) string {
	pt, err := l.lookup(name)
	if err != nil {
		return name
	}
	return l.localize(pt, data...)
}

// GetX returns the translation for name disambiguated by context.
// The context is appended as " <context>" to form the lookup key.
// For example, GetX("Post", "verb") looks up "Post <verb>".
func (l *Localizer) GetX(name, context string, data ...Vars) string {
	return l.Get(name+" <"+context+">", data...)
}

// Getf returns the translation for name formatted with fmt.Sprintf.
// Uses name as the format string if no translation is found.
func (l *Localizer) Getf(name string, args ...any) string {
	pt, err := l.lookup(name)
	if err != nil {
		return name
	}
	return fmt.Sprintf(l.localize(pt), args...)
}

// lookup resolves the translation for name by checking the locale's
// pre-parsed translations first, then falling back to runtime-parsed
// translations from the default locale. If no translation exists, it
// creates a new runtime translation using the name as the text.
func (l *Localizer) lookup(name string) (*parsedTranslation, error) {
	if pt, ok := l.bundle.parsedTranslations[l.locale][name]; ok {
		return pt, nil
	}
	if pt, ok := l.bundle.runtimeParsedTranslations[name]; ok {
		return pt, nil
	}
	pt, err := l.bundle.parseTranslation(l.bundle.defaultLocale, name, trimContext(name))
	if err != nil {
		return nil, err
	}
	l.bundle.runtimeParsedTranslations[name] = pt
	return pt, nil
}

// localize formats a parsed translation with the given variables.
// Without variables the raw text is returned. With variables and a
// compiled MessageFormat function, the formatted result is returned.
func (l *Localizer) localize(pt *parsedTranslation, data ...Vars) string {
	if pt.format == nil {
		return pt.text
	}
	params := varsToParams(data)
	if params == nil {
		return pt.text
	}
	result, err := pt.format(params)
	if err != nil {
		return pt.text
	}
	str, ok := result.(string)
	if !ok {
		return pt.text
	}
	return str
}

// Format compiles and formats a MessageFormat message directly.
// This bypasses translation lookup and is useful for dynamic messages
// not stored in translation files.
func (l *Localizer) Format(message string, data ...Vars) (string, error) {
	base, _ := language.MustParse(l.locale).Base()

	formatter, err := mf.New(base.String(), l.bundle.mfOptions)
	if err != nil {
		return "", fmt.Errorf("create formatter: %w", err)
	}

	compiled, err := formatter.Compile(message)
	if err != nil {
		return "", fmt.Errorf("compile message: %w", err)
	}

	params := varsToParams(data)

	result, err := compiled(params)
	if err != nil {
		return "", fmt.Errorf("format message: %w", err)
	}

	str, ok := result.(string)
	if !ok {
		return fmt.Sprintf("%v", result), nil
	}
	return str, nil
}

// varsToParams converts optional Vars arguments to a params value
// suitable for a compiled MessageFormat function. Returns nil when
// no variables are provided. Only the first Vars argument is used.
func varsToParams(data []Vars) any {
	if len(data) == 0 {
		return nil
	}
	return map[string]any(data[0])
}
