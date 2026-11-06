package sqlite3_wrap

import (
	"io"

	"github.com/ncruces/go-sqlite3/internal/testenv"
)

const _MAX_NAME = 1e6 // Self-imposed limit for most NUL terminated strings.

func (w *Wrapper) Xexit(c int32) {
	testenv.Exit(c)
}

func (w *Wrapper) Xsystem(ptr int32) int32 {
	if testenv.TB == nil || ptr == 0 {
		return 0
	}
	s := w.ReadString(Ptr_t(ptr), _MAX_NAME)
	return testenv.System(s)
}

func (w *Wrapper) Xputs(ptr int32) int32 {
	if testenv.TB == nil {
		return -1
	}
	s := w.ReadString(Ptr_t(ptr), _MAX_NAME)
	testenv.WriteString(s)
	testenv.WriteByte('\n')
	return 0
}

func (w *Wrapper) Xfclose(h int32) int32 {
	if testenv.TB == nil {
		return -1
	}
	if w.DelHandle(Ptr_t(h)) != nil {
		return -1
	}
	return 0
}

func (w *Wrapper) Xfopen(path, mode int32) int32 {
	if testenv.TB == nil {
		return 0
	}
	p := w.ReadString(Ptr_t(path), _MAX_NAME)
	f, err := testenv.FS.Open(p)
	if err != nil {
		return 0
	}
	return int32(w.AddHandle(f))
}

func (w *Wrapper) Xfflush(h int32) int32 {
	if testenv.TB == nil {
		return -1
	}
	return 0
}

func (w *Wrapper) Xfputc(c, h int32) int32 {
	if testenv.TB == nil {
		return -1
	}
	if testenv.WriteByte(byte(c)) != nil {
		return -1
	}
	return 0
}

func (w *Wrapper) Xfwrite(ptr, sz, cnt, h int32) int32 {
	if testenv.TB == nil {
		return 0
	}
	b := w.Buf[ptr:][:sz*cnt]
	n, _ := testenv.Write(b)
	return int32(n / int(sz))
}

func (w *Wrapper) Xfread(ptr, sz, cnt, h int32) int32 {
	if testenv.TB == nil {
		return 0
	}
	f := w.GetHandle(Ptr_t(h)).(io.Reader)
	b := w.Buf[ptr:][:sz*cnt]
	n, _ := f.Read(b)
	return int32(n / int(sz))
}

func (w *Wrapper) Xftell(h int32) int32 {
	if testenv.TB == nil {
		return -1
	}
	f := w.GetHandle(Ptr_t(h)).(io.Seeker)
	n, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return -1
	}
	return int32(n)
}

func (w *Wrapper) Xfseek(h, offset, whence int32) int32 {
	if testenv.TB == nil {
		return -1
	}
	f := w.GetHandle(Ptr_t(h)).(io.Seeker)
	_, err := f.Seek(int64(offset), int(whence))
	if err != nil {
		return -1
	}
	return 0
}
