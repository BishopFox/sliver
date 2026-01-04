package table

// SizeOptions defines the way to control the width of the table output.
type SizeOptions struct {
	// WidthMax is the maximum allotted width for the full row;
	// any content beyond this will be truncated using the text
	// in Style.Box.UnfinishedRow
	WidthMax int
	// WidthMin is the minimum allotted width for the full row;
	// columns will be auto-expanded until the overall width
	// is met
	WidthMin int
}

var (
	// SizeOptionsDefault defines sensible size options - basically NONE.
	SizeOptionsDefault = SizeOptions{
		WidthMax: 0,
		WidthMin: 0,
	}
)
