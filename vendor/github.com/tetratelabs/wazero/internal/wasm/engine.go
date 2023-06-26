package wasm

import (
	"context"

	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"
)

// Engine is a Store-scoped mechanism to compile functions declared or imported by a module.
// This is a top-level type implemented by an interpreter or compiler.
type Engine interface {
	// Close closes this engine, and releases all the compiled cache.
	Close() (err error)

	// CompileModule implements the same method as documented on wasm.Engine.
	CompileModule(ctx context.Context, module *Module, listeners []experimental.FunctionListener, ensureTermination bool) error

	// CompiledModuleCount is exported for testing, to track the size of the compilation cache.
	CompiledModuleCount() uint32

	// DeleteCompiledModule releases compilation caches for the given module (source).
	// Note: it is safe to call this function for a module from which module instances are instantiated even when these
	// module instances have outstanding calls.
	DeleteCompiledModule(module *Module)

	// NewModuleEngine compiles down the function instances in a module, and returns ModuleEngine for the module.
	//
	// * module is the source module from which moduleFunctions are instantiated. This is used for caching.
	// * instance is the *ModuleInstance which is created from `module`.
	//
	// Note: Input parameters must be pre-validated with wasm.Module Validate, to ensure no fields are invalid
	// due to reasons such as out-of-bounds.
	NewModuleEngine(module *Module, instance *ModuleInstance) (ModuleEngine, error)
}

// ModuleEngine implements function calls for a given module.
type ModuleEngine interface {
	// NewFunction returns an api.Function for the given function pointed by the given Index.
	NewFunction(index Index) api.Function

	// ResolveImportedFunction is used to add imported functions needed to make this ModuleEngine fully functional.
	// 	- `index` is the function Index of this imported function.
	// 	- `indexInImportedModule` is the function Index of the imported function in the imported module.
	//	- `importedModuleEngine` is the ModuleEngine for the imported ModuleInstance.
	ResolveImportedFunction(index, indexInImportedModule Index, importedModuleEngine ModuleEngine)

	// LookupFunction returns the api.Function created from the function in the function table at the given offset.
	LookupFunction(t *TableInstance, typeId FunctionTypeID, tableOffset Index) (api.Function, error)

	// FunctionInstanceReference returns Reference for the given Index for a FunctionInstance. The returned values are used by
	// the initialization via ElementSegment.
	FunctionInstanceReference(funcIndex Index) Reference
}
