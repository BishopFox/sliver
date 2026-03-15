package i18n

import "golang.org/x/text/language"

// MatchAvailableLocale returns the best matching locale for the given
// Accept-Language header strings. Returns the default locale if no match is found.
func (i *I18n) MatchAvailableLocale(accepts ...string) string {
	tags := make([]language.Tag, 0, max(len(accepts)*3, 4))
	for _, s := range accepts {
		parsed, _, err := language.ParseAcceptLanguage(s)
		if err != nil {
			continue
		}
		tags = append(tags, parsed...)
	}

	if len(tags) == 0 {
		return i.languages[0].String()
	}

	_, idx, conf := i.languageMatcher.Match(tags...)
	if conf > language.No {
		return i.languages[idx].String()
	}

	return i.languages[0].String()
}
