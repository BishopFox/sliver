package protobufs

import (
	"embed"
)

var (

	// FS - Embedded FS access to proto files
	//go:embed commonpb/* sliverpb/* commpb/*
	FS embed.FS
)
