package table

// MarkdownOptions defines options to control Markdown rendering.
type MarkdownOptions struct {
	// PadContent pads each column content to match the longest content in
	// the column, and extends the separator dashes to match. This makes the
	// raw Markdown source more readable without affecting the rendered
	// output.
	//
	// When disabled (default):
	//  | # | First Name | Last Name | Salary |  |
	//  | ---:| --- | --- | ---:| --- |
	//  | 1 | Arya | Stark | 3000 |  |
	//
	// When enabled:
	//  | # | First Name | Last Name | Salary |                             |
	//  | ---:| ---------- | --------- | ------:| --------------------------- |
	//  | 1 | Arya       | Stark     |   3000 |                             |
	PadContent bool
}

var (
	// DefaultMarkdownOptions defines sensible Markdown rendering defaults.
	DefaultMarkdownOptions = MarkdownOptions{}
)
