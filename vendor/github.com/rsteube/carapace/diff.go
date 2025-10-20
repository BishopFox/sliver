package carapace

import (
	"github.com/rsteube/carapace/internal/common"
	"github.com/rsteube/carapace/pkg/style"
)

// Diff compares values of two actions.
// It overrides the style to hightlight changes.
//
//	red:   only present in original
//	dim:   present in both
//	green: only present in new
func Diff(original, new Action) Action {
	return ActionCallback(func(c Context) Action {
		invokedBatch := Batch(
			original,
			new,
		).Invoke(c)

		merged := make(map[string]common.RawValue)
		for _, v := range invokedBatch[0].action.rawValues {
			v.Style = style.Red
			merged[v.Value] = v
		}

		for _, v := range invokedBatch[1].action.rawValues {
			if _, ok := merged[v.Value]; ok {
				v.Style = style.Dim
				merged[v.Value] = v
			} else {
				v.Style = style.Green
				merged[v.Value] = v
			}
		}

		mergedBatch := invokedBatch.Merge()
		mergedBatch.action.rawValues = make(common.RawValues, 0)
		for _, v := range merged {
			mergedBatch.action.rawValues = append(mergedBatch.action.rawValues, v)
		}
		return mergedBatch.ToA()
	})
}
