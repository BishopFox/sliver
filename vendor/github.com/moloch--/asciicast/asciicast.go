package asciicast

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image/color"
	"strings"
)

// Version 2 is supposed by this package.
const Version = 2

// ContentType is the asciicast v2 MIME type.
const ContentType = "application/x-asciicast"

// Header contains contains recording meta-data.
type Header struct {
	Version       int               `json:"version"`
	Width         int               `json:"width"`
	Height        int               `json:"height"`
	Timestamp     int64             `json:"timestamp,omitempty"`
	Duration      float64           `json:"duration,omitempty"`
	IdleTimeLimit float64           `json:"idle_time_limit,omitempty"`
	Command       string            `json:"command,omitempty"`
	Title         string            `json:"title,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	Theme         *Theme            `json:"theme,omitempty"`
}

type EventType string

const (
	// Output signals data written to standard output.
	Output EventType = "o"

	// Input signals data read from standard input.
	Input EventType = "i"
)

type Event struct {
	Time float64
	Type EventType
	Data string
}

func (e Event) MarshalJSON() ([]byte, error) {
	return json.Marshal([]interface{}{e.Time, e.Type, e.Data})
}

func (e *Event) UnmarshalJSON(b []byte) error {
	var raw []interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	} else if l := len(raw); l != 3 {
		return fmt.Errorf("asciicast: expected a 3 value array, got %d value(s)", l)
	}

	var (
		eventTime float64
		eventType string
		eventData string
		ok        bool
	)
	if eventTime, ok = raw[0].(float64); !ok {
		return fmt.Errorf("asciicast: expected a float64 time, got %T", raw[0])
	}
	if eventType, ok = raw[1].(string); !ok {
		return fmt.Errorf("asciicast: expected a string event-type, got %T", raw[1])
	} else if eventType != "o" && eventType != "i" {
		return fmt.Errorf("asciicast: unexpected event-type %q", eventType)
	}
	if eventData, ok = raw[2].(string); !ok {
		return fmt.Errorf("asciicast: expected a string data, got %T", raw[2])
	}
	e.Time = eventTime
	e.Type = EventType(eventType)
	e.Data = eventData
	return nil
}

// Theme of the recorded terminal.
type Theme struct {
	Foreground string  `json:"fg"`
	Background string  `json:"bg"`
	Palette    Palette `json:"palette"`
}

type Palette color.Palette

func (p Palette) MarshalJSON() ([]byte, error) {
	if l := len(p); l != 8 && l != 16 {
		return nil, fmt.Errorf("asciicast: palette must have 8 or 16 colors, got %d", l)
	}
	h := make([]string, len(p))
	for i, c := range p {
		r, g, b, _ := c.RGBA()
		h[i] = fmt.Sprintf("#%02x%02x%02x", (r>>8)&0xff, (g>>8)&0xff, (b>>8)&0xff)
	}
	return json.Marshal(strings.Join(h, ":"))
}

func (p *Palette) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	h := strings.Split(s, ":")
	if l := len(h); l != 8 && l != 16 {
		return fmt.Errorf("asciicast: palette must have 8 or 16 colors, got %d", l)
	}
	*p = make(Palette, len(h))
	for i, v := range h {
		b, err := hex.DecodeString(v[1:])
		if err != nil {
			return err
		} else if len(b) != 3 {
			return fmt.Errorf("asciicast: not an rgb hex, expected 3 values, got %d", len(b))
		}
		(*p)[i] = color.RGBA{R: b[0], G: b[1], B: b[2]}
	}
	return nil
}
