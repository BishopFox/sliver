package table

import "github.com/jedib0t/go-pretty/v6/text"

// ColorOptions defines the ANSI colors to use for parts of the Table.
type ColorOptions struct {
	Border       text.Colors // borders (if nil, uses one of the below)
	Footer       text.Colors // footer row(s) colors
	Header       text.Colors // header row(s) colors
	IndexColumn  text.Colors // index-column colors (row #, etc.)
	Row          text.Colors // regular row(s) colors
	RowAlternate text.Colors // regular row(s) colors for the even-numbered rows
	Separator    text.Colors // separators (if nil, uses one of the above)
}

var (
	// ColorOptionsDefault defines sensible ANSI color options - basically NONE.
	ColorOptionsDefault = ColorOptions{}

	// ColorOptionsBright renders dark text on bright background.
	ColorOptionsBright = ColorOptionsBlackOnCyanWhite

	// ColorOptionsDark renders bright text on dark background.
	ColorOptionsDark = ColorOptionsCyanWhiteOnBlack

	// ColorOptionsBlackOnBlueWhite renders Black text on Blue/White background.
	ColorOptionsBlackOnBlueWhite = ColorOptions{
		Footer:       text.Colors{text.BgBlue, text.FgBlack},
		Header:       text.Colors{text.BgHiBlue, text.FgBlack},
		IndexColumn:  text.Colors{text.BgHiBlue, text.FgBlack},
		Row:          text.Colors{text.BgHiWhite, text.FgBlack},
		RowAlternate: text.Colors{text.BgWhite, text.FgBlack},
	}

	// ColorOptionsBlackOnCyanWhite renders Black text on Cyan/White background.
	ColorOptionsBlackOnCyanWhite = ColorOptions{
		Footer:       text.Colors{text.BgCyan, text.FgBlack},
		Header:       text.Colors{text.BgHiCyan, text.FgBlack},
		IndexColumn:  text.Colors{text.BgHiCyan, text.FgBlack},
		Row:          text.Colors{text.BgHiWhite, text.FgBlack},
		RowAlternate: text.Colors{text.BgWhite, text.FgBlack},
	}

	// ColorOptionsBlackOnGreenWhite renders Black text on Green/White
	// background.
	ColorOptionsBlackOnGreenWhite = ColorOptions{
		Footer:       text.Colors{text.BgGreen, text.FgBlack},
		Header:       text.Colors{text.BgHiGreen, text.FgBlack},
		IndexColumn:  text.Colors{text.BgHiGreen, text.FgBlack},
		Row:          text.Colors{text.BgHiWhite, text.FgBlack},
		RowAlternate: text.Colors{text.BgWhite, text.FgBlack},
	}

	// ColorOptionsBlackOnMagentaWhite renders Black text on Magenta/White
	// background.
	ColorOptionsBlackOnMagentaWhite = ColorOptions{
		Footer:       text.Colors{text.BgMagenta, text.FgBlack},
		Header:       text.Colors{text.BgHiMagenta, text.FgBlack},
		IndexColumn:  text.Colors{text.BgHiMagenta, text.FgBlack},
		Row:          text.Colors{text.BgHiWhite, text.FgBlack},
		RowAlternate: text.Colors{text.BgWhite, text.FgBlack},
	}

	// ColorOptionsBlackOnRedWhite renders Black text on Red/White background.
	ColorOptionsBlackOnRedWhite = ColorOptions{
		Footer:       text.Colors{text.BgRed, text.FgBlack},
		Header:       text.Colors{text.BgHiRed, text.FgBlack},
		IndexColumn:  text.Colors{text.BgHiRed, text.FgBlack},
		Row:          text.Colors{text.BgHiWhite, text.FgBlack},
		RowAlternate: text.Colors{text.BgWhite, text.FgBlack},
	}

	// ColorOptionsBlackOnYellowWhite renders Black text on Yellow/White
	// background.
	ColorOptionsBlackOnYellowWhite = ColorOptions{
		Footer:       text.Colors{text.BgYellow, text.FgBlack},
		Header:       text.Colors{text.BgHiYellow, text.FgBlack},
		IndexColumn:  text.Colors{text.BgHiYellow, text.FgBlack},
		Row:          text.Colors{text.BgHiWhite, text.FgBlack},
		RowAlternate: text.Colors{text.BgWhite, text.FgBlack},
	}

	// ColorOptionsBlueWhiteOnBlack renders Blue/White text on Black background.
	ColorOptionsBlueWhiteOnBlack = ColorOptions{
		Footer:       text.Colors{text.FgBlue, text.BgHiBlack},
		Header:       text.Colors{text.FgHiBlue, text.BgHiBlack},
		IndexColumn:  text.Colors{text.FgHiBlue, text.BgHiBlack},
		Row:          text.Colors{text.FgHiWhite, text.BgBlack},
		RowAlternate: text.Colors{text.FgWhite, text.BgBlack},
	}

	// ColorOptionsCyanWhiteOnBlack renders Cyan/White text on Black background.
	ColorOptionsCyanWhiteOnBlack = ColorOptions{
		Footer:       text.Colors{text.FgCyan, text.BgHiBlack},
		Header:       text.Colors{text.FgHiCyan, text.BgHiBlack},
		IndexColumn:  text.Colors{text.FgHiCyan, text.BgHiBlack},
		Row:          text.Colors{text.FgHiWhite, text.BgBlack},
		RowAlternate: text.Colors{text.FgWhite, text.BgBlack},
	}

	// ColorOptionsGreenWhiteOnBlack renders Green/White text on Black
	// background.
	ColorOptionsGreenWhiteOnBlack = ColorOptions{
		Footer:       text.Colors{text.FgGreen, text.BgHiBlack},
		Header:       text.Colors{text.FgHiGreen, text.BgHiBlack},
		IndexColumn:  text.Colors{text.FgHiGreen, text.BgHiBlack},
		Row:          text.Colors{text.FgHiWhite, text.BgBlack},
		RowAlternate: text.Colors{text.FgWhite, text.BgBlack},
	}

	// ColorOptionsMagentaWhiteOnBlack renders Magenta/White text on Black
	// background.
	ColorOptionsMagentaWhiteOnBlack = ColorOptions{
		Footer:       text.Colors{text.FgMagenta, text.BgHiBlack},
		Header:       text.Colors{text.FgHiMagenta, text.BgHiBlack},
		IndexColumn:  text.Colors{text.FgHiMagenta, text.BgHiBlack},
		Row:          text.Colors{text.FgHiWhite, text.BgBlack},
		RowAlternate: text.Colors{text.FgWhite, text.BgBlack},
	}

	// ColorOptionsRedWhiteOnBlack renders Red/White text on Black background.
	ColorOptionsRedWhiteOnBlack = ColorOptions{
		Footer:       text.Colors{text.FgRed, text.BgHiBlack},
		Header:       text.Colors{text.FgHiRed, text.BgHiBlack},
		IndexColumn:  text.Colors{text.FgHiRed, text.BgHiBlack},
		Row:          text.Colors{text.FgHiWhite, text.BgBlack},
		RowAlternate: text.Colors{text.FgWhite, text.BgBlack},
	}

	// ColorOptionsYellowWhiteOnBlack renders Yellow/White text on Black
	// background.
	ColorOptionsYellowWhiteOnBlack = ColorOptions{
		Footer:       text.Colors{text.FgYellow, text.BgHiBlack},
		Header:       text.Colors{text.FgHiYellow, text.BgHiBlack},
		IndexColumn:  text.Colors{text.FgHiYellow, text.BgHiBlack},
		Row:          text.Colors{text.FgHiWhite, text.BgBlack},
		RowAlternate: text.Colors{text.FgWhite, text.BgBlack},
	}
)
