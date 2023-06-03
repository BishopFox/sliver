package strutil

import "github.com/reeflective/readline/inputrc"

// ConvertMeta recursively searches for metafied keys in a sequence,
// and replaces them with an esc prefix and their unmeta equivalent.
func ConvertMeta(keys []rune) string {
	if len(keys) == 0 {
		return string(keys)
	}

	converted := make([]rune, 0)

	for i := 0; i < len(keys); i++ {
		char := keys[i]

		if !inputrc.IsMeta(char) {
			converted = append(converted, char)
			continue
		}

		// Replace the key with esc prefix and add the demetafied key.
		converted = append(converted, inputrc.Esc)
		converted = append(converted, inputrc.Demeta(char))
	}

	return string(converted)
}

// Quote translates one rune in its printable version,
// which might be different for Control/Meta characters.
// Returns the "translated" string and new length. (eg 0x04 => ^C = len:2).
func Quote(char rune) (res []rune, length int) {
	var inserted []rune

	// Special cases for keys that should not be quoted
	if char == inputrc.Tab {
		inserted = append(inserted, char)
		return inserted, len(inserted)
	}

	switch {
	case inputrc.IsMeta(char):
		inserted = append(inserted, '^', '[')
		inserted = append(inserted, inputrc.Demeta(char))
	case inputrc.IsControl(char):
		inserted = append(inserted, '^')
		inserted = append(inserted, inputrc.Decontrol(char))
	default:
		inserted = []rune(inputrc.Unescape(string(char)))
	}

	return inserted, len(inserted)
}
