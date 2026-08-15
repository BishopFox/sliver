package alias

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
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// Alias commands run with cobra's DisableFlagParsing enabled (see LoadAlias),
// the same treatment extension commands received for their arguments. Before
// that, an unknown flag such as the payload's own `-pid` was rejected outright
// by pflag ("unknown flag"), and operators had to inject cobra's "--"
// separator (documented on the wiki) to pass arguments through at all (#2264).
//
// With flag parsing disabled cobra hands Run the raw, unstripped slice, and we
// extract the Sliver-owned flags ourselves here. Everything we do not own is
// passed to the payload untouched, so all of these now work:
//
//	amsi-bypass -pid 6969
//	fakealias --process notepad.exe arg1 arg2
//	fakealias -- --anything still-here     (existing workaround, unchanged)
type ownedFlagKind int

const (
	ownedBool ownedFlagKind = iota
	ownedString
	ownedUint32
	ownedInt64
)

// ownedFlagSpec describes one Sliver-owned flag that splitAliasArgs extracts
// from the raw argument slice before the rest is handed to the alias payload.
type ownedFlagSpec struct {
	long  string        // canonical pflag name, e.g. "process-arguments"
	short string        // shorthand letter, e.g. "A" ("" if the flag has none)
	kind  ownedFlagKind // how the value is consumed and validated
}

// baseAliasOwnedFlags are registered for every alias command.
var baseAliasOwnedFlags = []ownedFlagSpec{
	{long: "process", short: "p", kind: ownedString},
	{long: "process-arguments", short: "A", kind: ownedString},
	{long: "ppid", short: "P", kind: ownedUint32},
	{long: "save", short: "s", kind: ownedBool},
	{long: "timeout", short: "t", kind: ownedInt64},
}

// assemblyAliasOwnedFlags are only owned when the alias wraps a .NET assembly;
// for every other alias they are unknown tokens and pass through to the payload.
var assemblyAliasOwnedFlags = []ownedFlagSpec{
	{long: "method", short: "m", kind: ownedString},
	{long: "class", short: "c", kind: ownedString},
	{long: "app-domain", short: "d", kind: ownedString},
	{long: "arch", short: "a", kind: ownedString},
	{long: "in-process", short: "i", kind: ownedBool},
	{long: "runtime", short: "r", kind: ownedString},
	{long: "amsi-bypass", short: "M", kind: ownedBool},
	{long: "etw-bypass", short: "E", kind: ownedBool},
}

// aliasOwnedFlagSpecs returns the flag set owned by a particular alias.
func aliasOwnedFlagSpecs(isAssembly bool) []ownedFlagSpec {
	if !isAssembly {
		return baseAliasOwnedFlags
	}
	specs := make([]ownedFlagSpec, 0, len(baseAliasOwnedFlags)+len(assemblyAliasOwnedFlags))
	specs = append(specs, baseAliasOwnedFlags...)
	return append(specs, assemblyAliasOwnedFlags...)
}

// splitAliasArgs separates the Sliver-owned flags from the alias payload's own
// arguments in a raw (DisableFlagParsing) argument slice. It accepts the long
// forms --flag, --flag=value, --flag value and the shorthand forms -f, -f=value,
// -f value, mirroring pflag. Two deliberate differences from pflag:
//
//   - a glued shorthand value (-pnotepad) does NOT match --process: tokens
//     like the payload's own "-pid" must survive intact (#2264);
//   - after a literal "--" every token is copied through verbatim, preserving
//     the long-standing separator workaround.
//
// The returned map holds the raw string value of each flag that was provided;
// callers re-publish the values with applyOwnedAliasFlags.
func splitAliasArgs(raw []string, specs []ownedFlagSpec) (map[string]string, bool, []string, error) {
	values := make(map[string]string)
	extArgs := make([]string, 0, len(raw))

	byLong := make(map[string]ownedFlagSpec, len(specs))
	byShort := make(map[string]ownedFlagSpec, len(specs))
	for _, spec := range specs {
		byLong[spec.long] = spec
		if spec.short != "" {
			byShort[spec.short] = spec
		}
	}

	passthrough := false
	for i := 0; i < len(raw); i++ {
		tok := raw[i]

		// Once we have seen "--", copy everything through untouched.
		if passthrough {
			extArgs = append(extArgs, tok)
			continue
		}
		if tok == "--" {
			passthrough = true
			continue
		}
		if tok == "--help" || tok == "-h" {
			help := true
			return values, help, extArgs, nil
		}

		var (
			spec   ownedFlagSpec
			value  string
			inline bool
			isFlag bool
		)
		switch {
		case strings.HasPrefix(tok, "--"):
			name, val, hasValue := strings.Cut(tok[2:], "=")
			if spec, isFlag = byLong[name]; isFlag && hasValue {
				value, inline = val, true
			}
		case strings.HasPrefix(tok, "-") && len(tok) > 1:
			sh, val, hasValue := strings.Cut(tok[1:], "=")
			if len(sh) == 1 {
				if spec, isFlag = byShort[sh]; isFlag && hasValue {
					value, inline = val, true
				}
			}
		}

		if !isFlag {
			// Not one of ours (including glued shorthands and the payload's
			// own flags): the alias owns the token, pass it through untouched.
			extArgs = append(extArgs, tok)
			continue
		}

		switch {
		case spec.kind == ownedBool:
			if !inline {
				value = "true"
			}
			values[spec.long] = value
		case inline:
			values[spec.long] = value
		case i+1 < len(raw):
			i++
			values[spec.long] = raw[i]
		default:
			// Value flag as the very last token with no value: keep the
			// flag's default and drop the dangling token.
		}
	}

	for _, spec := range specs {
		value, provided := values[spec.long]
		if !provided {
			continue
		}
		var err error
		switch spec.kind {
		case ownedUint32:
			_, err = strconv.ParseUint(value, 10, 32)
		case ownedInt64:
			_, err = strconv.ParseInt(value, 10, 64)
		}
		if err != nil {
			return nil, false, nil, fmt.Errorf("invalid value %q for --%s", value, spec.long)
		}
	}

	return values, false, extArgs, nil
}

// applyOwnedAliasFlags republishes the manually parsed Sliver-owned flag values
// onto the command's flag set. DisableFlagParsing prevents pflag from
// populating them, so readers such as runAliasCommand's Get* calls and
// ActiveTarget.Request (which derives the gRPC timeout from --timeout) would
// otherwise observe only the defaults.
func applyOwnedAliasFlags(cmd *cobra.Command, values map[string]string) error {
	for name, value := range values {
		flag := cmd.Flags().Lookup(name)
		if flag == nil {
			continue // e.g. an assembly flag on a non-assembly alias
		}
		if err := flag.Value.Set(value); err != nil {
			return fmt.Errorf("invalid value %q for --%s: %w", value, name, err)
		}
	}
	return nil
}
