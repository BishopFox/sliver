package server

type contextKey int

const (
	// This const is used as key for context value lookup
	requestHeader contextKey = iota
)
