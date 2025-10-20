package kong

// BeforeResolve is a documentation-only interface describing hooks that run before resolvers are applied.
type BeforeResolve interface {
	// This is not the correct signature - see README for details.
	BeforeResolve(args ...interface{}) error
}

// BeforeApply is a documentation-only interface describing hooks that run before values are set.
type BeforeApply interface {
	// This is not the correct signature - see README for details.
	BeforeApply(args ...interface{}) error
}

// AfterApply is a documentation-only interface describing hooks that run after values are set.
type AfterApply interface {
	// This is not the correct signature - see README for details.
	AfterApply(args ...interface{}) error
}
