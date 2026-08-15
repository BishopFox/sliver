package extensions

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// splitExtensionArgs separates Sliver-owned flags (--save, --timeout, --help)
// from the extension's own arguments in a raw argument slice.
//
// Extension commands run with cobra's DisableFlagParsing enabled (see
// ExtensionRegisterCommand). Without it, pflag parses the command line first
// and, because the extension's own flags (e.g. --type, --fqdn) are unknown to
// pflag, ParseErrorsWhitelist.UnknownFlags causes pflag to silently drop each
// unknown flag together with its value; the BOF argument parser never saw them
// unless the operator injected cobra's "--" separator (#2309). Running with
// flag parsing disabled hands us the raw, unstripped slice, so we separate the
// Sliver-owned flags ourselves here.
//
// A literal "--" stops Sliver-flag scanning; every token after it is passed
// through verbatim, preserving the long-standing workaround syntax. Unknown
// tokens (the extension's named flags, their values and positional arguments)
// are never consumed or reordered, so both forms work:
//
//	delegationbof --timeout 30 --type 6 --fqdn child.htb.local
//	delegationbof -- --type 6 --fqdn child.htb.local
func splitExtensionArgs(raw []string) (save bool, timeout int64, help bool, extArgs []string) {
	timeout = defaultTimeout
	extArgs = make([]string, 0, len(raw))

	passthrough := false
	for i := 0; i < len(raw); i++ {
		tok := raw[i]

		// Once we have seen "--", copy everything through untouched so the
		// existing `ext -- --foo bar` syntax keeps working identically.
		if passthrough {
			extArgs = append(extArgs, tok)
			continue
		}

		switch {
		case tok == "--":
			passthrough = true

		case tok == "--help" || tok == "-h":
			help = true

		case tok == "--save" || tok == "-s":
			save = true

		case strings.HasPrefix(tok, "--save="):
			save = parseBoolValue(tok[len("--save="):], true)

		case tok == "--timeout" || tok == "-t":
			// Consume the following token as the value, mirroring pflag. If no
			// token follows, leave the default in place rather than erroring.
			if i+1 < len(raw) {
				if n, err := strconv.ParseInt(raw[i+1], 10, 64); err == nil {
					timeout = n
				}
				i++
			}

		case strings.HasPrefix(tok, "--timeout="):
			if n, err := strconv.ParseInt(tok[len("--timeout="):], 10, 64); err == nil {
				timeout = n
			}

		default:
			// Not a Sliver-owned flag: hand it to the extension parser untouched.
			extArgs = append(extArgs, tok)
		}
	}

	return save, timeout, help, extArgs
}

// applyOwnedFlags republishes the manually parsed Sliver-owned flag values back
// onto the command's flag set. DisableFlagParsing prevents pflag from populating
// them, so helpers that read them via cmd.Flags() (notably ActiveTarget.Request,
// which derives the gRPC timeout from --timeout) would otherwise observe only the
// defaults.
func applyOwnedFlags(cmd *cobra.Command, save bool, timeout int64) {
	if flag := cmd.Flags().Lookup("timeout"); flag != nil {
		_ = flag.Value.Set(strconv.FormatInt(timeout, 10))
	}
	if flag := cmd.Flags().Lookup("save"); flag != nil {
		_ = flag.Value.Set(strconv.FormatBool(save))
	}
}

// parseBoolValue parses a --flag=value bool, defaulting to defVal when the value
// is empty (the bare-flag form). It accepts the same spellings pflag does.
func parseBoolValue(v string, defVal bool) bool {
	switch strings.ToLower(v) {
	case "", "true", "t", "1":
		return true
	case "false", "f", "0":
		return false
	}
	return defVal
}
