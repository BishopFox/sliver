package inputrc

import (
	"os"
)

// Handler is the handler interface.
type Handler interface {
	// ReadFile reads a file.
	ReadFile(name string) ([]byte, error)
	// Do handles $constructs.
	Do(typ string, param string) error
	// Set sets the value.
	Set(name string, value interface{}) error
	// Get gets the value.
	Get(name string) interface{}
	// Bind binds a key sequence to an action for the current keymap.
	Bind(keymap, sequence, action string, macro bool) error
}

// Config is a inputrc config handler.
type Config struct {
	ReadFileFunc func(string) ([]byte, error)
	Vars         map[string]interface{}
	Binds        map[string]map[string]Bind
	Funcs        map[string]func(string, string) error
}

// NewConfig creates a new inputrc config.
func NewConfig() *Config {
	return &Config{
		Vars:  make(map[string]interface{}),
		Binds: make(map[string]map[string]Bind),
		Funcs: make(map[string]func(string, string) error),
	}
}

// NewDefaultConfig creates a new inputrc config with default values.
func NewDefaultConfig(opts ...ConfigOption) *Config {
	cfg := &Config{
		ReadFileFunc: os.ReadFile,
		Vars:         DefaultVars(),
		Binds:        DefaultBinds(),
		Funcs:        make(map[string]func(string, string) error),
	}
	for _, o := range opts {
		o(cfg)
	}

	return cfg
}

// ReadFile satisfies the Handler interface.
func (cfg *Config) ReadFile(name string) ([]byte, error) {
	if cfg.ReadFileFunc != nil {
		return cfg.ReadFileFunc(name)
	}

	return nil, os.ErrNotExist
}

// Do satisfies the Handler interface.
func (cfg *Config) Do(name, value string) error {
	if f, ok := cfg.Funcs[name]; ok {
		return f(name, value)
	}

	if f, ok := cfg.Funcs[""]; ok {
		return f(name, value)
	}

	return nil
}

// Get satisfies the Handler interface.
func (cfg *Config) Get(name string) interface{} {
	return cfg.Vars[name]
}

// Set satisfies the Handler interface.
func (cfg *Config) Set(name string, value interface{}) error {
	cfg.Vars[name] = value
	return nil
}

// Bind satisfies the Handler interface.
func (cfg *Config) Bind(keymap, sequence, action string, macro bool) error {
	if cfg.Binds[keymap] == nil {
		cfg.Binds[keymap] = make(map[string]Bind)
	}

	cfg.Binds[keymap][sequence] = Bind{
		Action: action,
		Macro:  macro,
	}

	return nil
}

// GetString returns the var name as a string.
func (cfg *Config) GetString(name string) string {
	if v, ok := cfg.Vars[name]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}

	return ""
}

// GetInt returns the var name as a int.
func (cfg *Config) GetInt(name string) int {
	if v, ok := cfg.Vars[name]; ok {
		if i, ok := v.(int); ok {
			return i
		}
	}

	return 0
}

// GetBool returns the var name as a bool.
func (cfg *Config) GetBool(name string) bool {
	if v, ok := cfg.Vars[name]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}

	return false
}

// Bind represents a key binding.
type Bind struct {
	Action string
	Macro  bool
}

// ConfigOption is a inputrc config handler option.
type ConfigOption func(*Config)

// WithConfigReadFileFunc is a inputrc config option to set the func used
// for ReadFile operations.
func WithConfigReadFileFunc(readFileFunc func(string) ([]byte, error)) ConfigOption {
	return func(cfg *Config) {
		cfg.ReadFileFunc = readFileFunc
	}
}
