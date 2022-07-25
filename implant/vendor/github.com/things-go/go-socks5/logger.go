package socks5

import (
	"log"
)

// Logger is used to provide debug logger
type Logger interface {
	Errorf(format string, arg ...interface{})
}

// Std std logger
type Std struct {
	*log.Logger
}

// NewLogger new std logger with log.logger
func NewLogger(l *log.Logger) *Std {
	return &Std{l}
}

// Errorf implement interface Logger
func (sf Std) Errorf(format string, args ...interface{}) {
	sf.Logger.Printf("[E]: "+format, args...)
}
