package opfor

import "sort"

// AggressorContractResult describes the result visible to script code after a
// successful call through a typed Aggressor provider. Generic Host fallback is
// deliberately outside this contract because importers may preserve legacy
// result conventions there.
type AggressorContractResult string

const (
	// AggressorContractResultValue transfers the provider's Value to the script.
	AggressorContractResultValue AggressorContractResult = "value"
	// AggressorContractResultNull discards the provider's Value and returns null.
	AggressorContractResultNull AggressorContractResult = "null"
	// AggressorContractResultPredicate coerces the provider's result to a Sleep
	// predicate value.
	AggressorContractResultPredicate AggressorContractResult = "predicate"
)

// AggressorContractProviderErrors describes what a typed provider error means
// at the native wrapper boundary.
type AggressorContractProviderErrors string

const (
	// AggressorContractProviderErrorsAuthoritative means OPFOR returns null and
	// the provider error without retrying through Host, because an external
	// effect may already have occurred.
	AggressorContractProviderErrorsAuthoritative AggressorContractProviderErrors = "authoritative"
)

// AggressorCallbackContract describes one callback-shaped source argument.
// Position is one-based so it matches Aggressor documentation and diagnostics.
// Arguments is meaningful only when ArgumentsKnown is true.
type AggressorCallbackContract struct {
	Position       int  `json:"position"`
	Required       bool `json:"required"`
	Nullable       bool `json:"nullable"`
	Retained       bool `json:"retained"`
	ArgumentsKnown bool `json:"arguments_known"`
	Arguments      int  `json:"arguments"`
}

// AggressorArgumentConstraint describes one source-audited positional-value
// constraint enforced before a typed Aggressor provider is called. Position is
// one-based. Kind identifies the comparison rule; the current inventory uses
// "enum" for exact Sleep string-coercion matches. In Values, "$null" denotes
// the null Value and an empty string denotes a non-null blank string.
//
// Constraints apply only to the typed-provider route. The generic Host fallback
// deliberately receives the original Invocation without resolving or
// validating its arguments so legacy importers retain authority over it.
type AggressorArgumentConstraint struct {
	Position int      `json:"position"`
	Kind     string   `json:"kind"`
	Values   []string `json:"values"`
}

// AggressorFunctionContract is the runtime-enforced portion of a native
// Aggressor function boundary. MinimumArguments and MaximumArguments are
// inclusive. Provider identifies the public typed provider interface; a valid
// call falls back to Host when that provider is not configured.
//
// The inventory intentionally contains only function-specific contracts
// enforced by OPFOR. Its absence does not mean a name is unavailable: generic
// Host functions and portable Sleep functions are cataloged separately by the
// aggressor package.
type AggressorFunctionContract struct {
	Name                string                          `json:"name"`
	MinimumArguments    int                             `json:"minimum_arguments"`
	MaximumArguments    int                             `json:"maximum_arguments"`
	Callbacks           []AggressorCallbackContract     `json:"callbacks,omitempty"`
	ArgumentConstraints []AggressorArgumentConstraint   `json:"argument_constraints,omitempty"`
	TypedProvider       string                          `json:"typed_provider"`
	TypedResult         AggressorContractResult         `json:"typed_result"`
	ProviderErrors      AggressorContractProviderErrors `json:"provider_errors"`
	HostFallback        bool                            `json:"host_fallback"`
	Deprecated          bool                            `json:"deprecated,omitempty"`
}

// DefaultAggressorFunctionContracts returns a sorted, detached inventory of
// native Aggressor contracts enforced by New's typed-provider wrappers.
func DefaultAggressorFunctionContracts() []AggressorFunctionContract {
	contracts := make(map[string]AggressorFunctionContract)
	add := func(contract AggressorFunctionContract) {
		if contract.Name == "" || contract.MinimumArguments < 0 ||
			contract.MaximumArguments < contract.MinimumArguments ||
			contract.TypedProvider == "" || contract.TypedResult == "" {
			panic("opfor: invalid native Aggressor function contract")
		}
		if _, exists := contracts[contract.Name]; exists {
			panic("opfor: duplicate native Aggressor function contract: " + contract.Name)
		}
		for _, constraint := range contract.ArgumentConstraints {
			if constraint.Position < 1 || constraint.Position > contract.MaximumArguments ||
				constraint.Kind == "" || len(constraint.Values) == 0 {
				panic("opfor: invalid native Aggressor argument constraint: " + contract.Name)
			}
		}
		contract.ProviderErrors = AggressorContractProviderErrorsAuthoritative
		contract.HostFallback = true
		contracts[contract.Name] = contract
	}
	addSpecs := func(
		provider string,
		arities map[string][2]int,
		result func(string) AggressorContractResult,
	) {
		for name, arity := range arities {
			add(AggressorFunctionContract{
				Name: name, MinimumArguments: arity[0], MaximumArguments: arity[1],
				TypedProvider: provider, TypedResult: result(name),
			})
		}
	}
	value := func(string) AggressorContractResult { return AggressorContractResultValue }

	for name, spec := range aggressorArtifactSpecs {
		contract := AggressorFunctionContract{
			Name: name, MinimumArguments: spec.minimum, MaximumArguments: spec.maximum,
			TypedProvider: "AggressorArtifactProvider", TypedResult: AggressorContractResultValue,
		}
		if spec.kind == AggressorArtifactStageless {
			contract.TypedResult = AggressorContractResultNull
			contract.Deprecated = true
			contract.Callbacks = []AggressorCallbackContract{{
				Position: 5, Required: true, Retained: true, ArgumentsKnown: true, Arguments: 1,
			}}
		}
		add(contract)
	}
	add(AggressorFunctionContract{
		Name:             "bof_extract",
		MinimumArguments: aggressorBOFExtractionMinimumArguments,
		MaximumArguments: aggressorBOFExtractionMaximumArguments,
		TypedProvider:    "AggressorBOFExtractor",
		TypedResult:      AggressorContractResultValue,
	})
	for name, spec := range aggressorBeaconActionSpecs {
		contract := AggressorFunctionContract{
			Name: name, MinimumArguments: spec.minimum, MaximumArguments: spec.maximum,
			TypedProvider: "AggressorBeaconActionProvider", TypedResult: AggressorContractResultNull,
		}
		if spec.hasCallback {
			contract.Callbacks = []AggressorCallbackContract{{
				Position: spec.callbackIndex + 1, Required: spec.callbackRequired,
				Nullable: !spec.callbackRequired, Retained: true,
			}}
		}
		add(contract)
	}
	for name, spec := range aggressorBeaconExecutionSpecs {
		result := AggressorContractResultNull
		if spec.returnsValue {
			result = AggressorContractResultValue
		}
		add(AggressorFunctionContract{
			Name: name, MinimumArguments: spec.minimum, MaximumArguments: spec.maximum,
			TypedProvider: "AggressorBeaconExecutionProvider", TypedResult: result,
		})
	}
	for name, spec := range aggressorClientServiceSpecs {
		result := AggressorContractResultNull
		if spec.returnsValue {
			result = AggressorContractResultValue
		}
		contract := AggressorFunctionContract{
			Name: name, MinimumArguments: spec.minimum, MaximumArguments: spec.maximum,
			TypedProvider: "AggressorClientServiceProvider", TypedResult: result,
		}
		if spec.callback >= 0 {
			contract.Callbacks = []AggressorCallbackContract{{
				Position: spec.callback + 1, Nullable: true, Retained: true,
			}}
		}
		add(contract)
	}
	for name, spec := range aggressorClientUISpecs {
		result := AggressorContractResultValue
		if spec.discardResult {
			result = AggressorContractResultNull
		}
		contract := AggressorFunctionContract{
			Name: name, MinimumArguments: spec.minimum, MaximumArguments: spec.maximum,
			TypedProvider: "AggressorClientUIProvider", TypedResult: result,
		}
		if spec.callback >= 0 {
			contract.Callbacks = []AggressorCallbackContract{{
				Position: spec.callback + 1, Required: true, Retained: true,
			}}
		}
		add(contract)
	}
	addSpecs("AggressorCodeTransformProvider", exactContractArities(aggressorCodeTransformSpecs, func(spec aggressorCodeTransformSpec) int { return spec.arguments }), value)

	for name, spec := range aggressorDataModelQuerySpecs {
		arity := 0
		if spec.keyed {
			arity = 1
		}
		add(AggressorFunctionContract{
			Name: name, MinimumArguments: arity, MaximumArguments: arity,
			TypedProvider: "AggressorDataModelQueryProvider", TypedResult: AggressorContractResultValue,
		})
	}
	for name, spec := range aggressorDataStoreSpecs {
		result := AggressorContractResultValue
		if spec.discardResult {
			result = AggressorContractResultNull
		}
		add(AggressorFunctionContract{
			Name: name, MinimumArguments: spec.minimum, MaximumArguments: spec.maximum,
			TypedProvider: "AggressorDataStoreProvider", TypedResult: result,
		})
	}
	for name, spec := range aggressorListenerSpecs {
		result := AggressorContractResultNull
		if spec.returns {
			result = AggressorContractResultValue
		}
		add(AggressorFunctionContract{
			Name: name, MinimumArguments: spec.minimum, MaximumArguments: spec.maximum,
			TypedProvider: "AggressorListenerProvider", TypedResult: result,
		})
	}
	for name, spec := range aggressorPayloadSpecs {
		result := AggressorContractResultValue
		if spec.predicate {
			result = AggressorContractResultPredicate
		}
		add(AggressorFunctionContract{
			Name: name, MinimumArguments: spec.minimum, MaximumArguments: spec.maximum,
			TypedProvider: "AggressorPayloadProvider", TypedResult: result,
			ArgumentConstraints: cloneAggressorArgumentConstraints(spec.argumentConstraints),
			Deprecated:          name == "powershell",
		})
	}
	for name, spec := range aggressorPayloadStoreSpecs {
		result := AggressorContractResultNull
		if spec.returns {
			result = AggressorContractResultValue
		}
		add(AggressorFunctionContract{
			Name: name, MinimumArguments: spec.minimum, MaximumArguments: spec.maximum,
			TypedProvider: "AggressorPayloadStoreProvider", TypedResult: result,
		})
	}
	for name, spec := range aggressorPEProviderSpecs {
		add(AggressorFunctionContract{
			Name: name, MinimumArguments: spec.minimum, MaximumArguments: spec.maximum,
			TypedProvider: "AggressorPEProvider", TypedResult: AggressorContractResultValue,
		})
	}
	for name, spec := range aggressorPreferenceSpecs {
		result := AggressorContractResultNull
		if spec.returnsValue {
			result = AggressorContractResultValue
		}
		add(AggressorFunctionContract{
			Name: name, MinimumArguments: spec.arguments, MaximumArguments: spec.arguments,
			TypedProvider: "AggressorPreferenceProvider", TypedResult: result,
		})
	}
	for name, spec := range aggressorProcessInjectionSpecs {
		result := AggressorContractResultNull
		if spec.returnsValue {
			result = AggressorContractResultValue
		}
		add(AggressorFunctionContract{
			Name: name, MinimumArguments: spec.arguments, MaximumArguments: spec.arguments,
			TypedProvider: "AggressorProcessInjectionProvider", TypedResult: result,
		})
	}
	addSpecs("AggressorProfileProvider", exactContractArities(aggressorProfileSpecs, func(spec aggressorProfileSpec) int { return spec.arguments }), value)
	for name, spec := range aggressorSessionQuerySpecs {
		result := AggressorContractResultValue
		if spec.predicate {
			result = AggressorContractResultPredicate
		}
		add(AggressorFunctionContract{
			Name: name, MinimumArguments: spec.arity, MaximumArguments: spec.arity,
			TypedProvider: "AggressorSessionQueryProvider", TypedResult: result,
		})
	}
	for name, spec := range aggressorSiteSpecs {
		result := AggressorContractResultNull
		if spec.returnsValue {
			result = AggressorContractResultValue
		}
		add(AggressorFunctionContract{
			Name: name, MinimumArguments: spec.minimum, MaximumArguments: spec.maximum,
			TypedProvider: "AggressorSiteProvider", TypedResult: result,
		})
	}
	addDialogContracts(add)
	for name, spec := range aggressorVPNSpecs {
		result := AggressorContractResultNull
		if spec.returnsValue {
			result = AggressorContractResultValue
		}
		add(AggressorFunctionContract{
			Name: name, MinimumArguments: spec.minimum, MaximumArguments: spec.maximum,
			TypedProvider: "AggressorVPNProvider", TypedResult: result,
		})
	}

	names := make([]string, 0, len(contracts))
	for name := range contracts {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]AggressorFunctionContract, 0, len(names))
	for _, name := range names {
		contract := contracts[name]
		contract.Callbacks = append([]AggressorCallbackContract(nil), contract.Callbacks...)
		contract.ArgumentConstraints = cloneAggressorArgumentConstraints(contract.ArgumentConstraints)
		result = append(result, contract)
	}
	return result
}

func cloneAggressorArgumentConstraints(source []AggressorArgumentConstraint) []AggressorArgumentConstraint {
	if source == nil {
		return nil
	}
	result := make([]AggressorArgumentConstraint, len(source))
	for index, constraint := range source {
		result[index] = constraint
		result[index].Values = append([]string(nil), constraint.Values...)
	}
	return result
}

func exactContractArities[T any](specs map[string]T, arguments func(T) int) map[string][2]int {
	result := make(map[string][2]int, len(specs))
	for name, spec := range specs {
		arity := arguments(spec)
		result[name] = [2]int{arity, arity}
	}
	return result
}

func addDialogContracts(add func(AggressorFunctionContract)) {
	add(AggressorFunctionContract{Name: "dialog", MinimumArguments: 3, MaximumArguments: 3, TypedProvider: "AggressorDialogProvider", TypedResult: AggressorContractResultValue, Callbacks: []AggressorCallbackContract{{Position: 3, Required: true, Retained: true}}})
	add(AggressorFunctionContract{Name: "dialog_description", MinimumArguments: 2, MaximumArguments: 3, TypedProvider: "AggressorDialogProvider", TypedResult: AggressorContractResultNull})
	add(AggressorFunctionContract{Name: "dialog_show", MinimumArguments: 1, MaximumArguments: 1, TypedProvider: "AggressorDialogProvider", TypedResult: AggressorContractResultNull})
	add(AggressorFunctionContract{Name: "dbutton_action", MinimumArguments: 2, MaximumArguments: 2, TypedProvider: "AggressorDialogProvider", TypedResult: AggressorContractResultNull})
	add(AggressorFunctionContract{Name: "dbutton_help", MinimumArguments: 2, MaximumArguments: 2, TypedProvider: "AggressorDialogProvider", TypedResult: AggressorContractResultNull})
	for name, spec := range aggressorDialogRowSpecs {
		add(AggressorFunctionContract{
			Name: name, MinimumArguments: spec.minimum, MaximumArguments: spec.maximum,
			TypedProvider: "AggressorDialogProvider", TypedResult: AggressorContractResultNull,
			Deprecated: spec.deprecated,
		})
	}
	for name, spec := range aggressorPromptSpecs {
		add(AggressorFunctionContract{
			Name: name, MinimumArguments: spec.arguments, MaximumArguments: spec.arguments,
			TypedProvider: "AggressorPromptProvider", TypedResult: AggressorContractResultNull,
			Callbacks: []AggressorCallbackContract{{
				Position: spec.callbackIndex + 1, Required: true, Retained: true,
				ArgumentsKnown: true, Arguments: spec.callbackArity,
			}},
		})
	}
}
