package extension

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

// WasmExtensionOption configures a Wasm extension runtime without changing
// the backwards-compatible NewWasmExtension signature.
type WasmExtensionOption func(*wasmExtensionConfig)

type wasmExtensionConfig struct {
	memoryFSOptions []WasmMemoryFSOption
}

// WithReadOnlyMemFS configures the extension's /memfs namespace as read-only.
func WithReadOnlyMemFS() WasmExtensionOption {
	return func(config *wasmExtensionConfig) {
		config.memoryFSOptions = append(config.memoryFSOptions, WithWasmMemoryFSReadOnly())
	}
}

func applyWasmExtensionOptions(options []WasmExtensionOption) wasmExtensionConfig {
	config := wasmExtensionConfig{}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	return config
}
