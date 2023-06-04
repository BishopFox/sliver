package wazeroir

import (
	"bytes"
)

const EntrypointLabel = ".entrypoint"

func Format(ops []UnionOperation) string {
	buf := bytes.NewBuffer(nil)

	_, _ = buf.WriteString(EntrypointLabel + "\n")
	for i := range ops {
		op := &ops[i]
		str := op.String()
		isLabel := op.Kind == OperationKindLabel
		if !isLabel {
			const indent = "\t"
			str = indent + str
		}
		_, _ = buf.WriteString(str + "\n")
	}
	return buf.String()
}
