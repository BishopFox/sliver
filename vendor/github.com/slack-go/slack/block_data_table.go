package slack

import (
	"encoding/json"
	"fmt"
)

// DataTableCellType identifies the variant of a cell inside a DataTableBlock row.
type DataTableCellType string

const (
	DataTableCellRawText   DataTableCellType = "raw_text"
	DataTableCellRawNumber DataTableCellType = "raw_number"
	DataTableCellRichText  DataTableCellType = "rich_text"
)

// DataTableCell is implemented by every cell type valid inside a DataTableBlock row:
// DataTableRawTextCell, DataTableRawNumberCell, and DataTableRichTextCell.
type DataTableCell interface {
	DataTableCellType() DataTableCellType
}

// DataTableRawTextCell is a plain-text cell in a DataTableBlock.
type DataTableRawTextCell struct {
	Type DataTableCellType `json:"type"`
	Text string            `json:"text"`
}

// DataTableCellType returns the cell variant.
func (c DataTableRawTextCell) DataTableCellType() DataTableCellType {
	return c.Type
}

// NewDataTableRawTextCell returns a raw_text cell with the given text.
func NewDataTableRawTextCell(text string) *DataTableRawTextCell {
	return &DataTableRawTextCell{Type: DataTableCellRawText, Text: text}
}

// DataTableRawNumberCell is a numeric cell in a DataTableBlock. When every cell in a
// column is a raw_number cell, Slack performs a numeric sort on that column instead of
// the default alphabetic sort. Text, when set, overrides the displayed value.
type DataTableRawNumberCell struct {
	Type  DataTableCellType `json:"type"`
	Value float64           `json:"value"`
	Text  string            `json:"text,omitempty"`
}

// DataTableCellType returns the cell variant.
func (c DataTableRawNumberCell) DataTableCellType() DataTableCellType {
	return c.Type
}

// NewDataTableRawNumberCell returns a raw_number cell with the given value.
func NewDataTableRawNumberCell(value float64) *DataTableRawNumberCell {
	return &DataTableRawNumberCell{Type: DataTableCellRawNumber, Value: value}
}

// WithText sets the display text shown in place of the numeric value.
func (c *DataTableRawNumberCell) WithText(text string) *DataTableRawNumberCell {
	c.Text = text
	return c
}

// DataTableRichTextCell is a cell holding rich text formatting. Rich text cells cannot
// appear in the header row.
type DataTableRichTextCell struct {
	Type     DataTableCellType `json:"type"`
	Elements []RichTextElement `json:"elements"`
}

// DataTableCellType returns the cell variant.
func (c DataTableRichTextCell) DataTableCellType() DataTableCellType {
	return c.Type
}

// NewDataTableRichTextCell returns a rich_text cell with the given rich text elements.
func NewDataTableRichTextCell(elements ...RichTextElement) *DataTableRichTextCell {
	return &DataTableRichTextCell{Type: DataTableCellRichText, Elements: elements}
}

// UnmarshalJSON delegates rich text element parsing to RichTextBlock so the cell handles
// the same set of inner elements (sections, lists, quotes, preformatted, unknown).
func (c *DataTableRichTextCell) UnmarshalJSON(data []byte) error {
	var rt RichTextBlock
	if err := json.Unmarshal(data, &rt); err != nil {
		return err
	}
	c.Type = DataTableCellRichText
	c.Elements = rt.Elements
	return nil
}

// DataTableBlock displays paginated tabular data with optional numeric sorting.
//
// Caption is required. Rows is an array of cell arrays; the first row is the header and
// must contain only raw_text cells. PageSize defaults to 5 (min 1, max 100) and
// RowHeaderColumnIndex defaults to 0 when omitted.
//
// More Information: https://docs.slack.dev/reference/block-kit/blocks/data-table-block/
type DataTableBlock struct {
	Type                 MessageBlockType  `json:"type"`
	BlockID              string            `json:"block_id,omitempty"`
	Caption              string            `json:"caption"`
	Rows                 [][]DataTableCell `json:"rows"`
	PageSize             int               `json:"page_size,omitempty"`
	RowHeaderColumnIndex int               `json:"row_header_column_index,omitempty"`
}

// BlockType returns the type of the block.
func (s DataTableBlock) BlockType() MessageBlockType {
	return s.Type
}

// ID returns the ID of the block.
func (s DataTableBlock) ID() string {
	return s.BlockID
}

// UnmarshalJSON parses the heterogeneous cell types in each row.
func (s *DataTableBlock) UnmarshalJSON(data []byte) error {
	var raw struct {
		Type                 MessageBlockType    `json:"type"`
		BlockID              string              `json:"block_id"`
		Caption              string              `json:"caption"`
		PageSize             int                 `json:"page_size"`
		RowHeaderColumnIndex int                 `json:"row_header_column_index"`
		Rows                 [][]json.RawMessage `json:"rows"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	rows := make([][]DataTableCell, 0, len(raw.Rows))
	for _, rawRow := range raw.Rows {
		row := make([]DataTableCell, 0, len(rawRow))
		for _, rawCell := range rawRow {
			var probe struct {
				Type DataTableCellType `json:"type"`
			}
			if err := json.Unmarshal(rawCell, &probe); err != nil {
				return err
			}
			var cell DataTableCell
			switch probe.Type {
			case DataTableCellRawText:
				cell = &DataTableRawTextCell{}
			case DataTableCellRawNumber:
				cell = &DataTableRawNumberCell{}
			case DataTableCellRichText:
				cell = &DataTableRichTextCell{}
			default:
				return fmt.Errorf("unsupported data_table cell type %q", probe.Type)
			}
			if err := json.Unmarshal(rawCell, cell); err != nil {
				return err
			}
			row = append(row, cell)
		}
		rows = append(rows, row)
	}

	s.Type = raw.Type
	s.BlockID = raw.BlockID
	s.Caption = raw.Caption
	s.PageSize = raw.PageSize
	s.RowHeaderColumnIndex = raw.RowHeaderColumnIndex
	s.Rows = rows
	return nil
}

// DataTableBlockOption configures optional fields on a new DataTableBlock.
type DataTableBlockOption func(*DataTableBlock)

// DataTableBlockOptionBlockID sets the block ID.
func DataTableBlockOptionBlockID(blockID string) DataTableBlockOption {
	return func(b *DataTableBlock) { b.BlockID = blockID }
}

// NewDataTableBlock returns a new DataTableBlock with the given caption. Add header and
// data rows with AddRow.
func NewDataTableBlock(caption string, options ...DataTableBlockOption) *DataTableBlock {
	block := &DataTableBlock{
		Type:    MBTDataTable,
		Caption: caption,
		Rows:    make([][]DataTableCell, 0),
	}
	for _, opt := range options {
		if opt != nil {
			opt(block)
		}
	}
	return block
}

// WithPageSize sets the number of rows per page (min 1, max 100).
func (s *DataTableBlock) WithPageSize(pageSize int) *DataTableBlock {
	s.PageSize = pageSize
	return s
}

// WithRowHeaderColumnIndex sets the 0-based index of the column that uniquely identifies
// each row.
func (s *DataTableBlock) WithRowHeaderColumnIndex(idx int) *DataTableBlock {
	s.RowHeaderColumnIndex = idx
	return s
}

// AddRow appends a row of cells to the DataTableBlock.
func (s *DataTableBlock) AddRow(cells ...DataTableCell) *DataTableBlock {
	s.Rows = append(s.Rows, append([]DataTableCell{}, cells...))
	return s
}
