Handlers 
=========

Notes on naming:
    - `generic` refers to the handler's calling convention, it is called and returns a generic message format
    - `_default.go` implementations are pure Go, we compile this file for unsupported GOOS/GOARCH combinations
