package huh

import "charm.land/lipgloss/v2"

func wrap(s string, limit int) string {
	return lipgloss.Wrap(s, limit, ",.-; ")
}
