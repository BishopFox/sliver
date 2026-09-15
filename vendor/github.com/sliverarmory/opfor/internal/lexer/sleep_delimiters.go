package lexer

import "unicode/utf8"

const (
	diagnosticMismatchedParenthesis = "LEX006"
	diagnosticMismatchedBrace       = "LEX007"
	diagnosticMismatchedIndex       = "LEX008"
)

// Sleep's reference lexer audits delimiter rules in a fixed order after its
// recursive term scan. Its Rule implementation pairs witnesses from the same
// line immediately and otherwise reconciles counts at end of input. Repeating
// that behavior matters for diagnostics such as an unmatched close index
// followed by a balanced index on the next line: the latter is the location
// selected by the reference implementation.
func sleepStructuralDiagnostics(source Source) []Diagnostic {
	rules := []*sleepDelimiterRule{
		newSleepPairRule(
			diagnosticMismatchedParenthesis,
			"Mismatched Parentheses - missing open paren",
			"Mismatched Parentheses - missing close paren",
			'(',
			')',
		),
		newSleepPairRule(
			diagnosticMismatchedBrace,
			"Mismatched Braces - missing open brace",
			"Mismatched Braces - missing close brace",
			'{',
			'}',
		),
		newSleepQuoteRule('"'),
		newSleepQuoteRule('\''),
		newSleepPairRule(
			diagnosticMismatchedIndex,
			"Mismatched Indices - missing open index",
			"Mismatched Indices - missing close index",
			'[',
			']',
		),
		newSleepQuoteRule('`'),
	}

	cursor := sleepDelimiterCursor{data: source.Data, line: 1, column: 1}
	scanSleepDelimiterTerms(&cursor, rules, nil)

	diagnostics := make([]Diagnostic, 0, len(rules))
	for _, rule := range rules {
		diagnostic, ok := rule.diagnostic(sourceName(source))
		if ok {
			diagnostics = append(diagnostics, diagnostic)
		}
	}
	return diagnostics
}

type sleepDelimiterRule struct {
	code         string
	missingOpen  string
	missingClose string
	open         rune
	close        rune
	quote        rune
	opens        []Span
	closes       []Span
}

func newSleepPairRule(code, missingOpen, missingClose string, open, close rune) *sleepDelimiterRule {
	return &sleepDelimiterRule{
		code:         code,
		missingOpen:  missingOpen,
		missingClose: missingClose,
		open:         open,
		close:        close,
	}
}

func newSleepQuoteRule(quote rune) *sleepDelimiterRule {
	return &sleepDelimiterRule{
		code:         diagnosticUnterminatedQuote,
		missingOpen:  "Runaway string",
		missingClose: "Runaway string",
		quote:        quote,
	}
}

func (r *sleepDelimiterRule) isQuote() bool { return r.quote != 0 }

func (r *sleepDelimiterRule) starts(value rune) bool {
	if r.isQuote() {
		return value == r.quote
	}
	return value == r.open
}

func (r *sleepDelimiterRule) ends(value rune) bool {
	if r.isQuote() {
		return value == r.quote
	}
	return value == r.close
}

func (r *sleepDelimiterRule) witnessOpen(span Span) {
	r.opens = append(r.opens, span)
	r.adjustSameLineWitnesses()
}

func (r *sleepDelimiterRule) witnessClose(span Span) {
	if r.isQuote() {
		r.closes = append(r.closes, span)
	} else {
		r.closes = append([]Span{span}, r.closes...)
	}
	r.adjustSameLineWitnesses()
}

func (r *sleepDelimiterRule) adjustSameLineWitnesses() {
	if len(r.opens) == 0 || len(r.closes) == 0 {
		return
	}
	if r.opens[len(r.opens)-1].Start.Line != r.closes[len(r.closes)-1].Start.Line {
		return
	}
	r.opens = r.opens[:len(r.opens)-1]
	r.closes = r.closes[:len(r.closes)-1]
}

func (r *sleepDelimiterRule) diagnostic(sourceName string) (Diagnostic, bool) {
	if len(r.opens) == len(r.closes) {
		return Diagnostic{}, false
	}

	opens := append([]Span(nil), r.opens...)
	closes := append([]Span(nil), r.closes...)
	for len(opens) != 0 && len(closes) != 0 {
		opens = opens[:len(opens)-1]
		closes = closes[:len(closes)-1]
	}

	message := r.missingOpen
	var span Span
	if len(opens) != 0 {
		message = r.missingClose
		span = opens[0]
	} else {
		span = closes[0]
	}
	span.Source = sourceName
	return Diagnostic{
		Severity: SeverityError,
		Code:     r.code,
		Message:  message,
		Span:     span,
	}, true
}

type sleepDelimiterCursor struct {
	data   []byte
	offset int
	line   int
	column int
}

func (c *sleepDelimiterCursor) atEnd() bool { return c.offset >= len(c.data) }

func (c *sleepDelimiterCursor) position() Position {
	return Position{Offset: c.offset, Line: c.line, Column: c.column}
}

func (c *sleepDelimiterCursor) next() (rune, Span) {
	start := c.position()
	r, size := utf8.DecodeRune(c.data[c.offset:])
	c.offset += size
	if r == '\r' {
		if c.offset < len(c.data) && c.data[c.offset] == '\n' {
			c.offset++
			c.line++
			c.column = 1
			r = '\n'
		} else {
			c.column++
		}
	} else if r == '\n' {
		c.line++
		c.column = 1
	} else {
		c.column++
	}
	return r, Span{Start: start, End: c.position()}
}

func scanSleepDelimiterTerms(cursor *sleepDelimiterCursor, rules []*sleepDelimiterRule, active *sleepDelimiterRule) bool {
	for !cursor.atEnd() {
		value, span := cursor.next()

		if active != nil && active.isQuote() {
			if value == '\\' && !cursor.atEnd() {
				cursor.next()
				continue
			}
			if active.ends(value) {
				active.witnessClose(span)
				return true
			}
			continue
		}

		if active != nil && active.ends(value) {
			active.witnessClose(span)
			return true
		}

		// Comments are a preservation rule in the reference scanner. Delimiters
		// and quotes inside them do not participate in balancing.
		if value == '#' {
			for !cursor.atEnd() {
				commentValue, _ := cursor.next()
				if commentValue == '\n' {
					break
				}
			}
			continue
		}

		matched := false
		for _, rule := range rules {
			if !rule.starts(value) {
				continue
			}
			rule.witnessOpen(span)
			if !scanSleepDelimiterTerms(cursor, rules, rule) {
				return false
			}
			matched = true
			break
		}
		if matched {
			continue
		}

		for _, rule := range rules {
			if !rule.isQuote() && rule != active && rule.ends(value) {
				rule.witnessClose(span)
				break
			}
		}
	}
	return false
}
