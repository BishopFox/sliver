package opfor

import "context"

// reserveCollectionEntries charges persistent collection entries created by
// runtime code. Public NewArray/NewHash constructors deliberately remain
// unmetered: values assembled by an importer are trusted at ingress. If script
// code subsequently grows one of those values, the execution-aware mutation
// paths below charge the acting Runtime family before committing the growth.
func reserveCollectionEntries(runtime *Runtime, amount int) error {
	if amount <= 0 {
		return nil
	}
	return reserveCollectionEntryAmount(runtime, uint64(amount))
}

func reserveCollectionEntryAmount(runtime *Runtime, amount uint64) error {
	if amount == 0 {
		return nil
	}
	return runtime.reserveResource(resourceCollectionEntries, amount)
}

func reserveCollectionEntriesAtExecution(ctx context.Context, script *Script, amount int) error {
	if amount <= 0 {
		return nil
	}
	if account, _ := ctx.Value(runtimeResourceAccountKey{}).(*runtimeResourceAccount); account != nil {
		return account.reserve(resourceCollectionEntries, uint64(amount))
	}
	if script == nil {
		return nil
	}
	return reserveCollectionEntries(script.runtime, amount)
}

func newRuntimeArray(runtime *Runtime, values ...Value) (*Array, error) {
	if err := reserveCollectionEntries(runtime, len(values)); err != nil {
		return nil, err
	}
	return NewArray(values...), nil
}

func newRuntimeReadOnlyArray(runtime *Runtime, values ...Value) (*Array, error) {
	if err := reserveCollectionEntries(runtime, len(values)); err != nil {
		return nil, err
	}
	return NewReadOnlyArray(values...), nil
}

func newRuntimeReadOnlyHash(runtime *Runtime, values map[string]Value) (*Hash, error) {
	if err := reserveCollectionEntries(runtime, len(values)); err != nil {
		return nil, err
	}
	return NewReadOnlyHash(values), nil
}

func newRuntimeArrayFromCells(runtime *Runtime, cells []*Cell) (*Array, error) {
	if err := reserveCollectionEntries(runtime, len(cells)); err != nil {
		return nil, err
	}
	return newArrayFromCells(cells), nil
}
