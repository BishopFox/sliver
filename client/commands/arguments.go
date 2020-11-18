package commands

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

// Command/option argument choices
var (
	implantOS   = []string{"windows", "linux", "darwin"}
	implantArch = []string{"amd64", "x86"}
	implantFmt  = []string{"exe", "shared", "service", "shellcode"}

	msfTransformFormats = []string{
		"bash",
		"c",
		"csharp",
		"dw",
		"dword",
		"hex",
		"java",
		"js_be",
		"js_le",
		"num",
		"perl",
		"pl",
		"powershell",
		"ps1",
		"py",
		"python",
		"raw",
		"rb",
		"ruby",
		"sh",
		"vbapplication",
		"vbscript",
	}
)
