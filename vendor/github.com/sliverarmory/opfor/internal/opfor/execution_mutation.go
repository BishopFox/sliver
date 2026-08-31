package opfor

import "context"

// setAtExecution commits one evaluator-visible scalar mutation. Script unload
// is linearized with the write: either the write holds a read admission before
// unload closes the script, or it observes the closed admission and returns
// without changing the Cell.
//
// The optimistic Cell-first path is important for cancellation. A mutation
// may already be blocked on Cell.mu when Unload begins; it must not hold
// Script.mu and thereby prevent Unload from publishing cancellation. TryRLock
// never waits. When a Script writer already owns or is waiting for Script.mu,
// the fallback drops Cell.mu and reacquires locks in the repository's normal
// Script-then-Cell order, avoiding inversion with Script.Set.
func (cell *Cell) setAtExecution(ctx context.Context, script *Script, value Value, span Span) error {
	if cell == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if script == nil {
		cell.mu.Lock()
		err := scriptExecutionError(ctx)
		var watcher func(Value, Span)
		if err == nil {
			cell.value = value
			watcher = cell.watcher
		}
		cell.mu.Unlock()
		if watcher != nil {
			watcher(value, span)
		}
		return err
	}

	cell.mu.Lock()
	if script.mu.TryRLock() {
		err, watcher := commitExecutionCellLocked(ctx, script, cell, value)
		script.mu.RUnlock()
		cell.mu.Unlock()
		if watcher != nil {
			watcher(value, span)
		}
		return err
	}
	cell.mu.Unlock()

	// A Script writer has priority. Let it publish its state transition before
	// retrying in the established Script.mu -> Cell.mu order.
	script.mu.RLock()
	cell.mu.Lock()
	err, watcher := commitExecutionCellLocked(ctx, script, cell, value)
	cell.mu.Unlock()
	script.mu.RUnlock()
	if watcher != nil {
		watcher(value, span)
	}
	return err
}

// commitExecutionCellLocked requires both script.mu (read or write) and
// cell.mu. Holding script.mu prevents requestUnload from changing active until
// the Cell write has reached its linearization point.
func commitExecutionCellLocked(
	ctx context.Context,
	script *Script,
	cell *Cell,
	value Value,
) (error, func(Value, Span)) {
	if err := scriptExecutionError(ctx); err != nil {
		return err, nil
	}
	if !script.active {
		return ErrScriptUnloaded, nil
	}
	cell.value = value
	return nil, cell.watcher
}

func (f *fiber) setCellAtExecution(ctx context.Context, cell *Cell, value Value, span Span) error {
	if f == nil || f.closure == nil {
		return cell.setAtExecution(ctx, nil, value, span)
	}
	return cell.setAtExecution(ctx, f.closure.script, value, span)
}
