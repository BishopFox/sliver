package jsonschema

import (
	"strconv"
	"strings"
	"time"
)

// FunctionCall represents a parsed function call with name and arguments
type FunctionCall struct {
	Name string
	Args []any
}

// parseFunctionCall parses a string to determine if it's a function call
// Returns nil if the string is not a function call format
func parseFunctionCall(input string) (*FunctionCall, error) {
	// Check if it's in function call format: functionName()
	if len(input) < 3 || !strings.HasSuffix(input, ")") {
		return nil, nil // Not a function call
	}

	parenIndex := strings.IndexByte(input, '(')
	if parenIndex <= 0 {
		return nil, nil // Not a function call
	}

	name := strings.TrimSpace(input[:parenIndex])
	argsStr := strings.TrimSpace(input[parenIndex+1 : len(input)-1])

	// Parse arguments
	var args []any
	if argsStr != "" {
		args = parseArgs(argsStr)
	}

	return &FunctionCall{
		Name: name,
		Args: args,
	}, nil
}

// parseArgs parses function arguments from a string
// Simple implementation that handles basic types
func parseArgs(argsStr string) []any {
	parts := strings.Split(argsStr, ",")
	args := make([]any, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Try to parse as integer
		if i, err := strconv.ParseInt(part, 10, 64); err == nil {
			args = append(args, i)
			continue
		}

		// Try to parse as float
		if f, err := strconv.ParseFloat(part, 64); err == nil {
			args = append(args, f)
			continue
		}

		// Default to string
		args = append(args, part)
	}

	return args
}

// DefaultNowFunc generates current timestamp in various formats
// This function must be manually registered by developers
func DefaultNowFunc(args ...any) (any, error) {
	format := time.RFC3339 // Default format

	if len(args) > 0 {
		if f, ok := args[0].(string); ok {
			format = f
		}
	}

	return time.Now().Format(format), nil
}
