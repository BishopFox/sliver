package table

// PagerOption helps control Paging.
type PagerOption func(t *Table)

// PageSize sets the size of each page rendered.
func PageSize(pageSize int) PagerOption {
	return func(t *Table) {
		t.pager.size = pageSize
	}
}
