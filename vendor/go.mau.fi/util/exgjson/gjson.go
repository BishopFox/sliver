// Copyright (c) 2022 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package exgjson

import (
	"strings"
)

// Escaper escapes a string for use in a GJSON path.
var Escaper = strings.NewReplacer(
	`\`, `\\`,
	".", `\.`,
	"|", `\|`,
	"#", `\#`,
	"@", `\@`,
	"*", `\*`,
	"?", `\?`)

// Path returns a GJSON path pointing at a nested object, with each provided string being a key.
func Path(path ...string) string {
	var result strings.Builder
	for i, part := range path {
		_, _ = Escaper.WriteString(&result, part)
		if i < len(path)-1 {
			result.WriteRune('.')
		}
	}
	return result.String()
}
