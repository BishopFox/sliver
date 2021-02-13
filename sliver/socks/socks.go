package socks

import (
	"io"
)

type Socks struct {
	ID      uint64
	Stdout  io.ReadCloser
	Stdin   io.WriteCloser
}