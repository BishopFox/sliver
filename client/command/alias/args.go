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

// ownedFlagMatch is a raw token that spelled one of the Sliver-owned flags.
type ownedFlagMatch struct {
	spec   ownedFlagSpec
	value  string // value from a "--flag=value" / "-f=value" spelling
	inline bool   // whether the token carried its value inline
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
	byLong, byShort := ownedFlagTables(specs)

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
			return values, true, extArgs, nil
		}

		match, ok := matchOwnedFlag(tok, byLong, byShort)
		if !ok {
			// Not one of ours (including glued shorthands and the payload's
			// own flags): the alias owns the token, pass it through untouched.
			extArgs = append(extArgs, tok)
			continue
		}
		i = takeOwnedValue(match, raw, i, values)
	}

	if err := validateOwnedValues(specs, values); err != nil {
		return nil, false, nil, err
	}
	return values, false, extArgs, nil
}

// ownedFlagTables indexes the specs by long name and shorthand.
func ownedFlagTables(specs []ownedFlagSpec) (byLong, byShort map[string]ownedFlagSpec) {
	byLong = make(map[string]ownedFlagSpec, len(specs))
	byShort = make(map[string]ownedFlagSpec, len(specs))
	for _, spec := range specs {
		byLong[spec.long] = spec
		if spec.short != "" {
			byShort[spec.short] = spec
		}
	}
	return byLong, byShort
}

// matchOwnedFlag checks whether tok spells one of the Sliver-owned flags, in
// the long form (--name, --name=value) or shorthand form (-f, -f=value). A
// glued shorthand value (-pnotepad) deliberately does not match: pflag would
// read "-pid" as -p with value "id", which is exactly the mangling that broke
// #2264, so such tokens must fall through to the payload.
func matchOwnedFlag(tok string, byLong, byShort map[string]ownedFlagSpec) (ownedFlagMatch, bool) {
	var match ownedFlagMatch
	var name, value string
	var hasValue, isShort bool

	switch {
	case strings.HasPrefix(tok, "--"):
		name, value, hasValue = strings.Cut(tok[2:], "=")
	case strings.HasPrefix(tok, "-") && len(tok) > 1:
		name, value, hasValue = strings.Cut(tok[1:], "=")
		isShort = len(name) == 1
	default:
		return match, false
	}

	spec, ok := byLong[name]
	if isShort {
		spec, ok = byShort[name]
	}
	if !ok {
		return match, false
	}
	match.spec, match.value, match.inline = spec, value, hasValue
	return match, true
}

// takeOwnedValue records the flag's value in values and returns the index of
// the last raw token it consumed. The "--flag value" / "-f value" spellings
// consume the following token; a bool without an inline value defaults to
// true; a value flag as the very last token is dropped and keeps its default,
// mirroring the extension behaviour.
func takeOwnedValue(match ownedFlagMatch, raw []string, i int, values map[string]string) int {
	switch {
	case match.spec.kind == ownedBool:
		if !match.inline {
			match.value = "true"
		}
		values[match.spec.long] = match.value
	case match.inline:
		values[match.spec.long] = match.value
	case i+1 < len(raw):
		values[match.spec.long] = raw[i+1]
		return i + 1
	}
	return i
}

// validateOwnedValues rejects non-numeric values for the numeric flag kinds,
// the way pflag's typed values would have.
func validateOwnedValues(specs []ownedFlagSpec, values map[string]string) error {
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
			return fmt.Errorf("invalid value %q for --%s", value, spec.long)
		}
	}
	return nil
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
