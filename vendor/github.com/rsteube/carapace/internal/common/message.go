package common

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/rsteube/carapace/pkg/style"
)

type Messages struct {
	messages map[string]bool
}

func (m *Messages) init() {
	if m.messages == nil {
		m.messages = make(map[string]bool)
	}
}

func (m Messages) IsEmpty() bool {
	// TODO replacement for Action.skipCache - does this need to consider suppressed messages or is this fine?
	return len(m.messages) == 0
}

func (m *Messages) Add(s string) {
	m.init()
	m.messages[s] = true
}

func (m Messages) Get() []string {
	messages := make([]string, 0)
	for message := range m.messages {
		messages = append(messages, message)
	}
	sort.Strings(messages)
	return messages
}

func (m *Messages) Suppress(expr ...string) error {
	m.init()

	for _, e := range expr {
		r, err := regexp.Compile(e)
		if err != nil {
			return err
		}

		for key := range m.messages {
			if r.MatchString(key) {
				delete(m.messages, key)
			}
		}
	}
	return nil
}

func (m *Messages) Merge(other Messages) {
	if other.messages == nil {
		return
	}

	for key := range other.messages {
		m.Add(key)
	}
}

func (m Messages) Integrate(values RawValues, prefix string) RawValues {
	m.init()

	if len(m.messages) == 0 {
		return values
	}

	sorted := make([]string, 0)
	for message := range m.messages {
		sorted = append(sorted, message)
	}
	sort.Strings(sorted)

	switch {
	case strings.HasSuffix(prefix, "ERR"):
		prefix = strings.TrimSuffix(prefix, "ERR")
	case strings.HasSuffix(prefix, "ER"):
		prefix = strings.TrimSuffix(prefix, "ER")
	case strings.HasSuffix(prefix, "E"):
		prefix = strings.TrimSuffix(prefix, "E")
	}

	i := 0
	for _, message := range sorted {
		value := prefix + "ERR"
		display := "ERR"
		for {
			if i > 0 {
				value = fmt.Sprintf("%vERR%v", prefix, i)
				display = fmt.Sprintf("ERR%v", i)
			}
			i += 1

			if !values.contains(value) {
				break
			}
		}

		values = append(values, RawValue{
			Value:       value,
			Display:     display,
			Description: message,
			Style:       style.Carapace.Error,
		})
	}

	if len(values) == 1 {
		values = append(values, RawValue{
			Value:       prefix + "_",
			Display:     "_",
			Description: "",
			Style:       style.Default,
		})
	}
	sort.Sort(ByDisplay(values))
	return values
}

func (m Messages) MarshalJSON() ([]byte, error) {
	var result []string = make([]string, 0, len(m.messages))
	for key := range m.messages {
		result = append(result, key)
	}
	sort.Strings(result)
	return json.Marshal(&result)
}

func (m *Messages) UnmarshalJSON(data []byte) (err error) {
	var result []string
	if err = json.Unmarshal(data, &result); err != nil {
		return err
	}
	for _, item := range result {
		m.Add(item)
	}
	return
}
