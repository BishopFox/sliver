package gobfuscate

import (
	"bytes"
	"fmt"
	insecureRand "math/rand"
	"text/template"
)

// strObfuscationCodeGen - Generic string obfuscation interface used so
// we can dynamically swap out obfuscators at runtime
type strObfuscationCodeGen func(str string) []byte

var defaultStrObfuscationCodeGens = []strObfuscationCodeGen{
	xorStringObfuscator,
	xorStringObfuscator,
	xorStringObfuscator,
	xorStringObfuscator,
}

// [ XOR ] ----------------------------------------------------------------------------

// xorStringData - Data related to xor'ing string obfuscation
type xorStringData struct {
	Mask      string
	MaskedStr string
}

var xorStringTmpl = `(func() string {
	mask := []byte("{{.Mask}}")
	masked := []byte("{{.MaskedStr}}")
	result := make([]byte, len(mask))
	for i, m := range mask {
		result[i] = m ^ masked[i]
	}
	return string(result)
}())`

// simple string xor mask obfuscation
func xorStringObfuscator(str string) []byte {
	xorStr := xorStringData{Mask: "", MaskedStr: ""}
	mask := make([]byte, len(str))
	for i := range mask {
		mask[i] = byte(insecureRand.Intn(256))
		xorStr.Mask += fmt.Sprintf("\\x%02x", mask[i])
	}
	for i, x := range []byte(str) {
		xorStr.MaskedStr += fmt.Sprintf("\\x%02x", x^mask[i])
	}
	buf := bytes.NewBuffer([]byte{})
	xorCode, _ := template.New("obfuscate").Parse(xorStringTmpl)
	xorCode.Execute(buf, xorStr)
	return buf.Bytes()
}

// [ APT ] ----------------------------------------------------------------------------

// [ AES ] ----------------------------------------------------------------------------
