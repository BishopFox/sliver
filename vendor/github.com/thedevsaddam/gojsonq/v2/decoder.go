package gojsonq

import "encoding/json"

// Decoder provide contract to decode JSON using custom decoder
type Decoder interface {
	Decode(data []byte, v interface{}) error
}

// DefaultDecoder use json.Unmarshal to decode JSON
type DefaultDecoder struct{}

// Decode decodes using json.Unmarshal
func (u *DefaultDecoder) Decode(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
