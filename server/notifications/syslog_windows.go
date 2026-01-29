//go:build windows

package notifications

import (
	"errors"

	"github.com/bishopfox/sliver/server/configs"
	"github.com/nikoksr/notify"
)

func buildSyslog(_ *configs.SyslogConfig) (notify.Notifier, error) {
	return nil, errors.New("syslog notifications are not supported on windows")
}
