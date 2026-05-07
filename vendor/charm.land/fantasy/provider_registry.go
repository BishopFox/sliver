package fantasy

import (
	"encoding/json"
	"fmt"
	"sync"
)

// providerDataJSON is the serialized wrapper used by the registry.
type providerDataJSON struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// UnmarshalFunc converts raw JSON into a ProviderOptionsData implementation.
type UnmarshalFunc func([]byte) (ProviderOptionsData, error)

// providerRegistry uses sync.Map for lock-free reads after initialization.
// All registrations happen in init() functions before concurrent access.
var providerRegistry sync.Map

// RegisterProviderType registers a provider type ID with its unmarshal function.
// Type IDs must be globally unique (e.g. "openai.options").
// This should only be called during package initialization (init functions).
func RegisterProviderType(typeID string, unmarshalFn UnmarshalFunc) {
	providerRegistry.Store(typeID, unmarshalFn)
}

// unmarshalProviderData routes a typed payload to the correct constructor.
func unmarshalProviderData(data []byte) (ProviderOptionsData, error) {
	var pj providerDataJSON
	if err := json.Unmarshal(data, &pj); err != nil {
		return nil, err
	}

	val, exists := providerRegistry.Load(pj.Type)
	if !exists {
		return nil, fmt.Errorf("unknown provider data type: %s", pj.Type)
	}

	unmarshalFn := val.(UnmarshalFunc) //nolint:forcetypeassert // type enforced by RegisterProviderType
	return unmarshalFn(pj.Data)
}

// unmarshalProviderDataMap is a helper for unmarshaling maps of provider data.
func unmarshalProviderDataMap(data map[string]json.RawMessage) (map[string]ProviderOptionsData, error) {
	result := make(map[string]ProviderOptionsData)
	for provider, rawData := range data {
		providerData, err := unmarshalProviderData(rawData)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal provider data for %s: %w", provider, err)
		}
		result[provider] = providerData
	}
	return result, nil
}

// UnmarshalProviderOptions unmarshals a map of provider options by type.
func UnmarshalProviderOptions(data map[string]json.RawMessage) (ProviderOptions, error) {
	return unmarshalProviderDataMap(data)
}

// UnmarshalProviderMetadata unmarshals a map of provider metadata by type.
func UnmarshalProviderMetadata(data map[string]json.RawMessage) (ProviderMetadata, error) {
	return unmarshalProviderDataMap(data)
}

// MarshalProviderType marshals provider data with a type wrapper using generics.
// To avoid infinite recursion, use the "type plain T" pattern before calling this.
//
// Usage in provider types:
//
//	func (o ProviderOptions) MarshalJSON() ([]byte, error) {
//	    type plain ProviderOptions
//	    return fantasy.MarshalProviderType(TypeProviderOptions, plain(o))
//	}
func MarshalProviderType[T any](typeID string, data T) ([]byte, error) {
	rawData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return json.Marshal(providerDataJSON{
		Type: typeID,
		Data: json.RawMessage(rawData),
	})
}

// UnmarshalProviderType unmarshals provider data without type wrapper using generics.
// To avoid infinite recursion, unmarshal to a plain type first.
// Note: This receives the inner 'data' field after type routing by the registry.
//
// Usage in provider types:
//
//	func (o *ProviderOptions) UnmarshalJSON(data []byte) error {
//	    type plain ProviderOptions
//	    var p plain
//	    if err := fantasy.UnmarshalProviderType(data, &p); err != nil {
//	        return err
//	    }
//	    *o = ProviderOptions(p)
//	    return nil
//	}
func UnmarshalProviderType[T any](data []byte, target *T) error {
	return json.Unmarshal(data, target)
}
