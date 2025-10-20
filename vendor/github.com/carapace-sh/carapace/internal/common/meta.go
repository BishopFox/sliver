package common

type Meta struct {
	Messages Messages      `json:"messages"`
	Nospace  SuffixMatcher `json:"nospace"`
	Usage    string        `json:"usage"`
	Queries  Queries       `json:"queries,omitempty"`
}

func (m *Meta) Merge(other Meta) {
	if other.Usage != "" {
		m.Usage = other.Usage
	}
	m.Nospace.Merge(other.Nospace)
	m.Messages.Merge(other.Messages)
	m.Queries.Merge(other.Queries)
}
