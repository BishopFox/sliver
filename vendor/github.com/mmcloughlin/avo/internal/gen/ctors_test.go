package gen

import (
	"testing"

	"github.com/mmcloughlin/avo/internal/inst"
)

func TestParamsUniqueArgNames(t *testing.T) {
	for _, i := range inst.Instructions {
		s := params(i)
		for _, n := range i.Arities() {
			if n == 0 {
				continue
			}
			names := map[string]bool{}
			for j := 0; j < n; j++ {
				names[s.ParameterName(j)] = true
			}
			if len(names) != n {
				t.Errorf("repeated argument for instruction %s", i.Opcode)
			}
			if _, found := names[""]; found {
				t.Errorf("empty argument name for instruction %s", i.Opcode)
			}
		}
	}
}
