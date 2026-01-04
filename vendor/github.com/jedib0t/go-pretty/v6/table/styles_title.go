package table

import "github.com/jedib0t/go-pretty/v6/text"

// TitleOptions defines the way the title text is to be rendered.
type TitleOptions struct {
	Align  text.Align
	Colors text.Colors
	Format text.Format
}

var (
	// TitleOptionsDefault defines sensible title options - basically NONE.
	TitleOptionsDefault = TitleOptions{}

	// TitleOptionsBright renders Bright Bold text on Dark background.
	TitleOptionsBright = TitleOptionsBlackOnCyan

	// TitleOptionsDark renders Dark Bold text on Bright background.
	TitleOptionsDark = TitleOptionsCyanOnBlack

	// TitleOptionsBlackOnBlue renders Black text on Blue background.
	TitleOptionsBlackOnBlue = TitleOptions{
		Colors: append(ColorOptionsBlackOnBlueWhite.Header, text.Bold),
	}

	// TitleOptionsBlackOnCyan renders Black Bold text on Cyan background.
	TitleOptionsBlackOnCyan = TitleOptions{
		Colors: append(ColorOptionsBlackOnCyanWhite.Header, text.Bold),
	}

	// TitleOptionsBlackOnGreen renders Black Bold text onGreen background.
	TitleOptionsBlackOnGreen = TitleOptions{
		Colors: append(ColorOptionsBlackOnGreenWhite.Header, text.Bold),
	}

	// TitleOptionsBlackOnMagenta renders Black Bold text on Magenta background.
	TitleOptionsBlackOnMagenta = TitleOptions{
		Colors: append(ColorOptionsBlackOnMagentaWhite.Header, text.Bold),
	}

	// TitleOptionsBlackOnRed renders Black Bold text on Red background.
	TitleOptionsBlackOnRed = TitleOptions{
		Colors: append(ColorOptionsBlackOnRedWhite.Header, text.Bold),
	}

	// TitleOptionsBlackOnYellow renders Black Bold text on Yellow background.
	TitleOptionsBlackOnYellow = TitleOptions{
		Colors: append(ColorOptionsBlackOnYellowWhite.Header, text.Bold),
	}

	// TitleOptionsBlueOnBlack renders Blue Bold text on Black background.
	TitleOptionsBlueOnBlack = TitleOptions{
		Colors: append(ColorOptionsBlueWhiteOnBlack.Header, text.Bold),
	}

	// TitleOptionsCyanOnBlack renders Cyan Bold text on Black background.
	TitleOptionsCyanOnBlack = TitleOptions{
		Colors: append(ColorOptionsCyanWhiteOnBlack.Header, text.Bold),
	}

	// TitleOptionsGreenOnBlack renders Green Bold text on Black background.
	TitleOptionsGreenOnBlack = TitleOptions{
		Colors: append(ColorOptionsGreenWhiteOnBlack.Header, text.Bold),
	}

	// TitleOptionsMagentaOnBlack renders Magenta Bold text on Black background.
	TitleOptionsMagentaOnBlack = TitleOptions{
		Colors: append(ColorOptionsMagentaWhiteOnBlack.Header, text.Bold),
	}

	// TitleOptionsRedOnBlack renders Red Bold text on Black background.
	TitleOptionsRedOnBlack = TitleOptions{
		Colors: append(ColorOptionsRedWhiteOnBlack.Header, text.Bold),
	}

	// TitleOptionsYellowOnBlack renders Yellow Bold text on Black background.
	TitleOptionsYellowOnBlack = TitleOptions{
		Colors: append(ColorOptionsYellowWhiteOnBlack.Header, text.Bold),
	}
)
