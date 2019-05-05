package inst

//go:generate avogen -bootstrap -data ../data -output ztable.go godata
//go:generate avogen -bootstrap -data ../data -output ztable_test.go godatatest

// Lookup returns the instruction with the given opcode. Boolean return value
// indicates whether the instruction was found.
func Lookup(opcode string) (Instruction, bool) {
	for _, i := range Instructions {
		if i.Opcode == opcode {
			return i, true
		}
	}
	return Instruction{}, false
}
