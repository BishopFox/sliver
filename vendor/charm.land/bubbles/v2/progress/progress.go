package progress

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	DefaultFullCharHalfBlock = '▌'
	DefaultFullCharFullBlock = '█'
	DefaultEmptyCharBlock    = '░'

	defaultWidth = 40
)

var (
	defaultFullColor  = lipgloss.Color("#7571F9")
	defaultEmptyColor = lipgloss.Color("#606060")
)

type Option func(*Model)

func WithColors(colors ...color.Color) Option {
	if len(colors) == 0 {
		return func(m *Model) {
			m.FullColor = defaultFullColor
			m.blend = nil
		}
	}
	if len(colors) == 1 {
		return func(m *Model) {
			m.FullColor = colors[0]
			m.blend = nil
		}
	}
	return func(m *Model) {
		m.blend = colors
	}
}

func WithFillCharacters(full, empty rune) Option {
	return func(m *Model) {
		m.Full = full
		m.Empty = empty
	}
}

func WithoutPercentage() Option {
	return func(m *Model) {
		m.ShowPercentage = false
	}
}

func WithWidth(w int) Option {
	return func(m *Model) {
		m.SetWidth(w)
	}
}

type Model struct {
	width           int
	Full            rune
	FullColor       color.Color
	Empty           rune
	EmptyColor      color.Color
	ShowPercentage  bool
	PercentFormat   string
	PercentageStyle lipgloss.Style
	blend           []color.Color
}

func New(opts ...Option) Model {
	m := Model{
		width:          defaultWidth,
		Full:           DefaultFullCharHalfBlock,
		FullColor:      defaultFullColor,
		Empty:          DefaultEmptyCharBlock,
		EmptyColor:     defaultEmptyColor,
		ShowPercentage: true,
		PercentFormat:  " %3.0f%%",
	}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

func (m *Model) SetWidth(w int) {
	m.width = w
}

func (m Model) Width() int {
	return m.width
}

func (m Model) View() string {
	return m.ViewAs(0)
}

func (m Model) ViewAs(percent float64) string {
	b := strings.Builder{}
	percentView := m.percentageView(percent)
	m.barView(&b, percent, ansi.StringWidth(percentView))
	b.WriteString(percentView)
	return b.String()
}

func (m Model) percentageView(percent float64) string {
	if !m.ShowPercentage {
		return ""
	}
	percent = math.Max(0, math.Min(1, percent))
	percentage := fmt.Sprintf(m.PercentFormat, percent*100)
	return m.PercentageStyle.Inline(true).Render(percentage)
}

func (m Model) barView(b *strings.Builder, percent float64, textWidth int) {
	totalWidth := maxInt(0, m.width-textWidth)
	filledWidth := int(math.Round(float64(totalWidth) * math.Max(0, math.Min(1, percent))))
	filledWidth = clampInt(filledWidth, 0, totalWidth)

	if len(m.blend) > 1 {
		blend := lipgloss.Blend1D(totalWidth, m.blend...)
		for i := 0; i < filledWidth && i < len(blend); i++ {
			b.WriteString(lipgloss.NewStyle().
				Foreground(blend[i]).
				Render(string(m.Full)))
		}
	} else {
		b.WriteString(lipgloss.NewStyle().
			Foreground(m.FullColor).
			Render(strings.Repeat(string(m.Full), filledWidth)))
	}

	emptyWidth := maxInt(0, totalWidth-filledWidth)
	if emptyWidth == 0 {
		return
	}
	b.WriteString(lipgloss.NewStyle().
		Foreground(m.EmptyColor).
		Render(strings.Repeat(string(m.Empty), emptyWidth)))
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
