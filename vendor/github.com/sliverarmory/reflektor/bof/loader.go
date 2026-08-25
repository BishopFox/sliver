package bof

import (
	"fmt"

	"github.com/sliverarmory/reflektor/internal/bofloader"
)

func loadObject(data []byte, options LoadOptions) (*bofloader.Loader, error) {
	loaderOptions := bofloader.LoadOptions{EntryPoint: options.EntryPoint}
	if options.ValidateImports != nil {
		loaderOptions.ValidateImports = func(imports []bofloader.Import) error {
			converted := make([]Import, len(imports))
			for index, imported := range imports {
				converted[index] = Import{
					Name:         imported.Name,
					Weak:         imported.Weak,
					Builtin:      imported.Builtin,
					RequiresHost: imported.RequiresHost,
				}
			}
			return options.ValidateImports(converted)
		}
	}
	if options.ResolveSymbol != nil {
		loaderOptions.ResolveSymbol = func(imported bofloader.Import) (uintptr, bool, error) {
			return options.ResolveSymbol(Import{
				Name:         imported.Name,
				Weak:         imported.Weak,
				Builtin:      imported.Builtin,
				RequiresHost: imported.RequiresHost,
			})
		}
	}
	loader, err := bofloader.LoadWithOptions(data, loaderOptions)
	if err != nil {
		return nil, fmt.Errorf("reflektor: load BOF: %w", err)
	}
	return loader, nil
}
