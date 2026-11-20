package rpc

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/server/buildtransform"
)

const maxBuildInputBytes = 128 << 20
const maxBuildInputsBytes = 256 << 20

type buildInputKey struct {
	kind string
	name string
}

// buildInputResolver copies ephemeral request material into a per-build map.
// The neutral plan and record contain only typed references, never these
// bytes. The returned cleanup overwrites the private copies after execution.
func buildInputResolver(inputs []*clientpb.BuildInput) (buildtransform.ResolveInput, func(), error) {
	if len(inputs) == 0 {
		return nil, func() {}, nil
	}
	material := make(map[buildInputKey][]byte, len(inputs))
	cleanup := func() {
		for key, value := range material {
			clear(value)
			delete(material, key)
		}
	}
	var total int64
	for _, input := range inputs {
		if input == nil {
			cleanup()
			return nil, nil, errors.New("nil build input")
		}
		kind, name, data := input.GetKind(), input.GetName(), input.GetData()
		switch kind {
		case "secret", "input-artifact", "metadata-config":
		default:
			cleanup()
			return nil, nil, fmt.Errorf("unsupported build input kind %q", kind)
		}
		if name == "" || len(name) > 128 || !utf8.ValidString(name) || strings.TrimSpace(name) != name ||
			strings.ContainsAny(name, "\x00\r\n") {
			cleanup()
			return nil, nil, errors.New("invalid build input name")
		}
		limit := maxBuildInputBytes
		if kind == "secret" {
			limit = 256
		}
		if len(data) == 0 || len(data) > limit {
			cleanup()
			return nil, nil, fmt.Errorf("build input %q has invalid byte length", name)
		}
		total += int64(len(data))
		if total > maxBuildInputsBytes {
			cleanup()
			return nil, nil, errors.New("build inputs exceed total byte limit")
		}
		key := buildInputKey{kind: kind, name: name}
		if _, exists := material[key]; exists {
			cleanup()
			return nil, nil, fmt.Errorf("duplicate build input %q", name)
		}
		material[key] = bytes.Clone(data)
	}
	resolve := func(kind, name string) ([]byte, error) {
		value, exists := material[buildInputKey{kind: kind, name: name}]
		if !exists {
			return nil, errors.New("build input is unavailable")
		}
		return bytes.Clone(value), nil
	}
	return resolve, cleanup, nil
}
