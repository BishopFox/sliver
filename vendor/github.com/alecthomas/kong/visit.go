package kong

import (
	"fmt"
)

// Next should be called by Visitor to proceed with the walk.
//
// The walk will terminate if "err" is non-nil.
type Next func(err error) error

// Visitor can be used to walk all nodes in the model.
type Visitor func(node Visitable, next Next) error

// Visit all nodes.
func Visit(node Visitable, visitor Visitor) error {
	return visitor(node, func(err error) error {
		if err != nil {
			return err
		}
		switch node := node.(type) {
		case *Application:
			return visitNodeChildren(node.Node, visitor)
		case *Node:
			return visitNodeChildren(node, visitor)
		case *Value:
		case *Flag:
			return Visit(node.Value, visitor)
		default:
			panic(fmt.Sprintf("unsupported node type %T", node))
		}
		return nil
	})
}

func visitNodeChildren(node *Node, visitor Visitor) error {
	if node.Argument != nil {
		if err := Visit(node.Argument, visitor); err != nil {
			return err
		}
	}
	for _, flag := range node.Flags {
		if err := Visit(flag, visitor); err != nil {
			return err
		}
	}
	for _, pos := range node.Positional {
		if err := Visit(pos, visitor); err != nil {
			return err
		}
	}
	for _, child := range node.Children {
		if err := Visit(child, visitor); err != nil {
			return err
		}
	}
	return nil
}
