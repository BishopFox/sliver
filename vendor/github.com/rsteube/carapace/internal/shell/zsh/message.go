package zsh

import (
	"fmt"
	"strings"

	"github.com/rsteube/carapace/internal/common"
	"github.com/rsteube/carapace/pkg/style"
)

type message struct {
	common.Meta
}

func (m message) Format() string {
	formatted := make([]string, 0)
	for _, message := range m.Messages.Get() {
		formatted = append(formatted, m.formatMessage(message, style.Carapace.Error))
	}
	if m.Usage != "" {
		formatted = append(formatted, m.formatMessage(m.Usage, style.Carapace.Usage))
	}

	if len(formatted) > 0 {
		return strings.Join(formatted, "\n")
	}
	return ""
}

func (m message) formatMessage(message, _style string) string {
	msg := strings.NewReplacer(
		"\n", ``,
		"\r", ``,
		"\t", ``,
		"\v", ``,
		"\f", ``,
		"\b", ``,
	).Replace(message)

	return fmt.Sprintf("\x1b[%vm%v\x1b[%vm", style.SGR(_style), msg, style.SGR("fg-default"))
}
