package bofloader

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

const maxFormatAllocation = 16 << 20

var (
	executionLock   sync.Mutex
	activeExecution atomic.Pointer[executionContext]
)

type executionContext struct {
	mu          sync.Mutex
	outputs     []Output
	errors      []error
	allocations map[uintptr][]byte
}

func newExecutionContext() *executionContext {
	return &executionContext{allocations: make(map[uintptr][]byte)}
}

func (context *executionContext) appendOutput(outputType int, data []byte) {
	if context == nil {
		return
	}
	copyOfData := append([]byte(nil), data...)
	context.mu.Lock()
	context.outputs = append(context.outputs, Output{Type: outputType, Data: copyOfData})
	context.mu.Unlock()
}

func (context *executionContext) addError(err error) {
	if context == nil || err == nil {
		return
	}
	context.mu.Lock()
	context.errors = append(context.errors, err)
	context.mu.Unlock()
}

func (context *executionContext) allocate(size int) (uintptr, []byte, error) {
	if context == nil {
		return 0, nil, errors.New("no BOF execution is active")
	}
	if size <= 0 || size > maxFormatAllocation {
		return 0, nil, fmt.Errorf("format allocation size %d is outside 1..%d", size, maxFormatAllocation)
	}
	data := make([]byte, size)
	address := byteSliceAddress(data)
	if address == 0 {
		return 0, nil, errors.New("format allocation returned a nil address")
	}
	context.mu.Lock()
	context.allocations[address] = data
	context.mu.Unlock()
	return address, data, nil
}

func (context *executionContext) allocation(address uintptr) ([]byte, bool) {
	if context == nil || address == 0 {
		return nil, false
	}
	context.mu.Lock()
	data, ok := context.allocations[address]
	context.mu.Unlock()
	return data, ok
}

func (context *executionContext) release(address uintptr) {
	if context == nil || address == 0 {
		return
	}
	context.mu.Lock()
	delete(context.allocations, address)
	context.mu.Unlock()
}

func (context *executionContext) result() ([]Output, error) {
	if context == nil {
		return nil, errors.New("nil BOF execution context")
	}
	context.mu.Lock()
	defer context.mu.Unlock()

	outputs := make([]Output, len(context.outputs))
	for index := range context.outputs {
		outputs[index] = Output{
			Type: context.outputs[index].Type,
			Data: append([]byte(nil), context.outputs[index].Data...),
		}
	}
	return outputs, errors.Join(context.errors...)
}
