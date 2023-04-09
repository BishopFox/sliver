package driver

type errorString string

func (e errorString) Error() string { return string(e) }

const (
	assertErr    = errorString("sqlite3: assertion failed")
	tailErr      = errorString("sqlite3: multiple statements")
	isolationErr = errorString("sqlite3: unsupported isolation level")
)
