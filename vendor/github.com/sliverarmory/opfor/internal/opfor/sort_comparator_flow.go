package opfor

import (
	"context"
	"errors"
)

// sleepSortComparatorFlow models the shared ScriptEnvironment return state
// observed by Java Comparator bridges. SleepClosure.callClosure keeps
// FLOW_CONTROL_THROW pending, so Java TimSort finishes without invoking the
// closure again. BasicStrings.CompareFunction observes the thrown scalar's
// integer coercion; reflection-proxy Comparator calls observe the empty scalar
// instead. comparison records the value selected by the active bridge.
//
// Only an isolated Sleep thrown value enters this compatibility path. A
// cancellation, resource failure, importer error, or other independent error
// branch remains authoritative and aborts the copy-on-write sort.
type sleepSortComparatorFlow struct {
	thrown     *scriptThrow
	comparison int
}

func (flow *sleepSortComparatorFlow) compare(
	ctx context.Context,
	invoke func() (Value, error),
) (int, error) {
	return flow.compareThrownAs(ctx, invoke, func(thrown *scriptThrow) int {
		return int(sleepInt32(thrown.value))
	})
}

func (flow *sleepSortComparatorFlow) compareProxy(
	ctx context.Context,
	invoke func() (Value, error),
) (int, error) {
	return flow.compareThrownAs(ctx, invoke, func(*scriptThrow) int { return 0 })
}

func (flow *sleepSortComparatorFlow) compareThrownAs(
	ctx context.Context,
	invoke func() (Value, error),
	coerce func(*scriptThrow) int,
) (int, error) {
	if err := executionContextError(ctx); err != nil {
		return 0, err
	}
	if flow != nil && flow.thrown != nil {
		return flow.comparison, nil
	}

	value, err := invoke()
	if err == nil {
		return int(sleepInt32(value)), nil
	}
	thrown, isolated := isolatedSleepSortThrow(err)
	if !isolated {
		return 0, err
	}
	flow.thrown = thrown
	flow.comparison = coerce(thrown)
	return flow.comparison, nil
}

func (flow *sleepSortComparatorFlow) pendingThrow() error {
	if flow == nil || flow.thrown == nil {
		return nil
	}
	return flow.thrown
}

func isolatedSleepSortThrow(err error) (*scriptThrow, bool) {
	var thrown *scriptThrow
	if !errors.As(err, &thrown) || thrown == nil || !sleepSortErrorOnlyThrows(err) {
		return nil, false
	}
	return thrown, true
}

func sleepSortErrorOnlyThrows(err error) bool {
	if err == nil {
		return true
	}
	switch err.(type) {
	case *scriptThrow:
		return true
	case *uncaughtScriptWarning, *nativeBoundaryError, *portableObjectCallbackError:
		return false
	}
	switch wrapped := err.(type) {
	case interface{ Unwrap() []error }:
		children := wrapped.Unwrap()
		if len(children) == 0 {
			return false
		}
		for _, child := range children {
			if !sleepSortErrorOnlyThrows(child) {
				return false
			}
		}
		return true
	case interface{ Unwrap() error }:
		return sleepSortErrorOnlyThrows(wrapped.Unwrap())
	default:
		return false
	}
}
