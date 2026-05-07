package table

// Style declares how to render the Table and provides very fine-grained control
// on how the Table gets rendered on the Console.
type Style struct {
	Name    string        // name of the Style
	Box     BoxStyle      // characters to use for the boxes
	Color   ColorOptions  // colors to use for the rows and columns
	Format  FormatOptions // formatting options for the rows and columns
	HTML    HTMLOptions   // rendering options for HTML mode
	Options Options       // misc. options for the table
	Size    SizeOptions   // size (width) options for the table
	Title   TitleOptions  // formation options for the title text
}

var (
	// StyleDefault renders a Table like below:
	//  +-----+------------+-----------+--------+-----------------------------+
	//  |   # | FIRST NAME | LAST NAME | SALARY |                             |
	//  +-----+------------+-----------+--------+-----------------------------+
	//  |   1 | Arya       | Stark     |   3000 |                             |
	//  |  20 | Jon        | Snow      |   2000 | You know nothing, Jon Snow! |
	//  | 300 | Tyrion     | Lannister |   5000 |                             |
	//  +-----+------------+-----------+--------+-----------------------------+
	//  |     |            | TOTAL     |  10000 |                             |
	//  +-----+------------+-----------+--------+-----------------------------+
	StyleDefault = Style{
		Name:    "StyleDefault",
		Box:     StyleBoxDefault,
		Color:   ColorOptionsDefault,
		Format:  FormatOptionsDefault,
		HTML:    DefaultHTMLOptions,
		Options: OptionsDefault,
		Size:    SizeOptionsDefault,
		Title:   TitleOptionsDefault,
	}

	// StyleBold renders a Table like below:
	//  ┏━━━━━┳━━━━━━━━━━━━┳━━━━━━━━━━━┳━━━━━━━━┳━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
	//  ┃   # ┃ FIRST NAME ┃ LAST NAME ┃ SALARY ┃                             ┃
	//  ┣━━━━━╋━━━━━━━━━━━━╋━━━━━━━━━━━╋━━━━━━━━╋━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫
	//  ┃   1 ┃ Arya       ┃ Stark     ┃   3000 ┃                             ┃
	//  ┃  20 ┃ Jon        ┃ Snow      ┃   2000 ┃ You know nothing, Jon Snow! ┃
	//  ┃ 300 ┃ Tyrion     ┃ Lannister ┃   5000 ┃                             ┃
	//  ┣━━━━━╋━━━━━━━━━━━━╋━━━━━━━━━━━╋━━━━━━━━╋━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┫
	//  ┃     ┃            ┃ TOTAL     ┃  10000 ┃                             ┃
	//  ┗━━━━━┻━━━━━━━━━━━━┻━━━━━━━━━━━┻━━━━━━━━┻━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
	StyleBold = Style{
		Name:    "StyleBold",
		Box:     StyleBoxBold,
		Color:   ColorOptionsDefault,
		Format:  FormatOptionsDefault,
		HTML:    DefaultHTMLOptions,
		Options: OptionsDefault,
		Size:    SizeOptionsDefault,
		Title:   TitleOptionsDefault,
	}

	// StyleColoredBright renders a Table without any borders or separators,
	// and with Black text on Cyan background for Header/Footer and
	// White background for other rows.
	StyleColoredBright = Style{
		Name:    "StyleColoredBright",
		Box:     StyleBoxDefault,
		Color:   ColorOptionsBright,
		Format:  FormatOptionsDefault,
		HTML:    DefaultHTMLOptions,
		Options: OptionsNoBordersAndSeparators,
		Size:    SizeOptionsDefault,
		Title:   TitleOptionsDark,
	}

	// StyleColoredDark renders a Table without any borders or separators, and
	// with Header/Footer in Cyan text and other rows with White text, all on
	// Black background.
	StyleColoredDark = Style{
		Name:    "StyleColoredDark",
		Box:     StyleBoxDefault,
		Color:   ColorOptionsDark,
		Format:  FormatOptionsDefault,
		HTML:    DefaultHTMLOptions,
		Options: OptionsNoBordersAndSeparators,
		Size:    SizeOptionsDefault,
		Title:   TitleOptionsBright,
	}

	// StyleColoredBlackOnBlueWhite renders a Table without any borders or
	// separators, and with Black text on Blue background for Header/Footer and
	// White background for other rows.
	StyleColoredBlackOnBlueWhite = Style{
		Name:    "StyleColoredBlackOnBlueWhite",
		Box:     StyleBoxDefault,
		Color:   ColorOptionsBlackOnBlueWhite,
		Format:  FormatOptionsDefault,
		HTML:    DefaultHTMLOptions,
		Options: OptionsNoBordersAndSeparators,
		Size:    SizeOptionsDefault,
		Title:   TitleOptionsBlueOnBlack,
	}

	// StyleColoredBlackOnCyanWhite renders a Table without any borders or
	// separators, and with Black text on Cyan background for Header/Footer and
	// White background for other rows.
	StyleColoredBlackOnCyanWhite = Style{
		Name:    "StyleColoredBlackOnCyanWhite",
		Box:     StyleBoxDefault,
		Color:   ColorOptionsBlackOnCyanWhite,
		Format:  FormatOptionsDefault,
		HTML:    DefaultHTMLOptions,
		Options: OptionsNoBordersAndSeparators,
		Size:    SizeOptionsDefault,
		Title:   TitleOptionsCyanOnBlack,
	}

	// StyleColoredBlackOnGreenWhite renders a Table without any borders or
	// separators, and with Black text on Green background for Header/Footer and
	// White background for other rows.
	StyleColoredBlackOnGreenWhite = Style{
		Name:    "StyleColoredBlackOnGreenWhite",
		Box:     StyleBoxDefault,
		Color:   ColorOptionsBlackOnGreenWhite,
		Format:  FormatOptionsDefault,
		HTML:    DefaultHTMLOptions,
		Options: OptionsNoBordersAndSeparators,
		Size:    SizeOptionsDefault,
		Title:   TitleOptionsGreenOnBlack,
	}

	// StyleColoredBlackOnMagentaWhite renders a Table without any borders or
	// separators, and with Black text on Magenta background for Header/Footer and
	// White background for other rows.
	StyleColoredBlackOnMagentaWhite = Style{
		Name:    "StyleColoredBlackOnMagentaWhite",
		Box:     StyleBoxDefault,
		Color:   ColorOptionsBlackOnMagentaWhite,
		Format:  FormatOptionsDefault,
		HTML:    DefaultHTMLOptions,
		Options: OptionsNoBordersAndSeparators,
		Size:    SizeOptionsDefault,
		Title:   TitleOptionsMagentaOnBlack,
	}

	// StyleColoredBlackOnYellowWhite renders a Table without any borders or
	// separators, and with Black text on Yellow background for Header/Footer and
	// White background for other rows.
	StyleColoredBlackOnYellowWhite = Style{
		Name:    "StyleColoredBlackOnYellowWhite",
		Box:     StyleBoxDefault,
		Color:   ColorOptionsBlackOnYellowWhite,
		Format:  FormatOptionsDefault,
		HTML:    DefaultHTMLOptions,
		Options: OptionsNoBordersAndSeparators,
		Size:    SizeOptionsDefault,
		Title:   TitleOptionsYellowOnBlack,
	}

	// StyleColoredBlackOnRedWhite renders a Table without any borders or
	// separators, and with Black text on Red background for Header/Footer and
	// White background for other rows.
	StyleColoredBlackOnRedWhite = Style{
		Name:    "StyleColoredBlackOnRedWhite",
		Box:     StyleBoxDefault,
		Color:   ColorOptionsBlackOnRedWhite,
		Format:  FormatOptionsDefault,
		HTML:    DefaultHTMLOptions,
		Options: OptionsNoBordersAndSeparators,
		Size:    SizeOptionsDefault,
		Title:   TitleOptionsRedOnBlack,
	}

	// StyleColoredBlueWhiteOnBlack renders a Table without any borders or
	// separators, and with Header/Footer in Blue text and other rows with
	// White text, all on Black background.
	StyleColoredBlueWhiteOnBlack = Style{
		Name:    "StyleColoredBlueWhiteOnBlack",
		Box:     StyleBoxDefault,
		Color:   ColorOptionsBlueWhiteOnBlack,
		Format:  FormatOptionsDefault,
		HTML:    DefaultHTMLOptions,
		Options: OptionsNoBordersAndSeparators,
		Size:    SizeOptionsDefault,
		Title:   TitleOptionsBlackOnBlue,
	}

	// StyleColoredCyanWhiteOnBlack renders a Table without any borders or
	// separators, and with Header/Footer in Cyan text and other rows with
	// White text, all on Black background.
	StyleColoredCyanWhiteOnBlack = Style{
		Name:    "StyleColoredCyanWhiteOnBlack",
		Box:     StyleBoxDefault,
		Color:   ColorOptionsCyanWhiteOnBlack,
		Format:  FormatOptionsDefault,
		HTML:    DefaultHTMLOptions,
		Options: OptionsNoBordersAndSeparators,
		Size:    SizeOptionsDefault,
		Title:   TitleOptionsBlackOnCyan,
	}

	// StyleColoredGreenWhiteOnBlack renders a Table without any borders or
	// separators, and with Header/Footer in Green text and other rows with
	// White text, all on Black background.
	StyleColoredGreenWhiteOnBlack = Style{
		Name:    "StyleColoredGreenWhiteOnBlack",
		Box:     StyleBoxDefault,
		Color:   ColorOptionsGreenWhiteOnBlack,
		Format:  FormatOptionsDefault,
		HTML:    DefaultHTMLOptions,
		Options: OptionsNoBordersAndSeparators,
		Size:    SizeOptionsDefault,
		Title:   TitleOptionsBlackOnGreen,
	}

	// StyleColoredMagentaWhiteOnBlack renders a Table without any borders or
	// separators, and with Header/Footer in Magenta text and other rows with
	// White text, all on Black background.
	StyleColoredMagentaWhiteOnBlack = Style{
		Name:    "StyleColoredMagentaWhiteOnBlack",
		Box:     StyleBoxDefault,
		Color:   ColorOptionsMagentaWhiteOnBlack,
		Format:  FormatOptionsDefault,
		HTML:    DefaultHTMLOptions,
		Options: OptionsNoBordersAndSeparators,
		Size:    SizeOptionsDefault,
		Title:   TitleOptionsBlackOnMagenta,
	}

	// StyleColoredRedWhiteOnBlack renders a Table without any borders or
	// separators, and with Header/Footer in Red text and other rows with
	// White text, all on Black background.
	StyleColoredRedWhiteOnBlack = Style{
		Name:    "StyleColoredRedWhiteOnBlack",
		Box:     StyleBoxDefault,
		Color:   ColorOptionsRedWhiteOnBlack,
		Format:  FormatOptionsDefault,
		HTML:    DefaultHTMLOptions,
		Options: OptionsNoBordersAndSeparators,
		Size:    SizeOptionsDefault,
		Title:   TitleOptionsBlackOnRed,
	}

	// StyleColoredYellowWhiteOnBlack renders a Table without any borders or
	// separators, and with Header/Footer in Yellow text and other rows with
	// White text, all on Black background.
	StyleColoredYellowWhiteOnBlack = Style{
		Name:    "StyleColoredYellowWhiteOnBlack",
		Box:     StyleBoxDefault,
		Color:   ColorOptionsYellowWhiteOnBlack,
		Format:  FormatOptionsDefault,
		HTML:    DefaultHTMLOptions,
		Options: OptionsNoBordersAndSeparators,
		Size:    SizeOptionsDefault,
		Title:   TitleOptionsBlackOnYellow,
	}

	// StyleDouble renders a Table like below:
	//  ╔═════╦════════════╦═══════════╦════════╦═════════════════════════════╗
	//  ║   # ║ FIRST NAME ║ LAST NAME ║ SALARY ║                             ║
	//  ╠═════╬════════════╬═══════════╬════════╬═════════════════════════════╣
	//  ║   1 ║ Arya       ║ Stark     ║   3000 ║                             ║
	//  ║  20 ║ Jon        ║ Snow      ║   2000 ║ You know nothing, Jon Snow! ║
	//  ║ 300 ║ Tyrion     ║ Lannister ║   5000 ║                             ║
	//  ╠═════╬════════════╬═══════════╬════════╬═════════════════════════════╣
	//  ║     ║            ║ TOTAL     ║  10000 ║                             ║
	//  ╚═════╩════════════╩═══════════╩════════╩═════════════════════════════╝
	StyleDouble = Style{
		Name:    "StyleDouble",
		Box:     StyleBoxDouble,
		Color:   ColorOptionsDefault,
		Format:  FormatOptionsDefault,
		HTML:    DefaultHTMLOptions,
		Options: OptionsDefault,
		Size:    SizeOptionsDefault,
		Title:   TitleOptionsDefault,
	}

	// StyleLight renders a Table like below:
	//  ┌─────┬────────────┬───────────┬────────┬─────────────────────────────┐
	//  │   # │ FIRST NAME │ LAST NAME │ SALARY │                             │
	//  ├─────┼────────────┼───────────┼────────┼─────────────────────────────┤
	//  │   1 │ Arya       │ Stark     │   3000 │                             │
	//  │  20 │ Jon        │ Snow      │   2000 │ You know nothing, Jon Snow! │
	//  │ 300 │ Tyrion     │ Lannister │   5000 │                             │
	//  ├─────┼────────────┼───────────┼────────┼─────────────────────────────┤
	//  │     │            │ TOTAL     │  10000 │                             │
	//  └─────┴────────────┴───────────┴────────┴─────────────────────────────┘
	StyleLight = Style{
		Name:    "StyleLight",
		Box:     StyleBoxLight,
		Color:   ColorOptionsDefault,
		Format:  FormatOptionsDefault,
		HTML:    DefaultHTMLOptions,
		Options: OptionsDefault,
		Size:    SizeOptionsDefault,
		Title:   TitleOptionsDefault,
	}

	// StyleRounded renders a Table like below:
	//  ╭─────┬────────────┬───────────┬────────┬─────────────────────────────╮
	//  │   # │ FIRST NAME │ LAST NAME │ SALARY │                             │
	//  ├─────┼────────────┼───────────┼────────┼─────────────────────────────┤
	//  │   1 │ Arya       │ Stark     │   3000 │                             │
	//  │  20 │ Jon        │ Snow      │   2000 │ You know nothing, Jon Snow! │
	//  │ 300 │ Tyrion     │ Lannister │   5000 │                             │
	//  ├─────┼────────────┼───────────┼────────┼─────────────────────────────┤
	//  │     │            │ TOTAL     │  10000 │                             │
	//  ╰─────┴────────────┴───────────┴────────┴─────────────────────────────╯
	StyleRounded = Style{
		Name:    "StyleRounded",
		Box:     StyleBoxRounded,
		Color:   ColorOptionsDefault,
		Format:  FormatOptionsDefault,
		HTML:    DefaultHTMLOptions,
		Options: OptionsDefault,
		Size:    SizeOptionsDefault,
		Title:   TitleOptionsDefault,
	}

	// styleTest renders a Table like below:
	//  (-----^------------^-----------^--------^-----------------------------)
	//  [<  #>|<FIRST NAME>|<LAST NAME>|<SALARY>|<                           >]
	//  {-----+------------+-----------+--------+-----------------------------}
	//  [<  1>|<Arya      >|<Stark    >|<  3000>|<                           >]
	//  [< 20>|<Jon       >|<Snow     >|<  2000>|<You know nothing, Jon Snow!>]
	//  [<300>|<Tyrion    >|<Lannister>|<  5000>|<                           >]
	//  {-----+------------+-----------+--------+-----------------------------}
	//  [<   >|<          >|<TOTAL    >|< 10000>|<                           >]
	//  \-----v------------v-----------v--------v-----------------------------/
	styleTest = Style{
		Name:    "styleTest",
		Box:     styleBoxTest,
		Color:   ColorOptionsDefault,
		Format:  FormatOptionsDefault,
		HTML:    DefaultHTMLOptions,
		Options: OptionsDefault,
		Size:    SizeOptionsDefault,
		Title:   TitleOptionsDefault,
	}
)
