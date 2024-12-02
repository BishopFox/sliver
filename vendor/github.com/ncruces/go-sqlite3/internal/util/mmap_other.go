//go:build !unix || sqlite3_nosys

package util

type mmapState struct{}
