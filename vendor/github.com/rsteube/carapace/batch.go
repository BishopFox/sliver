package carapace

import "sync"

type (
	batch        []Action
	invokedBatch []InvokedAction
)

// Batch creates a batch of Actions that can be invoked in parallel.
func Batch(actions ...Action) batch {
	return batch(actions)
}

// Invoke invokes contained Actions of the batch using goroutines.
func (b batch) Invoke(c Context) invokedBatch {
	invokedActions := make([]InvokedAction, len(b))
	functions := make([]func(), len(b))

	for index, action := range b {
		localIndex := index
		localAction := action
		functions[index] = func() {
			invokedActions[localIndex] = localAction.Invoke(c)
		}
	}
	parallelize(functions...)
	return invokedActions
}

// ToA converts the batch to an implicitly merged action which is a shortcut for:
//
//	ActionCallback(func(c Context) Action {
//		return batch.Invoke(c).Merge().ToA()
//	})
func (b batch) ToA() Action {
	return ActionCallback(func(c Context) Action {
		return b.Invoke(c).Merge().ToA()
	})
}

// Merge merges Actions of a batch.
func (b invokedBatch) Merge() InvokedAction {
	switch len(b) {
	case 0:
		return ActionValues().Invoke(Context{})
	case 1:
		return b[0]
	default:
		return b[0].Merge(b[1:]...)
	}
}

// Parallelize parallelizes the function calls (https://stackoverflow.com/a/44402936)
func parallelize(functions ...func()) {
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(functions))

	defer waitGroup.Wait()

	for _, function := range functions {
		go func(copy func()) {
			defer waitGroup.Done()
			copy()
		}(function)
	}
}
