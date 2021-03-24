package readline

import "strings"

// WrapText - Wraps a text given a specified width, and returns the formatted
// string as well the number of lines it will occupy
func WrapText(text string, lineWidth int) (wrapped string, lines int) {
	words := strings.Fields(text)
	if len(words) == 0 {
		return
	}
	wrapped = words[0]
	spaceLeft := lineWidth - len(wrapped)
	// There must be at least a line
	if text != "" {
		lines++
	}
	for _, word := range words[1:] {
		if len(word)+1 > spaceLeft {
			lines++
			wrapped += "\n" + word
			spaceLeft = lineWidth - len(word)
		} else {
			wrapped += " " + word
			spaceLeft -= 1 + len(word)
		}
	}
	return
}
