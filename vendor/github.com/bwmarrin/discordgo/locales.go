package discordgo

// Locale represents the accepted languages for Discord.
// https://discord.com/developers/docs/reference#locales
type Locale string

// String returns the human-readable string of the locale
func (l Locale) String() string {
	if name, ok := Locales[l]; ok {
		return name
	}
	return Unknown.String()
}

// All defined locales in Discord
const (
	EnglishUS    Locale = "en-US"
	EnglishGB    Locale = "en-GB"
	Bulgarian    Locale = "bg"
	ChineseCN    Locale = "zh-CN"
	ChineseTW    Locale = "zh-TW"
	Croatian     Locale = "hr"
	Czech        Locale = "cs"
	Danish       Locale = "da"
	Dutch        Locale = "nl"
	Finnish      Locale = "fi"
	French       Locale = "fr"
	German       Locale = "de"
	Greek        Locale = "el"
	Hindi        Locale = "hi"
	Hungarian    Locale = "hu"
	Italian      Locale = "it"
	Japanese     Locale = "ja"
	Korean       Locale = "ko"
	Lithuanian   Locale = "lt"
	Norwegian    Locale = "no"
	Polish       Locale = "pl"
	PortugueseBR Locale = "pt-BR"
	Romanian     Locale = "ro"
	Russian      Locale = "ru"
	SpanishES    Locale = "es-ES"
	SpanishLATAM Locale = "es-419"
	Swedish      Locale = "sv-SE"
	Thai         Locale = "th"
	Turkish      Locale = "tr"
	Ukrainian    Locale = "uk"
	Vietnamese   Locale = "vi"
	Unknown      Locale = ""
)

// Locales is a map of all the languages codes to their names.
var Locales = map[Locale]string{
	EnglishUS:    "English (United States)",
	EnglishGB:    "English (Great Britain)",
	Bulgarian:    "Bulgarian",
	ChineseCN:    "Chinese (China)",
	ChineseTW:    "Chinese (Taiwan)",
	Croatian:     "Croatian",
	Czech:        "Czech",
	Danish:       "Danish",
	Dutch:        "Dutch",
	Finnish:      "Finnish",
	French:       "French",
	German:       "German",
	Greek:        "Greek",
	Hindi:        "Hindi",
	Hungarian:    "Hungarian",
	Italian:      "Italian",
	Japanese:     "Japanese",
	Korean:       "Korean",
	Lithuanian:   "Lithuanian",
	Norwegian:    "Norwegian",
	Polish:       "Polish",
	PortugueseBR: "Portuguese (Brazil)",
	Romanian:     "Romanian",
	Russian:      "Russian",
	SpanishES:    "Spanish (Spain)",
	SpanishLATAM: "Spanish (LATAM)",
	Swedish:      "Swedish",
	Thai:         "Thai",
	Turkish:      "Turkish",
	Ukrainian:    "Ukrainian",
	Vietnamese:   "Vietnamese",
	Unknown:      "unknown",
}
