package readline

/*
   console - Closed-loop console application for cobra commands
   Copyright (C) 2023 Reeflective

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"fmt"
	"strings"
)

// manages display of .inputrc-compliant listings/snippets.
type cfgBuilder struct {
	buf   *strings.Builder
	names []string
}

// Write writes a single inputrc line with the appropriate contextual indent.
func (cfg *cfgBuilder) Write(data []byte) (int, error) {
	indent := strings.Repeat(" ", 4*len(cfg.names))

	iLen, _ := cfg.buf.Write([]byte(indent))
	bLen, err := cfg.buf.Write(data)

	return iLen + bLen, err
}

func (cfg *cfgBuilder) newCond(name string) {
	cfg.Write([]byte(fmt.Sprintf("$if %s\n", name)))
	cfg.names = append(cfg.names, name)
}

func (cfg *cfgBuilder) endCond() {
	if len(cfg.names) == 0 {
		return
	}

	cfg.names = cfg.names[:len(cfg.names)-1]
	cfg.Write([]byte("$endif\n"))
}
