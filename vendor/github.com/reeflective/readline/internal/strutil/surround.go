package strutil

// MatchSurround returns the matching character of a rune that
// is either a bracket/brace/parenthesis, or a single/double quote.
func MatchSurround(r rune) (bchar, echar rune) {
	bchar = r
	echar = r

	switch bchar {
	case '{':
		echar = '}'
	case '(':
		echar = ')'
	case '[':
		echar = ']'
	case '<':
		echar = '>'
	case '}':
		bchar = '{'
		echar = '}'
	case ')':
		bchar = '('
		echar = ')'
	case ']':
		bchar = '['
		echar = ']'
	case '>':
		bchar = '<'
		echar = '>'
	case '"':
		bchar = '"'
		echar = '"'
	case '\'':
		bchar = '\''
		echar = '\''
	}

	return bchar, echar
}

// IsSurround returns true if the character is a quote or a bracket/brace, etc.
func IsSurround(bchar, echar rune) bool {
	switch bchar {
	case '{':
		return echar == '}'
	case '(':
		return echar == ')'
	case '[':
		return echar == ']'
	case '<':
		return echar == '>'
	case '"':
		return echar == '"'
	case '\'':
		return echar == '\''
	}

	return echar == bchar
}

// SurroundType says if the character is a pairing one (first boolean),
// and if the character is the closing part of the pair (second boolean).
func SurroundType(char rune) (surround, closer bool) {
	switch char {
	case '{':
		return true, false
	case '}':
		return true, true
	case '(':
		return true, false
	case ')':
		return true, true
	case '[':
		return true, false
	case ']':
		return true, true
	case '<':
		return true, false
	case '>':
	case '"':
		return true, true
	case '\'':
		return true, true
	}

	return false, false
}

// AdjustSurroundQuotes returns the correct mark and cursor positions when
// we want to know where a shell word enclosed with quotes (and potentially
// having inner ones) starts and ends.
func AdjustSurroundQuotes(dBpos, dEpos, sBpos, sEpos int) (mark, cpos int) {
	mark = -1
	cpos = -1

	if (sBpos == -1 || sEpos == -1) && (dBpos == -1 || dEpos == -1) {
		return
	}

	doubleFirstAndValid := (dBpos < sBpos && // Outtermost
		dBpos >= 0 && // Double found
		sBpos >= 0 && // compared with a found single
		dEpos > sEpos) // ensuring that we are not comparing unfound

	singleFirstAndValid := (sBpos < dBpos &&
		sBpos >= 0 &&
		dBpos >= 0 &&
		sEpos > dEpos)

	if (sBpos == -1 || sEpos == -1) || doubleFirstAndValid {
		mark = dBpos
		cpos = dEpos
	} else if (dBpos == -1 || dEpos == -1) || singleFirstAndValid {
		mark = sBpos
		cpos = sEpos
	}

	return
}

// IsBracket returns true if the character is an opening/closing bracket/brace/parenthesis.
func IsBracket(char rune) bool {
	if char == '(' ||
		char == ')' ||
		char == '{' ||
		char == '}' ||
		char == '[' ||
		char == ']' {
		return true
	}

	return false
}

// GetQuotedWordStart returns the position of the outmost containing quote
// of the word (going backward from the end of the provided line), if the
// current word is a shell word that is not closed yet.
// Ex: `this 'quote contains "surrounded" words`. the outermost quote is the single one.
func GetQuotedWordStart(line []rune) (unclosed bool, pos int) {
	var (
		single, double bool
		spos, dpos     = -1, -1
	)

	for pos, char := range line {
		switch char {
		case '\'':
			single = !single
			spos = pos
		case '"':
			double = !double
			dpos = pos
		default:
			continue
		}
	}

	if single && double {
		unclosed = true

		if spos < dpos {
			pos = spos
		} else {
			pos = dpos
		}

		return
	}

	if single {
		unclosed = true
		pos = spos
	} else if double {
		unclosed = true
		pos = dpos
	}

	return unclosed, pos
}
