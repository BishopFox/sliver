package callstack

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"runtime"
	"strconv"
	"strings"
)

type FrameInfo struct {
	CallStack string
	Func      string
	File      string
	LineNo    int
}

func GetCallStack(frames StackTrace) string {
	var trace []string
	for i := len(frames) - 1; i >= 0; i-- {
		trace = append(trace, fmt.Sprintf("%v", frames[i]))
	}
	return strings.Join(trace, " ")
}

// GetLastFrame returns Caller information on the first frame in the stack trace.
func GetLastFrame(frames StackTrace) FrameInfo {
	if len(frames) == 0 {
		return FrameInfo{}
	}
	pc := uintptr(frames[0]) - 1
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return FrameInfo{Func: fmt.Sprintf("unknown func at %v", pc)}
	}
	filePath, lineNo := fn.FileLine(pc)
	return FrameInfo{
		CallStack: GetCallStack(frames),
		Func:      FuncName(fn),
		File:      filePath,
		LineNo:    lineNo,
	}
}

// FuncName given a runtime function spec returns a short function name in
// format `<package name>.<function name>` or if the function has a receiver
// in format `<package name>.(<receiver>).<function name>`.
func FuncName(fn *runtime.Func) string {
	if fn == nil {
		return ""
	}
	funcPath := fn.Name()
	idx := strings.LastIndex(funcPath, "/")
	if idx == -1 {
		return funcPath
	}
	return funcPath[idx+1:]
}

// CallStack represents a stack of program counters.
type CallStack []uintptr

func (cs *CallStack) Format(st fmt.State, verb rune) {
	if verb == 'v' && st.Flag('+') {
		for _, pc := range *cs {
			f := Frame(pc)
			_, _ = fmt.Fprintf(st, "\n%+v", f)
		}
	}
}

func (cs *CallStack) StackTrace() StackTrace {
	f := make([]Frame, len(*cs))
	for i := 0; i < len(f); i++ {
		f[i] = Frame((*cs)[i])
	}
	return f
}

// New creates a new CallStack struct from current stack minus 'skip' number of frames.
func New(skip int) *CallStack {
	skip += 2
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(skip, pcs[:])
	var st CallStack = pcs[0:n]
	return &st
}

// GoRoutineID returns the current goroutine id.
func GoRoutineID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

// Frame represents a program counter inside a stack frame.
// For historical reasons if Frame is interpreted as a uintptr
// its value represents the program counter + 1.
type Frame uintptr

// pc returns the program counter for this frame;
// multiple frames may have the same PC value.
func (f Frame) pc() uintptr { return uintptr(f) - 1 }

// file returns the full path to the file that contains the
// function for this Frame's pc.
func (f Frame) file() string {
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return "unknown"
	}
	file, _ := fn.FileLine(f.pc())
	return file
}

// line returns the line number of source code of the
// function for this Frame's pc.
func (f Frame) line() int {
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return 0
	}
	_, line := fn.FileLine(f.pc())
	return line
}

// name returns the name of this function, if known.
func (f Frame) name() string {
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return "unknown"
	}
	return fn.Name()
}

// Format formats the frame according to the fmt.Formatter interface.
//
//	%s    source file
//	%d    source line
//	%n    function name
//	%v    equivalent to %s:%d
//
// Format accepts flags that alter the printing of some verbs, as follows:
//
//	%+s   function name and path of source file relative to the compile time
//	      GOPATH separated by \n\t (<funcname>\n\t<path>)
//	%+v   equivalent to %+s:%d
func (f Frame) Format(s fmt.State, verb rune) {
	switch verb {
	case 's':
		switch {
		case s.Flag('+'):
			_, _ = io.WriteString(s, f.name())
			_, _ = io.WriteString(s, "\n\t")
			_, _ = io.WriteString(s, f.file())
		default:
			_, _ = io.WriteString(s, path.Base(f.file()))
		}
	case 'd':
		_, _ = io.WriteString(s, strconv.Itoa(f.line()))
	case 'n':
		_, _ = io.WriteString(s, funcname(f.name()))
	case 'v':
		f.Format(s, 's')
		_, _ = io.WriteString(s, ":")
		f.Format(s, 'd')
	}
}

// MarshalText formats a stacktrace Frame as a text string. The output is the
// same as that of fmt.Sprintf("%+v", f), but without newlines or tabs.
func (f Frame) MarshalText() ([]byte, error) {
	name := f.name()
	if name == "unknown" {
		return []byte(name), nil
	}
	return []byte(fmt.Sprintf("%s %s:%d", name, f.file(), f.line())), nil
}

type HasStackTrace interface {
	StackTrace() StackTrace
}

// StackTrace is stack of Frames from innermost (newest) to outermost (oldest).
type StackTrace []Frame

// Format formats the stack of Frames according to the fmt.Formatter interface.
//
//	%s	lists source files for each Frame in the stack
//	%v	lists the source file and line number for each Frame in the stack
//
// Format accepts flags that alter the printing of some verbs, as follows:
//
//	%+v   Prints filename, function, and line number for each Frame in the stack.
func (st StackTrace) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		switch {
		case s.Flag('+'):
			for _, f := range st {
				_, _ = io.WriteString(s, "\n")
				f.Format(s, verb)
			}
		case s.Flag('#'):
			_, _ = fmt.Fprintf(s, "%#v", []Frame(st))
		default:
			st.formatSlice(s, verb)
		}
	case 's':
		st.formatSlice(s, verb)
	}
}

// formatSlice will format this StackTrace into the given buffer as a slice of
// Frame, only valid when called with '%s' or '%v'.
func (st StackTrace) formatSlice(s fmt.State, verb rune) {
	_, _ = io.WriteString(s, "[")
	for i, f := range st {
		if i > 0 {
			_, _ = io.WriteString(s, " ")
		}
		f.Format(s, verb)
	}
	_, _ = io.WriteString(s, "]")
}

// funcname removes the path prefix component of a function's name reported by func.Name().
func funcname(name string) string {
	i := strings.LastIndex(name, "/")
	name = name[i+1:]
	i = strings.Index(name, ".")
	return name[i+1:]
}
