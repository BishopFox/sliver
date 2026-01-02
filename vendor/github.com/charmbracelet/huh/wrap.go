package huh

import "github.com/charmbracelet/x/cellbuf"

func wrap(s string, limit int) string {
	return cellbuf.Wrap(s, limit, ",.-; ")
}
