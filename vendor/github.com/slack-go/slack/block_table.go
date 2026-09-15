package slack

import (
	"encoding/json"
	"fmt"
)

// TableCellType identifies the variant of a cell inside a TableBlock row.
type TableCellType string

const (
	TableCellRawText   TableCellType = "raw_text"
	TableCellRawNumber TableCellType = "raw_number"
	TableCellRichText  TableCellType = "rich_text"
)

// TableCell is implemented by every cell type valid inside a TableBlock row:
// TableRichTextCell, TableRawTextCell, and TableRawNumberCell. A nil TableCell
// represents an empty cell, which Slack sends as null (common in user-pasted tables).
type TableCell interface {
	TableCellType() TableCellType
}

// TableRichTextCell is a cell holding rich text formatting. Slack uses rich_text cells
// for styled content such as the bold header row of a pasted table.
type TableRichTextCell struct {
	Type     TableCellType     `json:"type"`
	Elements []RichTextElement `json:"elements"`
}

// TableCellType returns the cell variant.
func (c TableRichTextCell) TableCellType() TableCellType {
	return c.Type
}

// NewTableRichTextCell returns a rich_text cell with the given rich text elements.
func NewTableRichTextCell(elements ...RichTextElement) *TableRichTextCell {
	return &TableRichTextCell{Type: TableCellRichText, Elements: elements}
}

// UnmarshalJSON delegates rich text element parsing to RichTextBlock so the cell handles
// the same set of inner elements (sections, lists, quotes, preformatted, unknown).
func (c *TableRichTextCell) UnmarshalJSON(data []byte) error {
	var rt RichTextBlock
	if err := json.Unmarshal(data, &rt); err != nil {
		return err
	}
	c.Type = TableCellRichText
	c.Elements = rt.Elements
	return nil
}

// TableRawTextCell is a plain-text cell in a TableBlock. Slack sends raw_text cells for
// non-numeric data, including text pasted from a spreadsheet.
type TableRawTextCell struct {
	Type TableCellType `json:"type"`
	Text string        `json:"text"`
}

// TableCellType returns the cell variant.
func (c TableRawTextCell) TableCellType() TableCellType {
	return c.Type
}

// NewTableRawTextCell returns a raw_text cell with the given text.
func NewTableRawTextCell(text string) *TableRawTextCell {
	return &TableRawTextCell{Type: TableCellRawText, Text: text}
}

// TableRawNumberCell is a numeric cell in a TableBlock. Text, when set, overrides the
// displayed value.
//
// raw_number cells are receive-only: Slack emits them for numeric columns in user-pasted
// tables, but chat.postMessage rejects them. To post a number, use a TableRawTextCell. See
// TableBlock for the full list of postable vs receive-only cell types.
type TableRawNumberCell struct {
	Type  TableCellType `json:"type"`
	Value float64       `json:"value"`
	Text  string        `json:"text,omitempty"`
}

// TableCellType returns the cell variant.
func (c TableRawNumberCell) TableCellType() TableCellType {
	return c.Type
}

// NewTableRawNumberCell returns a raw_number cell with the given value.
func NewTableRawNumberCell(value float64) *TableRawNumberCell {
	return &TableRawNumberCell{Type: TableCellRawNumber, Value: value}
}

// WithText sets the display text shown in place of the numeric value.
func (c *TableRawNumberCell) WithText(text string) *TableRawNumberCell {
	c.Text = text
	return c
}

// TableBlock defines a block that lets you use a table to display your data.
//
// Rows is an array of cell arrays. Each cell is a TableRichTextCell, TableRawTextCell,
// or TableRawNumberCell; a nil cell represents an empty cell (Slack sends null).
//
// Not every cell type can be posted. chat.postMessage validates a table cell against two
// schemas only: raw_text (TableRawTextCell, requires "text") and rich_text
// (TableRichTextCell, requires "elements"). TableRawNumberCell and nil cells are produced
// by Slack on output — numeric columns and empty cells in user-pasted tables — but the API
// rejects them on input with invalid_blocks. So you may receive all cell types, but post
// only raw_text and rich_text; render a number you want to send as a raw_text cell.
//
// More Information: https://docs.slack.dev/reference/block-kit/blocks/table-block/
type TableBlock struct {
	Type           MessageBlockType `json:"type"`
	BlockID        string           `json:"block_id,omitempty"`
	Rows           [][]TableCell    `json:"rows"`
	ColumnSettings []ColumnSetting  `json:"column_settings,omitempty"`
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

// UnmarshalJSON parses the heterogeneous cell types in each row. A null cell is decoded
// as a nil TableCell.
func (s *TableBlock) UnmarshalJSON(data []byte) error {
	var raw struct {
		Type           MessageBlockType    `json:"type"`
		BlockID        string              `json:"block_id"`
		ColumnSettings []ColumnSetting     `json:"column_settings"`
		Rows           [][]json.RawMessage `json:"rows"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	rows := make([][]TableCell, 0, len(raw.Rows))
	for _, rawRow := range raw.Rows {
		row := make([]TableCell, 0, len(rawRow))
		for _, rawCell := range rawRow {
			if len(rawCell) == 0 || string(rawCell) == "null" {
				row = append(row, nil)
				continue
			}
			var probe struct {
				Type TableCellType `json:"type"`
			}
			if err := json.Unmarshal(rawCell, &probe); err != nil {
				return err
			}
			var cell TableCell
			switch probe.Type {
			case TableCellRawText:
				cell = &TableRawTextCell{}
			case TableCellRawNumber:
				cell = &TableRawNumberCell{}
			case TableCellRichText:
				cell = &TableRichTextCell{}
			default:
				return fmt.Errorf("unsupported table cell type %q", probe.Type)
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
	s.ColumnSettings = raw.ColumnSettings
	s.Rows = rows
	return nil
}

// WithColumnSettings sets the column settings for the Table Block
func (s *TableBlock) WithColumnSettings(columnSettings ...ColumnSetting) *TableBlock {
	s.ColumnSettings = columnSettings
	return s
}

// AddRow adds a new row of cells to the Table Block
func (s *TableBlock) AddRow(cells ...TableCell) *TableBlock {
	s.Rows = append(s.Rows, append([]TableCell{}, cells...))
	return s
}

// NewTableBlock returns an instance of a Table Block type
func NewTableBlock(blockID string) *TableBlock {
	return &TableBlock{
		Type:    MBTTable,
		BlockID: blockID,
		Rows:    make([][]TableCell, 0),
	}
}
