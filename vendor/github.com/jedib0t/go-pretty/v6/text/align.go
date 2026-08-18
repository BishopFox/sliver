package text

import (
	"strconv"
	"strings"
)

// Align denotes how text is to be aligned horizontally.
type Align int

// Align enumerations
const (
	AlignDefault Align = iota // same as AlignLeft
	AlignLeft                 // "left        "
	AlignCenter               // "   center   "
	AlignJustify              // "justify   it"
	AlignRight                // "       right"
	AlignAuto                 // AlignRight for numbers, AlignLeft for the rest
)

// Apply aligns the text as directed. For ex.:
//   - AlignDefault.Apply("Jon Snow", 12) returns "Jon Snow    "
//   - AlignLeft.Apply("Jon Snow",    12) returns "Jon Snow    "
//   - AlignCenter.Apply("Jon Snow",  12) returns "  Jon Snow  "
//   - AlignJustify.Apply("Jon Snow", 12) returns "Jon     Snow"
//   - AlignRight.Apply("Jon Snow",   12) returns "    Jon Snow"
//   - AlignAuto.Apply("Jon Snow",    12) returns "Jon Snow    "
func (a Align) Apply(text string, maxLength int) string {
	aComputed := a
	if aComputed == AlignAuto {
		_, err := strconv.ParseFloat(text, 64)
		if err == nil { // was able to parse a number out of the string
			aComputed = AlignRight
		} else {
			aComputed = AlignLeft
		}
	}

	text = aComputed.trimString(text)
	sLenWoE := StringWidthWithoutEscSequences(text)
	padding := maxLength - sLenWoE

	// now, align the text by padding the missing display-width with spaces
	switch aComputed {
	case AlignDefault, AlignLeft:
		return padText(text, 0, padding)
	case AlignCenter:
		// extra space on the left when the padding is odd
		return padText(text, padding-padding/2, padding/2)
	case AlignJustify:
		return justifyText(text, sLenWoE, maxLength)
	}
	return padText(text, padding, 0)
}

// padText returns the text with the given number of spaces on either side;
// non-positive counts add nothing.
func padText(text string, left int, right int) string {
	if left <= 0 && right <= 0 {
		return text
	}

	var out strings.Builder
	out.Grow(len(text) + left + right)
	for idx := 0; idx < left; idx++ {
		out.WriteByte(' ')
	}
	out.WriteString(text)
	for idx := 0; idx < right; idx++ {
		out.WriteByte(' ')
	}
	return out.String()
}

// HTMLProperty returns the equivalent HTML horizontal-align tag property.
func (a Align) HTMLProperty() string {
	switch a {
	case AlignLeft:
		return "align=\"left\""
	case AlignCenter:
		return "align=\"center\""
	case AlignJustify:
		return "align=\"justify\""
	case AlignRight:
		return "align=\"right\""
	default:
		return ""
	}
}

// MarkdownProperty returns the equivalent Markdown horizontal-align separator.
// An optional minLength can be provided to extend the dashes to match the
// column content width; the result will be max(minLength, 3)+2 wide (including
// leading/trailing space or colon). Without minLength (or 0), it defaults to 3.
func (a Align) MarkdownProperty(minLength ...int) string {
	length := 3
	if len(minLength) > 0 && minLength[0] > length {
		length = minLength[0]
	}
	dashes := strings.Repeat("-", length)
	switch a {
	case AlignLeft:
		return ":" + dashes + " "
	case AlignCenter:
		return ":" + dashes + ":"
	case AlignRight:
		return " " + dashes + ":"
	default:
		return " " + dashes + " "
	}
}

func (a Align) trimString(text string) string {
	switch a {
	case AlignDefault, AlignLeft:
		if strings.HasSuffix(text, " ") {
			return strings.TrimRight(text, " ")
		}
	case AlignRight:
		if strings.HasPrefix(text, " ") {
			return strings.TrimLeft(text, " ")
		}
	default:
		if strings.HasPrefix(text, " ") || strings.HasSuffix(text, " ") {
			return strings.Trim(text, " ")
		}
	}
	return text
}

func justifyText(text string, textLength int, maxLength int) string {
	// split the text into individual words
	words := Filter(strings.Split(text, " "), func(item string) bool {
		return item != ""
	})
	// empty string implies result is just spaces for maxLength
	if len(words) == 0 {
		return strings.Repeat(" ", maxLength)
	}

	// get the number of spaces to insert into the text
	numSpacesNeeded := maxLength - textLength + strings.Count(text, " ")
	if numSpacesNeeded < 0 {
		// textLength (display-width) exceeds maxLength; this can happen
		// when the cell contains wide Unicode characters (e.g. CJK) whose
		// display width is greater than their rune count. Return the text
		// as-is; truncation is the caller's responsibility.
		return text
	}
	numSpacesNeededBetweenWords := 0
	if len(words) > 1 {
		numSpacesNeededBetweenWords = numSpacesNeeded / (len(words) - 1)
	}
	// create the output string word by word with spaces in between
	var outText strings.Builder
	outText.Grow(maxLength)
	for idx, word := range words {
		if idx > 0 {
			// insert spaces only after the first word
			if idx == len(words)-1 {
				// insert all the remaining space before the last word
				outText.WriteString(strings.Repeat(" ", numSpacesNeeded))
				numSpacesNeeded = 0
			} else {
				// insert the determined number of spaces between each word
				outText.WriteString(strings.Repeat(" ", numSpacesNeededBetweenWords))
				// and reduce the number of spaces needed after this
				numSpacesNeeded -= numSpacesNeededBetweenWords
			}
		}
		outText.WriteString(word)
		if idx == len(words)-1 && numSpacesNeeded > 0 {
			outText.WriteString(strings.Repeat(" ", numSpacesNeeded))
		}
	}
	return outText.String()
}
