package main

var (
	darwinHandlers = map[string]interface{}{
		"ping": pingHandler,
	}
)

func getSystemHandlers() map[string]interface{} {
	return darwinHandlers
}
