package sqlite

import (
	"errors"
	"fmt"
)

type parseAllColumnsState int

const (
	parseAllColumnsState_NONE parseAllColumnsState = iota
	parseAllColumnsState_Beginning
	parseAllColumnsState_ReadingRawName
	parseAllColumnsState_ReadingQuotedName
	parseAllColumnsState_EndOfName
	parseAllColumnsState_State_End
)

func parseAllColumns(in string) ([]string, error) {
	s := []rune(in)
	columns := make([]string, 0)
	state := parseAllColumnsState_NONE
	quote := rune(0)
	name := make([]rune, 0)
	for i := 0; i < len(s); i++ {
		switch state {
		case parseAllColumnsState_NONE:
			if s[i] == '(' {
				state = parseAllColumnsState_Beginning
			}
		case parseAllColumnsState_Beginning:
			if isSpace(s[i]) {
				continue
			}
			if isQuote(s[i]) {
				state = parseAllColumnsState_ReadingQuotedName
				quote = s[i]
				continue
			}
			if s[i] == '[' {
				state = parseAllColumnsState_ReadingQuotedName
				quote = ']'
				continue
			} else if s[i] == ')' {
				return columns, fmt.Errorf("unexpected token: %s", string(s[i]))
			}
			state = parseAllColumnsState_ReadingRawName
			name = append(name, s[i])
		case parseAllColumnsState_ReadingRawName:
			if isSeparator(s[i]) {
				state = parseAllColumnsState_Beginning
				columns = append(columns, string(name))
				name = make([]rune, 0)
				continue
			}
			if s[i] == ')' {
				state = parseAllColumnsState_State_End
				columns = append(columns, string(name))
			}
			if isQuote(s[i]) {
				return nil, fmt.Errorf("unexpected token: %s", string(s[i]))
			}
			if isSpace(s[i]) {
				state = parseAllColumnsState_EndOfName
				columns = append(columns, string(name))
				name = make([]rune, 0)
				continue
			}
			name = append(name, s[i])
		case parseAllColumnsState_ReadingQuotedName:
			if s[i] == quote {
				// check if quote character is escaped
				if i+1 < len(s) && s[i+1] == quote {
					name = append(name, quote)
					i++
					continue
				}
				state = parseAllColumnsState_EndOfName
				columns = append(columns, string(name))
				name = make([]rune, 0)
				continue
			}
			name = append(name, s[i])
		case parseAllColumnsState_EndOfName:
			if isSpace(s[i]) {
				continue
			}
			if isSeparator(s[i]) {
				state = parseAllColumnsState_Beginning
				continue
			}
			if s[i] == ')' {
				state = parseAllColumnsState_State_End
				continue
			}
			return nil, fmt.Errorf("unexpected token: %s", string(s[i]))
		case parseAllColumnsState_State_End:
			break
		}
	}
	if state != parseAllColumnsState_State_End {
		return nil, errors.New("unexpected end")
	}
	return columns, nil
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

func isQuote(r rune) bool {
	return r == '`' || r == '"' || r == '\''
}

func isSeparator(r rune) bool {
	return r == ','
}
