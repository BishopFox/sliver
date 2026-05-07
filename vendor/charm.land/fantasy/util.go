package fantasy

import "github.com/go-viper/mapstructure/v2"

// Opt creates a pointer to the given value.
func Opt[T any](v T) *T {
	return &v
}

// ParseOptions parses the given options map into the provided struct.
func ParseOptions[T any](options map[string]any, m *T) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "json",
		Result:  m,
	})
	if err != nil {
		return err
	}
	return decoder.Decode(options)
}
