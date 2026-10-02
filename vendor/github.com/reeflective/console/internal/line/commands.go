package line

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// ParseCommands splits input at newlines between shell statements and parses
// each resulting command with the same rules as Parse. Newlines within quotes
// and escaped newlines remain part of their command. Semicolons on one line do
// not introduce a new command.
func ParseCommands(input string) ([][]string, error) {
	file, err := syntax.NewParser(syntax.KeepComments(false)).Parse(strings.NewReader(input), "")
	if err != nil {
		return nil, err
	}

	var commands [][]string
	var start, end uint
	for i, stmt := range file.Stmts {
		pos := stmt.Pos().Offset()
		if i == 0 {
			start = pos
		} else if strings.ContainsAny(input[end:pos], "\r\n") {
			args, err := Parse(input[start:end])
			if err != nil {
				return nil, err
			}
			if len(args) > 0 {
				commands = append(commands, args)
			}
			start = pos
		}
		end = stmt.End().Offset()
	}

	if len(file.Stmts) > 0 {
		args, err := Parse(input[start:end])
		if err != nil {
			return nil, err
		}
		if len(args) > 0 {
			commands = append(commands, args)
		}
	}

	return commands, nil
}
