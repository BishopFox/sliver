//go:build !windows

package notifications

import (
	"errors"
	"log/syslog"
	"strconv"
	"strings"

	"github.com/bishopfox/sliver/server/configs"
	"github.com/nikoksr/notify"
	notifysyslog "github.com/nikoksr/notify/service/syslog"
)

func buildSyslog(cfg *configs.SyslogConfig) (notify.Notifier, error) {
	if cfg == nil {
		return nil, errors.New("syslog config is nil")
	}
	priority := parseSyslogPriority(cfg.Priority)
	network := strings.TrimSpace(cfg.Network)
	address := strings.TrimSpace(cfg.Address)
	tag := strings.TrimSpace(cfg.Tag)
	if network != "" || address != "" {
		return notifysyslog.NewFromDial(network, address, priority, tag)
	}
	return notifysyslog.New(priority, tag)
}

func parseSyslogPriority(value string) syslog.Priority {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return syslog.LOG_INFO
	}
	switch value {
	case "emerg", "emergency":
		return syslog.LOG_EMERG
	case "alert":
		return syslog.LOG_ALERT
	case "crit", "critical":
		return syslog.LOG_CRIT
	case "err", "error":
		return syslog.LOG_ERR
	case "warning", "warn":
		return syslog.LOG_WARNING
	case "notice":
		return syslog.LOG_NOTICE
	case "info":
		return syslog.LOG_INFO
	case "debug":
		return syslog.LOG_DEBUG
	default:
		if numeric, err := strconv.Atoi(value); err == nil {
			return syslog.Priority(numeric)
		}
	}
	return syslog.LOG_INFO
}
