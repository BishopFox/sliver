package implant

/*
	Sliver Implant Framework
	Copyright (C) 2020  Bishop Fox

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

import "embed"

var (
	// FS - Embedded FS access to sliver implant code
	//go:embed sliver/**
	FS embed.FS

	// GoMod - Templated go.mod module file for implant builds
	//go:embed go-mod
	GoMod string

	// GoSum - Templated go.sum module file for implant builds
	//go:embed go-sum
	GoSum string

	// Vendor - Embedded FS access to vendored sliver implant dependencies
	//go:embed vendor/**
	Vendor embed.FS
)
