// Package thinking provides Crush's pending-request animation as a reusable
// Bubble Tea component.
package thinking

import (
	"fmt"
	"image/color"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/exp/charmtone"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/zeebo/xxh3"
)

const (
	fps           = 20
	initialChar   = '.'
	labelGap      = " "
	labelGapWidth = 1

	// If the FPS is 20 (50 milliseconds) this means that the ellipsis will
	// change every 8 frames (400 milliseconds).
	ellipsisAnimSpeed = 8

	// The maximum amount of time that can pass before a character appears.
	// This is used to create a staggered entrance effect.
	maxBirthOffset = time.Second

	// Number of frames to prerender for the animation. After this number of
	// frames, the animation will loop. This only applies when color cycling is
	// disabled.
	prerenderedFrames = 10

	// Default number of cycling chars.
	defaultNumCyclingChars = 10
)

var (
	availableRunes = []rune("0123456789abcdefABCDEF~!@#$£€%^&*()+=_")
	ellipsisFrames = []string{".", "..", "...", ""}
)

// Internal ID management. Used during animating to ensure that frame messages
// are received only by animation components that sent them.
var lastID int64

func nextID() int {
	return int(atomic.AddInt64(&lastID, 1))
}

type animCache struct {
	initialFrames  [][]string
	cyclingFrames  [][]string
	width          int
	labelWidth     int
	label          []string
	ellipsisFrames []string
}

var animCacheMap sync.Map

// Colors groups the animation's configurable colors.
type Colors struct {
	GradientA color.Color
	GradientB color.Color
	Label     color.Color
}

// DefaultColors returns the same color palette Crush uses today for pending
// and thinking animations.
func DefaultColors() Colors {
	return Colors{
		GradientA: charmtone.Charple,
		GradientB: charmtone.Dolly,
		Label:     charmtone.Ash,
	}
}

func settingsHash(opts Settings) string {
	h := xxh3.New()
	fmt.Fprintf(h, "%d-%s-%v-%v-%v-%t",
		opts.Size, opts.Label, opts.LabelColor, opts.GradColorA, opts.GradColorB, opts.CycleColors)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func withDefaultColors(opts Settings) Settings {
	defaults := DefaultColors()
	if colorIsUnset(opts.GradColorA) {
		opts.GradColorA = defaults.GradientA
	}
	if colorIsUnset(opts.GradColorB) {
		opts.GradColorB = defaults.GradientB
	}
	if colorIsUnset(opts.LabelColor) {
		opts.LabelColor = defaults.Label
	}
	return opts
}

// StepMsg is a message type used to trigger the next step in the animation.
type StepMsg struct{ ID string }

// Settings defines animation settings.
type Settings struct {
	ID          string
	Size        int
	Label       string
	LabelColor  color.Color
	GradColorA  color.Color
	GradColorB  color.Color
	CycleColors bool
}

// Anim is a Bubble Tea component for Crush's pending/thinking animation.
type Anim struct {
	mu               sync.RWMutex
	width            int
	cyclingCharWidth int
	label            []string
	labelWidth       int
	labelColor       color.Color
	startTime        time.Time
	birthOffsets     []time.Duration
	initialFrames    [][]string
	initialized      atomic.Bool
	cyclingFrames    [][]string
	step             atomic.Int64
	ellipsisStep     atomic.Int64
	ellipsisFrames   []string
	id               string
}

// New creates a new animation instance with the specified settings.
func New(opts Settings) *Anim {
	a := &Anim{}

	if opts.Size < 1 {
		opts.Size = defaultNumCyclingChars
	}
	opts = withDefaultColors(opts)

	if opts.ID != "" {
		a.id = opts.ID
	} else {
		a.id = fmt.Sprintf("%d", nextID())
	}
	a.startTime = time.Now()
	a.cyclingCharWidth = opts.Size
	a.labelColor = opts.LabelColor

	cacheKey := settingsHash(opts)
	if cached, ok := loadAnimCache(cacheKey); ok {
		a.width = cached.width
		a.labelWidth = cached.labelWidth
		a.label = slices.Clone(cached.label)
		a.ellipsisFrames = slices.Clone(cached.ellipsisFrames)
		a.initialFrames = cached.initialFrames
		a.cyclingFrames = cached.cyclingFrames
	} else {
		a.labelWidth = lipgloss.Width(opts.Label)
		a.width = opts.Size
		if opts.Label != "" {
			a.width += labelGapWidth + a.labelWidth
		}

		a.renderLabel(opts.Label)

		var ramp []color.Color
		numFrames := prerenderedFrames
		if opts.CycleColors {
			ramp = makeGradientRamp(a.width*3, opts.GradColorA, opts.GradColorB, opts.GradColorA, opts.GradColorB)
			numFrames = a.width * 2
		} else {
			ramp = makeGradientRamp(a.width, opts.GradColorA, opts.GradColorB)
		}

		a.initialFrames = make([][]string, numFrames)
		offset := 0
		for i := range a.initialFrames {
			a.initialFrames[i] = make([]string, a.width+labelGapWidth+a.labelWidth)
			for j := range a.initialFrames[i] {
				if j+offset >= len(ramp) {
					continue
				}

				c := opts.LabelColor
				if j <= a.cyclingCharWidth {
					c = ramp[j+offset]
				}

				a.initialFrames[i][j] = lipgloss.NewStyle().
					Foreground(c).
					Render(string(initialChar))
			}
			if opts.CycleColors {
				offset++
			}
		}

		a.cyclingFrames = make([][]string, numFrames)
		offset = 0
		for i := range a.cyclingFrames {
			a.cyclingFrames[i] = make([]string, a.width)
			for j := range a.cyclingFrames[i] {
				if j+offset >= len(ramp) {
					continue
				}

				r := availableRunes[rand.IntN(len(availableRunes))]
				a.cyclingFrames[i][j] = lipgloss.NewStyle().
					Foreground(ramp[j+offset]).
					Render(string(r))
			}
			if opts.CycleColors {
				offset++
			}
		}

		storeAnimCache(cacheKey, &animCache{
			initialFrames:  a.initialFrames,
			cyclingFrames:  a.cyclingFrames,
			width:          a.width,
			labelWidth:     a.labelWidth,
			label:          slices.Clone(a.label),
			ellipsisFrames: slices.Clone(a.ellipsisFrames),
		})
	}

	a.birthOffsets = make([]time.Duration, a.width)
	for i := range a.birthOffsets {
		a.birthOffsets[i] = time.Duration(rand.N(int64(maxBirthOffset))) * time.Nanosecond
	}

	return a
}

func loadAnimCache(key string) (*animCache, bool) {
	cached, ok := animCacheMap.Load(key)
	if !ok {
		return nil, false
	}
	animCache, ok := cached.(*animCache)
	return animCache, ok
}

func storeAnimCache(key string, cached *animCache) {
	animCacheMap.Store(key, cached)
}

// SetLabel updates the label text and re-renders it.
func (a *Anim) SetLabel(newLabel string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.labelWidth = lipgloss.Width(newLabel)
	a.width = a.cyclingCharWidth
	if newLabel != "" {
		a.width += labelGapWidth + a.labelWidth
	}

	a.renderLabel(newLabel)
}

func (a *Anim) renderLabel(label string) {
	if a.labelWidth > 0 {
		labelRunes := []rune(label)
		a.label = make([]string, 0, len(labelRunes))
		for i := range labelRunes {
			rendered := lipgloss.NewStyle().
				Foreground(a.labelColor).
				Render(string(labelRunes[i]))
			a.label = append(a.label, rendered)
		}

		a.ellipsisFrames = make([]string, 0, len(ellipsisFrames))
		for _, frame := range ellipsisFrames {
			rendered := lipgloss.NewStyle().
				Foreground(a.labelColor).
				Render(frame)
			a.ellipsisFrames = append(a.ellipsisFrames, rendered)
		}
		return
	}

	a.label = nil
	a.ellipsisFrames = nil
}

// Width returns the total width of the animation, including the animated
// ellipsis when a label is present.
func (a *Anim) Width() int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	w := a.cyclingCharWidth
	if a.labelWidth > 0 {
		w += labelGapWidth + a.labelWidth

		var widestEllipsisFrame int
		for _, frame := range ellipsisFrames {
			widestEllipsisFrame = max(widestEllipsisFrame, lipgloss.Width(frame))
		}
		w += widestEllipsisFrame
	}
	return w
}

// Start starts the animation.
func (a *Anim) Start() tea.Cmd {
	return a.Step()
}

// Animate advances the animation to the next step.
func (a *Anim) Animate(msg StepMsg) tea.Cmd {
	if msg.ID != a.id {
		return nil
	}

	step := a.step.Add(1)
	if int(step) >= len(a.cyclingFrames) {
		a.step.Store(0)
	}

	a.mu.RLock()
	labelWidth := a.labelWidth
	a.mu.RUnlock()

	if a.initialized.Load() && labelWidth > 0 {
		ellipsisStep := a.ellipsisStep.Add(1)
		if int(ellipsisStep) >= ellipsisAnimSpeed*len(ellipsisFrames) {
			a.ellipsisStep.Store(0)
		}
	} else if !a.initialized.Load() && time.Since(a.startTime) >= maxBirthOffset {
		a.initialized.Store(true)
	}
	return a.Step()
}

// Render renders the current state of the animation.
func (a *Anim) Render() string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var b strings.Builder
	step := int(a.step.Load())
	for i := range a.width {
		switch {
		case !a.initialized.Load() && i < len(a.birthOffsets) && time.Since(a.startTime) < a.birthOffsets[i]:
			b.WriteString(a.initialFrames[step][i])
		case i < a.cyclingCharWidth:
			b.WriteString(a.cyclingFrames[step][i])
		case i == a.cyclingCharWidth:
			b.WriteString(labelGap)
		case i > a.cyclingCharWidth:
			labelIdx := i - a.cyclingCharWidth - labelGapWidth
			if labelIdx >= 0 && labelIdx < len(a.label) {
				b.WriteString(a.label[labelIdx])
			}
		}
	}

	if a.initialized.Load() && a.labelWidth > 0 {
		ellipsisStep := int(a.ellipsisStep.Load())
		frameIdx := ellipsisStep / ellipsisAnimSpeed
		if frameIdx >= 0 && frameIdx < len(a.ellipsisFrames) {
			b.WriteString(a.ellipsisFrames[frameIdx])
		}
	}

	return b.String()
}

// Step is a command that triggers the next step in the animation.
func (a *Anim) Step() tea.Cmd {
	return tea.Tick(time.Second/time.Duration(fps), func(t time.Time) tea.Msg {
		return StepMsg{ID: a.id}
	})
}

// makeGradientRamp returns a slice of colors blended between the given keys.
// Blending is done as Hcl to stay in gamut.
func makeGradientRamp(size int, stops ...color.Color) []color.Color {
	if len(stops) < 2 {
		return nil
	}

	points := make([]colorful.Color, len(stops))
	for i, stop := range stops {
		points[i], _ = colorful.MakeColor(stop)
	}

	numSegments := len(stops) - 1
	if numSegments == 0 {
		return nil
	}
	blended := make([]color.Color, 0, size)

	segmentSizes := make([]int, numSegments)
	baseSize := size / numSegments
	remainder := size % numSegments
	for i := range numSegments {
		segmentSizes[i] = baseSize
		if i < remainder {
			segmentSizes[i]++
		}
	}

	for i := range numSegments {
		c1 := points[i]
		c2 := points[i+1]
		segmentSize := segmentSizes[i]

		for j := range segmentSize {
			if segmentSize == 0 {
				continue
			}
			t := float64(j) / float64(segmentSize)
			c := c1.BlendHcl(c2, t)
			blended = append(blended, c)
		}
	}

	return blended
}

func colorIsUnset(c color.Color) bool {
	if c == nil {
		return true
	}
	_, _, _, a := c.RGBA()
	return a == 0
}
