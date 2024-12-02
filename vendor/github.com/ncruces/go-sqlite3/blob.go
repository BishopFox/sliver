package sqlite3

import (
	"io"

	"github.com/ncruces/go-sqlite3/internal/util"
)

// ZeroBlob represents a zero-filled, length n BLOB
// that can be used as an argument to
// [database/sql.DB.Exec] and similar methods.
type ZeroBlob int64

// Blob is an handle to an open BLOB.
//
// It implements [io.ReadWriteSeeker] for incremental BLOB I/O.
//
// https://sqlite.org/c3ref/blob.html
type Blob struct {
	c      *Conn
	bytes  int64
	offset int64
	handle uint32
	bufptr uint32
	buflen int64
}

var _ io.ReadWriteSeeker = &Blob{}

// OpenBlob opens a BLOB for incremental I/O.
//
// https://sqlite.org/c3ref/blob_open.html
func (c *Conn) OpenBlob(db, table, column string, row int64, write bool) (*Blob, error) {
	defer c.arena.mark()()
	blobPtr := c.arena.new(ptrlen)
	dbPtr := c.arena.string(db)
	tablePtr := c.arena.string(table)
	columnPtr := c.arena.string(column)

	var flags uint64
	if write {
		flags = 1
	}

	c.checkInterrupt(c.handle)
	r := c.call("sqlite3_blob_open", uint64(c.handle),
		uint64(dbPtr), uint64(tablePtr), uint64(columnPtr),
		uint64(row), flags, uint64(blobPtr))

	if err := c.error(r); err != nil {
		return nil, err
	}

	blob := Blob{c: c}
	blob.handle = util.ReadUint32(c.mod, blobPtr)
	blob.bytes = int64(c.call("sqlite3_blob_bytes", uint64(blob.handle)))
	return &blob, nil
}

// Close closes a BLOB handle.
//
// It is safe to close a nil, zero or closed Blob.
//
// https://sqlite.org/c3ref/blob_close.html
func (b *Blob) Close() error {
	if b == nil || b.handle == 0 {
		return nil
	}

	r := b.c.call("sqlite3_blob_close", uint64(b.handle))
	b.c.free(b.bufptr)
	b.handle = 0
	return b.c.error(r)
}

// Size returns the size of the BLOB in bytes.
//
// https://sqlite.org/c3ref/blob_bytes.html
func (b *Blob) Size() int64 {
	return b.bytes
}

// Read implements the [io.Reader] interface.
//
// https://sqlite.org/c3ref/blob_read.html
func (b *Blob) Read(p []byte) (n int, err error) {
	if b.offset >= b.bytes {
		return 0, io.EOF
	}

	want := int64(len(p))
	avail := b.bytes - b.offset
	if want > avail {
		want = avail
	}
	if want > b.buflen {
		b.bufptr = b.c.realloc(b.bufptr, uint64(want))
		b.buflen = want
	}

	r := b.c.call("sqlite3_blob_read", uint64(b.handle),
		uint64(b.bufptr), uint64(want), uint64(b.offset))
	err = b.c.error(r)
	if err != nil {
		return 0, err
	}
	b.offset += want
	if b.offset >= b.bytes {
		err = io.EOF
	}

	copy(p, util.View(b.c.mod, b.bufptr, uint64(want)))
	return int(want), err
}

// WriteTo implements the [io.WriterTo] interface.
//
// https://sqlite.org/c3ref/blob_read.html
func (b *Blob) WriteTo(w io.Writer) (n int64, err error) {
	if b.offset >= b.bytes {
		return 0, nil
	}

	want := int64(1024 * 1024)
	avail := b.bytes - b.offset
	if want > avail {
		want = avail
	}
	if want > b.buflen {
		b.bufptr = b.c.realloc(b.bufptr, uint64(want))
		b.buflen = want
	}

	for want > 0 {
		r := b.c.call("sqlite3_blob_read", uint64(b.handle),
			uint64(b.bufptr), uint64(want), uint64(b.offset))
		err = b.c.error(r)
		if err != nil {
			return n, err
		}

		mem := util.View(b.c.mod, b.bufptr, uint64(want))
		m, err := w.Write(mem[:want])
		b.offset += int64(m)
		n += int64(m)
		if err != nil {
			return n, err
		}
		if int64(m) != want {
			// notest // Write misbehaving
			return n, io.ErrShortWrite
		}

		avail = b.bytes - b.offset
		if want > avail {
			want = avail
		}
	}
	return n, nil
}

// Write implements the [io.Writer] interface.
//
// https://sqlite.org/c3ref/blob_write.html
func (b *Blob) Write(p []byte) (n int, err error) {
	want := int64(len(p))
	if want > b.buflen {
		b.bufptr = b.c.realloc(b.bufptr, uint64(want))
		b.buflen = want
	}
	util.WriteBytes(b.c.mod, b.bufptr, p)

	r := b.c.call("sqlite3_blob_write", uint64(b.handle),
		uint64(b.bufptr), uint64(want), uint64(b.offset))
	err = b.c.error(r)
	if err != nil {
		return 0, err
	}
	b.offset += int64(len(p))
	return len(p), nil
}

// ReadFrom implements the [io.ReaderFrom] interface.
//
// https://sqlite.org/c3ref/blob_write.html
func (b *Blob) ReadFrom(r io.Reader) (n int64, err error) {
	want := int64(1024 * 1024)
	avail := b.bytes - b.offset
	if l, ok := r.(*io.LimitedReader); ok && want > l.N {
		want = l.N
	}
	if want > avail {
		want = avail
	}
	if want < 1 {
		want = 1
	}
	if want > b.buflen {
		b.bufptr = b.c.realloc(b.bufptr, uint64(want))
		b.buflen = want
	}

	for {
		mem := util.View(b.c.mod, b.bufptr, uint64(want))
		m, err := r.Read(mem[:want])
		if m > 0 {
			r := b.c.call("sqlite3_blob_write", uint64(b.handle),
				uint64(b.bufptr), uint64(m), uint64(b.offset))
			err := b.c.error(r)
			if err != nil {
				return n, err
			}
			b.offset += int64(m)
			n += int64(m)
		}
		if err == io.EOF {
			return n, nil
		}
		if err != nil {
			return n, err
		}

		avail = b.bytes - b.offset
		if want > avail {
			want = avail
		}
		if want < 1 {
			want = 1
		}
	}
}

// Seek implements the [io.Seeker] interface.
func (b *Blob) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	default:
		return 0, util.WhenceErr
	case io.SeekStart:
		break
	case io.SeekCurrent:
		offset += b.offset
	case io.SeekEnd:
		offset += b.bytes
	}
	if offset < 0 {
		return 0, util.OffsetErr
	}
	b.offset = offset
	return offset, nil
}

// Reopen moves a BLOB handle to a new row of the same database table.
//
// https://sqlite.org/c3ref/blob_reopen.html
func (b *Blob) Reopen(row int64) error {
	b.c.checkInterrupt(b.c.handle)
	err := b.c.error(b.c.call("sqlite3_blob_reopen", uint64(b.handle), uint64(row)))
	b.bytes = int64(b.c.call("sqlite3_blob_bytes", uint64(b.handle)))
	b.offset = 0
	return err
}
