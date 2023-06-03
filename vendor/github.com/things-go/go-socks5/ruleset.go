package socks5

import (
	"context"

	"github.com/things-go/go-socks5/statute"
)

// RuleSet is used to provide custom rules to allow or prohibit actions
type RuleSet interface {
	Allow(ctx context.Context, req *Request) (context.Context, bool)
}

// PermitCommand is an implementation of the RuleSet which
// enables filtering supported commands
type PermitCommand struct {
	EnableConnect   bool
	EnableBind      bool
	EnableAssociate bool
}

// NewPermitNone returns a RuleSet which disallows all types of connections
func NewPermitNone() RuleSet {
	return &PermitCommand{false, false, false}
}

// NewPermitAll returns a RuleSet which allows all types of connections
func NewPermitAll() RuleSet {
	return &PermitCommand{true, true, true}
}

// NewPermitConnAndAss returns a RuleSet which allows Connect and Associate connection
func NewPermitConnAndAss() RuleSet {
	return &PermitCommand{true, false, true}
}

// Allow implement interface RuleSet
func (p *PermitCommand) Allow(ctx context.Context, req *Request) (context.Context, bool) {
	switch req.Command {
	case statute.CommandConnect:
		return ctx, p.EnableConnect
	case statute.CommandBind:
		return ctx, p.EnableBind
	case statute.CommandAssociate:
		return ctx, p.EnableAssociate
	}
	return ctx, false
}
