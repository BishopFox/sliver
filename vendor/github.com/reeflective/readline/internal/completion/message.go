package completion

import (
	"regexp"
	"sort"
)

// Messages is a list of messages to be displayed
// below the input line, above completions. It is
// used to show usage and/or error status hints.
type Messages struct {
	messages map[string]bool
}

func (m *Messages) init() {
	if m.messages == nil {
		m.messages = make(map[string]bool)
	}
}

// IsEmpty returns true if there are no messages to display.
func (m Messages) IsEmpty() bool {
	// TODO replacement for Action.skipCache - does this need to consider suppressed messages or is this fine?
	return len(m.messages) == 0
}

// Add adds a message to the list of messages.
func (m *Messages) Add(s string) {
	m.init()
	m.messages[s] = true
}

// Get returns the list of messages to display.
func (m Messages) Get() []string {
	messages := make([]string, 0)
	for message := range m.messages {
		messages = append(messages, message)
	}

	sort.Strings(messages)

	return messages
}

// Suppress removes messages matching the given regular expressions from the list of messages.
func (m *Messages) Suppress(expr ...string) error {
	m.init()

	for _, e := range expr {
		char, err := regexp.Compile(e)
		if err != nil {
			return err
		}

		for key := range m.messages {
			if char.MatchString(key) {
				delete(m.messages, key)
			}
		}
	}

	return nil
}

// Merge merges the given messages into the current list of messages.
func (m *Messages) Merge(other Messages) {
	if other.messages == nil {
		return
	}

	for key := range other.messages {
		m.Add(key)
	}
}
