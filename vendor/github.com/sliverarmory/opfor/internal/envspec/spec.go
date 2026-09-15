// Package envspec defines the built-in Sleep and Aggressor environment
// declaration metadata shared by the lexer, parser, and runtime.
//
// The package deliberately has no dependencies on the syntax tree or runtime
// packages. Keeping it as a leaf prevents the lexer -> AST -> lexer import
// cycle while giving every language layer one authoritative keyword table.
package envspec

import "strings"

// Form identifies the parser and bridge ABI used by an environment
// declaration.
type Form uint8

const (
	Ordinary Form = iota
	Filter
	Predicate
)

// ClosureMode selects how a declaration body becomes a callable.
type ClosureMode uint8

const (
	// ClosureCurrent retains the declaration's active scope. Host-defined
	// environments use it so nested composition can inherit invocation state.
	ClosureCurrent ClosureMode = iota
	// ClosureRoot gives a declaration a fresh SleepClosure rooted at globals.
	ClosureRoot
	// ClosureInline executes a declaration body in its caller's active frame.
	ClosureInline
)

// Lifetime identifies a declaration's default registration lifetime.
type Lifetime uint8

const (
	Persistent Lifetime = iota
	Once
)

const (
	BindingSub      = "sub"
	BindingInline   = "inline"
	BindingEvent    = "on"
	BindingCommand  = "command"
	BindingAlias    = "alias"
	BindingSSHAlias = "ssh_alias"
	BindingHook     = "set"
	BindingPopup    = "popup"
	BindingMenu     = "menu"
	BindingItem     = "item"
	BindingKey      = "bind"
	BindingFilter   = "filter"
)

// Spec describes one built-in environment declaration keyword. Keyword is
// the canonical lowercase spelling. Binding is the runtime registry kind;
// several declaration aliases intentionally share one kind.
type Spec struct {
	Keyword              string
	LexicalKeyword       bool
	Form                 Form
	Binding              string
	Closure              ClosureMode
	Lifetime             Lifetime
	RecomposeDescendants bool
}

var builtins = [...]Spec{
	{Keyword: "sub", LexicalKeyword: true, Form: Ordinary, Binding: BindingSub, Closure: ClosureRoot, Lifetime: Persistent},
	{Keyword: "inline", LexicalKeyword: true, Form: Ordinary, Binding: BindingInline, Closure: ClosureInline, Lifetime: Persistent},
	{Keyword: "on", LexicalKeyword: true, Form: Ordinary, Binding: BindingEvent, Closure: ClosureCurrent, Lifetime: Persistent},
	{Keyword: "when", LexicalKeyword: true, Form: Ordinary, Binding: BindingEvent, Closure: ClosureCurrent, Lifetime: Once},
	{Keyword: "command", LexicalKeyword: true, Form: Ordinary, Binding: BindingCommand, Closure: ClosureCurrent, Lifetime: Persistent},
	{Keyword: "alias", LexicalKeyword: true, Form: Ordinary, Binding: BindingAlias, Closure: ClosureCurrent, Lifetime: Persistent},
	{Keyword: "ssh_alias", LexicalKeyword: true, Form: Ordinary, Binding: BindingSSHAlias, Closure: ClosureCurrent, Lifetime: Persistent},
	{Keyword: "set", LexicalKeyword: true, Form: Ordinary, Binding: BindingHook, Closure: ClosureCurrent, Lifetime: Persistent},
	{Keyword: "hook", LexicalKeyword: true, Form: Ordinary, Binding: BindingHook, Closure: ClosureCurrent, Lifetime: Persistent},
	{Keyword: "popup", LexicalKeyword: true, Form: Ordinary, Binding: BindingPopup, Closure: ClosureCurrent, Lifetime: Persistent, RecomposeDescendants: true},
	{Keyword: "menu", LexicalKeyword: true, Form: Ordinary, Binding: BindingMenu, Closure: ClosureCurrent, Lifetime: Persistent, RecomposeDescendants: true},
	{Keyword: "menubar", LexicalKeyword: true, Form: Ordinary, Binding: BindingMenu, Closure: ClosureCurrent, Lifetime: Persistent, RecomposeDescendants: true},
	{Keyword: "item", LexicalKeyword: true, Form: Ordinary, Binding: BindingItem, Closure: ClosureCurrent, Lifetime: Persistent},
	// Sleep recognizes bind as an environment declaration, but the reference
	// lexer leaves it as an identifier. Keep lexical and parser recognition
	// separate so centralizing the table does not change its token kind.
	{Keyword: "bind", LexicalKeyword: false, Form: Ordinary, Binding: BindingKey, Closure: ClosureCurrent, Lifetime: Persistent},
	{Keyword: "filter", LexicalKeyword: true, Form: Ordinary, Binding: BindingFilter, Closure: ClosureCurrent, Lifetime: Persistent},
}

var byKeyword = func() map[string]Spec {
	result := make(map[string]Spec, len(builtins))
	for _, spec := range builtins {
		result[spec.Keyword] = spec
	}
	return result
}()

var recomposingBindings = func() map[string]struct{} {
	result := make(map[string]struct{})
	for _, spec := range builtins {
		if spec.RecomposeDescendants {
			result[spec.Binding] = struct{}{}
		}
	}
	return result
}()

// Lookup returns the specification for an exact canonical keyword. Callers
// intentionally retain their existing normalization rules: lexical lookup is
// case-sensitive, while parser and runtime lookup normalize first.
func Lookup(keyword string) (Spec, bool) {
	spec, ok := byKeyword[keyword]
	return spec, ok
}

// LookupFold returns the specification matching keyword under Unicode simple
// case folding. The VM uses it only where the previous implementation used
// strings.EqualFold; lexical, parser, and registry lookup deliberately retain
// their narrower historical normalization rules through Lookup.
func LookupFold(keyword string) (Spec, bool) {
	for _, spec := range builtins {
		if strings.EqualFold(keyword, spec.Keyword) {
			return spec, true
		}
	}
	return Spec{}, false
}

// Builtins returns the specifications in stable declaration order. The
// returned slice is detached from the authoritative table.
func Builtins() []Spec {
	return append([]Spec(nil), builtins[:]...)
}

// RecomposesDescendants reports whether invoking a binding kind replaces its
// previous ephemeral composition tree. It is keyed by binding kind rather
// than declaration keyword so programmatically registered popup/menu bindings
// retain the same behavior as parsed declarations.
func RecomposesDescendants(binding string) bool {
	_, ok := recomposingBindings[binding]
	return ok
}
