package readline

import "strings"

func tokeniseLine(line []rune, linePos int) ([]string, int, int) {
	if len(line) == 0 {
		return nil, 0, 0
	}

	var index, pos int
	var punc bool

	split := make([]string, 1)

	for i, r := range line {
		switch {
		case (r >= 33 && 47 >= r) ||
			(r >= 58 && 64 >= r) ||
			(r >= 91 && 94 >= r) ||
			r == 96 ||
			(r >= 123 && 126 >= r):

			if i > 0 && line[i-1] != r {
				split = append(split, "")
			}
			split[len(split)-1] += string(r)
			punc = true

		case r == ' ' || r == '\t':
			split[len(split)-1] += string(r)
			punc = true

		default:
			if punc {
				split = append(split, "")
			}
			split[len(split)-1] += string(r)
			punc = false
		}

		if i == linePos {
			index = len(split) - 1
			pos = len(split[index]) - 1
		}
	}

	return split, index, pos
}

func tokeniseSplitSpaces(line []rune, linePos int) ([]string, int, int) {
	if len(line) == 0 {
		return nil, 0, 0
	}

	var index, pos int
	split := make([]string, 1)

	for i, r := range line {
		switch {
		case r == ' ' || r == '\t':
			split[len(split)-1] += string(r)

		default:
			if i > 0 && (line[i-1] == ' ' || line[i-1] == '\t') {
				split = append(split, "")
			}
			split[len(split)-1] += string(r)
		}

		if i == linePos {
			index = len(split) - 1
			pos = len(split[index]) - 1
		}
	}

	return split, index, pos
}

func tokeniseBrackets(line []rune, linePos int) ([]string, int, int) {
	var (
		open, close    rune
		split          []string
		count          int
		pos            = make(map[int]int)
		match          int
		single, double bool
	)

	switch line[linePos] {
	case '(', ')':
		open = '('
		close = ')'

	case '{', '[':
		open = line[linePos]
		close = line[linePos] + 2

	case '}', ']':
		open = line[linePos] - 2
		close = line[linePos]

	default:
		return nil, 0, 0
	}

	for i := range line {
		switch line[i] {
		case '\'':
			if !single {
				double = !double
			}

		case '"':
			if !double {
				single = !single
			}

		case open:
			if !single && !double {
				count++
				pos[count] = i
				if i == linePos {
					match = count
					split = []string{string(line[:i-1])}
				}

			} else if i == linePos {
				return nil, 0, 0
			}

		case close:
			if !single && !double {
				if match == count {
					split = append(split, string(line[pos[count]:i]))
					return split, 1, 0
				}
				if i == linePos {
					split = []string{
						string(line[:pos[count]-1]),
						string(line[pos[count]:i]),
					}
					return split, 1, len(split[1])
				}
				count--

			} else if i == linePos {
				return nil, 0, 0
			}
		}
	}

	return nil, 0, 0
}

func rTrimWhiteSpace(oldString string) (newString string) {
	return strings.TrimRight(oldString, " ")
	// TODO: support tab chars
	/*defer fmt.Println(">" + oldString + "<" + newString + ">")
	newString = oldString
	for len(oldString) > 0 {
		if newString[len(newString)-1] == ' ' || newString[len(newString)-1] == '\t' {
			newString = newString[:len(newString)-1]
		} else {
			break
		}
	}
	return*/
}
