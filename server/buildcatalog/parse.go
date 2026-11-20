package buildcatalog

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"gopkg.in/yaml.v3"
)

const (
	// MaxSpecYAMLBytes limits the size of a catalog spec document.
	MaxSpecYAMLBytes = 1 << 20
	maxSafeInteger   = int64(9007199254740991)
)

var (
	canonicalNameRE = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[._-][a-z0-9]+)*(?:/[a-z][a-z0-9]*(?:[._-][a-z0-9]+)*)+$`)
	versionRE       = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-(?:0|[1-9][0-9]*|[0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*))*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)
	digestRE        = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	identifierRE    = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	optionNameRE    = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)
	mageTargetRE    = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*$`)
	decimalIntRE    = regexp.MustCompile(`^-?(?:0|[1-9][0-9]*)$`)
)

type parsedSpec struct {
	kind      string
	reference Reference
	raw       []byte
	core      *CoreSpec
	transform *TransformSpec
}

func parseSpec(raw []byte) (parsedSpec, error) {
	if len(raw) == 0 || len(raw) > MaxSpecYAMLBytes {
		return parsedSpec{}, fmt.Errorf("spec YAML size must be 1..%d bytes", MaxSpecYAMLBytes)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return parsedSpec{}, fmt.Errorf("decode spec YAML: %w", err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return parsedSpec{}, errors.New("spec YAML contains multiple documents")
		}
		return parsedSpec{}, fmt.Errorf("decode trailing YAML: %w", err)
	}
	if document.Kind != yaml.DocumentNode || len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return parsedSpec{}, errors.New("spec YAML must contain one mapping document")
	}
	if err := validateYAMLNode(document.Content[0]); err != nil {
		return parsedSpec{}, err
	}
	root := document.Content[0]
	parsed, err := decodeSpecFields(root)
	if err != nil {
		return parsedSpec{}, err
	}
	unsignedJSON, err := canonicalJSON(root, true)
	if err != nil {
		return parsedSpec{}, err
	}
	digest := sha256.Sum256(unsignedJSON)
	actual := "sha256:" + hex.EncodeToString(digest[:])
	if parsed.reference.Digest != actual {
		return parsedSpec{}, fmt.Errorf("spec digest mismatch: got %s, calculated %s", parsed.reference.Digest, actual)
	}
	parsed.raw = append([]byte(nil), raw...)
	return parsed, nil
}

func decodeSpecFields(root *yaml.Node) (parsedSpec, error) {
	fullJSON, err := canonicalJSON(root, false)
	if err != nil {
		return parsedSpec{}, err
	}
	var header struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(fullJSON, &header); err != nil {
		return parsedSpec{}, fmt.Errorf("decode spec kind: %w", err)
	}
	if err := validateRequiredFields(root, header.Kind); err != nil {
		return parsedSpec{}, err
	}
	parsed := parsedSpec{kind: header.Kind}
	switch header.Kind {
	case "CoreSpec":
		var spec CoreSpec
		if err := decodeStrict(fullJSON, &spec); err != nil {
			return parsedSpec{}, err
		}
		if err := validateCoreSpec(&spec); err != nil {
			return parsedSpec{}, err
		}
		parsed.reference, parsed.core = spec.Metadata, &spec
	case "TransformSpec":
		var spec TransformSpec
		if err := decodeStrict(fullJSON, &spec); err != nil {
			return parsedSpec{}, err
		}
		if err := validateTransformSpec(&spec); err != nil {
			return parsedSpec{}, err
		}
		parsed.reference, parsed.transform = spec.Metadata, &spec
	default:
		return parsedSpec{}, fmt.Errorf("unsupported spec kind %q", header.Kind)
	}
	return parsed, nil
}

// RestrictedYAMLJSON decodes one mapping document using the same scalar,
// duplicate-key, alias, and merge restrictions as immutable catalog specs.
// Profile documents use this before strict field decoding.
func RestrictedYAMLJSON(raw []byte, maxBytes int) ([]byte, error) {
	if len(raw) == 0 || len(raw) > maxBytes {
		return nil, fmt.Errorf("YAML size must be 1..%d bytes", maxBytes)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("decode YAML: %w", err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("YAML contains multiple documents")
		}
		return nil, fmt.Errorf("decode trailing YAML: %w", err)
	}
	if document.Kind != yaml.DocumentNode || len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return nil, errors.New("YAML must contain one mapping document")
	}
	if err := validateYAMLNode(document.Content[0]); err != nil {
		return nil, err
	}
	return canonicalJSON(document.Content[0], false)
}

func decodeStrict[T any](data []byte, target *T) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode spec fields: %w", err)
	}
	return nil
}

func yamlField(node *yaml.Node, name string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for index := 0; index < len(node.Content); index += 2 {
		if node.Content[index].Value == name {
			return node.Content[index+1]
		}
	}
	return nil
}

func requireYAMLFields(node *yaml.Node, path string, fields ...string) error {
	if node == nil || node.Kind != yaml.MappingNode {
		return fmt.Errorf("%s must be a mapping", path)
	}
	for _, field := range fields {
		if yamlField(node, field) == nil {
			return fmt.Errorf("%s.%s is required", path, field)
		}
	}
	return nil
}

func validateRequiredFields(root *yaml.Node, kind string) error {
	if err := requireYAMLFields(root, "spec", "apiVersion", "kind", "metadata", "executor", "options", "capabilities"); err != nil {
		return err
	}
	if err := requireYAMLFields(yamlField(root, "metadata"), "metadata", "name", "version", "digest"); err != nil {
		return err
	}
	if err := requireYAMLFields(yamlField(root, "executor"), "executor", "id"); err != nil {
		return err
	}
	if kind == "CoreSpec" {
		if err := validateCoreFields(root); err != nil {
			return err
		}
	}
	if err := validateOptionFields(yamlField(root, "options")); err != nil {
		return err
	}
	capabilities := yamlField(root, "capabilities")
	if capabilities.Kind != yaml.SequenceNode {
		return errors.New("capabilities must be a sequence")
	}
	for index, capability := range capabilities.Content {
		if err := validateCapabilityFields(capability, kind, fmt.Sprintf("capabilities[%d]", index)); err != nil {
			return err
		}
	}
	return nil
}

func validateCoreFields(root *yaml.Node) error {
	if err := requireYAMLFields(root, "spec", "runtime"); err != nil {
		return err
	}
	if err := requireYAMLFields(yamlField(root, "runtime"), "runtime", "protocol", "commandSource"); err != nil {
		return err
	}
	if source := yamlField(root, "source"); source != nil {
		if err := requireYAMLFields(source, "source", "mode"); err != nil {
			return err
		}
	}
	if builder := yamlField(root, "builder"); builder != nil {
		if err := requireYAMLFields(builder, "builder", "engine", "magefile", "target"); err != nil {
			return err
		}
	}
	return nil
}

func validateOptionFields(options *yaml.Node) error {
	if options.Kind != yaml.SequenceNode {
		return errors.New("options must be a sequence")
	}
	for index, option := range options.Content {
		if err := requireYAMLFields(option, fmt.Sprintf("options[%d]", index), "name", "type", "required"); err != nil {
			return err
		}
		if description := yamlField(option, "description"); description != nil && description.Value == "" {
			return fmt.Errorf("options[%d].description cannot be empty", index)
		}
	}
	return nil
}

func validateCapabilityFields(capability *yaml.Node, kind, path string) error {
	required := []string{"targets", "toolchain"}
	switch kind {
	case "CoreSpec":
		required = append(required, "mode", "protocols", "output")
	case "TransformSpec":
		required = append(required, "input", "output")
	}
	if err := requireYAMLFields(capability, path, required...); err != nil {
		return err
	}
	targets := yamlField(capability, "targets")
	if targets.Kind != yaml.SequenceNode {
		return fmt.Errorf("%s.targets must be a sequence", path)
	}
	for targetIndex, target := range targets.Content {
		if err := requireYAMLFields(target, fmt.Sprintf("%s.targets[%d]", path, targetIndex), "os", "arch"); err != nil {
			return err
		}
	}
	toolchain := yamlField(capability, "toolchain")
	if err := requireYAMLFields(toolchain, path+".toolchain", "driver", "requirements"); err != nil {
		return err
	}
	if resources := yamlField(capability, "resources"); resources != nil {
		if err := requireYAMLFields(resources, path+".resources", "estimatedPeakMemoryBytes"); err != nil {
			return err
		}
	}
	switch kind {
	case "CoreSpec":
		return validateCoreOutputFields(yamlField(capability, "output"), path+".output", yamlField(toolchain, "driver").Value)
	case "TransformSpec":
		return validateTransformArtifactFields(capability, path)
	}
	return nil
}

func validateCoreOutputFields(output *yaml.Node, path, driver string) error {
	if err := requireYAMLFields(output, path, "artifact", "operatorFormats"); err != nil {
		return err
	}
	artifact := yamlField(output, "artifact")
	if err := requireYAMLFields(artifact, path+".artifact", "kind", "container"); err != nil {
		return err
	}
	if driver == "go" && yamlField(artifact, "goBuildMode") == nil {
		return fmt.Errorf("%s.artifact.goBuildMode is required for Go", path)
	}
	return rejectRawGoMode(artifact, path+".artifact")
}

func validateTransformArtifactFields(capability *yaml.Node, path string) error {
	for _, name := range []string{"input", "output"} {
		artifact := yamlField(capability, name)
		if err := requireYAMLFields(artifact, path+"."+name, "kind", "container"); err != nil {
			return err
		}
		if err := rejectRawGoMode(artifact, path+"."+name); err != nil {
			return err
		}
	}
	return nil
}

func rejectRawGoMode(artifact *yaml.Node, path string) error {
	kind := yamlField(artifact, "kind")
	if kind != nil && (kind.Value == "shellcode" || kind.Value == "stage") && yamlField(artifact, "goBuildMode") != nil {
		return fmt.Errorf("%s.goBuildMode is forbidden for raw artifacts", path)
	}
	return nil
}

func validateYAMLNode(node *yaml.Node) error {
	switch node.Kind {
	case yaml.AliasNode:
		return errors.New("YAML aliases are not permitted")
	case yaml.MappingNode:
		return validateYAMLMapping(node)
	case yaml.SequenceNode:
		if node.Tag != "!!seq" {
			return fmt.Errorf("unsupported YAML sequence tag %q", node.Tag)
		}
		for _, child := range node.Content {
			if err := validateYAMLNode(child); err != nil {
				return err
			}
		}
	case yaml.ScalarNode:
		return validateYAMLScalar(node)
	default:
		return fmt.Errorf("unsupported YAML node kind %d", node.Kind)
	}
	return nil
}

func validateYAMLMapping(node *yaml.Node) error {
	if node.Tag != "!!map" {
		return fmt.Errorf("unsupported YAML mapping tag %q", node.Tag)
	}
	keys := make(map[string]struct{}, len(node.Content)/2)
	foldedKeys := make(map[string]string, len(node.Content)/2)
	for index := 0; index < len(node.Content); index += 2 {
		key, value := node.Content[index], node.Content[index+1]
		if key.Kind != yaml.ScalarNode || key.Tag != "!!str" || key.Value == "<<" {
			return errors.New("YAML mapping keys must be plain strings without merges")
		}
		if _, exists := keys[key.Value]; exists {
			return fmt.Errorf("duplicate YAML key %q", key.Value)
		}
		folded := cases.Fold().String(key.Value)
		if prior, exists := foldedKeys[folded]; exists {
			return fmt.Errorf("case-equivalent YAML keys %q and %q", prior, key.Value)
		}
		foldedKeys[folded] = key.Value
		keys[key.Value] = struct{}{}
		if err := validateYAMLNode(value); err != nil {
			return fmt.Errorf("%s: %w", key.Value, err)
		}
	}
	return nil
}

func validateYAMLScalar(node *yaml.Node) error {
	if !utf8.ValidString(node.Value) {
		return errors.New("invalid Unicode scalar")
	}
	switch node.Tag {
	case "!!str":
		if node.Style == 0 {
			switch strings.ToLower(node.Value) {
			case "yes", "no", "on", "off", "~":
				return fmt.Errorf("ambiguous YAML scalar %q must be quoted", node.Value)
			}
		}
	case "!!bool":
		if node.Value != "true" && node.Value != "false" {
			return fmt.Errorf("ambiguous YAML boolean %q", node.Value)
		}
	case "!!int":
		if !decimalIntRE.MatchString(node.Value) {
			return fmt.Errorf("ambiguous YAML integer %q", node.Value)
		}
		number, err := strconv.ParseInt(node.Value, 10, 64)
		if err != nil || number < -maxSafeInteger || number > maxSafeInteger {
			return fmt.Errorf("YAML integer %q exceeds I-JSON safe range", node.Value)
		}
	case "!!float":
		return errors.New("floating-point spec values require an RFC 8785 serializer and are not yet supported")
	case "!!null":
		return errors.New("null values are not permitted in build specs")
	default:
		return fmt.Errorf("unsupported YAML scalar tag %q", node.Tag)
	}
	return nil
}

func canonicalJSON(root *yaml.Node, omitDigest bool) ([]byte, error) {
	var output bytes.Buffer
	if err := writeCanonical(&output, root, "", omitDigest); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func writeCanonical(output *bytes.Buffer, node *yaml.Node, path string, omitDigest bool) error {
	switch node.Kind {
	case yaml.MappingNode:
		return writeCanonicalMapping(output, node, path, omitDigest)
	case yaml.SequenceNode:
		output.WriteByte('[')
		for index, child := range node.Content {
			if index != 0 {
				output.WriteByte(',')
			}
			if err := writeCanonical(output, child, path, omitDigest); err != nil {
				return err
			}
		}
		output.WriteByte(']')
	case yaml.ScalarNode:
		switch node.Tag {
		case "!!str":
			writeJSONString(output, node.Value)
		case "!!bool", "!!null":
			output.WriteString(node.Value)
		case "!!int":
			number, _ := strconv.ParseInt(node.Value, 10, 64)
			output.WriteString(strconv.FormatInt(number, 10))
		default:
			return fmt.Errorf("unsupported YAML scalar tag %q", node.Tag)
		}
	default:
		return fmt.Errorf("unsupported YAML node kind %d", node.Kind)
	}
	return nil
}

func writeCanonicalMapping(output *bytes.Buffer, node *yaml.Node, path string, omitDigest bool) error {
	values := make(map[string]*yaml.Node, len(node.Content)/2)
	keys := make([]string, 0, len(node.Content)/2)
	for index := 0; index < len(node.Content); index += 2 {
		key := node.Content[index].Value
		if omitDigest && path == "metadata" && key == "digest" {
			continue
		}
		keys = append(keys, key)
		values[key] = node.Content[index+1]
	}
	sort.Strings(keys)
	output.WriteByte('{')
	for index, key := range keys {
		if index != 0 {
			output.WriteByte(',')
		}
		writeJSONString(output, key)
		output.WriteByte(':')
		childPath := key
		if path != "" {
			childPath = path + "." + key
		}
		if err := writeCanonical(output, values[key], childPath, omitDigest); err != nil {
			return err
		}
	}
	output.WriteByte('}')
	return nil
}

func writeJSONString(output *bytes.Buffer, value string) {
	output.WriteByte('"')
	for _, character := range value {
		switch character {
		case '"':
			output.WriteString(`\"`)
		case '\\':
			output.WriteString(`\\`)
		case '\b':
			output.WriteString(`\b`)
		case '\t':
			output.WriteString(`\t`)
		case '\n':
			output.WriteString(`\n`)
		case '\f':
			output.WriteString(`\f`)
		case '\r':
			output.WriteString(`\r`)
		default:
			if character < 0x20 {
				fmt.Fprintf(output, `\u%04x`, character)
			} else {
				output.WriteRune(character)
			}
		}
	}
	output.WriteByte('"')
}

func safeFloat(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= -float64(maxSafeInteger) && value <= float64(maxSafeInteger)
}
