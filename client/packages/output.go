package packages

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
)

/*
Sliver Implant Framework
Copyright (C) 2024  Bishop Fox

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
const (
	KeyValueSchemaName        = "key-value"
	GroupedKeyValueSchemaName = "grouped-key-value"
)

var validSchemas []string = []string{KeyValueSchemaName, GroupedKeyValueSchemaName}

type OutputSchema struct {
	Name       string   `json:"name"`
	RawColumns string   `json:"columns"`
	GroupBy    string   `json:"group_by"`
	cols       []string `json:"-"`
}

func (schema *OutputSchema) IngestColumns() {
	schema.cols = strings.Split(schema.RawColumns, ",")
}

func (schema *OutputSchema) Columns() []string {
	return schema.cols
}

type PackageOutput interface {
	// The names of columns followed by any other strings needed, like the data itself
	// Each type that implements this interface will have to make sure the correct number
	// of strings are provided
	IngestData([]byte, []string, ...string) error
	// Renders the data into a table
	CreateTable() string
	Name() string
}

func IsValidSchemaType(name string) bool {
	return slices.Contains(validSchemas, name)
}

func GetNewPackageOutput(name string) PackageOutput {
	switch name {
	case KeyValueSchemaName:
		return &KeyValuePairs{}
	case GroupedKeyValueSchemaName:
		return &GroupedKeyValuePairs{}
	default:
		return nil
	}
}

/*
This output schema is suitable for data that is an array of key-value pairs
*/
type KeyValueData map[string]string
type KeyValuePairs struct {
	columnOrder []string
	data        []KeyValueData
	// This field is optional
	columnToGroupBy string
}

/*
This output schema is suitable for one or more key-values pairs grouped
by a group name. This schema assumes the data is already grouped and
does not perform the grouping itself.
*/
type GroupedKeyValuePairs struct {
	groupedBy   string
	columnOrder []string
	data        map[string][]KeyValueData
}

func (kvd *KeyValueData) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*kvd = make(KeyValueData, len(raw))
	for key, value := range raw {
		key = strings.ToLower(key)
		switch v := value.(type) {
		case float64:
			(*kvd)[key] = strconv.FormatFloat(v, 'f', -1, 64)
		case string:
			// Newlines and extra whitespace mess up the table
			(*kvd)[key] = strings.TrimSpace(v)
		case bool:
			(*kvd)[key] = strconv.FormatBool(v)
		default:
			return fmt.Errorf("unsupported type %T", v)
		}
	}

	return nil
}

func (kvp *KeyValuePairs) IngestData(data []byte, order []string, arguments ...string) error {
	if len(arguments) > 1 {
		// Only an optional a column to group by
		return fmt.Errorf("unexpected number of arguments - expected only an optional column name to group by")
	}
	groupByColumn := ""
	if len(arguments) == 1 {
		groupByColumn = arguments[0]
	}
	err := json.Unmarshal(data, &kvp.data)
	if err != nil {
		return err
	}

	if len(kvp.data) == 0 {
		return nil
	}

	// If the wrong columns were provided, there will be empty cells
	kvp.columnOrder = order
	// If an invalid column name is supplied, it will have no effect
	kvp.columnToGroupBy = groupByColumn
	return nil
}

func (kvp *KeyValuePairs) CreateTable() string {
	if len(kvp.data) == 0 {
		return ""
	}

	t := table.NewWriter()

	header := table.Row{}
	for _, column := range kvp.columnOrder {
		header = append(header, column)
	}

	t.AppendHeader(header)
	for _, dataRow := range kvp.data {
		tableRow := table.Row{}

		for _, column := range kvp.columnOrder {
			column = strings.ToLower(column)
			tableRow = append(tableRow, dataRow[column])
		}
		t.AppendRow(tableRow)
	}

	t.SetColumnConfigs([]table.ColumnConfig{{Name: kvp.columnToGroupBy, AutoMerge: true}})
	if kvp.columnToGroupBy != "" {
		t.SortBy([]table.SortBy{{Name: kvp.columnToGroupBy, Mode: table.Asc}})
	} else {
		t.SortBy([]table.SortBy{{Number: 1, Mode: table.Asc}})
	}

	return t.Render()
}

func (kvp *KeyValuePairs) Name() string {
	return KeyValueSchemaName
}

func (gkvp *GroupedKeyValuePairs) IngestData(data []byte, order []string, arguments ...string) error {
	if len(arguments) != 1 {
		return fmt.Errorf("unexpected number of arguments - a grouped by column name")
	}

	groupedBy := arguments[0]

	err := json.Unmarshal(data, &gkvp.data)
	if err != nil {
		return err
	}

	if len(gkvp.data) == 0 {
		return nil
	}

	// If the wrong columns were provided, there will be empty cells
	gkvp.columnOrder = order
	// If an invalid grouped by column is provided, the table will not render correctly,
	// but there will not be any panics
	gkvp.groupedBy = groupedBy

	return nil
}

func (gkvp *GroupedKeyValuePairs) CreateTable() string {
	if len(gkvp.data) == 0 {
		return ""
	}

	t := table.NewWriter()

	header := table.Row{}
	for _, column := range gkvp.columnOrder {
		header = append(header, column)
	}

	t.AppendHeader(header)
	groupColumn := strings.ToLower(gkvp.groupedBy)
	for groupName, groupData := range gkvp.data {
		for _, group := range groupData {
			tableRow := table.Row{}
			for _, column := range gkvp.columnOrder {
				column = strings.ToLower(column)
				if column == groupColumn {
					tableRow = append(tableRow, groupName)
				} else {
					tableRow = append(tableRow, group[column])
				}
			}
			t.AppendRow(tableRow)
		}
	}

	t.SetColumnConfigs([]table.ColumnConfig{{Name: gkvp.groupedBy, AutoMerge: true}})

	if gkvp.groupedBy != "" {
		t.SortBy([]table.SortBy{{Name: gkvp.groupedBy, Mode: table.Asc}})
	} else {
		t.SortBy([]table.SortBy{{Number: 1, Mode: table.Asc}})
	}

	return t.Render()
}

func (gkvp *GroupedKeyValuePairs) Name() string {
	return GroupedKeyValueSchemaName
}
