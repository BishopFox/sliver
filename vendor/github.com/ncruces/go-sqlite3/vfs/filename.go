package vfs

import (
	"net/url"
	"strconv"

	"github.com/ncruces/go-sqlite3/internal/sqlite3_wrap"
	"github.com/ncruces/go-sqlite3/internal/util"
)

// Filename is used by SQLite to pass filenames
// to the Open method of a VFS.
//
// https://sqlite.org/c3ref/filename.html
type Filename struct {
	wrp   *sqlite3_wrap.Wrapper
	zPath ptr_t
	flags OpenFlag
}

// GetFilename is an internal API users should not call directly.
func GetFilename(wrp *sqlite3_wrap.Wrapper, id ptr_t, flags OpenFlag) *Filename {
	if id == 0 {
		return nil
	}
	return &Filename{
		wrp:   wrp,
		zPath: id,
		flags: flags,
	}
}

// String returns this filename as a string.
func (n *Filename) String() string {
	if n == nil || n.zPath == 0 {
		return ""
	}
	return n.wrp.ReadString(n.zPath, _MAX_PATHNAME)
}

// Database returns the name of the corresponding database file.
//
// https://sqlite.org/c3ref/filename_database.html
func (n *Filename) Database() string {
	if n == nil || n.zPath == 0 {
		return ""
	}
	return n.path(n.wrp.Xsqlite3_filename_database)
}

// Journal returns the name of the corresponding rollback journal file.
//
// https://sqlite.org/c3ref/filename_database.html
func (n *Filename) Journal() string {
	if n == nil || n.zPath == 0 {
		return ""
	}
	return n.path(n.wrp.Xsqlite3_filename_journal)
}

// WAL returns the name of the corresponding WAL file.
//
// https://sqlite.org/c3ref/filename_database.html
func (n *Filename) WAL() string {
	if n == nil || n.zPath == 0 {
		return ""
	}
	return n.path(n.wrp.Xsqlite3_filename_wal)
}

func (n *Filename) path(fn func(int32) int32) string {
	if n.flags&(OPEN_MAIN_DB|OPEN_MAIN_JOURNAL|OPEN_WAL) == 0 {
		return ""
	}
	name := ptr_t(fn(int32(n.zPath)))
	return n.wrp.ReadString(name, _MAX_PATHNAME)
}

// DatabaseFile returns the main database [File] corresponding to a journal.
//
// https://sqlite.org/c3ref/database_file_object.html
func (n *Filename) DatabaseFile() File {
	if n == nil || n.zPath == 0 {
		return nil
	}
	if n.flags&(OPEN_MAIN_DB|OPEN_MAIN_JOURNAL|OPEN_WAL) == 0 {
		return nil
	}

	pFile := ptr_t(n.wrp.Xsqlite3_database_file_object(int32(n.zPath)))
	file, _ := vfsFileGet(n.wrp, ptr_t(pFile)).(File)
	return file
}

// URIParameter returns the value of a URI parameter.
//
// https://sqlite.org/c3ref/uri_boolean.html
func (n *Filename) URIParameter(key string) string {
	if n == nil || n.zPath == 0 {
		return ""
	}

	ptr := ptr_t(n.wrp.Xsqlite3_uri_key(int32(n.zPath), 0))
	if ptr == 0 {
		return ""
	}

	// Parse the format from:
	// https://github.com/sqlite/sqlite/blob/41fda52/src/pager.c#L4821-L4864
	// This avoids having to alloc/free the key just to find a value.
	mem := n.wrp.Memory
	for {
		k := mem.ReadString(ptr, _MAX_NAME)
		if k == "" {
			return ""
		}
		ptr += ptr_t(len(k)) + 1

		v := mem.ReadString(ptr, _MAX_NAME)
		if k == key {
			return v
		}
		ptr += ptr_t(len(v)) + 1
	}
}

// URIParameters obtains values for URI parameters.
//
// https://sqlite.org/c3ref/uri_boolean.html
func (n *Filename) URIParameters() url.Values {
	if n == nil || n.zPath == 0 {
		return nil
	}

	ptr := ptr_t(n.wrp.Xsqlite3_uri_key(int32(n.zPath), 0))
	if ptr == 0 {
		return nil
	}

	var params url.Values

	// Parse the format from:
	// https://github.com/sqlite/sqlite/blob/41fda52/src/pager.c#L4821-L4864
	// This is the only way to support multiple valued keys.
	mem := n.wrp.Memory
	for {
		k := mem.ReadString(ptr, _MAX_NAME)
		if k == "" {
			return params
		}
		ptr += ptr_t(len(k)) + 1

		v := mem.ReadString(ptr, _MAX_NAME)
		if params == nil {
			params = url.Values{}
		}
		params.Add(k, v)
		ptr += ptr_t(len(v)) + 1
	}
}

// URIBoolean returns the value of a URI parameter as a boolean.
//
// https://sqlite.org/c3ref/uri_boolean.html
func (n *Filename) URIBoolean(key string, def bool) bool {
	b, ok := util.ParseBool(n.URIParameter(key))
	if ok {
		return b
	}
	return def
}

// URIInt64 returns the value of a URI parameter converted
// to a 64-bit signed integer.
//
// https://sqlite.org/c3ref/uri_boolean.html
func (n *Filename) URIInt64(key string, def int64) int64 {
	s := n.URIParameter(key)
	if s == "" {
		return def
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return def
	}
	return i
}
