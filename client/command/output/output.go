package output

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox

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

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// OutputFormat represents the supported output formats.
type OutputFormat string

const (
	FormatText OutputFormat = "text"
	FormatJSON OutputFormat = "json"
	FormatYAML OutputFormat = "yaml"
)

// GetOutputFormat resolves the requested output format from the command's
// flags. Shorthand flags (--json, --yaml) take precedence over the
// --output-format flag. Returns FormatText when no format is requested or
// the flag is not bound to the command.
func GetOutputFormat(cmd *cobra.Command) OutputFormat {
	if cmd == nil {
		return FormatText
	}
	if f := cmd.Flags().Lookup("json"); f != nil && f.Value.String() == "true" {
		return FormatJSON
	}
	if f := cmd.Flags().Lookup("yaml"); f != nil && f.Value.String() == "true" {
		return FormatYAML
	}
	if f := cmd.Flags().Lookup("output-format"); f != nil {
		switch OutputFormat(f.Value.String()) {
		case FormatJSON:
			return FormatJSON
		case FormatYAML:
			return FormatYAML
		default:
			return FormatText
		}
	}
	return FormatText
}

// PrintStructured renders v as JSON or YAML to stdout.
func PrintStructured(v any, format OutputFormat) error {
	switch format {
	case FormatJSON:
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return fmt.Errorf("json marshal: %w", err)
		}
		fmt.Println(string(data))
	case FormatYAML:
		data, err := yaml.Marshal(v)
		if err != nil {
			return fmt.Errorf("yaml marshal: %w", err)
		}
		fmt.Print(string(data))
	}
	return nil
}

// BindOutputFlags adds --output-format, --json, and --yaml flags to a command.
func BindOutputFlags(cmd *cobra.Command) {
	cmd.Flags().String("output-format", "text", "output format: text, json, or yaml")
	cmd.Flags().Bool("json", false, "output as JSON (shorthand for --output-format json)")
	cmd.Flags().Bool("yaml", false, "output as YAML (shorthand for --output-format yaml)")
}
