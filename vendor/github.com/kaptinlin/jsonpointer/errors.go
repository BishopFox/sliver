package jsonpointer

import "errors"

// Predefined errors matching TypeScript exactly

// ErrInvalidIndex is returned when an invalid array index is encountered.
// TypeScript original code from find.ts:
// throw new Error('INVALID_INDEX');
var ErrInvalidIndex = errors.New("invalid array index")

// ErrNotFound is returned when a path cannot be traversed.
// TypeScript original code from find.ts:
// throw new Error('NOT_FOUND');
var ErrNotFound = errors.New("not found")

// ErrNoParent is returned when trying to get parent of root path.
// TypeScript original code from util.ts:
// if (path.length < 1) throw new Error('NO_PARENT');
var ErrNoParent = errors.New("no parent")

// ErrPointerInvalid is returned when a JSON Pointer string is invalid.
// TypeScript original code from validate.ts:
// if (pointer[0] !== '/') throw new Error('POINTER_INVALID');
var ErrPointerInvalid = errors.New("pointer invalid")

// ErrPointerTooLong is returned when a JSON Pointer string exceeds maximum length.
// TypeScript original code from validate.ts:
// if (pointer.length > 1024) throw new Error('POINTER_TOO_LONG');
var ErrPointerTooLong = errors.New("pointer too long")

// ErrInvalidPath is returned when a path is not an array.
// TypeScript original code from validate.ts:
// if (!isArray(path)) throw new Error('Invalid path.');
var ErrInvalidPath = errors.New("invalid path")

// ErrPathTooLong is returned when a path array exceeds maximum length.
// TypeScript original code from validate.ts:
// if (path.length > 256) throw new Error('Path too long.');
var ErrPathTooLong = errors.New("path too long")

// ErrInvalidPathStep is returned when a path step is not string or number.
// TypeScript original code from validate.ts:
// throw new Error('Invalid path step.');
var ErrInvalidPathStep = errors.New("invalid path step")

// ErrIndexOutOfBounds is returned when array index is out of bounds.
var ErrIndexOutOfBounds = errors.New("array index out of bounds")

// ErrNilPointer is returned when trying to access through nil pointer.
var ErrNilPointer = errors.New("cannot traverse through nil pointer")

// ErrFieldNotFound is returned when trying to access a non-existent struct field.
var ErrFieldNotFound = errors.New("struct field not found")

// ErrKeyNotFound is returned when trying to access a non-existent map key.
var ErrKeyNotFound = errors.New("map key not found")
