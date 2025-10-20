package kong

import (
	"fmt"
	"regexp"
)

var interpolationRegex = regexp.MustCompile(`(\$\$)|((?:\${([[:alpha:]_][[:word:]]*))(?:=([^}]+))?})|(\$)|([^$]+)`)

// HasInterpolatedVar returns true if the variable "v" is interpolated in "s".
func HasInterpolatedVar(s string, v string) bool {
	matches := interpolationRegex.FindAllStringSubmatch(s, -1)
	for _, match := range matches {
		if name := match[3]; name == v {
			return true
		}
	}
	return false
}

// Interpolate variables from vars into s for substrings in the form ${var} or ${var=default}.
func interpolate(s string, vars Vars, updatedVars map[string]string) (string, error) {
	out := ""
	matches := interpolationRegex.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return s, nil
	}
	for key, val := range updatedVars {
		if vars[key] != val {
			vars = vars.CloneWith(updatedVars)
			break
		}
	}
	for _, match := range matches {
		if dollar := match[1]; dollar != "" {
			out += "$"
		} else if name := match[3]; name != "" {
			value, ok := vars[name]
			if !ok {
				// No default value.
				if match[4] == "" {
					return "", fmt.Errorf("undefined variable ${%s}", name)
				}
				value = match[4]
			}
			out += value
		} else {
			out += match[0]
		}
	}
	return out, nil
}
