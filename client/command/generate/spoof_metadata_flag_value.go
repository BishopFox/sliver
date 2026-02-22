package generate

import "strings"

type spoofMetadataFlagValue struct {
	value string
}

func newSpoofMetadataFlagValue() *spoofMetadataFlagValue {
	return &spoofMetadataFlagValue{}
}

func (s *spoofMetadataFlagValue) Set(value string) error {
	value = strings.TrimSpace(value)
	switch strings.ToLower(value) {
	case "", "true":
		s.value = "true"
	case "false":
		s.value = "false"
	default:
		s.value = value
	}
	return nil
}

func (s *spoofMetadataFlagValue) String() string {
	return s.value
}

func (s *spoofMetadataFlagValue) Type() string {
	return "bool"
}

func (s *spoofMetadataFlagValue) IsBoolFlag() bool {
	return true
}
