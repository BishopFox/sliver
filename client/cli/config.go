package cli

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

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
	"fmt"
	"path/filepath"
	"sort"

	"github.com/bishopfox/sliver/client/assets"
	"github.com/bishopfox/sliver/client/forms"
)

func selectConfig(configPath string, stdinTTY bool) (string, *assets.ClientConfig, error) {
	if configPath != "" {
		config, err := loadMCPClientConfig(configPath)
		if err != nil {
			return "", nil, fmt.Errorf("load client config: %w", err)
		}
		return filepath.Base(configPath), config, nil
	}
	return selectConfigFrom(assets.GetConfigs(), stdinTTY)
}

func selectConfigFrom(configs map[string]*assets.ClientConfig, stdinTTY bool) (string, *assets.ClientConfig, error) {
	if len(configs) == 0 {
		return "", nil, fmt.Errorf("no config files found at %s", assets.GetConfigDir())
	}

	if len(configs) == 1 {
		for key, config := range configs {
			return key, config, nil
		}
	}
	if !stdinTTY {
		return "", nil, fmt.Errorf("multiple configs found; use --config to select one")
	}

	keys := make([]string, 0, len(configs))
	for key := range configs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	selection := keys[0]
	err := forms.Select("Select a server:", keys, &selection)
	if err != nil {
		return "", nil, err
	}

	return selection, configs[selection], nil
}
