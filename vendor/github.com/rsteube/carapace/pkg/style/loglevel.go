package style

import (
	"strings"
)

// ForLogLevel returns the style for given log level.
func ForLogLevel(s string, _ Context) string {
	return map[string]string{
		"trace":    Carapace.LogLevelTrace,
		"debug":    Carapace.LogLevelDebug,
		"vdebug":   Carapace.LogLevelDebug,
		"info":     Carapace.LogLevelInfo,
		"warn":     Carapace.LogLevelWarning,
		"warning":  Carapace.LogLevelWarning,
		"err":      Carapace.LogLevelError,
		"error":    Carapace.LogLevelError,
		"crit":     Carapace.LogLevelCritical,
		"critical": Carapace.LogLevelCritical,
		"fatal":    Carapace.LogLevelFatal,
		"panic":    Carapace.LogLevelFatal,
	}[strings.ToLower(s)]
}
