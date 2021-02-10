# lmorg/readline

## Changes

### 2.1.0

Error returns from `readline` have been created as error a variable, which is
more idiomatic to Go than the err constants that existed previously. Currently
both are still available to use however I will be deprecating the the constants
in a latter release.

**Deprecated constants:**
```go
const (
	// ErrCtrlC is returned when ctrl+c is pressed
	ErrCtrlC = "Ctrl+C"

	// ErrEOF is returned when ctrl+d is pressed
	ErrEOF = "EOF"
)
```

**New error variables:**
```go
var (
	// CtrlC is returned when ctrl+c is pressed
	CtrlC = errors.New("Ctrl+C")

	// EOF is returned when ctrl+d is pressed
	// (this is actually the same value as io.EOF)
	EOF = errors.New("EOF")
)
```

## Version Information

`readline`'s version numbers are based on Semantic Versioning. More details can
be found in the [README.md](README.md#version-information).