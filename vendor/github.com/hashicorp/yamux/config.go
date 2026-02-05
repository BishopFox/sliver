package yamux

// Config is a placeholder for API-compatibility with the upstream yamux
// library. This vendored implementation currently ignores configuration.
type Config struct{}

// DefaultConfig returns a default Config.
func DefaultConfig() *Config { return &Config{} }
