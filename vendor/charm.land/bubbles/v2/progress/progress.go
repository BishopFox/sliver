// Package progress provides a simple progress bar for Bubble Tea applications.
package progress

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/harmonica"
	"github.com/charmbracelet/x/ansi"
)

// ColorFunc is a function that can be used to dynamically fill the progress
// bar based on the current percentage. total is the total filled percentage,
// and current is the current percentage that is actively being filled with a
// color.
type ColorFunc func(total, current float64) color.Color

// Internal ID management. Used during animating to assure that frame messages
// can only be received by progress components that sent them.
var lastID int64

func nextID() int {
	return int(atomic.AddInt64(&lastID, 1))
}

const (
	// DefaultFullCharHalfBlock is the default character used to fill the progress
	// bar. It is a half block, which allows more granular color blending control,
	// by having a different foreground and background color, doubling blending
	// resolution.
	DefaultFullCharHalfBlock = '▌'

	// DefaultFullCharFullBlock can also be used as a fill character for the
	// progress bar. Use this to disable the higher resolution blending which is
	// enabled when using [DefaultFullCharHalfBlock].
	DefaultFullCharFullBlock = '█'

	// DefaultEmptyCharBlock is the default character used to fill the empty
	// portion of the progress bar.
	DefaultEmptyCharBlock = '░'

	fps              = 60
	defaultWidth     = 40
	defaultFrequency = 18.0
	defaultDamping   = 1.0
)

var (
	defaultBlendStart = lipgloss.Color("#5A56E0") // Purple haze.
	defaultBlendEnd   = lipgloss.Color("#EE6FF8") // Neon pink.
	defaultFullColor  = lipgloss.Color("#7571F9") // Blueberry.
	defaultEmptyColor = lipgloss.Color("#606060") // Slate gray.
)

// Option is used to set options in [New]. For example:
//
//	progress := New(
//		WithColors(
//			lipgloss.Color("#5A56E0"),
//			lipgloss.Color("#EE6FF8"),
//		),
//		WithoutPercentage(),
//	)
type Option func(*Model)

// WithDefaultBlend sets a default blend of colors, which is a blend of purple
// haze to neon pink.
func WithDefaultBlend() Option {
	return WithColors(
		defaultBlendStart,
		defaultBlendEnd,
	)
}

// WithColors sets the colors to use to fill the progress bar. Depending on the
// number of colors passed in, will determine whether to use a solid fill or a
// blend of colors.
//
//   - 0 colors: clears all previously set colors, setting them back to defaults.
//   - 1 color: uses a solid fill with the given color.
//   - 2+ colors: uses a blend of the provided colors.
func WithColors(colors ...color.Color) Option {
	if len(colors) == 0 {
		return func(m *Model) {
			m.FullColor = defaultFullColor
			m.blend = nil
			m.colorFunc = nil
		}
	}
	if len(colors) == 1 {
		return func(m *Model) {
			m.FullColor = colors[0]
			m.colorFunc = nil
			m.blend = nil
		}
	}
	return func(m *Model) {
		m.blend = colors
	}
}

// WithColorFunc sets a function that can be used to dynamically fill the progress
// bar based on the current percentage. total is the total filled percentage, and
// current is the current percentage that is actively being filled with a color.
// When specified, this overrides any other defined colors and scaling.
//
// Example: A progress bar that changes color based on the total completed
// percentage:
//
//	WithColorFunc(func(total, current float64) color.Color {
//		if total <= 0.3 {
//			return lipgloss.Color("#FF0000")
//		}
//		if total <= 0.7 {
//			return lipgloss.Color("#00FF00")
//		}
//		return lipgloss.Color("#0000FF")
//	}),
func WithColorFunc(fn ColorFunc) Option {
	return func(m *Model) {
		m.colorFunc = fn
		m.blend = nil
	}
}

// WithFillCharacters sets the characters used to construct the full and empty
// components of the progress bar.
func WithFillCharacters(full rune, empty rune) Option {
	return func(m *Model) {
		m.Full = full
		m.Empty = empty
	}
}

// WithoutPercentage hides the numeric percentage.
func WithoutPercentage() Option {
	return func(m *Model) {
		m.ShowPercentage = false
	}
}

// WithWidth sets the initial width of the progress bar. Note that you can also
// set the width via the Width property, which can come in handy if you're
// waiting for a tea.WindowSizeMsg.
func WithWidth(w int) Option {
	return func(m *Model) {
		m.SetWidth(w)
	}
}

// WithSpringOptions sets the initial frequency and damping options for the
// progress bar's built-in spring-based animation. Frequency corresponds to
// speed, and damping to bounciness. For details see:
//
// https://github.com/charmbracelet/harmonica
func WithSpringOptions(frequency, damping float64) Option {
	return func(m *Model) {
		m.SetSpringOptions(frequency, damping)
		m.springCustomized = true
	}
}

// WithScaled sets whether to scale the blend/gradient to fit the width of only
// the filled portion of the progress bar. The default is false, which means the
// percentage must be 100% to see the full color blend/gradient.
//
// This is ignored when not using blending/multiple colors.
func WithScaled(enabled bool) Option {
	return func(m *Model) {
		m.scaleBlend = enabled
	}
}

// FrameMsg indicates that an animation step should occur.
type FrameMsg struct {
	id  int
	tag int
}

// Model stores values we'll use when rendering the progress bar.
type Model struct {
	// An identifier to keep us from receiving messages intended for other
	// progress bars.
	id int

	// An identifier to keep us from receiving frame messages too quickly.
	tag int

	// Total width of the progress bar, including percentage, if set.
	width int

	// "Filled" sections of the progress bar.
	Full      rune
	FullColor color.Color

	// "Empty" sections of the progress bar.
	Empty      rune
	EmptyColor color.Color

	// Settings for rendering the numeric percentage.
	ShowPercentage  bool
	PercentFormat   string // a fmt string for a float
	PercentageStyle lipgloss.Style

	// Members for animated transitions.
	spring           harmonica.Spring
	springCustomized bool
	percentShown     float64 // percent currently displaying
	targetPercent    float64 // percent to which we're animating
	velocity         float64

	// Blend of colors to use. When len < 1, we use FullColor.
	blend []color.Color

	// When true, we scale the blended colors to fit the width of the filled
	// section of the progress bar. When false, the width of the blend will be
	// set to the full width of the progress bar.
	scaleBlend bool

	// colorFunc is used to dynamically fill the progress bar based on the
	// current percentage.
	colorFunc ColorFunc
}

// New returns a model with default values.
func New(opts ...Option) Model {
	m := Model{
		id:             nextID(),
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

	if !m.springCustomized {
		m.SetSpringOptions(defaultFrequency, defaultDamping)
	}

	return m
}

// Init exists to satisfy the tea.Model interface.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update is used to animate the progress bar during transitions. Use
// SetPercent to create the command you'll need to trigger the animation.
//
// If you're rendering with ViewAs you won't need this.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case FrameMsg:
		if msg.id != m.id || msg.tag != m.tag {
			return m, nil
		}

		// If we've more or less reached equilibrium, stop updating.
		if !m.IsAnimating() {
			return m, nil
		}

		m.percentShown, m.velocity = m.spring.Update(m.percentShown, m.velocity, m.targetPercent)
		return m, m.nextFrame()

	default:
		return m, nil
	}
}

// SetSpringOptions sets the frequency and damping for the current spring.
// Frequency corresponds to speed, and damping to bounciness. For details see:
//
// https://github.com/charmbracelet/harmonica
func (m *Model) SetSpringOptions(frequency, damping float64) {
	m.spring = harmonica.NewSpring(harmonica.FPS(fps), frequency, damping)
}

// Percent returns the current visible percentage on the model. This is only
// relevant when you're animating the progress bar.
//
// If you're rendering with ViewAs you won't need this.
func (m Model) Percent() float64 {
	return m.targetPercent
}

// SetPercent sets the percentage state of the model as well as a command
// necessary for animating the progress bar to this new percentage.
//
// If you're rendering with ViewAs you won't need this.
func (m *Model) SetPercent(p float64) tea.Cmd {
	m.targetPercent = math.Max(0, math.Min(1, p))
	m.tag++
	return m.nextFrame()
}

// IncrPercent increments the percentage by a given amount, returning a command
// necessary to animate the progress bar to the new percentage.
//
// If you're rendering with ViewAs you won't need this.
func (m *Model) IncrPercent(v float64) tea.Cmd {
	return m.SetPercent(m.Percent() + v)
}

// DecrPercent decrements the percentage by a given amount, returning a command
// necessary to animate the progress bar to the new percentage.
//
// If you're rendering with ViewAs you won't need this.
func (m *Model) DecrPercent(v float64) tea.Cmd {
	return m.SetPercent(m.Percent() - v)
}

// View renders an animated progress bar in its current state. To render
// a static progress bar based on your own calculations use ViewAs instead.
func (m Model) View() string {
	return m.ViewAs(m.percentShown)
}

// ViewAs renders the progress bar with a given percentage.
func (m Model) ViewAs(percent float64) string {
	b := strings.Builder{}
	percentView := m.percentageView(percent)
	m.barView(&b, percent, ansi.StringWidth(percentView))
	b.WriteString(percentView)
	return b.String()
}

// SetWidth sets the width of the progress bar.
func (m *Model) SetWidth(w int) {
	m.width = w
}

// Width returns the width of the progress bar.
func (m Model) Width() int {
	return m.width
}

func (m *Model) nextFrame() tea.Cmd {
	return tea.Tick(time.Second/time.Duration(fps), func(time.Time) tea.Msg {
		return FrameMsg{id: m.id, tag: m.tag}
	})
}

func (m Model) barView(b *strings.Builder, percent float64, textWidth int) {
	var (
		tw = max(0, m.width-textWidth)                // total width
		fw = int(math.Round((float64(tw) * percent))) // filled width
	)

	fw = max(0, min(tw, fw))

	isHalfBlock := m.Full == DefaultFullCharHalfBlock

	if m.colorFunc != nil { //nolint:nestif
		var style lipgloss.Style
		var current float64
		halfBlockPerc := 0.5 / float64(tw)
		for i := range fw {
			current = float64(i) / float64(tw)
			style = style.Foreground(m.colorFunc(percent, current))
			if isHalfBlock {
				style = style.Background(m.colorFunc(percent, min(current+halfBlockPerc, 1)))
			}
			b.WriteString(style.Render(string(m.Full)))
		}
	} else if len(m.blend) > 0 {
		var blend []color.Color

		multiplier := 1
		if isHalfBlock {
			multiplier = 2
		}

		if m.scaleBlend {
			blend = lipgloss.Blend1D(fw*multiplier, m.blend...)
		} else {
			blend = lipgloss.Blend1D(tw*multiplier, m.blend...)
		}

		// Blend fill.
		var blendIndex int
		for i := range fw {
			if !isHalfBlock {
				b.WriteString(lipgloss.NewStyle().
					Foreground(blend[i]).
					Render(string(m.Full)))
				continue
			}

			b.WriteString(lipgloss.NewStyle().
				Foreground(blend[blendIndex]).
				Background(blend[blendIndex+1]).
				Render(string(m.Full)))
			blendIndex += 2
		}
	} else {
		// Solid fill.
		b.WriteString(lipgloss.NewStyle().
			Foreground(m.FullColor).
			Render(strings.Repeat(string(m.Full), fw)))
	}

	// Empty fill.
	n := max(0, tw-fw)
	b.WriteString(lipgloss.NewStyle().
		Foreground(m.EmptyColor).
		Render(strings.Repeat(string(m.Empty), n)))
}

func (m Model) percentageView(percent float64) string {
	if !m.ShowPercentage {
		return ""
	}
	percent = math.Max(0, math.Min(1, percent))
	percentage := fmt.Sprintf(m.PercentFormat, percent*100) //nolint:mnd
	percentage = m.PercentageStyle.Inline(true).Render(percentage)
	return percentage
}

// IsAnimating returns false if the progress bar reached equilibrium and is no
// longer animating.
func (m *Model) IsAnimating() bool {
	dist := math.Abs(m.percentShown - m.targetPercent)
	return !(dist < 0.001 && m.velocity < 0.01)
}
