package style

import "strings"

var keywords = map[string]*string{
	"y": &Carapace.KeywordPositive,
	"n": &Carapace.KeywordNegative,

	"yes": &Carapace.KeywordPositive,
	"no":  &Carapace.KeywordNegative,

	"true":  &Carapace.KeywordPositive,
	"false": &Carapace.KeywordNegative,

	"on":  &Carapace.KeywordPositive,
	"off": &Carapace.KeywordNegative,

	"all":  &Carapace.KeywordPositive,
	"some": &Carapace.KeywordAmbiguous,
	"none": &Carapace.KeywordNegative,

	"full":  &Carapace.KeywordPositive,
	"empty": &Carapace.KeywordNegative,

	"loose":  &Carapace.KeywordPositive,
	"strict": &Carapace.KeywordNegative,

	"public":  &Carapace.KeywordPositive,
	"private": &Carapace.KeywordNegative,

	"internal": &Carapace.KeywordPositive,
	"external": &Carapace.KeywordNegative,

	"asc":        &Carapace.KeywordPositive,
	"ascending":  &Carapace.KeywordPositive,
	"desc":       &Carapace.KeywordNegative,
	"descending": &Carapace.KeywordNegative,

	"open":   &Carapace.KeywordPositive,
	"opened": &Carapace.KeywordPositive,
	"close":  &Carapace.KeywordNegative,
	"closed": &Carapace.KeywordNegative,

	"always": &Carapace.KeywordPositive,
	"auto":   &Carapace.KeywordAmbiguous,
	"never":  &Carapace.KeywordNegative,

	"start":      &Carapace.KeywordPositive,
	"started":    &Carapace.KeywordPositive,
	"starting":   &Carapace.KeywordPositive,
	"run":        &Carapace.KeywordPositive,
	"running":    &Carapace.KeywordPositive,
	"inprogress": &Carapace.KeywordAmbiguous,
	"pause":      &Carapace.KeywordAmbiguous,
	"paused":     &Carapace.KeywordAmbiguous,
	"pausing":    &Carapace.KeywordAmbiguous,
	"restart":    &Carapace.KeywordAmbiguous,
	"restarted":  &Carapace.KeywordAmbiguous,
	"restarting": &Carapace.KeywordAmbiguous,
	"remove":     &Carapace.KeywordNegative,
	"removed":    &Carapace.KeywordNegative,
	"removing":   &Carapace.KeywordNegative,
	"stop":       &Carapace.KeywordNegative,
	"stopped":    &Carapace.KeywordNegative,
	"stopping":   &Carapace.KeywordNegative,
	"exit":       &Carapace.KeywordNegative,
	"exited":     &Carapace.KeywordNegative,
	"exiting":    &Carapace.KeywordNegative,
	"dead":       &Carapace.KeywordNegative,

	"create":  &Carapace.KeywordPositive,
	"created": &Carapace.KeywordPositive,
	"delete":  &Carapace.KeywordNegative,
	"deleted": &Carapace.KeywordNegative,

	"onsuccess": &Carapace.KeywordPositive,
	"onfailure": &Carapace.KeywordNegative,
	"onerror":   &Carapace.KeywordNegative,

	"success": &Carapace.KeywordPositive,
	"unknown": &Carapace.KeywordUnknown,
	"backoff": &Carapace.KeywordUnknown,
	"warn":    &Carapace.KeywordAmbiguous,
	"error":   &Carapace.KeywordNegative,
	"failed":  &Carapace.KeywordNegative,
	"fatal":   &Carapace.KeywordNegative,

	"nonblock": &Carapace.KeywordAmbiguous,
	"block":    &Carapace.KeywordNegative,

	"ondemand": &Carapace.KeywordAmbiguous,
}

var keywordReplacer = strings.NewReplacer(
	"-", "",
	"_", "",
)

// ForKeyword returns the style for given keyword.
func ForKeyword(s string, _ Context) string {
	if _style, ok := keywords[keywordReplacer.Replace(strings.ToLower(s))]; ok {
		return *_style
	}
	return Default
}
