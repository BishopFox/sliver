package opfor

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
)

// scope is one Sleep variable frame. Sleep variables are global unless a
// local()/this() declaration creates a nearer cell, so unresolved writes fall
// through to the script root rather than implicitly becoming lexical locals.
//
// A nil container is OPFOR's built-in synchronized map. A non-nil container is
// importer-owned and is always accessed through VariableContainer; cells is
// retained only for the built-in path and Sleep DefaultVariable serialization.
type scope struct {
	mu        sync.RWMutex
	parent    *scope
	root      *scope
	cells     map[string]*Cell
	known     map[string]*Cell
	container VariableContainer
	initErr   error

	runtimeID RuntimeID
	scriptID  ScriptID
	kind      VariableContainerKind
}

func newRootScope() *scope {
	s := &scope{cells: make(map[string]*Cell), kind: VariableContainerGlobal}
	s.root = s
	return s
}

func newVariableRootScope(runtimeID RuntimeID, scriptID ScriptID, container VariableContainer) *scope {
	s := &scope{
		container: container,
		known:     make(map[string]*Cell),
		runtimeID: runtimeID,
		scriptID:  scriptID,
		kind:      VariableContainerGlobal,
	}
	s.root = s
	return s
}

// forkRootAt asks the parent Script's global container for an internal
// container, then installs that container as a detached root for the fork. The
// resulting scope never falls through to the parent's globals.
func (s *scope) forkRootAt(ctx context.Context, runtimeID RuntimeID, scriptID ScriptID, span Span) (*scope, error) {
	if s == nil || s.root == nil || s.root.container == nil {
		root := newRootScope()
		root.runtimeID = runtimeID
		root.scriptID = scriptID
		root.kind = VariableContainerInternal
		return root, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	request := VariableContainerRequest{
		Kind: VariableContainerInternal, RuntimeID: runtimeID, Script: scriptID, Span: span,
	}
	container, providerErr := s.root.container.CreateInternalVariableContainer(ctx, request)
	err := joinExecutionContextError(ctx, variableProviderError(
		VariableProviderCreateInternal, runtimeID, scriptID, span, "", providerErr,
	))
	if err != nil {
		return nil, err
	}
	if isNilInterface(container) {
		return nil, variableProviderError(VariableProviderCreateInternal, runtimeID, scriptID, span, "", errors.New("provider returned a nil container"))
	}
	root := newVariableRootScope(runtimeID, scriptID, container)
	root.kind = VariableContainerInternal
	return root, nil
}

func (s *scope) internalChildAt(ctx context.Context, span Span) (*scope, error) {
	return s.childAt(ctx, span, VariableContainerInternal)
}

func (s *scope) localChildAt(ctx context.Context, span Span) (*scope, error) {
	return s.childAt(ctx, span, VariableContainerLocal)
}

func (s *scope) childAt(ctx context.Context, span Span, kind VariableContainerKind) (*scope, error) {
	if s == nil {
		return newRootScope(), nil
	}
	root := s.root
	if root == nil {
		root = s
	}
	child := &scope{
		parent:    s,
		root:      root,
		runtimeID: root.runtimeID,
		scriptID:  root.scriptID,
		kind:      kind,
	}
	if root.container == nil {
		child.cells = make(map[string]*Cell)
		return child, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	request := VariableContainerRequest{
		Kind: kind, RuntimeID: root.runtimeID, Script: root.scriptID, Span: span,
	}
	var container VariableContainer
	var err error
	var operation VariableProviderOperation
	switch kind {
	case VariableContainerLocal:
		operation = VariableProviderCreateLocal
		container, err = root.container.CreateLocalVariableContainer(ctx, request)
	case VariableContainerInternal:
		operation = VariableProviderCreateInternal
		container, err = root.container.CreateInternalVariableContainer(ctx, request)
	default:
		return nil, errors.New("opfor: invalid child variable container kind")
	}
	err = joinExecutionContextError(ctx, variableProviderError(
		operation, request.RuntimeID, request.Script, request.Span, "", err,
	))
	if err != nil {
		return nil, err
	}
	if isNilInterface(container) {
		return nil, variableProviderError(operation, request.RuntimeID, request.Script, request.Span, "", errors.New("provider returned a nil container"))
	}
	child.container = container
	child.known = make(map[string]*Cell)
	return child, nil
}

// child is retained for DefaultVariable serialization and package-internal
// tests. Interpreter paths use internalChildAt/localChildAt and propagate
// provider errors. If a legacy caller reaches a provider failure, initErr makes
// every subsequent operation return the same authoritative error.
func (s *scope) child() *scope {
	child, err := s.internalChildAt(context.Background(), Span{})
	if err == nil {
		return child
	}
	return s.failedChild(VariableContainerInternal, err)
}

// nextLocal creates another local level for the same closure invocation. A
// Sleep pushl frame replaces the active local lookup level: variables absent
// from it fall through to closure/global scope, not to the local frame it
// temporarily hides.
func (s *scope) nextLocalAt(ctx context.Context, span Span) (*scope, error) {
	if s == nil {
		return newRootScope(), nil
	}
	child, err := s.localChildAt(ctx, span)
	if err != nil {
		return nil, err
	}
	child.parent = s.parent
	return child, nil
}

func (s *scope) nextLocal() *scope {
	child, err := s.nextLocalAt(context.Background(), Span{})
	if err == nil {
		return child
	}
	return s.failedChild(VariableContainerLocal, err)
}

func (s *scope) failedChild(kind VariableContainerKind, err error) *scope {
	if s == nil {
		failed := newRootScope()
		failed.initErr = err
		return failed
	}
	root := s.root
	if root == nil {
		root = s
	}
	return &scope{
		parent: s, root: root, initErr: err, kind: kind,
		runtimeID: root.runtimeID, scriptID: root.scriptID,
	}
}

func normalizeVariableName(name string) string {
	// Compiler-produced names already carry a Sleep sigil and have no source
	// whitespace. They dominate evaluator lookups, so avoid a Unicode trim scan
	// on every loop operand while retaining the general importer-facing
	// normalization below. Non-ASCII endings take the slow path because
	// strings.TrimSpace recognizes Unicode whitespace.
	if len(name) != 0 {
		switch name[0] {
		case '$', '@', '%':
			last := name[len(name)-1]
			if last > ' ' && last < 0x80 {
				return name
			}
		}
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "$"
	}
	switch name[0] {
	case '$', '@', '%':
		return name
	default:
		return "$" + name
	}
}

func splitVariableNames(value string) []string {
	fields := strings.Fields(value)
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		if field == "" {
			continue
		}
		result = append(result, normalizeVariableName(field))
	}
	return result
}

func (s *scope) access(name string, span Span) VariableAccess {
	name = normalizeVariableName(name)
	root := s
	if s != nil && s.root != nil {
		root = s.root
	}
	if root == nil {
		return VariableAccess{Name: name, Span: span}
	}
	return VariableAccess{
		RuntimeID: root.runtimeID, Script: root.scriptID, Span: span, Name: name,
	}
}

func (s *scope) ownCellAt(ctx context.Context, name string, span Span) (*Cell, bool, error) {
	if s == nil {
		return nil, false, nil
	}
	if s.initErr != nil {
		return nil, false, s.initErr
	}
	if s.container == nil {
		name = normalizeVariableName(name)
		s.mu.RLock()
		cell := s.cells[name]
		s.mu.RUnlock()
		return cell, cell != nil, nil
	}
	access := s.access(name, span)
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	exists, providerErr := s.container.ScalarExists(ctx, access)
	err := joinExecutionContextError(ctx, variableProviderError(
		VariableProviderExists, access.RuntimeID, access.Script, access.Span, access.Name, providerErr,
	))
	if err != nil {
		return nil, false, err
	}
	if !exists {
		s.mu.Lock()
		delete(s.known, access.Name)
		s.mu.Unlock()
		return nil, false, nil
	}
	cell, providerErr := s.container.GetScalar(ctx, access)
	err = joinExecutionContextError(ctx, variableProviderError(
		VariableProviderGet, access.RuntimeID, access.Script, access.Span, access.Name, providerErr,
	))
	if err != nil {
		return nil, false, err
	}
	if cell == nil {
		return nil, false, variableProviderError(VariableProviderGet, access.RuntimeID, access.Script, access.Span, access.Name, errors.New("ScalarExists returned true but GetScalar returned a nil cell"))
	}
	s.mu.Lock()
	s.known[access.Name] = cell
	s.mu.Unlock()
	return cell, true, nil
}

func (s *scope) scalarExistsAt(ctx context.Context, name string, span Span) (bool, error) {
	if s == nil {
		return false, nil
	}
	if s.initErr != nil {
		return false, s.initErr
	}
	if s.container == nil {
		name = normalizeVariableName(name)
		s.mu.RLock()
		exists := s.cells[name] != nil
		s.mu.RUnlock()
		return exists, nil
	}
	access := s.access(name, span)
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	exists, providerErr := s.container.ScalarExists(ctx, access)
	err := joinExecutionContextError(ctx, variableProviderError(
		VariableProviderExists, access.RuntimeID, access.Script, access.Span, access.Name, providerErr,
	))
	if err != nil {
		return false, err
	}
	return exists, nil
}

// levelAt mirrors ScriptVariables.getScalarLevel: only the current local,
// current closure, and global containers are considered, in that order.
func (s *scope) levelAt(ctx context.Context, name string, span Span) (*scope, bool, error) {
	name = normalizeVariableName(name)
	for current := s; current != nil; current = current.parent {
		exists, err := current.scalarExistsAt(ctx, name, span)
		if err != nil {
			return nil, false, err
		}
		if exists {
			return current, true, nil
		}
	}
	if s == nil {
		return nil, false, nil
	}
	return s.root, false, nil
}

func (s *scope) putCellAt(ctx context.Context, name string, cell *Cell, span Span) error {
	if s == nil {
		return errors.New("opfor: variable scope is nil")
	}
	if s.initErr != nil {
		return s.initErr
	}
	if cell == nil {
		access := s.access(name, span)
		return variableProviderError(VariableProviderPut, access.RuntimeID, access.Script, access.Span, access.Name, errors.New("cannot store a nil cell"))
	}
	if s.container == nil {
		name = normalizeVariableName(name)
		s.mu.Lock()
		s.cells[name] = cell
		s.mu.Unlock()
		return nil
	}
	access := s.access(name, span)
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	_, providerErr := s.container.PutScalar(ctx, access, cell)
	err := joinExecutionContextError(ctx, variableProviderError(
		VariableProviderPut, access.RuntimeID, access.Script, access.Span, access.Name, providerErr,
	))
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.known[access.Name] = cell
	s.mu.Unlock()
	return nil
}

func (s *scope) removeOwnAt(ctx context.Context, name string, span Span) error {
	if s == nil {
		return nil
	}
	if s.initErr != nil {
		return s.initErr
	}
	if s.container == nil {
		name = normalizeVariableName(name)
		s.mu.Lock()
		delete(s.cells, name)
		s.mu.Unlock()
		return nil
	}
	access := s.access(name, span)
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	providerErr := s.container.RemoveScalar(ctx, access)
	err := joinExecutionContextError(ctx, variableProviderError(
		VariableProviderRemove, access.RuntimeID, access.Script, access.Span, access.Name, providerErr,
	))
	if err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.known, access.Name)
	s.mu.Unlock()
	return nil
}

func (s *scope) localAt(ctx context.Context, name string, span Span) (*Cell, error) {
	if s == nil {
		return nil, errors.New("opfor: variable scope is nil")
	}
	if cell, ok, err := s.ownCellAt(ctx, name, span); err != nil || ok {
		return cell, err
	}
	name = normalizeVariableName(name)
	cell := NewCell(defaultVariableValue(name))
	if err := s.putCellAt(ctx, name, cell, span); err != nil {
		return nil, err
	}
	return cell, nil
}

func (s *scope) local(name string) *Cell {
	cell, _ := s.localAt(context.Background(), name, Span{})
	return cell
}

func (s *scope) globalAt(ctx context.Context, name string, span Span) (*Cell, error) {
	if s == nil {
		return nil, errors.New("opfor: variable scope is nil")
	}
	return s.root.localAt(ctx, name, span)
}

func (s *scope) global(name string) *Cell {
	cell, _ := s.globalAt(context.Background(), name, Span{})
	return cell
}

func (s *scope) lookupAt(ctx context.Context, name string, span Span) (*Cell, bool, error) {
	name = normalizeVariableName(name)
	for current := s; current != nil; current = current.parent {
		cell, ok, err := current.ownCellAt(ctx, name, span)
		if err != nil {
			return nil, false, err
		}
		if ok {
			return cell, true, nil
		}
	}
	return nil, false, nil
}

func (s *scope) lookup(name string) (*Cell, bool) {
	cell, ok, _ := s.lookupAt(context.Background(), name, Span{})
	return cell, ok
}

func (s *scope) resolveAt(ctx context.Context, name string, span Span) (*Cell, error) {
	if cell, ok, err := s.lookupAt(ctx, name, span); err != nil || ok {
		return cell, err
	}
	name = normalizeVariableName(name)
	cell := NewCell(defaultVariableValue(name))
	if err := s.root.putCellAt(ctx, name, cell, span); err != nil {
		return nil, err
	}
	return cell, nil
}

func (s *scope) resolve(name string) *Cell {
	cell, _ := s.resolveAt(context.Background(), name, Span{})
	return cell
}

func (s *scope) getAt(ctx context.Context, name string, span Span) (Value, error) {
	if cell, ok, err := s.lookupAt(ctx, name, span); err != nil {
		return Null(), err
	} else if ok {
		return cell.Get(), nil
	}
	name = normalizeVariableName(name)
	if name[0] == '@' || name[0] == '%' {
		cell := NewCell(defaultVariableValue(name))
		if err := s.root.putCellAt(ctx, name, cell, span); err != nil {
			return Null(), err
		}
		return cell.Get(), nil
	}
	return Null(), nil
}

func (s *scope) get(name string) Value {
	value, _ := s.getAt(context.Background(), name, Span{})
	return value
}

// bindAt replaces the nearest visible slot with cell. Foreach uses this to
// make its value variable an alias of the current array/hash element, matching
// Sleep's mutation-through-iteration behavior.
func (s *scope) bindAt(ctx context.Context, name string, cell *Cell, span Span) error {
	if s == nil || cell == nil {
		return nil
	}
	name = normalizeVariableName(name)
	for current := s; current != nil; current = current.parent {
		_, ok, err := current.ownCellAt(ctx, name, span)
		if err != nil {
			return err
		}
		if ok {
			return current.putCellAt(ctx, name, cell, span)
		}
	}
	return s.root.putCellAt(ctx, name, cell, span)
}

func (s *scope) bind(name string, cell *Cell) {
	_ = s.bindAt(context.Background(), name, cell, Span{})
}

func defaultVariableValue(name string) Value {
	if name == "" {
		return Null()
	}
	switch name[0] {
	case '@':
		return ArrayValue(NewArray())
	case '%':
		return HashValue(NewHash())
	default:
		return Null()
	}
}

func (s *scope) snapshotRootAt(ctx context.Context) (map[string]Value, error) {
	if s == nil {
		return nil, nil
	}
	root := s.root
	if root == nil {
		root = s
	}
	if root.initErr != nil {
		return nil, root.initErr
	}
	if root.container == nil {
		root.mu.RLock()
		cells := make(map[string]*Cell, len(root.cells))
		for name, cell := range root.cells {
			cells[name] = cell
		}
		root.mu.RUnlock()
		result := make(map[string]Value, len(cells))
		for name, cell := range cells {
			result[name] = cell.Get()
		}
		return result, nil
	}
	root.mu.RLock()
	names := make([]string, 0, len(root.known))
	for name := range root.known {
		names = append(names, name)
	}
	root.mu.RUnlock()
	result := make(map[string]Value, len(names))
	for _, name := range names {
		cell, ok, err := root.ownCellAt(ctx, name, Span{})
		if err != nil {
			return nil, err
		}
		if ok {
			result[name] = cell.Get()
		}
	}
	return result, nil
}

// snapshotOwnAt captures only this variable level. Breakpoint inspection uses
// it to keep locals, closure captures, and globals distinct without exposing
// the scope or its Cells. Importer-backed levels can enumerate only names the
// interpreter has observed, matching GlobalsContext's provider boundary.
func (s *scope) snapshotOwnAt(ctx context.Context, span Span) (map[string]Value, error) {
	if s == nil {
		return nil, nil
	}
	if s.initErr != nil {
		return nil, s.initErr
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.container == nil {
		s.mu.RLock()
		cells := make(map[string]*Cell, len(s.cells))
		for name, cell := range s.cells {
			cells[name] = cell
		}
		s.mu.RUnlock()
		result := make(map[string]Value, len(cells))
		for name, cell := range cells {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			result[name] = cell.Get()
		}
		return result, nil
	}

	s.mu.RLock()
	names := make([]string, 0, len(s.known))
	for name := range s.known {
		names = append(names, name)
	}
	s.mu.RUnlock()
	sort.Strings(names)
	result := make(map[string]Value, len(names))
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		cell, ok, err := s.ownCellAt(ctx, name, span)
		if err != nil {
			return nil, err
		}
		if ok {
			result[name] = cell.Get()
		}
	}
	return result, nil
}

func (s *scope) snapshotRoot() map[string]Value {
	result, _ := s.snapshotRootAt(context.Background())
	return result
}

func (s *scope) defaultCellsSnapshot() (map[string]*Cell, error) {
	if s == nil {
		return nil, nil
	}
	if s.container != nil || (s.root != nil && s.root.container != nil) {
		return nil, &UnsupportedError{Operation: "serialization", Name: "importer variable container"}
	}
	if s.initErr != nil {
		return nil, s.initErr
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]*Cell, len(s.cells))
	for name, cell := range s.cells {
		result[name] = cell
	}
	return result, nil
}

func variableProviderError(operation VariableProviderOperation, runtimeID RuntimeID, script ScriptID, span Span, name string, cause error) error {
	if cause == nil {
		return nil
	}
	var existing *VariableProviderError
	if errors.As(cause, &existing) {
		return cause
	}
	return &VariableProviderError{
		Operation: operation, RuntimeID: runtimeID, Script: script, Span: span, Name: name, Cause: cause,
	}
}
