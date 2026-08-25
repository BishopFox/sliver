package bofloader

import (
	"fmt"
	"sort"
	"strings"
)

// Import describes one external symbol referenced by an object.
type Import struct {
	Name         string
	Weak         bool
	Builtin      bool
	RequiresHost bool
}

// LoadOptions controls entry selection and import resolution. Its zero value
// preserves Load's behavior.
type LoadOptions struct {
	EntryPoint      string
	ValidateImports func([]Import) error
	ResolveSymbol   func(Import) (address uintptr, handled bool, err error)
}

func selectedEntryNames(options LoadOptions) ([]string, error) {
	if options.EntryPoint == "" {
		return []string{"go", "_go", "coffee", "_coffee"}, nil
	}
	if len(options.EntryPoint) > maxObjectNameSize {
		return nil, fmt.Errorf("bofloader: entry symbol name exceeds %d bytes", maxObjectNameSize)
	}
	if strings.IndexByte(options.EntryPoint, 0) >= 0 {
		return nil, fmt.Errorf("bofloader: entry symbol name contains NUL")
	}
	return []string{options.EntryPoint}, nil
}

func objectImports(object *objectFile, referenced []uint32) []Import {
	byName := make(map[string]Import)
	for _, index := range referenced {
		symbol, ok := object.symbols[index]
		if !ok || symbol.section != sectionUndefined ||
			(object.format == "elf" && (symbol.name == "_GLOBAL_OFFSET_TABLE_" || object.arch == "ppc64le" && symbol.name == ".TOC.")) {
			continue
		}
		imported := classifyImport(symbol)
		if existing, ok := byName[imported.Name]; ok {
			// One strong reference makes the imported name strong.
			imported.Weak = existing.Weak && imported.Weak
		}
		byName[imported.Name] = imported
	}
	imports := make([]Import, 0, len(byName))
	for _, imported := range byName {
		imports = append(imports, imported)
	}
	sort.Slice(imports, func(left, right int) bool {
		return imports[left].Name < imports[right].Name
	})
	return imports
}

func classifyImport(symbol objectSymbol) Import {
	name := normalizeImportedSymbol(symbol.name)
	_, builtin := builtinBeaconCallbackNames[name]
	return Import{
		Name:         symbol.name,
		Weak:         symbol.weak,
		Builtin:      builtin,
		RequiresHost: !builtin && strings.HasPrefix(name, "Beacon"),
	}
}

func resolveImportedSymbol(symbol objectSymbol, options LoadOptions) (uintptr, error) {
	if address, ok, err := resolveBeaconCallback(symbol.name); err != nil {
		if symbol.weak {
			return 0, nil
		}
		return 0, err
	} else if ok {
		return address, nil
	}

	imported := classifyImport(symbol)
	if options.ResolveSymbol != nil {
		address, handled, err := options.ResolveSymbol(imported)
		if err != nil {
			return 0, fmt.Errorf("host resolver: %w", err)
		}
		if handled {
			if address == 0 {
				return 0, fmt.Errorf("host resolver returned a zero address for %q", imported.Name)
			}
			return address, nil
		}
	}
	if imported.RequiresHost {
		return 0, fmt.Errorf("Beacon callback %q requires a host-provided resolver", imported.Name)
	}
	address, err := resolveSymbol(symbol.name)
	if err != nil && symbol.weak {
		return 0, nil
	}
	return address, err
}
