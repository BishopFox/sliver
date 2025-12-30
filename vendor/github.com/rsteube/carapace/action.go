package carapace

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	shlex "github.com/rsteube/carapace-shlex"
	"github.com/rsteube/carapace/internal/cache"
	"github.com/rsteube/carapace/internal/common"
	"github.com/rsteube/carapace/pkg/cache/key"
	"github.com/rsteube/carapace/pkg/match"
	"github.com/rsteube/carapace/pkg/style"
	pkgtraverse "github.com/rsteube/carapace/pkg/traverse"
)

// Action indicates how to complete a flag or positional argument.
type Action struct {
	meta      common.Meta
	rawValues common.RawValues
	callback  CompletionCallback
}

// ActionMap maps Actions to an identifier.
type ActionMap map[string]Action

// CompletionCallback is executed during completion of associated flag or positional argument.
type CompletionCallback func(c Context) Action

// Cache cashes values of a CompletionCallback for given duration and keys.
func (a Action) Cache(timeout time.Duration, keys ...key.Key) Action {
	if a.callback != nil { // only relevant for callback actions
		cachedCallback := a.callback
		_, file, line, _ := runtime.Caller(1) // generate uid from wherever Cache() was called
		a.callback = func(c Context) Action {
			cacheFile, err := cache.File(file, line, keys...)
			if err != nil {
				return cachedCallback(c)
			}

			if cached, err := cache.LoadE(cacheFile, timeout); err == nil {
				return Action{meta: cached.Meta, rawValues: cached.Values}
			}

			invokedAction := (Action{callback: cachedCallback}).Invoke(c)
			if invokedAction.action.meta.Messages.IsEmpty() {
				if cacheFile, err := cache.File(file, line, keys...); err == nil { // regenerate as cache keys might have changed due to invocation
					_ = cache.WriteE(cacheFile, invokedAction.export())
				}
			}
			return invokedAction.ToA()
		}
	}
	return a
}

// Chdir changes the current working directory to the named directory for the duration of invocation.
func (a Action) Chdir(dir string) Action {
	return ActionCallback(func(c Context) Action {
		abs, err := c.Abs(dir)
		if err != nil {
			return ActionMessage(err.Error())
		}
		if info, err := os.Stat(abs); err != nil {
			return ActionMessage(err.Error())
		} else if !info.IsDir() {
			return ActionMessage("not a directory: %v", abs)
		}
		c.Dir = abs
		return a.Invoke(c).ToA()
	})
}

// ChdirF is like Chdir but uses a function.
func (a Action) ChdirF(f func(tc pkgtraverse.Context) (string, error)) Action {
	return ActionCallback(func(c Context) Action {
		newDir, err := f(c)
		if err != nil {
			return ActionMessage(err.Error())
		}
		return a.Chdir(newDir)
	})
}

// Filter filters given values.
//
//	carapace.ActionValues("A", "B", "C").Filter("B") // ["A", "C"]
func (a Action) Filter(values ...string) Action {
	return ActionCallback(func(c Context) Action {
		return a.Invoke(c).Filter(values...).ToA()
	})
}

// FilterArgs filters Context.Args.
func (a Action) FilterArgs() Action {
	return ActionCallback(func(c Context) Action {
		return a.Filter(c.Args...)
	})
}

// FilterArgs filters Context.Parts.
func (a Action) FilterParts() Action {
	return ActionCallback(func(c Context) Action {
		return a.Filter(c.Parts...)
	})
}

// Invoke executes the callback of an action if it exists (supports nesting).
func (a Action) Invoke(c Context) InvokedAction {
	if c.Args == nil {
		c.Args = []string{}
	}
	if c.Env == nil {
		c.Env = []string{}
	}
	if c.Parts == nil {
		c.Parts = []string{}
	}

	if a.rawValues == nil && a.callback != nil {
		result := a.callback(c).Invoke(c)
		result.action.meta.Merge(a.meta)
		return result
	}
	return InvokedAction{a}
}

// List wraps the Action in an ActionMultiParts with given divider.
func (a Action) List(divider string) Action {
	return ActionMultiParts(divider, func(c Context) Action {
		return a.Invoke(c).ToA().NoSpace()
	})
}

// MultiParts splits values of an Action by given dividers and completes each segment separately.
func (a Action) MultiParts(dividers ...string) Action {
	return ActionCallback(func(c Context) Action {
		return a.Invoke(c).ToMultiPartsA(dividers...)
	})
}

// MultiPartsP is like MultiParts but with placeholders.
func (a Action) MultiPartsP(delimiter string, pattern string, f func(placeholder string, matches map[string]string) Action) Action {
	return ActionCallback(func(c Context) Action {
		invoked := a.Invoke(c)

		return ActionMultiParts(delimiter, func(c Context) Action {
			rPlaceholder := regexp.MustCompile(pattern)
			matchedData := make(map[string]string)
			matchedSegments := make(map[string]common.RawValue)
			staticMatches := make(map[int]bool)

		path:
			for index, value := range invoked.action.rawValues {
				segments := strings.Split(value.Value, delimiter)
			segment:
				for index, segment := range segments {
					if index > len(c.Parts)-1 {
						break segment
					} else {
						if segment != c.Parts[index] {
							if !rPlaceholder.MatchString(segment) {
								continue path // skip this path as it doesn't match and is not a placeholder
							} else {
								matchedData[segment] = c.Parts[index] // store entered data for placeholder (overwrite if duplicate)
							}
						} else {
							staticMatches[index] = true // static segment matches so placeholders should be ignored for this index
						}
					}
				}

				if len(segments) < len(c.Parts)+1 {
					continue path // skip path as it is shorter than what was entered (must be after staticMatches being set)
				}

				for key := range staticMatches {
					if segments[key] != c.Parts[key] {
						continue path // skip this path as it has a placeholder where a static segment was matched
					}
				}

				// store segment as path matched so far and this is currently being completed
				if len(segments) == (len(c.Parts) + 1) {
					matchedSegments[segments[len(c.Parts)]] = invoked.action.rawValues[index]
				} else {
					matchedSegments[segments[len(c.Parts)]+delimiter] = common.RawValue{}
				}
			}

			actions := make([]Action, 0, len(matchedSegments))
			for key, value := range matchedSegments {
				if trimmedKey := strings.TrimSuffix(key, delimiter); rPlaceholder.MatchString(trimmedKey) {
					suffix := ""
					if strings.HasSuffix(key, delimiter) {
						suffix = delimiter
					}
					actions = append(actions, ActionCallback(func(c Context) Action {
						invoked := f(trimmedKey, matchedData).Invoke(c).Suffix(suffix)
						for index := range invoked.action.rawValues {
							invoked.action.rawValues[index].Display += suffix
						}
						return invoked.ToA()
					}))
				} else {
					actions = append(actions, ActionStyledValuesDescribed(key, value.Description, value.Style)) // TODO tag,..
				}
			}

			a := Batch(actions...).ToA()
			a.meta.Merge(invoked.action.meta)
			return a
		})
	})
}

// NoSpace disables space suffix for given characters (or all if none are given).
func (a Action) NoSpace(suffixes ...rune) Action {
	return ActionCallback(func(c Context) Action {
		if len(suffixes) == 0 {
			a.meta.Nospace.Add('*')
		}
		a.meta.Nospace.Add(suffixes...)
		return a
	})
}

// Prefix adds a prefix to values (only the ones inserted, not the display values).
//
//	carapace.ActionValues("melon", "drop", "fall").Prefix("water")
func (a Action) Prefix(prefix string) Action {
	return ActionCallback(func(c Context) Action {
		switch {
		case match.HasPrefix(c.Value, prefix):
			c.Value = match.TrimPrefix(c.Value, prefix)
		case match.HasPrefix(prefix, c.Value):
			c.Value = ""
		default:
			return ActionValues()
		}
		return a.Invoke(c).Prefix(prefix).ToA()
	})
}

// Retain retains given values.
//
//	carapace.ActionValues("A", "B", "C").Retain("A", "C") // ["A", "C"]
func (a Action) Retain(values ...string) Action {
	return ActionCallback(func(c Context) Action {
		return a.Invoke(c).Retain(values...).ToA()
	})
}

// Shift shifts positional arguments left `n` times.
func (a Action) Shift(n int) Action {
	return ActionCallback(func(c Context) Action {
		switch {
		case n < 0:
			return ActionMessage("invalid argument [ActionShift]: %v", n)
		case len(c.Args) < n:
			c.Args = []string{}
		default:
			c.Args = c.Args[n:]
		}
		return a.Invoke(c).ToA()
	})
}

// Split splits `Context.Value` lexicographically and replaces `Context.Args` with the tokens.
func (a Action) Split() Action {
	return a.split(false)
}

// SplitP is like Split but supports pipelines.
func (a Action) SplitP() Action {
	return a.split(true)
}

func (a Action) split(pipelines bool) Action {
	return ActionCallback(func(c Context) Action {
		tokens, err := shlex.Split(c.Value)
		if err != nil {
			return ActionMessage(err.Error())
		}

		var context Context
		if pipelines {
			tokens = tokens.CurrentPipeline()
			context = NewContext(tokens.FilterRedirects().Words().Strings()...)
		} else {
			context = NewContext(tokens.Words().Strings()...)
		}

		originalValue := c.Value
		prefix := originalValue[:tokens.Words().CurrentToken().Index]
		c.Args = context.Args
		c.Parts = []string{}
		c.Value = context.Value

		if pipelines { // support redirects
			if len(tokens) > 1 && tokens[len(tokens)-2].WordbreakType.IsRedirect() {
				LOG.Printf("completing files for redirect arg %#v", tokens.Words().CurrentToken().Value)
				prefix = originalValue[:tokens.CurrentToken().Index]
				c.Value = tokens.CurrentToken().Value
				a = ActionFiles()
			}
		}

		invoked := a.Invoke(c)
		for index, value := range invoked.action.rawValues {
			if !invoked.action.meta.Nospace.Matches(value.Value) || strings.Contains(value.Value, " ") { // TODO special characters
				switch tokens.CurrentToken().State {
				case shlex.QUOTING_ESCAPING_STATE:
					invoked.action.rawValues[index].Value = fmt.Sprintf(`"%v"`, strings.ReplaceAll(value.Value, `"`, `\"`))
				case shlex.QUOTING_STATE:
					invoked.action.rawValues[index].Value = fmt.Sprintf(`'%v'`, strings.ReplaceAll(value.Value, `'`, `'"'"'`))
				default:
					invoked.action.rawValues[index].Value = strings.Replace(value.Value, ` `, `\ `, -1)
				}
			}
			if !invoked.action.meta.Nospace.Matches(value.Value) {
				invoked.action.rawValues[index].Value += " "
			}
		}
		return invoked.Prefix(prefix).ToA().NoSpace()
	})
}

// Style sets the style.
//
//	ActionValues("yes").Style(style.Green)
//	ActionValues("no").Style(style.Red)
func (a Action) Style(s string) Action {
	return a.StyleF(func(_ string, _ style.Context) string {
		return s
	})
}

// Style sets the style using a function.
//
//	ActionValues("dir/", "test.txt").StyleF(style.ForPathExt)
//	ActionValues("true", "false").StyleF(style.ForKeyword)
func (a Action) StyleF(f func(s string, sc style.Context) string) Action {
	return ActionCallback(func(c Context) Action {
		invoked := a.Invoke(c)
		for index, v := range invoked.action.rawValues {
			invoked.action.rawValues[index].Style = f(v.Value, c)
		}
		return invoked.ToA()
	})
}

// Style sets the style using a reference.
//
//	ActionValues("value").StyleR(&style.Carapace.Value)
//	ActionValues("description").StyleR(&style.Carapace.Value)
func (a Action) StyleR(s *string) Action {
	return ActionCallback(func(c Context) Action {
		if s != nil {
			return a.Style(*s)
		}
		return a
	})
}

// Suffix adds a suffx to values (only the ones inserted, not the display values).
//
//	carapace.ActionValues("apple", "melon", "orange").Suffix("juice")
func (a Action) Suffix(suffix string) Action {
	return ActionCallback(func(c Context) Action {
		return a.Invoke(c).Suffix(suffix).ToA()
	})
}

// Suppress suppresses specific error messages using regular expressions.
func (a Action) Suppress(expr ...string) Action {
	return ActionCallback(func(c Context) Action {
		invoked := a.Invoke(c)
		if err := invoked.action.meta.Messages.Suppress(expr...); err != nil {
			return ActionMessage(err.Error())
		}
		return invoked.ToA()
	})
}

// Tag sets the tag.
//
//	ActionValues("192.168.1.1", "127.0.0.1").Tag("interfaces").
func (a Action) Tag(tag string) Action {
	return a.TagF(func(value string) string {
		return tag
	})
}

// Tag sets the tag using a function.
//
//	ActionValues("192.168.1.1", "127.0.0.1").TagF(func(value string) string {
//		return "interfaces"
//	})
func (a Action) TagF(f func(s string) string) Action {
	return ActionCallback(func(c Context) Action {
		invoked := a.Invoke(c)
		for index, v := range invoked.action.rawValues {
			invoked.action.rawValues[index].Tag = f(v.Value)
		}
		return invoked.ToA()
	})
}

// Timeout sets the maximum duration an Action may take to invoke.
//
//	carapace.ActionCallback(func(c carapace.Context) carapace.Action {
//		time.Sleep(2*time.Second)
//		return carapace.ActionValues("done")
//	}).Timeout(1*time.Second, carapace.ActionMessage("timeout exceeded"))
func (a Action) Timeout(d time.Duration, alternative Action) Action {
	return ActionCallback(func(c Context) Action {
		currentChannel := make(chan string, 1)

		var result InvokedAction
		go func() {
			result = a.Invoke(c)
			currentChannel <- ""
		}()

		select {
		case <-currentChannel:
		case <-time.After(d):
			return alternative
		}
		return result.ToA()
	})
}

// UniqueList wraps the Action in an ActionMultiParts with given divider.
func (a Action) UniqueList(divider string) Action {
	return ActionMultiParts(divider, func(c Context) Action {
		return a.FilterParts().NoSpace()
	})
}

// UniqueListF is like UniqueList but uses a function to transform values before filtering.
func (a Action) UniqueListF(divider string, f func(s string) string) Action {
	return ActionMultiParts(divider, func(c Context) Action {
		for i := range c.Parts {
			c.Parts[i] = f(c.Parts[i])
		}
		return a.Filter(c.Parts...).NoSpace()
	})
}

// Unless skips invokation if given condition succeeds.
func (a Action) Unless(condition func(c Context) bool) Action {
	return ActionCallback(func(c Context) Action {
		if condition(c) {
			return ActionValues()
		}
		return a
	})
}

// Usage sets the usage.
func (a Action) Usage(usage string, args ...interface{}) Action {
	return a.UsageF(func() string {
		return fmt.Sprintf(usage, args...)
	})
}

// Usage sets the usage using a function.
func (a Action) UsageF(f func() string) Action {
	return ActionCallback(func(c Context) Action {
		if usage := f(); usage != "" {
			a.meta.Usage = usage
		}
		return a
	})
}
