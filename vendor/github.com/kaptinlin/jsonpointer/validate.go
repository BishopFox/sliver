package jsonpointer

// Validation limits aligned with TypeScript implementation
const (
	// MaxPointerLength is the maximum allowed length for JSON Pointer strings.
	// Aligned with TypeScript: > 1024 is invalid
	MaxPointerLength = 1024

	// MaxPathLength is the maximum allowed length for Path arrays.
	// Aligned with TypeScript: > 256 is invalid
	MaxPathLength = 256
)

// validatePointerString validates a JSON Pointer string.
func validatePointerString(pointer string) error {
	// Empty string is valid (root pointer)
	if pointer == "" {
		return nil
	}

	// Must start with "/"
	if pointer[0] != '/' {
		return ErrPointerInvalid
	}

	// Check length limit (aligned with TypeScript: > 1024)
	if len(pointer) > MaxPointerLength {
		return ErrPointerTooLong
	}

	// Validate escape sequences
	for i := 0; i < len(pointer); i++ {
		if pointer[i] == '~' {
			if i+1 >= len(pointer) {
				return ErrPointerInvalid // Invalid escape at end
			}
			next := pointer[i+1]
			if next != '0' && next != '1' {
				return ErrPointerInvalid // Invalid escape sequence
			}
			i++ // Skip the next character
		}
	}

	return nil
}

// validatePath validates a Path array.
// Returns an error if the path exceeds the maximum allowed length.
func validatePath(path Path) error {
	if len(path) > MaxPathLength {
		return ErrPathTooLong
	}
	return nil
}
