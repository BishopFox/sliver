//go:build linux || darwin || windows

package handlers

import (
	"errors"
	"fmt"

	"github.com/sliverarmory/reflektor/bof"
)

type bofObject interface {
	Execute([]byte) ([]bof.Output, error)
	Close() error
}

type bofLoadFunc func([]byte, bof.LoadOptions) (bofObject, error)

type bofExecuteFunc func([]byte, string, []byte) ([]bof.Output, error)

func loadReflektorBOF(data []byte, options bof.LoadOptions) (bofObject, error) {
	if options.EntryPoint == "" {
		return bof.Load(data)
	}
	return bof.LoadWithOptions(data, options)
}

func executeBOF(data []byte, entryPoint string, args []byte) ([]bof.Output, error) {
	return executeBOFWithLoader(data, entryPoint, args, loadReflektorBOF)
}

func executeBOFWithLoader(data []byte, entryPoint string, args []byte, load bofLoadFunc) (outputs []bof.Output, err error) {
	options := bof.LoadOptions{}
	if !isDefaultBOFEntryPoint(entryPoint) {
		options.EntryPoint = entryPoint
	}

	object, err := load(data, options)
	if err != nil {
		return nil, fmt.Errorf("load BOF: %w", err)
	}
	defer func() {
		if closeErr := object.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close BOF: %w", closeErr))
		}
	}()

	outputs, executeErr := object.Execute(args)
	if executeErr != nil {
		err = fmt.Errorf("execute BOF: %w", executeErr)
	}
	return outputs, err
}

func isDefaultBOFEntryPoint(entryPoint string) bool {
	switch entryPoint {
	case "", "go", "_go", "coffee", "_coffee":
		return true
	default:
		return false
	}
}
