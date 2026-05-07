package assets

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

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
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const spoofMetadataFileName = "spoof-metadata.yaml"

const defaultSpoofMetadataConfigTemplate = `# Sliver executable metadata spoofing configuration.
# Used by generate commands when --spoof-metadata is set.
#
# Fields that reference file content support either:
# - path: read bytes from a local file path
# - base64: inline base64 file content
# Do not set both path and base64 for the same object.
#
# Example 1: Minimal path-based metadata cloning (most common)
pe:
  source:
    name: metadata-source.exe
    path: ""

  # Optional icon donor. Reserved for future standalone icon mutation support.
  icon:
    name: icon.ico
    path: ""

  # Example 2: Inline/base64 source (uncomment and remove "path" above)
  # source:
  #   name: metadata-source.exe
  #   base64: "TVqQAAMAAAAEAAAA..."

  # Example 3: Optional PE structure overrides (advanced)
  # Applied after metadata/resources are cloned from source.
  # Omitted fields keep cloned values.
  # resource_directory:
  #   characteristics: 0
  #   time_date_stamp: 0
  #   major_version: 1
  #   minor_version: 0
  #   number_of_named_entries: 0
  #   number_of_id_entries: 0
  #
  # resource_directory_entries:
  #   - name: 1
  #     offset_to_data: 0
  #
  # resource_data_entries:
  #   - offset_to_data: 0
  #     size: 0
  #     code_page: 0
  #     reserved: 0
  #
  # export_directory:
  #   characteristics: 0
  #   time_date_stamp: 0
  #   major_version: 1
  #   minor_version: 0
  #   name: 0
  #   base: 0
  #   number_of_functions: 0
  #   number_of_names: 0
  #   address_of_functions: 0
  #   address_of_names: 0
  #   address_of_name_ordinals: 0
`

// SpoofMetadataConfig stores default metadata spoofing inputs for generate commands.
type SpoofMetadataConfig struct {
	PE *PESpoofMetadataConfig `json:"pe,omitempty" yaml:"pe,omitempty"`
}

// PESpoofMetadataConfig stores PE-specific spoofing inputs.
type PESpoofMetadataConfig struct {
	Source *SpoofMetadataDataSource `json:"source,omitempty" yaml:"source,omitempty"`
	Icon   *SpoofMetadataDataSource `json:"icon,omitempty" yaml:"icon,omitempty"`

	ResourceDirectory        *ImageResourceDirectory        `json:"resource_directory,omitempty" yaml:"resource_directory,omitempty"`
	ResourceDirectoryEntries []*ImageResourceDirectoryEntry `json:"resource_directory_entries,omitempty" yaml:"resource_directory_entries,omitempty"`
	ResourceDataEntries      []*ImageResourceDataEntry      `json:"resource_data_entries,omitempty" yaml:"resource_data_entries,omitempty"`
	ExportDirectory          *ImageExportDirectory          `json:"export_directory,omitempty" yaml:"export_directory,omitempty"`
}

// SpoofMetadataDataSource points to bytes that should be sent to the server.
// Either Path or Base64 may be set.
type SpoofMetadataDataSource struct {
	Name   string `json:"name,omitempty" yaml:"name,omitempty"`
	Path   string `json:"path,omitempty" yaml:"path,omitempty"`
	Base64 string `json:"base64,omitempty" yaml:"base64,omitempty"`
}

type ImageResourceDirectory struct {
	Characteristics      uint32 `json:"characteristics,omitempty" yaml:"characteristics,omitempty"`
	TimeDateStamp        uint32 `json:"time_date_stamp,omitempty" yaml:"time_date_stamp,omitempty"`
	MajorVersion         uint32 `json:"major_version,omitempty" yaml:"major_version,omitempty"`
	MinorVersion         uint32 `json:"minor_version,omitempty" yaml:"minor_version,omitempty"`
	NumberOfNamedEntries uint32 `json:"number_of_named_entries,omitempty" yaml:"number_of_named_entries,omitempty"`
	NumberOfIDEntries    uint32 `json:"number_of_id_entries,omitempty" yaml:"number_of_id_entries,omitempty"`
}

type ImageResourceDirectoryEntry struct {
	Name         uint32 `json:"name,omitempty" yaml:"name,omitempty"`
	OffsetToData uint32 `json:"offset_to_data,omitempty" yaml:"offset_to_data,omitempty"`
}

type ImageResourceDataEntry struct {
	OffsetToData uint32 `json:"offset_to_data,omitempty" yaml:"offset_to_data,omitempty"`
	Size         uint32 `json:"size,omitempty" yaml:"size,omitempty"`
	CodePage     uint32 `json:"code_page,omitempty" yaml:"code_page,omitempty"`
	Reserved     uint32 `json:"reserved,omitempty" yaml:"reserved,omitempty"`
}

type ImageExportDirectory struct {
	Characteristics       uint32 `json:"characteristics,omitempty" yaml:"characteristics,omitempty"`
	TimeDateStamp         uint32 `json:"time_date_stamp,omitempty" yaml:"time_date_stamp,omitempty"`
	MajorVersion          uint32 `json:"major_version,omitempty" yaml:"major_version,omitempty"`
	MinorVersion          uint32 `json:"minor_version,omitempty" yaml:"minor_version,omitempty"`
	Name                  uint32 `json:"name,omitempty" yaml:"name,omitempty"`
	Base                  uint32 `json:"base,omitempty" yaml:"base,omitempty"`
	NumberOfFunctions     uint32 `json:"number_of_functions,omitempty" yaml:"number_of_functions,omitempty"`
	NumberOfNames         uint32 `json:"number_of_names,omitempty" yaml:"number_of_names,omitempty"`
	AddressOfFunctions    uint32 `json:"address_of_functions,omitempty" yaml:"address_of_functions,omitempty"`
	AddressOfNames        uint32 `json:"address_of_names,omitempty" yaml:"address_of_names,omitempty"`
	AddressOfNameOrdinals uint32 `json:"address_of_name_ordinals,omitempty" yaml:"address_of_name_ordinals,omitempty"`
}

// SpoofMetadataConfigPath returns the spoof metadata config file path.
func SpoofMetadataConfigPath() string {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	return filepath.Join(rootDir, spoofMetadataFileName)
}

// LoadSpoofMetadataConfig loads spoof metadata config from disk, writing defaults
// if the file is missing.
func LoadSpoofMetadataConfig() (*SpoofMetadataConfig, error) {
	config := defaultSpoofMetadataConfig()
	configPath := SpoofMetadataConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return defaultSpoofMetadataConfig(), err
		}
		if err := os.WriteFile(configPath, []byte(defaultSpoofMetadataConfigTemplate), 0o600); err != nil {
			return defaultSpoofMetadataConfig(), err
		}
		data = []byte(defaultSpoofMetadataConfigTemplate)
	}
	if err := yaml.Unmarshal(data, config); err != nil {
		return defaultSpoofMetadataConfig(), err
	}

	normalizeSpoofMetadataConfig(config)
	return config, nil
}

// SaveSpoofMetadataConfig writes spoof metadata config to disk.
func SaveSpoofMetadataConfig(config *SpoofMetadataConfig) error {
	if config == nil {
		config = defaultSpoofMetadataConfig()
	}
	normalizeSpoofMetadataConfig(config)
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile(SpoofMetadataConfigPath(), data, 0o600)
}

func normalizeSpoofMetadataConfig(config *SpoofMetadataConfig) {
	if config.PE == nil {
		config.PE = &PESpoofMetadataConfig{}
	}
	if config.PE.Source == nil {
		config.PE.Source = &SpoofMetadataDataSource{Name: "metadata-source.exe"}
	}
	if config.PE.Icon == nil {
		config.PE.Icon = &SpoofMetadataDataSource{Name: "icon.ico"}
	}
}

func defaultSpoofMetadataConfig() *SpoofMetadataConfig {
	return &SpoofMetadataConfig{
		PE: &PESpoofMetadataConfig{
			Source: &SpoofMetadataDataSource{
				Name: "metadata-source.exe",
				Path: "",
			},
			Icon: &SpoofMetadataDataSource{
				Name: "icon.ico",
				Path: "",
			},
		},
	}
}
