package console

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

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

	"github.com/maxlandon/readline"
)

var (
	Info    = fmt.Sprintf("%s[-]%s ", readline.BLUE, readline.RESET)   // Info - All normal messages
	Warn    = fmt.Sprintf("%s[!]%s ", readline.YELLOW, readline.RESET) // Warn - Errors in parameters, notifiable events in modules/sessions
	Error   = fmt.Sprintf("%s[!]%s ", readline.RED, readline.RESET)    // Error - Error in commands, filters, modules and implants.
	Success = fmt.Sprintf("%s[*]%s ", readline.GREEN, readline.RESET)  // Success - Success events

	Infof   = fmt.Sprintf("%s[-] ", readline.BLUE)   // Infof - formatted
	Warnf   = fmt.Sprintf("%s[!] ", readline.YELLOW) // Warnf - formatted
	Errorf  = fmt.Sprintf("%s[!] ", readline.RED)    // Errorf - formatted
	Sucessf = fmt.Sprintf("%s[*] ", readline.GREEN)  // Sucessf - formatted

	RPCError     = fmt.Sprintf("%s[RPC Error]%s ", readline.RED, readline.RESET)     // RPCError - Errors from the server
	CommandError = fmt.Sprintf("%s[Command Error]%s ", readline.RED, readline.RESET) // CommandError - Command input error
	ParserError  = fmt.Sprintf("%s[Parser Error]%s ", readline.RED, readline.RESET)  // ParserError - Failed to parse some tokens in the input
	DBError      = fmt.Sprintf("%s[DB Error]%s ", readline.RED, readline.RESET)      // DBError - Data Service error
)

const (
	// ANSI Colors
	normal    = "\033[0m"
	black     = "\033[30m"
	red       = "\033[31m"
	green     = "\033[32m"
	orange    = "\033[33m"
	blue      = "\033[34m"
	purple    = "\033[35m"
	cyan      = "\033[36m"
	gray      = "\033[37m"
	bold      = "\033[1m"
	clearln   = "\r\x1b[2K"
	upN       = "\033[%dA"
	downN     = "\033[%dB"
	underline = "\033[4m"

	// Info - Display colorful information
	// Info = bold + cyan + "[*] " + normal
	// Warn - Warn a user
	// Warn = bold + red + "[!] " + normal
	// Debug - Display debug information
	// Debug = bold + purple + "[-] " + normal
	// Woot - Display success
	// Woot = bold + green + "[$] " + normal
)
