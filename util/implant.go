package util

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
	"errors"
	"regexp"
)

func AllowedName(name string) error {
	if name != "" {
		// allow for alphanumeric, periods, dashes, and underscores in name
		isAllowed := regexp.MustCompile(`^[[:alnum:]\.\-_]+$`).MatchString
		// do not allow for files ".", "..", or anything starting with ".."
		additionalDeny := regexp.MustCompile(`^\.\.|^\.$`).MatchString
		if !isAllowed(name) {
			return errors.New("Name must be alphanumeric or .-_ only\n")
		} else if additionalDeny(name) {
			return errors.New("Name cannot be \".\", \"..\", or start with \"..\"")
		} else {
			return nil
		}
	} else {
		return errors.New("Name cannot be blank!")
	}
}
