package opfor

// runtimeExtensionProfile owns construction-time extension state which must be
// recreated in a portable ScriptLoader child. A profile clone returns an
// Option for the fresh child Runtime; it must not copy Runtime-bound native
// method values or mutable script/lifecycle state from the parent.
//
// Profiles are internal and immutable after New. Keeping the active set on the
// Runtime makes child and nested-child construction follow the exact profile
// set selected for their parent without exposing a new public API.
type runtimeExtensionProfile interface {
	cloneForScriptLoader(parent *Runtime) Option
}

func defaultRuntimeExtensionProfiles() []runtimeExtensionProfile {
	return []runtimeExtensionProfile{aggressorRuntimeExtensionProfile{}}
}

func cloneRuntimeExtensionProfiles(profiles []runtimeExtensionProfile) []runtimeExtensionProfile {
	return append([]runtimeExtensionProfile(nil), profiles...)
}

// withRuntimeExtensionProfiles is private constructor plumbing. It replaces,
// rather than appends to, New's default set so a child never acquires an
// extension profile which was not active on its parent.
func withRuntimeExtensionProfiles(profiles []runtimeExtensionProfile) Option {
	snapshot := cloneRuntimeExtensionProfiles(profiles)
	return func(config *runtimeConfig) error {
		config.extensionProfiles = cloneRuntimeExtensionProfiles(snapshot)
		return nil
	}
}

// scriptLoaderProfileOptions snapshots every active extension profile and
// preserves the active set for nested ScriptLoader children. Profile options
// run while New constructs a fresh Runtime, before child-bound native methods
// are installed.
func (r *Runtime) scriptLoaderProfileOptions() []Option {
	if r == nil {
		return nil
	}
	profiles := cloneRuntimeExtensionProfiles(r.extensionProfiles)
	options := make([]Option, 0, len(profiles)+1)
	options = append(options, withRuntimeExtensionProfiles(profiles))
	for _, profile := range profiles {
		if isNilInterface(profile) {
			continue
		}
		if option := profile.cloneForScriptLoader(r); option != nil {
			options = append(options, option)
		}
	}
	return options
}
