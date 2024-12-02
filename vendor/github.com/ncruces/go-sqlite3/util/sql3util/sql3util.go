// Package sql3util implements SQLite utilities.
package sql3util

// ValidPageSize returns true if s is a valid page size.
//
// https://sqlite.org/fileformat.html#pages
func ValidPageSize(s int) bool {
	return 512 <= s && s <= 65536 && s&(s-1) == 0
}
