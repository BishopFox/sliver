package spec

type Command struct {
	Name            string            `yaml:"name"`
	Aliases         []string          `yaml:"aliases,omitempty"`
	Description     string            `yaml:"description,omitempty"`
	Group           string            `yaml:"group,omitempty"`
	Hidden          bool              `yaml:"hidden,omitempty"`
	ExclusiveFlags  [][]string        `yaml:"exclusiveflags,omitempty"`
	Flags           map[string]string `yaml:"flags,omitempty"`
	PersistentFlags map[string]string `yaml:"persistentflags,omitempty"`
	Completion      struct {
		Flag          map[string][]string `yaml:"flag,omitempty"`
		Positional    [][]string          `yaml:"positional,omitempty"`
		PositionalAny []string            `yaml:"positionalany,omitempty"`
		Dash          [][]string          `yaml:"dash,omitempty"`
		DashAny       []string            `yaml:"dashany,omitempty"`
	} `yaml:"completion,omitempty"`
	Commands []Command `yaml:"commands,omitempty"`
}
