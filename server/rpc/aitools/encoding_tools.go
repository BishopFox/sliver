package aitools

import (
	"context"
	stdbase32 "encoding/base32"
	stdbase64 "encoding/base64"
	"fmt"
	"strings"
	"unicode/utf8"

	serverai "github.com/bishopfox/sliver/server/ai"
	utilencoders "github.com/bishopfox/sliver/util/encoders"
)

const aiEncodingDefaultMaxBytes = 64 * 1024

type encodingInputArgs struct {
	Text       *string `json:"text,omitempty"`
	DataBase64 *string `json:"data_base64,omitempty"`
}

type encodedValueArgs struct {
	Value    *string `json:"value,omitempty"`
	MaxBytes *int64  `json:"max_bytes,omitempty"`
}

type gzipDecodeArgs struct {
	DataBase64 *string `json:"data_base64,omitempty"`
	MaxBytes   *int64  `json:"max_bytes,omitempty"`
}

type encodingValueResult struct {
	Encoding  string `json:"encoding"`
	Operation string `json:"operation"`
	Variant   string `json:"variant,omitempty"`
	Value     string `json:"value"`
	ByteLen   int    `json:"byte_len"`
}

type encodingBytesResult struct {
	Encoding   string `json:"encoding"`
	Operation  string `json:"operation"`
	Variant    string `json:"variant,omitempty"`
	ByteLen    int    `json:"byte_len"`
	SHA256     string `json:"sha256,omitempty"`
	DataBase64 string `json:"data_base64,omitempty"`
	Text       string `json:"text,omitempty"`
}

type standardBase32Encoder struct{}

type standardBase64Encoder struct{}

func encodingToolDefinitions() []serverai.AgenticToolDefinition {
	return []serverai.AgenticToolDefinition{
		{
			Name:        "base32_encode",
			Description: "Encode bytes using standard RFC 4648 base32 with padding. Provide either text or data_base64 input.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"data_base64": map[string]any{"type": "string", "description": "Optional standard base64 wrapper for arbitrary input bytes."},
					"text":        map[string]any{"type": "string", "description": "Optional UTF-8 text to encode."},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "base32_decode",
			Description: "Decode standard RFC 4648 base32. Returns the decoded bytes as standard data_base64 plus UTF-8 text when valid.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"max_bytes": map[string]any{"type": "integer", "description": "Optional maximum decoded byte length. Defaults to 65536."},
					"value":     map[string]any{"type": "string", "description": "Base32 value to decode."},
				},
				"required":             []string{"value"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "base64_encode",
			Description: "Encode bytes using standard RFC 4648 base64 with padding. Provide either text or data_base64 input.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"data_base64": map[string]any{"type": "string", "description": "Optional standard base64 wrapper for arbitrary input bytes."},
					"text":        map[string]any{"type": "string", "description": "Optional UTF-8 text to encode."},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "base64_decode",
			Description: "Decode standard RFC 4648 base64. Returns the decoded bytes as standard data_base64 plus UTF-8 text when valid.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"max_bytes": map[string]any{"type": "integer", "description": "Optional maximum decoded byte length. Defaults to 65536."},
					"value":     map[string]any{"type": "string", "description": "Base64 value to decode."},
				},
				"required":             []string{"value"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "hex_encode",
			Description: "Encode bytes as lowercase hexadecimal. Provide either text or data_base64 input.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"data_base64": map[string]any{"type": "string", "description": "Optional standard base64 wrapper for arbitrary input bytes."},
					"text":        map[string]any{"type": "string", "description": "Optional UTF-8 text to encode."},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "hex_decode",
			Description: "Decode a hexadecimal string. Returns the decoded bytes as standard data_base64 plus UTF-8 text when valid.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"max_bytes": map[string]any{"type": "integer", "description": "Optional maximum decoded byte length. Defaults to 65536."},
					"value":     map[string]any{"type": "string", "description": "Hexadecimal value to decode."},
				},
				"required":             []string{"value"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "gzip_encode",
			Description: "Compress bytes with gzip. Provide either text or data_base64 input. Returns the compressed bytes as standard data_base64.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"data_base64": map[string]any{"type": "string", "description": "Optional standard base64 wrapper for arbitrary input bytes."},
					"text":        map[string]any{"type": "string", "description": "Optional UTF-8 text to compress."},
				},
				"additionalProperties": false,
			},
		},
		{
			Name:        "gzip_decode",
			Description: "Decompress gzip bytes supplied as standard data_base64. Returns the decoded bytes as standard data_base64 plus UTF-8 text when valid.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"data_base64": map[string]any{"type": "string", "description": "Standard base64 wrapper for gzip-compressed bytes."},
					"max_bytes":   map[string]any{"type": "integer", "description": "Optional maximum decoded byte length. Defaults to 65536."},
				},
				"required":             []string{"data_base64"},
				"additionalProperties": false,
			},
		},
	}
}

func (e *executor) callEncodingTool(_ context.Context, name string, arguments string) (string, bool, error) {
	switch strings.TrimSpace(name) {
	case "base32_encode":
		var args encodingInputArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", true, err
		}
		result, err := callEncodeValueTool("base32", "standard", args, standardBase32Encoder{})
		return result, true, err
	case "base32_decode":
		var args encodedValueArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", true, err
		}
		result, err := callDecodeValueTool("base32", "standard", args, standardBase32Encoder{})
		return result, true, err
	case "base64_encode":
		var args encodingInputArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", true, err
		}
		result, err := callEncodeValueTool("base64", "standard", args, standardBase64Encoder{})
		return result, true, err
	case "base64_decode":
		var args encodedValueArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", true, err
		}
		result, err := callDecodeValueTool("base64", "standard", args, standardBase64Encoder{})
		return result, true, err
	case "hex_encode":
		var args encodingInputArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", true, err
		}
		result, err := callEncodeValueTool("hex", "", args, utilencoders.Hex{})
		return result, true, err
	case "hex_decode":
		var args encodedValueArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", true, err
		}
		result, err := callDecodeValueTool("hex", "", args, utilencoders.Hex{})
		return result, true, err
	case "gzip_encode":
		var args encodingInputArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", true, err
		}
		input, err := resolveEncodingInput(args)
		if err != nil {
			return "", true, err
		}
		output, err := new(utilencoders.Gzip).Encode(input)
		if err != nil {
			return "", true, err
		}
		result, err := marshalToolResult(newEncodingBytesResult("gzip", "encode", "", output))
		return result, true, err
	case "gzip_decode":
		var args gzipDecodeArgs
		if err := decodeToolArgs(arguments, &args); err != nil {
			return "", true, err
		}
		data, err := requireStandardBase64(args.DataBase64, "data_base64")
		if err != nil {
			return "", true, err
		}
		output, err := new(utilencoders.Gzip).DecodeWithMaxLen(data, resolveMaxBytes(args.MaxBytes))
		if err != nil {
			return "", true, err
		}
		result, err := marshalToolResult(newEncodingBytesResult("gzip", "decode", "", output))
		return result, true, err
	default:
		return "", false, nil
	}
}

func callEncodeValueTool(name string, variant string, args encodingInputArgs, encoder utilencoders.Encoder) (string, error) {
	input, err := resolveEncodingInput(args)
	if err != nil {
		return "", err
	}

	output, err := encoder.Encode(input)
	if err != nil {
		return "", err
	}
	return marshalToolResult(encodingValueResult{
		Encoding:  name,
		Operation: "encode",
		Variant:   variant,
		Value:     string(output),
		ByteLen:   len(output),
	})
}

func (standardBase32Encoder) Encode(data []byte) ([]byte, error) {
	return []byte(stdbase32.StdEncoding.EncodeToString(data)), nil
}

func (standardBase32Encoder) Decode(data []byte) ([]byte, error) {
	value := strings.TrimSpace(string(data))
	decoded, err := stdbase32.StdEncoding.DecodeString(value)
	if err == nil {
		return decoded, nil
	}
	return stdbase32.StdEncoding.WithPadding(stdbase32.NoPadding).DecodeString(value)
}

func (standardBase64Encoder) Encode(data []byte) ([]byte, error) {
	return []byte(stdbase64.StdEncoding.EncodeToString(data)), nil
}

func (standardBase64Encoder) Decode(data []byte) ([]byte, error) {
	return decodeStandardBase64(strings.TrimSpace(string(data)))
}

func callDecodeValueTool(name string, variant string, args encodedValueArgs, encoder utilencoders.Encoder) (string, error) {
	value, err := requireString(args.Value, "value")
	if err != nil {
		return "", err
	}

	output, err := encoder.Decode([]byte(value))
	if err != nil {
		return "", err
	}

	maxBytes := resolveMaxBytes(args.MaxBytes)
	if int64(len(output)) > maxBytes {
		return "", fmt.Errorf("%s decoded payload exceeds %d bytes", name, maxBytes)
	}

	return marshalToolResult(newEncodingBytesResult(name, "decode", variant, output))
}

func resolveEncodingInput(args encodingInputArgs) ([]byte, error) {
	hasText := args.Text != nil
	hasData := args.DataBase64 != nil
	switch {
	case hasText && hasData:
		return nil, fmt.Errorf("provide only one of text or data_base64")
	case hasText:
		return []byte(*args.Text), nil
	case hasData:
		return decodeStandardBase64(*args.DataBase64)
	default:
		return nil, fmt.Errorf("text or data_base64 is required")
	}
}

func requireString(value *string, field string) (string, error) {
	if value == nil {
		return "", fmt.Errorf("%s is required", field)
	}
	return *value, nil
}

func requireStandardBase64(value *string, field string) ([]byte, error) {
	if value == nil {
		return nil, fmt.Errorf("%s is required", field)
	}
	data, err := decodeStandardBase64(*value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", field, err)
	}
	return data, nil
}

func decodeStandardBase64(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	data, err := stdbase64.StdEncoding.DecodeString(value)
	if err == nil {
		return data, nil
	}
	return stdbase64.RawStdEncoding.DecodeString(value)
}

func resolveMaxBytes(maxBytes *int64) int64 {
	if maxBytes != nil && *maxBytes > 0 {
		return *maxBytes
	}
	return aiEncodingDefaultMaxBytes
}

func newEncodingBytesResult(name string, operation string, variant string, data []byte) encodingBytesResult {
	result := encodingBytesResult{
		Encoding:   name,
		Operation:  operation,
		Variant:    variant,
		ByteLen:    len(data),
		DataBase64: stdbase64.StdEncoding.EncodeToString(data),
	}
	if len(data) > 0 {
		result.SHA256 = sha256Hex(data)
		if utf8.Valid(data) {
			result.Text = string(data)
		}
	}
	return result
}
