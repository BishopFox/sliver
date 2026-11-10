package testenv

import (
	"bytes"
	"context"
	"io/fs"
	"sync"
)

type Context interface {
	Context() context.Context
	Logf(format string, args ...any)
}

var (
	FS     fs.FS
	TB     Context
	Exit   func(int32)
	System func(string) int32

	buf []byte
	mtx sync.Mutex
)

func WriteByte(c byte) error {
	mtx.Lock()
	defer mtx.Unlock()

	if c == '\n' {
		TB.Logf("%s", buf)
		buf = buf[:0]
	} else {
		buf = append(buf, c)
	}
	return nil
}

func Write(p []byte) (n int, err error) {
	mtx.Lock()
	defer mtx.Unlock()

	buf = append(buf, p...)
	for {
		before, after, found := bytes.Cut(buf, []byte("\n"))
		if !found {
			return len(p), nil
		}
		TB.Logf("%s", before)
		buf = after
	}
}

func WriteString(s string) (n int, err error) {
	return Write([]byte(s))
}
