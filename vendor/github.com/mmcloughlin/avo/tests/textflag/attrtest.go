// Package textflag tests that avo attribute constants agree with textflag.h.
package textflag

//go:generate go run make_attrtest.go -output zattrtest.s -seed 42 -num 256

func attrtest() bool
