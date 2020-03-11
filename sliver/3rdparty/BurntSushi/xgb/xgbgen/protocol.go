package main

import (
	"log"
	"strings"
)

// Protocol is a type that encapsulates all information about one
// particular XML file. It also contains links to other protocol types
// if this protocol imports other other extensions. The import relationship
// is recursive.
type Protocol struct {
	Parent       *Protocol
	Name         string
	ExtXName     string
	ExtName      string
	MajorVersion string
	MinorVersion string

	Imports  []*Protocol
	Types    []Type
	Requests []*Request
}

type Protocols []*Protocol

func (ps Protocols) Len() int           { return len(ps) }
func (ps Protocols) Swap(i, j int)      { ps[i], ps[j] = ps[j], ps[i] }
func (ps Protocols) Less(i, j int) bool { return ps[i].ExtName < ps[j].ExtName }

// Initialize traverses all structures, looks for 'Translation' type,
// and looks up the real type in the namespace. It also sets the source
// name for all relevant fields/structures.
// This is necessary because we don't traverse the XML in order initially.
func (p *Protocol) Initialize() {
	for _, typ := range p.Types {
		typ.Initialize(p)
	}
	for _, req := range p.Requests {
		req.Initialize(p)
	}
}

// PkgName returns the name of this package.
// i.e., 'xproto' for the core X protocol, 'randr' for the RandR extension, etc.
func (p *Protocol) PkgName() string {
	return strings.Replace(p.Name, "_", "", -1)
}

// ProtocolGet searches the current context for the protocol with the given
// name. (i.e., the current protocol and its imports.)
// It is an error if one is not found.
func (p *Protocol) ProtocolFind(name string) *Protocol {
	if p.Name == name {
		return p // that was easy
	}
	for _, imp := range p.Imports {
		if imp.Name == name {
			return imp
		}
	}
	log.Panicf("Could not find protocol with name '%s'.", name)
	panic("unreachable")
}

// isExt returns true if this protocol is an extension.
// i.e., it's name isn't "xproto".
func (p *Protocol) isExt() bool {
	return strings.ToLower(p.Name) != "xproto"
}
