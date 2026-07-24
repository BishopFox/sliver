package traffic

import "github.com/tetratelabs/wazero"

// CreateTrafficEncoder initializes a WASM traffic-encoder runtime. Wazero
// selects its compiler when the host supports executable memory and otherwise
// falls back to the portable interpreter.
func CreateTrafficEncoder(name string, wasm []byte, logger TrafficEncoderLogCallback) (*TrafficEncoder, error) {
	return createTrafficEncoder(name, wasm, logger, wazero.NewRuntimeConfig())
}
