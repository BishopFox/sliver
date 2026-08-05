package table

// CSVOptions defines options to control CSV rendering.
type CSVOptions struct {
	// FieldProtection neutralizes fields that spreadsheet applications could
	// interpret as formulas (fields beginning with =, +, -, @, tab or CR) by
	// prefixing them with a single-quote, preventing CSV formula injection
	// when the rendered output is opened in such an application.
	//
	// Note that the single-quote becomes part of the field content for
	// standards-compliant CSV readers; enable this only when the output is
	// destined for a spreadsheet application.
	FieldProtection bool
}

var (
	// DefaultCSVOptions defines sensible CSV rendering defaults.
	DefaultCSVOptions = CSVOptions{}
)
