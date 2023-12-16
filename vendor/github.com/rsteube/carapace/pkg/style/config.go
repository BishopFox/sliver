package style

import (
	"github.com/rsteube/carapace/internal/config"
)

// Register a style configuration
//
//	var Carapace = struct {
//		Value       string `description:"default style for values"`
//		Description string `description:"default style for descriptions"`
//	}{
//		Value:       Default,
//		Description: Gray,
//	}
//
//	func init() {
//		Register("carapace", &Carapace)
//	}
func Register(name string, i interface{}) { config.RegisterStyle(name, i) }

// Set a style
//
//	Set("carapace.Value", "bold magenta")
func Set(key, value string) error { return config.SetStyle(key, value) }

type carapace struct {
	Value       string `description:"default style for values" tag:"core styles"`
	Description string `description:"default style for descriptions" tag:"core styles"`
	Error       string `description:"default style for errors" tag:"core styles"`
	Usage       string `description:"default style for usage" tag:"core styles"`

	KeywordAmbiguous string `description:"keyword describing a ambiguous state" tag:"keyword styles"`
	KeywordNegative  string `description:"keyword describing a negative state" tag:"keyword styles"`
	KeywordPositive  string `description:"keyword describing a positive state" tag:"keyword styles"`
	KeywordUnknown   string `description:"keyword describing an unknown state" tag:"keyword styles"`

	LogLevelTrace    string `description:"LogLevel TRACE" tag:"loglevel styles"`
	LogLevelDebug    string `description:"LogLevel DEBUG" tag:"loglevel styles"`
	LogLevelInfo     string `description:"LogLevel INFO" tag:"loglevel styles"`
	LogLevelWarning  string `description:"LogLevel WARNING" tag:"loglevel styles"`
	LogLevelError    string `description:"LogLevel ERROR" tag:"loglevel styles"`
	LogLevelCritical string `description:"LogLevel CRITICAL" tag:"loglevel styles"`
	LogLevelFatal    string `description:"LogLevel FATAL" tag:"loglevel styles"`

	Highlight1  string `description:"Highlight 1" tag:"highlight styles"`
	Highlight2  string `description:"Highlight 2" tag:"highlight styles"`
	Highlight3  string `description:"Highlight 3" tag:"highlight styles"`
	Highlight4  string `description:"Highlight 4" tag:"highlight styles"`
	Highlight5  string `description:"Highlight 5" tag:"highlight styles"`
	Highlight6  string `description:"Highlight 6" tag:"highlight styles"`
	Highlight7  string `description:"Highlight 7" tag:"highlight styles"`
	Highlight8  string `description:"Highlight 8" tag:"highlight styles"`
	Highlight9  string `description:"Highlight 9" tag:"highlight styles"`
	Highlight10 string `description:"Highlight 10" tag:"highlight styles"`
	Highlight11 string `description:"Highlight 11" tag:"highlight styles"`
	Highlight12 string `description:"Highlight 12" tag:"highlight styles"`

	FlagArg      string `description:"flag with argument" tag:"flag styles"`
	FlagMultiArg string `description:"flag with multiple arguments" tag:"flag styles"`
	FlagNoArg    string `description:"flag without argument" tag:"flag styles"`
	FlagOptArg   string `description:"flag with optional argument" tag:"flag styles"`
}

var Carapace = carapace{
	Value:       Default,
	Description: Dim,
	Error:       Of(Bold, Red),
	Usage:       Dim,

	KeywordAmbiguous: Yellow,
	KeywordNegative:  Red,
	KeywordPositive:  Green,
	KeywordUnknown:   Of(Dim, White),

	LogLevelTrace:    Blue,
	LogLevelDebug:    Of(Dim, White),
	LogLevelInfo:     Green,
	LogLevelWarning:  Yellow,
	LogLevelError:    Magenta,
	LogLevelCritical: Red,
	LogLevelFatal:    Cyan,

	Highlight1:  Blue,
	Highlight2:  Yellow,
	Highlight3:  Magenta,
	Highlight4:  Cyan,
	Highlight5:  Green,
	Highlight6:  Of(Dim, Blue),
	Highlight7:  Of(Dim, Yellow),
	Highlight8:  Of(Dim, Magenta),
	Highlight9:  Of(Dim, Cyan),
	Highlight10: Of(Dim, Green),
	Highlight11: Bold,
	Highlight12: Of(Dim, Bold),

	FlagArg:      Blue,
	FlagMultiArg: Magenta,
	FlagNoArg:    Default,
	FlagOptArg:   Yellow,
}

// Highlight returns the style for given level (0..n)
func (c carapace) Highlight(level int) string {
	switch level {
	case 0:
		return c.Highlight1
	case 1:
		return c.Highlight2
	case 2:
		return c.Highlight3
	case 3:
		return c.Highlight4
	case 4:
		return c.Highlight5
	case 5:
		return c.Highlight6
	case 6:
		return c.Highlight7
	case 7:
		return c.Highlight8
	case 8:
		return c.Highlight9
	case 9:
		return c.Highlight10
	case 10:
		return c.Highlight11
	case 11:
		return c.Highlight12
	default:
		return Default
	}
}

func init() {
	Register("carapace", &Carapace)
}
