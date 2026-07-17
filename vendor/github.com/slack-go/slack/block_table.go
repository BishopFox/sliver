package slack

// TableBlock defines a block that lets you use a table to display your data.
//
// More Information: https://docs.slack.dev/reference/block-kit/blocks/table-block/
type TableBlock struct {
	Type           MessageBlockType   `json:"type"`
	BlockID        string             `json:"block_id,omitempty"`
	Rows           [][]*RichTextBlock `json:"rows"`
	ColumnSettings []ColumnSetting    `json:"column_settings,omitempty"`
}

type ColumnAlignment string

const (
	ColumnAlignmentLeft   ColumnAlignment = "left"
	ColumnAlignmentCenter ColumnAlignment = "center"
	ColumnAlignmentRight  ColumnAlignment = "right"
)

type ColumnSetting struct {
	Align     ColumnAlignment `json:"align"`
	IsWrapped bool            `json:"is_wrapped"`
}

// BlockType returns the type of the block
func (s TableBlock) BlockType() MessageBlockType {
	return s.Type
}

// ID returns the ID of the block
func (s TableBlock) ID() string {
	return s.BlockID
}

// WithColumnSettings sets the column settings for the Table Block
func (s *TableBlock) WithColumnSettings(columnSettings ...ColumnSetting) *TableBlock {
	s.ColumnSettings = columnSettings
	return s
}

// AddRow adds a new row of cells to the Table Block
func (s *TableBlock) AddRow(cells ...*RichTextBlock) *TableBlock {
	s.Rows = append(s.Rows, append([]*RichTextBlock{}, cells...))
	return s
}

// NewTableBlock returns an instance of a Table Block type
func NewTableBlock(blockID string) *TableBlock {
	return &TableBlock{
		Type:    MBTTable,
		BlockID: blockID,
		Rows:    make([][]*RichTextBlock, 0),
	}
}
