package opfor

// sleepBridgeArguments converts script syntax into the argument stack shape
// seen by stock Sleep Function bridges. Source key/value pairs and explicit
// pass-by-name references are each one positional sleep.bridges.KeyValuePair
// object. Their private origins remain distinct for OPFOR's importer-facing
// reference contract.
//
// Importer boundaries receive the original Arguments instead. In particular,
// their public Name and Reference fields retain the existing OPFOR contract.
func sleepBridgeArguments(arguments []Argument) []Argument {
	var converted []Argument
	for index, argument := range arguments {
		if argument.bridgeMaterialized {
			continue
		}
		switch argument.syntax {
		case argumentSyntaxPair, argumentSyntaxReference:
			if converted == nil {
				converted = append([]Argument(nil), arguments...)
			}
			name := argument.syntaxName
			converted[index] = Argument{
				Value: ObjectValue(sleepKeyValue{
					key:   String(name),
					value: argument.Resolve(),
				}),
				syntax:             argument.syntax,
				syntaxName:         name,
				syntaxCell:         argument.Reference,
				bridgeMaterialized: true,
			}
		}
	}
	if converted != nil {
		return converted
	}
	return arguments
}

// sleepNamedArgument recognizes every shape from which a Sleep bridge may
// deliberately extract a named parameter: source pair syntax, explicit
// pass-by-name syntax, importer-constructed named Arguments, and an already
// materialized KeyValuePair object.
func sleepNamedArgument(argument Argument) (string, Value, bool) {
	if argument.syntax == argumentSyntaxPair || argument.syntax == argumentSyntaxReference {
		name := argument.syntaxName
		value := argument.Resolve()
		if argument.bridgeMaterialized {
			if key, pairValue, ok := sleepKeyValueParts(value); ok {
				return key, pairValue, true
			}
		}
		return name, value, true
	}
	if argument.Name != "" {
		return argument.Name, argument.Resolve(), true
	}
	return sleepKeyValueParts(argument.Resolve())
}

func sleepKeyValueParts(value Value) (string, Value, bool) {
	object, ok := value.Object()
	if !ok {
		return "", Null(), false
	}
	switch pair := object.(type) {
	case sleepKeyValue:
		return pair.key.String(), pair.value, true
	case *sleepKeyValue:
		if pair != nil {
			return pair.key.String(), pair.value, true
		}
	}
	return "", Null(), false
}

// extractSleepNamedArguments mirrors BridgeUtilities.extractNamedParameters:
// recognized pairs are removed from the positional list, and the leftmost
// source duplicate wins under OPFOR's source-ordered Argument representation.
func extractSleepNamedArguments(arguments []Argument) ([]Argument, map[string]Argument) {
	positional := make([]Argument, 0, len(arguments))
	named := make(map[string]Argument)
	for _, argument := range arguments {
		name, value, ok := sleepNamedArgument(argument)
		if !ok {
			positional = append(positional, argument)
			continue
		}
		if _, duplicate := named[name]; duplicate {
			continue
		}
		normalized := Argument{Name: name, Value: value}
		normalized.Reference = sleepArgumentReference(argument)
		named[name] = normalized
	}
	return positional, named
}

// sleepExtractedArguments preserves source order while turning the pair
// objects consumed by initLocalScope-style bridges back into OPFOR's internal
// named Argument form. The private source Cell restores pass-by-name identity
// without exposing a Reference on the raw KeyValuePair bridge argument.
func sleepExtractedArguments(arguments []Argument) []Argument {
	var extracted []Argument
	for index, argument := range arguments {
		name, value, ok := sleepNamedArgument(argument)
		if !ok {
			continue
		}
		if extracted == nil {
			extracted = append([]Argument(nil), arguments...)
		}
		extracted[index] = Argument{
			Name: name, Value: value, Reference: sleepArgumentReference(argument),
		}
	}
	if extracted != nil {
		return extracted
	}
	return arguments
}

func sleepArgumentReference(argument Argument) *Cell {
	if argument.Reference != nil {
		return argument.Reference
	}
	return argument.syntaxCell
}
