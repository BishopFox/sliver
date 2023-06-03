package sqlite3

import "io"

// ZeroBlob represents a zero-filled, length n BLOB
// that can be used as an argument to
// [database/sql.DB.Exec] and similar methods.
type ZeroBlob int64

// Blob is a handle to an open BLOB.
//
// It implements [io.ReadWriteSeeker] for incremental BLOB I/O.
//
// https://www.sqlite.org/c3ref/blob.html
type Blob struct {
	c      *Conn
	handle uint32
	bytes  int64
	offset int64
}

var _ io.ReadWriteSeeker = &Blob{}

// OpenBlob opens a BLOB for incremental I/O.
//
// https://www.sqlite.org/c3ref/blob_open.html
func (c *Conn) OpenBlob(db, table, column string, row int64, write bool) (*Blob, error) {
	c.checkInterrupt()
	defer c.arena.reset()
	blobPtr := c.arena.new(ptrlen)
	dbPtr := c.arena.string(db)
	tablePtr := c.arena.string(table)
	columnPtr := c.arena.string(column)

	var flags uint64
	if write {
		flags = 1
	}

	r := c.call(c.api.blobOpen, uint64(c.handle),
		uint64(dbPtr), uint64(tablePtr), uint64(columnPtr),
		uint64(row), flags, uint64(blobPtr))

	if err := c.error(r[0]); err != nil {
		return nil, err
	}

	blob := Blob{c: c}
	blob.handle = c.mem.readUint32(blobPtr)
	blob.bytes = int64(c.call(c.api.blobBytes, uint64(blob.handle))[0])
	return &blob, nil
}

// Close closes a BLOB handle.
//
// It is safe to close a nil, zero or closed Blob.
//
// https://www.sqlite.org/c3ref/blob_close.html
func (b *Blob) Close() error {
	if b == nil || b.handle == 0 {
		return nil
	}

	r := b.c.call(b.c.api.blobClose, uint64(b.handle))

	b.handle = 0
	return b.c.error(r[0])
}

// Size returns the size of the BLOB in bytes.
//
// https://www.sqlite.org/c3ref/blob_bytes.html
func (b *Blob) Size() int64 {
	return b.bytes
}

// Read implements the [io.Reader] interface.
//
// https://www.sqlite.org/c3ref/blob_read.html
func (b *Blob) Read(p []byte) (n int, err error) {
	if b.offset >= b.bytes {
		return 0, io.EOF
	}

	want := int64(len(p))
	avail := b.bytes - b.offset
	if want > avail {
		want = avail
	}

	ptr := b.c.new(uint64(want))
	defer b.c.free(ptr)

	r := b.c.call(b.c.api.blobRead, uint64(b.handle),
		uint64(ptr), uint64(want), uint64(b.offset))
	err = b.c.error(r[0])
	if err != nil {
		return 0, err
	}

	mem := b.c.mem.view(ptr, uint64(want))
	copy(p, mem)
	b.offset += want
	if b.offset >= b.bytes {
		err = io.EOF
	}
	return int(want), err
}

// Write implements the [io.Writer] interface.
//
// https://www.sqlite.org/c3ref/blob_write.html
func (b *Blob) Write(p []byte) (n int, err error) {
	offset := b.offset
	if offset > b.bytes {
		offset = b.bytes
	}

	ptr := b.c.newBytes(p)
	defer b.c.free(ptr)

	r := b.c.call(b.c.api.blobWrite, uint64(b.handle),
		uint64(ptr), uint64(len(p)), uint64(offset))
	err = b.c.error(r[0])
	if err != nil {
		return 0, err
	}
	b.offset += int64(len(p))
	return len(p), nil
}

// Seek implements the [io.Seeker] interface.
func (b *Blob) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	default:
		return 0, whenceErr
	case io.SeekStart:
		break
	case io.SeekCurrent:
		offset += b.offset
	case io.SeekEnd:
		offset += b.bytes
	}
	if offset < 0 {
		return 0, offsetErr
	}
	b.offset = offset
	return offset, nil
}

// Reopen moves a BLOB handle to a new row of the same database table.
//
// https://www.sqlite.org/c3ref/blob_reopen.html
func (b *Blob) Reopen(row int64) error {
	r := b.c.call(b.c.api.blobReopen, uint64(b.handle), uint64(row))
	b.bytes = int64(b.c.call(b.c.api.blobBytes, uint64(b.handle))[0])
	b.offset = 0
	return b.c.error(r[0])
}
