package opcodescsv

import (
	"reflect"
	"strconv"
	"strings"

	"golang.org/x/arch/x86/x86csv"
)

// Alias records another possible name for an instruction configuration.
type Alias struct {
	Opcode      string
	DataSize    int
	NumOperands int
}

// BuildAliasMap constructs a map from AT&T/GNU/Intel to Go syntax.
func BuildAliasMap(is []*x86csv.Inst) (map[Alias]string, error) {
	m := map[Alias]string{}
	for _, i := range is {
		if skip(i) {
			continue
		}

		s, err := strconv.Atoi("0" + i.DataSize)
		if err != nil {
			return nil, err
		}

		for _, alt := range []string{i.IntelOpcode(), i.GNUOpcode()} {
			if strings.ToUpper(alt) != i.GoOpcode() {
				a := Alias{
					Opcode:      strings.ToLower(alt),
					DataSize:    s,
					NumOperands: len(i.GoArgs()),
				}
				m[a] = i.GoOpcode()
			}
		}
	}
	return m, nil
}

// OperandOrder describes the order an instruction takes its operands.
type OperandOrder uint8

// Possible operand orders.
const (
	UnknownOrder = iota
	IntelOrder
	ReverseIntelOrder
	CMP3Order
)

// BuildOrderMap collects operand order information from the instruction list.
func BuildOrderMap(is []*x86csv.Inst) map[string]OperandOrder {
	s := map[string]OperandOrder{}
	for _, i := range is {
		if skip(i) {
			continue
		}
		s[i.GoOpcode()] = order(i)
	}
	return s
}

// order categorizes the operand order of an instruction.
func order(i *x86csv.Inst) OperandOrder {
	// Is it Intel order already?
	intel := i.IntelArgs()
	if reflect.DeepEqual(i.GoArgs(), intel) {
		return IntelOrder
	}

	// Check if it's reverse Intel.
	for l, r := 0, len(intel)-1; l < r; l, r = l+1, r-1 {
		intel[l], intel[r] = intel[r], intel[l]
	}
	if reflect.DeepEqual(i.GoArgs(), intel) {
		return ReverseIntelOrder
	}

	// Otherwise we could be in the bizarre special-case of 3-argument CMP instructions.
	//
	// Reference: https://github.com/golang/arch/blob/b19384d3c130858bb31a343ea8fce26be71b5998/x86/x86spec/format.go#L138-L144
	//
	//			case "CMPPD", "CMPPS", "CMPSD", "CMPSS":
	//				// rotate destination to end but don't swap comparison operands
	//				if len(args) == 3 {
	//					args[0], args[1], args[2] = args[2], args[0], args[1]
	//					break
	//				}
	//				fallthrough
	//
	switch i.GoOpcode() {
	case "CMPPD", "CMPPS", "CMPSD", "CMPSS":
		if len(i.GoArgs()) == 3 {
			return CMP3Order
		}
	}

	return UnknownOrder
}

// skip decides whether to ignore the instruction for analysis purposes.
func skip(i *x86csv.Inst) bool {
	switch {
	case strings.ContainsAny(i.GoOpcode(), "/*"):
		return true
	case i.Mode64 == "I": // Invalid in 64-bit mode.
		return true
	}
	return false
}
