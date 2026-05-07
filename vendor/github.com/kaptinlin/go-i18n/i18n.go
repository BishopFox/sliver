package i18n

import (
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/go-json-experiment/json"
	mf "github.com/kaptinlin/messageformat-go/v1"
	"golang.org/x/text/language"
)

// Unmarshaler unmarshals the translation files, can be `json.Unmarshal` or `yaml.Unmarshal`.
type Unmarshaler func(data []byte, v any) error

// I18n is the main internationalization core.
type I18n struct {
	defaultLocale             string
	defaultLanguage           language.Tag
	languages                 []language.Tag
	unmarshaler               Unmarshaler
	languageMatcher           language.Matcher // matcher is a language.Matcher configured for all supported languages.
	fallbacks                 map[string][]string
	parsedTranslations        map[string]map[string]*parsedTranslation
	runtimeParsedTranslations map[string]*parsedTranslation
	mfOptions                 *mf.MessageFormatOptions
}

// WithUnmarshaler replaces the default translation file unmarshaler.
func WithUnmarshaler(u Unmarshaler) func(*I18n) {
	return func(bundle *I18n) {
		bundle.unmarshaler = u
	}
}

// WithFallback changes fallback settings.
func WithFallback(f map[string][]string) func(*I18n) {
	return func(bundle *I18n) {
		bundle.fallbacks = f
	}
}

func WithDefaultLocale(locale string) func(*I18n) {
	return func(bundle *I18n) {
		bundle.defaultLanguage = language.Make(locale)
		bundle.defaultLocale = bundle.defaultLanguage.String()
	}
}

func WithLocales(languages ...string) func(*I18n) {
	return func(bundle *I18n) {
		var tags []language.Tag
		for _, lang := range languages {
			tag, err := language.Parse(lang)
			if err == nil && tag != language.Und {
				tags = append(tags, tag)
			}
		}
		bundle.languages = tags
	}
}

// WithMessageFormatOptions sets MessageFormat options
func WithMessageFormatOptions(opts *mf.MessageFormatOptions) func(*I18n) {
	return func(bundle *I18n) {
		bundle.mfOptions = opts
	}
}

// WithCustomFormatters sets custom formatters for MessageFormat
func WithCustomFormatters(formatters map[string]interface{}) func(*I18n) {
	return func(bundle *I18n) {
		if bundle.mfOptions == nil {
			bundle.mfOptions = &mf.MessageFormatOptions{}
		}
		bundle.mfOptions.CustomFormatters = formatters
	}
}

// WithStrictMode sets strict parsing mode
func WithStrictMode(strict bool) func(*I18n) {
	return func(bundle *I18n) {
		if bundle.mfOptions == nil {
			bundle.mfOptions = &mf.MessageFormatOptions{}
		}
		bundle.mfOptions.Strict = strict
	}
}

// NewBundle creates a new internationalization bundle.
func NewBundle(options ...func(*I18n)) *I18n {
	// Pre-allocate with reasonable default capacities
	bundle := &I18n{
		languages:                 make([]language.Tag, 0, max(len(options), 4)), // Estimate 4 languages
		unmarshaler:               func(data []byte, v any) error { return json.Unmarshal(data, v) },
		fallbacks:                 make(map[string][]string, max(len(options), 4)),
		runtimeParsedTranslations: make(map[string]*parsedTranslation, 100), // Estimate 100 translations
		parsedTranslations:        make(map[string]map[string]*parsedTranslation, max(len(options), 4)),
	}
	for _, o := range options {
		o(bundle)
	}
	if bundle.defaultLanguage == language.Und {
		bundle.defaultLanguage = bundle.languages[0]
		bundle.defaultLocale = bundle.defaultLanguage.String()
	}
	if len(bundle.languages) > 0 && bundle.languages[0] != bundle.defaultLanguage {
		for i, t := range bundle.languages {
			if t == bundle.defaultLanguage {
				bundle.languages = append(bundle.languages[:i], bundle.languages[i+1:]...)
				break
			}
		}
		bundle.languages = append([]language.Tag{bundle.defaultLanguage}, bundle.languages...)
	} else if len(bundle.languages) == 0 {
		bundle.languages = append(bundle.languages, bundle.defaultLanguage)
	}
	bundle.languageMatcher = language.NewMatcher(bundle.languages)
	return bundle
}

func (bundle *I18n) SupportedLanguages() []language.Tag {
	return bundle.languages
}

func (bundle *I18n) getExactSupportedLocale(locale string) string {
	_, i, confidence := bundle.languageMatcher.Match(language.Make(locale))

	if confidence == language.Exact {
		return bundle.languages[i].String()
	}

	return ""
}

// IsLanguageSupported indicates whether a language can be translated.
// The check is done by the bundle's matcher and therefore languages that are not returned by
// SupportedLanguages can be supported.
func (bundle *I18n) IsLanguageSupported(lang language.Tag) bool {
	_, _, confidence := bundle.languageMatcher.Match(lang)
	return confidence > language.No
}

// NewLocalizer reads a locale from the internationalization core.
func (bundle *I18n) NewLocalizer(locales ...string) *Localizer {
	selectedLocale := bundle.defaultLocale
	for _, locale := range locales {
		locale = bundle.getExactSupportedLocale(locale)
		if locale != "" {
			if _, ok := bundle.parsedTranslations[locale]; ok {
				selectedLocale = locale
				break
			}
		}
	}

	return &Localizer{
		bundle: bundle,
		locale: selectedLocale,
	}
}

var contextRegExp = regexp.MustCompile("<(.*?)>$")

// parsedTranslation
type parsedTranslation struct {
	locale string
	name   string
	text   string
	format mf.MessageFunction
}

// trimContext
func trimContext(v string) string {
	return contextRegExp.ReplaceAllString(v, "")
}

// parseTranslation
func (bundle *I18n) parseTranslation(locale, name, text string) (*parsedTranslation, error) {
	parsedTrans := &parsedTranslation{
		name:   name,
		locale: locale,
		text:   text,
	}

	base, _ := language.MustParse(locale).Base()

	// Create new MessageFormat instance
	messageFormat, err := mf.New(base.String(), bundle.mfOptions)
	if err != nil {
		return parsedTrans, nil //nolint:nilerr // Intentionally ignore error for graceful fallback
	}

	compiled, err := messageFormat.Compile(text)
	if err != nil {
		return parsedTrans, nil //nolint:nilerr // Intentionally ignore error for graceful fallback
	}

	parsedTrans.format = compiled
	return parsedTrans, nil
}

// nameInsensitive converts `zh_CN.music.json`, `zh_CN` and `zh-TW` to `zh-CN`.
func nameInsensitive(v string) string {
	v = filepath.Base(v)

	// Use strings.Cut instead of Split for better performance (Go 1.18+)
	if before, _, found := strings.Cut(v, "."); found {
		v = before
	}

	// Use Builder to reduce memory allocations
	var result strings.Builder
	result.Grow(len(v)) // Pre-allocate capacity

	for _, r := range v {
		switch r {
		case '_':
			result.WriteByte('-')
		default:
			result.WriteRune(unicode.ToLower(r))
		}
	}

	return result.String()
}

// formatFallbacks
func (bundle *I18n) formatFallbacks() {
	for _, grandTrans := range bundle.parsedTranslations[bundle.defaultLocale] {
		for locale, trans := range bundle.parsedTranslations {
			//
			if locale == bundle.defaultLocale {
				continue
			}
			//
			if _, ok := trans[grandTrans.name]; !ok {
				if bestfit := bundle.lookupBestFallback(locale, grandTrans.name); bestfit != nil {
					bundle.parsedTranslations[locale][grandTrans.name] = bestfit
				}
			}
		}
	}
}

// lookupBestFallback
func (bundle *I18n) lookupBestFallback(locale, name string) *parsedTranslation {
	fallbacks, ok := bundle.fallbacks[locale]
	if !ok {
		if v, ok := bundle.parsedTranslations[bundle.defaultLocale][name]; ok {
			return v
		}
	}
	for _, fallback := range fallbacks {
		if v, ok := bundle.parsedTranslations[fallback][name]; ok {
			return v
		}
		if j := bundle.lookupBestFallback(fallback, name); j != nil {
			return j
		}
	}
	return nil
}
