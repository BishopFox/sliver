package opfor

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"
	"sync"
)

// SourceRequest describes a source requested by Sleep's include function.
// Name is the file or archive member named by the script. Container is empty
// for include("file.sl") and contains the first argument for the two-argument
// include("container", "member.sl") form. IncludingSource and Span identify
// the call site without exposing interpreter internals to an embedder.
type SourceRequest struct {
	Script          ScriptID
	IncludingSource string
	Container       string
	Name            string
	Span            Span
}

type sourceLookupMode uint8

const (
	sourceLookupDirect sourceLookupMode = iota
	sourceLookupClasspath
)

// SourceResolver loads source for include. Implementations may resolve files,
// embedded assets, databases, archives, or application-owned virtual modules.
// ResolveSource is synchronous and may be called concurrently by independent
// script callbacks. Implementations should observe ctx and must not retain it
// after returning.
//
// OPFOR copies Source.Data while compiling and does not retain the resolver's
// backing slice in a Program. That copy or ScriptLoader decoding occurs after
// ResolveSource returns, however, so the resolver must treat returned Data as
// immutable after return rather than immediately reusing or mutating its
// backing storage.
type SourceResolver interface {
	ResolveSource(context.Context, SourceRequest) (Source, error)
}

// SourceResolverFunc adapts a function to SourceResolver.
type SourceResolverFunc func(context.Context, SourceRequest) (Source, error)

// ResolveSource calls f with request.
func (f SourceResolverFunc) ResolveSource(ctx context.Context, request SourceRequest) (Source, error) {
	if f == nil {
		return Source{}, errors.New("opfor: source resolver function is nil")
	}
	return f(ctx, request)
}

// WithSourceResolver replaces the filesystem resolver used by include.
func WithSourceResolver(resolver SourceResolver) Option {
	return func(config *runtimeConfig) error {
		if function, ok := resolver.(SourceResolverFunc); ok && function == nil {
			return errors.New("opfor: source resolver function is nil")
		}
		if isNilInterface(resolver) {
			return errors.New("opfor: source resolver is nil")
		}
		if config.sleepClasspathSet {
			return errors.New("opfor: WithSleepClasspath cannot be combined with WithSourceResolver")
		}
		config.sourceResolver = resolver
		return nil
	}
}

// WithSleepClasspath configures the semicolon- or colon-separated source and
// container search path used by include(...) and import ... from:. The
// option configures OPFOR's runtime-owned default FileSourceResolver and is
// intentionally incompatible with WithSourceResolver: importer-provided
// resolvers remain entirely importer-owned and may expose different lookup
// semantics.
func WithSleepClasspath(classPath string) Option {
	return func(config *runtimeConfig) error {
		if config.sourceResolver != nil {
			return errors.New("opfor: WithSleepClasspath cannot be combined with WithSourceResolver")
		}
		config.sleepClasspath = classPath
		config.sleepClasspathSet = true
		return nil
	}
}

// FileSourceResolver implements Sleep's ordinary file include form and its
// two-argument directory/JAR form. Relative paths are rooted at the resolver's
// base directory; it does not mutate the process working directory.
type FileSourceResolver struct {
	mu            sync.RWMutex
	baseDirectory string
	classPath     []string
}

// resolvedIncludeSource keeps the source name used by the compiler separate
// from the file value assigned to $__INCLUDE__. Sleep compiles includes using
// the requested logical name, while exposing the resolved java.io.File to the
// script. modificationPath is populated only when the resolver has concrete
// local mtime evidence; custom SourceResolver names are not inferred as paths.
type resolvedIncludeSource struct {
	source           Source
	includeValue     string
	identity         string
	modificationPath string
	reservedBytes    uint64
}

type missingIncludeContainerError struct {
	container string
}

func (e *missingIncludeContainerError) Error() string {
	if e == nil {
		return "&include: could not locate source"
	}
	return fmt.Sprintf("&include: could not locate source '%s'", e.container)
}

// NewFileSourceResolver constructs a filesystem resolver. An empty base uses
// the process working directory at construction time. JAR and ZIP containers
// are read with Go's archive/zip package and do not require Java or CGO.
func NewFileSourceResolver(baseDirectory string) (*FileSourceResolver, error) {
	if baseDirectory == "" {
		workingDirectory, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("opfor: determine source base directory: %w", err)
		}
		baseDirectory = workingDirectory
	}
	absolute, err := filepath.Abs(baseDirectory)
	if err != nil {
		return nil, fmt.Errorf("opfor: resolve source base directory %q: %w", baseDirectory, err)
	}
	return &FileSourceResolver{baseDirectory: filepath.Clean(absolute)}, nil
}

// BaseDirectory returns the absolute directory used for relative paths.
func (r *FileSourceResolver) BaseDirectory() string {
	if r == nil {
		return ""
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.baseDirectory
}

func (r *FileSourceResolver) setBaseDirectory(directory string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.baseDirectory = filepath.Clean(directory)
	r.mu.Unlock()
}

// SetSleepClasspath configures the semicolon- or colon-separated source and
// container search path used by include(...) and import ... from:.
// Relative entries follow the resolver's current base directory. An empty
// path restores Sleep's default single current-directory entry.
func (r *FileSourceResolver) SetSleepClasspath(classPath string) {
	if r == nil {
		return
	}
	parts := splitSleepClasspath(classPath)
	if classPath == "" {
		parts = []string{"."}
	}
	entries := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			part = "."
		}
		entries = append(entries, filepath.FromSlash(part))
	}
	r.mu.Lock()
	r.classPath = entries
	r.mu.Unlock()
}

// splitSleepClasspath accepts both separators recognized by Sleep without
// fragmenting Windows volume names when a classpath is configured or tested
// on another operating system. Semicolons always split; colons split unless
// they are the drive designator at the start of the current entry.
func splitSleepClasspath(classPath string) []string {
	parts := make([]string, 0, strings.Count(classPath, ":")+strings.Count(classPath, ";")+1)
	start := 0
	for index := 0; index < len(classPath); index++ {
		separator := classPath[index]
		if separator != ':' && separator != ';' {
			continue
		}
		if separator == ':' && sleepClasspathVolumeColon(classPath, start, index) {
			continue
		}
		parts = append(parts, classPath[start:index])
		start = index + 1
	}
	return append(parts, classPath[start:])
}

func sleepClasspathVolumeColon(classPath string, start, index int) bool {
	if index != start+1 || start < 0 || index+1 >= len(classPath) {
		return false
	}
	letter := classPath[start]
	if (letter < 'A' || letter > 'Z') && (letter < 'a' || letter > 'z') {
		return false
	}
	return classPath[index+1] == '/' || classPath[index+1] == '\\'
}

// SleepClasspath returns a detached snapshot of the configured search path.
func (r *FileSourceResolver) SleepClasspath() []string {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.classPath) == 0 {
		return []string{"."}
	}
	return append([]string(nil), r.classPath...)
}

func (r *FileSourceResolver) setSleepClasspathEntries(entries []string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.classPath = append([]string(nil), entries...)
	r.mu.Unlock()
}

// ResolveSource reads one file, a file beneath a directory container, or a
// member of a ZIP/JAR container.
func (r *FileSourceResolver) ResolveSource(ctx context.Context, request SourceRequest) (Source, error) {
	resolved, err := r.resolveIncludeSource(ctx, request)
	if err != nil {
		return Source{}, err
	}
	return resolved.source, nil
}

func (r *FileSourceResolver) resolveIncludeSource(ctx context.Context, request SourceRequest) (resolvedIncludeSource, error) {
	return r.resolveIncludeSourceReserved(ctx, request, nil)
}

func (r *FileSourceResolver) resolveIncludeSourceReserved(
	ctx context.Context,
	request SourceRequest,
	reserve func(uint64) error,
) (resolvedIncludeSource, error) {
	return r.resolveIncludeSourceReservedMode(ctx, request, reserve, sourceLookupClasspath)
}

func (r *FileSourceResolver) resolveIncludeSourceReservedMode(
	ctx context.Context,
	request SourceRequest,
	reserve func(uint64) error,
	mode sourceLookupMode,
) (resolvedIncludeSource, error) {
	if r == nil {
		return resolvedIncludeSource{}, errors.New("opfor: file source resolver is nil")
	}
	if err := ctx.Err(); err != nil {
		return resolvedIncludeSource{}, err
	}
	name := request.Name
	if name == "" {
		return resolvedIncludeSource{}, errors.New("opfor: include source name is empty")
	}
	if request.Container == "" {
		// BasicUtilities routes the one-argument include form through
		// ParserConfig.findJarFile too. Preserve its direct-path-first lookup,
		// then search the configured Sleep classpath before reporting the direct
		// path's FileNotFoundException.
		filename := r.resolvePath(name)
		if mode == sourceLookupClasspath {
			filename = r.resolveContainer(name)
		}
		reader, err := os.Open(filename)
		if err != nil {
			return resolvedIncludeSource{}, portableFileNotFoundError(filename, err)
		}
		data, reserved, readErr := readReservedSource(ctx, reader, reserve)
		closeErr := reader.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			if isExecutionResourceError(err) {
				return resolvedIncludeSource{}, err
			}
			return resolvedIncludeSource{}, portableFileNotFoundError(filename, err)
		}
		return resolvedIncludeSource{
			source:           NewSource(filepath.Base(filepath.FromSlash(name)), data),
			includeValue:     filename,
			identity:         filename,
			modificationPath: filename,
			reservedBytes:    reserved,
		}, nil
	}

	container := r.resolveContainer(request.Container)
	info, err := os.Stat(container)
	if err != nil {
		return resolvedIncludeSource{}, &missingIncludeContainerError{container: request.Container}
	}
	if info.IsDir() {
		filename := filepath.Clean(filepath.Join(container, filepath.FromSlash(name)))
		reader, openErr := os.Open(filename)
		if openErr != nil {
			return resolvedIncludeSource{
				includeValue:     filename,
				identity:         filename,
				modificationPath: filename,
			}, fmt.Errorf("java.io.IOException: unable to locate %s from: %s", name, request.Container)
		}
		data, reserved, readErr := readReservedSource(ctx, reader, reserve)
		closeErr := reader.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			if isExecutionResourceError(err) {
				return resolvedIncludeSource{modificationPath: filename}, err
			}
			return resolvedIncludeSource{
				includeValue:     filename,
				identity:         filename,
				modificationPath: filename,
			}, fmt.Errorf("java.io.IOException: unable to locate %s from: %s", name, request.Container)
		}
		return resolvedIncludeSource{
			source:           NewSource(name, data),
			includeValue:     filename,
			identity:         filename,
			modificationPath: filename,
			reservedBytes:    reserved,
		}, nil
	}
	return resolveArchiveSource(ctx, container, request.Container, name, reserve)
}

type sourceContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r sourceContextReader) Read(destination []byte) (int, error) {
	if err := executionContextError(r.ctx); err != nil {
		return 0, err
	}
	read, err := r.reader.Read(destination)
	if fatalErr := executionContextError(r.ctx); fatalErr != nil {
		return 0, errors.Join(fatalErr, err)
	}
	return read, err
}

type reservedSourceBuffer struct {
	data     []byte
	reserve  func(uint64) error
	reserved uint64
}

func (b *reservedSourceBuffer) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	if b.reserve != nil {
		if err := b.reserve(uint64(len(data))); err != nil {
			return 0, err
		}
		b.reserved += uint64(len(data))
	}
	b.data = append(b.data, data...)
	return len(data), nil
}

// readReservedSource charges each bounded input chunk before growing the
// retained source buffer. A nil reservation preserves standalone
// FileSourceResolver behavior without imposing a Runtime policy.
func readReservedSource(ctx context.Context, reader io.Reader, reserve func(uint64) error) ([]byte, uint64, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	buffer := &reservedSourceBuffer{reserve: reserve}
	_, err := io.CopyBuffer(buffer, sourceContextReader{ctx: ctx, reader: reader}, make([]byte, 32*1024))
	return buffer.data, buffer.reserved, err
}

func (r *FileSourceResolver) resolvePath(name string) string {
	name = filepath.FromSlash(name)
	if filepath.IsAbs(name) {
		return filepath.Clean(name)
	}
	r.mu.RLock()
	baseDirectory := r.baseDirectory
	r.mu.RUnlock()
	return filepath.Clean(filepath.Join(baseDirectory, name))
}

func (r *FileSourceResolver) resolveContainer(name string) string {
	direct := r.resolvePath(name)
	if _, err := os.Stat(direct); err == nil {
		return direct
	}
	r.mu.RLock()
	baseDirectory := r.baseDirectory
	classPath := append([]string(nil), r.classPath...)
	r.mu.RUnlock()
	if len(classPath) == 0 {
		classPath = []string{"."}
	}
	for _, entry := range classPath {
		if !filepath.IsAbs(entry) {
			entry = filepath.Join(baseDirectory, entry)
		}
		candidate := filepath.Clean(filepath.Join(entry, filepath.FromSlash(name)))
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return direct
}

type importArchiveInspection struct {
	local        bool
	exists       bool
	entryChecked bool
	hasEntry     bool
}

// inspectImportArchive performs only filesystem and ZIP-directory checks. It
// never loads or interprets a class. An arbitrary custom SourceResolver leaves
// archive ownership with the embedding Host instead of treating its virtual
// path as a missing local file; the concrete FileSourceResolver keeps its
// documented local-file behavior even when importer-supplied.
func (r *Runtime) inspectImportArchive(ctx context.Context, target, archive string) (importArchiveInspection, error) {
	resolver := r.concreteFileSourceResolver()
	if resolver == nil {
		return importArchiveInspection{}, nil
	}
	if err := ctx.Err(); err != nil {
		return importArchiveInspection{}, err
	}
	path := resolver.resolveContainer(archive)
	inspection := importArchiveInspection{local: true}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return inspection, nil
		}
		return inspection, err
	}
	inspection.exists = true
	if strings.HasSuffix(strings.TrimSpace(target), ".*") {
		return inspection, nil
	}

	inspection.entryChecked = true
	entry := strings.ReplaceAll(strings.TrimSpace(target), ".", "/") + ".class"
	if info.IsDir() {
		entryInfo, statErr := os.Stat(filepath.Join(path, filepath.FromSlash(entry)))
		if statErr == nil && !entryInfo.IsDir() {
			inspection.hasEntry = true
		} else if statErr != nil && !os.IsNotExist(statErr) {
			return inspection, statErr
		}
		return inspection, nil
	}

	reader, err := zip.OpenReader(path)
	if err != nil {
		// The archive exists but cannot supply the requested class entry. The
		// caller may still let an importer Host handle it before reporting the
		// deterministic missing-class boundary.
		return inspection, nil
	}
	defer reader.Close()
	for _, file := range reader.File {
		if err := ctx.Err(); err != nil {
			return importArchiveInspection{}, err
		}
		if file.Name == entry && !file.FileInfo().IsDir() {
			inspection.hasEntry = true
			break
		}
	}
	return inspection, nil
}

// concreteFileSourceResolver returns only OPFOR's exact filesystem resolver.
// Arbitrary SourceResolver implementations retain ownership of virtual import
// archives, while an importer that deliberately supplies *FileSourceResolver
// keeps its documented filesystem and sleep.classpath behavior.
func (r *Runtime) concreteFileSourceResolver() *FileSourceResolver {
	if r == nil {
		return nil
	}
	if r.defaultFileResolver != nil {
		return r.defaultFileResolver
	}
	resolver, _ := r.resolver.(*FileSourceResolver)
	return resolver
}

func resolveArchiveSource(
	ctx context.Context,
	container, displayContainer, member string,
	reserve func(uint64) error,
) (resolvedIncludeSource, error) {
	archive, err := zip.OpenReader(container)
	if err != nil {
		return resolvedIncludeSource{
			includeValue:     container,
			identity:         container + "!/" + strings.TrimPrefix(pathpkg.Clean("/"+filepath.ToSlash(member)), "/"),
			modificationPath: container,
		}, fmt.Errorf("java.io.IOException: unable to locate %s from: %s", member, displayContainer)
	}
	defer archive.Close()

	wanted := strings.TrimPrefix(pathpkg.Clean("/"+filepath.ToSlash(member)), "/")
	for _, entry := range archive.File {
		if err := ctx.Err(); err != nil {
			return resolvedIncludeSource{modificationPath: container}, err
		}
		candidate := strings.TrimPrefix(pathpkg.Clean("/"+entry.Name), "/")
		if candidate != wanted || entry.FileInfo().IsDir() {
			continue
		}
		reader, openErr := entry.Open()
		if openErr != nil {
			return resolvedIncludeSource{modificationPath: container}, fmt.Errorf("opfor: open %q from %q: %w", member, container, openErr)
		}
		data, reserved, readErr := readReservedSource(ctx, reader, reserve)
		closeErr := reader.Close()
		if readErr != nil {
			return resolvedIncludeSource{modificationPath: container}, fmt.Errorf("opfor: read %q from %q: %w", member, container, readErr)
		}
		if closeErr != nil {
			return resolvedIncludeSource{modificationPath: container}, fmt.Errorf("opfor: close %q from %q: %w", member, container, closeErr)
		}
		return resolvedIncludeSource{
			source:           NewSource(member, data),
			includeValue:     container,
			identity:         container + "!/" + wanted,
			modificationPath: container,
			reservedBytes:    reserved,
		}, nil
	}
	return resolvedIncludeSource{
		includeValue:     container,
		identity:         container + "!/" + wanted,
		modificationPath: container,
	}, fmt.Errorf("java.io.IOException: unable to locate %s from: %s", member, displayContainer)
}

// ErrIncludeCycle identifies a recursive include chain stopped by OPFOR.
var ErrIncludeCycle = errors.New("opfor: include cycle")

// IncludeCycleError reports the source chain that attempted to recurse.
type IncludeCycleError struct {
	Chain []string
}

// Error formats the recursive source chain.
func (e *IncludeCycleError) Error() string {
	if e == nil || len(e.Chain) == 0 {
		return ErrIncludeCycle.Error()
	}
	return ErrIncludeCycle.Error() + ": " + strings.Join(e.Chain, " -> ")
}

// Unwrap supports errors.Is(err, ErrIncludeCycle).
func (e *IncludeCycleError) Unwrap() error { return ErrIncludeCycle }

type currentFiberContextKey struct{}
type includeChainContextKey struct{}

type includeChainEntry struct {
	identity string
	name     string
}

func withCurrentFiber(ctx context.Context, fiber *fiber) context.Context {
	return context.WithValue(ctx, currentFiberContextKey{}, fiber)
}

func currentFiber(ctx context.Context) *fiber {
	if ctx == nil {
		return nil
	}
	fiber, _ := ctx.Value(currentFiberContextKey{}).(*fiber)
	return fiber
}

func (r *Runtime) dynamicSourceFunctions() map[string]NativeFunc {
	return map[string]NativeFunc{
		"eval":    r.evalSource,
		"expr":    r.evalExpression,
		"include": r.includeSource,
	}
}

func (r *Runtime) evalSource(ctx context.Context, invocation Invocation) (Value, error) {
	if len(invocation.Arguments) == 0 {
		if currentFiber(ctx) != nil {
			return Null(), sleepBridgeEmptyStack()
		}
		return Null(), errors.New("&eval: missing source code")
	}
	caller, err := dynamicSourceCaller(r, ctx, invocation)
	if err != nil {
		return Null(), err
	}
	program, err := r.CompileString("eval", invocation.Arg(0).String())
	if err != nil {
		return r.flagSourceError(invocation, err)
	}
	return runDynamicProgram(ctx, caller, program, false)
}

func (r *Runtime) evalExpression(ctx context.Context, invocation Invocation) (Value, error) {
	if len(invocation.Arguments) == 0 {
		if currentFiber(ctx) != nil {
			return Null(), sleepBridgeEmptyStack()
		}
		return Null(), errors.New("&expr: missing expression")
	}
	caller, err := dynamicSourceCaller(r, ctx, invocation)
	if err != nil {
		return Null(), err
	}
	expression := invocation.Arg(0).String()
	const prefix, suffix = "return (", ");"
	if _, err := r.reserveSourceLength(len(prefix)+len(expression)+len(suffix), 0); err != nil {
		return Null(), err
	}
	if err := validateDynamicClassLiterals(caller, expression); err != nil {
		return r.flagSourceError(invocation, err)
	}
	program, err := r.compileReservedSource(NewSource("eval", []byte(prefix+expression+suffix)))
	if err != nil {
		return r.flagSourceError(invocation, err)
	}
	return runDynamicProgram(ctx, caller, program, false)
}

func (r *Runtime) resolveSourceRequest(ctx context.Context, request SourceRequest) (resolvedIncludeSource, error) {
	return r.resolveSourceRequestMode(ctx, request, sourceLookupClasspath)
}

func (r *Runtime) resolveSourceRequestMode(ctx context.Context, request SourceRequest, mode sourceLookupMode) (resolvedIncludeSource, error) {
	var resolved resolvedIncludeSource
	var err error
	if resolver, ok := r.resolver.(interface {
		resolveIncludeSourceReservedMode(context.Context, SourceRequest, func(uint64) error, sourceLookupMode) (resolvedIncludeSource, error)
	}); ok {
		resolved, err = resolver.resolveIncludeSourceReservedMode(ctx, request, func(amount uint64) error {
			return r.reserveResource(resourceSourceBytes, amount)
		}, mode)
	} else if resolver, ok := r.resolver.(interface {
		resolveIncludeSourceReserved(context.Context, SourceRequest, func(uint64) error) (resolvedIncludeSource, error)
	}); ok {
		resolved, err = resolver.resolveIncludeSourceReserved(ctx, request, func(amount uint64) error {
			return r.reserveResource(resourceSourceBytes, amount)
		})
	} else if resolver, ok := r.resolver.(interface {
		resolveIncludeSource(context.Context, SourceRequest) (resolvedIncludeSource, error)
	}); ok {
		resolved, err = resolver.resolveIncludeSource(ctx, request)
	} else {
		resolved.source, err = r.resolver.ResolveSource(ctx, request)
		resolved.includeValue = resolved.source.Name
		resolved.identity = resolved.source.Name
	}
	err = joinExecutionContextError(ctx, err)
	if err != nil {
		return resolved, err
	}
	resolved.reservedBytes, err = r.reserveSourceLength(len(resolved.source.Data), resolved.reservedBytes)
	return resolved, err
}

func (r *Runtime) includeSource(ctx context.Context, invocation Invocation) (Value, error) {
	if len(invocation.Arguments) == 0 {
		// Sleep 2.1's BasicUtilities pops the first Scalar before entering its
		// include error handler. An empty argument stack therefore aborts the
		// current block with this internal warning, without touching
		// $__INCLUDE__ or checkError().
		return Null(), &uncaughtScriptWarning{
			err: errors.New("internal error - class java.util.EmptyStackException"),
		}
	}
	caller, err := dynamicSourceCaller(r, ctx, invocation)
	if err != nil {
		return Null(), err
	}
	request := SourceRequest{
		Script:          invocation.Script,
		IncludingSource: invocation.Span.Source,
		Name:            invocation.Arg(0).String(),
		Span:            invocation.Span,
	}
	if len(invocation.Arguments) == 2 {
		request.Container = request.Name
		request.Name = invocation.Arg(1).String()
	}
	// Sleep selects the container/member form only when the argument stack has
	// exactly two entries. Every other positive arity consumes argument zero as
	// a standalone source and silently ignores the remaining values.
	resolved, err := r.resolveSourceRequestMode(ctx, request, sourceLookupClasspath)
	if r.scriptLoaderInstance != nil && resolved.modificationPath != "" {
		r.scriptLoaderInstance.associateSourceFile(resolved.modificationPath)
	}
	// BasicUtilities assigns $__INCLUDE__ after resolving a two-argument
	// container but before opening the requested member. A missing member
	// therefore leaves the member file (directory) or archive file visible.
	// The one-argument FileInputStream path fails before this assignment.
	includeAssigned := strings.TrimSpace(resolved.includeValue) != ""
	if includeAssigned {
		cell, resolveErr := caller.scope.resolveAt(ctx, "$__INCLUDE__", invocation.Span)
		if resolveErr != nil {
			return Null(), resolveErr
		}
		if err := caller.setCellAtExecution(ctx, cell, String(resolved.includeValue), Span{}); err != nil {
			return Null(), err
		}
	}
	if err != nil {
		if isExecutionResourceError(err) {
			return Null(), err
		}
		var missingContainer *missingIncludeContainerError
		if errors.As(err, &missingContainer) {
			return Null(), &uncaughtScriptWarning{err: missingContainer}
		}
		return r.flagSourceError(invocation, err)
	}
	source := resolved.source
	if source.Name == "" {
		source.Name = request.Name
		if request.Container == "" {
			source.Name = filepath.Base(filepath.FromSlash(request.Name))
		}
	}
	if resolved.includeValue == "" {
		resolved.includeValue = source.Name
	}
	if resolved.identity == "" {
		resolved.identity = resolved.includeValue
	}
	if !includeAssigned {
		cell, resolveErr := caller.scope.resolveAt(ctx, "$__INCLUDE__", invocation.Span)
		if resolveErr != nil {
			return Null(), resolveErr
		}
		if err := caller.setCellAtExecution(ctx, cell, String(resolved.includeValue), Span{}); err != nil {
			return Null(), err
		}
	}

	ctx, err = pushIncludedSource(ctx, caller, r.includeCycles, source.Name, resolved.identity)
	if err != nil {
		// Recursive include rejection is an intentional OPFOR resource-safety
		// extension. Keep its typed fatal error instead of routing it through
		// Sleep's ordinary source-open soft-error slot.
		if errors.Is(err, ErrIncludeCycle) {
			return Null(), err
		}
		return r.flagSourceError(invocation, err)
	}
	program, err := r.compileReservedSource(source)
	if err != nil {
		return r.flagSourceError(invocation, &sourceCompileFailure{source: source, cause: err})
	}
	_, err = runDynamicProgram(ctx, caller, program, true)
	if err != nil {
		return Null(), err
	}
	return Null(), nil
}

func dynamicSourceCaller(runtime *Runtime, ctx context.Context, invocation Invocation) (*fiber, error) {
	caller := currentFiber(ctx)
	if caller == nil || caller.closure == nil || caller.closure.script == nil {
		return nil, fmt.Errorf("&%s: dynamic source evaluation requires an active script", invocation.Name)
	}
	if runtime == nil || caller.closure.script.runtime != runtime {
		return nil, fmt.Errorf("&%s: active script belongs to a different runtime", invocation.Name)
	}
	if invocation.Script != 0 && caller.closure.script.id != invocation.Script {
		return nil, fmt.Errorf("&%s: active script does not match invocation owner", invocation.Name)
	}
	return caller, nil
}

type dynamicSourceExecution struct {
	discardResult bool
	includeChain  []includeChainEntry
}

func runDynamicProgram(ctx context.Context, caller *fiber, program *Program, discardResult bool) (Value, error) {
	if caller == nil || caller.closure == nil || caller.closure.script == nil || program == nil || program.function == nil {
		return Null(), errors.New("opfor: invalid dynamic source execution")
	}
	function := *program.function
	// BasicUtilities routes eval, expr, and include through
	// SleepUtils.runCode/InlineCallRequest in the owning SleepClosure's active
	// ScriptEnvironment. A dynamic yield saves that Block's context but
	// runCode clears the flow-control flag, so its value returns normally and
	// the surrounding expression continues. The owning closure retains the
	// dynamic context group only after the outer invocation finishes.
	function.Name = "<eval>"
	collector := caller.continuation
	if collector == nil {
		collector = &continuationCollector{}
		caller.continuation = collector
	}

	chain, _ := ctx.Value(includeChainContextKey{}).([]includeChainEntry)
	dynamic := &fiber{
		closure: caller.closure, function: &function, scope: caller.scope,
		locals:    append([]*scope(nil), caller.locals...),
		lastMatch: append([]Value(nil), caller.lastMatch...), regexCursors: caller.regexCursors,
		continuation:  collector,
		swallowCallCC: true,
		dynamicSource: &dynamicSourceExecution{
			discardResult: discardResult,
			includeChain:  append([]includeChainEntry(nil), chain...),
		},
	}
	value, yielded, err := dynamic.run(ctx)
	dynamic.swallowCallCC = false
	caller.adoptInlineState(dynamic)
	var transfer *callCCTransfer
	if errors.As(err, &transfer) && transfer.fiber == dynamic && transfer.source == caller.closure {
		// SleepUtils.runCode resets FLOW_CONTROL_CALLCC before returning to the
		// eval/include bridge. The target closure is therefore the eval scalar;
		// it is not invoked. The saved dynamic context resumes after callcc on
		// the owning closure's next invocation.
		collector.append(dynamic)
		if discardResult {
			return Null(), nil
		}
		return FunctionValue(transfer.target), nil
	}
	var control *loopControl
	if errors.As(err, &control) {
		// SleepUtils.runCode resets an unmatched break/continue request before
		// returning from eval, expr, or include. The evaluated operand (if any)
		// has already run, but the dynamic block's result is the empty scalar.
		return Null(), nil
	}
	var exited *scriptExit
	if errors.As(err, &exited) {
		// SleepUtils.runCode resets an empty-throw exit request after executing
		// eval/include source. The dynamic Block stops and any non-empty exit
		// message has already been reported, but the owning caller resumes.
		return Null(), nil
	}
	if err != nil {
		return Null(), err
	}
	if yielded {
		collector.append(dynamic)
		if discardResult {
			return Null(), nil
		}
		return value, nil
	}
	if discardResult {
		return Null(), nil
	}
	return value, nil
}

func dynamicSourceResumeContext(ctx context.Context, suspended *fiber) context.Context {
	if ctx == nil || suspended == nil || suspended.dynamicSource == nil || len(suspended.dynamicSource.includeChain) == 0 {
		return ctx
	}
	chain := append([]includeChainEntry(nil), suspended.dynamicSource.includeChain...)
	return context.WithValue(ctx, includeChainContextKey{}, chain)
}

func (r *Runtime) flagSourceError(invocation Invocation, err error) (Value, error) {
	if err == nil {
		return Null(), nil
	}
	// Resource exhaustion is authoritative. Sleep compatibility softens
	// ordinary source-open and compile failures, but a script must not clear a
	// quota violation through checkError() and continue allocating.
	if isExecutionResourceError(err) {
		return Null(), err
	}
	script := r.script(invocation.Script)
	if script == nil {
		return Null(), err
	}
	message := sleepSourceErrorMessage(invocation.Name, err)
	errorValue := sourceErrorValue(message, err)
	script.mu.Lock()
	script.lastError = errorValue
	debug := script.debug
	script.mu.Unlock()
	if debug&2 == 2 {
		if debug&32 == 32 {
			// Sleep's DEBUG_THROW_WARNINGS path throws checkError(), which
			// consumes the pending soft error before transferring control.
			script.mu.Lock()
			script.lastError = Null()
			script.mu.Unlock()
			return Null(), &scriptThrow{value: errorValue}
		}
		r.writeWarning("checkError(): "+message, invocation.Span)
	}
	return Null(), nil
}

func sourceErrorValue(message string, cause error) Value {
	var compileException *portableCompileException
	if errors.As(cause, &compileException) && compileException != nil {
		return ObjectValue(compileException)
	}
	if strings.HasPrefix(message, "YourCodeSucksException:") {
		detail := strings.TrimSpace(strings.TrimPrefix(message, "YourCodeSucksException:"))
		return ObjectValue(&portableJavaException{
			class:   "sleep.error.YourCodeSucksException",
			message: detail,
			text:    message,
			cause:   cause,
		})
	}
	exception := newPortableJavaException(fmt.Errorf("%s", message))
	exception.cause = cause
	return ObjectValue(exception)
}

func sleepSourceErrorMessage(operation string, err error) string {
	var compileError *CompileError
	if !errors.As(err, &compileError) || len(compileError.Diagnostics) == 0 {
		return err.Error()
	}
	hasRunawayString := false
	var runawayString Diagnostic
	for _, diagnostic := range compileError.Diagnostics {
		if diagnostic.Severity == SeverityError && diagnostic.Code == "LEX002" {
			hasRunawayString = true
			runawayString = diagnostic
			break
		}
	}
	var failure *sourceCompileFailure
	if hasRunawayString && errors.As(err, &failure) {
		if message := sleepRunawayCompileMessage(failure.source, runawayString); message != "" {
			return message
		}
	}
	parts := make([]string, 0, len(compileError.Diagnostics))
	appendDiagnostic := func(diagnostic Diagnostic, message string) {
		line := diagnostic.Span.Start.Line
		if operation == "eval" || operation == "expr" || operation == "compile_closure" {
			line--
			if line < 0 {
				line = 0
			}
		}
		parts = append(parts, fmt.Sprintf("%s at %d", message, line))
	}
	// The reference parser reports a missing close parenthesis before the
	// runaway string that caused it. Suppress our extra missing-comma cascade.
	if hasRunawayString {
		for _, diagnostic := range compileError.Diagnostics {
			if diagnostic.Severity == SeverityError && diagnostic.Code == "PAR001" && strings.Contains(diagnostic.Message, "')' after arguments") {
				appendDiagnostic(runawayString, "Mismatched Parentheses - missing close paren")
			}
		}
	}
	for _, diagnostic := range compileError.Diagnostics {
		if diagnostic.Severity != SeverityError {
			continue
		}
		if hasRunawayString {
			switch diagnostic.Code {
			case "LEX002":
				appendDiagnostic(diagnostic, "Runaway string")
				continue
			case "PAR006":
				continue
			case "PAR001":
				if strings.Contains(diagnostic.Message, "')' after arguments") {
					continue
				}
			}
		}
		appendDiagnostic(diagnostic, diagnostic.Message)
	}
	if len(parts) == 0 {
		return err.Error()
	}
	return fmt.Sprintf("YourCodeSucksException: %d error(s): %s", len(parts), strings.Join(parts, "; "))
}

type sourceCompileFailure struct {
	source Source
	cause  error
}

func (e *sourceCompileFailure) Error() string {
	if e == nil || e.cause == nil {
		return "opfor: compilation failed"
	}
	return e.cause.Error()
}

func (e *sourceCompileFailure) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// sleepRunawayCompileMessage reproduces the compact parser summary used by
// YourCodeSucksException for a string which consumes later delimiters. OPFOR's
// lexer correctly identifies the final unmatched quote; Sleep attributes the
// cascade to the earlier quote whose parsed literal crossed source lines.
func sleepRunawayCompileMessage(source Source, runaway Diagnostic) string {
	if len(source.Data) == 0 {
		return ""
	}
	runawayOffset, runawayLine := sleepRunawayOpening(source.Data, runaway)
	if runawayLine <= 0 {
		return ""
	}
	delimiters := scanUnmatchedSourceDelimiters(source.Data, runawayOffset)
	parts := make([]string, 0, 4)
	parts = appendDelimiterDiagnostic(parts, "Parentheses", "paren", delimiters.parentheses)
	parts = appendDelimiterDiagnostic(parts, "Braces", "brace", delimiters.braces)
	parts = append(parts, fmt.Sprintf("Runaway string at %d", runawayLine))
	parts = appendDelimiterDiagnostic(parts, "Indices", "index", delimiters.indices)
	return fmt.Sprintf("YourCodeSucksException: %d error(s): %s", len(parts), strings.Join(parts, "; "))
}

func sleepRunawayOpening(data []byte, runaway Diagnostic) (int, int) {
	offset := runaway.Span.Start.Offset
	if offset < 0 || offset >= len(data) {
		return offset, runaway.Span.Start.Line
	}
	quote := data[offset]
	if quote != '\'' && quote != '"' && quote != '`' {
		return offset, runaway.Span.Start.Line
	}
	positions := sourceQuotePositions(data, offset, quote)
	if len(positions) < 3 {
		return offset, runaway.Span.Start.Line
	}
	previousLine := sourceLineAtOffset(data, positions[len(positions)-2])
	openingOffset := positions[len(positions)-3]
	openingLine := sourceLineAtOffset(data, openingOffset)
	if openingLine < previousLine {
		return openingOffset, openingLine
	}
	return offset, runaway.Span.Start.Line
}

func sourceQuotePositions(data []byte, through int, wanted byte) []int {
	positions := make([]int, 0, 4)
	quoted := byte(0)
	escaped := false
	comment := false
	for index := 0; index <= through && index < len(data); index++ {
		current := data[index]
		if comment {
			if current == '\n' {
				comment = false
			}
			continue
		}
		if quoted != 0 {
			if !escaped && current == quoted {
				if current == wanted {
					positions = append(positions, index)
				}
				quoted = 0
			}
			if current == '\\' {
				escaped = !escaped
			} else {
				escaped = false
			}
			continue
		}
		if current == '#' {
			comment = true
			continue
		}
		if current == '\'' || current == '"' || current == '`' {
			quoted = current
			escaped = false
			if current == wanted {
				positions = append(positions, index)
			}
		}
	}
	return positions
}

type sourceDelimiterBalance struct {
	opens  []int
	closes []int
}

type sourceDelimiterSet struct {
	parentheses sourceDelimiterBalance
	braces      sourceDelimiterBalance
	indices     sourceDelimiterBalance
}

func scanUnmatchedSourceDelimiters(data []byte, beforeOffset int) sourceDelimiterSet {
	var result sourceDelimiterSet
	if beforeOffset < 0 || beforeOffset > len(data) {
		beforeOffset = len(data)
	}
	line := 1
	quoted := byte(0)
	escaped := false
	comment := false
	for _, current := range data[:beforeOffset] {
		if current == '\n' {
			line++
			comment = false
			escaped = false
			continue
		}
		if comment {
			continue
		}
		if quoted != 0 {
			if !escaped && current == quoted {
				quoted = 0
			}
			escaped = current == '\\' && !escaped
			if current != '\\' {
				escaped = false
			}
			continue
		}
		if current == '#' {
			comment = true
			continue
		}
		if current == '\'' || current == '"' || current == '`' {
			quoted = current
			continue
		}
		switch current {
		case '(':
			result.parentheses.opens = append(result.parentheses.opens, line)
		case ')':
			closeSourceDelimiter(&result.parentheses, line)
		case '{':
			result.braces.opens = append(result.braces.opens, line)
		case '}':
			closeSourceDelimiter(&result.braces, line)
		case '[':
			result.indices.opens = append(result.indices.opens, line)
		case ']':
			closeSourceDelimiter(&result.indices, line)
		}
	}
	return result
}

func closeSourceDelimiter(balance *sourceDelimiterBalance, line int) {
	if len(balance.opens) == 0 {
		balance.closes = append(balance.closes, line)
		return
	}
	balance.opens = balance.opens[:len(balance.opens)-1]
}

func appendDelimiterDiagnostic(parts []string, label, noun string, balance sourceDelimiterBalance) []string {
	if len(balance.opens) != 0 {
		return append(parts, fmt.Sprintf("Mismatched %s - missing close %s at %d", label, noun, balance.opens[0]))
	}
	if len(balance.closes) != 0 {
		return append(parts, fmt.Sprintf("Mismatched %s - missing open %s at %d", label, noun, balance.closes[0]))
	}
	return parts
}

func sourceLineAtOffset(data []byte, offset int) int {
	if offset > len(data) {
		offset = len(data)
	}
	line := 1
	for _, current := range data[:offset] {
		if current == '\n' {
			line++
		}
	}
	return line
}

func pushIncludedSource(ctx context.Context, caller *fiber, policy IncludeCyclePolicy, name string, identityNames ...string) (context.Context, error) {
	chain, _ := ctx.Value(includeChainContextKey{}).([]includeChainEntry)
	chain = append([]includeChainEntry(nil), chain...)
	if len(chain) == 0 {
		current := ""
		if caller != nil && caller.function != nil {
			current = caller.function.Span.Source
		}
		if current == "" && caller != nil && caller.closure != nil && caller.closure.script != nil && caller.closure.script.program != nil {
			current = caller.closure.script.program.source.Name
		}
		if current != "" {
			chain = append(chain, includeChainEntry{identity: sourceIdentity(current), name: current})
		}
	}
	identityName := name
	if len(identityNames) != 0 && identityNames[0] != "" {
		identityName = identityNames[0]
	}
	identity := sourceIdentity(identityName)
	if policy == IncludeCycleReject {
		for _, entry := range chain {
			if entry.identity != identity {
				continue
			}
			names := make([]string, 0, len(chain)+1)
			for _, item := range chain {
				names = append(names, item.name)
			}
			names = append(names, name)
			return ctx, &IncludeCycleError{Chain: names}
		}
	}
	chain = append(chain, includeChainEntry{identity: identity, name: name})
	return context.WithValue(ctx, includeChainContextKey{}, chain), nil
}

func sourceIdentity(name string) string {
	if name == "" {
		return ""
	}
	if index := strings.Index(name, "!/"); index >= 0 {
		container := sourceIdentity(name[:index])
		member := strings.TrimPrefix(pathpkg.Clean("/"+name[index+2:]), "/")
		return container + "!/" + member
	}
	if strings.HasPrefix(name, "<") && strings.HasSuffix(name, ">") {
		return name
	}
	absolute, err := filepath.Abs(filepath.FromSlash(name))
	if err != nil {
		return filepath.Clean(filepath.FromSlash(name))
	}
	return filepath.Clean(absolute)
}

func isExecutionResourceError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, ErrResourceLimit) || errors.Is(err, ErrScriptUnloaded)
}
