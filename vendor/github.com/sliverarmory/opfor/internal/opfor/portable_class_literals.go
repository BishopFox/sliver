package opfor

import (
	"fmt"
	"strings"

	"github.com/sliverarmory/opfor/internal/lexer"
)

// portableCompileException is the inert, pure-Go counterpart of Sleep's
// YourCodeSucksException. Keeping it as an object matters to scripts which
// call formatErrors() on checkError(); String retains the compact summary used
// by interpolation and debug warnings.
type portableCompileException struct {
	compile   *CompileError
	summary   string
	formatted string
}

func (exception *portableCompileException) Error() string {
	if exception == nil {
		return ""
	}
	return exception.summary
}

func (exception *portableCompileException) String() string { return exception.Error() }

func (exception *portableCompileException) Unwrap() error {
	if exception == nil {
		return nil
	}
	return exception.compile
}

func (exception *portableCompileException) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if exception == nil {
		return Null(), false, nil
	}
	if invocation.Op == ObjectTypeCheck {
		class := resolvePortableClassName(invocation.Class)
		return Bool(class == "sleep.error.YourCodeSucksException" || class == "java.lang.RuntimeException" ||
			class == "java.lang.Exception" || class == "java.lang.Throwable" || class == "java.lang.Object"), true, nil
	}
	if invocation.Op != ObjectInvoke || len(invocation.Arguments) != 0 {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "formatErrors":
		return String(exception.formatted), true, nil
	case "getMessage":
		return String(strings.TrimPrefix(exception.summary, "YourCodeSucksException: ")), true, nil
	case "toString":
		return String(exception.summary), true, nil
	}
	return Null(), false, nil
}

func validateDynamicClassLiterals(caller *fiber, sourceText string) error {
	if caller == nil || caller.closure == nil || caller.closure.script == nil {
		return nil
	}
	lexed := lexer.Lex(lexer.NewSource("eval", []byte(sourceText)))
	if len(lexed.Diagnostics) != 0 {
		// The ordinary dynamic compiler owns lexical and syntax diagnostics.
		return nil
	}
	var unresolved []lexer.Token
	for _, token := range lexed.Tokens {
		if token.Kind != lexer.Class {
			continue
		}
		resolved := caller.closure.script.resolveClass(token.Text)
		if portableClassCatalogContains(token.Text, resolved) || caller.closure.script.importedClass(token.Text, resolved) {
			continue
		}
		unresolved = append(unresolved, token)
	}
	if len(unresolved) == 0 {
		return nil
	}

	diagnostics := make([]Diagnostic, len(unresolved))
	parts := make([]string, len(unresolved))
	var formatted strings.Builder
	for index, token := range unresolved {
		description := "unable to resolve class: " + token.Text
		diagnostics[index] = Diagnostic{
			Severity: SeverityError,
			Code:     "CLASS001",
			Message:  description,
			Span:     token.Span,
		}
		line := token.Span.Start.Line - 1
		if line < 0 {
			line = 0
		}
		parts[index] = fmt.Sprintf("%s at %d", description, line)
		fmt.Fprintf(&formatted, "Error: %s at line %d\n       %s\n", description, line, sourceLine(sourceText, token.Span.Start.Line))
	}
	return &portableCompileException{
		compile:   &CompileError{Diagnostics: diagnostics},
		summary:   fmt.Sprintf("YourCodeSucksException: %d error(s): %s", len(parts), strings.Join(parts, "; ")),
		formatted: formatted.String(),
	}
}

func sourceLine(sourceText string, line int) string {
	if line <= 0 {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(sourceText, "\r\n", "\n"), "\n")
	if line > len(lines) {
		return ""
	}
	return lines[line-1]
}

func portableClassCatalogContains(original, resolved string) bool {
	baseOriginal := strings.TrimSuffix(original, "[]")
	baseResolved := strings.TrimSuffix(resolved, "[]")
	if _, ok := portableDefaultClasses[baseOriginal]; ok {
		return true
	}
	for _, class := range portableDefaultClasses {
		if baseResolved == class {
			return true
		}
	}
	if _, ok := portableJavaInterfaces[baseResolved]; ok {
		return true
	}
	if _, ok := portableImportedClasses[baseResolved]; ok {
		return true
	}
	switch baseResolved {
	case "java.lang.Character$Subset", "java.lang.reflect.Array", "java.awt.Point", "java.awt.geom.Point2D",
		"java.io.PrintStream", "java.io.FilterOutputStream", "java.io.OutputStream", "java.security.MessageDigest":
		return true
	default:
		return false
	}
}

func portableCompileTarget(target any, invocation ObjectInvocation) (Value, bool, error) {
	exception, ok := target.(*portableCompileException)
	if !ok || exception == nil {
		return Null(), false, nil
	}
	return exception.invoke(invocation)
}
