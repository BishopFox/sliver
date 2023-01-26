package wasm

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/tetratelabs/wazero/api"
)

// moduleListNode is a node in a doubly linked list of names.
type moduleListNode struct {
	name       string
	module     *ModuleInstance
	next, prev *moduleListNode
}

// Namespace is a collection of instantiated modules which cannot conflict on name.
type Namespace struct {
	// moduleList ensures modules are closed in reverse initialization order.
	moduleList *moduleListNode // guarded by mux

	// nameToNode holds the instantiated Wasm modules by module name from Instantiate.
	// It ensures no race conditions instantiating two modules of the same name.
	nameToNode map[string]*moduleListNode // guarded by mux

	// mux is used to guard the fields from concurrent access.
	mux sync.RWMutex

	// closed is the pointer used both to guard Namespace.CloseWithExitCode.
	//
	// Note: Exclusively reading and updating this with atomics guarantees cross-goroutine observations.
	// See /RATIONALE.md
	closed *uint32
}

// newNamespace returns an empty namespace.
func newNamespace() *Namespace {
	return &Namespace{
		moduleList: nil,
		nameToNode: map[string]*moduleListNode{},
		closed:     new(uint32),
	}
}

// setModule makes the module visible for import.
func (ns *Namespace) setModule(m *ModuleInstance) error {
	if atomic.LoadUint32(ns.closed) != 0 {
		return errors.New("module set on closed namespace")
	}
	ns.mux.Lock()
	defer ns.mux.Unlock()
	node, ok := ns.nameToNode[m.Name]
	if !ok {
		return fmt.Errorf("module[%s] name has not been required", m.Name)
	}

	node.module = m
	return nil
}

// deleteModule makes the moduleName available for instantiation again.
func (ns *Namespace) deleteModule(moduleName string) error {
	if atomic.LoadUint32(ns.closed) != 0 {
		return fmt.Errorf("module[%s] deleted from closed namespace", moduleName)
	}
	ns.mux.Lock()
	defer ns.mux.Unlock()
	node, ok := ns.nameToNode[moduleName]
	if !ok {
		return nil
	}

	// remove this module name
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		ns.moduleList = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	}
	delete(ns.nameToNode, moduleName)
	return nil
}

// module returns the module of the given name or error if not in this namespace
func (ns *Namespace) module(moduleName string) (*ModuleInstance, error) {
	if atomic.LoadUint32(ns.closed) != 0 {
		return nil, fmt.Errorf("module[%s] requested from closed namespace", moduleName)
	}
	ns.mux.RLock()
	defer ns.mux.RUnlock()
	node, ok := ns.nameToNode[moduleName]
	if !ok {
		return nil, fmt.Errorf("module[%s] not in namespace", moduleName)
	}

	if node.module == nil {
		return nil, fmt.Errorf("module[%s] not set in namespace", moduleName)
	}

	return node.module, nil
}

// requireModules returns all instantiated modules whose names equal the keys in the input, or errs if any are missing.
func (ns *Namespace) requireModules(moduleNames map[string]struct{}) (map[string]*ModuleInstance, error) {
	if atomic.LoadUint32(ns.closed) != 0 {
		return nil, errors.New("modules required from closed namespace")
	}
	ret := make(map[string]*ModuleInstance, len(moduleNames))

	ns.mux.RLock()
	defer ns.mux.RUnlock()

	for n := range moduleNames {
		node, ok := ns.nameToNode[n]
		if !ok {
			return nil, fmt.Errorf("module[%s] not instantiated", n)
		}
		ret[n] = node.module
	}
	return ret, nil
}

// requireModuleName is a pre-flight check to reserve a module.
// This must be reverted on error with deleteModule if initialization fails.
func (ns *Namespace) requireModuleName(moduleName string) error {
	if atomic.LoadUint32(ns.closed) != 0 {
		return fmt.Errorf("module[%s] name required on closed namespace", moduleName)
	}
	ns.mux.Lock()
	defer ns.mux.Unlock()
	if _, ok := ns.nameToNode[moduleName]; ok {
		return fmt.Errorf("module[%s] has already been instantiated", moduleName)
	}

	// add the newest node to the moduleNamesList as the head.
	node := &moduleListNode{
		name: moduleName,
		next: ns.moduleList,
	}
	if node.next != nil {
		node.next.prev = node
	}
	ns.moduleList = node
	ns.nameToNode[moduleName] = node
	return nil
}

// AliasModule aliases the instantiated module named `src` as `dst`.
//
// Note: This is only used for spectests.
func (ns *Namespace) AliasModule(src, dst string) error {
	if atomic.LoadUint32(ns.closed) != 0 {
		return fmt.Errorf("module[%s] alias created on closed namespace", src)
	}
	ns.mux.Lock()
	defer ns.mux.Unlock()
	ns.nameToNode[dst] = ns.nameToNode[src]
	return nil
}

// CloseWithExitCode implements the same method as documented on wazero.Namespace.
func (ns *Namespace) CloseWithExitCode(ctx context.Context, exitCode uint32) (err error) {
	if !atomic.CompareAndSwapUint32(ns.closed, 0, 1) {
		return nil
	}
	ns.mux.Lock()
	defer ns.mux.Unlock()
	// Close modules in reverse initialization order.
	for node := ns.moduleList; node != nil; node = node.next {
		// If closing this module errs, proceed anyway to close the others.
		if m := node.module; m != nil {
			if _, e := m.CallCtx.close(ctx, exitCode); e != nil && err == nil {
				err = e // first error
			}
		}
	}
	ns.moduleList = nil
	ns.nameToNode = nil
	return
}

// Module implements wazero.Namespace Module
func (ns *Namespace) Module(moduleName string) api.Module {
	m, err := ns.module(moduleName)
	if err != nil {
		return nil
	}
	return m.CallCtx
}
