package sqlite3

import (
	"io"

	"github.com/ncruces/go-sqlite3/internal/errutil"
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
	handle ptr_t
	bufptr ptr_t
	buflen int64
}

var _ io.ReadWriteSeeker = &Blob{}

// OpenBlob opens a BLOB for incremental I/O.
//
// https://sqlite.org/c3ref/blob_open.html
func (c *Conn) OpenBlob(db, table, column string, row int64, write bool) (*Blob, error) {
	if c.interrupt.Err() != nil {
		return nil, INTERRUPT
	}

	defer c.arena.Mark()()
	blobPtr := c.arena.New(ptrlen)
	dbPtr := c.arena.String(db)
	tablePtr := c.arena.String(table)
	columnPtr := c.arena.String(column)

	var flags int32
	if write {
		flags = 1
	}

	rc := res_t(c.wrp.Xsqlite3_blob_open(int32(c.handle),
		int32(dbPtr), int32(tablePtr), int32(columnPtr),
		row, flags, int32(blobPtr)))

	if err := c.error(rc); err != nil {
		return nil, err
	}

	blob := Blob{c: c}
	blob.handle = ptr_t(c.wrp.Read32(blobPtr))
	blob.bytes = int64(c.wrp.Xsqlite3_blob_bytes(int32(blob.handle)))
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

	rc := res_t(b.c.wrp.Xsqlite3_blob_close(int32(b.handle)))
	b.c.wrp.Free(b.bufptr)
	b.handle = 0
	return b.c.error(rc)
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
		b.bufptr = b.c.wrp.Realloc(b.bufptr, want)
		b.buflen = want
	}

	rc := res_t(b.c.wrp.Xsqlite3_blob_read(int32(b.handle),
		int32(b.bufptr), int32(want), int32(b.offset)))
	err = b.c.error(rc)
	if err != nil {
		return 0, err
	}
	b.offset += want
	if b.offset >= b.bytes {
		err = io.EOF
	}

	copy(p, b.c.wrp.Bytes(b.bufptr, want))
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
		b.bufptr = b.c.wrp.Realloc(b.bufptr, want)
		b.buflen = want
	}

	for want > 0 {
		rc := res_t(b.c.wrp.Xsqlite3_blob_read(int32(b.handle),
			int32(b.bufptr), int32(want), int32(b.offset)))
		err = b.c.error(rc)
		if err != nil {
			return n, err
		}

		mem := b.c.wrp.Bytes(b.bufptr, want)
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
		b.bufptr = b.c.wrp.Realloc(b.bufptr, want)
		b.buflen = want
	}
	b.c.wrp.WriteBytes(b.bufptr, p)

	rc := res_t(b.c.wrp.Xsqlite3_blob_write(int32(b.handle),
		int32(b.bufptr), int32(want), int32(b.offset)))
	err = b.c.error(rc)
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
		b.bufptr = b.c.wrp.Realloc(b.bufptr, want)
		b.buflen = want
	}

	for {
		mem := b.c.wrp.Bytes(b.bufptr, want)
		m, err := r.Read(mem[:want])
		if m > 0 {
			rc := res_t(b.c.wrp.Xsqlite3_blob_write(int32(b.handle),
				int32(b.bufptr), int32(m), int32(b.offset)))
			err := b.c.error(rc)
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
		return 0, errutil.WhenceErr
	case io.SeekStart:
		break
	case io.SeekCurrent:
		offset += b.offset
	case io.SeekEnd:
		offset += b.bytes
	}
	if offset < 0 {
		return 0, errutil.OffsetErr
	}
	b.offset = offset
	return offset, nil
}

// Reopen moves a BLOB handle to a new row of the same database table.
//
// https://sqlite.org/c3ref/blob_reopen.html
func (b *Blob) Reopen(row int64) error {
	if b.c.interrupt.Err() != nil {
		return INTERRUPT
	}
	err := b.c.error(res_t(b.c.wrp.Xsqlite3_blob_reopen(int32(b.handle), row)))
	b.bytes = int64(b.c.wrp.Xsqlite3_blob_bytes(int32(b.handle)))
	b.offset = 0
	return err
}
