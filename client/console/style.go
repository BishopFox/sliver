package console

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

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
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/bishopfox/sliver/client/theme"
)

// TextStyle is the shared styling type for the client/server consoles.
// Keep this type in the console package so most callsites don't need to import
// lipgloss directly.
type TextStyle = lipgloss.Style

var (
	StyleNormal    = lipgloss.NewStyle()
	StyleBold      = lipgloss.NewStyle().Bold(true)
	StyleUnderline = lipgloss.NewStyle().Underline(true)

	// Theme palette styles.
	StylePrimary   = lipgloss.NewStyle()
	StyleSecondary = lipgloss.NewStyle()
	StyleDefault   = lipgloss.NewStyle()
	StyleSuccess   = lipgloss.NewStyle()
	StyleWarning   = lipgloss.NewStyle()
	StyleDanger    = lipgloss.NewStyle()

	StyleBoldPrimary   = lipgloss.NewStyle()
	StyleBoldSecondary = lipgloss.NewStyle()
	StyleBoldDefault   = lipgloss.NewStyle()
	StyleBoldSuccess   = lipgloss.NewStyle()
	StyleBoldWarning   = lipgloss.NewStyle()
	StyleBoldDanger    = lipgloss.NewStyle()

	// Legacy color names used across the codebase.
	StyleBlack  = lipgloss.NewStyle()
	StyleRed    = lipgloss.NewStyle()
	StyleGreen  = lipgloss.NewStyle()
	StyleOrange = lipgloss.NewStyle()
	StyleBlue   = lipgloss.NewStyle()
	StylePurple = lipgloss.NewStyle()
	StyleCyan   = lipgloss.NewStyle()
	StyleGray   = lipgloss.NewStyle()

	StyleBoldBlack  = lipgloss.NewStyle()
	StyleBoldRed    = lipgloss.NewStyle()
	StyleBoldGreen  = lipgloss.NewStyle()
	StyleBoldOrange = lipgloss.NewStyle()
	StyleBoldBlue   = lipgloss.NewStyle()
	StyleBoldPurple = lipgloss.NewStyle()
	StyleBoldCyan   = lipgloss.NewStyle()
	StyleBoldGray   = lipgloss.NewStyle()
)

// Prefix tags used across the client/server consoles.
var (
	Info    string
	Warn    string
	Debug   string
	Woot    string
	Success string
)

func init() {
	// Apply a usable theme without doing any disk I/O.
	ApplyTheme(theme.Current())
}

// ApplyTheme updates all exported styles/prefixes to match the provided theme.
//
// Note: this is intentionally a "global theme" for the console, since most of
// the existing code expects these styles to be package-level values.
func ApplyTheme(t theme.Theme) {
	palette := func(p theme.Palette) color.Color {
		if p.Default != "" {
			return lipgloss.Color(p.Default)
		}
		return lipgloss.Color("#ffffff")
	}
	paletteMod := func(p theme.Palette, mod int) color.Color {
		if mod != 0 {
			if v, ok := p.Mods[mod]; ok && v != "" {
				return lipgloss.Color(v)
			}
		}
		return palette(p)
	}

	// Palette styles.
	StylePrimary = lipgloss.NewStyle().Foreground(palette(t.Primary))
	StyleSecondary = lipgloss.NewStyle().Foreground(palette(t.Secondary))
	StyleDefault = lipgloss.NewStyle().Foreground(palette(t.Default))
	StyleSuccess = lipgloss.NewStyle().Foreground(palette(t.Success))
	StyleWarning = lipgloss.NewStyle().Foreground(palette(t.Warning))
	StyleDanger = lipgloss.NewStyle().Foreground(palette(t.Danger))

	StyleBoldPrimary = lipgloss.NewStyle().Bold(true).Foreground(palette(t.Primary))
	StyleBoldSecondary = lipgloss.NewStyle().Bold(true).Foreground(palette(t.Secondary))
	StyleBoldDefault = lipgloss.NewStyle().Bold(true).Foreground(palette(t.Default))
	StyleBoldSuccess = lipgloss.NewStyle().Bold(true).Foreground(palette(t.Success))
	StyleBoldWarning = lipgloss.NewStyle().Bold(true).Foreground(palette(t.Warning))
	StyleBoldDanger = lipgloss.NewStyle().Bold(true).Foreground(palette(t.Danger))

	// Legacy mapping.
	StyleBlack = lipgloss.NewStyle().Foreground(paletteMod(t.Default, 50))
	// ANSI "gray" (37) is typically light; map to the lightest neutral.
	StyleGray = lipgloss.NewStyle().Foreground(paletteMod(t.Default, 900))

	StyleBlue = StylePrimary
	// Keep Cyan distinct from Blue using a lighter primary tint.
	StyleCyan = lipgloss.NewStyle().Foreground(paletteMod(t.Primary, 500))
	StylePurple = StyleSecondary

	StyleGreen = StyleSuccess
	StyleOrange = StyleWarning
	StyleRed = StyleDanger

	StyleBoldBlack = lipgloss.NewStyle().Bold(true).Foreground(paletteMod(t.Default, 50))
	StyleBoldGray = lipgloss.NewStyle().Bold(true).Foreground(paletteMod(t.Default, 900))

	StyleBoldBlue = StyleBoldPrimary
	StyleBoldCyan = lipgloss.NewStyle().Bold(true).Foreground(paletteMod(t.Primary, 500))
	StyleBoldPurple = StyleBoldSecondary

	StyleBoldGreen = StyleBoldSuccess
	StyleBoldOrange = StyleBoldWarning
	StyleBoldRed = StyleBoldDanger

	// Prefix tags.
	Info = StyleBoldCyan.Render("[*] ")
	Warn = StyleBoldDanger.Render("[!] ")
	Debug = StyleBoldSecondary.Render("[-] ")
	Woot = StyleBoldSuccess.Render("[$] ")
	Success = StyleBoldSuccess.Render("[+] ")
}
