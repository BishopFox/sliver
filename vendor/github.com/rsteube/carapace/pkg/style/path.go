package style

import (
	"github.com/rsteube/carapace/third_party/github.com/elves/elvish/pkg/cli/lscolors"
	"github.com/rsteube/carapace/third_party/github.com/elves/elvish/pkg/ui"
)

type Context interface {
	Abs(s string) (string, error)
	Getenv(key string) string
	LookupEnv(key string) (string, bool)
}

// ForPath returns the style for given path
//
//	/tmp/locally/reachable/file.txt
func ForPath(path string, sc Context) string {
	if abs, err := sc.Abs(path); err == nil {
		path = abs
	}
	return fromSGR(lscolors.GetColorist(sc.Getenv("LS_COLORS")).GetStyle(path))
}

// ForPath returns the style for given path by extension only
//
//	/tmp/non/existing/file.txt
func ForPathExt(path string, sc Context) string {
	return fromSGR(lscolors.GetColorist(sc.Getenv("LS_COLORS")).GetStyleExt(path))
}

// ForExtension returns the style for given extension
//
//	json
func ForExtension(path string, sc Context) string {
	return ForPathExt("."+path, sc)
}

func fromSGR(sgr string) string {
	s := ui.StyleFromSGR(sgr)
	result := []string{}
	if s.Foreground != nil {
		result = append(result, s.Foreground.String())
	}
	if s.Background != nil {
		result = append(result, "bg-"+s.Background.String())
	}
	if s.Bold {
		result = append(result, Bold)
	}
	if s.Dim {
		result = append(result, Dim)
	}
	if s.Italic {
		result = append(result, Italic)
	}
	if s.Underlined {
		result = append(result, Underlined)
	}
	if s.Blink {
		result = append(result, Blink)
	}
	if s.Inverse {
		result = append(result, Inverse)
	}
	return Of(result...)
}
