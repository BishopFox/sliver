package strutil

import (
	"io"
	"strings"
	"text/template"
)

// Template executes the given template text on data, writing the result to w.
func Template(w io.Writer, text string, data any) error {
	t := template.New("top")
	t.Funcs(templateFuncs)
	template.Must(t.Parse(text))

	return t.Execute(w, data)
}

var templateFuncs = template.FuncMap{
	"trim": strings.TrimSpace,
}
